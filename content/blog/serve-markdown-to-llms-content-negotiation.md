---
title: "Why we joined the content negotiation camp"
date: 2026-08-23
description: Agents reading your docs want text, not rendered HTML. The industry's first answer was llms.txt, and the data says it did not work. We shipped content negotiation instead — the Accept header, one URL, no convention to discover. Here is why, plus the Vary caching gotcha and the denial-of-service bug we shipped and fixed on the way.
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1993991074138779648/Up6HP-Jw_reasonably_small.jpg
---

A growing share of the traffic to your docs is no longer a person. It is an agent — a coding assistant pulling your API reference, a research tool summarizing your guide, an LLM answering a question about your product. And every one of them is doing the same wasteful thing: downloading a full HTML page, then throwing away the nav, the scripts, the cookie banner, and the markup, to get at the few hundred words of text it actually wanted.

The prose was probably written in markdown to begin with. Your site rendered it to HTML for browsers. The agent would much rather have the markdown back.

There are two answers on the table for how to give it to them. We picked the older one, and we hit two problems on the way that nobody had written down. This is why we chose it, and what it cost.

## The first answer was llms.txt, and the numbers are in

The `llms.txt` convention says: publish a separate file — sometimes a single `/llms.txt`, sometimes a parallel tree of `*.md` files — that holds a markdown copy of your content for machines to read.

