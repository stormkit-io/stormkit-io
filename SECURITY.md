# Security Policy

## Supported Versions

We actively support security updates for the following versions of Stormkit:

| Version | Supported |
| ------- | --------- |
| Latest  | ✅ Yes    |
| v1.x.x  | ✅ Yes    |
| < 1.0   | ❌ No     |

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security vulnerability in Stormkit, please report it responsibly.

### How to Report

**DO NOT** create a public GitHub issue for security vulnerabilities.

Instead, please:

1. **Email us directly** at: hello@stormkit.io
2. **Include detailed information**:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)
   - Your contact information

### What to Expect

- **Acknowledgment**: We'll acknowledge receipt within 24 hours
- **Initial Assessment**: We'll provide an initial assessment within 72 hours
- **Regular Updates**: We'll keep you informed of our progress
- **Resolution Timeline**: Most issues are resolved within 30 days
- **Credit**: We'll credit you in our security advisory (if desired)

### Responsible Disclosure Guidelines

- Give us reasonable time to fix the issue before public disclosure
- Don't access or modify data that doesn't belong to you
- Don't perform actions that could harm our services or users
- Don't share vulnerability details with others until we've addressed it

### Bug Bounty

While we don't currently have a formal bug bounty program, we do recognize and appreciate security researchers who help make Stormkit more secure. Depending on the severity and impact of the vulnerability, we may offer:

- Public recognition in our security advisories
- Stormkit swag and merchandise
- Credits on our platform

### Security Best Practices

When deploying Stormkit:

#### Self-Hosted Deployments

- Keep your Stormkit installation updated
- Use strong, unique passwords
- Enable HTTPS/TLS encryption
- Regularly backup your data
- Monitor logs for suspicious activity
- Restrict database and Redis access
- Use environment variables for secrets
- Keep your host OS and dependencies updated

#### Network Security

- Use firewalls to restrict access
- Implement proper network segmentation
- Monitor network traffic

#### Access Control

- Follow the principle of least privilege
- Use strong authentication methods
- Regularly review user permissions

## Compliance

Stormkit is designed with compliance in mind:

- **GDPR**: Data protection and privacy controls

_This security policy is reviewed and updated regularly to ensure it reflects current best practices and our commitment to security._
