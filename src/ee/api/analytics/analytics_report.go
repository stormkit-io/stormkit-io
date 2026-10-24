package analytics

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const (
	MetricVisitors  = "visitors"
	MetricPaths     = "paths"
	MetricReferrers = "referrers"
	MetricCountries = "countries"
	MetricCampaigns = "campaigns"
	MetricEvents    = "events"

	SourceServer = "server"
	SourceClient = "client"

	// ReportDefaultDays is the range a report covers when none is given.
	ReportDefaultDays = 30

	// ReportMaxDays bounds a report's range; it leaves room for a
	// year-over-year comparison.
	ReportMaxDays = 400

	ReportDefaultLimit = 10
	ReportMaxLimit     = 50
)

var reportMetrics = []string{MetricVisitors, MetricPaths, MetricReferrers, MetricCountries, MetricCampaigns, MetricEvents}

// Notes explaining how a report's numbers are counted, returned alongside them
// so an agent or a person reading the raw response does not misread them.
const (
	noteDailyUniques   = "visitors is the sum of daily unique visitors: visitor ids rotate daily, so the same person on two days counts twice."
	notePageVisitors   = "pageVisitors sums each matching page's daily unique visitors: a person reading three matching pages on one day counts three times. Use an exact path for unique visitors."
	noteSourceViews    = "source filters page views only; visitor counts include both sources."
	noteRawReferrers   = "referrers are host names as sent by browsers, and include this site's own domain for in-site navigation."
	noteNoDomains      = "this environment has no custom domains; analytics are only recorded for custom domains."
	noteCampaignVisits = "campaigns are ranked by visits; visitors are only reported with an exact path, since a person landing on two pages from one campaign would otherwise count twice."
	notePathlessEvents = "events are counted for the whole site; path filters do not apply."
)

// ReportParams selects what a report covers. Use ReportParamsFromQuery to
// build and validate one from a request's query string.
type ReportParams struct {
	Metric     string
	DomainIDs  []types.ID
	From       time.Time
	To         time.Time
	Path       string
	PathPrefix string
	Source     string
	Compare    bool
	Limit      int
}

// ReportRange is an inclusive range of UTC days.
type ReportRange struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
}

// ReportDay is one day of a visitors report.
type ReportDay struct {
	Date         string `json:"date"`
	PageViews    int64  `json:"pageViews"`
	Visitors     *int64 `json:"visitors,omitempty"`
	PageVisitors *int64 `json:"pageVisitors,omitempty"`
}

// ReportRow is one row of a ranked report. Only the fields relevant to the
// metric are set.
type ReportRow struct {
	Path        string `json:"path,omitempty"`
	Referrer    string `json:"referrer,omitempty"`
	Country     string `json:"country,omitempty"`
	UTMSource   string `json:"utmSource,omitempty"`
	UTMMedium   string `json:"utmMedium,omitempty"`
	UTMCampaign string `json:"utmCampaign,omitempty"`
	Ref         string `json:"ref,omitempty"`
	Event       string `json:"event,omitempty"`

	PageViews    *int64 `json:"pageViews,omitempty"`
	Visits       *int64 `json:"visits,omitempty"`
	Visitors     *int64 `json:"visitors,omitempty"`
	PageVisitors *int64 `json:"pageVisitors,omitempty"`
	Count        *int64 `json:"count,omitempty"`

	// Previous is the row's main number in the previous period, set when the
	// report compares periods.
	Previous *int64 `json:"previous,omitempty"`

	key string
}

// ReportTotals sums a report across its rows or days.
type ReportTotals struct {
	PageViews    *int64 `json:"pageViews,omitempty"`
	Visitors     *int64 `json:"visitors,omitempty"`
	PageVisitors *int64 `json:"pageVisitors,omitempty"`
	Visits       *int64 `json:"visits,omitempty"`
	Direct       *int64 `json:"direct,omitempty"`
	Count        *int64 `json:"count,omitempty"`
}

// ReportPeriod is the previous period of a compared report.
type ReportPeriod struct {
	Range  ReportRange  `json:"range"`
	Totals ReportTotals `json:"totals"`
}

