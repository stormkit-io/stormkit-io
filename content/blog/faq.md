---
title: FAQ
description: Our Frequently Asked Questions (FAQ) page is designed to provide you with quick and helpful information about Stormkit.
date: 2023-08-17
---

<details>
<summary>How Stormkit is different than Heroku?</summary>

Both are deployment platforms that take your code and run it. Stormkit hosts static sites, single-page apps, server-side rendered apps, serverless functions and long-running server processes, in any language — runtimes are provisioned with [mise](https://mise.jdx.dev), and system-level packages can come from a `flake.nix`. On top of deployment it ships a [PostgreSQL database](/docs/features/database), [end-user authentication](/docs/features/authentication), a [mailer](/docs/features/mailer), [periodic triggers](/docs/features/periodic-triggers) and [analytics](/docs/features/analytics), so you are not assembling those from separate vendors.

The bigger difference is that Stormkit can be [self-hosted](/docs/self-hosting/getting-started) on your own infrastructure, which is what most people are looking for when they compare the two.

</details>

<details>
<summary>Can I run my Node.js applications on Stormkit?</summary>

Yes. On a [self-hosted instance](/docs/self-hosting/getting-started), set a **Start command** and Stormkit runs your server as a long-lived process — see [Application runtime](/docs/deployments/application-runtime). This is the recommended way to run backend code when self-hosting, and it is not limited to Node: a Go, Python or Ruby server works the same way.

On Stormkit Cloud, backend code runs as [serverless functions](/docs/features/writing-api) — same interface as Node.js request handlers — which are stateless, short-lived and subject to a 15 second timeout. If you need long-lived processes, connection pools or a warm cache, self-host.

</details>

<details>

<summary>Is it possible to establish a database connection using Stormkit?</summary>

Certainly! Stormkit can provision a [PostgreSQL database](/docs/features/database) for your environment and run migrations on deployment. You can also connect to an external database by injecting its credentials as environment variables into your backend functions.

</details>

<details>

<summary>How do I deploy my monorepo using Stormkit?</summary>

If your repository encompasses multiple projects, Stormkit offers two approaches for deployment. Firstly, you can create distinct projects within the same repository and set up automated deployment based on branch naming conventions. For instance, if your repository hosts both frontend and backend applications, establish two projects triggering deployment when branches start with `fe-` and `be-` respectively. Configure these settings in the environment's configuration section.

Alternatively, you can create two separate environments within a single app, one for frontend and the other for backend. Utilizing the [SK_CWD](/docs/deployments/configuration) variable, you can build each project accordingly. As before, deploy triggers can be set up based on branch naming conventions.

</details>

<details>
<summary>Are you using AWS?</summary>

Yes, we leverage specific AWS solutions such as Lambda and S3 to enhance our services. Our approach involves utilizing certain AWS services to minimize dependence on AWS. This strategy ensures our platform's adaptability for potential portability to on-premise environments or alternate cloud providers in the future.

</details>

<details>
<summary>Do you support Next.js?</summary>

As of May 21, 2023, we [have made the decision](/blog/why-we-are-dropping-support-for-next-js) to drop **serverless** support for Next.js. You can still use Next.js but you won't able to SSR.

</details>

<details>
<summary>Why there is no free tier?</summary>

At Stormkit, we're dedicated to offering an exceptional user experience through our carefully crafted product. As Stormkit is self-funded, we're investing our energy into developing a solution that truly addresses your requirements.

To maintain our commitment to quality, our cloud solution remains as a paid service, whereas self-hosted users can explore freely the product. This approach enables us to focus on users genuinely interested in exploring our offering.

It's crucial to understand that we're entirely self-funded, without external backing. Our product's growth and development rely solely on revenue generated.

Should you desire an extended trial or a different package, don't hesitate to reach out. As a self-funded entity, we prioritize flexibility in accommodating our customers' financial situations.

</details>

<details>
<summary>Are you GDPR compliant?</summary>

Yes.

</details>

<details>
<summary>Are you Payment Card Industry Data Security Standard (PCI) compliant?</summary>

Yes.

</details>

<details>
<summary>How my data is protected?</summary>

Stormkit employs robust security measures to safeguard your data. This includes encrypting data on disk using the highly secure 256-bit Advanced Encryption Standard (AES-256). Your valuable customer data is backed up hourly to ensure its safety. Additionally, we prioritize security by default through our utilization of HTTPS/SSL protocols.

</details>

<details>
<summary> What redundancies does Stormkit.io have in place? </summary>
At Stormkit.io, reliability is a top priority. We leverage the robust infrastructure provided by Amazon Web Services (AWS) to build our platform. This ensures that our services are built on a foundation known for its scalability, durability, and high availability. We understand the critical nature of your applications and websites. That's why we've implemented redundancy measures across our entire platform. This includes redundancy at both hardware and software levels, ensuring that in the unlikely event of a failure, there are backup systems in place to seamlessly take over.
</details>

<details>
<summary> How does Stormkit.io handle regional availability? </summary>
Stormkit.io serves content from multiple geographic zones in Europe, ensuring that your applications and websites are delivered reliably and quickly to users. Moreover, we have the capability to open new regions on demand, providing you with even greater flexibility.
</details>

<details>

<summary> How does Stormkit.io handle unexpected traffic spikes? </summary>

We're prepared for unexpected traffic spikes. Our platform is designed to scale dynamically, automatically adjusting resources to meet demand. This ensures that your applications remain responsive and available, even during periods of sudden increased traffic.

</details>

<details>
<summary> How does Stormkit.io manage updates and maintenance? </summary>
We understand the importance of minimizing disruptions. Our team carefully plans updates and maintenance activities to ensure they have minimal impact on your services. When updates are required, we provide advance notice and select time windows that have the least impact on your users.

</details>

<style>
/* Style the summary element */
details summary {
  cursor: pointer;
}

/* Style the content of the collapsible section */
details:not([open]) > *:not(summary) {
  display: none;
}
</style>
