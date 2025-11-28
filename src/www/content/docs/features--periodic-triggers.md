---
title: Periodic Triggers
description: Periodic Triggers allow you to set up automated HTTP requests to your endpoints on a scheduled basis. These triggers can be used to automate recurring tasks, perform health checks, or schedule any API calls that need to run at regular intervals.
---

# Periodic triggers

Periodic Triggers allow you to set up automated HTTP requests to your endpoints on a scheduled basis. These triggers can be used to automate recurring tasks, perform health checks, or schedule any API calls that need to run at regular intervals.

<div class="img-wrapper">
  <img src="/assets/docs/features/periodic-triggers.png" alt="Periodic Triggers" />
</div>

<section>

Trigger Functions can only be called on your custom domains.

To set up a new Trigger:

1. Go to **Application** > **Environment** > **Triggers**
1. Click on **New trigger** button
1. Fill the inputs in the modal
1. Click on **Create** button

This will call the specified endpoint with the configured cron periodicity. The timezone is **UTC**.

</section>

## Debugging

Stormkit saves the request and response for each periodic task. You can view the last 25 logs for each trigger by expanding the dot menu `(...)` and clicking on the `Past triggers` menu item.

## Self Hosting

<section>
If you are self-hosting Stormkit, the periodic jobs are handled by the workerserver.
</section>
