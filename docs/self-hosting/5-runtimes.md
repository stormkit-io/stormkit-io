---
title: Runtime Management
description: Learn how to manage programming language runtimes, package managers, and development tools in your self-hosted Stormkit instance using the Admin Dashboard and mise runtime manager.
---

# Runtime Management

This document explains how to manage runtimes in your **self-hosted Stormkit instance** using the **Admin Dashboard**.

## Overview

Stormkit’s runtime management system allows you to control which programming languages, package managers, and tools are available during your app deployments.

You can:

- Install and manage multiple runtimes (Node.js, Go, npm, Angular CLI, etc.)
- Specify exact versions or use `latest`
- Enable or disable automatic runtime installation
- Use a `flake.nix` file to provide system-level tools via Nix
- Upgrade the underlying runtime manager (**mise**)

## Accessing the Runtime Management Page

<div class="blog-alert">

Note: You have to be an administrator to access this area.

</div>

1. Log into your **Stormkit Dashboard**.
2. Click on your **Profile** > **Admin**
3. Navigate to:
   **System** > **Installed runtimes**

## Managing Installed Runtimes

### Adding a Runtime

1. Click **Add Row**.
2. Enter:
   - **Runtime name** — e.g., `node`, `go`, `npm`, `npm:@angular/cli`
   - **Runtime version** — Specific version (e.g., `24`, `1.24`) or `latest`
3. Click **Save**.

<div class="blog-tip">

**Tip:** Refer to the [mise documentation](https://mise.jdx.dev/) for a complete list of supported tools.

</div>

### Removing a Runtime

- Click the **`×`** icon next to the runtime you want to remove.
- Click **Save** to apply changes.

### Auto Install

When **Auto install** is enabled, Stormkit automatically installs required runtimes during deployment based on your app’s version configuration files.

- **Enabled**: Runtimes will be installed automatically if missing.
- **Disabled**: Only pre-installed runtimes will be available.

To toggle:

1. Use the switch under **Auto install**.
2. Save your changes.

The following files are recognized automatically:

| Runtime | Files                                 |
| ------- | ------------------------------------- |
| go      | `.go-version`                         |
| node    | `.nvmrc`, `.node-version`             |
| python  | `.python-version`, `.python-versions` |
| ruby    | `.ruby-version`, `Gemfile`            |

## Nix Flakes

If your repository contains a `flake.nix` file, Stormkit will automatically run `nix develop` during the **install runtimes** step to bootstrap the Nix development shell. The packages defined in the flake are then available for all subsequent build commands — no additional configuration required — and, as described under **Availability at runtime** below, to your deployed server as well.

A typical `flake.nix` that provides `ffmpeg` and `imagemagick` during builds:

```javascript
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = [ pkgs.ffmpeg pkgs.imagemagick ];
    };
  };
}
```

`flake.nix` and `mise.toml` can coexist — mise handles language runtimes while the flake covers system-level tools.

### Availability at runtime

The flake is not a build-only feature. `flake.nix` and `flake.lock` are copied into
the server artifact, and when your environment has a **Start command**, Stormkit wraps
that command in `nix develop` before spawning the process. Your server therefore starts
_inside_ the same shell your build commands ran in, so every package in the flake is on
`PATH` — system binaries such as headless-browser dependencies can be launched from your
application at request time, not only from build commands.

This also explains why `/nix` has to be mounted on the `hosting` (serving) service and
not only on the `workerserver` (builds, periodic jobs): the store has to be there when
the app runs, too. Give each service its own volume rather than sharing one — see
**Persisting the Nix Store** below.

<div class="blog-alert">

Two conditions apply:

- **Auto install** must be enabled — the `nix develop` wrapper is skipped when it is off.
- It applies to the [Application Runtime](/docs/deployments/application-runtime)
  (**Start command**) only. [Serverless functions](/docs/features/writing-api) are
  invoked as a plain `node` process with no Nix shell around them, so flake packages are
  **not** on `PATH` there. If your code needs system binaries, run it behind a start
  command.

</div>

The first request after a deployment may take a while, or briefly show a "dependencies
are being installed" page, while Nix realises the shell. Subsequent requests reuse the
warm store.

A `flake.nix` providing headless Chromium to a start-command server:

```javascript
{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }: let
    system = "x86_64-linux";
    pkgs = nixpkgs.legacyPackages.${system};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = [ pkgs.playwright-driver.browsers pkgs.chromium ];

      shellHook = ''
        export PLAYWRIGHT_BROWSERS_PATH=${pkgs.playwright-driver.browsers}
        export PLAYWRIGHT_SKIP_VALIDATE_HOST_REQUIREMENTS=true
      '';
    };
  };
}
```

## Mise Runtime Manager

Stormkit relies on the **[mise](https://mise.jdx.dev/)** open-source tool for runtime management. **Current version** is displayed in the **Mise** section.

### Upgrading Mise

1. Click **Upgrade to latest**.
2. Stormkit will fetch and install the newest stable release of `mise`.

> **Note:** Upgrading `mise` does not automatically upgrade installed runtimes. You’ll need to update those manually.

## Persisting the Nix Store

Stormkit uses [Nix](https://nixos.org/) as a backend for maintaining system level dependencies. By default, the Nix store lives at `/nix` inside the container. Without a persistent volume at that path, packages are re-downloaded every time the container restarts.

As of **April 2026**, the official `docker-compose.yaml` mounts a named `nix` volume at `/nix` for both the `workerserver` and `hosting` services. If you set up Stormkit before this date, add the following to your `docker-compose.yaml` to benefit from cached Nix packages:

**1. Declare the volumes** at the top-level `volumes` section:

```yaml
volumes:
  workerserver_nix:
  hosting_nix:
```

**2. Mount them** in the respective service definitions:

```yaml
# workerserver
- workerserver_nix:/nix

# hosting
- hosting_nix:/nix
```

After updating, run `docker compose up -d` to recreate the containers with the new mount. The Nix store will be populated on first use and reused on subsequent restarts.

<div class="blog-alert">

**Existing installations:** If you see an error like `failed to mkdir .../nix/_data/store/...: file exists` on startup, it means both services raced to seed the same volume simultaneously. Use separate named volumes to avoid this:

```yaml
volumes:
  workerserver_nix:
  hosting_nix:
```

Then mount them individually:

```yaml
# workerserver
- workerserver_nix:/nix

# hosting
- hosting_nix:/nix
```

</div>

## Best Practices

- **Pin versions** for production apps to ensure predictable builds.
- Keep **mise** updated for the latest runtime management features.
- Use `latest` only for development or experimental environments.
- Regularly review installed runtimes and remove unused ones.

## Related Documentation

- [Mise Runtime Manager](https://mise.jdx.dev/)
- [Nix Package Manager](https://nix.dev/reference/nix-manual.html)
