---
title: How We Migrated from AWS to Hetzner Without Downtime
date: 2026-01-30
description: A technical deep dive into migrating Stormkit's infrastructure from AWS to Hetzner with zero downtime. Learn about database migrations, multi-ENI routing challenges, and keeping serverless functions running while switching cloud providers.
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1681635649298874370/IMQmYpcA_400x400.jpg
---

Recently, we completed a full infrastructure migration from AWS to Hetzner. Zero downtime. Production websites kept humming. Users didn't notice a thing. Here's how we pulled it off—and the rabbit holes we fell into along the way 🐇

## Why Hetzner?

Let's address the elephant in the room: AWS is expensive. For a bootstrapped company like ours, cutting infrastructure costs by 60-70% while maintaining (or improving) performance is a no-brainer. Hetzner offers bare-metal-like performance at a fraction of the cost, and their European data centers aligned perfectly with our user base.

But migrating isn't just about spinning up new servers. We had to move databases, redirect traffic from existing AWS IPs, and keep everything running while we did it. No pressure.

## The Migration Checklist

Before diving in, here's what we were dealing with:

1. **PostgreSQL database** with production data that couldn't afford a single lost transaction
2. **Multiple domains** pointing to AWS Elastic IPs
3. **Serverless functions** (Lambda) that don't have a Hetzner equivalent
4. **Production websites** serving real users 24/7

Let's tackle each one.

## Database Migration: The Scary Part

Database migrations are where things can go spectacularly wrong. One missed transaction, one replication lag spike, and you've got angry users and corrupted data. We needed a zero-downtime approach.

### The Strategy: Logical Replication

PostgreSQL's logical replication was our friend here. The plan:

1. Set up a fresh PostgreSQL instance on Hetzner
2. Configure logical replication from AWS RDS to Hetzner
3. Let it sync while production continues on AWS
4. Switch over when replication is caught up

```sql
-- On the source (AWS RDS), create a publication
CREATE PUBLICATION stormkit_pub FOR ALL TABLES;

-- On the target (Hetzner), create a subscription
CREATE SUBSCRIPTION stormkit_sub
  CONNECTION 'host=aws-rds-endpoint dbname=stormkit user=replicator password=xxx'
  PUBLICATION stormkit_pub;
```

The beauty of logical replication is that it keeps both databases in sync in near real-time. We monitored the replication lag obsessively:

```sql
SELECT * FROM pg_stat_subscription;
```

When lag hit zero consistently, we were ready to switch. The actual cutover was anticlimactic—update the connection string, restart the app, done. Total write downtime: ~2 seconds while the app restarted.

## The AWS IP Problem

Here's where things got spicy 🌶️

We had multiple Elastic IPs on AWS that customers had configured in their DNS. Changing DNS records takes time to propagate (thanks, TTL), and we couldn't ask customers to update their records on a schedule. We needed those AWS IPs to keep working but forward traffic to Hetzner.

### Enter: The EC2 NAT Gateway

The solution was surprisingly simple in concept but devilishly complex in execution: keep an EC2 instance on AWS purely to forward traffic to Hetzner.

```
Client → AWS Elastic IP → EC2 (NAT) → Hetzner Load Balancer → Hetzner Server
```

Easy, right? Just some iptables rules:

```bash
# Forward incoming traffic to Hetzner
iptables -t nat -A PREROUTING -i enX0 -p tcp --dport 443 -j DNAT --to-destination HETZNER_LB_IP:443
iptables -t nat -A PREROUTING -i enX0 -p tcp --dport 80 -j DNAT --to-destination HETZNER_LB_IP:80

# SNAT for return traffic
iptables -t nat -A POSTROUTING -d HETZNER_LB_IP -j MASQUERADE
```

Ship it! ✅

Except... we had **two Elastic IPs**. Which meant **two ENIs (Elastic Network Interfaces)**. Which meant... asymmetric routing hell.

### The Multi-ENI Nightmare

When you have multiple ENIs on an EC2 instance, traffic can come in on one interface and try to leave on another. This breaks TCP connections because the source IP doesn't match what the client expects.

Here's what we saw in `conntrack`:

```
src=CLIENT_IP dst=172.31.25.182 sport=39132 dport=443
src=HETZNER_LB_IP dst=172.31.31.236 sport=443 dport=39132
```

Traffic came in on ENI #2 (`172.31.25.182`) but responses were trying to go out via ENI #1 (`172.31.31.236`). The client would never receive the response.

