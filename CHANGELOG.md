# Changelog

## [0.6.0] - 2026-10-20

### Added
- **test-runner**: New MCP plugin (`go_test`) that runs `go test -json`, summarizes results, and returns
  failure logs plus `rerun` arguments for the failing tests. Includes bundled skill `test-runner-gemini`
  and native prebuilt binaries (`darwin-arm64`, `linux-amd64`, `windows`).
- **settings-advisor**: Broader scan detection and traits-driven model selection (`models.json` updated).
- **agy-plugin-kit/validator**: Checks updated for agy 1.4.2. C2 now flags `${PLUGIN_ROOT}` /
  `${PLUGIN_DATA}` in MCP `command` (including `gemini-extension.json` `mcpServers`), since agy 1.4.2
  leaves them unexpanded; manifests placed in a subdirectory are accepted. New check C11 warns when a
  `hooks.json` `timeout` looks like milliseconds (agy uses seconds).
- **agy-plugin-kit/validator**: New `--inject` mode (PreInvocation hook). `--hook` (PostToolUse) writes
  its findings to a per-conversation relay file and `--inject` delivers them to the model as an
  `ephemeralMessage`, because agy 1.4.2 drops PostToolUse hook output.

### Changed
- **mcp-go**: Migrated all seven MCP server modules from v0.54.1 to v1.3.0. No source changes; old and
  new binaries return byte-identical `initialize` / `tools/list` responses and matching `tools/call`
  results, and all seven servers were re-verified on agy 1.4.2.
- **Go toolchain**: The pinned patch version now lives in `.go-version`; `build.sh`, `build.ps1` and CI
  enforce it via `GOTOOLCHAIN`. Bumped to 1.26.9 (via 1.26.8) to pick up fixes for reachable stdlib
  vulnerabilities, and rebuilt all native binaries. `go.mod` directives are unchanged.
- **CI / Build Verification**: `govulncheck` now fails when a stdlib finding is fixable by a patch bump
  within the pinned minor; findings that need a minor bump remain warnings.
- **agy-plugin-kit**: `hooks.json` uses agy's named hook format with `timeout` in seconds (`10`; the
  previous `10000` meant about 2.8 hours).

### Fixed
- **test-runner**: Invalid `packages` (resolved via `go list` discovery) and malformed `-run`
  expressions are rejected up front as input errors instead of surfacing as package failures; the
  `-run` expression is capped at 64 KiB.
- **settings-advisor**: Path-based CI / production-config detection is scoped to the scan root, the
  scan root itself is no longer skipped when named like a build directory, and qualified `.env`
  templates are excluded.
- **agy-plugin-kit**: Hook `command`s chain the Windows form and the Unix dispatcher with `||`, so the
  bundled validator starts on Windows (`cmd /c` previously read `/bin` as a switch). Verified on Linux,
  macOS and Windows.

## [0.5.0] - 2026-09-10

### Added
- **worktree-manager**: New MCP plugin for safe Git worktree management (`worktree_list`, `worktree_add`, `worktree_remove`, `worktree_prune`). Refuses removal of the main working tree and includes bundled skill `worktree-manager-gemini`.
- Native prebuilt binaries (`darwin-arm64`, `linux-amd64`, `windows`) for `worktree-manager`.

### Fixed
- **CI / Build Verification**: Configured `continue-on-error: true` for `govulncheck` to prevent standard library CVE reports for pinned Go toolchain (1.26.5) from blocking deterministic rebuild and binary divergence gates.
- Updated `agy-plugin-kit/validator` and `worktree-manager` binaries to match Go 1.26.5 deterministic rebuilds.

## [0.4.0] - 2026-06-20

### Added
- **macOS (Apple Silicon / arm64) support** for all Go binary plugins (`github`, `ast-grep`,
  `retro-status`, `agy-plugin-kit/validator`). Each plugin now ships a native `darwin-arm64` binary
  alongside `linux-amd64` and `windows`.
- **OS-dispatch shebang script** at `bin/<name>`: a POSIX `sh` dispatcher (OS-independent text) that
  `exec`s the matching native binary by `uname`. Since a single fixed `command` string cannot switch
  by OS (agy only resolves `${extensionPath}`), this folds all platforms into one install target.
  Verified end-to-end on macOS arm64: MCP startup (V1), hook firing (V2), and `+x` preserved through
  `agy plugin install` (V4). See LESSONS #40.

### Changed
- **Repository layout**: every Go plugin is split into `src/` (sources: `main.go`, `go.mod`[, `go.sum`])
  and `bin/` (distribution: native binaries + dispatcher). Manifest `command` is now
  `${extensionPath}${/}bin${/}<name>`; the validator hook is `validator/bin/validator --hook`.
- **`build.sh` / `build.ps1`** build from `<plugin>/src/` and emit `<plugin>/bin/<name>-<os>-<arch>`
  for three targets (linux-amd64 / darwin-arm64 / windows); the dispatcher is hand-written text, not a
  build output.
- **CI (`build-verify.yml`)** runs vet/test/govulncheck in `<plugin>/src` and diffs the three native
  binaries under `<plugin>/bin/`.
- **agy-plugin-kit** templates/skill/commands updated to the `src/`+`bin/`+dispatcher convention;
  retracted the previous "Linux-only / macOS unsupported" notes.

## [0.3.0] - 2026-06-14

### Changed
- **gitlab**: Migrated from `gemini-extension.json` to native Antigravity CLI format:
  `plugin.json` (name / description) + `mcp_config.json` (mcpServers). Since `glab` is resolved
  from PATH, no `${extensionPath}` substitution is needed and the new format works directly.
- **github**: Keeps `gemini-extension.json` in source (required for `${extensionPath}` resolution at
  install time — agy resolves the variable only when reading `gemini-extension.json` from source, not
  from a pre-existing `mcp_config.json`). The installed directory continues to have `plugin.json` +
  `mcp_config.json` (absolute path) generated by `agy plugin install`.

### Removed
- `gitlab/gemini-extension.json` (superseded by `plugin.json` + `mcp_config.json`).

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
