---
title: 'Strapi Hosting: How to Self-Host Strapi CMS on Your Own Server'
description: Where to host Strapi, what it needs to run, and a step-by-step guide to deploying a self-hosted Strapi CMS on your own infrastructure with persistent uploads and a database that survives redeploys.
date: 2025-05-01
category: deployment guides
---

Strapi is self-hosted by design: there is no free tier you can push to and forget.
You bring a server, and you decide where it lives, what it costs, and who can
reach the data. This guide covers both halves of that — what Strapi actually
needs from a host, and then a complete walkthrough of deploying it on your own
infrastructure with Stormkit.

<div class="img-wrapper">

![Strapi Login](/assets/tutorials/how-to-deploy-your-self-hosted-strapi-instance/strapi-login.png)

</div>

## Your Strapi hosting options

Strapi is a Node.js application with a database and an uploads directory. That
rules out static hosts and most frontend platforms, and leaves three routes:

**Strapi Cloud** — the managed service from Strapi's own team. Least work,
priced per project, and your content sits on their infrastructure.

**A plain VPS** — Hetzner, DigitalOcean, Scaleway, or a machine you already own.
Cheapest in euros, most expensive in time: you are responsible for Node
versions, process supervision, TLS certificates, backups and redeploys.

**A self-hosted platform on your own server** — the middle path, and what this
guide covers. You still own the machine and the data, but deployments, TLS,
environment variables and persistent storage are handled for you. Stormkit's
self-hosted edition does this; so do other open-source platforms.

There is no meaningful "free Strapi hosting". Strapi holds a database and serves
an admin panel, so it needs a machine that stays running. The honest floor is a
small VPS, which is usually a few euros a month.

## What Strapi needs from a host

