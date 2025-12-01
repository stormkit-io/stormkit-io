---
title: Deployment configuration
description: How to configure Stormkit to deploy your applications.
keywords: preview links, auto publish, auto deploy, build root, environment variables
---

# Configuration

<section>

You can configure your deployments by visiting **Your App** > **Environments** > **Config** page. This page is divided
into different sections.

## Settings

<!-- prettier-ignore -->
| General settings     |             |
| -------------------- | ----------- |
| Environment name     | The name of the environment. Except the reserved name `production`, you can use any alphanumeric characters to name your environments. This setting is used purely to distinguish your environments in the User Interface. |
| Branch               | The default branch of this environment. Pick the best long-lived branch that represents this environment. |
| Auto publish         | When turned on, any successful auto deployment that is triggered by a commit to the `default branch` will be published automatically. |
| Auto deploy          | Use this setting to configure your auto deployments. See [Auto Deployments](/docs/deployments/auto-deployments) for more information. |
| Auto deploy branches | When `Auto deploy` is set to be `Custom branches`, you can use this field to configure which branches to deploy automatically. This field supports `regexp` matching. |
| Preview links        | When `Auto deploy` is turned on, Stormkit will leave the link of the deployment to the pull/merge request page. You can turn off this functionality by disabling this setting. |

<!-- prettier-ignore -->
| Build settings       |             |
| -------------------- | ----------- |
| Build command        | Use this setting to configure the build command. You can chain multiple commands with the logical and (<code>&amp;&amp;</code>) operator. This setting defaults to the `build` script in `package.json` (e.g. `npm run build`). |
| Output folder        | The folder that will be deployed. Visit [How do we deploy](/docs/deployments/how-do-we-deploy) to read more about this. |
| Build root           | The working directory relative to the Repository root. By default this is the top-level directory of the repository. The `build command` and `output folder` will be relative to this directory. |

<!-- prettier-ignore -->
| Serverless           |             |
| -------------------- | ----------- |
| API Folder           | The relative path to the `api` folder where your serverless functions reside. This path is relative to the repository root and not the `build root`. Note that changing this folder will also affect the relative path which is used to serve api functions. |

The `Serverless` settings are used only for `API` functions. Read more about [Writing API](/docs/features/writing-api).

<!-- prettier-ignore -->
| Headers              |             |
| -------------------- | ----------- |
| Headers file location | The path to the [custom headers](/docs/features/custom-headers) file. Default location is `_headers`. This path is relative to the `build root`. |

<!-- prettier-ignore -->
| Redirects            |             |
| -------------------- | ----------- |
| Redirects file location | The path to the [custom redirects](/docs/features/redirects-and-path-rewrites) file. Default location is `redirects.json`. This path is relative to the `build root`. |
| Overwrite redirects | When turned on, you can specify redirect rules that will apply to all of your deployments instantly. |

**Environment Variables**

The environment variables that will be injected during build time. These variables are also available for serverless functions runtime. Visit [System Variables](/docs/deployments/system-variables) to see which variables are injected
automatically.

Note: Variable names matching following regex will be obfuscated <code>/secret\|\_key\|password/i</code>.

**API Keys**

Generate API Keys to interact with our API and modify this environment.