// Report is the result of a report query.
type Report struct {
	Metric   string        `json:"metric"`
	Range    ReportRange   `json:"range"`
	Totals   ReportTotals  `json:"totals"`
	Series   []ReportDay   `json:"series,omitempty"`
	Rows     []ReportRow   `json:"rows,omitempty"`
	Previous *ReportPeriod `json:"previous,omitempty"`
	Notes    []string      `json:"notes,omitempty"`
}

// ReportParamsFromQuery validates a report's query string. The caller sets
// DomainIDs. It returns the validation errors, if any.
func ReportParamsFromQuery(q url.Values) (ReportParams, []string) {
	errs := []string{}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	p := ReportParams{
		Metric:     q.Get("metric"),
		Path:       q.Get("path"),
		PathPrefix: q.Get("pathPrefix"),
		Source:     q.Get("source"),
		Compare:    q.Get("compare") == "previous",
		Limit:      ReportDefaultLimit,
		To:         today,
	}

	if p.Metric == "" {
		p.Metric = MetricVisitors
	}

	if !slices.Contains(reportMetrics, p.Metric) {
		errs = append(errs, "metric must be one of: "+strings.Join(reportMetrics, ", "))
	}

	if p.Path != "" && p.PathPrefix != "" {
		errs = append(errs, "path and pathPrefix cannot be combined")
	}

	for _, v := range []string{p.Path, p.PathPrefix} {
		if v != "" && !strings.HasPrefix(v, "/") {
			errs = append(errs, "path and pathPrefix must start with '/'")
			break
		}
	}

	if p.Source != "" && p.Source != SourceServer && p.Source != SourceClient {
		errs = append(errs, "source must be server or client")
	}

	if c := q.Get("compare"); c != "" && c != "previous" {
		errs = append(errs, "compare must be 'previous'")
	}

	if limit := q.Get("limit"); limit != "" {
		p.Limit = utils.StringToInt(limit)

		if p.Limit < 1 || p.Limit > ReportMaxLimit {
			errs = append(errs, fmt.Sprintf("limit must be between 1 and %d", ReportMaxLimit))
		}
	}

	days := ReportDefaultDays

	if span := q.Get("span"); span != "" {
		days = utils.StringToInt(strings.TrimSuffix(span, "d"))

		if !strings.HasSuffix(span, "d") || days < 1 || days > ReportMaxDays {
			errs = append(errs, fmt.Sprintf("span must be a number of days between 1d and %dd, e.g. 90d", ReportMaxDays))
		}
	}

	if to := q.Get("to"); to != "" {
		t, err := time.Parse(time.DateOnly, to)

		if err != nil {
			errs = append(errs, "to must be a date in YYYY-MM-DD format")
		}

		p.To = t
	}

	p.From = p.To.AddDate(0, 0, -(days - 1))

	if from := q.Get("from"); from != "" {
		if q.Get("span") != "" {
			errs = append(errs, "from and span cannot be combined")
		}

		t, err := time.Parse(time.DateOnly, from)

		if err != nil {
			errs = append(errs, "from must be a date in YYYY-MM-DD format")
		}

		p.From = t
	}

	if len(errs) == 0 {
		if p.From.After(p.To) {
			errs = append(errs, "from must not be after to")
		} else if rangeDays(p.From, p.To) > ReportMaxDays {
			errs = append(errs, fmt.Sprintf("a report covers at most %d days", ReportMaxDays))
		}
	}

	return p, errs
}

func rangeDays(from, to time.Time) int {
	return int(to.Sub(from).Hours()/24) + 1
}

func newReportRange(from, to time.Time) ReportRange {
	return ReportRange{
		From: from.Format(time.DateOnly),
		To:   to.Format(time.DateOnly),
		Days: rangeDays(from, to),
	}
}

var errUnknownMetric = errors.New("unknown metric")

