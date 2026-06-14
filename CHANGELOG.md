# Changelog

## [gitlab 0.2.0] - 2026-06-14

### Changed
- GitLab plugin now uses the **official** GitLab CLI's built-in MCP server (`glab mcp serve`, EXPERIMENTAL) instead of the in-house Go server. Exposes issues / merge requests / projects and adds CI/CD pipelines & jobs.
- `gemini-extension.json` points `command: "glab"`, `args: ["mcp", "serve"]` at the system `glab` on PATH (requires glab >= v1.74.0; apt builds are too old — install via `go install gitlab.com/gitlab-org/cli/cmd/glab@latest`).
- No wrapper needed: `glab mcp serve` reuses glab's own auth config (`~/.config/glab-cli/config.yml`), unlike github-mcp-server.
- `build.sh` no longer builds a gitlab binary.

### Removed
- Custom GitLab MCP server source: `gitlab/main.go`, `gitlab/go.mod`, `gitlab/go.sum` (replaced by the official `glab mcp serve`).

## [github 0.2.0] - 2026-06-14

### Changed
- GitHub plugin now wraps the **official** [github/github-mcp-server](https://github.com/github/github-mcp-server) (v1.3.0) instead of the in-house Go server. Exposes the full official toolset (issues / pull_requests / repos / actions / code_security / discussions / ...) — 43 tools by default.
- `build.sh` installs the official binary via `go install ...@v1.3.0` into `github/mcpServers/` (version pinned via `GITHUB_MCP_VERSION`).

### Added
- `github/github-mcp-wrapper.sh`: resolves a token (`GITHUB_PERSONAL_ACCESS_TOKEN` → `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token`) and exports `GITHUB_PERSONAL_ACCESS_TOKEN` before exec'ing the official binary, since the official server only reads that one variable. Preserves the previous gh-CLI-based auth behavior.

### Removed
- Custom GitHub MCP server source: `github/main.go`, `github/go.mod`, `github/go.sum` (replaced by the official binary).

## [0.1.0] - 2026-06-14

### Added
- GitHub MCP plugin: `get_repo_info`, `list_issues`, `create_issue`, `get_issue`, `create_issue_comment`, `list_prs`, `create_pr`, `get_pr`, `merge_pr`
- GitLab MCP plugin: `get_project_info`, `list_issues`, `create_issue`, `get_issue`, `create_issue_comment`, `list_mrs`, `create_mr`, `get_mr`, `merge_mr`
- Authentication auto-discovery: env vars → CLI (`gh`/`glab`) fallback for both plugins

### Changed
- Replaced `plugin.json` + `mcp_config.json` with `gemini-extension.json` using `${extensionPath}` variable substitution — no manual path editing required after `agy plugin install`
- `build.sh` now uses script-relative paths instead of hardcoded absolute paths
- Module paths updated to `github.com/kwrkb/agy-plugins/{github,gitlab}`

### Removed
- `commit_and_push` tool from both plugins — git CLI shell-out does not belong in an API-wrapper MCP server and was the source of reported failures; the agy host agent has direct git access
