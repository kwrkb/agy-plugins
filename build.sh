#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

cd "$SCRIPT_DIR/github"
go mod tidy
go build -o mcpServers/github-plugin main.go
echo "github-plugin built."

cd "$SCRIPT_DIR/gitlab"
go mod tidy
go build -o mcpServers/gitlab-plugin main.go
echo "gitlab-plugin built."
