---
title: Authentication
description: Documentation on accessing Stormkit API.
---

# API Authentication

You can access Stormkit API by using API Keys. Currently, there are two-level API keys:

## Team Level API Key

1. Expand the Team toggle on top-left corner of the page
1. Select the `gear` icon of the Team you would like to access to
1. Create a new API Key

This API Key will grant access to all applications owned by the team.

## Environment Level API KEy

1. Visit **Your App** > **Your Environment** > **Config** > **Other** > **API Keys**
1. Create a new API Key

This API Key will grant access to the specified environment.

## Authenticating

Once the API Key is obtained, add an `Authorization` header and use the API key. For example:

```bash
# Using the Environment Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/redirects'
```

```bash
# Using the Team Level API Key:

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/redirects?appId=48961&envId=58181'
```
