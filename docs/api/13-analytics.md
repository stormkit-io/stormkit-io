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
| `metric`     | string | No          | What to report: `visitors` (default), `paths`, `referrers`, `channels`, `entries`, `countries`, `campaigns` or `events`. |
| `from`       | string | No          | First day of the range, `YYYY-MM-DD` in UTC. Cannot be combined with `span`. |
| `to`         | string | No          | Last day of the range, `YYYY-MM-DD` in UTC, inclusive. Defaults to today. |
| `span`       | string | No          | Range length ending at `to`, e.g. `7d`, `30d`, `90d`. Defaults to `30d`, maximum `400d`. |
| `domainId`   | string | No          | Only report this domain. Defaults to all of the environment's domains. |
| `path`       | string | No          | Only count this exact page path, e.g. `/blog/my-post`. |
| `pathPrefix` | string | No          | Only count pages whose path starts with this value, e.g. `/blog/`. |
| `source`     | string | No          | `server` counts only full page loads, `client` only navigations reported by the [client script](/docs/features/analytics). Applies to page views only. |
| `compare`    | string | No          | `previous` also returns the previous period of the same length. |
| `limit`      | number | No          | How many rows to return. Defaults to `10`, maximum `50`. |

`path` and `pathPrefix` filter every metric except `events`. For `campaigns`
and `entries` they select the landing page.

### Metrics

| Metric      | Returns |
| ----------- | ------- |
| `visitors`  | `series`: page views and visitors per day, plus `totals`. |
| `paths`     | Top pages by page views, each with its visitors. |
| `referrers` | Top referring hosts by visits, each with its `channel`. The site's own domains are left out and counted in `totals.internal`; `totals.direct` counts visits without a referrer. |
| `channels`  | Visits by channel (see below), plus the `pageViews` and `visitors` behind them. With `path`, this answers "how many people read this page, and where did they come from?" in one call. |
| `entries`   | Landing pages: visits that arrived on a page from another site or directly, each with a `channels` breakdown. Navigation within the site is not counted. |
| `countries` | Visitors by country. With `pathPrefix`, `pageVisitors` instead (see below). |
| `campaigns` | Visits by `utm_source`, `utm_medium`, `utm_campaign` and `ref`, read from the landing URL, ranked by visits. Visitors are only included with an exact `path`, because a person landing on two pages from one campaign would otherwise count twice. Visits recorded before campaign tracking was added have no campaign. |
| `events`    | [Custom events](/docs/features/analytics) by count, each with the number of visitors who sent it. |

### Channels

Referrers are grouped by their host:

| Channel    | Referrer |
| ---------- | -------- |
| `search`   | Search engines: Google, Bing, DuckDuckGo, Yahoo, Yandex, Baidu, Ecosia, Brave, Qwant, Startpage, Kagi, Naver, Seznam. |
| `ai`       | AI assistants: ChatGPT, Perplexity, Claude, Gemini, Copilot, DeepSeek, Mistral, Grok, Meta AI, You.com, Phind, Poe. |
| `social`   | Social sites and communities: X, Facebook, Instagram, LinkedIn, Reddit, Hacker News, YouTube, TikTok, Pinterest, Threads, Bluesky, Mastodon, Discord, Telegram, Product Hunt, Indie Hackers, DEV. |
| `email`    | Webmail: Gmail, Outlook, Yahoo Mail, Proton Mail. |
| `internal` | The environment's own domains and their subdomains, i.e. navigation within the site. |
| `direct`   | No referrer: typed or bookmarked URLs, apps, and links or browsers that hide the referrer. |
| `referral` | Any other site. |

Channels are assigned when page views are rolled up, every hour. A domain
added to the environment later is only treated as internal from then on.
Page view, path, campaign, referrer, channel and entry data starts on the
release that introduced this API; earlier days are empty. Site-wide visitor
and country counts go back further, so for older days a `visitors` series can
show visitors with no page views.

### The `(other)` path

To keep junk URLs (such as scanners probing a single-page app that answers
every path) from growing the stored data without bound, each domain keeps, per
day, every path with two or more views and its first 1,000 single-view paths,
up to 10,000 paths in total. The remaining page views are counted under the
path `(other)`, so site-wide totals stay exact; the visitors of `(other)` are
the sum of the folded paths' visitors. Folded page views do not match a `path`
or `pathPrefix` filter, so on days with folding, filtered totals undercount by
the folded single-view pages.

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
| `totals`   | object | Totals for the range. Only the counts relevant to the metric are present: `pageViews`, `visitors`, `pageVisitors`, `visits`, `direct`, `internal`, `count`. |
| `series`   | array  | One entry per day with `date`, `pageViews` and `visitors` or `pageVisitors` (`visitors` metric only). |
| `rows`     | array  | Top rows for the other metrics. Each carries its key (`path`, `referrer`, `channel`, `country`, `utmSource`/`utmMedium`/`utmCampaign`/`ref`, or `event`) and its counts; `entries` rows also carry `channels`. With `compare=previous`, each row also has `previous`: its main count in the previous period. |
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
  'https://api.stormkit.io/v1/analytics?metric=channels&path=/blog/my-post&span=7d'
```

```json
// Example response
{
  "metric": "channels",
  "range": { "from": "2026-10-18", "to": "2026-10-24", "days": 7 },
  "totals": { "pageViews": 412, "visitors": 355, "visits": 412 },
  "rows": [
    { "channel": "search", "visits": 201 },
    { "channel": "direct", "visits": 96 },
    { "channel": "social", "visits": 87 },
    { "channel": "internal", "visits": 28 }
  ],
  "notes": [
    "channels group referrers: direct has no referrer (including browsers and links that hide it), internal is navigation within this site, and referral is any other site.",
    "visitors is the sum of daily unique visitors: visitor ids rotate daily, so the same person on two days counts twice."
  ]
}
```
