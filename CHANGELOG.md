# Changelog

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
