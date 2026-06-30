package analytics

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

var stmt = struct {
	insertRecord                string
	insertEvents                string
	selectEvents                string
	breakdownEvents             string
	eventPropertyKeys           string
	visitors                    string
	topReferrers                string
	topPaths                    string
	byCountries                 string
	totalRequestsByTeam         string
	totalAppsByTeam             string
	totalDeploymentsByTeam      string
	avgDeploymentDurationByTeam string
	topPerformingDomains        string
}{
	insertRecord: `
		INSERT INTO analytics (
			app_id, env_id, visitor_ip,
			request_path, request_timestamp,
			response_code, user_agent, referrer,
			domain_id, country_iso_code, request_id, source
		)
		VALUES {{ range $i, $record := .records }}
			(
				${{ $record.p1 }}, ${{ $record.p2 }}, ${{ $record.p3 }}::inet,
				${{ $record.p4 }}, ${{ $record.p5 }}, ${{ $record.p6 }},
				${{ $record.p7 }}, ${{ $record.p8 }}, ${{ $record.p9 }},
				{{ if $record.geoLocation }} (
					SELECT
						gc.country_iso_code
					FROM
						geo_ips gi
					JOIN
						geo_countries gc ON gi.geoname_id = gc.geoname_id
					WHERE
						${{ $record.p3 }}::inet <<= network
					LIMIT 1
				) {{ else }}
					NULL
				{{ end }},
				${{ $record.p10 }}::uuid,
				${{ $record.p11 }}
			){{ if not (last $i $.records) }},{{ end }}
		{{ end }};
	`,

	insertEvents: `
		INSERT INTO analytics_events (
			app_id, env_id, domain_id,
			visitor_id, event_name, request_path,
			event_ts, request_id, metadata
		)
		VALUES {{ generateValues 9 (len .) }};
	`,

	// selectEvents aggregates rolled-up event counts for a domain over a window.
	// NOTE: unique_actors is summed across daily aggregates, so a visitor active
	// on multiple days is counted once per day (i.e. unique actor-days, not
	// unique actors over the whole window).
	// NOTE: event_name cardinality is client-controlled, so the result is capped
	// at the 100 highest-count events; lower-count events are omitted.
	selectEvents: `
		SELECT
			event_name,
			SUM(total_count) AS total,
			SUM(unique_actors) AS unique_actors
		FROM analytics_events_agg
		WHERE
			domain_id = $1 AND
			aggregate_date >= current_date - make_interval(days => $2)
		GROUP BY event_name
		ORDER BY total DESC
		LIMIT 100;
	`,

	// breakdownEvents groups a single event by one metadata property, computed
	// from the raw events (within the retention window). The (domain_id,
	// event_name, event_ts) index bounds the scan to that event's rows. The
	// property key ($3) is a bind parameter, so it is not SQL-interpolated.
	breakdownEvents: `
		SELECT
			COALESCE(metadata->>$3, '(not set)') AS property_value,
			COUNT(*) AS total,
			COUNT(DISTINCT visitor_id) AS unique_actors
		FROM analytics_events
		WHERE
			domain_id = $1 AND
			event_name = $2 AND
			event_ts >= now() - make_interval(days => $4)
		GROUP BY 1
		ORDER BY total DESC
		LIMIT 100;
	`,

	// eventPropertyKeys returns the distinct metadata keys seen for an event,
	// sampled from up to 1000 recent rows to keep discovery cheap.
	eventPropertyKeys: `
		SELECT DISTINCT key
		FROM (
			SELECT metadata
			FROM analytics_events
			WHERE
				domain_id = $1 AND
				event_name = $2 AND
				event_ts >= now() - make_interval(days => $3) AND
				metadata IS NOT NULL
			ORDER BY event_ts DESC
			LIMIT 1000
		) sampled,
		LATERAL jsonb_object_keys(sampled.metadata) AS key
		ORDER BY key
		LIMIT 50;
	`,

	visitors: `
		SELECT
			{{ .aggregateColumn }}, unique_visitors, total_visitors
		FROM
			{{ .tableName }}
		WHERE
			domain_id = $1
			{{ if .interval }}
				AND {{ .aggregateColumn }} >= {{ .interval }}
			{{ end }}
		ORDER BY
			{{ .aggregateColumn }} DESC {{ if not .interval }} 
		LIMIT 24{{ end }};
	`,

	topReferrers: `
		SELECT
			referrer,
			SUM(visit_count) as total_count
		FROM
			analytics_referrers
		WHERE
			domain_id = $1 AND
			aggregate_date >= current_date - interval '30 days' AND
			referrer <> ''
			{{ if .requestPath }}
				AND request_path = $2
			{{ end }}
		GROUP BY
			referrer
		ORDER BY
			total_count DESC
		LIMIT 51;
	`,

	topPaths: `
		SELECT
			request_path,
			SUM(visit_count) as total_count
		FROM analytics_referrers
		WHERE
            domain_id = $1 AND
			aggregate_date >= current_date - interval '30 days'
		GROUP BY
			request_path
		ORDER BY
			total_count DESC
		LIMIT 50;
	`,

	byCountries: `
		SELECT
			country_iso_code,
			SUM(visit_count) as count
		FROM
			analytics_visitors_by_countries
		WHERE
			domain_id = $1 AND
			aggregate_date >= current_date - interval '30 days'
		GROUP BY country_iso_code
		ORDER BY count desc;
	`,

	totalRequestsByTeam: `
		WITH periods AS (
			SELECT 
				total_visitors,
				CASE
					WHEN aggregate_date >= current_date - interval '30 days' THEN 'current'
					WHEN aggregate_date >= current_date - interval '60 days' THEN 'previous'
				END as period
			FROM
				analytics_visitors_agg_200 ava
			WHERE
				ava.domain_id IN (
					SELECT
						d.domain_id
					FROM
						apps a
					LEFT JOIN
						domains d ON d.app_id = a.app_id
					WHERE
						a.team_id = $1 AND
						a.deleted_at IS NULL AND
						d.domain_id IS NOT NULL
				) AND
				aggregate_date >= current_date - interval '60 days'
		)
		SELECT
			period,
			SUM(total_visitors) as total_visitors
		FROM
			periods
		WHERE 
			period IS NOT NULL
		GROUP BY
			period;
	`,

	totalAppsByTeam: `
		SELECT
			COUNT(*) as total_apps,
    		COUNT(CASE WHEN deleted_at >= CURRENT_DATE - INTERVAL '30 days' THEN 1 END) as deleted_apps,
    		COUNT(CASE WHEN created_at >= CURRENT_DATE - INTERVAL '30 days' THEN 1 END) as new_apps
		FROM
			apps
		WHERE
			team_id = $1 AND deleted_at IS NULL;
	`,

	totalDeploymentsByTeam: `
		SELECT
    		COUNT(*) as total_deployments,
    		COUNT(
				CASE WHEN created_at >= CURRENT_DATE - INTERVAL '30 days' THEN 1 END
			) as current_period,
    		COUNT(
				CASE WHEN
					created_at <= CURRENT_DATE - interval '30 days' AND
					created_at >= CURRENT_DATE - INTERVAL '60 days' THEN 1 END
			) as last_period
		FROM
			deployments d
		WHERE
			d.app_id IN (
				SELECT a.app_id FROM apps a WHERE a.team_id = $1
			);
	`,

	avgDeploymentDurationByTeam: `
		WITH duration_periods AS (
			SELECT
				EXTRACT(EPOCH FROM (stopped_at - created_at)) as seconds,
				CASE
					WHEN created_at >= CURRENT_DATE - INTERVAL '30 days' THEN 'current_period'
					WHEN created_at >= CURRENT_DATE - INTERVAL '60 days' THEN 'previous_period'
				END as period
			FROM deployments
			WHERE
				created_at >= CURRENT_DATE - INTERVAL '60 days'
				AND stopped_at IS NOT NULL AND
				app_id IN (
					SELECT a.app_id FROM apps a WHERE a.team_id = $1
				)
		)
		SELECT
			period,
			AVG(seconds) as avg_seconds
		FROM duration_periods
		WHERE period IS NOT NULL
		GROUP BY period;
	`,

	topPerformingDomains: `
		SELECT
			ava.domain_id,
			d.domain_name,
			SUM(CASE
				WHEN ava.aggregate_date >= current_date - interval '30 days'
				THEN total_visitors
				ELSE 0
			END) AS current_30_days,
			SUM(CASE
				WHEN ava.aggregate_date >= current_date - interval '60 days'
				AND ava.aggregate_date < current_date - interval '30 days'
				THEN total_visitors
				ELSE 0
			END) AS previous_30_days
		FROM
			analytics_visitors_agg_200 ava
		INNER JOIN
			domains d ON d.domain_id = ava.domain_id
		WHERE
			ava.aggregate_date >= current_date - interval '60 days'
			AND EXISTS (
				SELECT 1
				FROM apps a
				INNER JOIN domains d ON d.app_id = a.app_id
				WHERE
					d.domain_id = ava.domain_id
					AND a.team_id = $1
					AND a.deleted_at IS NULL
			)
		GROUP BY
			ava.domain_id, d.domain_name
		ORDER BY
			current_30_days DESC
		LIMIT 100;
	`,
}