// Report runs the report the params describe. Every number comes from the
// daily aggregate tables, so a report never scans raw page views.
func (s *Store) Report(ctx context.Context, p ReportParams) (*Report, error) {
	report := &Report{
		Metric: p.Metric,
		Range:  newReportRange(p.From, p.To),
	}

	if len(p.DomainIDs) == 0 {
		report.Notes = append(report.Notes, noteNoDomains)
		return report, nil
	}

	r := &reporter{store: s, params: p}

	current, err := r.run(ctx, p.From, p.To)

	if err != nil {
		return nil, err
	}

	report.Totals = current.Totals
	report.Series = current.Series
	report.Rows = current.Rows
	report.Notes = r.notes()

	if p.Compare {
		days := rangeDays(p.From, p.To)
		prevTo := p.From.AddDate(0, 0, -1)
		prevFrom := prevTo.AddDate(0, 0, -(days - 1))
		// The previous period is looked up for the current rows' keys rather
		// than its own top N, so a row that ranked lower then still gets its
		// number, without reading every row of the previous period.
		keys := make([]string, 0, len(current.Rows))

		for _, row := range current.Rows {
			keys = append(keys, row.key)
		}

		prevReporter := &reporter{store: s, params: p, keys: keys}
		previous, err := prevReporter.run(ctx, prevFrom, prevTo)

		if err != nil {
			return nil, err
		}

		report.Previous = &ReportPeriod{Range: newReportRange(prevFrom, prevTo), Totals: previous.Totals}

		byKey := map[string]*int64{}

		for _, row := range previous.Rows {
			byKey[row.key] = row.main()
		}

		for i := range report.Rows {
			prev := int64(0)

			if v := byKey[report.Rows[i].key]; v != nil {
				prev = *v
			}

			report.Rows[i].Previous = &prev
		}
	}

	return report, nil
}

// main returns the number a row is ranked by.
func (row ReportRow) main() *int64 {
	for _, v := range []*int64{row.PageViews, row.Visits, row.Count, row.Visitors, row.PageVisitors} {
		if v != nil {
			return v
		}
	}

	return nil
}

// reporter builds the SQL for one report and runs it for a range of days.
type reporter struct {
	store  *Store
	params ReportParams

	// keys restricts ranked rows to these keys, without a limit. It is set when
	// looking up the previous period of a compared report.
	keys []string
}

type reportResult struct {
	Totals ReportTotals
	Series []ReportDay
	Rows   []ReportRow
}

func (r *reporter) notes() []string {
	notes := []string{}

	switch r.params.Metric {
	case MetricVisitors, MetricPaths:
		if r.params.PathPrefix != "" && r.params.Metric == MetricVisitors {
			notes = append(notes, notePageVisitors)
		} else {
			notes = append(notes, noteDailyUniques)
		}

		if r.params.Source != "" {
			notes = append(notes, noteSourceViews)
		}
	case MetricReferrers:
		notes = append(notes, noteRawReferrers)
	case MetricCountries:
		if r.params.PathPrefix != "" {
			notes = append(notes, notePageVisitors)
		} else {
			notes = append(notes, noteDailyUniques)
		}
	case MetricCampaigns:
		if r.params.Path != "" {
			notes = append(notes, noteDailyUniques)
		} else {
			notes = append(notes, noteCampaignVisits)
		}
	case MetricEvents:
		if r.params.Path != "" || r.params.PathPrefix != "" {
			notes = append(notes, notePathlessEvents)
		}
	}

	return notes
}

func (r *reporter) run(ctx context.Context, from, to time.Time) (*reportResult, error) {
	switch r.params.Metric {
	case MetricVisitors:
		return r.visitors(ctx, from, to)
	case MetricPaths:
		return r.paths(ctx, from, to)
	case MetricReferrers:
		return r.referrers(ctx, from, to)
	case MetricCountries:
		return r.countries(ctx, from, to)
	case MetricCampaigns:
		return r.campaigns(ctx, from, to)
	case MetricEvents:
		return r.events(ctx, from, to)
	}

	return nil, errUnknownMetric
}

// args returns the bind parameters shared by every report query: the domains
// ($1), the range ($2, $3) and the path filter ($4) when there is one.
func (r *reporter) args(from, to time.Time) []any {
	ids := make([]int64, len(r.params.DomainIDs))

	for i, id := range r.params.DomainIDs {
		ids[i] = int64(id)
	}

	args := []any{pq.Array(ids), from.Format(time.DateOnly), to.Format(time.DateOnly)}

	if v := r.pathValue(); v != "" {
		args = append(args, v)
	}

	return args
}

func (r *reporter) pathValue() string {
	if r.params.Path != "" {
		return r.params.Path
	}

	return r.params.PathPrefix
}

