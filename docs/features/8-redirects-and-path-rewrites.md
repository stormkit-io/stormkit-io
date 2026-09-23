---
title: Redirects and path rewrites
description: Handle redirects and path rewrites with Stormkit.
---

# Redirects and Path Rewrites

<section>
Stormkit is able to handle the path rewrites and redirects on the load balancer level. In order to make use of this feature, create a `redirects.json` file at root level of your repository. This file will be parsed on each deployment, hence if you change this file previous deployments won't be affected. The syntax is as follows:

```json
[
  {
    "from": "string", // (required): The path. Supports regexp syntax.
    "to": "string", // (required): The destination path.
    "status": "number", // (optional): The HTTP Status Code for redirect. Default is empty.
    "assets": "boolean", // (optional): Whether to apply the redirect/rewrite to any static file that is not an html file. Default is false.
    "hosts": "Array<string>", // (optional): When provided, the redirect rule will apply only when the host name matches.
    "data": "object" // (optional): Fetches a JSON document and fills it into the page the rule rewrites to. See Dynamic pages.
  }
]
```

</section>

## Path rewrites

<section>

If you omit the `status` property, or provide a `status` different than `3xx`, Stormkit will not redirect the
request but will simply rewrite the path.

```json
[
  {
    "from": "/my-path/*",
    "to": "/my-new-path/$1"
  }
]
```

In this case, all requests coming to `/my-path` will be served as if they were coming to `/my-new-path`.

</section>

## Proxies

<section>

You can also use redirects as a proxy. If your redirect is an absolute URL (starting with `http`),
the request will be proxied.

```json
[
  {
    "from": "/my-path/*",
    "to": "https://example.com/my-new-path/$1",
    "status": 200
  }
]
```

In this case, all requests coming to `/my-path` will be proxied to `https://example.com/my-new-path/*`.

</section>

## Dynamic pages

<section>

A static site sometimes needs one dynamic route: a page per video, product or profile, with the right `<title>` and Open Graph tags for crawlers and link previews. Add a `data` loader to a rewrite rule. Stormkit fetches a JSON document for each request and fills its values into the HTML file the rule rewrites to. Your markup, styles and asset URLs stay in your build, and your API only returns JSON.

```json
[
  {
    "from": "/v/*",
    "to": "/videos.html",
    "data": {
      "url": "https://api.example.com/v/$1.json"
    }
  }
]
```

A request to `/v/abc123` serves `/videos.html` from your deployment, filled with the document at `https://api.example.com/v/abc123.json`.

| Property                 | Required | Description                                                                                                                         |
| ------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `data.url`               | Yes      | The JSON document to fetch. Must be `https`. Wildcard captures from `from` are available as `$1`, `$2`, and so on.                   |
| `data.passthroughStatus` | No       | When `true`, an error status from the API is returned to the visitor instead of the page. Default is `false`.                        |

Captures are URL-encoded before they are inserted, so a visitor's path can add path segments to `data.url` but cannot add a query string or fragment. A path whose capture contains a `.` or `..` segment does not match the rule.

A loader only works on a rewrite to a file in your deployment. It is rejected on a proxy rule (a `to` starting with `http`) and on a `3xx` redirect.

### Placeholders

In the HTML file, reference the document with `{{data.<field>}}`. Use dots for nested fields and `:-` for a default value:

```html
<title>{{data.title:-Videos}} · Example</title>
<meta property="og:title" content="{{data.title}}" />
<meta property="og:image" content="{{data.thumbnail:-/og.png}}" />
<meta name="robots" content="{{data.robots:-noindex}}" />
<span>{{data.owner.name}}</span>
<script id="__sk_data__" type="application/json">{{data}}</script>
```

- `{{data}}` renders the whole document as JSON. Put it in a `<script type="application/json">` block and read it from your client code to hydrate the page.
- Values from the API are HTML-escaped, so they are safe in text and in attributes. Your own default values are inserted as written.
- The default applies when a field is missing, `null` or an empty string. It does not apply to `false` or `0`: `{{data.public:-true}}` renders `false` when the API returns `false`.
- A placeholder with no value and no default renders as an empty string.
- Numbers render without a trailing decimal (`12`, not `12.0`). Objects and arrays render as JSON.

### When the API fails

Your page is never blocked on the API for long. The fetch times out after 2 seconds and reads at most 1 MB.

- **The API is unreachable, times out or returns invalid JSON:** the page is served with every placeholder set to its default.
- **The API returns an error status (e.g. `404`):** by default the page is served with defaults, and the error response body is ignored. With `"passthroughStatus": true`, the visitor gets that status instead: a `404` serves your deployment's 404 page (`/404.html`, `/error.html`, or the custom error file), and any other status is returned with an empty body.

### Caching

