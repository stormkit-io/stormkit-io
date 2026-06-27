package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

const insertEventsColumns = `
	INSERT INTO analytics_events
		(app_id, env_id, domain_id, visitor_id, event_name, request_path, event_ts, request_id, metadata)
	VALUES `

// selectEvents aggregates rolled-up event counts for a domain over a window.
// NOTE: unique_actors is summed across daily aggregates, so a visitor active on
// multiple days is counted once per day (i.e. unique actor-days, not unique
// actors over the whole window).
// NOTE: event_name cardinality is client-controlled, so the result is capped at
// the 100 highest-count events; lower-count events are omitted from the window.
const selectEvents = `
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
	LIMIT 100`

const eventInsertFields = 9

// InsertEvents batch-inserts custom events. Unset domain_id/visitor_id/
// request_id/metadata are stored as NULL so they do not pollute aggregates.
func (s *Store) InsertEvents(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}

	params := make([]any, 0, len(events)*eventInsertFields)
	values := make([]string, 0, len(events))

	for i, event := range events {
		base := i * eventInsertFields

		values = append(values, fmt.Sprintf(
			"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d::uuid, $%d::jsonb)",
			base+1, base+2, base+3, base+4, base+5, base+6, base+7, base+8, base+9,
		))

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

	_, err := s.Exec(ctx, insertEventsColumns+strings.Join(values, ", "), params...)

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

	rows, err := s.Query(ctx, selectEvents, args.DomainID, days)

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
