---
title: 'Self-Hosting with Stormkit: Authentication Setup'
description: Learn how to configure authentication for your self-hosted Stormkit instance. Set up admin accounts and integrate with GitHub, GitLab, or BitBucket for seamless repository management.
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
- **BitBucket**

## Create an oAuth Application

Visit the developer portal of your chosen provider and create an oAuth application:

- [GitHub Developer Settings](https://github.com/settings/apps)
- [GitLab Applications](https://gitlab.com/-/user_settings/applications)
- [BitBucket App Management](https://bitbucket.org/account/settings/app-passwords/)

### GitHub Authentication

To enable GitHub authentication, you need to create a **GitHub App** (not an oAuth App) and configure the following environment variables:

#### Step 1: Create a GitHub App

1. Go to [GitHub Developer Settings](https://github.com/settings/apps)
2. Click **New GitHub App**
3. Fill in the required fields:
   - **GitHub App name**: Choose a unique name for your app
   - **Homepage URL**: Your Stormkit instance URL (e.g., `https://api.your-stormkit-domain.com`)
   - **Callback URL**: `https://api.your-stormkit-domain.com/auth/github/callback`
   - **Webhook URL**: `https://api.your-stormkit-domain.com/app/webhooks/github/deploy`
   - **Post Installation Setup URL**: `https://api.your-stormkit-domain.com/auth/github/installation`
4. Under **Events**, select following items:
   - **Pull request**
   - **Push**
5. Under **Permissions**, grant the following permissions:
   - **Repository:Administration**: Read and Write
   - **Repository:Checks**: Read and Write
   - **Repository:Commit statuses**: Read and Write
   - **Repository:Contents**: Read
   - **Repository:Pull requests**: Read and Write
   - **Repository:Webhooks**: Read
   - **Account:Email addresses**: Read
6. Click **Create GitHub App**

#### Step 2: Generate Private Key

1. After creating the app, scroll down to **Private keys** section
2. Click **Generate a private key**
3. Download the `.pem` file

#### Step 3: Configure Environment Variables

Add the following variables to your environment configuration. If you used the `install.sh` script, your environment variables are stored in `~/.profile` file.

```bash
# The GitHub App name you chose during creation
GITHUB_APP_NAME=your-app-name

# Found under the General tab of your GitHub App page
GITHUB_APP_ID=123456

# The client secret (generate one if not created automatically)
GITHUB_SECRET=your_client_secret

# Found under the General tab of your GitHub App page
GITHUB_CLIENT_ID=Iv1.abc123def456

# Base64 encoded private key from the .pem file
# Encode using: cat your-private-key.pem | openssl base64 -A
GITHUB_PRIV_KEY="LS0tLS1CRUdJTi..."
```

**Important**: Make sure to enclose the `GITHUB_PRIV_KEY` value with quotes since it contains special characters.

### GitLab Authentication

To enable GitLab authentication, you need to create a **GitLab App** and configure the following environment variables:

#### Step 1: Create a GitLab App

1. Go to [GitLab Developer Settings](https://gitlab.com/-/user_settings/applications)
2. Click **Add new application**
3. Fill in the required fields:
   - **Name**: Choose a unique name for your app
   - **Redirect URI**: `https://api.your-stormkit-domain.com/auth/gitlab/callback`
4. Select following scopes:
   - **read_user**
   - **read_repository**
   - **write_repository**

#### Step 2: Configure Environment Variables

Add the following variables to your environment configuration. If you used the `install.sh` script, your environment variables are stored in `~/.profile` file.

```bash
# The Secret key
GITLAB_SECRET=gloas-abc123...

# Your Application ID
GITLAB_CLIENT_ID=af845...
```

### Restart your Stormkit Instance

For the environment variables to take effect, make sure to restart Stormkit. For an environment using docker compose, the command to restart the services is:

```bash
docker compose down workerserver hosting && docker compose up workerserver hosting -d
```

<div class="blog-alert">

Note: If you are providing environment variables using the `~/.profile` method, make sure to source that file before restarting the server.

```bash
source ~/.profile
```

</div>

</section>
