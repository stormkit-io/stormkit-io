---
title: Custom headers
description: Custom headers enable you to modify or supplement the default HTTP headers provided by Stormkit when a client requests your site.
keywords: custom headers
---

# Custom Headers

By default, Stormkit looks for a `_headers` file in your repository root. If you need to use a different location, you can specify it by navigating to **Environment > Config > Headers > File Location** in your Stormkit dashboard.

To enable custom headers for your served files:

1. Create a `_headers` file in your repository root (or specify a custom location in your Stormkit dashboard)
2. Define your header rules using the format described below

Check out our [YouTube](https://www.youtube.com/watch?v=0-JE_MoXP68) video to see it in action.

**Note: Custom headers are not applied to responses from serverless functions.**

Header rules are structured in multi-line blocks. Each block begins with a URL or URL pattern that specifies where the rule's headers should take effect. Following this, header names and their corresponding values are listed on indented lines.

```yaml
# Apply X-Message to all requests
/*
  X-Message: Hello World!

# Apply following headers to .js files only
/*.js
  Access-Control-Allow-Origin: *
  Access-Control-Allow-Headers: *
  Access-Control-Allow-Methods: *
```

You can review the deployment manifest to understand how Stormkit builds your code. This allows you to easily see which headers are applied to which files.

<div class="img-wrapper">
    <img src="/assets/blog/manifest.gif" alt="Deployment manifest showing header details" />
</div>

## Environment level custom headers

You can specify the same rules at an environment level. To do so:

1. Go to **Environment Config** > **Headers**
1. Switch **Overwrite headers**
1. Specify the rules from the Headers Editor and
1. Click save.

These rules will be applied to **all** of your deployments and take effect instantly. There is no need for a deployment.
