---
title: "Self-Hosting with Stormkit: User Management"
description: Learn how to manage user access to your self-hosted Stormkit instance. Configure sign-up modes, whitelist domains, and approve or reject user registrations.
---

# Managing Users

<section id="user-management">

**Available since:** Stormkit v1.25.0

User management allows administrators to control who can sign up and access your self-hosted Stormkit instance. This feature provides fine-grained control over user registrations, making it ideal for organizations that need to restrict access to specific team members or domains.

## Accessing User Management

To access the user management configuration:

1. Click on your **profile** in the top right corner
2. Select **Admin** from the dropdown menu
3. Navigate to **Authentication** (or go directly to `/admin/auth-config`)

## Sign Up Modes

Stormkit offers three different sign-up modes to control how users can register for your instance:

### 1. Off (No new users allowed)

- Completely disables new user registrations
- Existing users can continue to log in
- Useful when you want to lock down your instance to current users only

### 2. On (All users are allowed)

- Allows anyone to sign up freely
- New users are automatically approved upon registration
- Best for open or development environments

### 3. Approval Mode (Waitlist) `Enterprise Edition`

- Requires admin approval for new user registrations
- Users can sign up, but their accounts remain pending until approved
- Supports domain-based whitelisting for automatic approval
- This option is available for enterprise customers

## Domain Whitelisting

When using **Approval Mode**, you can configure a whitelist to automatically approve or deny users based on their email domain.

### Allow Specific Domains

To automatically approve users from specific domains, enter the domains separated by commas:

```
example.org, stormkit.io
```

Users with email addresses like `user@example.org` or `user@stormkit.io` will be automatically approved, while all others will require manual approval.

### Deny Specific Domains

To automatically reject users from specific domains, prefix each domain with an exclamation mark (`!`):

```
!spam.com, !blocked-domain.org
```

Users with email addresses from these domains will be automatically rejected, while all others will require manual approval.

<div class="blog-alert">

**Important:** You cannot mix allowed and denied domains in the same whitelist. All domains must either be in allow mode (without `!`) or deny mode (with `!`).

</div>

## Managing Pending Users

When Approval Mode is enabled, users who sign up will appear in the **Pending Users** section below the user management configuration.

### Approving Users

1. Navigate to the **Pending Users** section
2. Select the users you want to approve by checking the boxes next to their names
3. Click the **Approve** button
4. Approved users will receive access to the Stormkit instance

### Rejecting Users

1. Navigate to the **Pending Users** section
2. Select the users you want to reject by checking the boxes next to their names
3. Click the **Reject** button
4. Rejected users will be removed from the pending list and will not be able to access the instance

## Best Practices

### Security Recommendations

- **Use Approval Mode** for production environments to maintain strict control over who can access your instance
- **Configure domain whitelisting** to reduce manual approval overhead for trusted domains
- **Regularly review** pending users to ensure timely access for legitimate users
- **Monitor user activity** and remove access for inactive or former team members

## Additional Notes

- User management configuration is stored in the database and persists across service restarts
- Changes to sign-up mode take effect immediately for new registrations
- Existing approved users are not affected by changes to the sign-up configuration
- Existing pending users will be affected by changes to the sign-up configuration
- Domain whitelisting is case-insensitive for email addresses

</section>
