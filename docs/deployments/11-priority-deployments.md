---
title: Priority Deployments
description: Learn how to move a queued deployment to the front of the build queue and how to auto-prioritize deployments based on commit message patterns.
---

# Priority deployments

When multiple deployments are waiting in the queue, a critical production fix can end up blocked behind routine builds. Priority deployments solve this by letting you move any queued deployment to the front of the line — without cancelling the others.

There are two ways to prioritize a deployment:

- **Manually** — using the "Run next" action on any queued deployment
- **Automatically** — by configuring a commit message pattern that triggers prioritization for matching auto-deploys

## Running a deployment next

From any environment's deployment list, open the dot menu `(...)` on a queued deployment and select **Run next**. The deployment is immediately moved to the front of the queue and a **priority** label appears on the row to confirm its status.

> Only deployments that are still waiting in the queue can be reordered. Deployments that are already being built cannot be moved.

## Auto-prioritizing by commit pattern

You can configure a regular expression that Stormkit will match against the commit message of every incoming auto-deploy. When the message matches, the deployment is automatically sent to the priority queue.

To set up auto-prioritization:

1. Go to **Application** > **Environment** > **Configuration** > **General**
1. Scroll to the **Priority deployment pattern** field
1. Enter a regular expression (e.g. `hotfix|urgent|critical`)
1. Click **Save**

The pattern is matched case-insensitively. Only auto-deploys are evaluated — manually triggered deployments are not affected.

**Examples:**

| Pattern | Matches |
|---|---|
| `hotfix\|urgent` | Any commit containing "hotfix" or "urgent" |
| `^chore\(release\):.+` | Conventional-commit release entries |
| `BREAKING` | Any commit flagging a breaking change |

## Self Hosting

<section>
Priority deployments rely on two Asynq queues — <code>deploy-service</code> (standard) and <code>deploy-service-priority</code> (high priority) — processed by the workerserver with strict priority ordering. No additional configuration is required; both queues are active by default.
</section>
