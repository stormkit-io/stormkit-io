---
title: Environments API
description: API Documentation for managing environments through Stormkit API.
---

# Environments API

<details>

<summary>
  <span>POST </span><span>/v1/env</span>
</summary>

Create an environment. An application or team level API key is required to use this endpoint.

```typescript
interface StatusCheck {
  name: string
  cmd: string
  description: string
}

interface Request {
  apiFolder?: string // The folder within the repository which contains API functions
  autoDeploy: boolean // Whether to auto deploy or not
  autoDeployBranches?: string // A regexp/glob pattern to filter which branches to automatically deploy
  autoDeployCommits?: string // A regexp/glob pattern to filter which commits to automatically deploy
  autoPublish: boolean // Whether auto publish is turned on for new deployments
  branch: string // The default branch name of this environment
  buildCmd?: string // The command to build your application
  distFolder?: string // The output folder where build assets are located
  envVars?: Record<string, string> // The environment variables
  errorFile?: string // The error file that will be served in case of an error. This file must be inside your "distFolder".
  headersFile?: string // The location to the custom headers file
  name: string // The environment name
  previewLinks?: boolean // Whether Stormkit should leave a preview link on the pull/merge requests or not
  redirects?: Redirect[] // The redirects for this environment. See Redirects and Path Rewrites for more information.
  redirectsFile?: string // The file that contains custom redirects
  serverCmd?: string // The command to run your application. This is available only to self-hosted users.
  statusChecks?: StatusCheck[] // The commands to run post-deployment
}

interface Response {
  envId: string
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/env' \
     -d '{ "appId": 1510, "branch": "my-branch", "name": "development": "envVars": { "NODE_ENV": "development" } }'
```

</details>

<details>

<summary>
  <span>DELETE </span><span>/v1/env</span>
</summary>

Delete Environment with the given id.

```typescript
interface QueryString {
  id: string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X DELETE \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/env?id=5061'
```

</details>
