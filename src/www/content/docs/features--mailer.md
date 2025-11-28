---
title: Mailer
description: Stormkit Mailer is a simple email service that allows you send transactional emails with ease.
---

# Mailer

Stormkit Mailer is designed to provide a straightforward way to send transactional emails directly from your application. This feature is ideal for sending automated notifications, password resets, and other essential email communications.

With just a few configuration settings, you can connect Stormkit Mailer to your SMTP provider and start sending emails quickly and reliably.

## Configuration

- Navigate to the Environments tab and select the environment (e.g., production or staging) where you want to set up email functionality.
- Go to Config > Mailer Configuration.

In the Mailer Configuration panel, you will find the following fields. Fill them out according to your SMTP provider’s details:

| Setting       | Description                                                                                           |
| ------------- | ----------------------------------------------------------------------------------------------------- |
| **SMTP Host** | Enter the hostname of your SMTP server                                                                |
| **SMTP Port** | Set the port for your SMTP server. Common ports are 587 (TLS), 465 (SSL)                              |
| **Username**  | Enter the username for SMTP authentication. Typically the email address associated with your account. |
| **Password**  | Enter the password for SMTP authentication. Ensure this is securely stored and not shared publicly.   |

## Testing the Configuration

After entering your SMTP credentials, click the `Send test email` button to verify the configuration.

You should receive a confirmation email if the SMTP settings are correct. If there’s an error, double-check your SMTP settings or contact your SMTP provider for assistance.

Once you’re satisfied with the settings, click `Save` to store the SMTP configuration for your environment.

## Example (Gmail)

| Setting   | Value                                                                     |
| --------- | ------------------------------------------------------------------------- |
| SMTP Host | smtp.gmail.com                                                            |
| SMTP Port | 587                                                                       |
| Username  | your-email@example.com                                                    |
| Password  | Your Gmail app password (if using 2FA, generate an app-specific password) |

## Sending emails programmatically

- Generate an API key for your environment: **Environment** > **Config** > **API Keys**
- Refer to our [API documentation](/docs/api/mailer)

## Additional Notes

- SMTP Security: Ensure that your SMTP credentials are kept secure and only accessible by authorized personnel.
- App Passwords for Gmail: If you’re using Gmail as your SMTP provider and have two-factor authentication (2FA) enabled, you’ll need to create an app-specific password in your Google account.

## Troubleshooting:

If you encounter issues, confirm the following:

- Your SMTP host and port are correct.
- Your email provider supports the port and authentication type you’ve selected.
- Any firewall or security settings on your network allow outbound SMTP traffic.

For further assistance, refer to your SMTP provider’s documentation or reach out to us.
