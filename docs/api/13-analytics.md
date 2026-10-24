---
title: Analytics API
description: Documentation on reading visitor analytics through the Stormkit API.
---

# Analytics API

The analytics API returns the numbers behind the dashboard's analytics:
page views, visitors, top pages, referrers, countries, campaigns and custom
events for an environment's custom domains. Bots are excluded, and every number
comes from daily rollups, so a report is cheap to run and kept beyond the raw
data's retention window.

Analytics are only recorded for custom domains and are an Enterprise feature.

This endpoint is also available as the `get_analytics` MCP tool — see the
[MCP Server](/docs/api/mcp) docs.

---

## GET /v1/analytics

Returns one report for the environment.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Query parameters

| Parameter    | Type   | Required    | Description |
| ------------ | ------ | ----------- | ----------- |
| `envId`      | string | Conditional | Required when using an app, team, or user-level API key. Environment-level keys do not require this parameter. |
| `metric`     | string | No          | What to report: `visitors` (default), `paths`, `referrers`, `countries`, `campaigns` or `events`. |
| `from`       | string | No          | First day of the range, `YYYY-MM-DD` in UTC. Cannot be combined with `span`. |
| `to`         | string | No          | Last day of the range, `YYYY-MM-DD` in UTC, inclusive. Defaults to today. |
| `span`       | string | No          | Range length ending at `to`, e.g. `7d`, `30d`, `90d`. Defaults to `30d`, maximum `400d`. |
| `domainId`   | string | No          | Only report this domain. Defaults to all of the environment's domains. |
| `path`       | string | No          | Only count this exact page path, e.g. `/blog/my-post`. |
| `pathPrefix` | string | No          | Only count pages whose path starts with this value, e.g. `/blog/`. |
| `source`     | string | No          | `server` counts only full page loads, `client` only navigations reported by the [client script](/docs/features/analytics). Applies to page views only. |
| `compare`    | string | No          | `previous` also returns the previous period of the same length. |
| `limit`      | number | No          | How many rows to return. Defaults to `10`, maximum `50`. |

`path` and `pathPrefix` filter `visitors`, `paths`, `referrers`, `countries`
and `campaigns` (by landing page). They do not apply to `events`.

### Metrics

| Metric      | Returns |
| ----------- | ------- |
| `visitors`  | `series`: page views and visitors per day, plus `totals`. |
| `paths`     | Top pages by page views, each with its visitors. |
| `referrers` | Top referring hosts by visits. `totals.direct` counts visits without a referrer. |
| `countries` | Visitors by country. With `pathPrefix`, `pageVisitors` instead (see below). |
| `campaigns` | Visits by `utm_source`, `utm_medium`, `utm_campaign` and `ref`, read from the landing URL, ranked by visits. Visitors are only included with an exact `path`, because a person landing on two pages from one campaign would otherwise count twice. Visits recorded before campaign tracking was added have no campaign. |
| `events`    | [Custom events](/docs/features/analytics) by count, each with the number of visitors who sent it. |

### How visitors are counted

Visitors are identified without cookies, by a hash of the IP address and user
agent that rotates every day. That keeps visitors anonymous, but it means the
same person can't be recognized on two different days:

- A day's `visitors` is that day's unique visitors.
- A range's `visitors` is the **sum** of each day's unique visitors, so a
  person who visits on three days counts three times.
- With `pathPrefix`, `visitors` (and a country's visitors) is replaced by
  `pageVisitors`, which sums each matching page's daily unique visitors. A person who reads three matching
  pages on one day counts three times. Use an exact `path` for unique visitors
  of a page.

Every response carries a `notes` array that says which of these applies.

### Response — 200 OK

| Field      | Type   | Description |
| ---------- | ------ | ----------- |
| `metric`   | string | The metric reported. |
| `range`    | object | `from`, `to` and `days` of the reported range. |
| `totals`   | object | Totals for the range. Only the counts relevant to the metric are present: `pageViews`, `visitors`, `pageVisitors`, `visits`, `direct`, `count`. |
| `series`   | array  | One entry per day with `date`, `pageViews` and `visitors` or `pageVisitors` (`visitors` metric only). |
| `rows`     | array  | Top rows for the other metrics. Each carries its key (`path`, `referrer`, `country`, `utmSource`/`utmMedium`/`utmCampaign`/`ref`, or `event`) and its counts. With `compare=previous`, each row also has `previous`: its main count in the previous period. |
| `previous` | object | `range` and `totals` of the previous period, when `compare=previous`. |
| `notes`    | array  | How the numbers in this response are counted. |

### Error responses

| Status | Condition |
| ------ | --------- |
| `400`  | Invalid parameters. The response lists every problem under `errors`. |
| `402`  | The account is not on an Enterprise license. |
| `403`  | Missing/invalid API key, or the token has no access to the environment. |
| `404`  | `domainId` is not a domain of this environment. |
| `500`  | Internal server error. |

### Example

How many people read a blog post this week, and where did they come from?

```bash
curl -H 'Authorization: <api_key>' \
  'https://api.stormkit.io/v1/analytics?metric=referrers&path=/blog/my-post&span=7d'
```

```json
// Example response
{
  "metric": "referrers",
  "range": { "from": "2026-10-18", "to": "2026-10-24", "days": 7 },
  "totals": { "visits": 412, "direct": 96 },
  "rows": [
    { "referrer": "google.com", "visits": 201 },
    { "referrer": "news.ycombinator.com", "visits": 87 },
    { "referrer": "www.example.org", "visits": 28 }
  ],
  "notes": [
    "referrers are host names as sent by browsers, and include this site's own domain for in-site navigation."
  ]
}
```
