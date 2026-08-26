---
title: Application runtime
description: Run long-running server processes on self-hosted Stormkit instances, including Go programs.
keywords: go, golang, runtime, start command, self-hosted
---

# Application runtime

<section>

Stormkit can run long-running server processes (for example Go HTTP servers) by using the **Start command** setting.
This option is available only on **self-hosted** Stormkit instances.

</section>

## When to use it

<section>

On a self-hosted instance, the Application Runtime is the **recommended way to run
backend code**. [Serverless functions](/docs/features/writing-api) are a Stormkit Cloud
feature; self-hosted instances execute them by forking a short-lived `node` process per
request, which is fine for parity and local development but not for production traffic.

Reach for a Start command whenever your backend:

- pays a cost per request it should pay once — booting a runtime, opening a connection,
  loading a model or a large file into memory,
- benefits from state that outlives a single request — connection pools, warm caches,
  a headless browser you keep open,
- needs system binaries from a [Nix flake](/docs/self-hosting/runtimes) at request time,
  not only during the build.

The process is reaped after 10 minutes without a request. Published environments are kept
warm by the domain ping, so this is invisible in practice; preview deployments, which are
not pinged, pay a cold start after an idle spell. Set `STORMKIT_MAX_IDLE` (in minutes) as
an environment variable to widen the window.

Requests reach your server through Stormkit's proxy, which applies
`STORMKIT_HTTP_PROXY_TIMEOUT`
([default 30 seconds](/docs/self-hosting/advanced-configuration)) to the time your server
may take to start sending response headers. This is the only request deadline on a
self-hosted instance — the 15 second function timeout is a Stormkit Cloud limit and does
not apply here. Raise it, or set it to `0`, if a request legitimately needs longer.

When the deadline is hit, Stormkit answers `504 Gateway Timeout` with a page that names
`STORMKIT_HTTP_PROXY_TIMEOUT` and marks the response with `X-Stormkit-Error: proxy-timeout`.
Your server is usually still running and finishes the work afterwards — a 504 here means the
proxy stopped waiting, not that the process crashed.

The runtime is not Go-specific — any process that listens on `PORT` works, including a
Node/Express or Fastify server. Go is used in the examples below because compiling a
binary is the most common case.

</section>

## Go programs

<section>

To run a Go program, you typically compile a binary during the build step and start it with the Start command.

</section>

### Requirements

- A `go.mod` file in your build root.
- A Go runtime version set via `.go-version` or `mise.toml`.
- Your server must listen on the `PORT` environment variable.

Example `.go-version`:

```bash
1.22.5
```

Example `mise.toml`:

```bash
[tools]
go = "1.22.5"
```

### Configuration

In **Your App** > **Environments** > **Config**:

- **Build command**: `go build -o dist/app ./cmd/server`
- **Output folder**: `dist`
- **Start command**: `./dist/app`

This configuration will:

- Build your Go binary into `dist`.
- Upload that folder as the server artifact.
- Start the binary when requests arrive.

The start command runs from the root of the uploaded artifact, so it has to
include the server folder in the path (`./dist/app`, not `./app`).

### Minimal server example

```go
package main

import (
  "log"
  "net/http"
  "os"
)

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "3000"
  }

  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello from Go"))
  })

  log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### Notes

- You can also use `go run ./cmd/server` as the Start command, but compiling a binary is faster and more reliable.
- If your binary needs extra assets (templates, migrations, etc.), place them under your server folder as part of the build step.
