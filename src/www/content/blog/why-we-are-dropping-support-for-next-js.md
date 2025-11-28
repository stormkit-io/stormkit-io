---
title: Why we are dropping support for Next.js
description: In our continuous pursuit of delivering the best developer experience, we've made a bold decision. We're dropping serverless support for Next.js.
date: 2023-05-21
search: true
---

<section class="blog-alert">

**Update 28 Aug, 2024** <br />
We recently announced our self-hosted edition which supports full features of Next.js
out-of-the-box. Check our [self-hosted](/docs/self-hosting/getting-started) guide for more information.

</section>

In our continuous pursuit of delivering the best developer experience, we've made a bold decision. We're dropping serverless support for Next.js. While this framework has its merits, we believe that simplicity and maintainability are paramount for us as a bootstrapped company. In this blog post, we'll explain the reasons behind our choice and shed light on our vision for the future.

## Struggling with Backwards Compatibility

One of the primary factors influencing our decision is the frequent and sometimes drastic changes introduced by Next.js. As a small company, keeping up with these changes and ensuring backward compatibility becomes increasingly challenging. It diverts our limited resources from focusing on delivering new features and improvements to our users.

## Competing with Giants

Next.js, in particular, has gained significant momentum with its integration with Vercel. However, it also tends to write features that are tightly coupled with Vercel's infrastructure, making it difficult for us to compete head-on. Rather than engaging in an uphill battle, we'd rather focus on providing a unique value proposition and differentiate ourselves in other areas.

## Embracing Simplicity

As developers ourselves, we value simplicity and the ability to have full control over our codebase. Frameworks often introduce a learning curve and require developers to adapt to their specific paradigms. We believe that achieving similar results can be done with a simple build file, reducing unnecessary complexity and dependencies.

## Moving Forward

By dropping serverless support for Next.js, we're streamlining our efforts to concentrate on improving our core offerings. This decision aligns with our commitment to empowering developers with a lightweight and flexible deployment and hosting solution. We'll be investing our resources in enhancing our existing features, expanding compatibility with popular tools and libraries, and providing unparalleled developer experience.

As a first step, we're thrilled to introduce our [React Starter Template](https://github.com/stormkit-io/monorepo-template-react). This template is designed to empower developers by providing all the necessary functionality without the need to rely on a specific framework. With this template, you can enjoy a simple build process and unleash your creativity to build applications tailored to your exact requirements. We believe in giving developers the freedom to shape their projects and take control of their codebase. The React Starter Template is just one example of how we're committed to enabling developers to bring their ideas to life while maintaining a lightweight and flexible approach.

It's also important to clarify that single-page applications (SPAs) and prerendered applications built with Next.js will continue to function seamlessly on Stormkit, we are only dropping support for the serverless part.

## Conclusion

As a bootstrapped company, we must make tough decisions to ensure our long-term success. Dropping serverless support for Next.js is one such decision driven by our desire to embrace simplicity and focus on our unique strengths. We're excited about the future and look forward to delivering an exceptional experience for developers who value efficiency, flexibility, and control.