// pathFilter returns the SQL condition for the path filter ($4), or "".
func (r *reporter) pathFilter() string {
	if r.params.Path != "" {
		return "AND request_path_hash = decode(md5($4), 'hex')"
	}

	if r.params.PathPrefix != "" {
		return "AND left(request_path, length($4)) = $4"
	}

	return ""
}

func (r *reporter) where() string {
	return "domain_id = ANY($1::bigint[]) AND aggregate_date BETWEEN $2::date AND $3::date " + r.pathFilter()
}

// viewsColumn is the page-view count the source filter selects.
func (r *reporter) viewsColumn() string {
	switch r.params.Source {
	case SourceServer:
		return "server_views"
	case SourceClient:
		return "client_views"
	}

	return "page_views"
}

// keyFilter returns the SQL condition restricting column to the reporter's
// keys, bound at the parameter after the shared ones, or "".
func (r *reporter) keyFilter(column string) string {
	if r.keys == nil {
		return ""
	}

	index := 4

	if r.pathValue() != "" {
		index = 5
	}

	return fmt.Sprintf("AND %s = ANY($%d::text[])", column, index)
}

// rankedArgs returns args plus the keys, when the reporter has them.
func (r *reporter) rankedArgs(from, to time.Time) []any {
	args := r.args(from, to)

	if r.keys != nil {
		args = append(args, pq.Array(r.keys))
	}

	return args
}

func (r *reporter) limit() string {
	if r.keys != nil {
		return ""
	}

	return fmt.Sprintf("LIMIT %d", r.params.Limit)
}

func (r *reporter) visitors(ctx context.Context, from, to time.Time) (*reportResult, error) {
	query := fmt.Sprintf(`
		SELECT aggregate_date::text, SUM(%s), SUM(unique_visitors)
		FROM analytics_paths_agg
		WHERE %s
		GROUP BY aggregate_date
		ORDER BY aggregate_date`, r.viewsColumn(), r.where())

	rows, err := r.store.Query(ctx, query, r.args(from, to)...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	byDate := map[string]*ReportDay{}

	for rows.Next() {
		var day ReportDay
		var uniques int64

		if err := rows.Scan(&day.Date, &day.PageViews, &uniques); err != nil {
			return nil, err
		}

		switch {
		case r.params.Path != "":
			day.Visitors = &uniques
		case r.params.PathPrefix != "":
			day.PageVisitors = &uniques
		}

		byDate[day.Date] = &day
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Site-wide unique visitors cannot be summed from per-path rows, which
	// count a visitor once per page; the domain-level aggregate holds them.
	if r.params.Path == "" && r.params.PathPrefix == "" {
		if err := r.siteVisitors(ctx, from, to, byDate); err != nil {
			return nil, err
		}
	}

	result := &reportResult{Series: []ReportDay{}}
	totals := ReportDay{}

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		date := d.Format(time.DateOnly)
		day, ok := byDate[date]

		if !ok {
			day = r.emptyDay(date)
		}

		if r.pathValue() == "" && day.Visitors == nil {
			zero := int64(0)
			day.Visitors = &zero
		}

		totals.PageViews += day.PageViews
		totals.Visitors = addPtr(totals.Visitors, day.Visitors)
		totals.PageVisitors = addPtr(totals.PageVisitors, day.PageVisitors)
		result.Series = append(result.Series, *day)
	}

	result.Totals = ReportTotals{PageViews: &totals.PageViews, Visitors: totals.Visitors, PageVisitors: totals.PageVisitors}

	return result, nil
}

// emptyDay returns a day with no traffic, carrying zeroes for the counts the
// report's filters produce.
func (r *reporter) emptyDay(date string) *ReportDay {
	zero := int64(0)
	day := &ReportDay{Date: date}

	if r.params.PathPrefix != "" {
		day.PageVisitors = &zero
	} else {
		day.Visitors = &zero
	}

	return day
}

func (r *reporter) siteVisitors(ctx context.Context, from, to time.Time, byDate map[string]*ReportDay) error {
	rows, err := r.store.Query(ctx, `
		SELECT aggregate_date::text, SUM(unique_visitors)
		FROM analytics_visitors_agg_200
		WHERE domain_id = ANY($1::bigint[]) AND aggregate_date BETWEEN $2::date AND $3::date
		GROUP BY aggregate_date`, r.args(from, to)...)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		var date string
		var uniques int64

		if err := rows.Scan(&date, &uniques); err != nil {
			return err
		}

		day, ok := byDate[date]

		if !ok {
			day = &ReportDay{Date: date}
			byDate[date] = day
		}

		day.Visitors = &uniques
	}

	return rows.Err()
}

