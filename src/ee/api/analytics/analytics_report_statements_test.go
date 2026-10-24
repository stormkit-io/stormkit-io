package analytics

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ReportStatementsSuite struct {
	suite.Suite
}

var placeholder = regexp.MustCompile(`\$(\d+)`)

func mustDate(v string) time.Time {
	t, err := time.Parse(time.DateOnly, v)

	if err != nil {
		panic(err)
	}

	return t
}

func (s *ReportStatementsSuite) query() reportQuery {
	r := &reporter{params: ReportParams{
		Metric:     MetricReferrers,
		From:       mustDate("2026-10-08"),
		To:         mustDate("2026-10-14"),
		PathPrefix: "/blog/'; DROP TABLE analytics; --",
		Limit:      7,
	}, from: mustDate("2026-10-01")}

	return r.query()
}

// Test_ValuesAreBound verifies that values reach the query only as arguments,
// numbered in order with none skipped.
func (s *ReportStatementsSuite) Test_ValuesAreBound() {
	for _, name := range []string{
		"visitorsSeries", "siteVisitors", "paths", "pathsTotals", "channels",
		"referrers", "entries", "countries", "campaigns", "campaignsTotals",
		"events", "eventsTotals",
	} {
		sql, args, err := renderReportQuery(name, s.query())

		s.Require().NoError(err, name)
		s.NotContains(sql, "DROP TABLE", name)
		s.NotContains(sql, "2026-10-01", name)
		s.NotContains(sql, "2026-10-08", name)

		seen := map[string]bool{}

		for _, m := range placeholder.FindAllStringSubmatch(sql, -1) {
			seen[m[1]] = true
		}

		s.Len(seen, len(args), "%s: every argument has a placeholder", name)
	}
}

func (s *ReportStatementsSuite) Test_LimitIsBound() {
	sql, args, err := renderReportQuery("paths", s.query())

	s.Require().NoError(err)
	s.Contains(sql, "LIMIT $")
	s.Contains(args, 7)
}

func (s *ReportStatementsSuite) Test_SourceSelectsViewsColumn() {
	q := s.query()

	for source, column := range map[string]string{"": "page_views", "server": "server_views", "client": "client_views"} {
		q.Source = source
		sql, _, err := renderReportQuery("pathsTotals", q)

		s.Require().NoError(err)
		s.Contains(sql, "SUM("+column+")", source)
	}
}

func (s *ReportStatementsSuite) Test_ChannelsReadStoredChannels() {
	for _, name := range []string{"channels", "referrers", "entries"} {
		sql, _, err := renderReportQuery(name, s.query())

		s.Require().NoError(err)
		s.Contains(sql, "FROM analytics_channels_agg", name)
		s.NotContains(sql, "~", "%s classifies nothing at read time", name)
	}
}

func TestReportStatementsSuite(t *testing.T) {
	suite.Run(t, new(ReportStatementsSuite))
}
