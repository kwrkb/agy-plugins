# GitLab MCP Server Plugin (Cross-Platform)

An MCP server plugin that lets AI assistants work with GitLab (Issues, Merge Requests, CI/CD, Releases, etc.) through the official GitLab CLI [`glab`](https://gitlab.com/gitlab-org/cli).

> **Supported OS**: Linux / macOS / Windows

## Overview

This plugin runs as an independent Go-based MCP server that executes the **system-installed `glab` command**, the same way the [github](../github/README.md) plugin wraps `gh`.

- Uses your existing `glab` configuration and `glab auth login` session; no separate token setup.
- Functions as a single unified plugin across all major operating systems.

> **Why not `glab mcp serve`?** Earlier versions of this plugin used glab's built-in `glab mcp serve`. It was EXPERIMENTAL and became **deprecated in glab v1.124.0** in favor of the [GitLab MCP Server](https://docs.gitlab.com/user/model_context_protocol/mcp_server/), which uses HTTP + OAuth and has not been verified with agy. The plugin therefore moved to wrapping `glab` itself.

## Structure

| File | Purpose |
| :--- | :--- |
| `gemini-extension.json` | Plugin manifest. Resolves `command` to absolute path `${extensionPath}${/}bin${/}gitlab` |
| `src/main.go` / `src/go.mod` / `src/go.sum` | MCP server Go source code interfacing with `glab` |
| `bin/gitlab-linux-amd64` / `bin/gitlab-darwin-arm64` / `bin/gitlab.exe` | Precompiled native binaries for each OS/architecture |
| `bin/gitlab` | OS dispatcher script (`#!/bin/sh`) executing native binary via `uname` |
| `skills/gitlab/SKILL.md` | Agent skill guide loaded on invocation (argument formatting, mandatory `-R`, `--output json`, commands to avoid, frequent patterns) |

## Prerequisites

### 1. GitLab CLI (`glab`) on PATH
Install [`glab`](https://gitlab.com/gitlab-org/cli) and make sure `glab` runs from your terminal. Verified with **v1.124.0**.

- The Ubuntu `apt` package is outdated. Install the latest release, e.g. with Go:
  ```bash
  go install gitlab.com/gitlab-org/cli/cmd/glab@latest
  ```
- A `go install` build prints `DEV` for `glab --version`. Check the actual version with:
  ```bash
  go version -m "$(command -v glab)" | awk '$1 == "mod" { print $3 }'
  ```

### 2. GitLab Authentication

```bash
glab auth login        # Authenticate if not logged in
glab auth status       # Check authentication status
```

The MCP server reuses this configuration (`~/.config/glab-cli/config.yml`).

## Provided Tools

- **`glab_command`**: Runs an arbitrary `glab` subcommand. Arguments are passed as an array of string tokens in `args` (e.g. `["issue", "list", "-R", "group/project"]`, `["mr", "view", "17", "-R", "group/project"]`).
  - Values with spaces (titles, descriptions, queries) must be passed as a **single array element**. No shell escaping or manual quoting is needed.
  - Always pass `-R group/project`: the server's working directory is not your workspace, so glab's current-repository detection is unreliable.
  - Interactive prompts are disabled (`GLAB_NO_PROMPT=1`, no stdin). A command missing a required flag fails immediately instead of waiting for input.

> ⚠️ **Warning (Arbitrary Command Execution)**: `glab_command` can execute **any** `glab` subcommand. Beyond read operations, it can perform destructive actions such as `glab mr merge`, `glab issue close`, `glab repo delete`, and arbitrary REST calls via `glab api` under the logged-in user's credentials. Ensure agent tasks are appropriately bounded and run within trusted environments.

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

### Upgrading from the `glab mcp serve` version

`agy plugin install` does not remove files left by a previous install, so the old `plugin.json` / `mcp_config.json` would stay next to the new manifest. Delete the installed plugin first, then install again:

```bash
rm -rf ~/.gemini/config/plugins/gitlab
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

> **Note**: When modifying Go sources (`src/main.go`), rebuild and commit the native binaries for all platforms using `./build.sh gitlab` (or `./build.ps1 gitlab` on Windows). `agy plugin install` copies committed binaries rather than compiling them on install.

## License & Attribution

This plugin is a wrapper that executes the user-installed `glab` from [gitlab-org/cli](https://gitlab.com/gitlab-org/cli) (MIT License). The `glab` binary is not redistributed.
