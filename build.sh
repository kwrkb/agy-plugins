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
# 注意: 決定論ビルドは Go ツールチェーンのパッチ版まで一致が前提。版は .go-version が
#       唯一の定義箇所で、CI（setup-go の go-version-file）もこのファイルを読む。
#       バージョンを上げる時は .go-version を書き換え、全バイナリを再ビルドしてコミットすること。
# Windows ネイティブで実行する場合は WSL または git-bash を使う。
set -eu

# スクリプトの位置をリポジトリルートとして扱う（呼び出し元 CWD に依存しない）
cd "$(dirname "$0")"

# 決定論フラグ（CI ゲートと一致させる唯一の定義箇所）
FLAGS="-trimpath -buildvcs=false -ldflags=-buildid="

# Go ツールチェーンを .go-version に固定する。GOTOOLCHAIN の既定は auto で、
# go.mod の `go` ディレクティブは下限でしかないため、ローカルに新しい Go があると
# 黙ってそちらが使われ、正常な再ビルド出力と見分けのつかない差分が出る（stale ゲートが落ちる）。
# 指定版が無ければ Go が自動ダウンロードするので、開発者側の事前準備は不要。
# Git Bash + core.autocrlf=true では .go-version が CRLF で checkout されうる。
# CR が残ると go は `invalid GOTOOLCHAIN "go1.26.5\r"` で即死するため空白類を除去する
# （.gitattributes で LF 固定もしているが、既存クローンを救うため読み取り側でも守る）。
GOTOOLCHAIN="go$(tr -d "[:space:]" < .go-version)"
export GOTOOLCHAIN

# build <plugin-dir> <output-basename>
# <plugin-dir>/src/ のソースから、ネイティブバイナリを <plugin-dir>/bin/ に生成する。
#   <base>-linux-amd64   <base>-darwin-arm64   <base>.exe
# 拡張子なしの <base>（OS 分岐 dispatcher）は build.sh では触らない（git 追跡の
# テキストスクリプト。write_dispatcher() 参照）。Windows の agy は <base>.exe を
# 直接起動するため dispatcher を経由しない。
build() {
	dir="$1"
	base="$2"
	echo "==> building $base (linux-amd64, darwin-arm64, windows) from $dir/src/  [$(go version | awk '{print $3}')]"
	(
		cd "$dir/src"
		CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $FLAGS -o "../bin/$base-linux-amd64"  .
		CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $FLAGS -o "../bin/$base-darwin-arm64" .
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $FLAGS -o "../bin/$base.exe"          .
	)
}

target="${1:-all}"
case "$target" in
	github)       build github github ;;
	validator)    build agy-plugin-kit/validator validator ;;
	ast-grep)     build ast-grep ast-grep ;;
	retro-status) build retro-status retro-status ;;
	settings-advisor) build settings-advisor settings-advisor ;;
	go-lsp)       build go-lsp go-lsp ;;
	worktree-manager) build worktree-manager worktree-manager ;;
	all)
		build github github
		build agy-plugin-kit/validator validator
		build ast-grep ast-grep
		build retro-status retro-status
		build settings-advisor settings-advisor
		build go-lsp go-lsp
		build worktree-manager worktree-manager
		;;
	*)
		echo "unknown target: $target (expected: github | validator | ast-grep | retro-status | settings-advisor | go-lsp | worktree-manager | all)" >&2
		exit 2
		;;
esac