Two years on, the evidence is not kind. [Ahrefs analyzed 137,210 domains](https://ahrefs.com/blog/llmstxt-study/) in June 2026 and found that 28% of them published an llms.txt file — and that 97% of those files received zero traffic in May. Not "little traffic." None. Of the 3% that were fetched at all, almost every request came from a crawler rather than a reader.

Google's John Mueller [put it bluntly on Reddit](https://www.searchenginejournal.com/google-says-llms-txt-comparable-to-keywords-meta-tag/544804/) as early as April 2025: "AFAIK none of the AI services have said they're using LLMs.TXT... To me, it's comparable to the keywords meta tag" — a signal search engines abandoned precisely because a site describing itself is too easy to game. No major AI provider has committed to reading the file since. The crawlers that were supposed to consume it are, overwhelmingly, still fetching your HTML.

That is the empirical case. The structural case matters more, because it explains *why* it went that way.

The single-file flavour is the worst of it: `/llms.txt` is a hand-curated index, authored once and maintained by hand forever. Authored artifacts rot. In practice it drifts from the site it describes, because it is the copy humans never look at, and six months later the agent is reading a description of a feature you removed.

And there is a discovery problem stacked on the freshness one. A client that lands on `https://yoursite.com/docs/deploying` has no reliable way to know an agent-friendly twin exists somewhere else. Does it guess `.md`? Fetch `/llms.txt` and parse it for a mapping? Every tool invents its own convention, and the page author has to know all of them.

## HTTP already solved this in 1999

The web has always served more than one representation of the same thing. That is what the `Accept` header is for. The client says what formats it can use, the server picks the best match, and the URL never changes. This is content negotiation, and it has been in the spec since HTTP/1.1.

An agent that wants markdown says so:

```
GET /docs/deploying HTTP/1.1
Accept: text/markdown
```

A browser asks for what it always asks for:

```
GET /docs/deploying HTTP/1.1
Accept: text/html
```

Same URL. One canonical resource, two representations, and the client picks. No sidecar index to discover, no per-tool convention — just the header the client was already sending anyway. `text/markdown` is a real registered media type (RFC 7763); this is not a hack, it is the mechanism working as designed.

It is worth being precise about what this does and does not remove, because it is easy to oversell. The markdown still has to exist. In our implementation it is a real file in the deployment, and it is reachable on its own at `/docs/deploying.md` if you ask for it directly. Content negotiation does not abolish the second file.

What it abolishes is the second *interface*. That file stops being an address anyone has to discover, guess, or agree on a convention for. Clients ask for the page they already know about and state the format they want, and every one of them does it the same way.

Drift, meanwhile, is solved by your build rather than by any header. If the `.md` is generated from the same source that renders the HTML — the normal case for a docs site, where markdown is the source and HTML is the artifact — the two cannot disagree. If you hand-write both, they will drift, and no amount of content negotiation will save you. So the honest claim is narrower than "no second tree": the second tree should be build output, never something a human maintains.

This is not theoretical any more. When Checkly [tested seven coding agents](https://www.checklyhq.com/blog/state-of-ai-agent-content-negotation/) in February 2026, Claude Code, Cursor, and OpenCode all sent `Accept: text/markdown` when fetching documentation. Codex, Gemini CLI, Copilot, and Windsurf did not — they asked for HTML or sent a generic `*/*`. Three out of seven is not a mandate, but it is three more than were doing it a year ago, and the direction is one-way.

## "But llms.txt solves discovery, and Accept doesn't"

This is the strongest objection, and the most common position in the field is that the two approaches are complementary: content negotiation is the transport layer, llms.txt is the discovery layer. Accept tells the server what format to send. It says nothing about which pages exist.

That is true, and it is not an argument for llms.txt. Discovery is also already solved, and has been for just as long — it is `sitemap.xml`. Every page you want a machine to find, listed in a file every crawler already fetches, generated by your build, describing pages rather than duplicating them.

So llms.txt is not reinventing one existing standard. It is reinventing two. Sitemap for discovery, Accept for representation, and both of them already work in every client and cache on the internet.

## This should not be an AI feature

Here is where I would go further than most posts on this topic: content negotiation for text is not an agent optimization. It is how the web should have been serving content all along, and agents are simply the first client population large enough to force the issue.

Look at reader mode. Safari and Firefox both ship a feature whose entire job is to strip a page back to its content — and they implement it by downloading the whole page, then *guessing*, running heuristics to work out which DOM subtree is the article. That is a hack made necessary by the absence of a way to just ask. A browser in reader mode should send `Accept: text/markdown` and be done.

The same is true for anything that wants content rather than chrome: text browsers, RSS and newsletter extractors, e-ink readers, archival tools, anything on a metered connection.

The bandwidth difference is not marginal. Two pages from our own docs, gzipped, as served today:

| Page | HTML | Markdown |
| --- | --- | --- |
| `/docs/deployments/configuration` | 16.1 KB | 1.5 KB |
| `/docs/api/deployments` | 24.8 KB | 5.2 KB |

Roughly 11x and 5x — and that is the document alone, before the JavaScript bundles a browser then pulls down and a text client never needs.

One caveat worth stating plainly, because it gets muddled in this discussion: this is a routing mechanism, not an accessibility feature. Screen readers are not HTTP clients — they sit on top of a browser and read the accessibility tree, so they never issue a request of their own and cannot ask for markdown. And they would not want to. Landmarks, ARIA roles, `lang`, heading hierarchy, table header scoping — the semantics assistive technology depends on live in HTML and mostly do not survive a conversion to markdown. Serving markdown to clients that ask for it does not reduce your obligation to write good HTML by one line.

## "Isn't this cloaking?"

If you have followed this topic you have seen the pushback. In February 2026 Google's John Mueller [called serving markdown to LLM crawlers "a stupid idea"](https://www.searchenginejournal.com/googles-mueller-calls-markdown-for-bots-idea-a-stupid-idea/566598/), and Microsoft's Fabrice Canel [warned that dedicated bot pages are cloaking](https://searchengineland.com/google-bing-dont-recommend-seperate-markdown-pages-for-llms-468365) and that search engines will crawl both versions to check they match. Mueller's reasoning: LLMs have parsed HTML since day one, so why serve a page no user ever sees?

It is worth reading what they were actually responding to. The thread that set it off described middleware that sniffs for `GPTBot` and `ClaudeBot` user agents and serves those requests different content. That *is* cloaking, by the ordinary definition. The server decides what you get based on who it thinks you are, and no human can obtain the bot version.

Accept-based negotiation inverts every part of that. The client declares a preference rather than an identity. Any client can request either representation. A person running `curl -H 'Accept: text/markdown'` receives byte-for-byte what the agent receives. There is no hidden variant, because there is no variant keyed to who you are — only to what you asked for.

The distinction is not a loophole, it is the whole difference between negotiation and discrimination. And it answers Mueller's question, too: the page is not one no user sees. It is the same page, and any user can see it.

## The catch nobody mentions: Vary

Content negotiation has one sharp edge, and it is the reason a lot of teams quietly reach for a separate URL instead. The moment a response depends on a request header, every cache in front of you — your CDN, the browser cache, any intermediary — needs to know, or it will serve the wrong representation to the wrong client.

Concretely: an agent requests `/docs/deploying` with `Accept: text/markdown`, your CDN caches the markdown, and the next human to hit that URL gets raw markdown in their browser. Or the reverse, and the agent gets a wall of HTML. The fix is one header:

```
Vary: Accept
```

`Vary: Accept` tells every cache that the `Accept` header is part of the cache key, so the two variants are stored and served separately. It is one line, it is mandatory, and forgetting it is the single most common way content negotiation goes wrong in production. If you take one operational thing from this post: negotiate on `Accept`, and you *must* emit `Vary: Accept` on every negotiated response — not just the markdown one.

## When Accept becomes an attack

Here is the part I did not see coming, and the reason I am a little more humble about "just use the header."

When you negotiate on `Accept`, you have to parse `Accept`. And `Accept` is attacker-controlled input. A real header from Chrome looks like `text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8` — four entries, a couple of quality values, parsed in microseconds. Nothing about it suggests danger.

But nothing in HTTP caps how big that header can be. The default ceiling in most servers is around a megabyte. Our parser split the header on every comma and gave each entry its own sub-parse — perfectly reasonable for four entries. Fed a one-megabyte header that is nothing but a million commas, the same code did a million allocations and burned tens of milliseconds of CPU, per request, before serving anything.

Now combine that with the header we just told you is mandatory. `Vary: Accept` puts an attacker-controlled value into the CDN cache key. So an attacker can append a junk quality value — `;q=0.900001`, `;q=0.900002`, and so on — and guarantee a cache miss on every single request, sending 100% of a flood straight to origin. On a shared, multi-tenant edge, that is one tenant's crafted header degrading every tenant on the node. The `Vary` header that makes negotiation correct is the same header that makes this amplification reliable.

The fix is boring, which is how you want security fixes to be. Cap the header length well below the default (a legitimate `Accept` header is a few hundred bytes), cap the number of entries parsed (real ones have fewer than a dozen), and bound the split *before* it allocates rather than after. An oversized header is treated as if no preference was sent at all — which, per the spec, means "anything is acceptable," so the client still gets a valid page. We also set an explicit maximum header size on the edge, below the language default, which additionally caps the HTTP/2 header-list size that derives from it.

The lesson is not that content negotiation is dangerous. It is that the header you negotiate on is untrusted input on a hot path, and it deserves the same bounds you would put on any other request field. llms.txt sidesteps this only by accident, and trades it for the freshness and discovery problems above.

## How it works on Stormkit

We built this into Stormkit because we kept hitting the drift problem on our own docs. It is opt-in per environment. When you enable it, a page is negotiable exactly when the deployment already contains its markdown twin — `/docs/deploying.md` next to `/docs/deploying` — so negotiation can never invent a URL that does not exist. A client asking for `text/markdown` gets the markdown, a browser gets the HTML, and both responses carry `Vary: Accept` so caches never cross the wires. Ties fall back to HTML, because a client that names both without a preference is a browser, not an agent.

The whole thing is one toggle and shipping the `.md` files your build already produces — which is the requirement worth stating plainly: if your pipeline does not emit markdown alongside the HTML, this gives you nothing until it does. What you get for it is that those files stop being a second interface. No sidecar index, no per-tool convention — the same URL, answering whoever asks in the format they asked for.

That is all content negotiation ever promised. It just took a new kind of client to make us use it.
