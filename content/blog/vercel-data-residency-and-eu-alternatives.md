---
title: Vercel data residency and the European alternatives in 2026
description: What data residency on Vercel and Netlify actually covers, why picking an EU region is not the same as sovereignty, and the European and self-hosted options - Scaleway, OVHcloud, Clever Cloud, Hetzner and Stormkit - if you need the data to stay put.
date: 2026-08-13
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1993991074138779648/Up6HP-Jw_reasonably_small.jpg
---

**Disclosure up front: we build Stormkit, one of the options below.** This is the
explainer we kept failing to find while answering procurement questionnaires, so
it is written to be useful even if you finish it and stay where you are. It is
also not legal advice — your DPO gets the final word, not a vendor blog.

Most people arrive at this question from one of two directions. Either a customer
has sent a security questionnaire asking where their data is processed, or
someone in the company has read about the CLOUD Act and wants to know whether
"our region is set to Frankfurt" is an answer. Those turn out to be different
questions with different answers.

## What "data residency" covers, and what it doesn't

A deployment platform is not one thing in one place. It is at least five, and
region settings usually govern only the first:

- **Function execution** — where your server-side code runs. This is what a
  region selector controls, and on Vercel or Netlify you can genuinely pin it to
  an EU region — on Netlify that control is a paid-plan feature, so check what
  your plan includes rather than assuming the dropdown is there.
- **The CDN edge** — where responses are cached and served from. A global edge
  network is the product; cached responses sit in points of presence worldwide
  by design. Anything you cache leaves the region on purpose.
- **Build infrastructure** — where your source is checked out and compiled. Your
  repository, your environment variables and your build logs all pass through
  it.
- **Platform telemetry** — analytics, logs, traces, error reporting. Often a
  different subsystem with different storage, and often not covered by the same
  region control.
- **The control plane** — accounts, teams, audit logs, billing. Almost always
  centralised.

When a questionnaire asks "where is our data processed", it means all five.
When a region dropdown says `fra1`, it means the first.

This is not a criticism of Vercel — the architecture is what makes the product
fast, and both Vercel and Netlify document their sub-processors and offer a DPA.
It is a mismatch between what the control does and what people assume it does.

## Region is not the same as sovereignty

Here is the part that surprises teams. Vercel and Netlify are US companies. Under
the CLOUD Act, a US-headquartered provider can be compelled to produce data it
controls regardless of which country the servers sit in. Choosing an EU region
changes the physical location of the bytes. It does not change who can be served
an order for them.

For a large majority of businesses this is a theoretical concern and an EU region
plus a signed DPA is a perfectly reasonable, defensible position. It stops being
theoretical in a specific and recognisable set of cases:

- Public sector procurement, especially German, French and Dutch, where
  frameworks increasingly ask about provider jurisdiction rather than data
  location
- Healthcare and anything touching patient records
- Financial services under DORA, where you owe regulators a concrete exit plan
  for every critical provider
- Defence, and any customer with their own defence customers
- Contracts that name a specific country, not a region — "data stays in
  Germany" is a stricter promise than "data stays in the EU"

If you are in one of those, the honest chain of reasoning goes: EU region →
EU-headquartered provider → your own infrastructure. Each step removes a
category of exposure and adds work. Most teams need step one. Some need step
two. A few genuinely need step three, and they usually already know it.

One more thing worth getting right rather than assuming, because it is usually
reported as more precarious than it is: transfers to US providers have relied on
the EU–US Data Privacy Framework since 2023, and it is still standing. The
General Court dismissed the first challenge to it in September 2025, holding
that the redress mechanism and the limits on bulk collection were adequate. That
ruling is under appeal at the Court of Justice, and separately the European Data
Protection Board asked the Commission in mid-2026 to reassess whether recent US
constitutional rulings on the independence of federal agencies undermine the
commitments the framework rests on. So: valid today, contested, and not
something to write into a compliance document without checking where the appeal
has got to.

## The European options

