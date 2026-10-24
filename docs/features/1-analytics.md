---
title: Web Analytics
description: Privacy-friendly, cookieless analytics with an optional client script, custom events and an API for agents.
---

# Analytics

Stormkit's Analytics shows who visits your site, which pages they read, where
they came from and how long they stay, without cookies or a consent banner.
Analytics is an Enterprise feature.

<div class="img-wrapper"> 
  <picture>
    <source media="(max-width: 1200px)" srcset="/assets/docs/features/analytics.png?size=600" />
    <img src="/assets/docs/features/analytics.png" alt="Web Analytics" />
  </picture>
</div>

## Enabling analytics

Analytics is recorded for custom domains only. Set one up for your environment
in **Environment** > **Config** > **Other** > **Custom Domains**, and page
views are counted from then on.

## Excluding a domain

An application often answers on more than one hostname for the same site — an
apex domain plus its `www` alias, or a staging hostname. Analytics are reported
per domain, so each of these shows up as its own entry in the domain picker and
in Team Insights, and keeps accumulating stored rows, even when you only ever
look at the production hostname.

To keep a domain out of analytics, visit your **Environment** > **Config** >
**Other** > **Custom Domains**, open the domain's menu and select
**Exclude from analytics**. The same menu offers **Include in analytics** to
undo it.

An excluded domain is still served normally and its requests are still written
to [access logs](/docs/api/access-logs) — only page views and custom events are
dropped. The change applies to requests served after the setting is saved;
data collected before it is not removed.

## What is collected

Every HTML page Stormkit serves is counted server-side, so analytics work
without any script and are not blocked by ad blockers. For each page view
Stormkit records:

- The path, without its query string
- The referrer's host name
- The response status
- The visitor's country, looked up from the IP address when the page view is
  stored
- Campaign parameters from the landing URL: `utm_source`, `utm_medium`,
  `utm_campaign` and `ref`

Bots are detected from their user agent and left out.

### Visitors without cookies

No cookies or browser storage are used. A visitor is identified by a salted
hash of their IP address and user agent that rotates every day; analytics never
store the IP address itself. That keeps visitors anonymous, but the same person
can't be recognized on two different days, so a multi-day visitor count is the
sum of each day's unique visitors.

Raw page views are kept for 180 days (configurable with
`STORMKIT_ANALYTICS_RETENTION_DAYS` on self-hosted installs); the daily totals
the dashboard and API read from are kept indefinitely.

## Client-side tracking

Server-side counting sees every full page load. Single-page apps change pages
without one, so for those turn on **Client-side tracking** in the **Events**
card of the environment's **Analytics** page. Stormkit then injects a small,
same-origin script into your HTML that adds:

- **Route changes:** each client-side navigation counts as a page view, credited
  to the page the visitor came from. Changes to only the query string or hash
  (filters, pagination) are not counted.
- **Prefetched pages:** pages the browser prefetched or prerendered are counted
  once the visitor actually sees them, not when they were fetched.
- **Time on page:** the time each page is visible in the foreground, reported
  when the visitor leaves it or hides the tab.
- **Custom events:** `window.stormkit.track()`, see below.

The script sets no cookies either. If you add it yourself instead of using the
toggle, include it in your HTML as a regular tag, not through a tag manager:

```html
<script src="/_stormkit/analytics.js" async></script>
```

## Custom events

With client-side tracking on, record any action as an event:

```js
window.stormkit.track("signup", { plan: "pro", ref: "pricing" });
```

The first argument is the event name; the optional second is a flat object of
properties stored with the event. Every property can be used to break an event
down in the dashboard, for example `signup` by `plan`.

For TypeScript, declare the global once:

```ts
declare global {
  interface Window {
    stormkit?: {
      track(name: string, properties?: Record<string, string | number | boolean>): void;
    };
  }
}

export {};
```

Call it with optional chaining (`window.stormkit?.track(...)`) so code still
runs where the script isn't loaded, such as local development.

Events can also be sent from your server, for actions that never reach the
browser:

```bash
curl -X POST https://www.example.org/_stormkit/collect \
  -H 'Content-Type: application/json' \
  -d '{"events":[{"name":"signup","metadata":{"plan":"pro"}}]}'
```

Limits:

- Only the first 20 events of a request are kept, and requests over 64 KB are
  rejected.
- Each IP address can send about 120 requests a minute.
- Event names and property values are stored as sent, so keep names to a fixed
  set; don't put user ids or other unbounded values in the name.

Events are counted per day with the number of distinct visitors who sent them,
and like page views are kept raw for 180 days.

## Reading analytics from the API or an agent

The same numbers are available through the
[Analytics API](/docs/api/analytics) and the `get_analytics`
[MCP tool](/docs/api/mcp), so an agent can answer questions like "how many
people read /blog/my-post this week, and where did they come from?" in one
call. The API also groups referrers into channels (search, AI, social, email,
referral, direct), lists landing pages, and compares a period with the one
before it.
