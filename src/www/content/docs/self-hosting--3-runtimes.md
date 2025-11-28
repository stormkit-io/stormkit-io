---
title: Runtime Management
description: Learn how to manage programming language runtimes, package managers, and development tools in your self-hosted Stormkit instance using the Admin Dashboard and mise runtime manager.
---

# Runtime Management

This document explains how to manage runtimes in your **self-hosted Stormkit instance** using the **Admin Dashboard**.

## Overview

Stormkit’s runtime management system allows you to control which programming languages, package managers, and tools are available during your app deployments.

You can:

- Install and manage multiple runtimes (Node.js, Go, npm, Angular CLI, etc.)
- Specify exact versions or use `latest`
- Enable or disable automatic runtime installation
- Upgrade the underlying runtime manager (**mise**)

## Accessing the Runtime Management Page

<div class="blog-alert">

Note: You have to be an administrator to access this area.

</div>

1. Log into your **Stormkit Dashboard**.
2. Click on your **Profile** > **Admin**
3. Navigate to:
   **System** > **Installed runtimes**

## Managing Installed Runtimes

### Adding a Runtime

1. Click **Add Row**.
2. Enter:
   - **Runtime name** — e.g., `node`, `go`, `npm`, `npm:@angular/cli`
   - **Runtime version** — Specific version (e.g., `24`, `1.24`) or `latest`
3. Click **Save**.

<div class="blog-tip">

**Tip:** Refer to the [mise documentation](https://mise.jdx.dev/) for a complete list of supported tools.

</div>

### Removing a Runtime

- Click the **`×`** icon next to the runtime you want to remove.
- Click **Save** to apply changes.

### Auto Install

When **Auto install** is enabled, Stormkit automatically installs required runtimes during deployment based on your app’s version configuration files.

- **Enabled**: Runtimes will be installed automatically if missing.
- **Disabled**: Only pre-installed runtimes will be available.

To toggle:

1. Use the switch under **Auto install**.
2. Save your changes.

The following files are recognized automatically:

| Runtime | Files                                 |
| ------- | ------------------------------------- |
| go      | `.go-version`                         |
| node    | `.nvmrc`, `.node-version`             |
| python  | `.python-version`, `.python-versions` |
| ruby    | `.ruby-version`, `Gemfile`            |

## Mise Runtime Manager

Stormkit relies on the **[mise](https://mise.jdx.dev/)** open-source tool for runtime management. **Current version** is displayed in the **Mise** section.

### Upgrading Mise

1. Click **Upgrade to latest**.
2. Stormkit will fetch and install the newest stable release of `mise`.

> **Note:** Upgrading `mise` does not automatically upgrade installed runtimes. You’ll need to update those manually.

## Best Practices

- **Pin versions** for production apps to ensure predictable builds.
- Keep **mise** updated for the latest runtime management features.
- Use `latest` only for development or experimental environments.
- Regularly review installed runtimes and remove unused ones.

## Related Documentation

- [Mise Runtime Manager](https://mise.jdx.dev/)
