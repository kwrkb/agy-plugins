#!/usr/bin/env sh
# govulncheck の結果に「引き上げ方針」を適用するゲート（CLAUDE.md「Go バージョンの引き上げ方針」）。
#
#   固定中のマイナー内のパッチで直る到達可能な stdlib 脆弱性 → fail（据え置く理由が無い）
#   マイナー跨ぎの引き上げが必要なもの                        → warning で許容
#   stdlib 以外の脆弱性                                       → govulncheck の終了コードをそのまま返す
#
# 一律 warning にしていた頃は誰も読まず、固定版が3パッチ分の既知 CVE を抱えたまま
# 気づけなかった。だから条件1だけは fail にする。
#
# 使い方（モジュールディレクトリで実行する）:
#   .github/scripts/govulncheck-gate.sh <.go-version のパス> [判定対象の出力ファイル]
#
# 第2引数を渡すと govulncheck を実行せず、そのファイルを判定対象にする（テスト用）。
set -eu

VERSION_FILE="${1:?usage: govulncheck-gate.sh <go-version-file> [captured-output-file]}"
CAPTURED="${2-}"

MINOR=$(tr -d "[:space:]" < "$VERSION_FILE" | cut -d. -f1,2)

STATUS=0
if [ -n "$CAPTURED" ]; then
	OUTPUT=$(cat "$CAPTURED")
	STATUS=1 # 取り込み済み出力は「脆弱性あり」の体で判定する
else
	OUTPUT=$("$(go env GOPATH)/bin/govulncheck" ./... 2>&1) || STATUS=$?
	echo "$OUTPUT"
fi

[ "$STATUS" -eq 0 ] && exit 0

# 到達可能な指摘の要約は golang.org/x/vuln の text handler が組み立てる1文で、
#   Your code is affected by <N> vulnerabilit{y,ies}[ from [<M> module{,s}][ and ]the Go standard library].
# という形になる（internal/scan/text.go summary()）。件数1件なら単数形になり、
# 約80桁で折り返すため、改行と連続空白を畳んでから判定する。
# 非到達の指摘を述べる別文は "in packages you import" / "in modules you require" を
# 使うので、"affected by" に錨を打てば到達可能分だけを見られる。
SUMMARY=$(printf '%s' "$OUTPUT" | tr '\n' ' ' | tr -s ' ')

# stdlib の据え置き例外は「到達可能な指摘が stdlib だけ」の時にしか適用してはならない。
# 第三者モジュール側の到達可能な指摘が混在していたら、stdlib 側がマイナー跨ぎでも
# 失敗させる（さもないと依存側の脆弱性を warning で隠してしまう）。
# 要約の並び順に依存せず、module 成分の有無を独立に見る。
if printf '%s' "$SUMMARY" | grep -qE "affected by [0-9]+ vulnerabilit(y|ies) from .*[0-9]+ module"; then
	exit "$STATUS"
fi

if ! printf '%s' "$SUMMARY" | grep -qE "affected by [0-9]+ vulnerabilit(y|ies) from the Go standard library"; then
	exit "$STATUS" # stdlib 以外の指摘はそのまま失敗させる
fi

if printf '%s\n' "$OUTPUT" | grep -qE "^ *Fixed in: .*@go${MINOR}\."; then
	echo "::error::Reachable stdlib vulnerabilities are fixed within go${MINOR}. Bump .go-version to the latest go${MINOR}.x, run ./build.sh, and commit the binaries."
	printf '%s\n' "$OUTPUT" | grep -E "^ *Fixed in: .*@go${MINOR}\." | sort -u
	exit 1
fi

echo "::warning::Tolerating stdlib vulnerabilities that need a minor-version bump (see the Go upgrade policy in CLAUDE.md)"
exit 0
