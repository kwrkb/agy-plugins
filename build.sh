#!/bin/bash
set -e
cd /home/yugosasaki/code/agy-plugins/github
go mod tidy
go build -o mcpServers/github-plugin main.go
echo "github-plugin built."

cd /home/yugosasaki/code/agy-plugins/gitlab
go mod tidy
go build -o mcpServers/gitlab-plugin main.go
echo "gitlab-plugin built."
