# agy-plugin-kit — Plugin Authoring Toolkit for agy

A meta-plugin designed for plugin authors to scaffold, validate, repair, and document agy plugins quickly and correctly. Incorporates verified solutions for common pitfalls (such as Issue #390 where `${extensionPath}` is unresolved in native `plugin.json` formats).

## Components

| Component | Description |
| :-- | :-- |
| **`validator/`** (Go `src/` + `bin/`) | Deterministic validator detecting traps C1–C10. Includes `--fix-paths` mode and precompiled multi-platform binaries |
| **Command** `/agy-plugin-kit:new` | Scaffolds compliant plugins with automatic manifest format selection |
| **Command** `/agy-plugin-kit:validate` | Statically checks target plugin and provides structured summary |
| **Command** `/agy-plugin-kit:doctor` | Combines `agy plugin validate` + kit checks + automated path fixes (Issue #390 workaround) |
| **Command** `/agy-plugin-kit:doc` | Generates or updates README and SKILL.md from existing plugin structure |
| **Skill** `agy-plugin-authoring` | Core authoring guidelines distilled from empirical lessons |
| **`templates/`** | Templates replicated by `/new` (`plugin.json`, `gemini-extension.json`, Go wrappers, etc.) |

## Validator Checks

| # | Condition Checked | Severity |
| :-- | :-- | :-- |
| C1 | Dual manifest (`plugin.json` and `gemini-extension.json` both present) | WARN |
| C2 | Using `${extensionPath}` in native `plugin.json` format (**Issue #390**) | ERROR |
| C3 | Direct execution of `.sh`/`.cmd`/`.bat` in `command` (fails on Windows) | ERROR |
| C4 | Referenced binary ignored by `.gitignore` (missing in URL installs) | ERROR |
| C5 | Missing or invalid JSON in manifests | ERROR |
| C6 | Appending `.exe` extension to `${extensionPath}` commands | WARN |
| C7 | Invoking token-only MCP servers directly without wrapper (heuristic) | WARN |
| C8 | Using `${CLAUDE_PLUGIN_ROOT}` (Claude-only variable, invalid in agy) | ERROR |
| C9 | Go wrappers writing diagnostic text to stdout instead of stderr | WARN |
| C10 | Using `${extensionPath}` in native `plugin.json` hooks | WARN |

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit
```

## Automatic Validation Hook (`hooks.json`)

Includes `hooks.json` to automatically trigger validation on editing plugin manifests (`plugin.json`, `gemini-extension.json`, `mcp_config.json`, `hooks.json`), outputting non-blocking feedback to stderr.
