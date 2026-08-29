# go-lsp Plugin

An MCP server leveraging the official Go language server [gopls](https://pkg.go.dev/golang.org/x/tools/gopls) to provide definition jumps, reference lookups, and hover documentation for Go codebases.

## Prerequisites

The `gopls` binary must be installed on your `PATH`.

**Installation:**
```bash
go install golang.org/x/tools/gopls@latest
```

## Installation

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/go-lsp
```

## Provided Tools

* **`go_definition`**: Locates the symbol definition at the specified file position (1-indexed line and character).
* **`go_references`**: Finds all references to the symbol at the specified file position (1-indexed line and character).
* **`go_hover`**: Retrieves type information and documentation for the symbol at the specified file position (1-indexed line and character).

## Skill (`SKILL.md`)

Bundles `skills/go-lsp-gemini/SKILL.md` to instruct AI agents on when and how to query Go symbol definitions, references, and type signatures.
