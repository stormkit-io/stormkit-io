---
title: "Self-Hosting with Stormkit: Authentication Setup"
description: Learn how to configure authentication for your self-hosted Stormkit instance. Set up admin accounts and integrate with GitHub, GitLab, or Bitbucket for seamless repository management.
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

## Configuring Git Providers

Git provider authentication is configured directly from the **Admin Interface** in Stormkit. To access this page:

1. Click on your **profile** in the top right corner
2. Select **Admin** from the dropdown menu
3. Navigate to **Git** (or go directly to `/admin/git`)

### GitHub Authentication

GitHub authentication is the simplest to set up. Stormkit automatically creates a GitHub App for you with all the necessary permissions and configurations.

#### Setup Steps

1. Navigate to `/admin/git` in your Stormkit instance
2. Click **GitHub** button
3. Enter a unique **App Name** for your GitHub App
4. Click **Create**

Stormkit will automatically create the GitHub App with the correct permissions, webhook configurations, and callback URLs. Once created, GitHub authentication will be immediately enabled for your instance.

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
5. Click **Save** to complete the configuration

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
4. Click **Save** to complete the configuration

Bitbucket authentication will be immediately enabled for your instance.

</section>
