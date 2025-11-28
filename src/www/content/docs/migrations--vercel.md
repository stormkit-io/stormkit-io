---
title: Migrating from Vercel to Stormkit
description: Learn how to migrate your projects from Vercel to Stormkit. Discover the differences, and the unique capabilities Stormkit offers to enhance your deployment experience.
---

# Migrating from Vercel to Stormkit

<section id="migration-guide">

Migrating from Vercel to Stormkit is a straightforward process that allows you to take advantage of Stormkit's unique features, such as self-hosting, advanced deployment previews, and environment-level configurations. This guide will walk you through the migration process, highlight the differences between the two platforms, and showcase the features Stormkit excels.

## Why Migrate to Stormkit?

While Vercel is a powerful platform, Stormkit provides additional flexibility and control for developers who need more than just a managed hosting solution. Here’s why you might consider migrating:

- **Self-Hosting Options:** Deploy Stormkit on your own infrastructure for complete control.
- **Price Control:** Since Stormkit is self-hosted, you get to choose your infrastructure and can keep your bills under control.
- **Advanced Deployment Previews:** Post-deployment checks to run end to end tests before going live.
- **Environment-Level Redirects:** Apply global rules across all deployments instantly.
- **Snippet Injection:** Dynamically inject code snippets based on request paths.

## Migration steps

Next.js is fully supported on Stormkit's Self-Hosted Edition. To migrate:

- [Install Stormkit on your VPS](/docs/self-hosting/getting-started)
- Create an application either by:
  - Importing from a public URL or
  - Importing from a private repository
- Build settings:
  - Output folder: `./.next`
- Server settings
  - Start command: `npm run start` (or your custom script which calls `next start`)
- Click Deploy

## Features supported

All features that are working locally should be working on Stormkit as well:

- [App Router](https://nextjs.org/docs/app) <br />
- [Route Handlers](https://nextjs.org/docs/app/building-your-application/routing/route-handlers) <br />
- [Dynamic routes](https://nextjs.org/docs/app/building-your-application/routing/dynamic-routes) <br />
- [Static Site Generation (SSG)](https://nextjs.org/docs/app/building-your-application/rendering/server-components#static-rendering-default) <br />
- [Server-Side Rendering (SSR)](https://nextjs.org/docs/app/building-your-application/rendering/server-components) <br />
- [Middleware](https://nextjs.org/docs/app/building-your-application/routing/middleware) <br />
- [Node Middleware](https://nextjs.org/docs/app/building-your-application/routing/middleware#runtime) <br />
- [Image optimization](https://nextjs.org/docs/app/building-your-application/optimizing/images) <br />
- [Partial Prerendering (PPR)](https://nextjs.org/docs/app/building-your-application/rendering/partial-prerendering) <br />
- [Pages Router](https://nextjs.org/docs/pages) <br />
- [Incremental Static Regeneration (ISR)](https://nextjs.org/docs/app/building-your-application/data-fetching/incremental-static-regeneration) <br />
- [Support for after](https://nextjs.org/blog/next-15-rc#executing-code-after-a-response-with-nextafter-experimental) <br />
- [Composable Caching](https://nextjs.org/blog/composable-caching) (`'use cache'`) <br/>

## Runtimes supported

Next.js has two runtimes: Edge and Node.js. When using Stormkit, you should use the Node.js runtime.
