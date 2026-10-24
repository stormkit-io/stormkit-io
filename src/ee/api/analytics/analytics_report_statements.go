package analytics

import (
	"bytes"
	"fmt"
	"text/template"
)

// reportStatements holds every report query as a named template.
//
// Each query reads the whole range, the previous period included when the
// report compares periods, and splits it with FILTER: "current" is true for
// days on or after .CurrentFrom. Without a comparison .From equals
// .CurrentFrom, so the previous columns are zero.
//
// No value is written into the SQL: templates pass values to bind, which
// records them as query arguments and returns their placeholder ($1, $2, ...).
const reportStatements = `
{{- define "range" -}}
domain_id = ANY({{ bind .Domains }}::bigint[])
AND aggregate_date BETWEEN {{ bind .From }}::date AND {{ bind .To }}::date
{{- if .Path }}
AND request_path_hash = decode(md5({{ bind .Path }}), 'hex')
{{- else if .PathPrefix }}
AND starts_with(request_path, {{ bind .PathPrefix }})
{{- end }}
{{- end }}

{{- define "current" }}aggregate_date >= {{ bind .CurrentFrom }}::date{{ end }}

{{- define "views" -}}
{{ if eq .Source "server" }}server_views{{ else if eq .Source "client" }}client_views{{ else }}page_views{{ end }}
{{- end }}

{{- define "visitorsSeries" -}}
SELECT aggregate_date::text, SUM({{ template "views" . }}), SUM(unique_visitors), SUM(engaged_seconds), SUM(engaged_views)
FROM analytics_paths_agg
WHERE {{ template "range" . }}
GROUP BY aggregate_date
{{- end }}

{{- define "siteVisitors" -}}
SELECT aggregate_date::text, SUM(unique_visitors)
FROM analytics_visitors_agg_200
WHERE domain_id = ANY({{ bind .Domains }}::bigint[])
AND aggregate_date BETWEEN {{ bind .From }}::date AND {{ bind .To }}::date
GROUP BY aggregate_date
{{- end }}

{{- define "paths" -}}
SELECT
	request_path,
	COALESCE(SUM({{ template "views" . }}) FILTER (WHERE {{ template "current" . }}), 0) AS views,
	COALESCE(SUM({{ template "views" . }}) FILTER (WHERE NOT {{ template "current" . }}), 0),
	COALESCE(SUM(unique_visitors) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_seconds) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_views) FILTER (WHERE {{ template "current" . }}), 0)
FROM analytics_paths_agg
WHERE {{ template "range" . }}
GROUP BY request_path
HAVING SUM({{ template "views" . }}) FILTER (WHERE {{ template "current" . }}) > 0
ORDER BY views DESC, request_path
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "pathsTotals" -}}
SELECT
	COALESCE(SUM({{ template "views" . }}) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM({{ template "views" . }}) FILTER (WHERE NOT {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_seconds) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_views) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_seconds) FILTER (WHERE NOT {{ template "current" . }}), 0),
	COALESCE(SUM(engaged_views) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_paths_agg
WHERE {{ template "range" . }}
{{- end }}

{{- define "channels" -}}
SELECT
	channel,
	COALESCE(SUM(visits) FILTER (WHERE {{ template "current" . }}), 0) AS visits,
	COALESCE(SUM(visits) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_channels_agg
WHERE {{ template "range" . }}
GROUP BY channel
ORDER BY visits DESC, channel
{{- end }}

{{- define "referrers" -}}
{{- /* A referrer's channel can change within a range, e.g. when the site adds
the host as its own domain, so each referrer is labelled with the channel most
of its visits had. */ -}}
SELECT referrer, (array_agg(channel ORDER BY visits DESC, channel))[1], SUM(visits) AS visits, SUM(previous)
FROM (
	SELECT
		referrer,
		channel,
		COALESCE(SUM(visits) FILTER (WHERE {{ template "current" . }}), 0) AS visits,
		COALESCE(SUM(visits) FILTER (WHERE NOT {{ template "current" . }}), 0) AS previous
	FROM analytics_channels_agg
	WHERE {{ template "range" . }} AND referrer <> '' AND channel <> 'internal'
	GROUP BY referrer, channel
) by_channel
GROUP BY referrer
HAVING SUM(visits) > 0
ORDER BY visits DESC, referrer
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "entries" -}}
{{- /* Entries are visits that arrived on a page from outside the site. */ -}}
SELECT request_path, SUM(visits) AS visits, SUM(previous), jsonb_object_agg(channel, visits) FILTER (WHERE visits > 0)
FROM (
	SELECT
		request_path,
		channel,
		COALESCE(SUM(visits) FILTER (WHERE {{ template "current" . }}), 0) AS visits,
		COALESCE(SUM(visits) FILTER (WHERE NOT {{ template "current" . }}), 0) AS previous
	FROM analytics_channels_agg
	WHERE {{ template "range" . }} AND channel <> 'internal'
	GROUP BY request_path, channel
) by_channel
GROUP BY request_path
HAVING SUM(visits) > 0
ORDER BY visits DESC, request_path
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "countries" -}}
{{- if or .Path .PathPrefix }}
SELECT
	country_iso_code,
	COALESCE(SUM(unique_visitors) FILTER (WHERE {{ template "current" . }}), 0) AS visitors,
	COALESCE(SUM(unique_visitors) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_path_countries_agg
WHERE {{ template "range" . }}
GROUP BY country_iso_code
HAVING SUM(unique_visitors) FILTER (WHERE {{ template "current" . }}) > 0
{{- else }}
SELECT
	country_iso_code,
	COALESCE(SUM(visit_count) FILTER (WHERE {{ template "current" . }}), 0) AS visitors,
	COALESCE(SUM(visit_count) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_visitors_by_countries
WHERE {{ template "range" . }}
GROUP BY country_iso_code
HAVING SUM(visit_count) FILTER (WHERE {{ template "current" . }}) > 0
{{- end }}
ORDER BY visitors DESC, country_iso_code
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "campaigns" -}}
SELECT
	utm_source, utm_medium, utm_campaign, ref,
	COALESCE(SUM(visits) FILTER (WHERE {{ template "current" . }}), 0) AS visits,
	COALESCE(SUM(visits) FILTER (WHERE NOT {{ template "current" . }}), 0),
	COALESCE(SUM(unique_visitors) FILTER (WHERE {{ template "current" . }}), 0)
FROM analytics_campaigns_agg
WHERE {{ template "range" . }}
GROUP BY utm_source, utm_medium, utm_campaign, ref
HAVING SUM(visits) FILTER (WHERE {{ template "current" . }}) > 0
ORDER BY visits DESC, utm_source, utm_medium, utm_campaign, ref
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "campaignsTotals" -}}
SELECT
	COALESCE(SUM(visits) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(visits) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_campaigns_agg
WHERE {{ template "range" . }}
{{- end }}

{{- define "events" -}}
SELECT
	event_name,
	COALESCE(SUM(total_count) FILTER (WHERE {{ template "current" . }}), 0) AS total,
	COALESCE(SUM(total_count) FILTER (WHERE NOT {{ template "current" . }}), 0),
	COALESCE(SUM(unique_actors) FILTER (WHERE {{ template "current" . }}), 0)
FROM analytics_events_agg
WHERE domain_id = ANY({{ bind .Domains }}::bigint[])
AND aggregate_date BETWEEN {{ bind .From }}::date AND {{ bind .To }}::date
GROUP BY event_name
HAVING SUM(total_count) FILTER (WHERE {{ template "current" . }}) > 0
ORDER BY total DESC, event_name
LIMIT {{ bind .Limit }}
{{- end }}

{{- define "eventsTotals" -}}
SELECT
	COALESCE(SUM(total_count) FILTER (WHERE {{ template "current" . }}), 0),
	COALESCE(SUM(total_count) FILTER (WHERE NOT {{ template "current" . }}), 0)
FROM analytics_events_agg
WHERE domain_id = ANY({{ bind .Domains }}::bigint[])
AND aggregate_date BETWEEN {{ bind .From }}::date AND {{ bind .To }}::date
{{- end }}
`

