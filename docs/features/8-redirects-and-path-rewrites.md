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
    "hosts": "Array<string>" // (optional): When provided, the redirect rule will apply only when the host name matches.
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
