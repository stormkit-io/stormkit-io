package analytics

import (
	"context"
	"encoding/json"
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
	MetricChannels  = "channels"
	MetricEntries   = "entries"

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

var reportMetrics = []string{MetricVisitors, MetricPaths, MetricReferrers, MetricChannels, MetricEntries, MetricCountries, MetricCampaigns, MetricEvents}

// Notes explaining how a report's numbers are counted, returned alongside them
// so an agent or a person reading the raw response does not misread them.
const (
	noteDailyUniques   = "visitors is the sum of daily unique visitors: visitor ids rotate daily, so the same person on two days counts twice."
	notePageVisitors   = "pageVisitors sums each matching page's daily unique visitors: a person reading three matching pages on one day counts three times. Use an exact path for unique visitors."
	noteSourceViews    = "source filters page views only; visitor and visit counts include both sources."
	noteReferrers      = "referrers exclude this site's own domains, whose visits are counted in totals.internal; totals.direct counts visits without a referrer, which includes browsers and links that hide it."
	noteChannels       = "channels group referrers: direct has no referrer (including browsers and links that hide it), internal is navigation within this site, and referral is any other site."
	noteEntries        = "entries are visits that arrived on a page from outside the site: from another site or direct. Navigation within the site is not counted."
	noteNoDomains      = "this environment has no custom domains; analytics are only recorded for custom domains."
	noteTimeOnPage     = "avgTimeOnPage is the average foreground time per page view, in seconds, measured by the client script; it is absent when no page view reported it."
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

	Compare bool
	Limit   int
}

// ReportRange is an inclusive range of UTC days.
type ReportRange struct {
	From string `json:"from"`
	To   string `json:"to"`
	Days int    `json:"days"`
}

// ReportDay is one day of a visitors report.
type ReportDay struct {
	Date          string `json:"date"`
	PageViews     int64  `json:"pageViews"`
	Visitors      *int64 `json:"visitors,omitempty"`
	PageVisitors  *int64 `json:"pageVisitors,omitempty"`
	AvgTimeOnPage *int64 `json:"avgTimeOnPage,omitempty"`

	engaged engagement
}

// engagement sums the foreground time the client script reported.
type engagement struct {
	seconds int64
	views   int64
}

func (e engagement) add(o engagement) engagement {
	return engagement{seconds: e.seconds + o.seconds, views: e.views + o.views}
}

// average returns the average seconds per engaged page view, or nil when no
// page view reported any.
func (e engagement) average() *int64 {
	if e.views == 0 {
		return nil
	}

	avg := (e.seconds + e.views/2) / e.views

	return &avg
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
	Channel     string `json:"channel,omitempty"`

	PageViews     *int64 `json:"pageViews,omitempty"`
	Visits        *int64 `json:"visits,omitempty"`
	Visitors      *int64 `json:"visitors,omitempty"`
	PageVisitors  *int64 `json:"pageVisitors,omitempty"`
	Count         *int64 `json:"count,omitempty"`
	AvgTimeOnPage *int64 `json:"avgTimeOnPage,omitempty"`

	// Channels breaks an entry page's visits down by channel.
	Channels map[string]int64 `json:"channels,omitempty"`

	// Previous is the row's main number in the previous period, set when the
	// report compares periods.
	Previous *int64 `json:"previous,omitempty"`
}

// ReportTotals sums a report across its rows or days.
type ReportTotals struct {
	PageViews     *int64 `json:"pageViews,omitempty"`
	Visitors      *int64 `json:"visitors,omitempty"`
	PageVisitors  *int64 `json:"pageVisitors,omitempty"`
	Visits        *int64 `json:"visits,omitempty"`
	Direct        *int64 `json:"direct,omitempty"`
	Internal      *int64 `json:"internal,omitempty"`
	Count         *int64 `json:"count,omitempty"`
	AvgTimeOnPage *int64 `json:"avgTimeOnPage,omitempty"`
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

	r := &reporter{store: s, params: p, from: p.From}

	if p.Compare {
		days := rangeDays(p.From, p.To)
		prevTo := p.From.AddDate(0, 0, -1)
		r.from = prevTo.AddDate(0, 0, -(days - 1))
		report.Previous = &ReportPeriod{Range: newReportRange(r.from, prevTo)}
	}

	result, err := r.run(ctx)

	if err != nil {
		return nil, err
	}

	report.Totals = result.Totals
	report.Series = result.Series
	report.Rows = result.Rows
	report.Notes = r.notes()

	if report.Previous != nil {
		report.Previous.Totals = result.Previous
	}

	return report, nil
}

// reporter runs the queries of one report.
type reporter struct {
	store  *Store
	params ReportParams

	// from is the first day the queries read: the previous period's first
	// day when the report compares periods, params.From otherwise.
	from time.Time
}

type reportResult struct {
	Totals   ReportTotals
	Previous ReportTotals
	Series   []ReportDay
	Rows     []ReportRow
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

		notes = append(notes, noteTimeOnPage)

		if r.params.Source != "" {
			notes = append(notes, noteSourceViews)
		}
	case MetricReferrers:
		notes = append(notes, noteReferrers)
	case MetricChannels:
		notes = append(notes, noteChannels)

		if r.params.PathPrefix != "" {
			notes = append(notes, notePageVisitors)
		} else {
			notes = append(notes, noteDailyUniques)
		}

		if r.params.Source != "" {
			notes = append(notes, noteSourceViews)
		}
	case MetricEntries:
		notes = append(notes, noteEntries)
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

func (r *reporter) run(ctx context.Context) (*reportResult, error) {
	switch r.params.Metric {
	case MetricVisitors:
		return r.visitors(ctx)
	case MetricPaths:
		return r.paths(ctx)
	case MetricReferrers:
		return r.referrers(ctx)
	case MetricChannels:
		return r.channels(ctx)
	case MetricEntries:
		return r.entries(ctx)
	case MetricCountries:
		return r.countries(ctx)
	case MetricCampaigns:
		return r.campaigns(ctx)
	case MetricEvents:
		return r.events(ctx)
	}

	return nil, errUnknownMetric
}

func (r *reporter) query() reportQuery {
	ids := make([]int64, len(r.params.DomainIDs))

	for i, id := range r.params.DomainIDs {
		ids[i] = int64(id)
	}

	return reportQuery{
		Domains:     pq.Array(ids),
		From:        r.from.Format(time.DateOnly),
		CurrentFrom: r.params.From.Format(time.DateOnly),
		To:          r.params.To.Format(time.DateOnly),
		Path:        r.params.Path,
		PathPrefix:  r.params.PathPrefix,
		Source:      r.params.Source,
		Limit:       r.params.Limit,
	}
}

func (r *reporter) pathValue() string {
	if r.params.Path != "" {
		return r.params.Path
	}

	return r.params.PathPrefix
}

// previous returns v when the report compares periods, so rows only carry a
// previous number when one was asked for.
func (r *reporter) previous(v int64) *int64 {
	if !r.params.Compare {
		return nil
	}

	return &v
}

// rows renders the named statement and runs it, calling scan for every row.
func (r *reporter) rows(ctx context.Context, name string, scan func(scan func(dest ...any) error) error) error {
	sql, args, err := renderReportQuery(name, r.query())

	if err != nil {
		return err
	}

	rows, err := r.store.Query(ctx, sql, args...)

	if err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		if err := scan(rows.Scan); err != nil {
			return err
		}
	}

	return rows.Err()
}

// totals runs a totals statement, which returns the current and the previous
// period's total.
func (r *reporter) totals(ctx context.Context, name string) (int64, int64, error) {
	var current, previous int64

	err := r.rows(ctx, name, func(scan func(dest ...any) error) error {
		return scan(&current, &previous)
	})

	return current, previous, err
}

func (r *reporter) visitors(ctx context.Context) (*reportResult, error) {
	current, err := r.visitorsBetween(ctx, r.params.From, r.params.To)

	if err != nil {
		return nil, err
	}

	if r.params.Compare {
		previous, err := r.visitorsBetween(ctx, r.from, r.params.From.AddDate(0, 0, -1))

		if err != nil {
			return nil, err
		}

		current.Previous = previous.Totals
	}

	return current, nil
}

// visitorsBetween returns the per-day series of a range of days.
func (r *reporter) visitorsBetween(ctx context.Context, from, to time.Time) (*reportResult, error) {
	period := &reporter{store: r.store, params: r.params, from: from}
	period.params.From, period.params.To = from, to
	byDate := map[string]*ReportDay{}

	err := period.rows(ctx, "visitorsSeries", func(scan func(dest ...any) error) error {
		var day ReportDay
		var uniques int64

		if err := scan(&day.Date, &day.PageViews, &uniques, &day.engaged.seconds, &day.engaged.views); err != nil {
			return err
		}

		day.AvgTimeOnPage = day.engaged.average()

		switch {
		case r.params.Path != "":
			day.Visitors = &uniques
		case r.params.PathPrefix != "":
			day.PageVisitors = &uniques
		}

		byDate[day.Date] = &day

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Site-wide unique visitors cannot be summed from per-path rows, which
	// count a visitor once per page; the domain-level aggregate holds them.
	if r.pathValue() == "" {
		err := period.rows(ctx, "siteVisitors", func(scan func(dest ...any) error) error {
			var date string
			var uniques int64

			if err := scan(&date, &uniques); err != nil {
				return err
			}

			if byDate[date] == nil {
				byDate[date] = &ReportDay{Date: date}
			}

			byDate[date].Visitors = &uniques

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	result := &reportResult{Series: []ReportDay{}}
	totals := ReportDay{}
	engaged := engagement{}

	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := byDate[d.Format(time.DateOnly)]

		if day == nil {
			day = &ReportDay{Date: d.Format(time.DateOnly)}
		}

		zero := int64(0)

		if r.params.PathPrefix != "" && day.PageVisitors == nil {
			day.PageVisitors = &zero
		} else if r.params.PathPrefix == "" && day.Visitors == nil {
			day.Visitors = &zero
		}

		totals.PageViews += day.PageViews
		engaged = engaged.add(day.engaged)
		totals.Visitors = addPtr(totals.Visitors, day.Visitors)
		totals.PageVisitors = addPtr(totals.PageVisitors, day.PageVisitors)
		result.Series = append(result.Series, *day)
	}

	result.Totals = ReportTotals{
		PageViews:     &totals.PageViews,
		Visitors:      totals.Visitors,
		PageVisitors:  totals.PageVisitors,
		AvgTimeOnPage: engaged.average(),
	}

	return result, nil
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

func (r *reporter) paths(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "paths", func(scan func(dest ...any) error) error {
		var row ReportRow
		var views, previous, visitors int64
		var engaged engagement

		if err := scan(&row.Path, &views, &previous, &visitors, &engaged.seconds, &engaged.views); err != nil {
			return err
		}

		row.PageViews, row.Visitors, row.Previous = &views, &visitors, r.previous(previous)
		row.AvgTimeOnPage = engaged.average()
		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	var current, previous int64
	var engaged, prevEngaged engagement

	err = r.rows(ctx, "pathsTotals", func(scan func(dest ...any) error) error {
		return scan(&current, &previous, &engaged.seconds, &engaged.views, &prevEngaged.seconds, &prevEngaged.views)
	})

	if err != nil {
		return nil, err
	}

	return &reportResult{
		Rows:     out,
		Totals:   ReportTotals{PageViews: &current, AvgTimeOnPage: engaged.average()},
		Previous: ReportTotals{PageViews: &previous, AvgTimeOnPage: prevEngaged.average()},
	}, nil
}

// channelTotals returns every channel's visits in the current and the
// previous period.
func (r *reporter) channelTotals(ctx context.Context) ([]ReportRow, map[string]int64, map[string]int64, error) {
	rows := []ReportRow{}
	current, previous := map[string]int64{}, map[string]int64{}

	err := r.rows(ctx, "channels", func(scan func(dest ...any) error) error {
		var channel string
		var visits, prev int64

		if err := scan(&channel, &visits, &prev); err != nil {
			return err
		}

		current[channel], previous[channel] = visits, prev

		if visits > 0 {
			rows = append(rows, ReportRow{Channel: channel, Visits: &visits, Previous: r.previous(prev)})
		}

		return nil
	})

	return rows, current, previous, err
}

func sumValues(m map[string]int64) int64 {
	total := int64(0)

	for _, v := range m {
		total += v
	}

	return total
}

// referrerTotals sums visits, direct visits and internal visits per period.
func referrerTotals(byChannel map[string]int64) ReportTotals {
	total := sumValues(byChannel)
	direct := byChannel[ChannelDirect]
	internal := byChannel[ChannelInternal]

	return ReportTotals{Visits: &total, Direct: &direct, Internal: &internal}
}

func (r *reporter) referrers(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "referrers", func(scan func(dest ...any) error) error {
		var row ReportRow
		var visits, previous int64

		if err := scan(&row.Referrer, &row.Channel, &visits, &previous); err != nil {
			return err
		}

		row.Visits, row.Previous = &visits, r.previous(previous)
		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	_, current, previous, err := r.channelTotals(ctx)

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: out, Totals: referrerTotals(current), Previous: referrerTotals(previous)}, nil
}

func (r *reporter) channels(ctx context.Context) (*reportResult, error) {
	rows, current, previous, err := r.channelTotals(ctx)

	if err != nil {
		return nil, err
	}

	// The page views and visitors behind the channels, so one call answers
	// "how many people read this and where did they come from".
	views, err := r.visitors(ctx)

	if err != nil {
		return nil, err
	}

	totals, prevTotals := views.Totals, views.Previous
	visits, prevVisits := sumValues(current), sumValues(previous)
	totals.Visits, prevTotals.Visits = &visits, &prevVisits

	return &reportResult{Rows: rows, Totals: totals, Previous: prevTotals}, nil
}

func (r *reporter) entries(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "entries", func(scan func(dest ...any) error) error {
		var row ReportRow
		var visits, previous int64
		var channels []byte

		if err := scan(&row.Path, &visits, &previous, &channels); err != nil {
			return err
		}

		if err := json.Unmarshal(channels, &row.Channels); err != nil {
			return err
		}

		row.Visits, row.Previous = &visits, r.previous(previous)
		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	_, current, previous, err := r.channelTotals(ctx)

	if err != nil {
		return nil, err
	}

	total := sumValues(current) - current[ChannelInternal]
	prevTotal := sumValues(previous) - previous[ChannelInternal]

	return &reportResult{Rows: out, Totals: ReportTotals{Visits: &total}, Previous: ReportTotals{Visits: &prevTotal}}, nil
}

func (r *reporter) countries(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "countries", func(scan func(dest ...any) error) error {
		var row ReportRow
		var visitors, previous int64

		if err := scan(&row.Country, &visitors, &previous); err != nil {
			return err
		}

		// Per-page rows count a reader of several matching pages once per
		// page, so a prefix reports them under their own name.
		if r.params.PathPrefix != "" {
			row.PageVisitors = &visitors
		} else {
			row.Visitors = &visitors
		}

		row.Previous = r.previous(previous)
		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: out}, nil
}

func (r *reporter) campaigns(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "campaigns", func(scan func(dest ...any) error) error {
		var row ReportRow
		var visits, previous, visitors int64

		if err := scan(&row.UTMSource, &row.UTMMedium, &row.UTMCampaign, &row.Ref, &visits, &previous, &visitors); err != nil {
			return err
		}

		row.Visits, row.Previous = &visits, r.previous(previous)

		// Visitors are summed across landing pages, so they are only unique
		// when a single landing path is selected.
		if r.params.Path != "" {
			row.Visitors = &visitors
		}

		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	current, previous, err := r.totals(ctx, "campaignsTotals")

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: out, Totals: ReportTotals{Visits: &current}, Previous: ReportTotals{Visits: &previous}}, nil
}

func (r *reporter) events(ctx context.Context) (*reportResult, error) {
	out := []ReportRow{}

	err := r.rows(ctx, "events", func(scan func(dest ...any) error) error {
		var row ReportRow
		var count, previous, actors int64

		if err := scan(&row.Event, &count, &previous, &actors); err != nil {
			return err
		}

		row.Count, row.Visitors, row.Previous = &count, &actors, r.previous(previous)
		out = append(out, row)

		return nil
	})

	if err != nil {
		return nil, err
	}

	current, previous, err := r.totals(ctx, "eventsTotals")

	if err != nil {
		return nil, err
	}

	return &reportResult{Rows: out, Totals: ReportTotals{Count: &current}, Previous: ReportTotals{Count: &previous}}, nil
}
