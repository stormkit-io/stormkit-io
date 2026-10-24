package analytics

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// MaxEngagementSeconds caps one engagement report, so a page left in the
// foreground for hours does not skew the averages.
const MaxEngagementSeconds = 30 * 60

// Engagement is the foreground time the client script reported for a page view.
type Engagement struct {
	DomainID    types.ID
	RequestPath string
	// ViewID identifies the page view, which reports each time it is hidden.
	ViewID    string
	Seconds   int
	EngagedTS utils.Unix
}

const (
	engagementInsertFields    = 5
	engagementInsertChunkSize = 1000
)

var insertEngagementsTemplate = template.Must(template.New("insertEngagements").
	Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
	Parse(`
		INSERT INTO analytics_engagement (engagement_ts, domain_id, request_path, view_id, seconds)
		VALUES {{ generateValues 5 (len .) }};
	`))

// InsertEngagements inserts the engagements in chunks, keeping each statement
// below Postgres's bind parameter limit.
func (s *Store) InsertEngagements(ctx context.Context, engagements []Engagement) error {
	for start := 0; start < len(engagements); start += engagementInsertChunkSize {
		chunk := engagements[start:min(start+engagementInsertChunkSize, len(engagements))]
		params := make([]any, 0, len(chunk)*engagementInsertFields)

		for _, e := range chunk {
			params = append(params, e.EngagedTS.UTC(), e.DomainID, e.RequestPath, e.ViewID, e.Seconds)
		}

		var sql bytes.Buffer

		if err := insertEngagementsTemplate.Execute(&sql, chunk); err != nil {
			return fmt.Errorf("error while compiling insertEngagements template: %v", err)
		}

		if _, err := s.Exec(ctx, sql.String(), params...); err != nil {
			return err
		}
	}

	return nil
}
