---
title: Deployment Status Checks
description: Learn how to implement post-deployment status checks to conditionally publish or withhold a deployment.
---

# Deployment Status Checks

Status checks allow you to automatically verify conditions after a successful deployment. Based on the results, you can choose to either publish the deployment or delay it, depending on the checks’ outcome.

## Configuring Status Checks

To set up status checks:

1. Navigate to **Environment Config** > **Deployment Settings** > **Status Checks**.
2. Click **Add Status Check**.
3. Enter the command to be executed for the status check.
4. (Optional) Provide a name and description for the check to give context to other developers.
5. Click **Save** to apply the status check.

The command will execute with the same environment variables available during the build process.

For an example of a status check script, visit our [sample repository](https://github.com/stormkit-io/sample-project/blob/main/scripts/puppeteer.ts).

## Modifying an Existing Status Check

To modify an existing status check:

1. Navigate to **Environment Config** > **Deployment Settings** > **Status Checks**.
2. Click the menu button ( **...** ) next to the status check you want to modify.
3. Update the necessary fields, such as the command, name, or description.
4. Click **Save** to apply the changes.

This will update the status check for future deployments.

---

By using status checks, you ensure that only deployments meeting your predefined conditions will be published, adding an additional layer of control and reliability to your CI/CD pipeline.