### The Fix: Policy-Based Routing + Packet Marking

We needed to ensure symmetric routing—packets that come in on interface X must go out on interface X. Here's the solution:

**1. Create routing tables for each ENI:**

```bash
echo "100 enX0_table" >> /etc/iproute2/rt_tables
echo "101 enX1_table" >> /etc/iproute2/rt_tables

ip route add default via 172.31.16.1 dev enX0 table enX0_table
ip route add default via 172.31.16.1 dev enX1 table enX1_table

ip rule add from 172.31.31.236 table enX0_table
ip rule add from 172.31.25.182 table enX1_table
```

**2. Mark packets based on incoming interface:**

```bash
iptables -t mangle -A PREROUTING -i enX0 -j MARK --set-mark 1
iptables -t mangle -A PREROUTING -i enX1 -j MARK --set-mark 2
```

**3. SNAT based on the mark:**

```bash
iptables -t nat -A POSTROUTING -m mark --mark 1 -d HETZNER_LB_IP -j SNAT --to-source 172.31.31.236
iptables -t nat -A POSTROUTING -m mark --mark 2 -d HETZNER_LB_IP -j SNAT --to-source 172.31.25.182
```

**4. Disable reverse path filtering (crucial!):**

```bash
sysctl -w net.ipv4.conf.all.rp_filter=0
```

**5. Disable source/destination checks on AWS:**

In the AWS console, we also had to disable source/destination checks for both the EC2 instance and each ENI. By default, AWS drops packets if the source or destination IP doesn't match the instance's assigned IP—which breaks NAT. Go to EC2 → Network Interfaces → Actions → Change Source/Dest Check → Disable.

After hours of tcpdump, conntrack debugging, and hair-pulling, HTTPS finally worked through both IPs. The feeling when `curl https://your-domain.com` returns a 200 after all that? _Chef's kiss_ 👨‍🍳

## The Serverless Dilemma

Hetzner doesn't have Lambda. No FaaS offering. Nada.

For Stormkit, serverless functions are a core feature. Users deploy functions, and they need to run somewhere. We had three options:

1. **Build our own FaaS platform** on Hetzner (way too complex)
2. **Migrate to a different serverless provider**
3. **Keep Lambda on AWS** and route function traffic there

We chose option 3. It's pragmatic—Lambda works, it's battle-tested, and rewriting our entire function runtime wasn't worth the engineering effort. Sometimes the boring solution is the right one.

The architecture now looks like:

```
┌─────────────────┐     ┌─────────────────┐
│   Hetzner LB    │────▶│  Hetzner Apps   │
└─────────────────┘     └─────────────────┘
         │
         ▼ (function requests)
┌─────────────────┐
│   AWS Lambda    │
└─────────────────┘
```

This hybrid approach lets us benefit from Hetzner's pricing for the heavy lifting while keeping Lambda for what it does best.

## Production Website Migration

The final piece: moving actual user traffic without anyone noticing.

Our strategy was gradual:

1. **Deploy the app on Hetzner** and run it in parallel with AWS
2. **Use health checks** to ensure Hetzner was serving correctly
3. **Update the load balancer** to send traffic to Hetzner
4. **Monitor aggressively** for the first 24 hours

Having AWS as a backup meant we could instantly rollback if something went wrong. Spoiler: we didn't need to.

## Lessons Learned

1. **Test NAT gateways from outside, not localhost.** We spent hours debugging HTTPS issues that only existed because we were testing from the EC2 itself. Packets were bypassing PREROUTING entirely.

2. **Multi-ENI routing is hard.** If you're doing anything with multiple network interfaces, budget extra time for debugging. `tcpdump` and `conntrack` are your best friends.

3. **Logical replication is magic.** PostgreSQL's replication made database migration almost boring (in a good way).

4. **Hybrid cloud is fine.** You don't have to migrate everything. Keeping Lambda on AWS while running compute on Hetzner is a perfectly valid architecture.

5. **Have a rollback plan.** Every step of the way, we knew exactly how to undo what we'd done. That confidence made us move faster.

## Wrapping Up

Cloud migrations are intimidating, but they don't have to be catastrophic. With the right planning, the right tools, and a healthy dose of paranoia, you can move mountains of infrastructure without dropping a single request.

If you're considering a similar migration, feel free to reach out. We've got the battle scars and the iptables rules to prove we survived 💪