// Store handles user logic in the database.
type Store struct {
	*database.Store
}

// NewStore returns a store instance.
func NewStore() *Store {
	return &Store{database.NewStore()}
}

// fallback to localhost if its not
// correct format
func toIP(ipAddress string) null.String {
	ip := net.ParseIP(ipAddress)

	if ip.To4() != nil || ip.To16() != nil {
		return null.NewString(ip.String(), true)
	} else {
		return null.NewString("", false)
	}
}

var refPrefixClean = regexp.MustCompile(`^(https?://)?(www\.)?`)

func cleanNullChars(s null.String) null.String {
	val := s.ValueOrZero()

	if val == "" {
		return s
	}

	return null.StringFrom(strings.ReplaceAll(val, "\x00", ""))
}

func cleanReferrer(referrer null.String) null.String {
	if !referrer.Valid {
		return referrer
	}

	return null.StringFrom(
		// Trim the end `/`
		strings.TrimSuffix(
			// Remove `http://`, `https://` and `www.`
			refPrefixClean.ReplaceAllString(cleanNullChars(referrer).ValueOrZero(), ""),
			"/",
		),
	)
}

// InsertRecords inserts given records into the database.
func (s *Store) InsertRecords(ctx context.Context, records []Record) error {
	params := []any{}
	rows := []map[string]any{}

	// number of fields to
	// be parameterized $1, $2
	insertFieldsSize := 11
	c := 0

	for _, record := range records {
		ip := toIP(record.VisitorIP)

		row := map[string]any{}

		for i := 0; i < insertFieldsSize; i++ {
			row["p"+strconv.Itoa(i+1)] = c + i + 1
		}

		params = append(params,
			record.AppID, record.EnvID, ip,
			record.RequestPath, record.RequestTS.UTC(), record.StatusCode,
			cleanNullChars(record.UserAgent), cleanReferrer(record.Referrer), record.DomainID,
			record.RequestID, record.Source,
		)

		if ip.Valid {
			row["geoLocation"] = true
		}

		rows = append(rows, row)
		c = c + insertFieldsSize
	}

	var wr bytes.Buffer

	data := map[string]any{
		"records": rows,
	}

	fns := template.FuncMap{
		"last": func(x int, a any) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	}

	query := template.Must(template.New("insertRecord").Funcs(fns).Parse(stmt.insertRecord))
	err := query.Execute(&wr, data)

	if err != nil {
		return fmt.Errorf("error while compiling insertRecord template: %v", err)
	}

	_, err = s.Exec(ctx, wr.String(), params...)

	return err
}

