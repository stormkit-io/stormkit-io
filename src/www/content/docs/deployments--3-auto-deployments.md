---
title: Auto Deployments
description: Deploy your code automatically when its merged.
---

# Auto deployments

<section>

Stormkit can automatically deploy your applications on each commit or on each pull request. By default, every application will come with auto deployments **not enabled**. You can enable this option from the application's settings page.

1. Visit **Application** > **Environment** > **Config**
1. Click on `Auto Deploy` dropdown and select an option

</section>

<section>

## Disabled

In this case, Stormkit will not automatically deploy any changes on your repository.

## All branches

When this option is selected, Stormkit deploys every branch that receives a commit automatically.

## Custom branches

You can specify a regexp/glob pattern to filter which branches to automatically deploy.

## Custom commits

Commit patterns allow you to selectively deploy commits that match specific patterns. This feature works in conjunction with your environment's branch configuration.

1. First, Stormkit checks if the commit's branch matches the environment's configured branch
1. Then, if the branch matches, it evaluates the commit against your specified pattern
1. Only commits that match both criteria will trigger a deployment

## Pattern Types

You can use either:

- Regular expressions (regexp)
- Glob patterns

```bash
# Deploy only commits that start with "release/"
release\/*

# Deploy commits that contain "hotfix"
.*hotfix.*

# Deploy commits matching semantic versioning
v[0-9]+\.[0-9]+\.[0-9]+
```

</section>