func addPtr(sum, v *int64) *int64 {
	if v == nil {
		return sum
	}

	total := *v

	if sum != nil {
		total += *sum
	}

	return &total
}

// ranked runs a ranked query whose rows are (key columns..., numbers...) and
// totals the numbers.
type rankedQuery struct {
	sql  string
	scan func(scan func(dest ...any) error) (ReportRow, error)
}

func (r *reporter) ranked(ctx context.Context, q rankedQuery, from, to time.Time) ([]ReportRow, error) {
	rows, err := r.store.Query(ctx, q.sql, r.rankedArgs(from, to)...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	out := []ReportRow{}

	for rows.Next() {
		row, err := q.scan(rows.Scan)

		if err != nil {
			return nil, err
		}

		out = append(out, row)
	}

	return out, rows.Err()
}

// total sums a row field across a filtered table, independent of the limit.
func (r *reporter) total(ctx context.Context, sql string, from, to time.Time) (*int64, error) {
	row, err := r.store.QueryRow(ctx, sql, r.args(from, to)...)

	if err != nil {
		return nil, err
	}

	var total int64

	if err := row.Scan(&total); err != nil {
		return nil, err
	}

	return &total, nil
}

func (r *reporter) paths(ctx context.Context, from, to time.Time) (*reportResult, error) {
	views := r.viewsColumn()
	rows, err := r.ranked(ctx, rankedQuery{
		sql: fmt.Sprintf(`
			SELECT request_path, SUM(%s) AS views, SUM(unique_visitors)
			FROM analytics_paths_agg
			WHERE %s %s
			GROUP BY request_path
			HAVING SUM(%s) > 0
			ORDER BY views DESC, request_path
			%s`, views, r.where(), r.keyFilter("request_path"), views, r.limit()),
		scan: func(scan func(dest ...any) error) (ReportRow, error) {
			var row ReportRow
			var pv, uv int64
			err := scan(&row.Path, &pv, &uv)
			row.PageViews, row.Visitors, row.key = &pv, &uv, row.Path
			return row, err
		},
	}, from, to)

	if err != nil {
		return nil, err
	}

	total, err := r.total(ctx, fmt.Sprintf(`SELECT COALESCE(SUM(%s), 0) FROM analytics_paths_agg WHERE %s`, views, r.where()), from, to)

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: rows, Totals: ReportTotals{PageViews: total}}, nil
}

func (r *reporter) referrers(ctx context.Context, from, to time.Time) (*reportResult, error) {
	rows, err := r.ranked(ctx, rankedQuery{
		sql: fmt.Sprintf(`
			SELECT referrer, SUM(visit_count) AS visits
			FROM analytics_referrers
			WHERE %s AND referrer <> '' %s
			GROUP BY referrer
			ORDER BY visits DESC, referrer
			%s`, r.where(), r.keyFilter("referrer"), r.limit()),
		scan: func(scan func(dest ...any) error) (ReportRow, error) {
			var row ReportRow
			var visits int64
			err := scan(&row.Referrer, &visits)
			row.Visits, row.key = &visits, row.Referrer
			return row, err
		},
	}, from, to)

	if err != nil {
		return nil, err
	}

	total, err := r.total(ctx, `SELECT COALESCE(SUM(visit_count), 0) FROM analytics_referrers WHERE `+r.where(), from, to)

	if err != nil {
		return nil, err
	}

	direct, err := r.total(ctx, `SELECT COALESCE(SUM(visit_count), 0) FROM analytics_referrers WHERE referrer = '' AND `+r.where(), from, to)

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: rows, Totals: ReportTotals{Visits: total, Direct: direct}}, nil
}

