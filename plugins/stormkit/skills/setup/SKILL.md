---
name: setup
description: Configure the Stormkit MCP connection. Use when the user wants to connect Claude Code to Stormkit, set their Stormkit host or API key, or is getting 401/connection errors from Stormkit tools.
---

# Stormkit setup

Help the user connect Claude Code to their Stormkit instance. The Stormkit MCP
server is configured in this plugin's `.mcp.json` via two environment variables:

- `STORMKIT_HOST` — base URL of the instance. Defaults to `https://api.stormkit.io`
  (Stormkit Cloud). Self-hosted users must set this to their own host, e.g.
  `https://stormkit.mycompany.com`. Do **not** include a trailing `/v1/mcp` — the
  plugin appends `/v1/mcp` automatically.
- `STORMKIT_API_KEY` — a Stormkit API key (starts with `SK_`). Created from the
  Stormkit UI under **User Settings → API Keys**.

## Steps

1. Ask the user whether they're on **Stormkit Cloud** or **self-hosted**.
   - Cloud → leave `STORMKIT_HOST` unset (the default is used).
   - Self-hosted → ask for their host URL and validate it looks like
     `https://<host>` with no path.

2. Ask for their API key (`SK_...`). Never echo the full key back; confirm only
   the last 4 characters.

3. Tell the user to export the variables in their shell profile so Claude Code
   picks them up on the next launch, for example:

   ```bash
   # Self-hosted
   export STORMKIT_HOST="https://stormkit.mycompany.com"
   export STORMKIT_API_KEY="SK_xxxxxxxx"

   # Cloud — only the key is needed
   export STORMKIT_API_KEY="SK_xxxxxxxx"
   ```

4. Have them restart Claude Code (or reload the shell) so the env vars are read,
   then run `/mcp` to confirm the `stormkit` server shows as connected.

## Troubleshooting

- **401 / unauthorized** → the API key is missing, malformed (must start with
  `SK_`), or revoked. Re-issue a key from the Stormkit UI.
- **Connection refused / DNS errors** → `STORMKIT_HOST` is wrong or the instance
  isn't reachable from the user's machine. Confirm the base URL and that the
  Stormkit API is up.
- **Config didn't take effect** → env vars are read at startup. The user must
  fully restart Claude Code after exporting them.
