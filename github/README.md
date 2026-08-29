# GitHub MCP Server Plugin (Cross-Platform)

An MCP server plugin that empowers AI assistants to interact with GitHub (Issues, Pull Requests, Repositories, etc.) via the GitHub CLI (`gh`).

> **Supported OS**: Linux / macOS / Windows

## Overview

Unlike the official `github/github-mcp-server`, this plugin runs as an independent Go-based MCP server executing the **system-installed `gh` command**.
This provides several key advantages:

- Uses existing `gh` configuration without requiring separate token setups or wrapper scripts.
- Inherits the active `gh auth login` session directly, eliminating the need to manage Personal Access Tokens (PAT).
- Functions as a single unified plugin across all major operating systems.

## Structure

| File | Purpose |
| :--- | :--- |
| `gemini-extension.json` | Plugin manifest. Resolves `command` to absolute path `${extensionPath}${/}bin${/}github` |
| `src/main.go` / `src/go.mod` / `src/go.sum` | MCP server Go source code interfacing with `gh` |
| `bin/github-linux-amd64` / `bin/github-darwin-arm64` / `bin/github.exe` | Precompiled native binaries for each OS/architecture |
| `bin/github` | OS dispatcher script (`#!/usr/bin/env sh`) executing native binary via `uname` |
| `skills/github/SKILL.md` | Agent skill guide loaded on invocation (argument formatting, mandatory `-R`, `--json` field selection, frequent patterns) |

## Prerequisites

### 1. GitHub CLI (`gh`) on PATH
Install the [GitHub CLI](https://cli.github.com/) and ensure `gh` is accessible from your terminal/command prompt.

### 2. GitHub Authentication
Authenticate with GitHub:

```bash
gh auth login
```

The MCP server will automatically utilize this active session.

## Provided Tools

- **`gh_command`**: Runs an arbitrary `gh` subcommand. Arguments are passed as an array of string tokens in `args` (e.g. `["issue", "list", "--limit", "10"]`, `["pr", "view", "123"]`).
  - Values with spaces (titles, bodies, queries) must be passed as a **single array element** (e.g. `["pr", "create", "--title", "My Title"]`). No internal shell escaping or manual quoting is needed.

> ⚠️ **Warning (Arbitrary Command Execution)**: `gh_command` can execute **any** `gh` subcommand. Beyond read operations, it can perform destructive actions such as `gh pr merge`, `gh issue close`, `gh repo delete`, and arbitrary REST calls via `gh api` under the logged-in user's credentials. Ensure agent tasks are appropriately bounded and run within trusted environments.

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/github
```

> **Note**: When modifying Go sources (`src/main.go`), rebuild and commit the native binaries for all platforms using `./build.sh github` (or `./build.ps1 github` on Windows). `agy plugin install` copies committed binaries rather than compiling them on install.