| Provider | Country | Shape | Best for |
| --- | --- | --- | --- |
| [Clever Cloud](https://www.clever-cloud.com) | France | Managed PaaS | The closest thing to a European Vercel — you push, it deploys |
| [Scaleway](https://www.scaleway.com) | France | Cloud provider | Containers, managed Postgres, object storage, serverless |
| [OVHcloud](https://www.ovhcloud.com) | France | Cloud provider | The largest EU cloud; strong public-sector track record |
| [Hetzner](https://www.hetzner.com) | Germany | VPS and bare metal | Cheapest capable servers in Europe, if you bring a platform |
| [IONOS](https://www.ionos.com) | Germany | Cloud provider | German public-sector procurement |
| [Exoscale](https://www.exoscale.com) | Switzerland | Cloud provider | Non-EU but adequacy-covered, when Switzerland is specified |
| [Stormkit](https://www.stormkit.io) | Estonia | Self-hosted platform | Vercel's workflow on infrastructure you own, anywhere |

The split matters. Clever Cloud is the only one there that replaces Vercel as a
*product* — a managed PaaS run by a European company. The cloud providers replace
the infrastructure underneath, and you still need something to turn a git push
into a deployment. That is where a self-hosted platform comes in.

### If Supabase is the thing you need a European answer for

It comes up in the same conversation often enough to mention. Supabase is
US-incorporated but offers EU regions and is open source, so the sovereignty
answer is to self-host it on a European provider rather than to switch products.
[Nhost](https://nhost.io) (Sweden) and a self-hosted
[Appwrite](https://appwrite.io) are the alternatives if you would rather have an
EU vendor than run it yourself.

## Where Stormkit fits

Stormkit's answer to this problem is that it is not a region setting — it is
where you install it. The [self-hosted edition](/docs/self-hosting/getting-started)
runs on your own servers, so all five of the layers above land wherever you put
them: a Hetzner box in Nuremberg, a Scaleway instance in Paris, an OVHcloud
tenancy, your own datacentre, or a private cloud VPC that never touches the
public internet.

```bash
curl -sSL https://www.stormkit.io/install.sh | sh
```

The workflow survives the move — a deployment per push, a preview URL per
branch, environments with their own variables and domains, automatic TLS. The
part that matters for this particular question is the sub-processor list. A
typical application answers to a database vendor, an auth vendor, an email
vendor, a scheduler and an analytics provider, and every one of them is a row
you document, assess, and assess again next year. Under Stormkit those are the
platform: [PostgreSQL](/docs/features/database),
[end-user authentication](/docs/features/authentication),
[transactional email](/docs/features/mailer),
[periodic triggers](/docs/features/periodic-triggers) and
[server-side analytics](/docs/features/analytics), all running wherever you
installed it. Shortening that list is frequently a bigger win in a compliance
review than the hosting change that prompted it.

Two honest caveats. **Stormkit Cloud is not the sovereign option** — it runs on
AWS Lambda in the US, and if data location is your requirement then the
self-hosted edition is the one to look at, not our managed service. And
**self-hosting moves the obligation rather than removing it**: uptime, patching,
backups and key management become yours. Stormkit automates a fair amount of
that. It does not make it someone else's problem, and any vendor telling you
otherwise is selling.

**Pick it if** you need the deployment platform itself inside your jurisdiction
or your own network, and you would rather not assemble a PaaS out of parts to get
there.

**Look elsewhere if** an EU region and a signed DPA already satisfy your
requirement. In that case staying on Vercel or Netlify is genuinely the lower-risk
choice, and this whole article is a problem you do not have. If you are only
weighing up one of them against Stormkit, the [Vercel](/vs-vercel) and
[Netlify](/vs-netlify) comparisons go feature by feature.

## Which one should you pick

- **A customer questionnaire asked where data is processed** — set an EU region,
  sign the DPA, document the sub-processors. Do not migrate.
- **You want a European vendor without operating anything** — Clever Cloud.
- **You want European infrastructure and will run the platform yourself** —
  Hetzner, Scaleway or OVHcloud underneath, Stormkit or a container platform on
  top.
- **A contract names a specific country, or you are in defence, health or public
  sector** — self-host, on infrastructure in that country.
- **DORA exit planning** — self-hosting is the exit plan, because the software
  keeps running if the vendor stops.

## Frequently asked questions

**Does Vercel offer data residency?**
It offers region selection for function execution, a DPA, and documented
sub-processors, with additional controls available on enterprise agreements.
Check the current documentation for what your plan includes — the specifics move.
The structural point stands regardless of plan: the edge network is global, and
the company is subject to US jurisdiction.

**Is choosing an EU region enough for GDPR?**
Usually yes, for a normal business, alongside a DPA and a transfer mechanism.
GDPR does not require EU-only processing. It requires a lawful basis for
transfers and appropriate safeguards. Sovereignty requirements that go further
than GDPR almost always come from a contract, a regulator or a procurement
framework rather than from the regulation itself.

**Is Vercel available on-premise?**
Not as an install you run yourself. There is a private-beta bring-your-own-cloud
option on AWS that puts the compute and data in your own account while Vercel
keeps operating the control plane, which covers "our tenancy" but not "our
building" and not an air gap. The [on-premise write-up](/blog/vercel-on-premise)
covers the distinction and what it takes to get the same workflow inside your
own network.

**What about the edge network — can I turn it off?**
You can reduce what is cached, but not being a global CDN defeats much of the
point of the platform. If nothing may leave the region, you want an
origin-in-region architecture rather than an edge-first one, which is an argument
for a different shape of platform rather than a different setting on this one.

**Does self-hosting make us compliant?**
No. It removes a set of transfer and jurisdiction questions and hands you
everything the provider was doing — patching, backups, access control, incident
response. That is a better trade for some organisations and a worse one for
others. It is never automatic.
