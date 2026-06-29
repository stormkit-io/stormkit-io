package analytics

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const eventInsertFields = 9

// InsertEvents batch-inserts custom events. Unset domain_id/visitor_id/
// request_id/metadata are stored as NULL so they do not pollute aggregates.
func (s *Store) InsertEvents(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	params := make([]any, 0, len(events)*eventInsertFields)

	for _, event := range events {
		params = append(params,
			event.AppID,
			event.EnvID,
			event.DomainID,
			event.VisitorID,
			event.EventName,
			event.RequestPath,
			event.EventTS.UTC(),
			event.RequestID,
			event.Metadata,
		)
	}

	var wr bytes.Buffer

	query := template.Must(template.New("insertEvents").
		Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
		Parse(stmt.insertEvents))

	if err := query.Execute(&wr, events); err != nil {
		return fmt.Errorf("error while compiling insertEvents template: %v", err)
	}

	_, err := s.Exec(ctx, wr.String(), params...)

	return err
}

type EventsArgs struct {
	DomainID types.ID
	Span     string
}

// EventCount is the rolled-up count for a single named event over the window.
type EventCount struct {
	Name         string `json:"name"`
	TotalCount   int    `json:"total"`
	UniqueActors int    `json:"unique"`
}

// Events returns custom event counts for a domain over the requested span.
func (s *Store) Events(ctx context.Context, args EventsArgs) ([]EventCount, error) {
	days := map[string]int{
		SPAN_24h: 1,
		SPAN_7D:  7,
		SPAN_30D: 30,
	}[args.Span]

	if days == 0 {
		days = 30
	}

	rows, err := s.Query(ctx, stmt.selectEvents, args.DomainID, days)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	events := []EventCount{}

	for rows.Next() {
		var event EventCount

		if err := rows.Scan(&event.Name, &event.TotalCount, &event.UniqueActors); err != nil {
			slog.Errorf("[analytics.Events]: error while scanning %s", err.Error())
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}
