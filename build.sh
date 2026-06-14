#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# github: install the official github/github-mcp-server binary + auth wrapper.
GITHUB_MCP_VERSION="v1.3.0"
rm -rf "$SCRIPT_DIR/github/mcpServers"
mkdir -p "$SCRIPT_DIR/github/mcpServers"
GOBIN="$SCRIPT_DIR/github/mcpServers" go install "github.com/github/github-mcp-server/cmd/github-mcp-server@$GITHUB_MCP_VERSION"
cp "$SCRIPT_DIR/github/github-mcp-wrapper.sh" "$SCRIPT_DIR/github/mcpServers/github-mcp-wrapper.sh"
chmod +x "$SCRIPT_DIR/github/mcpServers/github-mcp-wrapper.sh"
echo "official github-mcp-server ($GITHUB_MCP_VERSION) installed."

# gitlab: uses the system-installed `glab` CLI (`glab mcp serve`); no build step.
# Requires glab >= v1.74.0 on PATH (apt build is too old; install via
# `go install gitlab.com/gitlab-org/cli/cmd/glab@latest`).
echo "gitlab: uses system glab (glab mcp serve); nothing to build."
