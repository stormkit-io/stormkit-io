# Troubleshooting

<details>
<summary><code>go: command not found</code> after running <code>mise install</code></summary>

**Problem:** After running `mise install`, which reports "all tools are installed", running `make dev` fails with:

```
go: command not found
```

**Solution:** The mise tools aren't activated in your shell. You need to add mise activation to your shell configuration:

```bash
# Add mise activation to your shell config
echo 'eval "$(mise activate zsh)"' >> ~/.zshrc

# Reload the configuration
source ~/.zshrc

# Verify go is now available
which go
```

For other shells, replace `zsh` with your shell (e.g., `bash`, `fish`). See [mise activation docs](https://mise.jdx.dev/getting-started.html#_2-activate-mise) for more details.

</details>

<details>
<summary>Image optimization does not work on my local environment</summary>

Image optimization is disabled by default on local environments to avoid
requiring additional dependencies.

See [docs/image-optimization.md](docs/image-optimization.md) for more details on enabling and using image optimization.

</details>

<details>
<summary>API endpoints return 500 errors - <code>/api/auth/providers</code> and <code>/api/instance</code></summary>

**Problem:** When accessing the application at `https://localhost:5400`, the auth page may fail to load properly and you may see 500 Internal Server Error responses on API endpoints like `/api/auth/providers` and `/api/instance` in the browser's Network tab.

**Solution:**

### Unix

```bash
# Add api.localhost to your hosts file
echo "127.0.0.1       api.localhost" | sudo tee -a /etc/hosts

# Verify it resolves correctly
ping -c 1 api.localhost

# Start the services
make dev
```

### Windows 10 & 11

The hosts file in Windows 10 and 11 is a plain text file located at `C:\Windows\system32\drivers\etc` that maps hostnames to IP addresses.
To edit it, you must open a text editor, such as Notepad, with administrative privileges. To do so:

- Open the Global Search and type Notepad
- Click **File** > **Open**
- Locate the hosts file
- Type `127.0.0.1 api.localhost`

After applying this fix, the API proxy will work correctly and the endpoints will return proper responses.

</details>
