---
title: Troubleshooting
description: Help on common issues with Stormkit.
---

<h1>Troubleshooting</h1>

- [Top-level /index.html missing](#top-level-indexhtml-missing)
- [Repository is inaccessible](#repository-is-inaccessible)
- [Cannot find my app](#cannot-find-my-app)
- [Branch does not exist](#branch-does-not-exist)

<div id="index-html-missing">

### Top-level /index.html missing

</div>

<section>

This warning occurs when Stormkit is not able to find a top-level `/index.html` file and server side rendering is not detected. This does not necessarily mean that your deployment failed. It just means that your endpoint's root path returns 404. Other uploaded files will still return correctly.

To view your deployment package, find your deployment from <b>Deployments</b> list, expand the menu and click
on the <b>Manifest</b> button. The <b>CDN Files</b> section contains a list of deployed files.

If the deployed files are not the correct ones, you can change the output folder by modifying the environment configuration's <b>Output</b> option and re-deploy.

If you need a dynamic application you will need to enable Server Side Rendering. For Nuxt.js, we detect server side rendering automatically. For Next.js, we only support static applications. To enable server side rendering for other applications, your build must generate a `.stormkit/server` folder with a file called <b>server.js</b>. The file must export a method called `handler` like below:

```js
// .stormkit/server/server.js

export const handler = ({ req, res }) => {
  res.write('hello world')
  res.end()
}
```

<p class="mt-4">
You'll probably need a bundler like Vite to bundle your application to <b>.stormkit/server/server.js</b>.

When server side rendering is enabled, all requests not matching a CDN file will be forwarded to serverless
handler. When it is not enabled, Stormkit expects to find a file for the endpoint and returns 404 otherwise.

</p>

</section>

<div id="repository-is-inaccessible">

### Repository is inaccessible

</div>

<section id="repository-is-inaccessible">

This error occurs when Stormkit has no access to the repository. There may be two reasons for this:

**The repository URL is changed**

1. Visit your application
2. Navigate to **Settings** page
3. Change the Repository URL accordingly by pasting the `https` URL of the repository
4. If the problem persists, see next step.

**Credentials are either expired or invalid**

<div class="section-sub-header">

**GitHub**

</div>

If you're using a GitHub repository, you'll need to grant access to the Stormkit App. To do so:

1. Visit https://app.stormkit.io
2. Click on **Create new app**
3. Click on **Connect more repositories**
4. Grant access to the repository
5. Once the popup is closed, navigate to your application and try to re-deploy.

<div class="section-sub-header">

**GitLab**

</div>

1. Visit gitlab.com and login to your account
2. Click on your **Avatar** > **Preferences** > **Applications**
3. Revoke anything related to Stormkit
4. Visit https://app.stormkit.io
5. Make sure to login with GitLab - it will ask you to re-authorize
6. Go back to your application and try to re-deploy

</section>

<div id="cannot-find-my-app">

### Cannot find my app

</div>

<section>

If you logged in with different providers and they are registered through different emails, Stormkit will
create an account for each provider with different email. If you need to migrate your apps between accounts,
reach us out from Discord or email.

</section>

<div id="branch-does-not-exist">

### Branch does not exist

</div>

<section>

If you see a "Stormkit has no access to the repo or the branch does not exist." error, it may be because Storkmit cannot checkout the branch. The branch may be deleted or never existed before. If you are sure that
the branch is existing and you still receive this error, try to follow the steps described in [Repository is inaccessible](#repository-is-inaccessible).

</section>
