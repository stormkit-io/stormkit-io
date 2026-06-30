package buildconf

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var snippetsStmt = struct {
	selectSnippets string
	insertSnippets string
	updateSnippet  string
	deleteSnippet  string
	missingHosts   string
}{
	selectSnippets: `
		SELECT
			s.snippet_id, s.app_id, s.env_id,
			s.snippet_title, s.snippet_content, snippet_location,
			s.should_prepend, s.snippet_rules, s.is_enabled,
			s.snippet_interpolate
		FROM
			snippets s
		WHERE
			{{ .where }}
		ORDER BY s.snippet_id ASC
		LIMIT
			{{ or .limit 25 }};
	`,

	insertSnippets: `
		INSERT INTO snippets (
			app_id, env_id, snippet_title,
			snippet_content, snippet_content_hash,
			snippet_location, should_prepend,
			snippet_rules, is_enabled, snippet_interpolate
		)
		VALUES {{ range $i, $record := .records }}
			(
				${{ $record.p1 }}, ${{ $record.p2 }}, ${{ $record.p3 }},
				${{ $record.p4 }}, ${{ $record.p5 }}, ${{ $record.p6 }},
				${{ $record.p7 }}, ${{ $record.p8 }}, ${{ $record.p9 }},
				${{ $record.p10 }}
			){{ if not (last $i $.records) }}, {{ end }}
		{{ end }}
		RETURNING 
			snippet_id;
	`,

	updateSnippet: `
		WITH update_ts AS (
			UPDATE apps_build_conf e SET updated_at = NOW()
			WHERE e.env_id = $10
		)
		UPDATE snippets SET
			snippet_title = $1,
			snippet_content = $2,
			snippet_content_hash = $3,
			snippet_location = $4,
			snippet_rules = $5,
			should_prepend = $6,
			is_enabled = $7,
			snippet_interpolate = $8
		WHERE
			snippet_id = $9 AND
			env_id = $10;
	`,

	missingHosts: `
		SELECT
			hosts AS missing_host
		FROM
			unnest($1::text[]) AS hosts
		LEFT JOIN skitapi.domains d
			ON hosts = d.domain_name AND d.env_id = $2
		WHERE
			d.domain_name IS NULL;
	`,

	deleteSnippet: `
		DELETE FROM snippets d WHERE d.snippet_id = ANY($1) AND d.env_id = $2;
	`,
}

// Store represents a store for the deployments and deployment logs.
type SStore struct {
	*database.Store
	selectTmpl *template.Template
}

// NewStore returns a store instance.
func SnippetsStore() *SStore {
	tmpl, err := template.New("selectSnippets").Parse(snippetsStmt.selectSnippets)

	if err != nil {
		panic(err)
	}

	return &SStore{
		Store:      database.NewStore(),
		selectTmpl: tmpl,
	}
}

func (s *SStore) selectSnippet(ctx context.Context, data map[string]any, params ...any) (*Snippet, error) {
	var wr bytes.Buffer

	snippet := &Snippet{}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	row, err := s.QueryRow(ctx, wr.String(), params...)

	if err != nil {
		return nil, err
	}

	err = row.Scan(
		&snippet.ID, &snippet.AppID, &snippet.EnvID,
		&snippet.Title, &snippet.Content, &snippet.Location,
		&snippet.Prepend, &snippet.Rules, &snippet.Enabled,
		&snippet.Interpolate,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	return snippet, err
}

// SnippetByID returns a snippet by it's ID.
func (s *SStore) SnippetByID(ctx context.Context, snippetID types.ID) (*Snippet, error) {
	data := map[string]any{
		"where": "s.snippet_id = $1",
	}

	return s.selectSnippet(ctx, data, snippetID)
}

type SnippetFilters struct {
	EnvID   types.ID
	AfterID types.ID
	Hosts   []string
	Title   string
	Limit   int
}

// SnippetsByEnvID returns a list of snippets by their environment id.
func (s *SStore) SnippetsByEnvID(ctx context.Context, filters SnippetFilters) ([]*Snippet, error) {
	var wr bytes.Buffer

	where := []string{"s.env_id = $1"}
	params := []any{filters.EnvID}

	if len(filters.Hosts) > 0 {
		where = append(where, "(snippet_rules->'hosts')::JSONB @> $2::JSONB")
		jsonb, err := json.Marshal(filters.Hosts)

		if err != nil {
			return nil, err
		}

		params = append(params, string(jsonb))
	}

	if filters.AfterID > 0 {
		params = append(params, filters.AfterID)
		where = append(where, fmt.Sprintf("s.snippet_id > $%d", len(params)))
	}

	if filters.Limit == 0 {
		filters.Limit = 50
	}

	if filters.Title != "" {
		params = append(params, filters.Title)
		where = append(where, fmt.Sprintf("s.snippet_title = $%d", len(params)))
	}

	data := map[string]any{
		"where": strings.Join(where, " AND "),
		"limit": filters.Limit + 1,
	}

	if err := s.selectTmpl.Execute(&wr, data); err != nil {
		return nil, err
	}

	snippets := []*Snippet{}

	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil || rows == nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		snippet := &Snippet{}

		err := rows.Scan(
			&snippet.ID, &snippet.AppID, &snippet.EnvID,
			&snippet.Title, &snippet.Content, &snippet.Location,
			&snippet.Prepend, &snippet.Rules, &snippet.Enabled,
			&snippet.Interpolate,
		)

		if err != nil {
			return nil, err
		}

		snippets = append(snippets, snippet)
	}

	return snippets, nil
}

