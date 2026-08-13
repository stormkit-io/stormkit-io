---
title: Vercel on-premise - what it takes to run the same workflow yourself
description: Vercel's only self-managed option is a private-beta bring-your-own-cloud on AWS, so running deployments inside your own network means replacing it, not installing it.
date: 2026-08-13
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1993991074138779648/Up6HP-Jw_reasonably_small.jpg
---

**Disclosure up front: we build Stormkit, one of the options below.** The short
answer to the question in the title is a fact rather than a pitch, so it goes
first.

**Vercel does not offer an on-premise edition you can install in your own
datacentre.** What it offers instead is *bring your own cloud* on AWS:
your compute, build artifacts and application data run inside your own AWS
account and VPC, while Vercel continues to operate the control plane on top of
them. At the time of writing that is a private beta, AWS only, and not something
you can sign up for — you talk to their sales team. Netlify has no generally
available self-managed edition either.

That distinction is the whole article. Bring your own cloud answers "our account,
our VPC, our security group". It does not answer "our building", it does not
answer "no route to the public internet", and it does not answer "the platform
keeps working if the vendor stops" — because the control plane is still theirs
and still hosted. If your requirement is one of those three, or if you need
something generally available today on a cloud other than AWS, you are not
looking for a licence — you are looking for a replacement.

That is a more tractable problem than it sounds, because the thing you actually
depend on is a workflow, and the workflow is made of about six parts.

## What "on-premise" means in practice

Worth pinning down, because three quite different requirements arrive under the
same word and they cost wildly different amounts of effort:

- **Your own hardware** — a rack in your building, or a colocated cage. Rare
  now, and usually driven by a contract or a regulator.
- **Your own cloud tenancy** — a private VPC in AWS, Azure, GCP, OVHcloud or
  Hetzner, where "on-premise" means "our account, our network, our security
  group" rather than "our building". This is what most people mean, it is by
  far the easiest of the three, and it is the one Vercel's bring-your-own-cloud
  beta is aimed at.
- **Air-gapped** — no route to the public internet at all. Genuinely hard, and
  the section below is mostly about why.

If you have not yet worked out which one you are being asked for, do that before
choosing any tooling. The gap between the second and the third is larger than
the gap between any two platforms you might pick.

## What you are actually replacing

Strip the marketing off and the Vercel workflow is:

1. **A push triggers a build.** A webhook from GitHub, GitLab or Bitbucket, a
   checkout, an install, a build command.
2. **Every branch gets a URL.** Preview deployments are the feature people miss
   most when they leave, and the one most self-hosted setups quietly drop.
3. **Environments carry their own configuration.** Variables and domains per
   environment, not a single `.env` on a server.
4. **Something serves the result.** Static assets, SSR, API routes — with
   sensible caching and no cold-start surprises.
5. **TLS renews itself.** Nobody wants to be the person who forgot.
6. **Rollback is one click.** The previous deployment is still there and going
   back does not mean rebuilding.

Everything else you might miss — a database, auth, cron, transactional email —
sits alongside the platform rather than inside it. Whether they come with your
replacement or become four more things to run is the main thing that separates
the options.

## The parts that get hard behind a firewall

This is the section that tends to be missing from write-ups, and where projects
lose their schedule.

**Builds need the internet even when your application does not.** `npm install`
reaches out to a registry. Docker pulls base images. A build platform inside an
air-gapped network needs a package mirror — Verdaccio, Nexus or Artifactory —
and a registry mirror, and someone to keep both current. This is usually the
single largest piece of work, and it has nothing to do with which deployment
platform you chose.

**TLS is different inside a private network.** Automatic certificates normally
work by proving control of a public domain over HTTP. With no inbound path from
the internet, that fails, and the answer is to stop issuing certificates on the
box: bring your own, from an internal CA with the root distributed to your
machines, or issued out-of-band and installed. Whichever platform you pick, check
that it accepts a certificate you supply per domain rather than assuming it can
always fetch one. Decide this before installation, not after.

**Git has to be reachable from the builder**, not just from developer laptops. A
self-hosted GitLab or a network path to your provider.

