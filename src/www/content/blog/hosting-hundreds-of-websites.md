---
title: Hosting 300+ Websites - A Technical Deep Dive
description: Discover how Stormkit enables hosting over 300 websites on a lean infrastructure, with insights into setup, costs, reliability, and management.
date: 2025-05-21
---

It’s entirely feasible to host over 300 websites on a lean setup of just a few machines, and Stormkit makes it happen with a smart, scalable configuration. This post walks through the technical details how the infrastructure is set up, what it costs, how reliable it is, and what it takes to manage. Let’s dive in.

## Infrastructure Setup

The foundation is a compact yet powerful hardware and software stack designed to handle hundreds of sites efficiently. These websites generate millions of requests per month, so reliability is a core concern.

## Hardware

- **Machines**: The production setup runs on 3 servers.
- **Specs**: 2 vCPUs, 4 GiB RAM, SSD storage.
- **Cloud Provider**: Alibaba Cloud, chosen for its compliance with local regulations.

## Additional Resources

To support the hosting environment, we’ve added a few key components:

- **Network Load Balancer (NLB):** A Layer 4 load balancer ensures TLS termination happens at the source, enabling automatic TLS certificate issuance.
- **Redis Instance:** A Tair Redis OSS-compatible instance handles caching, keeps TLS certificates and acts as a message queue for storing logs and analytics.
- **PostgreSQL Database:** Apsara DB RDS provides a reliable and managed database solution.
- **Staging Machine:** A single server mirrors the production environment for testing and experimentation, keeping costs minimal.

## Deployments

Frequent deployments are a hallmark of modern development workflows. Here’s how we’ve streamlined the process:

- **GitHub Actions Integration:** Instead of scaling up worker servers, we offload CI/CD pipelines to GitHub Actions, which integrates seamlessly with Stormkit.
- **File Storage:** All static files are stored and served from Alibaba OSS (S3-compatible object storage), ensuring fast and reliable delivery.

## Software Configuration

Managing 300+ websites might sound daunting, but with the right tools, it’s surprisingly straightforward:

- **Monorepo Architecture:** All websites are managed in a single monorepo, simplifying code management and updates.
- **Stormkit Snippets:** Each website is customized using Stormkit’s Snippets feature. Conditional rules based on `domain` and `path` allow for unique configurations, ensuring every site has its own look and functionality.
- **Single Application, One Environment:** All domains are assigned to a single Stormkit environment, reducing complexity and overhead.

## Networking

Since the client primarily serves regional customers, global distribution wasn’t a priority. All resources are hosted in a single region, optimizing for simplicity and cost.

## Cost Breakdown

Here’s how the numbers stack up:

- **Alibaba Cloud Resources:** 4 servers, 1 database, 1 Redis instance, 1 NLB and around 300GB file storage cost approximately **$1000/month** with subscriptions.
- **Deployments:** GitHub Actions is included in the client’s paid tier, so there’s no additional cost for CI/CD.
- **Stormkit Services:** $500/month for infrastructure setup, service updates, monitoring, feature development, enterprise features and direct Slack support.

**Total Monthly Cost:** Around **$1500**.  
**Cost Per Website:** Roughly **$5 per site**.

It's important to note that these prices are mostly cloud provider dependent and can be further optimized by selecting alternative providers, leveraging reserved instances, or using spot instances where applicable.

## Reliability and Downtime

When hosting hundreds of websites, uptime is non-negotiable. Here’s how this setup performs:

- **Downtime:** Virtually zero downtime over the last six months.
- **Resource Usage:** Hosting machines maintain a CPU load of 0.8–1.9 and average memory consumption of 24%.
- **Efficiency:** With Stormkit’s monorepo and API automation, updates roll out to all sites quickly, minimizing manual effort.

## Key Takeaways

This setup demonstrates that hosting 300+ websites doesn’t require a sprawling infrastructure or a massive budget. With the right tools, a lean team can manage a large-scale operation efficiently.

Whether you’re managing a handful of sites or hundreds, this approach proves that scalability and cost-efficiency can go hand in hand.

## Need Help with Your Infrastructure?

If you’re looking to optimize your infrastructure or streamline your deployment process, Stormkit can help. Our platform supports deployments for JavaScript (frontend and backend), Go, and Rust applications, offering unparalleled flexibility and control.

Contact us today to learn how Stormkit can transform your hosting and deployment workflows. [Get in touch with us](https://www.stormkit.io/enterprise) for more details.