func (r *reporter) countries(ctx context.Context, from, to time.Time) (*reportResult, error) {
	table, column := "analytics_visitors_by_countries", "visit_count"

	if r.pathValue() != "" {
		table, column = "analytics_path_countries_agg", "unique_visitors"
	}

	rows, err := r.ranked(ctx, rankedQuery{
		sql: fmt.Sprintf(`
			SELECT country_iso_code, SUM(%s) AS visitors
			FROM %s
			WHERE %s %s
			GROUP BY country_iso_code
			ORDER BY visitors DESC, country_iso_code
			%s`, column, table, r.where(), r.keyFilter("country_iso_code"), r.limit()),
		scan: func(scan func(dest ...any) error) (ReportRow, error) {
			var row ReportRow
			var visitors int64
			err := scan(&row.Country, &visitors)
			row.key = row.Country

			// Per-page rows count a reader of several matching pages once per
			// page, so a prefix reports them under their own name.
			if r.params.PathPrefix != "" {
				row.PageVisitors = &visitors
			} else {
				row.Visitors = &visitors
			}

			return row, err
		},
	}, from, to)

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: rows}, nil
}

func (r *reporter) campaigns(ctx context.Context, from, to time.Time) (*reportResult, error) {
	rows, err := r.ranked(ctx, rankedQuery{
		sql: fmt.Sprintf(`
			SELECT utm_source, utm_medium, utm_campaign, ref, SUM(visits) AS visits, SUM(unique_visitors)
			FROM analytics_campaigns_agg
			WHERE %s %s
			GROUP BY utm_source, utm_medium, utm_campaign, ref
			ORDER BY visits DESC, utm_source, utm_medium, utm_campaign, ref
			%s`, r.where(), r.keyFilter("concat_ws(chr(31), utm_source, utm_medium, utm_campaign, ref)"), r.limit()),
		scan: func(scan func(dest ...any) error) (ReportRow, error) {
			var row ReportRow
			var visits, visitors int64
			err := scan(&row.UTMSource, &row.UTMMedium, &row.UTMCampaign, &row.Ref, &visits, &visitors)
			row.Visits = &visits

			// Visitors are summed across landing pages, so they are only
			// unique when a single landing path is selected.
			if r.params.Path != "" {
				row.Visitors = &visitors
			}

			row.key = strings.Join([]string{row.UTMSource, row.UTMMedium, row.UTMCampaign, row.Ref}, "\x1f")
			return row, err
		},
	}, from, to)

	if err != nil {
		return nil, err
	}

	total, err := r.total(ctx, `SELECT COALESCE(SUM(visits), 0) FROM analytics_campaigns_agg WHERE `+r.where(), from, to)

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: rows, Totals: ReportTotals{Visits: total}}, nil
}

func (r *reporter) events(ctx context.Context, from, to time.Time) (*reportResult, error) {
	// The events aggregate has no path, so the path filter is not applied.
	where := "domain_id = ANY($1::bigint[]) AND aggregate_date BETWEEN $2::date AND $3::date"
	args := r.args(from, to)[:3]

	keyFilter := ""
	rowArgs := args

	if r.keys != nil {
		keyFilter = "AND event_name = ANY($4::text[])"
		rowArgs = append(append([]any{}, args...), pq.Array(r.keys))
	}

	query := fmt.Sprintf(`
		SELECT event_name, SUM(total_count) AS total, SUM(unique_actors)
		FROM analytics_events_agg
		WHERE %s %s
		GROUP BY event_name
		ORDER BY total DESC, event_name
		%s`, where, keyFilter, r.limit())

	rows, err := r.store.Query(ctx, query, rowArgs...)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	out := []ReportRow{}

	for rows.Next() {
		var row ReportRow
		var count, actors int64

		if err := rows.Scan(&row.Event, &count, &actors); err != nil {
			return nil, err
		}

		row.Count, row.Visitors, row.key = &count, &actors, row.Event
		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalRow, err := r.store.QueryRow(ctx, `SELECT COALESCE(SUM(total_count), 0) FROM analytics_events_agg WHERE `+where, args...)

	if err != nil {
		return nil, err
	}

	var total int64

	if err := totalRow.Scan(&total); err != nil {
		return nil, err
	}

	return &reportResult{Rows: out, Totals: ReportTotals{Count: &total}}, nil
}
