---
title: "Self-Hosting with Stormkit: Authentication Setup"
description: Learn how to configure authentication for your self-hosted Stormkit instance. Set up admin accounts and integrate with GitHub, GitLab, Bitbucket, or Forgejo for seamless repository management.
---

# Authentication

<section id="authentication">

Authentication is a critical part of your self-hosted Stormkit instance. During the initial setup, you will create an **admin account** that has full access to the admin interface and can configure instance-wide options.

The admin account can:

- Access the admin interface.
- Configure global settings for your Stormkit instance.
- Import public repositories or create bare apps.

However, to import **private repositories**, you must configure at least one of the following authentication methods:

- **GitHub**
- **GitLab**
- **Bitbucket**
- **Forgejo** (self-hosted)

## Configuring Git Providers

Git provider authentication is configured directly from the **Admin Interface** in Stormkit. To access this page:

1. Click on your **profile** in the top right corner
2. Select **Admin** from the dropdown menu
3. Navigate to **Git** (or go directly to `/admin/git`)

### GitHub Authentication

GitHub authentication is the simplest to set up. Stormkit automatically creates a GitHub App for you with all the necessary permissions and configurations.

#### Setup Steps

1. Navigate to `/admin/git` in your Stormkit instance
2. Click **Configure GitHub**
3. Enter a unique **App Name** for your GitHub App
4. Click **Continue on Github**

Stormkit will automatically create the GitHub App with the correct permissions, webhook configuration, webhook secret, and callback URLs. Once created, GitHub authentication will be immediately enabled for your instance.

#### Webhook Secret

Stormkit verifies every webhook GitHub sends against the GitHub App's webhook secret, and rejects webhooks when no secret is configured. Apps created with the steps above store the secret automatically.

If you configured the GitHub App manually, or created it before `v2026.09.28.1`, set the secret yourself:

1. Open your GitHub App's settings on GitHub and, under **Webhook**, click **Change secret**
2. Enter a random value, for instance the output of `openssl rand -hex 32`, and save
3. In Stormkit, click **Configure GitHub** on `/admin/git`, paste the same value into **Webhook secret**, and click **Configure**. Leave **Client secret** and **Private key** empty to keep their current values. On `v2026.09.28.1` the form still asks for both, so re-enter them there

Alternatively, provide the value through the `GITHUB_WEBHOOK_SECRET` environment variable. A secret saved from the admin interface takes precedence.

Until a secret is configured, pushes and pull requests do not trigger deployments and `/admin/git` shows a warning.

### GitLab Authentication

To enable GitLab authentication, you need to create a GitLab Application first.

#### Step 1: Create a GitLab Application

1. Go to [GitLab Developer Settings](https://gitlab.com/-/user_settings/applications)
2. Click **Add new application**
3. Fill in the required fields:
   - **Name**: Choose a unique name for your app
   - **Redirect URI**: This will be provided in the Stormkit configuration modal (pre-configured)
4. Select following scopes:
   - **read_user**
   - **read_repository**
   - **write_repository**
5. Click **Save application**
6. Copy the **Application ID** and **Secret** that are displayed

#### Step 2: Configure in Stormkit

1. Navigate to `/admin/git` in your Stormkit instance
2. Click **GitLab** button
3. The **Redirect URI** will be displayed and pre-configured for you
4. Enter the following information from your GitLab Application:
   - **Client ID**: Your Application ID
   - **Client Secret**: The Secret key
5. Click **Configure** to complete the configuration

GitLab authentication will be immediately enabled for your instance.

### Bitbucket Authentication

To enable Bitbucket authentication, you need to create a Bitbucket OAuth Consumer first.

#### Step 1: Create a Bitbucket OAuth Consumer

1. Go to your Bitbucket workspace settings
2. Navigate to **OAuth consumers**
3. Click **Add consumer**
4. Fill in the required fields:
   - **Name**: Choose a unique name for your OAuth consumer
   - **Callback URL**: This will be provided in the Stormkit configuration modal (pre-configured)
5. Grant the necessary permissions for repository access
6. Click **Save**
7. Copy the **Key** (Client ID) and **Secret** that are displayed

#### Step 2: Configure in Stormkit

1. Navigate to `/admin/git` in your Stormkit instance
2. Click **Bitbucket** button
3. Enter the following information from your Bitbucket OAuth Consumer:
   - **Client ID**: Your OAuth consumer Key
   - **Client Secret**: The Secret key
   - **Deploy Key** (optional): If you want to use a specific deploy key for repository access
4. Click **Configure** to complete the configuration

Bitbucket authentication will be immediately enabled for your instance.

When a deploy key is configured, Stormkit does not register webhooks on your repositories. Add them yourself using the webhook URL shown in each app's settings under **Deploy triggers**. The URL carries a secret that identifies the app, and webhooks without it are rejected.

### Forgejo Authentication

Stormkit can connect to one self-hosted [Forgejo](https://forgejo.org) instance. Gitea exposes the same API, so a Gitea instance can be connected the same way.

#### Step 1: Create a Forgejo OAuth2 Application

1. In Forgejo, open **Settings** → **Applications** (or **Site Administration** → **Applications** for an instance-wide application)
2. Under **Manage OAuth2 applications**, fill in:
   - **Application name**: Choose a unique name, for instance `Stormkit`
   - **Redirect URIs**: `https://<your-stormkit-api>/auth/forgejo/callback`, or the **Redirect URL** you set in Stormkit
   - Keep **Confidential client** checked
3. Click **Create application**
4. Copy the **Client ID** and **Client secret** that are displayed

Forgejo does not support OAuth2 scopes yet, so the token Stormkit receives carries the full rights of the user who signs in. Stormkit keeps it on the server: it is never sent to the browser or to builds.

#### Step 2: Configure in Stormkit

1. Navigate to `/admin/git` in your Stormkit instance
2. Click **Configure Forgejo**
3. Enter the following information:
   - **Forgejo URL**: The address of your instance, for instance `https://git.example.org`
   - **Client ID**: Your application's Client ID
   - **Client Secret**: Your application's Client secret. Leave it empty to keep the stored one
4. Click **Configure** to complete the configuration

Alternatively, provide the values through the `FORGEJO_URL`, `FORGEJO_CLIENT_ID`, `FORGEJO_SECRET` and, optionally, `FORGEJO_REDIRECT_URL` environment variables. Environment variables are only read until a git provider is configured from `/admin/git`.

Stormkit registers a webhook and a read-only deploy key on each repository the first time it deploys it. Builds clone over SSH with that deploy key, so your Forgejo instance must have SSH enabled and its SSH port must be reachable from the machines that run builds. Pushes to the default branch and pull requests trigger deployments, and preview links are posted as pull request comments. The Forgejo instance must be able to reach your Stormkit API to deliver webhooks. If it rejects the delivery because the API resolves to a private address, add your Stormkit host to `[webhook] ALLOWED_HOST_LIST` in Forgejo's `app.ini`.

</section>