// reportTemplates is parsed once. Its bind function is a placeholder that each
// render replaces on a clone with one bound to that render's arguments.
var reportTemplates = template.Must(template.New("report").Funcs(template.FuncMap{
	"bind": func(any) string { return "" },
}).Parse(reportStatements))

// reportQuery is the data a report template renders with.
type reportQuery struct {
	Domains any

	// From is the first day read: the previous period's first day when the
	// report compares periods, CurrentFrom otherwise.
	From        string
	CurrentFrom string
	To          string

	Path       string
	PathPrefix string
	Source     string
	Limit      int
}

// sqlBinder collects the arguments of one rendered query.
type sqlBinder struct {
	args []any
}

// bind appends v and returns its placeholder.
func (b *sqlBinder) bind(v any) string {
	b.args = append(b.args, v)

	return fmt.Sprintf("$%d", len(b.args))
}

// renderReportQuery renders the named statement and returns the SQL with its
// arguments.
func renderReportQuery(name string, q reportQuery) (string, []any, error) {
	b := &sqlBinder{}
	tmpl, err := reportTemplates.Clone()

	if err != nil {
		return "", nil, err
	}

	var sql bytes.Buffer

	if err := tmpl.Funcs(template.FuncMap{"bind": b.bind}).ExecuteTemplate(&sql, name, q); err != nil {
		return "", nil, fmt.Errorf("render %s: %w", name, err)
	}

	return sql.String(), b.args, nil
}
