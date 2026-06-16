#!/usr/bin/env sh
# 全プラグインのバイナリを決定論ビルドする単一ソース。
# 開発者の再ビルドと CI の検証ゲート（.github/workflows/build-verify.yml）が
# 同じフラグを使うよう、ビルド設定はこのスクリプトに集約する。
#
# 使い方:
#   ./build.sh            # 全モジュール（= all）
#   ./build.sh github     # github プラグインのみ
#   ./build.sh validator  # agy-plugin-kit の validator のみ
#
# 注意: 決定論ビルドは Go ツールチェーンのバージョン一致が前提（現状 go 1.26.4）。
#       バージョンを上げる時は全バイナリを再ビルドしてコミットすること。
# Windows ネイティブで実行する場合は WSL または git-bash を使う。
set -eu

# 決定論フラグ（CI ゲートと一致させる唯一の定義箇所）
FLAGS="-trimpath -buildvcs=false -ldflags=-buildid="

# build <module-dir> <output-basename>
# <dir> 内で linux/windows amd64 のバイナリ（<base> と <base>.exe）を生成する。
build() {
	dir="$1"
	base="$2"
	echo "==> building $base (linux, windows) in $dir/  [$(go version | awk '{print $3}')]"
	(
		cd "$dir"
		CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $FLAGS -o "$base"     .
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $FLAGS -o "$base.exe" .
	)
}

# スクリプトの位置をリポジトリルートとして扱う（呼び出し元 CWD に依存しない）
cd "$(dirname "$0")"

target="${1:-all}"
case "$target" in
	github)    build github github ;;
	validator) build agy-plugin-kit/validator validator ;;
	ast-grep)  build ast-grep ast-grep ;;
	all)
		build github github
		build agy-plugin-kit/validator validator
		build ast-grep ast-grep
		;;
	*)
		echo "unknown target: $target (expected: github | validator | ast-grep | all)" >&2
		exit 2
		;;
esac
