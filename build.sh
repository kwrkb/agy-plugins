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

cd "$SCRIPT_DIR/gitlab"
go mod tidy
go build -o mcpServers/gitlab-plugin main.go
echo "gitlab-plugin built."
