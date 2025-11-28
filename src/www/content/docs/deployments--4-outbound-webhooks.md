---
title: Outbound Webhooks
description: Easily automate actions with outbound webhooks, enabling triggers for events like cache disabling and workflow initiation. Boost deployment flexibility and streamline notifications with this powerful feature.
---

# Outbound Webhooks

<section>

With outbound webhooks, you can conveniently trigger various actions when an event occurs, such as disabling cache or initiating workflows like notifications. This powerful feature enhances the flexibility and automation of your deployments.

To enable users to define outbound webhooks upon deployment completion, follow these simple steps:

- Go to **App** > **Settings** page
- Scroll down to **Outbound webhooks** section
- Click on **Add new webhook** to open the modal form
- Fill out the required information in the form and proceed by clicking the **Create outbound** webhook button

</section>

## Events

<section>

1.  After each successful deployment

    The webhook will be triggered after every successful deployment of
    your application. A successful deployment means that your code has
    been built and deployed without any errors. This event cycle
    occurs before status checks are run.

2.  After each failed deployment

    The webhook will be triggered after every failed deployment. This
    event cycle occurs before status checks are run.

3.  After a deployment is published

    The webhook will be triggered after a deployment is published.
    This event cycle occurs after status checks are run.

4.  After a cache purge
    Cache is purged after the following events:

    - A deployment is published
    - Any operation to snippets
    - The environment configuration is updated

</section>

## Special variables for payload

<section>

When the `Request method` is selected as `Post`, you can specify the request payload. This field accepts a raw string that will be send to the request URL. You can use the following variables to customize the request payload.

| Variable                       | Description                                                                                                                                                                                                                              |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$SK_NOW`                      | An ISO 8601 formatted time string. The time is computed when sending the webhook.                                                                                                                                                        |
| `$SK_NOW_UNIX`                 | The unix timestamp. The time is computed when sending the webhook.                                                                                                                                                                       |
| `$SK_APP_ID`                   | The application id.                                                                                                                                                                                                                      |
| `$SK_ENVIRONMENT`              | The environment name. When triggering an outbound webhook on publish, this is the name of the environment that the deployment is published to. Otherwise, it is the name of the environment which the build configuration is taken from. |
| `$SK_DEPLOYMENT_ID`            | The deployment id.                                                                                                                                                                                                                       |
| `$SK_DEPLOYMENT_ENDPOINT`      | The endpoint to preview the deployment.                                                                                                                                                                                                  |
| `$SK_DEPLOYMENT_LOGS_ENDPOINT` | The endpoint to preview the deployment logs. You will need to be authenticated to visit this URL.                                                                                                                                        |
| `$SK_DEPLOYMENT_STATUS`        | A string indicating the deployment status. It's either success or failed.                                                                                                                                                                |

</section>

## Examples

### Receive an Email Notification when a deployment fails

Requirements:

- Create an Environment-level API key (**Environment** > **Config** > **API Keys**)
- [Configure mailer](/docs/features/mailer) for your environment

Steps:

- Go to **App** > **Settings**
- Scroll down to **Outbound webhooks**
- Click on **Add new webhook**
- Fill the form like below and click **Create outbound webhook**

|                |                                   |                                                         |
| -------------- | --------------------------------- | ------------------------------------------------------- |
| Request URL    | `https://api.stormkit.io/v1/mail` | Replace `stormkit.io` with your instance if self-hosted |
| Enable headers | On                                |                                                         |
| Header 1       | `Content-Type`                    | `application/json`                                      |
| Header 2       | `Authorization`                   | `<your-api-key>`                                        |
| Request Method | `POST`                            |                                                         |
| Payload        | See example below                 |                                                         |
| Trigger When   | After each failed deployment      |                                                         |

```json
{
  "to": "joe@example.org",
  "from": "Jane <jane@example.org>",
  "subject": "Deployment failed",
  "body": "Hi, <br /> Deployment failed: $SK_DEPLOYMENT_ENDPOINT"
}
```
