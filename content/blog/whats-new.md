---
title: What's New?
description: Discover the latest changes and improvements to Stormkit. Stay up-to-date and get the most out of our platform.
---

Follow the latest developments on Stormkit.

## September 6th, 2025

- New event cycle for outbound webhooks: On Cache Purge
- Environment level custom headers can be set from the Build Configuration page

## July 18th, 2025

You can now sort runtime logs ascending or descending.

## May 20th, 2025

- The feature flags have been deprecated and removed from Stormkit. You can make use of Snippets
  instead.
- We lowered the prices for Self-Hosted seats. The base subscription is now $20.00 per month / per user instead of $39.00.

## May 8th, 2025

Stormkit now supports deploying applications directly from [zip files](/docs/deployments/zip-file-deployments), simplifying the deployment process.

## Apr 7th, 2025

Introducing the **Auth Wall** feature: Restrict access to your deployments with user authentication, protecting previews or all endpoints, including custom domains. Manage authorized users easily from the dashboard. Learn more in the [Auth Wall documentation](/docs/features/auth-wall).

## Mar 6th, 2025

Stormkit now offers framework-agnostic image optimization with on-the-fly resizing and smart cropping through simple URL parameters, enabling improved page load times and allowing up to 5 cached variants per image.

## Feb 12th, 2025

New public API endpoints to [manage environments](/docs/api/environments).

## Dec 26th, 2024

Added `commit` patterns to [automatic deployments](/docs/deployments/auto-deployments).

## Dec 24th, 2024

- Improved periodic triggers UI
- Introduced trigger logs

## Nov 4th, 2024

- Introducing [Persistent Volumes](/docs/features/volumes). Upload, manage and share files with ease.
- Introducing [Mailer](/docs/features/mailer). Send transactional emails directly from your application.

## Oct 23rd, 2024

With this new version:

- The last selected team will be displayed in the home page.
- Action buttons in the Team Settings page are now hidden for `Developer` roles.
- On self-hosted environments you can set a custom page title with the `STORMKIT_PAGE_TITLE` environment variable.
  This environment variable needs to be set in the Frontend application.

## Sep 12th, 2024

Introducing [Status Checks](/docs/deployments/status-checks): Automatically verify post-deployment conditions and decide whether to publish or withhold deployments based on customizable checks.

## Aug 30th, 2024

Introduced dark / light mode switcher. Click **Avatar** > **Theme** to switch.

## Aug 20th, 2024

You can now specify a [custom error file](/docs/features/redirects-and-path-rewrites) for 404 pages.

## Aug 16th, 2024

Introduced configuring [custom certificates](/docs/features/custom-certificates) for domains hosted on Stormkit.

## May 16th, 2024

You can specify environment level redirect rules now. These rules will be applied to **all** of your deployments and take effect instantly. There is no need for a deployment.

## May 14th, 2024

New configuration option for redirect rules: `hosts`. Read more about it [here](/docs/features/redirects-and-path-rewrites).

## May 6th, 2024

Introduced Redirects Playground feature. Visit your Environment's configuration page > Redirects section to open the playground and test redirects.

## March 3th, 2024

Inject snippets depending on request path

<iframe width="560" height="315" src="https://www.youtube.com/embed/A8Dzpes552w?si=KOwSXX9puPQ1LtHl" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## March 1th, 2024

Stormkit enables you to store and retrieve data via Redis. Checkout the documentation for further details

<iframe width="560" height="315" src="https://www.youtube.com/embed/yaCQTkm5Fb8?si=AifViRzYngLfcbhW" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## February 14th, 2024

You can configure multiple domains for same project and apply snippets depending on domain name

<iframe width="560" height="315" src="https://www.youtube.com/embed/3SvbLcDWfxA?si=GCIMosObQdDADlwm" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>

## January 28th, 2024

We made some improvements to the product:

- Environments config is now a unified page.
- When a team is selected, the team menu is automatically closed.
- You can preview all of your teams deployments by clicking on the **User Menu** > **Team Deployments**.

