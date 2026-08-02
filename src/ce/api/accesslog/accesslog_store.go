package accesslog

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// accessLogInsertFields is the number of columns written per row (log_id is
// generated and excluded).
const accessLogInsertFields = 16

// MaxInsertRows is how many access logs fit in one INSERT. Postgres accepts at
// most 65535 bind parameters per statement, and the driver refuses the statement
// outright beyond that — so a batch larger than this has to be split rather than
// attempted and lost.
const MaxInsertRows = 65535 / accessLogInsertFields

// DefaultLimit caps a single page of access-log results when the caller does
// not ask for a specific page size.
const DefaultLimit = 100

// MaxLimit is the largest page an API caller may ask for. Rows are wide (user
// agent, referrer, path) and the whole page is serialized into a single
// response, so this is kept well below what the index could stream. It bounds
// what arrives over HTTP only — internal callers set SelectLogsParams.Limit
// directly and are trusted with it.
const MaxLimit = 1000

var stmt = struct {
	insertLogs string
	selectLogs string
}{
	insertLogs: `
		INSERT INTO request_logs (
			app_id, env_id, deployment_id, domain_id, host_name,
			request_timestamp, method, request_path, response_code,
			client_ip, user_agent, referrer, is_bot, bytes_sent, request_id,
			duration_ms
		)
		VALUES {{ generateValues 16 (len .) }};
	`,

	selectLogs: `
		SELECT
			log_id, app_id, env_id, deployment_id, domain_id, host_name,
			request_timestamp, method, request_path, response_code,
			client_ip, user_agent, referrer, is_bot, bytes_sent, request_id,
			duration_ms
		FROM request_logs
		WHERE {{ .where }}
		ORDER BY request_timestamp DESC, log_id DESC
		LIMIT {{ .limit }};
	`,
}

// Store handles raw access-log persistence and retrieval.
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

// stripNull removes NUL bytes, which Postgres rejects in text/jsonb and which
// would otherwise abort an entire multi-row batch insert because of one row.
func stripNull(s string) string {
	if !strings.ContainsRune(s, '\x00') {
		return s
	}

	return strings.ReplaceAll(s, "\x00", "")
}

// InsertLogs writes a batch of access logs, splitting it into as many multi-row
// INSERTs as the bind-parameter limit requires.
//
// The ingest job hands over everything it drained in a tick — tens of thousands
// of rows under load — so the batch routinely exceeds what one statement can
// carry. Chunking here rather than at the call site keeps the limit an
// implementation detail of the write, and keeps a traffic burst from failing the
// insert outright and dropping every access log in it.
func (s *Store) InsertLogs(ctx context.Context, logs []AccessLog) error {
	for len(logs) > MaxInsertRows {
		if err := s.insertLogsChunk(ctx, logs[:MaxInsertRows]); err != nil {
			return err
		}

		logs = logs[MaxInsertRows:]
	}

	return s.insertLogsChunk(ctx, logs)
}

func (s *Store) insertLogsChunk(ctx context.Context, logs []AccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	params := make([]any, 0, len(logs)*accessLogInsertFields)

	for _, l := range logs {
		params = append(params,
			l.AppID, l.EnvID, l.DeploymentID, l.DomainID, stripNull(l.HostName),
			l.RequestTS.UTC(), stripNull(l.Method), stripNull(l.RequestPath), l.StatusCode,
			stripNull(l.ClientIP), stripNull(l.UserAgent), stripNull(l.Referrer),
			l.IsBot, l.BytesSent, l.RequestID, l.DurationMS,
		)
	}

	var wr bytes.Buffer

	query := template.Must(template.New("insertLogs").
		Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
		Parse(stmt.insertLogs))

	if err := query.Execute(&wr, logs); err != nil {
		return fmt.Errorf("error while compiling insertLogs template: %v", err)
	}

	_, err := s.Exec(ctx, wr.String(), params...)

	return err
}