const SPAN_24h = "24h"
const SPAN_7D = "7d"
const SPAN_30D = "30d"

type VisitorsArgs struct {
	Span       string
	DomainID   types.ID
	EnvID      types.ID
	StatusCode int
}

func (s *Store) prepareVisitorsQuery(args VisitorsArgs) string {
	data := map[string]any{
		"interval": map[string]string{
			SPAN_24h: "",
			SPAN_7D:  "CURRENT_DATE - INTERVAL '8 days'",
			SPAN_30D: "CURRENT_DATE - INTERVAL '31 days'",
		}[args.Span],
		"aggregateColumn": map[string]string{
			SPAN_24h: "TO_CHAR(DATE_TRUNC('hour', aggregate_date), 'YYYY-MM-DD HH24:MI')",
			SPAN_7D:  "aggregate_date",
			SPAN_30D: "aggregate_date",
		}[args.Span],
		"tableName": map[string]string{
			SPAN_24h: fmt.Sprintf("analytics_visitors_agg_hourly_%d", args.StatusCode),
			SPAN_7D:  fmt.Sprintf("analytics_visitors_agg_%d", args.StatusCode),
			SPAN_30D: fmt.Sprintf("analytics_visitors_agg_%d", args.StatusCode),
		}[args.Span],
	}

	tmpl, err := template.New("visitors").Parse(stmt.visitors)

	if err != nil {
		slog.Errorf("failed parsing visitors template: %s", err.Error())
		return ""
	}

	var wr bytes.Buffer

	if err = tmpl.Execute(&wr, data); err != nil {
		slog.Errorf("failed executing analytics visitors query template: %s", err.Error())
		return ""
	}

	return wr.String()
}

