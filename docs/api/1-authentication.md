---
title: Authentication
description: Documentation on accessing Stormkit API.
---

# API Authentication

You can access Stormkit API by using API Keys. Currently, there are three-level API keys:

## User Level API Key

1. Click on your profile picture on the top-right corner
1. Select **Account**
1. Scroll down to the **API Keys** section
1. Create a new API Key

This API Key will grant programmatic access to everything in your Stormkit account.

## Team Level API Key

1. Expand the Team toggle on top-left corner of the page
1. Select the `gear` icon of the Team you would like to access to
1. Create a new API Key

This API Key will grant access to all applications owned by the team.

## Environment Level API Key

1. Visit **Your App** > **Your Environment** > **Config** > **Other** > **API Keys**
1. Create a new API Key

This API Key will grant access to the specified environment.

> **Important:** The API key token is displayed **only once** immediately after creation. Make sure to copy it before closing the dialog — it cannot be retrieved afterwards. If you lose the key, delete it and create a new one.

## Authenticating

Once the API Key is obtained, add an `Authorization` header and use the API key. For example:

```bash
# Using the User Level API Key:

curl -X GET \
     -H 'Authorization: Bearer <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/snippets?envId=4151'
```

```bash
# Using the Team Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/apps'
```

```bash
# Using the Environment Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/json' \
     'https://api.stormkit.io/v1/redirects?appId=48961&envId=58181'
```
