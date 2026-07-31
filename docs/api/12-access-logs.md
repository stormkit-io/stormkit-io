---
title: Access Logs API
description: Documentation on reading raw HTTP access logs through the Stormkit API.
---

# Access Logs API

Access logs contain one entry per HTTP request served by Stormkit for your
environment — the path, status code, client IP, user agent and how many bytes
were sent. They are operational request logs, not visitor analytics: bot traffic
is included and the client IP and user agent are not masked.

These are distinct from **runtime logs**, which contain the output of your
server side rendered pages and API functions and are returned by
[`GET /v1/deployments/{id}/runtime-logs`](/docs/api/deployments).

This endpoint is also available as the `get_access_logs` MCP tool — see the
[MCP Server](/docs/api/mcp) docs.

---

## GET /v1/access-logs

Returns access logs for the environment, newest first.

**Base URL:** `https://api.stormkit.io`

**Authentication:** At least an environment-level API key passed as the `Authorization` header.

### Query parameters

All filters are optional and are combined with AND.

| Parameter  | Type    | Required    | Description                                                                                                                                       |
| ---------- | ------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `envId`    | string  | Conditional | Required when using an app, team, or user-level API key. Environment-level keys do not require this parameter.                                     |
| `from`     | string  | No          | Only return requests at or after this unix timestamp, in seconds. **Defaults to 24 hours ago.**                                                   |
| `to`       | string  | No          | Only return requests at or before this unix timestamp, in seconds. Defaults to now.                                                               |
| `domainId` | string  | No          | Only return requests served for this domain.                                                                                                      |
| `hostName` | string  | No          | Only return requests for this host name, e.g. `www.example.org`.                                                                                  |
| `clientIp` | string  | No          | Only return requests originating from this IP address.                                                                                            |
| `method`   | string  | No          | Only return requests with this HTTP method, e.g. `GET`.                                                                                           |
| `path`     | string  | No          | Only return requests whose path starts with this value.                                                                                           |
| `status`   | number  | No          | Only return requests answered with this HTTP status code.                                                                                         |
| `isBot`    | boolean | No          | `true` returns only bot traffic, `false` only non-bot traffic. Omit to return both.                                                               |
| `cursor`   | string  | No          | Pagination cursor. Pass `pagination.cursor` from the previous response back verbatim to fetch the next page. Treat it as opaque — its encoding may change. |

> Queries are always bounded in time so that they only scan the relevant
> partitions. If you omit `from`, the last 24 hours are returned.

### Response — 200 OK

| Field        | Type   | Description                                     |
| ------------ | ------ | ----------------------------------------------- |
| `accessLogs` | array  | Up to 100 entries, newest first.                |
| `pagination` | object | `hasNextPage`, and `cursor` when there is a next page. |

Each entry contains:

| Field              | Type    | Description                                            |
| ------------------ | ------- | ------------------------------------------------------ |
| `id`               | string  | Access log entry ID. Used as the pagination cursor.    |
| `appId`            | string  | App that served the request.                           |
| `envId`            | string  | Environment that served the request.                   |
| `deploymentId`     | string  | Deployment that served the request, `0` when unknown.  |
| `domainId`         | string  | Domain the request was served for, `0` for none.       |
| `hostName`         | string  | Host header of the request.                            |
| `requestTimestamp` | string  | Unix timestamp of the request as a string.             |
| `method`           | string  | HTTP method.                                           |
| `path`             | string  | Request path.                                          |
| `statusCode`       | number  | HTTP status code of the response.                      |
| `clientIp`         | string  | IP address of the client.                              |
| `userAgent`        | string  | User agent of the client.                              |
| `referrer`         | string  | Referrer header, empty when absent.                    |
| `isBot`            | boolean | Whether the request was classified as bot traffic.     |
| `bytesSent`        | number  | Size of the response body in bytes.                    |
| `requestId`        | string  | Correlation ID assigned to the request.                |

### Error responses

| Status | Condition                                                            |
| ------ | -------------------------------------------------------------------- |
| `403`  | Missing/invalid API key, or the token has no access to the environment. |
| `500`  | Internal server error.                                               |

### Example

```bash
curl -H 'Authorization: <api_key>' \
  'https://api.stormkit.io/v1/access-logs?status=404&isBot=false'
```

```json
// Example response
{
  "accessLogs": [
    {
      "id": "9912",
      "appId": "1000000",
      "envId": "1000001",
      "deploymentId": "8241",
      "domainId": "0",
      "hostName": "www.example.org",
      "requestTimestamp": "1696919445",
      "method": "GET",
      "path": "/missing-page",
      "statusCode": 404,
      "clientIp": "203.0.113.7",
      "userAgent": "Mozilla/5.0",
      "referrer": "",
      "isBot": false,
      "bytesSent": 512,
      "requestId": "x5818c-47818-ca2de192e"
    }
  ],
  "pagination": {
    "hasNextPage": false
  }
}
```
