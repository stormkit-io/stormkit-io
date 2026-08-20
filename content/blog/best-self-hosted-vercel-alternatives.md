---
title: Best self-hosted Vercel alternatives in 2026
description: An honest comparison of the open source platforms you can run on your own server instead of Vercel or Netlify - Coolify, Dokploy, Dokku, CapRover, Kamal and Stormkit - with what each is actually good at and who should pick which.
date: 2026-08-13
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1993991074138779648/Up6HP-Jw_reasonably_small.jpg
---

**Disclosure up front: we build Stormkit, one of the options below.** We have
tried to write the comparison we wanted to read before we started, which means
being specific about what each tool is for rather than declaring a winner.

People leave [Vercel](/vs-vercel) and [Netlify](/vs-netlify) for three reasons, roughly in this order: the
bill stops being predictable, the platform cannot run something they need, or
they have a compliance requirement about where data lives. Self-hosting solves
all three, and costs you the time you used to spend not thinking about
infrastructure. That trade is worth making deliberately, not on a bad billing
day.

The important thing to sort out first is which kind of tool you are looking for,
because these six are not really competing with each other:

- **Container platforms** — Coolify, Dokploy, CapRover. Somewhere to run any
  Docker image, including off-the-shelf software you did not write.
- **An application platform** — Stormkit. The database, authentication, email
  and scheduled jobs your product needs, provided by the platform.
- **Deployment tooling** — Dokku, Kamal. Get code onto a server you already
  manage; everything else is yours.

## The options at a glance

