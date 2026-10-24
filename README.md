# agy-plugins

A collection of Model Context Protocol (MCP) plugins designed for AI assistants, specifically tailored for the agy CLI.

## Available Plugins

| Plugin | Description |
| :-- | :-- |
| **[github](./github/README.md)** | MCP server wrapping GitHub CLI (`gh`) for cross-platform issue/PR/repository operations |
| **[gitlab](./gitlab/README.md)** | MCP server wrapping GitLab CLI (`glab`) for cross-platform issue/MR/CI/project operations |
| **[agy-plugin-kit](./agy-plugin-kit/README.md)** | Meta-helper for agy plugin development (scaffolding, static validation, path fixups, doc generation) |
| **[ast-grep](./ast-grep/README.md)** | MCP server leveraging `ast-grep` (`sg`) for AST-based code search and safe refactoring |
| **[go-lsp](./go-lsp/README.md)** | MCP server utilizing `gopls` for Go definitions, references, and hover information |
| **[retro-status](./retro-status/README.md)** | MCP server scanning repositories to render RPG-style retro status ASCII art |
| **[settings-advisor](./settings-advisor/README.md)** | MCP server analyzing workspace scale, languages, and settings to recommend optimal models and sandbox configurations |
| **[worktree-manager](./worktree-manager/README.md)** | MCP server providing safe git worktree operations (list, add, remove, prune) for isolated parallel workflows |
| **[test-runner](./test-runner/README.md)** | MCP server running Go tests with bounded failure logs and targeted rerun arguments |

## Installation

```bash
# GitHub plugin (Cross-Platform)
agy plugin install https://github.com/kwrkb/agy-plugins/github

# GitLab plugin
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab

# agy plugin authoring toolkit
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit

# ast-grep plugin
agy plugin install https://github.com/kwrkb/agy-plugins/ast-grep

# go-lsp plugin
agy plugin install https://github.com/kwrkb/agy-plugins/go-lsp

# retro-status plugin
agy plugin install https://github.com/kwrkb/agy-plugins/retro-status

# settings-advisor plugin
agy plugin install https://github.com/kwrkb/agy-plugins/settings-advisor

# worktree-manager plugin
agy plugin install https://github.com/kwrkb/agy-plugins/worktree-manager

# test-runner plugin
agy plugin install https://github.com/kwrkb/agy-plugins/test-runner
```

Please refer to the README in each plugin directory for specific prerequisites (required CLI binaries on PATH / authentication).

## Bundled Skills

Each plugin includes an agent guide in `skills/<name>/SKILL.md` (loaded on demand upon tool invocation to instruct the model on argument formatting, repo conventions, and common patterns). While agy 1.4+ also supports injecting `rules/*.md` with a `trigger:` frontmatter, bundled skills remain the primary way to deliver detailed tool reference knowledge without consuming the shared always-on rule token budget (20,000 tokens).

## Requirements

| Plugin | Required CLI / Binary | Authentication |
| :-- | :-- | :-- |
| github | `gh` (on PATH) | Authenticated with `gh auth login` |
| gitlab | `glab` (on PATH; verified with v1.124.0) | Authenticated with `glab auth login` |
| agy-plugin-kit | (Optional) `go` (only needed for rebuilding validator; prebuilt binaries included) | None |
| ast-grep | `ast-grep` (CLI on PATH) | None |
| go-lsp | `gopls` (on PATH) | None |
| retro-status | `git` (recommended), `rg` (optional) | None |
| settings-advisor | None (prebuilt binaries included) | None |
| worktree-manager | `git` (on PATH) | None |
| test-runner | `go` >= 1.24 (on PATH) | None |

### Bundled Platforms & Self-Build

Existing plugins have been verified on native Windows. `test-runner` includes CI tests for Linux, macOS, and Windows; local verification is on Linux. The `gitlab` plugin (`glab` wrapper) has so far been verified on Linux only.

Go-based plugins (`github`, `gitlab`, `ast-grep`, `go-lsp`, `retro-status`, `settings-advisor`, `worktree-manager`, `test-runner`, and `agy-plugin-kit` validator) include native binaries for **`linux/amd64`**, **`darwin/arm64` (Apple Silicon)**, and **`windows/amd64`** in `bin/`. The extensionless `bin/<name>` dispatcher script detects the platform via `uname` and executes `<name>-<goos>-<goarch>` (on Windows, agy directly invokes `<name>.exe`).

**Other platforms (e.g. `linux/arm64` / `darwin/amd64`) are not bundled by default**, but can be built on the target machine without modifying any scripts:

```sh
cd <plugin>/src && CGO_ENABLED=0 go build -o "../bin/<name>-$(go env GOOS)-$(go env GOARCH)" .
# Example: building retro-status on ARM Linux -> retro-status/bin/retro-status-linux-arm64
```

## License & Attribution

Each plugin acts as a wrapper for existing tools/servers and complies with their respective licenses:

| Plugin | Wrapped Target | License |
| :-- | :-- | :-- |
| github | `gh` CLI | MIT |
| gitlab | [gitlab-org/cli (`glab`)](https://gitlab.com/gitlab-org/cli) | MIT |
| ast-grep | [`ast-grep` CLI](https://ast-grep.github.io/) | MIT |
| go-lsp | [`gopls` (Go Language Server)](https://pkg.go.dev/golang.org/x/tools/gopls) | BSD-3-Clause |
| retro-status | Custom Go implementation | MIT |
| settings-advisor | Custom Go implementation | MIT |
| agy-plugin-kit | Custom Go implementation | MIT |
| worktree-manager | `git` CLI | MIT |
| test-runner | `go test` (Go toolchain) | BSD-3-Clause |

Plugins delegate execution to user-installed binaries/CLIs on PATH (no redistribution of external binaries).
