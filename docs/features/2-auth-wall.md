---
title: Protect your deployments with Auth Wall
description: Learn how to configure and use the Auth Wall feature to restrict access to your deployments, manage team-wide logins with per-app access, and pass login details to your app.
---

# Auth Wall

## Overview

The **Auth Wall** feature allows you to restrict access to your deployments by requiring a login. This is particularly useful for protecting sensitive environments, such as staging or preview deployments, ensuring that only authorized people can access them.

Auth Wall is managed per team: open your team and select **Auth Wall** in the top menu. Personal teams have the page too. Each environment's header shows whether its wall is on, and links to this page filtered to the app.

> **Auth Wall vs. Stormkit Auth**: The Auth Wall gates access to a deployment for *your* team and the people you share it with (staging, previews, sensitive environments). To add sign-in for *your application's own users* — email and password, magic links, or Google / X OAuth — see [Authentication](/docs/features/authentication) instead.

## Environments

The **Environments** tab lists the team's apps, their environments and the mode of each environment's wall:

- **Off**: no login is required.
- **Deployment previews only**: Stormkit-provided URLs require a login. Custom domains stay public.
- **All URLs, including domains**: every page and endpoint, on every domain, requires a login.

To change the mode, select one or more environments (click a row or its checkbox), pick a mode in the bar above the table and click **Apply**. The change applies immediately.

Every member of the team can change the mode of an environment.

## Logins

The **Logins** tab lists the people who can pass the walls of the team's apps. Only team owners and admins can see and manage it; other members see only the **Environments** tab.

A login has:

- **Email**: an email can be used once per team.
- **Password** and/or **magic link**: how the visitor signs in. A login needs at least one. A login that allows magic links doesn't need a password. See [Magic links](#magic-links).
- **Apps**: the apps the login opens, or **All apps**, which includes apps created later. A login opens every environment of its apps where the wall is on.
- **Metadata** (optional): a string of up to 1 KB, passed to your app after the visitor logs in. See [Using the login in your app](#using-the-login-in-your-app).

### Adding a login

Click the **New login** button (the person icon), enter the email and password, and choose the apps.

### Editing a login

Click the pencil icon next to a login to change its password, metadata or apps. Leave the password empty to keep the current one.

### Removing logins

1. Select the logins by checking the box next to their email.
2. Click **Remove selected**.

A removed login can't log in again. Visitors who are already logged in keep access until their session expires, at most 24 hours later. The same applies when you take away a login's access to an app.

## Magic links

A login that allows magic links can sign in without a password. On the login page, the visitor enters their email and clicks **Email me a sign-in link**. Stormkit emails a link that works once, within 15 minutes. Opening it shows a **Continue** button, and continuing signs the visitor in for up to 24 hours, the same as a password login.

- To allow magic links, check **Allow signing in with a link sent by email** when adding or editing a login. To make the login link-only, leave the password empty when adding it, or check **Remove the password** when editing it.
- The page always says a link was sent, so it doesn't reveal which emails have a login. Only logins that allow magic links and open the app get one.
- A link stops working once it's used, once it expires, or when the login is removed, stops allowing magic links or loses access to the app.
- Links are sent with the instance's system email. On self-hosted installations, configure SMTP or the instance's default mailer (**Admin** > **Mailer**). Until one is configured, the login page doesn't offer magic links.

The **Continue** step is there because some email providers open links to scan them before you do. Without it, the scan would use the link up.

## Using the login in your app

After a visitor logs in, Stormkit adds two headers to the requests it forwards to your app's server side (serverless functions, server-side rendering and proxied backends):

- `X-Auth-Wall-Email`: the email of the login.
- `X-Auth-Wall-Metadata`: the login's metadata, when it has any.

Stormkit removes these headers from incoming requests, so a visitor can't set them. Static files can't read request headers.

The metadata is captured when the visitor logs in: a change reaches your app on the visitor's next login. Admins can see it, so don't put secrets in it.

## Periodic triggers

[Periodic triggers](/docs/features/periodic-triggers) pass the wall of their own environment without a login. When a trigger calls one of the environment's Stormkit URLs or verified custom domains over HTTPS, Stormkit signs the request, and the wall lets it through. Signed requests don't follow redirects, so point the trigger at the final URL.

Your app receives the trigger's ID in the `X-Auth-Wall-Trigger` header. As with the login headers, Stormkit removes it from incoming requests, so a visitor can't set it.

Triggers that call any other host, another environment of the same app, or a plain HTTP URL are not signed and still hit the wall.

## Best Practices

- **Limit access**: Give each login only the apps it needs, and use **All apps** sparingly.
- **Regularly review logins**: Periodically review the list of logins and remove those who no longer need access.
- **Use for sensitive environments**: Enable the Auth Wall for environments that contain sensitive data or are not ready for public access.
- **Combine with other security measures**: The Auth Wall is a layer of protection but should not be used as the primary layer to secure your application.

## Troubleshooting

- **A visitor cannot access the deployment**:
  - Check that the login has access to the app, or to all apps.
  - Verify that the visitor is logging in with the correct credentials.
- **A periodic trigger gets the login page**:
  - Check that the trigger's URL starts with `https://` and is a Stormkit URL or a verified custom domain of the trigger's own environment.
- **The deployment is still public**:
  - Check the environment's mode. With **Deployment previews only**, custom domains remain public.
- **A removed visitor can still access the deployment**:
  - Their session is still valid. It expires at most 24 hours after they logged in.

## Additional Notes

- A visitor remains logged in for at most 24 hours.
- Moving an app to another team moves the logins that open it.
