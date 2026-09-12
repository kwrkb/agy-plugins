#!/usr/bin/env sh
# govulncheck-gate.sh の判定テーブルを固定した出力で検証する。
# 単数形（件数1件）と行折り返しは govulncheck の実出力で確認済みの挙動なので、
# 実際に再現できる Go 版が無いケースもフィクスチャで押さえる。
set -eu

cd "$(dirname "$0")"
GATE=./govulncheck-gate.sh
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
FAILED=0

# run <期待終了コード> <期待出力の grep パターン|-> <pinned version> <出力本文>
run() {
	want_code="$1"
	want_pat="$2"
	printf '%s\n' "$3" > "$TMP/version"
	printf '%s\n' "$4" > "$TMP/out"

	got=$("$GATE" "$TMP/version" "$TMP/out" 2>&1) && code=0 || code=$?
	if [ "$code" -ne "$want_code" ]; then
		echo "FAIL [$CASE] exit=$code want=$want_code"
		echo "  output: $got"
		FAILED=1
		return
	fi
	if [ "$want_pat" != "-" ] && ! printf '%s' "$got" | grep -q "$want_pat"; then
		echo "FAIL [$CASE] output missing '$want_pat'"
		echo "  output: $got"
		FAILED=1
		return
	fi
	echo "ok   [$CASE]"
}

CASE="複数件・同マイナー内で修正 → fail"
run 1 "::error::" 1.26.5 "Your code is affected by 3 vulnerabilities from the Go standard library.
    Fixed in: net/url@go1.26.6
    Fixed in: crypto/tls@go1.26.6"

CASE="単数形・同マイナー内で修正 → fail"
run 1 "::error::" 1.26.5 "Your code is affected by 1 vulnerability from the Go standard library.
    Fixed in: net/url@go1.26.6"

CASE="単数形・マイナー跨ぎが必要 → warning で許容"
run 0 "::warning::" 1.26.8 "Your code is affected by 1 vulnerability from the Go standard library.
    Fixed in: net/http@go1.27.2"

CASE="複数件・マイナー跨ぎが必要 → warning で許容"
run 0 "::warning::" 1.26.8 "Your code is affected by 2 vulnerabilities from the Go standard library.
    Fixed in: net/http@go1.27.2
    Fixed in: os/exec@go1.27.2"

CASE="要約行が折り返されていても拾う（複数形・折り返しのみを切り分け）"
run 1 "::error::" 1.26.5 "Your code is affected by 3
vulnerabilities from the Go standard library.
    Fixed in: net/url@go1.26.6"

CASE="混在: 第三者モジュール到達可能 + stdlib はマイナー跨ぎ → 許容せず fail"
run 1 "-" 1.26.8 "Vulnerability #1: GO-2026-9999
    Fixed in: github.com/example/lib@v1.0.1
Vulnerability #2: GO-2026-8888
  Standard library
    Fixed in: net/http@go1.27.2

Your code is affected by 2 vulnerabilities from 1 module and the Go standard library."

CASE="混在: stdlib 成分が先に並んでも module 成分があれば fail（並び順に依存しない）"
run 1 "-" 1.26.8 "Your code is affected by 2 vulnerabilities from the Go standard library and 1 module.
    Fixed in: net/http@go1.27.2
    Fixed in: github.com/example/lib@v1.0.1"

CASE="混在: 複数モジュール + stdlib → fail"
run 1 "-" 1.26.8 "Your code is affected by 3 vulnerabilities from 2 modules and the Go standard library.
    Fixed in: net/http@go1.27.2"

CASE="stdlib 以外のみ → govulncheck の失敗をそのまま返す"
run 1 "-" 1.26.8 "Your code is affected by 1 vulnerability from a module you require.
    Fixed in: github.com/example/lib@v1.2.3"

CASE="CRLF の .go-version でもマイナーを取り違えない"
printf '1.26.5\r\n' > "$TMP/version"
printf '%s\n' "Your code is affected by 1 vulnerability from the Go standard library.
    Fixed in: net/url@go1.26.6" > "$TMP/out"
if "$GATE" "$TMP/version" "$TMP/out" >/dev/null 2>&1; then
	echo "FAIL [$CASE] expected exit 1"
	FAILED=1
else
	echo "ok   [$CASE]"
fi

[ "$FAILED" -eq 0 ] && echo "--- all gate cases passed" || echo "--- FAILURES"
exit "$FAILED"
