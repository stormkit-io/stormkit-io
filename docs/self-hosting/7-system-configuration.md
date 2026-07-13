---
title: "System Configuration"
description: Configure system-wide settings for your self-hosted Stormkit instance, such as artifact retention, from the Admin Interface.
---

# System Configuration

The **System** tab in the Admin Interface exposes instance-wide settings that apply to all applications and deployments on your self-hosted Stormkit instance.

## Accessing System Configuration

1. Click on your **profile** in the top right corner
2. Select **Admin** from the dropdown menu
3. Navigate to **System** (or go directly to `/admin/system`)

## Settings

### Artifact Retention Days

| Field                    | Default | Description                                                                      |
| ------------------------ | ------- | -------------------------------------------------------------------------------- |
| Artifact retention days  | `30`    | Number of days to retain unpublished deployment artifacts before they are deleted. |

Stormkit periodically removes artifacts from deployments that have not been published. Lowering this value frees up storage sooner; increasing it gives you a longer window to roll back to or inspect older builds.