## January 27th, 2024

[Custom headers](/docs/features/custom-headers) enable you to modify or supplement the default HTTP headers provided by Stormkit when a client requests your site.

## December 7th, 2023

Introducing [Web Analytics](/docs/features/analytics). Simple, privacy-friendly, automatic setup and powerful.

## November 28th, 2023

Announcing advanced [Teams](/docs/features/teams-and-roles) for a seamless collaboration.

## September 18th, 2023

We now have a better design for redirects. We also introduced proxies. You can define them from your [redirects.json](/docs/features/redirects-and-path-rewrites) file.

## September 5th, 2023

Environment variables are now available for serverless functions runtime.

## September 4th, 2023

We updated the list of injected system environment variables. Visit our [doc page](/docs/deployments/system-variables) for more information.

## September 3rd, 2023

We added more availability zones to our servers in Europe.

## August 22th, 2023

By integrating our "Deploy" button into the markdown of your repository, you can streamline the process of cloning your GitHub repository and deploying it on Stormkit.

## August 20th, 2023

Our serverless deployments have no longer the 50MB zipped limit. Please note that the 250MB unzipped limit remains.

## August 17th, 2023

We're thrilled to announce a fantastic addition to our platform - a more convenient method for users to incorporate their environment variables. With our latest enhancement, you can now simply copy and paste all your essential environment variables in one go. Seamlessly streamline your setup process!

<div class="img-wrapper">
    <img src="/assets/docs/features/easyenv.png" alt="responsibilities of stormkit and supabase" />
</div>

## May 28th, 2023

You can now import repositories directly with an URL. To do so, go to home page and click on `Create new app` > `Import from URL`.

## May 23rd, 2023

We've redesigned the app's navigation menu for a better user experience. Streamlined layout, clear labels, and visual enhancements make navigation more intuitive. Enjoy faster access to features with our revamped menu!

## May 21st, 2023

- We've upgraded our checkout system to leverage the power of Stripe. With Stripe Checkout, you can enjoy a seamless and secure payment experience, ensuring your transactions are handled with utmost reliability and convenience. Go to [Account](https://app.stormkit.io/user/account) page to handle your billing and subscriptions.

