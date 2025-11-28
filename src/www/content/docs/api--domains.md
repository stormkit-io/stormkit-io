---
title: Domains API
description: Documentation on handling domains through Stormkit API.
---

# Domains API

<details>

<summary>
  <span>POST </span><span>/v1/domains</span>
</summary>

Add a domain.

```typescript
interface Request {
  domain: string
}

interface Response {
  domainId: string
  token: string
}
```

```bash
# Example

curl -X POST \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/domains' \
     -d '{ "domain": "example.org" }'
```

```json
{
  "domainId": "18914",
  "token": "AiX8xKhrGvsnwTTvT7yoxUFlTYzjn3bm"
}
```

Successful responses return the newly created domain id and a token to start the verification process.

</details>

<details>

<summary>
  <span>GET </span><span>/v1/domains</span>
</summary>

Return domains attached to an environment.

```typescript
interface QueryString {
  afterId?: string
  pageSize?: string // max 250
  verified?: "true"
}

interface Response {
  domains: []Domain
  pagination: Pagination
}
```

```bash
# Example

curl -X GET \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/domains'
```

```json
{
  "domains": [
    {
      "id": "18914",
      "domainName": "exmaple.org",
      "verified": false,
      "token": "AiX8xKhrGvsnwTTvT7yoxUFlTYzjn3bm"
    }
  ],
  "pagination": {
    "hasNextPage": false
  }
}
```

To filter domains by verification status, you can use the `verified` querystring parameter.

To paginate results, you can specify the `afterId` querystring parameter. This value is returned in the first `GET` request.

</details>

<details>

<summary>
  <span>DELETE </span><span>/v1/domains</span>
</summary>

Delete a Domain by it's id.

```typescript
interface QueryString {
  id: string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X PUT \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/domains?id=1501'
```

</details>

<details>

<summary>
  <span>PUT </span><span>/v1/domains/cert</span>
</summary>

Update the custom certificate of a domain.

- The `certKey` is the private key in PEM format.
- The `certValue` is the certificate value in PEM format.

You can use `openssl` to generate PEM files from `crt`:

```
openssl x509 -in example_org.crt -out example_org.pem -outform PEM
```

```typescript
interface Request {
  domainId: string
  certKey: string
  certValue: string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X PUT \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/domains/cert' \
     -d '{ "domainId": "2500", "certValue": "-----BEGIN CERTIFICATE-----", "certKey": "-----BEGIN PRIVATE KEY-----" }'
```

```json
{
  "ok": true
}
```

</details>

<details>

<summary>
  <span>DELETE </span><span>/v1/domains/cert</span>
</summary>

Deletes the custom certificate associated with the domain.

```typescript
interface Request {
  domainId: string
}

interface Response {
  ok: boolean
}
```

```bash
# Example

curl -X DELETE \
     -H 'Authorization: <api_key>' \
     -H 'Content-Type: application/javascript' \
     'https://api.stormkit.io/v1/domains/cert?id=2500'
```

```json
{
  "ok": true
}
```

</details>

#### Syntax

```typescript
interface Domain {
  id: string
  domainName: string
  verified: boolean
  token: string
  customCert?: {
    value: string
    key: string
  }
}

interface Pagination {
  hasNextPage: boolean
  afterId?: string
}
```

| Property          | Definition                                                                           |
| ----------------- | ------------------------------------------------------------------------------------ |
| id                | The unique id of the domain.                                                         |
| domainName        | The domain name.                                                                     |
| verified          | Whether the domain is verified or not. For self-hosted users, this is always `true`. |
| token             | A token used to create TXT record to verify a domain.                                |
| customCert?.value | If present, the value of the custom certificate.                                     |
| customCert?.key   | If present, the private key of the custom certificate.                               |
