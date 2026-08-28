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

That is the empirical case. The structural one explains why it went that way.

`/llms.txt` is a hand-curated index, authored once and maintained by hand forever — and authored artifacts rot. It is the copy humans never look at, so it drifts, and six months later the agent is reading a description of a feature you removed. Stacked on top of that is discovery: a client landing on `https://yoursite.com/docs/deploying` has no reliable way to know an agent-friendly twin exists somewhere else. Does it guess `.md`? Fetch `/llms.txt` and parse it for a mapping? Every tool invents its own convention, and the page author has to know all of them.

## HTTP already solved this in 1999

The web has always served more than one representation of the same thing. That is what the `Accept` header is for. The client says what formats it can use, the server picks the best match, and the URL never changes. This is content negotiation, and it has been in the spec since HTTP/1.1.

An agent that wants markdown says so; a browser asks for what it always asks for:

```
GET /docs/deploying HTTP/1.1
Accept: text/markdown

GET /docs/deploying HTTP/1.1
Accept: text/html
```

Same URL. One canonical resource, two representations, and the client picks. No sidecar index to discover, no per-tool convention — just the header the client was already sending anyway. `text/markdown` is a real registered media type (RFC 7763); this is not a hack, it is the mechanism working as designed.

It is easy to oversell this, so be precise about what it removes. The markdown still has to exist — in our implementation it is a real file in the deployment, reachable at `/docs/deploying.md` if you ask for it directly. What negotiation abolishes is not the second file but the second *interface*: that file stops being an address anyone has to discover, guess, or agree on a convention for.

Drift is solved by your build, not by any header. If the `.md` is generated from the same source that renders the HTML, the two cannot disagree. Hand-write both and they will drift, and no header will save you. So the honest claim is narrower than "no second tree": the second tree should be build output, never something a human maintains.

This is not theoretical any more. When Checkly [tested seven coding agents](https://www.checklyhq.com/blog/state-of-ai-agent-content-negotation/) in February 2026, Claude Code, Cursor, and OpenCode all sent `Accept: text/markdown` when fetching documentation. Codex, Gemini CLI, Copilot, and Windsurf did not — they asked for HTML or sent a generic `*/*`. Three out of seven is not a mandate, but it is three more than were doing it a year ago, and the direction is one-way.

## "But llms.txt solves discovery, and Accept doesn't"

This is the strongest objection, and the most common position in the field is that the two approaches are complementary: content negotiation is the transport layer, llms.txt is the discovery layer. Accept tells the server what format to send. It says nothing about which pages exist.

That is true, and it is not an argument for llms.txt. Discovery is also already solved, and has been for just as long — it is `sitemap.xml`. Every page you want a machine to find, listed in a file every crawler already fetches, generated by your build, describing pages rather than duplicating them.

So llms.txt is not reinventing one existing standard. It is reinventing two. Sitemap for discovery, Accept for representation, and both of them already work in every client and cache on the internet.

## This should not be an AI feature

Here is where I would go further than most posts on this topic: content negotiation for text is not an agent optimization. It is how the web should have been serving content all along, and agents are simply the first client population large enough to force the issue.

The same is true for anything that wants content rather than chrome: text browsers, reader modes, RSS and newsletter extractors, e-ink readers, archival tools, anything on a metered connection.

The saving is not marginal. Gzipped, `/docs/deployments/configuration` goes over the wire as 16.1 KB of HTML against 1.5 KB of markdown — and that is the document alone, before the JavaScript a browser then pulls down and a text client never needs.

For an agent, though, bandwidth is the wrong thing to measure, and measuring it hides the cost that matters. Gzip is very good at HTML precisely because HTML is repetitive: the same tags, the same class names, the same nav on every page. It squeezes that page from 130 KB to 16 KB. But the model never sees the compressed bytes. It reads the uncompressed document and pays per token. Compression flatters the wire and does nothing for the context window.

The same two pages, fetched today and counted with `o200k_base`:

| Page | HTML | Markdown |
| --- | --- | --- |
| `/docs/deployments/configuration` | 43,096 tokens | 811 tokens |
| `/docs/api/deployments` | 62,315 tokens | 6,218 tokens |

53x and 10x, against the 11x and 5x gzip suggested. Strip every `<script>` and `<style>` first, as any competent agent will, and the HTML drops to 20,826 and 34,057 — still 26x and 5.5x. What survives stripping is the markup itself, interleaved with the prose you wanted, and no preprocessing gets rid of that.

Sit with the first row. A page whose content is 811 tokens costs 43,096 to fetch as HTML: 98% of what the agent pays for is not the answer it came for.

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

Here is the part I did not see coming, and the reason I am less glib about "just use the header."

To negotiate on `Accept`, you have to parse `Accept` — and `Accept` is attacker-controlled input. A real header from Chrome looks like `text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8`: four entries, parsed in microseconds. Nothing about it suggests danger. But nothing in HTTP caps how big it can be either, and the default ceiling in most servers is around a megabyte. Our parser split on every comma and gave each entry its own sub-parse — perfectly reasonable for four entries. Fed a megabyte of nothing but commas, it did a million allocations and burned tens of milliseconds of CPU per request before serving anything.

Now combine that with the header we just called mandatory. `Vary: Accept` puts an attacker-controlled value into the CDN cache key, so appending a junk quality value — `;q=0.900001`, `;q=0.900002` — guarantees a miss on every request and sends 100% of a flood straight to origin. On a shared edge, that is one tenant's crafted header degrading every tenant on the node. The header that makes negotiation correct is the header that makes the amplification reliable.

The fix is boring, which is how you want security fixes. Cap the header length well below the default, cap the entries parsed, and bound the split *before* it allocates rather than after. An oversized header is treated as if no preference was sent — which per the spec means "anything is acceptable," so the client still gets a valid page.

The lesson is not that content negotiation is dangerous. It is that the header you negotiate on is untrusted input on a hot path, and deserves the bounds you would put on any other request field. llms.txt sidesteps this only by accident, and trades it for the freshness and discovery problems above.

## How it works on Stormkit

We built this into Stormkit because we kept hitting the drift problem on our own docs. It is opt-in per environment. With the toggle on, a page is negotiable exactly when the deployment already contains its markdown twin — `/docs/deploying.md` next to `/docs/deploying` — so negotiation can never invent a URL that does not exist. Markdown to whoever asks for it, HTML to browsers, `Vary: Accept` on both so caches never cross the wires. Ties fall back to HTML, because a client that names both without a preference is a browser, not an agent.

That leaves the obvious gap: a site whose pipeline does not emit markdown gets nothing until it does. So there is a second toggle, **Convert pages without a .md file**, which converts the page's own HTML on request. A twin you publish yourself always wins — conversion only fills a gap — and a page that renders in the browser has no prose to convert, so it keeps serving HTML rather than an empty document. Converted pages are cached per deployment, computed once, and can never describe a page that has since changed.

Here is the whole setup, end to end, in under a minute:

<iframe width="560" height="315" src="https://www.youtube.com/embed/CTBF5u3LDjw" title="Serve Markdown to AI agents from any site — Stormkit content negotiation" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

That is all content negotiation ever promised. It just took a new kind of client to make us use it.
