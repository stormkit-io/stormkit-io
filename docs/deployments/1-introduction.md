---
title: Deployments
description: Deploy your applications using Stormkit.
---

# Deployments

<section>
<p>
   Stormkit is able to build and deploy your application. The underlying infrastructure service is AWS. Static assets will be hosted on S3 buckets, which will then be served through our CDN. If you have server side logic, then your application will be run on AWS Lambdas.
</p>
<p>
   There are multiple ways to trigger a deployment:
</p>
</section>

## Git Push

<section>

1. Visit your application
1. Go to the **Settings** page
1. Make sure to have the **Auto Deployments** configuration turned on.
1. Submit a commit.

Stormkit will receive a push event and will build, deploy and leave a preview link under the Pull Request page
of your branch. Stormkit picks up the environment configuration based on the branch name. You can read more [here](/docs/deployments/auto-deployments).

</section>

## Through the UI

<section>

1. Visit your application
1. Click on **Deploy Now** button on the top right side
1. Select your **Environment** in the modal

By default, the **Environment** configuration will be used to deploy your application but you can overwrite
that configuration for one-time by specifying different values in the modal.

</section>

## Trigger

<section>

1. Visit your application
1. Go to the **Settings** page
1. Find the **Trigger Deploy** section
1. Click generate

This will create a link that will trigger a deployment. You can either use a `GET` or a `POST` request. Optionally, you can specify a `publish` and `branch` parameter.

Here's an example for the `POST` method:

```bash
curl -XPOST https://api.stormkit.io/hooks/app/:app-id/deploy/:token/:environment-id \
   -H 'Content-Type: application/json'
   -d '{"publish": true|false, "branch": ":branch-name"}'
```

<div>
Make sure not to share this URL publicly.
</div>

</section>
