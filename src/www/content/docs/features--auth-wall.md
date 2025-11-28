---
title: Protect your deployments with Auth Wall
description: Learn how to configure and use the Auth Wall feature to restrict access to your deployments, manage authorized users, and enhance security for sensitive environments.
---

# Auth Wall

## Overview

The **Auth Wall** feature allows you to restrict access to your deployments by requiring user authentication. This is particularly useful for protecting sensitive environments, such as staging or preview deployments, ensuring that only authorized users can access them.

You can locate the Auth Wall settings by navigating to **Environment** > **Config**. Access the configuration options by selecting the **Auth Wall** tab in the left-hand sidebar.

## Configuration Options

The Auth Wall has three main configuration modes:

### 1. Auth Wall is disabled

- No authentication requirement for any deployments

### 2. Protect all endpoints including your domains

- Every page and endpoint across all domains requires authentication

### 3. Protect only deployment previews

- Only preview deployments require authentication
- Production deployments remain publicly accessible

## Managing Authorized Users

The **Auth Users** section allows you to manage the list of users who have access to the deployment when the Auth Wall is enabled.

### 1. Viewing Authorized Users

- A table displays the list of users with access.
- Each row includes:
  - **Email**: The email address of the authorized user.
  - **Last Login**: The timestamp of the user's last login (if available).
  - **Checkbox**: A checkbox to select the user for removal.

### 2. Adding Users

- To add a new user, click the **Add User** button (indicated by a `+` icon).
- You will be prompted to enter the user's email address and choose a password for them.

### 3. Removing Users

To remove a user:

1. Select the user by checking the box next to their email.
2. Click the **Remove Selected** button to revoke their access.

**Note**: Removing a user will immediately revoke their access to the deployment, however user with sessions won't be invalidated.

## Best Practices

- **Limit access**: Only add users who need access to the deployment to minimize security risks.
- **Regularly review users**: Periodically review the list of authorized users and remove those who no longer need access.
- **Use for sensitive environments**: Enable the Auth Wall for environments that contain sensitive data or are not ready for public access.
- **Combine with other security measures**: The Auth Wall is a layer of protection but should not be used as the primary layer to secure your application.

## Troubleshooting

- **Users cannot access the deployment**:
  - Ensure the Auth Wall is enabled and the user is added to the authorized users list.
  - Verify that the user is logging in with the correct credentials.
- **Deployment is still public**:
  - Check the protection scope. If "Protect only deployment previews" is selected, published deployments will remain public.
  - Ensure the Auth Wall is enabled and the settings have been saved.
- **Changes not applied**:
  - Make sure to click the **Save** button after making changes to the Auth Wall settings.

## Additional Notes

- By default, a user remains logged in for maximum 24 hours.
