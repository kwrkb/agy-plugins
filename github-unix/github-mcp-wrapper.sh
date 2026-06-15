#!/bin/sh
# github-mcp-server は GITHUB_PERSONAL_ACCESS_TOKEN のみを読む。
# env に無ければ GITHUB_TOKEN / GH_TOKEN / gh auth token の順で解決する。
# 端末から起動した agy は PATH と gh の認証を継承するため、手動の export なしで動く。
TOKEN="${GITHUB_PERSONAL_ACCESS_TOKEN:-${GITHUB_TOKEN:-${GH_TOKEN:-}}}"
if [ -z "$TOKEN" ]; then
	TOKEN="$(gh auth token 2>/dev/null)"
fi
export GITHUB_PERSONAL_ACCESS_TOKEN="$TOKEN"

# github-mcp-server は PATH 上のユーザー導入バイナリを使う（同梱・再配布なし）。
exec github-mcp-server stdio "$@"
