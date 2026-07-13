---
title: How does Stormkit deploy?
description: Information about how Deployments work on Stormkit.
---

# Stormkit Deployment Overview

Stormkit leverages AWS infrastructure for its cloud deployment. Each deployment on Stormkit can encompass three types of files: static files, server files, and API files, all securely stored in our S3 buckets.

## Folder Structure

By default, Stormkit looks for a top-level `.stormkit` subfolder with the following structure:

- `public/`
- `server/`
- `api/`

To modify the working directory, navigate to **Your App** > **Environments** > **Config** > **Deployment settings** > **Build** and update the `Root Directory` setting.

To specify a different subfolder other than `.stormkit`, visit **Your App** > **Environments** > **Config** > **Deployment settings** > **Build** and update the `Output folder` setting. If changed, the folder structure mentioned above is also validated against this folder. If it differs, the entire content of the directory will be uploaded.

If the deployment lacks a `.stormkit` subfolder and the output folder isn't specified, Stormkit checks for these common subfolders:

- `out`
- `output`
- `dist`
- `build`
- `public`

If none are found, Stormkit uploads everything under the `Root Directory`.

## Server files

In the `server` subfolder, an entry point is determined by locating one of the following files:

- `index.js`
- `server.js`
- `main.js`

These files can also have the `mjs` and `cjs` extensions. If none are found, the function returns a 404 error.

The entry file must export a function named `handler`, wrapped by our `serverless` helper, to receive standard Node.js Request and Response objects.

```ts
import serverless from '@stormkit/serverless'

export const handler = serverless(
  async (req: http.IncomingMessage, res: http.ServerResponse) => {
    res.write('Hello from ' + req.url)
    res.end()
  }
)
```

## API files

Our API files follow the file system routing, as detailed in our [dedicated section](/docs/features/writing-api) for API Files.

Each function should be in a separate file and export a default method:

```ts
export default async (req: http.IncomingMessage, res: http.ServerResponse) => {
  res.write('Hello from ' + req.url)
  res.end()
}
```

In this case, the `serverless` wrapper is omitted because the API function has its own entry file, handling the routing mechanism and loading the appropriate file.

## Static files

All files under the `.stormkit/public` (or the configured output folder) will be deployed to our S3 bucket and served by our Load Balancer as static files.

## Example

Check out and build our [React Starter Template](https://github.com/stormkit-io/monorepo-template-react) to see an example of the `.stormkit` subfolder.
