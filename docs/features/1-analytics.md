---
title: Web Analytics
description: Simple, privacy-friendly, automatic setup and powerful analytics for web teams.
---

# Analytics

Stormkit's Analytics feature empowers users to gain valuable insights into their web applications.

<div class="img-wrapper"> 
  <picture>
    <source media="(max-width: 1200px)" srcset="/assets/docs/features/analytics.png?size=600" />
    <img src="/assets/docs/features/analytics.png" alt="Web Analytics" />
  </picture>
</div>

## Enabling analytics

To enable Analytics, users must set up a custom domain for their web application. This ensures accurate tracking of analytics data.

To do so, visit your **Environment** > **Config** > **Other** > **Custom Domains**

## Collecting stats

All stats are collected on the server-side, which makes this privacy friendly. We collect the following data:

- Visitor IP
- Timestamp
- Request path
- Referrer
- Response code
- User Agent

There are absolute no cookies stored on the client machine.

## Unique vs Total visitors

By default Stormkit loads both unique and total visitors on the Visitors panel. To view only unique, or total visitors, click on the related legend.

You can change the time span to 24 hours, 7 days or 30 days.

We automatically detect bots from user agents and exclude them from the statistics.

## Top referrers

You can view the list of top referrers in the last 30 days. Depending on the `Referrer-Policy` header set,
we may display the full URL or only the domain name.

## Top paths

You can view the list of top visited paths in the last 30 days. Client-side routing is excluded from these
statistics.