| Project | License | Written in | GitHub stars | Shape of the thing |
| --- | --- | --- | --- | --- |
| [Coolify](https://coolify.io) | Apache-2.0 | PHP | ~60,500 | General self-hosted PaaS |
| [Dokploy](https://dokploy.com) | Apache-2.0 + proprietary parts | TypeScript | ~36,500 | General self-hosted PaaS |
| [Dokku](https://dokku.com) | MIT | Shell | ~32,000 | Minimal Heroku-style PaaS |
| [CapRover](https://caprover.com) | Apache-2.0 with added restrictions | TypeScript | ~15,100 | Docker + nginx PaaS with a UI |
| [Kamal](https://kamal-deploy.org) | MIT | Ruby | ~14,500 | Deployment tool, not a platform |
| [Stormkit](https://www.stormkit.io) | AGPL-3.0 core + proprietary parts | Go | ~250 | Application platform with database, auth and email built in |

Star counts are included because they are a rough proxy for how much community
and documentation sits behind a project — useful context, not a ranking. The
container platforms have been around longer and address a broader audience than
an application platform does.

## Coolify

The most popular project in this list. Coolify runs anything you can put in a
Docker container: applications, databases, and a long list of one-click
services. If you are replacing Vercel *and* your Postgres *and* a handful of
self-hosted tools, Coolify covers all of it from one UI.

```bash
curl -fsSL https://cdn.coollabs.io/coolify/install.sh | bash
```

**Pick it if** you want the broadest coverage, the biggest community, and a
single place to run everything on your server.

**Look elsewhere if** you want application services — a database bound to an
environment, end-user auth, transactional email — provided by the platform
rather than deployed as more containers you then wire together yourself.

## Dokploy

Similar territory to Coolify, newer, with a UI many people prefer. Docker
Compose support, databases, multi-server management.

```bash
curl -sSL https://dokploy.com/install.sh | bash
```

Worth knowing: Dokploy is open core rather than fully open source. Most of it is
Apache-2.0, with a `/proprietary` directory under a separate license. Same model
we use, and worth checking against your own requirements in both cases.

**Pick it if** you like Coolify's scope but prefer Dokploy's interface, or you
need multi-server orchestration without reaching for Kubernetes.

## Dokku

The oldest and smallest-surface option, and the most Unix-shaped. Dokku is a
`git push` PaaS built on Docker, driven from the command line, with plugins for
everything else. No UI by default.

**Pick it if** you are comfortable in a terminal, want something that has worked
the same way for a decade, and value a small dependency footprint over features.

**Look elsewhere if** you want a dashboard, or you are setting this up for a
team that does not live in SSH.

## CapRover

Docker plus nginx with a web UI and a one-click app library, around since 2017.
Mature, stable, less actively fashionable than Coolify or Dokploy.

Worth knowing: CapRover's licence is the Apache-2.0 text plus an appendix that
supersedes it where the two conflict, restricting modification and
redistribution of the paid features. GitHub does not classify it as a standard
licence for that reason. Not fully permissive, in other words — the same caveat
that applies to Dokploy and to us.

**Pick it if** you want something proven and straightforward, and the app
catalogue covers what you need.

**Look elsewhere if** you want the largest community and the most one-click
services — Coolify and Dokploy have moved ahead there.

## Kamal

The odd one out, and included because people keep suggesting it in these
threads. Kamal is not a platform — it is a deployment tool from the Rails world
that pushes Docker containers to servers you already own. There is no dashboard,
no build service, no environment management. You bring the server and the
config.

**Pick it if** you want deployment automation without running a platform, and
you are happy configuring things in YAML.

**Look elsewhere if** you expected a UI, previews, or anything resembling the
Vercel workflow. That is not what it is for.

## Stormkit

Stormkit is aimed at a different problem from the platforms above. Coolify and
Dokploy give you somewhere to run containers, and the rest of the product — the
database, the sign-in flow, the transactional emails, the scheduled jobs, the
analytics — is yours to assemble and operate. Stormkit ships those as part of
the platform:

- **[PostgreSQL](/docs/features/database)** attached to an environment, with migrations run on deploy
- **[End-user authentication](/docs/features/authentication)** — email and password, magic links, Google and X OAuth, with session cookies handled for you
- **[Transactional email](/docs/features/mailer)** through a built-in mailer
- **[Periodic triggers](/docs/features/periodic-triggers)** for scheduled work, without a separate cron box
- **[Server-side analytics](/docs/features/analytics)** that do not depend on client-side tracking
- **[API endpoints and server-side rendering](/docs/features/writing-api)**, plus [volumes](/docs/features/volumes) for persistent files

On top of that sits what you would expect from a deployment platform: a
deployment per push, a preview URL per branch, environments with their own
variables and domains, and automatic TLS.

```bash
curl -sSL https://www.stormkit.io/install.sh | bash
```

It is open core: the community edition is AGPL-3.0, some enterprise components
are proprietary and ship in the published binaries.

**Pick it if** you are building a product rather than hosting containers, and
you would otherwise spend the first week wiring up a database, an auth
provider, an email service and a cron runner before writing any features.

**Look elsewhere if** your goal is running a lot of off-the-shelf software you
did not write, or you need a stack with no proprietary components at all. A
container platform fits the first better, and Coolify or Dokku the second.

If that sounds like your fit, the [self-hosting guide](/docs/self-hosting/getting-started)
gets you running on your own server in a few minutes, and the
[Vercel](/vs-vercel) and [Netlify](/vs-netlify) comparisons go deeper on the
migration.

## Which one should you pick

- **Running everything on one box, including off-the-shelf apps you did not write** — Coolify, or Dokploy if you prefer its UI. This is the common case, and where most people arriving from Vercel should start.
- **Terminal-first, minimal, `git push` and nothing else** — Dokku.
- **Proven and boring, with a UI** — CapRover.
- **You have servers and only want deployment automation** — Kamal.
- **Building a product, and you want the database, auth, email and scheduled jobs to come with the platform** — Stormkit.

## What self-hosting actually costs

The honest version: a VPS that can build and run these is typically $5–20 a
month, which is usually less than the platform bill you left. The real cost is
that you now own uptime, upgrades, backups and TLS renewals. Every project here
automates a good deal of that, but none of them makes it someone else's problem
the way a managed platform does.

If your Vercel bill is under about $20 a month and predictable, self-hosting is
probably a worse deal in total cost. It becomes compelling when the bill grows,
when you need something the platform will not run, or when the data has to stay
somewhere specific.

## Frequently asked questions

**Which is the closest thing to Vercel?**
Depends which half of Vercel you mean. For breadth of what you can run on one
server, Coolify is the closest to replacing your whole hosting setup, and it is
the answer most people are looking for. For the deployment workflow
specifically — previews per branch, environments, serverless functions —
Stormkit is the closest in shape (the [Vercel comparison](/vs-vercel) breaks
that down feature by feature).

**Are these really free?**
The software is. The server is not. Every option here needs a machine that stays
running, and several of the projects also sell a managed version.

**Can I self-host Next.js on these?**
Yes, though how much you get out of the box varies. On the container platforms
you run Next.js's standalone Docker output like any other image, and wire up
routing yourself. Stormkit's [self-hosted edition](/docs/self-hosting/getting-started)
supports full Next.js — SSR, API routes and the App Router — without that setup.
Either way, test the specific Next.js features you depend on before committing,
since some are tuned to Vercel's own infrastructure.

**What about Kubernetes?**
If you already run Kubernetes, you likely do not need any of this. These
platforms exist to give you most of what a PaaS provides without a cluster to
operate.
