---
title: Custom certificates
description: Configure custom TLS certificates for your domains hosted on Stormkit.
keywords: custom headers
---

# Custom certificates

By default, Stormkit uses LetsEncrypt to issue certificates automatically. If you want to change this behaviour
for specific domains, you can configure custom certificates.

To do so:

- Visit your **Application** > **Environment** > **Custom Domains** page and locate the domain that you'd
  like to use a custom certificate.
- Locate the `Expand (...)` button and click on that.
- Click on **Custom certificate**.
- Provide both the `Certificate` and `Private key` that is used while creating the certificate.

**Note** Stormkit accepts only PEM encoded files.

## CRT files

If you have a certificate with a `.crt` extension, you can use `openssl` to convert it into PEM format:

```
openssl x509 -in example_org.crt -out example_org.pem -outform PEM
```
