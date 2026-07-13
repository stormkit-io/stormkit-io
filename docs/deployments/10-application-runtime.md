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

- **Build command**: `go build -o .stormkit/server/app ./cmd/server`
- **Output folder**: `.stormkit`
- **Start command**: `./app`

This configuration will:

- Build your Go binary into `.stormkit/server`.
- Upload that folder as the server artifact.
- Start the binary when requests arrive.

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
- If your binary needs extra assets (templates, migrations, etc.), place them under `.stormkit/server` as part of the build step.
