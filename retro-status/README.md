# retro-status — Retro RPG Status Profiler

A playful repository profiler that scans a codebase and formats its activity, scale, and health into a retro RPG character status screen (Level, Class, HP/MP, ATK/DEF, Equipment, Spells/Skills, and Inventory) rendered in Famicom-style ASCII art.

## Provided Tools

### `retro_status`

Scans the repository at the specified path and outputs an RPG status screen in ASCII art or JSON.

#### Parameters

- `path` (string, optional): Path to the repository to scan (default: `.`).
- `format` (string, optional): Output format (`text` for ASCII art or `json`). Defaults to `text`.

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/retro-status
```

## RPG Stat Mapping

- **Level (LV)**: Increases with total commits and Lines of Code (LOC).
- **Dungeon Depth (DEPTH)**: Based on total LOC (deeper dungeon for larger codebases).
- **Monsters (MONSTERS)**: Total number of remaining `TODO` comments.
- **Attack Power (ATK)**: Commit frequency in the last 30 days (development velocity).
- **Defense Power (DEF)**: Test LOC, linter presence, and CI/CD pipelines (bug resilience).
- **Magic Points (MP)**: Number of external package dependencies in `go.mod`, `package.json`, etc.
- **Equipment**:
  - Weapon: Dominant programming language (e.g. Go -> `Go Scalpel`, TS -> `TS Grimoire`).
  - Shield: Lockfile presence (`go.sum`, `package-lock.json`, etc.).
  - Armor: Linters and automated test suites.
  - Helm: CI/CD configurations (`.github/workflows`, `.gitlab-ci.yml`).
  - Accessory: Containerization (`Dockerfile`).
- **Spells & Skills**: Acquired through automated tests, deploy pipelines, and high commit bursts.
- **Inventory**: Detects developer CLI tools installed on the host machine (`rg`, `fd`, `jq`, `gh`, `docker`, `nvim`, etc.).