// Insert inserts the given snippet records into the database.
func (s *SStore) Insert(ctx context.Context, snippets []*Snippet) error {
	params := []any{}
	records := []map[string]any{}
	data := map[string]any{}

	// number of fields to be parameterized $1, $2, etc...
	insertFieldsSize := 10
	c := 0

	for _, snippet := range snippets {
		record := map[string]any{}

		for i := 0; i < insertFieldsSize; i++ {
			record["p"+strconv.Itoa(i+1)] = c + i + 1
		}

		params = append(params,
			snippet.AppID, snippet.EnvID, snippet.Title,
			snippet.Content, snippet.ContentHash(),
			snippet.Location, snippet.Prepend,
			snippet.Rules, snippet.Enabled, snippet.Interpolate,
		)

		records = append(records, record)
		c = c + insertFieldsSize
	}

	var wr bytes.Buffer

	data["records"] = records

	fns := template.FuncMap{
		"last": func(x int, a any) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	}

	query := template.Must(template.New("insertSnippets").
		Funcs(fns).
		Parse(snippetsStmt.insertSnippets))

	if err := query.Execute(&wr, data); err != nil {
		return err
	}

	rows, err := s.Query(ctx, wr.String(), params...)

	if err != nil || rows == nil {
		return err
	}

	defer rows.Close()

	i := 0

	for rows.Next() {
		if err := rows.Scan(&snippets[i].ID); err != nil {
			return err
		}

		i = i + 1
	}

	return err
}

// Update updates the given snippet records in the database.
func (s *SStore) Update(ctx context.Context, snippet *Snippet) error {
	params := []any{
		snippet.Title, snippet.Content, snippet.ContentHash(),
		snippet.Location, snippet.Rules, snippet.Prepend, snippet.Enabled,
		snippet.Interpolate, snippet.ID, snippet.EnvID,
	}

	_, err := s.Exec(ctx, snippetsStmt.updateSnippet, params...)
	return err
}

// Delete the provided snippets from the database.
func (s *SStore) Delete(ctx context.Context, snippetIDs []types.ID, envID types.ID) error {
	_, err := s.Exec(ctx, snippetsStmt.deleteSnippet, pq.Array(snippetIDs), envID)
	return err
}

// MissingHosts returns the list of missing hosts for the given hosts slice.
func (s *SStore) MissingHosts(ctx context.Context, hosts []string, envID types.ID) ([]string, error) {
	rows, err := s.Query(ctx, snippetsStmt.missingHosts, pq.Array(hosts), envID)

	if err == sql.ErrNoRows || rows == nil {
		return nil, nil
	}

	defer rows.Close()

	missingHosts := []string{}

	for rows.Next() {
		var missingHost string

		if err := rows.Scan(&missingHost); err != nil {
			return nil, err
		}

		missingHosts = append(missingHosts, missingHost)
	}

	return missingHosts, nil
}
