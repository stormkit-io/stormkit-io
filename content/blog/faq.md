---
title: FAQ
description: Our Frequently Asked Questions (FAQ) page is designed to provide you with quick and helpful information about Stormkit.
date: 2023-08-17
---

<details>
<summary>How Stormkit is different than Heroku?</summary>

Stormkit stands out in its capability to host static websites, single-page applications (SPAs), and serverless functions. Its optimization for performance ensures a smooth journey for JavaScript developers. The platform offers a range of features, including dynamic injection of frontend code, instant rollbacks, customizable CDN storage, and trigger functions.

Conversely, Heroku provides a broader application platform that accommodates diverse application types, including web apps, APIs, and beyond. It enables developers to deploy applications constructed with a variety of programming languages and frameworks.

</details>

<details>
<summary>Can I run my Node.js applications on Stormkit?</summary>

Not directly, Stormkit is optimized for serverless deployments and provides a serverless computing environment through its serverless functions feature, which allows you to deploy pieces of code that respond to HTTP requests. These functions are stateless and designed to be short-lived. You can use Nuxt.js functions or [use plain functions](/docs/features/writing-api) which has same interface as Node.js. If you are looking for long lived executions please contact with us. We can tailor our platform according your needs.

</details>

<details>

<summary>Is it possible to establish a database connection using Stormkit?</summary>

Certainly! Explore our [detailed blog post](/blog/monitoring-app-using-stormkit-and-supabase) as an example, demonstrating the integration with Supabase, a PostgreSQL database. By injecting database credentials as environment variables into backend functions, you can seamlessly establish a connection.

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