// SelectLogsParams filters a page of access logs. From/To bound
// request_timestamp and should always be set so the query prunes partitions
// instead of scanning the full retention window.
type SelectLogsParams struct {
	AppID    types.ID
	EnvID    types.ID
	DomainID types.ID
	HostName string
	ClientIP string
	Method   string
	Path     string
	Status   int
	IsBot    *bool
	// MinDurationMS keeps only requests that took at least this many
	// milliseconds. Entries with a null duration (written before the column
	// existed, or never stamped) are excluded rather than treated as 0.
	MinDurationMS int
	From          utils.Unix
	To            utils.Unix
	BeforeID      types.ID
	BeforeTS      utils.Unix
	Limit         int
}

// SelectLogs returns access logs matching the filters, newest first, with one
// extra row beyond Limit so the caller can detect a next page.
func (s *Store) SelectLogs(ctx context.Context, p SelectLogsParams) ([]AccessLog, error) {
	where := []string{}
	params := []any{}

	add := func(clause string, value any) {
		params = append(params, value)
		where = append(where, fmt.Sprintf(clause, len(params)))
	}

	if p.From.Valid {
		add("request_timestamp >= $%d", p.From.UTC())
	}

	if p.To.Valid {
		add("request_timestamp <= $%d", p.To.UTC())
	}

	if p.AppID > 0 {
		add("app_id = $%d", p.AppID)
	}

	if p.EnvID > 0 {
		add("env_id = $%d", p.EnvID)
	}

	if p.DomainID > 0 {
		add("domain_id = $%d", p.DomainID)
	}

	if p.HostName != "" {
		add("host_name = $%d", p.HostName)
	}

	if p.ClientIP != "" {
		add("client_ip = $%d", p.ClientIP)
	}

	if p.Method != "" {
		add("method = $%d", p.Method)
	}

	if p.Path != "" {
		add("request_path LIKE $%d", p.Path+"%")
	}

	if p.Status > 0 {
		add("response_code = $%d", p.Status)
	}

	if p.IsBot != nil {
		add("is_bot = $%d", *p.IsBot)
	}

	if p.MinDurationMS > 0 {
		add("duration_ms >= $%d", p.MinDurationMS)
	}

	// The cursor compares on the same tuple the query orders by, otherwise
	// Postgres cannot seek to the page and re-walks the window from the newest
	// row every time. Both halves are required — see AccessLog.Cursor.
	if p.BeforeID > 0 && p.BeforeTS.Valid {
		params = append(params, p.BeforeTS.UTC(), p.BeforeID)
		where = append(where, fmt.Sprintf(
			"(request_timestamp, log_id) < ($%d, $%d)", len(params)-1, len(params),
		))
	}

	if len(where) == 0 {
		where = append(where, "TRUE")
	}

	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}

	var wr bytes.Buffer

	tmpl := template.Must(template.New("selectLogs").Parse(stmt.selectLogs))

	if err := tmpl.Execute(&wr, map[string]any{
		"where": strings.Join(where, " AND "),
		"limit": p.Limit + 1,
	}); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	logs := []AccessLog{}

	for rows.Next() {
		l := AccessLog{}

		var deploymentID, domainID, bytesSent, status null.Int
		var hostName, method, path, clientIP, userAgent, referrer null.String

		if err := rows.Scan(
			&l.ID, &l.AppID, &l.EnvID, &deploymentID, &domainID, &hostName,
			&l.RequestTS, &method, &path, &status,
			&clientIP, &userAgent, &referrer, &l.IsBot, &bytesSent, &l.RequestID,
			&l.DurationMS,
		); err != nil {
			return nil, err
		}

		l.DeploymentID = types.ID(deploymentID.ValueOrZero())
		l.DomainID = types.ID(domainID.ValueOrZero())
		l.BytesSent = bytesSent.ValueOrZero()
		l.StatusCode = int(status.ValueOrZero())
		l.HostName = hostName.ValueOrZero()
		l.Method = method.ValueOrZero()
		l.RequestPath = path.ValueOrZero()
		l.ClientIP = clientIP.ValueOrZero()
		l.UserAgent = userAgent.ValueOrZero()
		l.Referrer = referrer.ValueOrZero()

		logs = append(logs, l)
	}

	return logs, nil
}
