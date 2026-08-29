# GitLab MCP Plugin (`glab mcp serve`)

This plugin enables Antigravity (`agy`) to interface with GitLab using the built-in **`glab mcp serve`** feature of the official GitLab CLI [`glab`](https://gitlab.com/gitlab-org/cli).

> **Note**: `glab mcp serve` is marked as **EXPERIMENTAL** by GitLab and its specifications may change in future releases.

## Features (MCP Tools)

`glab mcp serve` exposes subcommands of `glab` as MCP tools (around 190 tools in `glab_<command>_<subcommand>` format). Major categories include:

- **Issues** (`glab_issue_*`): list, create, update, close, add notes, view, etc.
- **Merge Requests** (`glab_mr_*`): list, create, update, merge, diff, approve, notes, etc.
- **Projects / Repo** (`glab_repo_*`): list, view, create, clone, search, etc.
- **CI/CD** (`glab_ci_*`, `glab_job_*`): pipeline execution, status, trace, job artifacts, etc.
- Others: releases, labels, milestones, variables, schedules, and more.

## Bundled Skill

`skills/gitlab/SKILL.md` provides an operational guide for AI agents (loaded upon invocation). It details invocation argument formats (`args`, `flags`, `limit`, `offset`), project specification rules (`flags.repo`), and common usage patterns.

## Prerequisites

- **`glab` >= v1.74.0** with `mcp serve` support (v1.102.0+ recommended)
  - Installation via Go:
    ```bash
    go install gitlab.com/gitlab-org/cli/cmd/glab@latest
    ```
  - Verification:
    ```bash
    glab mcp serve --help
    ```

## Setup & Authentication

Authentication reuses existing `glab` CLI settings (`~/.config/glab-cli/config.yml`):

```bash
glab auth login        # Authenticate if not logged in
glab auth status       # Check authentication status
```

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

## License & Attribution

This plugin is a configuration wrapper calling `glab mcp serve` from [gitlab-org/cli (`glab`)](https://gitlab.com/gitlab-org/cli) (MIT License). It relies on the user's local `glab` binary without redistribution.