**Secrets probably already have an owner.** If Vault or a cloud secrets manager
is the standard in your organisation, check how the platform expects to receive
environment variables before you commit to it.

**Build capacity becomes a resource you plan.** On a managed platform, ten
people merging at once is someone else's problem. On one machine, it is a queue.
Either size for the peak or push builds out to your existing CI.

## Putting it back together

The [self-hosted Vercel alternatives roundup](/blog/best-self-hosted-vercel-alternatives)
compares the platforms in depth — Coolify, Dokploy, Dokku, CapRover, Kamal and
Stormkit — and the choice there does not change much just because the network is
private. The short version: container platforms if you are running a lot of
software you did not write, an application platform if you are running a product
you did.

Where Stormkit is relevant to this particular question is that it installs onto
machines you control and expects nothing from us at runtime:

```bash
curl -sSL https://www.stormkit.io/install.sh | sh
```

It ships as Docker images for `amd64` and `arm64`, installs via Docker Compose
on a single machine or Docker Swarm across several, and is tested on Ubuntu,
Debian, Fedora, Rocky Linux and macOS. Preview URLs per branch, environments
with their own variables and domains, and one-click rollback all survive the move
— which is the part people assume they are giving up. Builds run on the
`workerserver` service by default, or you can hand them to
[GitHub Actions](https://github.com/stormkit-io/runner) if you would rather your
existing CI owned that. On the TLS question above, domains with no public inbound
path take a [custom certificate](/docs/features/custom-certificates) you supply
as PEM, per domain, instead of automatic issuance.

The services that usually arrive as separate vendors are part of it:
[PostgreSQL](/docs/features/database) attached to an environment with migrations
on deploy, [end-user authentication](/docs/features/authentication),
[transactional email](/docs/features/mailer),
[periodic triggers](/docs/features/periodic-triggers),
[persistent volumes](/docs/features/volumes) and
[server-side analytics](/docs/features/analytics). In a private network that
consolidation matters more than it does on the public internet, because every
external service you remove is one less firewall exception to justify.

**Pick it if** you want the Vercel workflow specifically — previews,
environments, rollbacks — inside your own network, with the application services
included.

**Look elsewhere if** what you actually need is somewhere to run a lot of
off-the-shelf containers. Coolify covers that better, and the
[roundup](/blog/best-self-hosted-vercel-alternatives) explains why.

If the driver is jurisdiction rather than network topology, the
[data residency write-up](/blog/vercel-data-residency-and-eu-alternatives) is the
more directly useful one — the answer there is often an EU region rather than a
migration.

## Frequently asked questions

**Can I buy Vercel Enterprise and run it on our servers?**
Not on your own servers. The closest thing is bring your own cloud, which places
the compute and data in your AWS account while Vercel runs the control plane —
private beta, AWS only, enterprise sales rather than self-serve. There is still
no build of Vercel you install on hardware you own, and nothing that runs
without a control plane operated by Vercel.

**Is a private VPC good enough to count as on-premise?**
For most auditors and most contracts, yes — the requirement is usually about
control and network isolation rather than about owning the building. Check the
actual wording you are being held to. "Our own infrastructure" and "our own
premises" are different promises, and only one of them requires a rack.

**Does Stormkit work fully air-gapped?**
The platform installs and runs on machines with no inbound internet access. The
harder question is builds, which need a package registry and a container
registry — mirror both internally and you are in business. If you are planning a
genuinely air-gapped deployment, [talk to us](/contact) first rather than
discovering the gaps during a pilot.

**What happens if the vendor disappears?**
This is the underrated argument for self-hosting and the one regulators ask
about directly under DORA. Software running on your machines keeps running. The
Stormkit community edition is AGPL-3.0, so the source is available regardless of
what happens to the company; some enterprise components are proprietary and ship
in the published binaries.

**How much machine do we need?**
Less than people expect for serving, more than people expect for building. The
serving side is comfortable on a small VPS. Builds are CPU and memory hungry in
bursts — size for your worst simultaneous merge, or offload to CI.
