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

## Documentation

<section>

A trigger's cron and URL say when and where it fires, but not why it exists or
what breaks when it stops. The **Documentation** field on the trigger modal
holds that context: free-form **markdown**, with an Edit/Preview toggle while
you write it.

It is shown alongside the trigger's run details (expand the dot menu `(...)` >
**Past triggers** > a run), so whoever is looking at a failed run also sees what
the trigger is for and who to contact. The text is never sent with the request
and never affects execution.

Documentation can also be set through the [API](/docs/api/triggers) and the MCP
`create_trigger` / `update_trigger` tools, which is a convenient way to have an
agent write up a trigger it just created.

</section>

## Environment variables

<section>

The trigger's **URL**, **header values** and **payload** support environment
variable interpolation. Reference a variable with `$NAME` or `${NAME}` and it is
replaced at run time with the value of the matching variable from the
**environment's configuration** (**Environment** > **Config** > **Environment
variables**).

For example, define a `CRON_SECRET` variable on your environment and reference
it in the trigger's `Authorization` header:

```
Authorization: Bearer $CRON_SECRET
```

At run time `$CRON_SECRET` is replaced with the variable's value, so the secret
lives in your environment config instead of the trigger itself — and rotating it
is just an env-var change, no trigger edit needed.

A reference with no matching variable is left untouched (it is sent literally),
and variables are resolved **only when the trigger runs** — the stored trigger
always keeps the raw `$NAME` reference. Only the environment's own variables are
available; host/system variables are never exposed to a trigger.

</section>

## Debugging

Stormkit saves the request and response for each periodic task. You can view the last 25 logs for each trigger by expanding the dot menu `(...)` and clicking on the `Past triggers` menu item.

## Self Hosting

<section>
If you are self-hosting Stormkit, the periodic jobs are handled by the workerserver.
</section>