- We updated our pricing. Our new pricing options are carefully crafted to provide flexibility and scalability, ensuring that you have the right plan to meet your evolving needs. Existing users will continue to enjoy their current pricing plans without any changes. Check out our new pricing [here](https://www.stormkit.io/#pricing).

## May 20th, 2023

- We're excited to unveil our newly redesigned landing page, featuring a sleek and modern interface that makes it easier than ever to explore our offerings. Discover our products/services effortlessly with the improved navigation and user-friendly layout.

- Default branch for [www-stormkit-io](https://github.com/stormkit-io/www-stormkit-io) and [app-stormkit-io](https://github.com/stormkit-io/app-stormkit-io) is now `main` instead of `master`.

## March 16th, 2023

We have added support for NodeJS 18 runtime. You can configure that from your Application Settings page.

## December 27th, 2022

Recently we announced support for API endpoints for client-side applications. Now, you can call these endpoints periodically periodically with [Trigger Functions](/docs/features/trigger-functions).

## December 15th, 2022

Announcing feature flags (deprecated and removed): inject them into your client-side application and control them through the UI. Instant updates, no need for deployments.

## October 19th, 2022

You can now preview runtime logs from your server side rendered apps and APIs for each deployment. Expand the
**Deployment Menu** and click on **Runtime logs** to access the page.

<div class="img-wrapper">
    <img src="/assets/blog/whats-new/runtime-logs.gif" alt="Runtime logs" />
</div>

## September 15th, 2022

During the build process, Stormkit generates a manifest about your application to serve files in a more optimal ways. The manifest includes information about your CDN files, API routes, redirects and whether server side
rendering is enabled or not.

<div class="img-wrapper">
    <img src="/assets/blog/whats-new/manifest.png" alt="build manifest">
</div>

## September 13th, 2022

The new menu layout is released. From now on, the menu will always be on the left side on Desktop versions.
In order to change your environment, you can select the environment
dropdown next to the application name on top of the page. To navigate within your environment, hover
on the environments icon on the left menu.

## August 16th, 2022

Stormkit now enables users to set [feature flags](https://martinfowler.com/articles/feature-toggles.html) for their applications. They will be available client side through the `window.sk.features` object.

## August 10th, 2022

Stormkit now automatically detects `pnpm` projects and build them using `pnpm`.

## August 8th, 2022

We now display the commit sha of the deployment. Clicking on the takes you to the diff page to view
the changes.

<div class="img-wrapper">
    <img src="/assets/blog/whats-new/feat-commit-sha.png" alt="Commit SHA" />
</div>

## August 6th, 2022

Stormkit now supports building APIs using Node.js / Typescript 🎉. [Learn more](/docs/features/writing-api).

## July 28th, 2022

Stormkit now injects the git commit sha as an environment variable during build time. This is useful to track
changes with third party tools. For instance by providing the commit sha to Sentry, you can track after which
specific commit the errors started to occur.

<div class="img-wrapper">
    <img src="/assets/blog/whats-new/git-commit.png" alt="Injected Commit SHA" />
</div>

## June 30th, 2022

https://github.com/stormkit-io/serverless is now open source. With serverless you can make your nodejs app run on aws lambda with minimal efford

## June 9th, 2022

It's now possible to use Node.js 16 runtime. To change your Node version, go to your Application Settings
and change the selected Runtime.

## May 18th, 2022

Better support for next.js serverless projects. The bundle size is decreased and a bug has been fixed.

**Important** When deploying next.js serverless applications, do not forget to specify `target: "serverless"`
in `next.config.js` file.

## April 18th, 2022

- The Serverless toggle has been deprecated. Stormkit now understands whether your application is serverless
  or not from the artifacts.

- We have removed the limit on the number of applications you can host on Stormkit based on your tier. It's
  all unlimited now.

## April 17th, 2022

It is now possible to view usage data by visiting **App** > **Usage**.

## April 10th, 2022

Free users can now deploy up to 15 times per month.

## April 9th, 2022

Stormkit users now have more flexibility on what branches to deploy automatically. To use this new
feature, visit your application's page and click on **Edit configuration** on the environment that you would like to modify and find the **Auto Deploy Branches** field to instruct Stormkit for which branches
auto deployments are turned on. The field accepts a `regexp` pattern. Here are some examples:

```regex
^dependabot/.*             # Deploy only branches that start with dependabot/
^(?!dependabot|renovate).* # Deploy all branches that do not start with dependabot or renovate
release-*                  # Deploy all branches that start with release-
```

The logic to determine whether a branch should be automatically deployed or not is depicted
in the following diagram:

<div class="img-wrapper">
    <img src="/assets/blog/whats-new/auto-deployment-flow-transparent.svg" alt="Auto deployment flow" />
</div>

### Other changes

- `Commit prefix` configuration option located in the application's settings page is removed
- Providing `--skip-deploy` in a commit message is no longer considered

## April 1st, 2022

It is now possible to overwrite default environment configuration while
using Hooks API to trigger a deployment.

The Hooks API supports both `GET` and `POST` methods.

```bash
curl -XGET https://api.stormkit.io/hooks/app/:app-id/deploy/:token/:environment-id?publish=true|false&branch=:branch-name
```

```bash
curl -XPOST https://api.stormkit.io/hooks/app/:app-id/deploy/:token/:environment-id \
    -H 'Content-Type: application/json'
    -d '{"publish": true|false, "branch": ":branch-name"}'
```

You can turn on the deploy triggers from your application's settings page.

## March 24th, 2022

Stormkit now supports integrating with external services to reduce vendor-lock (feature deprecated).
