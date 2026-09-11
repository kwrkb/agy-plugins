# settings-advisor — Non-intrusive Settings Advisor

An MCP server that analyzes codebase scale, languages, and security/CI/production configurations to **unobtrusively recommend** optimal agy settings (model selection, sandbox mode, and tool permissions). It never mutates configuration files automatically, but outputs actionable suggestions for `/model` and `/settings`.

## Provided Tools

### `settings_advisor`

Scans the specified repository path and recommends settings based on workspace size and planned tasks.

#### Parameters

- `path` (string, optional): Target repository path (default: `.`).
- `task_hint` (string, optional): Description of the upcoming task (e.g. `large refactoring`) to refine recommendations.
- `format` (string, optional): Output format (`text` or `json`). Defaults to `text`.

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/settings-advisor
```

## Recommendation Logic

- **Model Tier**: Classifies into `light` / `mid` / `heavy` based on LOC and language diversity across supported languages (Go, TS, JS, Python, Rust, Dart, Swift, Vue, Svelte, PHP, Scala, C/C++, C#, Java, Kotlin, Ruby, Zig, Lua, SQL, Shell, Terraform). Excludes build artifacts and package directories (`.next`, `.nuxt`, `out`, `target`, `.dart_tool`, `Pods`, `node_modules`, `dist`, `build`, etc.). Elevates to `heavy` when `task_hint` contains keywords like refactoring or architecture design. Dynamically matches `traits` in `models.json` (e.g. `instruction-following` for Claude Sonnet, `quota-independent` for GPT-OSS).
- **Sandbox**: Recommends `enableTerminalSandbox: true` upon detecting actual `.env` configurations (templates such as `.env.example` or `.env.sample` are safely ignored).
- **Tool Permissions**: Recommends `proceed-in-sandbox` when CI/CD configurations are present (GitHub Actions, GitLab CI, CircleCI, Bitbucket Pipelines, Azure Pipelines), and `strict` when production configuration files or directory paths (`prod` / `production` tokens) are detected.

## Bundled Skill

Includes `skills/settings-advisor-gemini/SKILL.md` to guide agents to recommend invoking this tool when users inquire about optimal models or security configurations.