Check [Strapi's deployment documentation](https://docs.strapi.io/cms/deployment)
for the authoritative requirements. In practice, plan for:

| Resource | Recommended |
| --- | --- |
| RAM | 4 GB |
| CPU | 2 vCPUs |
| Storage | 20 GB |

The admin panel is rebuilt on deploy, which is the memory-hungry part — a 1 GB
instance will typically survive serving traffic but fall over during a build.

Two things matter more than the specs, because they are what break a Strapi
deployment that looked fine on day one:

**The database has to outlive the deployment.** Strapi defaults to SQLite at
`.tmp/data.db`. If that file is missing at build time, Strapi regenerates it,
and every piece of content is gone. The file has to live on a volume that
survives redeploys.

**Uploads have to outlive the deployment too.** Strapi writes to
`public/uploads`, a path inside the project directory. Without intervention,
every deploy ships a fresh empty folder and your media library breaks.

Both are solved below.

### A note on data residency

Because Strapi is self-hosted, the answer to "can I host Strapi in the UK, or
Germany, or anywhere specific?" is simply: pick a server there. This is one of
the few genuine advantages of self-hosting a CMS over a managed one — the
hosting region is your decision rather than a plan feature, which matters if you
have GDPR or contractual obligations about where content is stored.

## Prerequisites

- A GitHub account
- A [self-hosted Stormkit instance](/tutorials/how-to-self-host-stormkit-on-hetzner-cloud) for deployment
- Basic knowledge of Git and terminal commands

<div class="blog-alert">

Strapi is a long-running Node.js server, so it needs the Application Runtime.
That means a self-hosted Stormkit instance — this particular walkthrough does not
apply to Stormkit Cloud.

</div>

## 1. Create a Strapi Project

Start by creating a new Strapi project locally using the following command:

```
npx create-strapi@latest strapi
```

This command initializes a new Strapi project in a folder named strapi.

Follow the prompts to configure your project. For small-scale projects, using SQLite is fine, whereas for distributed systems PostgreSQL might be more appropriate. In this tutorial, we're going use an SQLite database.

Once complete, navigate to the project folder (cd strapi) and run npm run develop to verify the setup.

## 2. Modify the Code

<div class="blog-alert">

While SQLite works perfectly for small scale applications, we suggest using a PostgreSQL database for larger scale projects.

</div>

By default, Strapi automatically sets up an SQLite database at `.tmp/data.db`. If the database is missing during a project build, Strapi will regenerate it which will cause the data to be wiped out when using SQLite as the data source. To mitigate this, we're going make use of the Stormkit's [Persistent Volumes](https://www.stormkit.io/docs/features/volumes) feature to upload the database and tell Strapi where to locate it.

1. Go ahead and locate the `config/database.ts` file
2. Apply the following patch:

```diff
diff --git a/config/database.ts b/config/database.ts
index 1853ca4..ca8aeab 100644
--- a/config/database.ts
+++ b/config/database.ts
@@ -44,7 +44,7 @@ export default ({ env }) => {
     },
     sqlite: {
       connection: {
-        filename: path.join(__dirname, '..', '..', env('DATABASE_FILENAME', '.tmp/data.db')),
+        filename: env('DATABASE_FILENAME', path.join(__dirname, '..', '..', '.tmp/data.db')),
       },
       useNullAsDefault: true,
     },
```

This change will allow us using an absolute path for the `DATABASE_FILENAME` environment variable. Make sure to commit your changes:

```bash
git add .
git commit -m "chore: allow absolute path for sqlite databases"
```

Next, open your `package.json` and add the following script:

```js
{
  "scripts": {
    // ... other scripts
    "start:stormkit": "rm -rf public/uploads && ln -s $VOLUME_PATH public/uploads && strapi start"
   }
}
```

And commit the changes:

```bash
git add .
git commit -m "chore: prepare the stormkit script"
```

This is needed to overcome Strapi's hard-coded upload path, which is relative to the project level. If we don't provide this script the uploaded files on Strapi won't be persisted.

## 3. Push to your GitHub repository

Once you committed the changes, push your changes to your repository.

```bash
# Make sure that the remote address exists
git remote add origin git@github.com:<your-repository>
git push -u origin HEAD
```

## 4. Deploy on Stormkit

### Importing the project

- Log in to your Stormkit instance.
- Click to `Create new app` > `Import from GitHub`
- Choose your Strapi project and click `Import`

### Configuring Volumes

If you haven't configured the volumes, follow these steps:

- Click on the `Volumes` tab in your environment page
- Click on `Configure`.
- Make sure `Volume type` is `File system`
- Specify the root folder as `/shared/volumes`
- Click on `Save`

This will tell Stormkit to upload persistent files under the `/shared/volumes` folder. Each environment has it's own folder. You can guess the folder name from the application and environment IDs, which is easily collectible from the URL. The folder structure uses the following format:

```
<volumes-base-path/a<app-id>e<env-id>`
```

For instance, if your URL looks like: `/apps/5/environments/6` and your base path is `/shared/volumes`, the environment folder is located at `/shared/volumes/a5e6`. Note this path somewhere.

### Configure your deployment

- Click on the `Config` tab and locate `Server settings`
- Provide the command that will start the server: `npm run start:stormkit`
- Click on `Save`.

Next, we need to setup the environment variables.

- Scroll down to the `Environment variables` section
- Click on `Modify as String`
- Copy the environment variables located inside the `.env` file in your Strapi project
- Make sure `DATABASE_CLIENT` is `sqlite` and `DATABASE_FILENAME` points to your volumes folder that you noted earlier
- Add an additional environment variable `VOLUMES_PATH` which points to the volumes folder

<div class="img-wrapper"> 
  
  ![Stormkit-Deployment-Config](/assets/tutorials/how-to-deploy-your-self-hosted-strapi-instance/stormkit-deployment-configuration.png)
  
</div>

## 5. Verify the Deployment

Once you went through all the steps mentioned above, go ahead and `Deploy` your Strapi application. When the deployment is complete click on the `Preview` button to access your Strapi instance.

Redeploy once more before you trust it. The first deployment proves the build
works; the second proves your content and uploads survived it, which is the part
that actually distinguishes a working Strapi host from a broken one.

## Frequently asked questions

**Can I host Strapi for free?**
Not realistically. Strapi runs a persistent Node.js process and a database, so
it needs a machine that stays up. Free tiers that sleep on idle will lose SQLite
data and make the admin panel unusable. Budget for a small VPS.

**How much does it cost to host Strapi yourself?**
The server is the cost. A VPS meeting the 4 GB / 2 vCPU recommendation is
typically in the €5–15 per month range, and one server can host several projects
alongside Strapi.

**SQLite or PostgreSQL?**
SQLite is fine for a single instance with modest content, and it is what this
guide uses. Move to PostgreSQL when you need more than one instance, want
managed backups, or your content outgrows a single file.

**Why does my Strapi content disappear after a deploy?**
Almost always the two paths above: SQLite regenerated at `.tmp/data.db` because
the old file was not found, or `public/uploads` shipped empty with the new
build. Both need to point at persistent storage.

**Can I move from Strapi Cloud to self-hosted?**
Yes. Strapi is the same software in both places — export your data, point the
new instance at it, and repoint your DNS.
