---
title: Combining Vite.js with Stormkit for React Static Sites Guide
description: Learn how to leverage the lightning-fast Vite.js build system and Stormkit's seamless deployment platform for React static sites. Follow our quick start guide and explore migration options from Webpack. Plus, discover an all-in-one template project for server-side rendering, static page generation, and API development using React and Vite.js.
date: 2023-10-08
---

[Vite.js](https://vitejs.dev/) stands out in the world of build systems, offering unparalleled speed and developer experience. Its unique approach to front-end development leverages modern browser features and native ES modules, enabling blazing-fast development and hot module replacement (HMR) right out of the box. Whether you're working on a single-page application (SPA) or a complex web project, Vite.js ensures your workflow remains swift and efficient.

In this guide, we'll dive into harnessing Vite.js for your project and explore how it seamlessly integrates with Stormkit for deployment, resulting in a seamless end to end experience.

Quick start guide using React for statically generated websites

- Run the following command to create a new Vite.js project:
  ```
   npm create vite@latest
  ```
   (Alternatively, you can use pnpm or yarn)

- Push the project to your preferred git provider.
- Connect to Stormkit and import your project.
- Click to Deploy button

That is it, for existing projects using webpack you can follow our [migration guide](http://localhost:5173/blog/migrating-your-app-from-webpack-to-vite)

There's a wide array of possibilities with Vite.js. We've created a [template project](https://github.com/stormkit-io/monorepo-template-react) capable of server-side rendering, generating static pages, serving single-page applications, and providing an API using plain React and Vite.js.