# ast-grep Plugin

An MCP server leveraging [ast-grep (`sg`)](https://ast-grep.github.io/) to enable accurate, AST-based code search and refactoring directly from AI assistants.

## Prerequisites

The `ast-grep` executable must be installed on your `PATH` (on Linux, use the full name `ast-grep` rather than the `sg` alias to avoid collision with `setgroups`).

**Installation:**
```bash
brew install ast-grep
# or
npm install -g @ast-grep/cli
```

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/ast-grep
```

## Provided Tools

* **`ast_search`**: Performs AST pattern matching on files in the target directory and returns results in JSON.
* **`ast_replace`**: Performs AST pattern matching and in-place rewriting on files in the target directory.

## Skill (`SKILL.md`)

Bundles `skills/ast-grep/SKILL.md` to guide agents on `ast-grep` meta-variables (`$A`, `$$$ARGS`) and encourage a search-before-replace workflow.
