#!/bin/sh
# Wrapper for the official github/github-mcp-server binary.
#
# The official server only reads GITHUB_PERSONAL_ACCESS_TOKEN. This wrapper
# bridges from GITHUB_TOKEN / GH_TOKEN / `gh auth token` so that existing
# gh-CLI-based authentication keeps working without exporting a static PAT
# (preserves the behavior of the previous custom server).
if [ -z "$GITHUB_PERSONAL_ACCESS_TOKEN" ]; then
  GITHUB_PERSONAL_ACCESS_TOKEN="${GITHUB_TOKEN:-${GH_TOKEN:-$(gh auth token 2>/dev/null)}}"
  export GITHUB_PERSONAL_ACCESS_TOKEN
fi

exec "$(dirname "$0")/github-mcp-server" stdio "$@"
