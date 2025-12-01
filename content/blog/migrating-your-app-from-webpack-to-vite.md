---
title: Migrating your app from Webpack to Vite
description: Learn how to migrate your app from Webpack to Vite with this quick guide. Discover the configuration options used in Vite and the rationale behind the migration.
date: 2022-08-29
---

We recently migrated our application from Webpack to Vite.
Since I had to follow a trial and error methodology due to lack of documentation, I wanted to create
a quick guide on our migration.

For those who'd like to see directly the code [here are the changes](https://github.com/stormkit-io/app-stormkit-io/pull/406/files#diff-6a3b01ba97829c9566ef2d8dc466ffcffb4bdac08706d3d6319e42e0aa6890dd).

First thing you should know is that Vite uses `esbuild` for development and `rollup` for production. So while your development environment is working, your production build may fail. That was indeed the case for me and I'll explain in a moment why it happened.

Let's take a look to the configuration options used in vite:

```js
import { defineConfig } from "vite";

export default defineConfig(() => {
  return {
    // Although this is called root,
    // in our case this is "src",
    // where our application logic is located.
    // All configuration options defined
    // afterwards will be relative to this endpoint.
    root: "src",

    // This is where our assets are served from.
    // In webpack, this was "output.publicPath".
    base: "/",

    // This is similar to Webpack's DefinePlugin.
    // It is used to inject constants in your application.
    define: {
        // Using the following configuration made our build fail as
        // mentioned above. I had to change a couple of `global` usages
        // and remove this configuration to make them work.
        // global: { }
    }

    server: {
      // We need https for various 3rd party tools used
      // in local environment. It is similar to
      // Webpack Dev Server's 'server: "https"' config.
      // We require a certificate for this to work and that is
      // achieved through the 'basicSsl' plugin.
      // You can check the actual code to see which package is that.
      https: true,

      // We are proxying all requests from `/api` to our api server.
      // This config is very similar to Webpack Dev Server's proxy config.
      proxy: { }
    }
  },

  // This is almost equivalent to Webpack's "resolve" config.
  // It is used to import files using aliases.
  resolve: { },

  // Assets under this folder will be copied to the dist folder.
  // We used to achieve the same functionality using
  // the "copy-webpack-plugin" package.
  // Do not forget that this is relative to the "root" path configured above.
  publicDir: "./public",

  // List of plugins used to make various technologies work.
  plugins: [
     // used to add https to local environment
     basicSsl(),
     // used to include our bundles inside the html file
     createHtmlPlugin({}),
     // used to import svgs
     svgr(),
     // used to make react work with vite
     react(),
  ],

  build: {
    // Specify the dist folder
    outDir: "../dist"
  },

  // Loading a file was failing and adding this configuration
  // fixed it. It basically replaces top-level "this" usage
  // with the "window" object.
  esbuild: {
    define: {
      this: "window"
    }
  }
})
```

We do use Tailwind and Material UI in our project. A simple `postcss.config.js` was all that I needed to make Tailwind work properly:

```js
module.exports = {
  plugins: [require('tailwindcss'), require('autoprefixer')],
}
```

## The rationale behind the migration

This definitely adds no direct value for the end user. So why did we migrate?

The main reason is because we'll make a lot of changes to the frontend soon as we're releasing a V2 for Stormkit. When I realised that the HMR was broken and I heard that Vite is extremely fast for development I decided to give it a try. Turns out, this was a pretty good decision and it will also add value to the end user because it'll increase our development speed.

## Results

- The build time seems to be a bit slower than our previous configuration. This is mainly because we used to use `esbuild`, which is extremely fast, for production builds but `vite` uses `rollup`. However, to be honest, given the increase in the HMR, we're willing to sacrifice a few seconds during production builds.
- Whereas the HMR is extremely fast. When I make a change in the UI and switch to the browser to visualize it, I used to wait a few seconds till those changes were propagated. Now, by the time I switch the Browser tab, the changes are already there. This is fantastic!

Hope this mini tutorial was helpful to understand what is needed to migrate from Webpack to Vite. Stormkit is can be considered as a large application and the migration was possible through 1 PR with almost no breaking changes.

## React starter template

As a final note, we introduced a [React starter template](https://github.com/stormkit-io/monorepo-template-react) to write fullstack javascript applications. It is powered by Vite and is capable to do prerendering, server side rendering and creating single page applications. Check it out and let us know about your experience!
