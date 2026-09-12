#!/usr/bin/env sh
# govulncheck の結果に「引き上げ方針」を適用するゲート（CLAUDE.md「Go バージョンの引き上げ方針」）。
#
#   到達可能な指摘なし                                        → pass
#   到達可能な非 stdlib（依存モジュール）の指摘がある          → fail（stdlib 例外を適用しない）
#   到達可能な stdlib が固定中のマイナー内のパッチで直る       → fail（据え置く理由が無い）
#   到達可能な stdlib がマイナー跨ぎの引き上げを要する         → warning で許容
#
# 一律 warning にしていた頃は誰も読まず、固定版が3パッチ分の既知 CVE を抱えたまま
# 気づけなかった。だから同マイナー内で直るものだけは fail にする。
#
# 判定は `-format json` の構造化出力で行う。テキスト要約の grep は
#   - 件数1件で単数形になる
#   - 約80桁で折り返す
#   - 成分の並び順（module と stdlib）に依存する
#   - Symbol/Package/Module の全セクションの `Fixed in:` を拾ってしまう
# という4つの罠があり、いずれも実際にバグを生んだ（LESSONS 参照）。
#
# JSON の finding では
#   到達可能        = trace[0].function が存在する（テキストの "=== Symbol Results ===" と一致）
#   脆弱なモジュール = trace[0].module（stdlib なら "stdlib"）
#                      trace は「脆弱な関数が先頭・自コードのエントリポイントが末尾」の
#                      呼び出し経路。末尾を見ると自分のモジュール名を拾ってしまう。
#   修正版          = fixed_version（stdlib は "v1.26.6" 形式＝go 接頭辞は付かない）
#
# 使い方（モジュールディレクトリで実行する）:
#   .github/scripts/govulncheck-gate.sh <.go-version のパス> [判定対象の JSON ファイル]
#
# 第2引数を渡すと govulncheck を実行せず、その JSON を判定対象にする（テスト用）。
set -eu

VERSION_FILE="${1:?usage: govulncheck-gate.sh <go-version-file> [captured-json-file]}"
CAPTURED="${2-}"

MINOR=$(tr -d "[:space:]" < "$VERSION_FILE" | cut -d. -f1,2)

if [ -n "$CAPTURED" ]; then
	JSON=$(cat "$CAPTURED")
else
	# govulncheck 自体の失敗（ネットワーク断・引数誤り等）と「脆弱性を検出した」を
	# 区別する。-format json は脆弱性検出でも 0 を返すため、非 0 は道具の失敗。
	ERR=$(mktemp)
	if ! JSON=$("$(go env GOPATH)/bin/govulncheck" -format json ./... 2>"$ERR"); then
		echo "::error::govulncheck failed to run"
		cat "$ERR" >&2
		rm -f "$ERR"
		exit 1
	fi
	rm -f "$ERR"
fi

# 到達可能な finding を "<module>\t<fixed_version>" に落とす。出力は JSON の
# ストリーム（連結オブジェクト）なので -n '[inputs]' でまとめて読む。
REACHABLE=$(printf '%s' "$JSON" | jq -rn '
	[inputs]
	| map(select(has("finding")) | .finding)
	| map(select((.trace // []) | length > 0 and (.[0] | has("function"))))
	| map(((.trace[0].module) // "unknown") + "\t" + (.fixed_version // ""))
	| unique | .[]
')

[ -z "$REACHABLE" ] && exit 0

# ここから先は何かしら報告する。人が読める形をログに残す（判断には使わない）。
# 到達可能な指摘が無い大多数のケースで2回スキャンしないよう、ここまで来てから実行する。
if [ -z "$CAPTURED" ]; then
	"$(go env GOPATH)/bin/govulncheck" ./... 2>&1 || true
fi

# 依存モジュール側の到達可能な指摘は stdlib 例外の対象外。先に判定する。
MODULES=$(printf '%s\n' "$REACHABLE" | awk -F'\t' '$1 != "stdlib" { print $1 }' | sort -u)
if [ -n "$MODULES" ]; then
	echo "::error::Reachable vulnerabilities in modules you depend on. Update the dependency:"
	printf '%s\n' "$MODULES" | sed 's/^/    /'
	exit 1
fi

# 固定中のマイナー内のパッチで直る stdlib の指摘。stdlib の fixed_version は "v1.26.6"。
PATCHABLE=$(printf '%s\n' "$REACHABLE" | awk -F'\t' -v m="v$MINOR." 'index($2, m) == 1 { print $2 }' | sort -u)
if [ -n "$PATCHABLE" ]; then
	echo "::error::Reachable stdlib vulnerabilities are fixed within go${MINOR}. Bump .go-version to the latest go${MINOR}.x, run ./build.sh, and commit the binaries. Fixed in:"
	printf '%s\n' "$PATCHABLE" | sed 's/^/    /'
	exit 1
fi

echo "::warning::Tolerating reachable stdlib vulnerabilities that need a minor-version bump (see the Go upgrade policy in CLAUDE.md)"
exit 0
