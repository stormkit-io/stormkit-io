---
title: Hunting Zombie Processes in Go and Docker
date: 2025-04-16
description: A technical deep dive into debugging zombie processes in a Go and Docker setup, detailing how I fixed a server crash caused by Node.js process leaks in Stormkit. Learn about process groups, SIGCHLD handling, and using Tini for zombie reaping.
author-name: Savas Vedova
author-tw: @savasvedova
author-img: https://pbs.twimg.com/profile_images/1681635649298874370/IMQmYpcA_400x400.jpg
---

Last week, I wrestled with a nasty issue: zombie processes crashing my demo server 🧟‍♂️ If you’ve dealt with process management in Go or Docker, you might find this journey relatable. Here’s the technical deep dive into how I tracked down and fixed it.

## The setup

We have a feature in Stormkit that spins up Node.js servers on demand for self-hosted users, using dynamic port assignment to run multiple instances on one server. It’s built in Go, leveraging `os/exec` to manage processes. The system had been rock-solid—no downtime, happy users.

Recently, I set up a demo server for server-side Next.js and Svelte apps. Everything seemed fine until the server started crashing randomly with a Redis Pub/Sub error.

## Initial debugging

I upgraded Redis (from 6.x to 7.x), checked logs, and tried reproducing the issue locally—nothing. The crashes were sporadic and elusive. Then, I disabled the Next.js app, and the crashes stopped. I suspected a Next.js-specific issue and dug into its runtime behavior, but nothing stood out.

Looking at server metrics, I noticed memory usage spiking before crashes. A quick ps aux revealed a pile of lingering Next.js processes that should’ve been terminated. Our spin-down logic was failing, causing a memory leak that exhausted the server.

## Root cause: Go's `os.Process.Kill`

The culprit was in our Go code. I used `os.Process.Kill` to terminate the processes, but it wasn’t killing child processes spawned by npm (e.g., `npm run start` spawns `next start`). This left orphaned processes accumulating. Here’s a simplified version of the original code:

```go
func stopProcess(cmd *exec.Cmd) error {
  if cmd.Process != nil {
    return cmd.Process.Kill()
  }

  return nil
}
```

I reproduced this locally by spawning a Node.js process with children and killing the parent. Sure enough, the children lingered. In Go, `os.Process.Kill` sends a `SIGKILL` to the process but doesn’t handle its child processes.

## Fix attempt: Process groups

To kill child processes, I modified the code to use process groups. By setting a process group ID (PGID) with `syscall.SysProcAttr`, I could send signals to the entire group. Here’s the updated code (simplified):

```go
package main

import (
  "log"
  "os/exec"
  "syscall"
)

func startProcess() (*exec.Cmd, error) {
  cmd := exec.Command("npm", "run" "start")
  cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Assign PGID

  if err := cmd.Start(); err != nil {
    return nil, err
  }

  return cmd, nil
}

func stopProcess(cmd *exec.Cmd) error {
  if cmd.Process == nil {
    return nil
  }

  // Send SIGTERM to the process group
  pgid, err := syscall.Getpgid(cmd.Process.Pid)

  if err != nil {
      return err
  }

  return syscall.Kill(-pgid, syscall.SIGTERM) // Negative PGID targets group
}
```

This worked locally: killing the parent also terminated the children. I deployed an alpha version to our remote server, expecting victory. But `ps aux` showed `<defunct>` next to the processes — zombie processes! 🧠

## Zombie processes 101

In Linux, a zombie process occurs when a child process terminates, but its parent doesn’t collect its exit status (via wait or waitpid). The process stays in the process table, marked `<defunct>`. Zombies are harmless in small numbers but can exhaust the process table when accumulates, preventing new processes from starting.

Locally, my Go binary was reaping processes fine. Remotely, zombies persisted. The key difference? The remote server ran Stormkit in a Docker container.

## Docker’s zombie problem

Docker assigns PID 1 to the container’s entrypoint (our Go binary in this case). In Linux, PID 1 (`init/systemd`) is responsible for adopting orphaned processes and reaping its own zombie children, including former orphans it has adopted. If PID 1 doesn’t handle `SIGCHLD` signals and call wait, zombies accumulate. Our Go program wasn’t designed to act as an init system, so it ignored orphaned processes.

## The solution: Tini

After investigating a bit more, I found out that reaping zombie processes is a long-standing problem with Docker - so there were already solutions in the market. Finally I found Tini, a lightweight init system designed for containers. Tini runs as PID 1, properly reaping zombies by handling SIGCHLD and wait for all processes. I updated our Dockerfile:

```go
ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/app/stormkit"]
```

Alternatively, I could’ve used Docker’s --init flag, which adds Tini automatically.

After deploying with Tini, `ps aux` was clean — no zombies! 🎉 The server stabilized, and the Redis errors vanished as they were a side effect of resource exhaustion.

## Takeaways

- **Go process management**: os.Process.Kill doesn’t handle child processes. Use process groups or proper signal handling for clean termination.
- **Docker PID 1**: If your app runs as PID 1, it needs to reap zombies or delegate to an init system like Tini.
- **Debugging tip**: Always check ps aux for <defunct> processes when dealing with crashes.
- **Root cause matters**: The Redis error was a red herring — memory exhaustion from zombies was the real issue.

This was a very educative process for me, so I thought sharing it with the rest of the community. I hope you enjoyed it!
