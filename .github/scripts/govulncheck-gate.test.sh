#!/usr/bin/env sh
# govulncheck-gate.sh の判定テーブルを、govulncheck -format json と同じ構造の
# フィクスチャで検証する。到達可能性は trace[0].function の有無で表す。
set -eu

cd "$(dirname "$0")"
GATE=./govulncheck-gate.sh
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
FAILED=0

# finding <osv> <fixed_version> <脆弱なモジュール> <reachable|info>
# trace は govulncheck と同じ向き（脆弱な関数が先頭・自コードのエントリポイントが末尾）。
finding() {
	if [ "$4" = reachable ]; then
		trace='[{"module":"'$3'","function":"Vulnerable"},{"module":"example.com/app","function":"main"}]'
	else
		trace='[{"module":"'$3'","version":"v0.0.0"}]'
	fi
	printf '{"finding":{"osv":"%s","fixed_version":"%s","trace":%s}}\n' "$1" "$2" "$trace"
}

# run <期待終了コード> <期待出力パターン|-> <pinned version> <JSON 本文>
run() {
	want_code="$1"; want_pat="$2"
	printf '%s\n' "$3" > "$TMP/version"
	printf '%s' "$4" > "$TMP/out.json"

	got=$("$GATE" "$TMP/version" "$TMP/out.json" 2>&1) && code=0 || code=$?
	if [ "$code" -ne "$want_code" ]; then
		echo "FAIL [$CASE] exit=$code want=$want_code"; echo "  output: $got"; FAILED=1; return
	fi
	if [ "$want_pat" != "-" ] && ! printf '%s' "$got" | grep -q "$want_pat"; then
		echo "FAIL [$CASE] output missing '$want_pat'"; echo "  output: $got"; FAILED=1; return
	fi
	echo "ok   [$CASE]"
}

CASE="到達可能な指摘なし → pass"
run 0 "-" 1.26.8 "$(finding GO-1 v1.26.9 stdlib info)"

CASE="到達可能 stdlib・同マイナー内で修正 → fail"
run 1 "::error::" 1.26.5 "$(finding GO-1 v1.26.6 stdlib reachable)"

CASE="到達可能 stdlib が1件だけ・同マイナー内 → fail（単数形の罠を構造で回避）"
run 1 "Fixed in" 1.26.5 "$(finding GO-1 v1.26.6 stdlib reachable)"

CASE="到達可能 stdlib・マイナー跨ぎが必要 → warning で許容"
run 0 "::warning::" 1.26.8 "$(finding GO-1 v1.27.2 stdlib reachable)"

CASE="到達可能 stdlib はマイナー跨ぎ + 到達不能 stdlib が同マイナー内 → warning で許容"
run 0 "::warning::" 1.26.8 "$(finding GO-1 v1.27.2 stdlib reachable)$(finding GO-2 v1.26.9 stdlib info)"

CASE="混在: 到達可能な依存モジュール + 到達可能 stdlib はマイナー跨ぎ → fail"
run 1 "modules you depend on" 1.26.8 "$(finding GO-1 v1.27.2 stdlib reachable)$(finding GO-2 v1.2.3 golang.org/x/text reachable)"

CASE="到達可能な依存モジュールのみ → fail"
run 1 "modules you depend on" 1.26.8 "$(finding GO-1 v1.2.3 golang.org/x/text reachable)"

CASE="到達不能な依存モジュールのみ → pass"
run 0 "-" 1.26.8 "$(finding GO-1 v1.2.3 golang.org/x/text info)"

CASE="到達可能 stdlib が複数・一部が同マイナー内 → fail"
run 1 "::error::" 1.26.5 "$(finding GO-1 v1.27.2 stdlib reachable)$(finding GO-2 v1.26.6 stdlib reachable)"

CASE="CRLF の .go-version でもマイナーを取り違えない"
printf '1.26.5\r\n' > "$TMP/version"
printf '%s' "$(finding GO-1 v1.26.6 stdlib reachable)" > "$TMP/out.json"
if "$GATE" "$TMP/version" "$TMP/out.json" >/dev/null 2>&1; then
	echo "FAIL [$CASE] expected exit 1"; FAILED=1
else
	echo "ok   [$CASE]"
fi

CASE="finding が1件も無い（空出力）→ pass"
run 0 "-" 1.26.8 '{"config":{"protocol_version":"v1.0.0"}}'

[ "$FAILED" -eq 0 ] && echo "--- all gate cases passed" || echo "--- FAILURES"
exit "$FAILED"