// Visitors returns the number of unique and total vistors for the given environment.
func (s *Store) Visitors(ctx context.Context, args VisitorsArgs) (map[string]any, error) {
	if args.StatusCode != http.StatusOK && args.StatusCode != http.StatusNotFound {
		return nil, fmt.Errorf("invalid status code provided: %d - only 200 and 404 are allowed", args.StatusCode)
	}

	dates := map[string]any{}
	query := s.prepareVisitorsQuery(args)
	rows, err := s.Query(ctx, query, args.DomainID)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	for rows.Next() {
		var date string
		var unique int
		var total int

		if err := rows.Scan(&date, &unique, &total); err != nil {
			slog.Errorf("error while scanning analytics visitors: %s", err.Error())
			return nil, err
		}

		dates[strings.Split(date, "T")[0]] = map[string]int{
			"unique": unique,
			"total":  total,
		}
	}

	// Remove today because the data is not yet synced.
	// Users can use the 24h view to view last day.
	if args.Span != SPAN_24h {
		delete(dates, time.Now().Format(time.DateOnly))
	}

	return dates, nil
}

type TopReferrersArgs struct {
	DomainID      types.ID
	EnvID         types.ID
	NotLikeDomain string
	RequestPath   string
}

// TopReferrers returns the referrer domains ordered by most popular.
func (s *Store) TopReferrers(ctx context.Context, args TopReferrersArgs) (map[string]int, error) {
	refs := map[string]int{}
	params := []any{args.DomainID}
	data := map[string]any{}

	if args.RequestPath != "" {
		params = append(params, args.RequestPath)
		data["requestPath"] = "true"
	}

	tmpl, err := template.New("topReferrers").Parse(stmt.topReferrers)

	if err != nil {
		return nil, err
	}

	var wr bytes.Buffer

	if err = tmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	for rows.Next() {
		var referrer string
		var count int

		if err := rows.Scan(&referrer, &count); err != nil {
			slog.Errorf("[analytics.TopReferrers]: error while scanning %s", err.Error())
			return nil, err
		}

		refs[referrer] = count
	}

	return refs, nil
}

type TopPathsArgs struct {
	EnvID    types.ID
	DomainID types.ID
}

// TopPaths returns the paths that are most visited.
func (s *Store) TopPaths(ctx context.Context, args TopPathsArgs) (map[string]int, error) {
	paths := map[string]int{}
	rows, err := s.Query(ctx, stmt.topPaths, args.DomainID)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	for rows.Next() {
		var path string
		var count int

		if err := rows.Scan(&path, &count); err != nil {
			slog.Errorf("[analytics.TopPaths]: error while scanning %s", err.Error())
			return nil, err
		}

		paths[path] = count
	}

	return paths, nil
}

type ByCountriesArgs struct {
	DomainID types.ID
}

// Countries returns the list of country codes.
func (s *Store) ByCountries(ctx context.Context, args ByCountriesArgs) (map[string]int, error) {
	countries := map[string]int{}
	rows, err := s.Query(ctx, stmt.byCountries, args.DomainID)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	for rows.Next() {
		var country null.String
		var count int

		if err := rows.Scan(&country, &count); err != nil {
			slog.Errorf("[analytics.TopPaths]: error while scanning %s", err.Error())
			return nil, err
		}

		if country.Valid {
			countries[country.ValueOrZero()] = count
		}
	}

	return countries, nil
}

type TotalRequestsByTeam struct {
	CurrentPeriod  int
	PreviousPeriod int
}

