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

- **Model Tier**: Classifies into `light` / `mid` / `heavy` based on LOC and language diversity. Elevates to `heavy` when `task_hint` contains keywords like refactoring or architecture design. Prioritizes specialized models (e.g. Claude Sonnet for strict instruction following, GPT-OSS for fallback/quota diversity).
- **Sandbox**: Recommends `enableTerminalSandbox: true` upon detecting `.env` or secret configurations.
- **Tool Permissions**: Recommends `proceed-in-sandbox` when CI/CD workflows are present, and `strict` when production configuration files are detected.

## Bundled Skill

Includes `skills/settings-advisor-gemini/SKILL.md` to guide agents to recommend invoking this tool when users inquire about optimal models or security configurations.
