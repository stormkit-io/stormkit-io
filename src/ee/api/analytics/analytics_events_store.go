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

// spanDays maps an analytics span to a number of days, defaulting to 30.
func spanDays(span string) int {
	days := map[string]int{
		SPAN_24h: 1,
		SPAN_7D:  7,
		SPAN_30D: 30,
	}[span]

	if days == 0 {
		days = 30
	}

	return days
}

// Events returns custom event counts for a domain over the requested span.
func (s *Store) Events(ctx context.Context, args EventsArgs) ([]EventCount, error) {
	rows, err := s.Query(ctx, stmt.selectEvents, args.DomainID, spanDays(args.Span))

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

type EventBreakdownArgs struct {
	DomainID  types.ID
	EventName string
	Property  string
	Span      string
}

// EventBreakdown groups a single event by one metadata property over the span,
// returning the count and unique actors per property value. Reuses EventCount
// where Name holds the property value.
func (s *Store) EventBreakdown(ctx context.Context, args EventBreakdownArgs) ([]EventCount, error) {
	if args.EventName == "" || args.Property == "" {
		return []EventCount{}, nil
	}

	rows, err := s.Query(ctx, stmt.breakdownEvents, args.DomainID, args.EventName, args.Property, spanDays(args.Span))

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	breakdown := []EventCount{}

	for rows.Next() {
		var value EventCount

		if err := rows.Scan(&value.Name, &value.TotalCount, &value.UniqueActors); err != nil {
			slog.Errorf("[analytics.EventBreakdown]: error while scanning %s", err.Error())
			return nil, err
		}

		breakdown = append(breakdown, value)
	}

	return breakdown, nil
}

// EventPropertyKeys returns the distinct metadata property keys seen for an
// event over the span, sampled to keep discovery cheap.
func (s *Store) EventPropertyKeys(ctx context.Context, args EventBreakdownArgs) ([]string, error) {
	if args.EventName == "" {
		return []string{}, nil
	}

	rows, err := s.Query(ctx, stmt.eventPropertyKeys, args.DomainID, args.EventName, spanDays(args.Span))

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if rows == nil {
		return nil, nil
	}

	defer rows.Close()

	keys := []string{}

	for rows.Next() {
		var key string

		if err := rows.Scan(&key); err != nil {
			slog.Errorf("[analytics.EventPropertyKeys]: error while scanning %s", err.Error())
			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}