Stormkit does not cache pages with a loader or their JSON documents: every request calls your API. Stormkit also does not send `ETag` or `Last-Modified` for these pages, so browsers and CDNs always get the current data. If an endpoint is expensive, cache it in your API.

### Request headers

The request to your API carries the visitor's details, so it can count views and tell people apart from bots and link unfurlers:

| Header            | Value                                                                                  |
| ----------------- | -------------------------------------------------------------------------------------- |
| `User-Agent`      | The visitor's `User-Agent`                                                             |
| `X-Forwarded-For` | The visitor's IP address                                                               |
| `X-Real-IP`       | The visitor's IP address                                                               |
| `Accept`          | `application/json`                                                                     |

Identical requests from the same visitor that arrive at the same time share a single call to your API. Requests from different visitors are never combined.

### More than one route

Stormkit applies the first rule that matches, and `*` also matches `/`. List more specific rules first: here `/v/abc/embed` would match `/v/*` if that rule came first.

```json
[
  {
    "from": "/v/*/embed",
    "to": "/embed.html",
    "data": { "url": "https://api.example.com/v/$1.json" }
  },
  {
    "from": "/v/*",
    "to": "/videos.html",
    "data": { "url": "https://api.example.com/v/$1.json" }
  }
]
```

### Hide the template file

The file your rule rewrites to is a normal file in your deployment, so `/videos.html` can also be opened directly, showing unfilled placeholders. Redirect it away. List this rule **after** the loader rule:

```json
[
  {
    "from": "/v/*",
    "to": "/videos.html",
    "data": { "url": "https://api.example.com/v/$1.json" }
  },
  {
    "from": "/videos(.html)?",
    "to": "/",
    "status": 301
  }
]
```

Stormkit applies the first rule that matches, so `/v/abc123` still rewrites to the template while direct requests to `/videos` and `/videos.html` are redirected.

### Self-hosted instances

A loader only connects to public addresses. On a self-hosted instance whose API runs on a private network, set `STORMKIT_PAGE_DATA_INSECURE=true` to allow private addresses and plain `http` URLs. See [Advanced Configuration](/docs/self-hosting/advanced-configuration).

</section>

## Redirect non-www to www

<section>

```json
[
  {
    "from": "stormkit.io",
    "to": "www.stormkit.io",
    "assets": true,
    "status": 301
  }
]
```

</section>

## SPA config

<section>

```json
[
  {
    "from": "/*",
    "to": "/index.html"
  }
]
```

The above example will rewrite all requests to `index.html`. By setting `assets` false This is useful for single page applications.

</section>

## Regexp

<section>

```json
[
  {
    "from": "/documentation/*/page/*",
    "to": "/docs/$1/$2"
  },
  {
    "from": "/documentation$",
    "to": "/docs"
  }
]
```

You can use `regexp` syntax for redirects. The example above creates two redirects.

1. The first one will redirect `/documentation/welcome/page/getting-started` to `/docs/welcome/getting-started`.
2. The second will redirect `/documentation` to `/docs`.

Note the `$amp;` sign at the end of the string. That sign simply tells to redirect only the path `/documentation` and not anything that contains `/documentation`.

</section>

## Matching host names

```json
[
  {
    "from": "/path",
    "to": "/new-path",
    "hosts": ["example-a.org"]
  },
  {
    "from": "/path",
    "to": "/different-path",
    "hosts": ["example-b.org", "example-c.org"]
  }
]
```

If you have multiple domains configured for your environment, you can specify for which host name the redirect rule should
apply to.

The example above will rewrite the `/path` to `/new-path` for `example-a.org` and to `/different-path` for `example-b.org` and `example-c.org`.

## Redirecting API Routes

<section>

Please note that if your application contains `API` routes, paths starting with `/api` will not be matched.
This is to allow `/api` routes to handle the redirect themselves.
If you do not have any `API` function, this rule does not apply.

You can configure the API routes through the [Serverless configuration section](/docs/deployments/configuration).

</section>

## Custom 404 pages

By Default, when a page is not found, Stormkit will try to serve `/404.html` or `/error.html` if any of these files are found in your deployment. You can customize this behaviour as follows:

1. Go to **Environment Config** > **Redirects**
1. Find the `Custom Error File` field
1. Type the file that should be served instead (e.g. /index.html).

This setting will be applied to **all** of your deployments and take effect instantly. There is no need for a deployment.

**Note** If you have `API` routes configured, the custom error file will not be applied to the paths starting
with your `API Path` - which is `/api` by default.

**Note** Similarly, if you have serverless side logic, the custom error file will not be applied.

## Environment level redirects

You can specify the same rules at an environment level. To do so:

1. Go to **Environment Config** > **Redirects**
1. Switch **Overwrite redirects**
1. Specify the rules from the Redirects Editor and
1. Click save.

These rules will be applied to **all** of your deployments and take effect instantly. There is no need for a deployment.