// TotalRequestsByTeam returns the total number of requests for a team
// in the current and previous periods (30 days).
func (s *Store) TotalRequestsByTeam(ctx context.Context, teamID types.ID) (*TotalRequestsByTeam, error) {
	rows, err := s.Query(ctx, stmt.totalRequestsByTeam, teamID)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	periods := &TotalRequestsByTeam{
		CurrentPeriod:  0,
		PreviousPeriod: 0,
	}

	for rows.Next() {
		var period string
		var total int

		if err := rows.Scan(&period, &total); err != nil {
			slog.Errorf("[analytics.TotalRequestsByTeam]: error while scanning %s", err.Error())
			return nil, err
		}

		if period == "previous" {
			periods.PreviousPeriod = total
		} else {
			periods.CurrentPeriod = total
		}
	}

	return periods, nil
}

type TotalAppsByTeam struct {
	Total   int
	Deleted int
	New     int
}

// TotalAppsByTeam returns the total number of apps for a team,
// including the number of deleted and new apps in the last 30 days.
func (s *Store) TotalAppsByTeam(ctx context.Context, teamID types.ID) (*TotalAppsByTeam, error) {
	rows, err := s.QueryRow(ctx, stmt.totalAppsByTeam, teamID)

	if err != nil || rows == nil {
		return nil, err
	}

	total := &TotalAppsByTeam{}

	if err := rows.Scan(&total.Total, &total.Deleted, &total.New); err != nil {
		slog.Errorf("[analytics.TotalAppsByTeam]: error while scanning %s", err.Error())
		return nil, err
	}

	return total, nil
}

type TotalDeploymentsByTeam struct {
	Total    int
	Current  int
	Previous int
}

// TotalDeploymentsByTeam returns the total number of deployments for a team,
// including the number of deployments in the current and previous periods (30 days).
func (s *Store) TotalDeploymentsByTeam(ctx context.Context, teamID types.ID) (*TotalDeploymentsByTeam, error) {
	rows, err := s.QueryRow(ctx, stmt.totalDeploymentsByTeam, teamID)

	if err != nil || rows == nil {
		return nil, err
	}

	total := &TotalDeploymentsByTeam{}

	if err := rows.Scan(&total.Total, &total.Current, &total.Previous); err != nil {
		slog.Errorf("[analytics.TotalDeploymentsByTeam]: error while scanning %s", err.Error())
		return nil, err
	}

	return total, nil
}

type AvgDeploymentDurationByTeam struct {
	Current  float64
	Previous float64
}

// AvgDeploymentDurationByTeam returns the average deployment duration for a team
// in the current and previous periods (30 days).
func (s *Store) AvgDeploymentDurationByTeam(ctx context.Context, teamID types.ID) (*AvgDeploymentDurationByTeam, error) {
	rows, err := s.Query(ctx, stmt.avgDeploymentDurationByTeam, teamID)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	periods := &AvgDeploymentDurationByTeam{}

	for rows.Next() {
		var period string
		var avgSeconds float64

		if err := rows.Scan(&period, &avgSeconds); err != nil {
			slog.Errorf("[analytics.AvgDeploymentDurationByTeam]: error while scanning %s", err.Error())
			return nil, err
		}

		switch period {
		case "previous_period":
			periods.Previous = avgSeconds
		case "current_period":
			periods.Current = avgSeconds
		}
	}

	return periods, nil
}

type TopPerformingDomain struct {
	DomainID       types.ID `json:"id"`
	DomainName     string   `json:"domainName"`
	CurrentPeriod  int      `json:"current"`
	PreviousPeriod int      `json:"previous"`
}

// TopPerformingDomains returns the top performing domains for a team
// based on total visitors in the current 30 days period.
func (s *Store) TopPerformingDomains(ctx context.Context, teamID types.ID) ([]*TopPerformingDomain, error) {
	rows, err := s.Query(ctx, stmt.topPerformingDomains, teamID)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	var domains []*TopPerformingDomain

	for rows.Next() {
		domain := &TopPerformingDomain{}

		if err := rows.Scan(&domain.DomainID, &domain.DomainName, &domain.CurrentPeriod, &domain.PreviousPeriod); err != nil {
			slog.Errorf("[analytics.TopPerformingDomains]: error while scanning %s", err.Error())
			return nil, err
		}

		domains = append(domains, domain)
	}

	return domains, nil
}
