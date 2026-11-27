---
title: Headless Browsers
description: Use headless Chromium and Playwright in self-hosted Stormkit deployments for testing, scraping, and browser-based workflows.
---

# Headless Browsers

Stormkit can be used to run browser-based workloads in self-hosted instances.

This is especially useful for:

- End-to-end tests with Playwright
- Smoke tests against preview deployments
- Post-deploy checks that verify the application in a real browser before publishing
- Visiting JavaScript-heavy pages and extracting rendered HTML
- Running internal scraping, crawling, or snapshot workflows from your application

## How it works

On self-hosted instances, headless browser support is enabled through the runtime manager.

You can install tools such as:

- `nix:chromium` to provide a headless Chromium binary
- `npm:playwright` to make the Playwright CLI available on the instance

When a tool exposes an executable, Stormkit injects its resolved path as an environment variable using the `MISE_<TOOL>_PATH` format. For Chromium, this becomes `MISE_CHROMIUM_PATH`.

This lets your scripts run a browser without hardcoding system-specific paths.

## Typical setup

1. Go to **Admin** > **System** > **Installed runtimes**
2. Add `nix:chromium`
3. Add `npm:playwright`
4. Save the changes
5. Configure a deployment status check that runs after each successful deployment

With this setup, your deployment can stay unpublished until the browser-based check passes.

## Using it inside your application

Headless browsers are not limited to deployment checks.

You can also use them from your own application code when you need a real browser context to:

- Load pages that depend on client-side JavaScript before reading the final HTML
- Access pages that require browser APIs or complex navigation flows
- Extract content from third-party websites after the page has fully rendered

In these cases, your application can launch Chromium through Playwright and use `process.env.MISE_CHROMIUM_PATH` as the browser `executablePath`.

## Using it with status checks

Status checks are a good fit for browser-based validation because they run after deployment and can decide whether a deploy should be published.

For example, you can run a Playwright-based command that opens the preview deployment, waits for the application to load, and exits with a non-zero status when a critical path fails.

If your script needs the Chromium path explicitly, use `process.env.MISE_CHROMIUM_PATH`.

## Related Documentation

- [Runtime Management](/docs/self-hosting/runtimes)
- [Deployment Status Checks](/docs/deployments/status-checks)
- [System environment variables](/docs/deployments/system-variables)
