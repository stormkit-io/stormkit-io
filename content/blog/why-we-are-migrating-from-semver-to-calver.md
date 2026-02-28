---
title: Why We're Migrating from SemVer to CalVer
date: 2026-02-28
description: Learn why Stormkit is moving from Semantic Versioning to Calendar Versioning. Discover how CalVer enables faster deployments, better transparency for self-hosted users, and aligns with our continuous delivery philosophy.
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1681635649298874370/IMQmYpcA_400x400.jpg
---

Today, we're announcing a significant change to how we version Stormkit: we're migrating from Semantic Versioning (SemVer) to Calendar Versioning (CalVer). This might seem like a minor technical decision, but it reflects a fundamental shift in how we think about software releases and customer value.

## The Problem with SemVer for Continuous Deployment

Don't get us wrong—Semantic Versioning is fantastic for many projects. The `MAJOR.MINOR.PATCH` system (like `2.3.1`) provides clear signals about breaking changes, new features, and bug fixes. It works brilliantly for libraries, frameworks, and software with carefully planned release cycles.

But here's the thing: we deploy multiple times per day.

At Stormkit, we practice continuous deployment. When a feature is ready, tested, and merged, it goes live. We don't wait for arbitrary version milestones. We don't batch changes into quarterly releases. We ship value to our users as soon as it's ready.

This creates a mismatch with SemVer. Should today's third deployment be `2.3.4` or `2.4.0`? Does a small UI improvement count as a minor version bump? What about the database optimization we deployed this morning — is that a patch?

## The Self-Hosted Customer Problem

Stormkit offers both a cloud version and a self-hosted option. For self-hosted customers, releases are everything. They need to know:

- When updates are available
- What changed since their version
- Whether an update is critical or optional

With our rapid deployment cadence, SemVer created an artificial bottleneck. We'd deploy 15 changes to production throughout the week, but self-hosted customers wouldn't see those improvements until we decided to "cut a release" and assign it a semantic version number.

This meant **self-hosted customers were always behind**. Not because updates weren't ready, but because we were waiting for a "good time" to bump the version number.

That's backward. Our customers shouldn't wait for us to organize our versioning scheme—they should get value as soon as it's available.

## Enter Calendar Versioning

Calendar Versioning flips the script. Instead of `2.3.1`, our versions now look like `2026.02.28.1`. The format tells you exactly when that version was released and which deployment of the day it represents.

Here's what that looks like for Stormkit:

```
YYYY.MM.DD.MICRO format
2026.02.28.1 - First release on February 28, 2026
2026.03.01.1 - First release on March 1, 2026
2026.03.01.2 - Second release on March 1, 2026
2026.03.01.3 - Third release on March 1, 2026
```

Since we deploy multiple times per day (which we often do), the MICRO number increments with each deployment: `2026.02.28.1`, `2026.02.28.2`, `2026.02.28.3`, and so on.

## The Benefits We're Seeing

### 1. Faster Value Delivery

Self-hosted customers can now track our production deployments directly. When we ship a feature to our cloud users, self-hosted customers can grab that same update immediately—no more waiting for arbitrary version milestones.

### 2. Transparent Release Cadence

With CalVer, it's immediately obvious how active development is. Compare these two scenarios:

**SemVer**: `v2.3.1` to `v2.3.2` – When did these releases happen? A week apart? A month? No idea.

**CalVer**: `2026.01.15.1` to `2026.01.16.1` – Clear as day. We released something on consecutive days.

Self-hosted customers can see our velocity and commitment to continuous improvement at a glance.

### 3. No More Versioning Debates

The version debate is over. The version is the date plus a deployment counter. Deploy on February 28th? It's `2026.02.28.1`. Second deployment? `2026.02.28.2`. Done. We spend zero mental energy on "is this patch-worthy or minor-worthy?"

## Addressing the Common Concerns

### "How do I know if an update has breaking changes?"

Great question. CalVer doesn't inherently signal breaking changes like SemVer's major version bump does. Our solution:

1. **Detailed Changelog**: Every release includes a comprehensive changelog that explicitly calls out breaking changes
2. **Migration Guides**: For any breaking changes, we provide step-by-step migration guides
3. **Deprecation Warnings**: We deprecate features before removing them, giving self-hosted users time to adapt
4. **Semantic Release Tags**: We can still tag releases like `2026.02.28.1-breaking` if needed

In practice, we've found that clear documentation matters more than the version number itself. A version bump from `2.x` to `3.x` doesn't help if the changelog says "various improvements and fixes."

### "Doesn't this only work for web apps?"

CalVer is particularly well-suited for applications with continuous deployment, which includes most SaaS products, web applications, and platforms like Stormkit. It's less ideal for libraries with downstream dependencies—you probably wouldn't want `react@2026.02.28.1`.

But for self-hosted platforms? It's perfect. Ubuntu does it (`22.04`), Windows has done it (Windows 95, 98, 2000), and many modern SaaS companies are following suit.

## Real-World Examples

We're not pioneers here—we're joining good company:

- **Ubuntu**: Uses `YY.MM` format (22.04, 23.10)
- **Kubernetes**: Transitioned to `YYYY.MM` format
- **pip**: Python's package installer uses `YY.M` format
- **Unity**: Game engine uses `YYYY.M.build` format

These projects recognized what we did: for continuously developed software with actual users (not just library consumers), calendar versioning provides clearer communication.

## Effective Immediately

We're not phasing this in—CalVer is our versioning scheme starting today. All new releases will use the `YYYY.MM.DD.MICRO` format. Self-hosted customers can see their current version and update at their own pace, and all future releases will follow the new convention.

This immediate switch reflects our commitment to transparency and continuous improvement. No gradual rollout, no confusion about which scheme we're using—just a clean transition to a versioning system that better serves our development philosophy and our customers.

## Continuous Delivery Philosophy

This change is about more than version numbers—it's about our commitment to continuous delivery of value. We don't want artificial release cycles getting in the way of shipping great features.

Every day we're improving Stormkit:

- Fixing bugs
- Optimizing performance
- Adding features
- Enhancing security
- Improving developer experience

Our self-hosted customers deserve access to those improvements the moment they're ready, not when we decide to "cut a release."

CalVer removes that friction. It aligns our versioning with our deployment reality. It gives self-hosted customers transparency into our development velocity. And it lets us focus on what matters: building great software.

## For Self-Hosted Customers: What You Need to Know

If you're running Stormkit on your own infrastructure, here's the great news: **nothing changes for you if you're using the `latest` tag**. Our Docker deployments will continue to use the `latest` tag, so your existing setup keeps working exactly as before.

The versioning change is purely internal to how we label releases. Whether we call it `v2.4.0` or `2026.02.28.1`, the `latest` tag always points to the most recent stable release.

For those who prefer to pin specific versions or want to see what's new:

1. **Seamless Updates**: If you use `latest`, you're already set—no action required
2. **Clearer Release Timeline**: Version numbers now tell you when a release was made (e.g., `2026.02.28.1`)
3. **Same Quality Guarantees**: Our testing and quality standards remain unchanged
4. **Better Documentation**: Every release includes a detailed changelog with clear upgrade instructions
5. **Flexible Update Cadence**: Update daily, weekly, or monthly—whatever works for your team

You're in control of when you update. We're just making sure the updates are available when you're ready.

## Conclusion

Moving from SemVer to CalVer isn't just a technical change—it's a philosophical one. It reflects our belief that software development should be continuous, transparent, and user-focused.

We ship multiple times per day because we believe in delivering value incrementally. We're adopting CalVer because our versioning scheme should reflect that reality, not fight against it.

For our self-hosted customers, this means faster access to improvements, clearer release tracking, and a stronger connection to our development process. That's a win we're excited to deliver.

If you have questions about the migration or how it affects your self-hosted instance, feel free to reach out. We're here to make this transition as smooth as possible.

Here's to faster iterations and continuous delivery 🚀
