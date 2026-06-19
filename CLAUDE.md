# agy-plugins — プロジェクト固有ガイド

agy (Google Antigravity CLI) 向け MCP プラグイン集。グローバル CLAUDE.md のルールに加え、本リポジトリ固有の事実のみをここに記す。

## 構成（3プラグイン / 2 Go モジュール）

- `github/` — `gh` CLI を exec する自作 Go 製 MCP サーバー。module `github.com/kwrkb/agy-plugins/github`。バイナリ `github`(linux)/`github.exe`(windows) を **git にコミットして配布**。
- `gitlab/` — `glab mcp serve` を呼ぶ薄い設定のみ（`plugin.json` + `mcp_config.json`、ビルド物なし）。
- `agy-plugin-kit/` — プラグイン開発ヘルパー。`validator/`（Go 製・module `agy-plugin-validator`）＋ `skills/` `commands/` `templates/`。

## コマンド

```bash
# テスト・静的解析（モジュール別）
cd github && go vet ./... && go test ./...
cd agy-plugin-kit/validator && go vet ./... && go test ./...
# バイナリ再ビルド（go 1.26.4。Windows は ./build.ps1）
./build.sh                                    # 全プラグイン
./build.sh github                             # github だけ
./build.sh validator                          # validator だけ
```

**ソース変更時は必ず `./build.sh` で再ビルドしてコミット**（`agy plugin install` はビルドせず git 追跡バイナリをコピーするだけ）。決定論フラグは `build.sh` に集約され、Go 1.26.4 固定で bit-identical になる。CI の stale 検出ゲート（`.github/workflows/build-verify.yml`）がこれを前提にする。

## 実機検証（tmux + agy）

agy の対話セッションは PTY を要するため tmux 経由で起こす。クリーン install → ツール実行までを実環境で確認する。

```bash
# 1) クリーン install を再現（git 追跡ファイルのみ＝URL install と等価）
mkdir -p /tmp/ci && git archive HEAD github/ | tar -x -C /tmp/ci && \
  rm -rf ~/.gemini/config/plugins/github ~/.gemini/antigravity-cli/mcp/github && \
  agy plugin install /tmp/ci/github
# mcp_config.json の command が ${extensionPath} 解決済み絶対パスになっていること

# 2) tmux で agy を起こし、ツールを1回実行（スペース込み値で引数破損も同時に検証）
tmux new-session -d -s v -x 220 -y 50
tmux send-keys -t v 'agy -p "Use gh_command with args [\"search\",\"repos\",\"mark3labs mcp-go\",\"--limit\",\"1\"]" > /tmp/agy.txt 2>&1; echo DONE > /tmp/agy.done' Enter
# /tmp/agy.done を待ってから /tmp/agy.txt を確認

# 3) 起動成功の証拠は MCP キャッシュの「ツール名」で確認（mtime だけでは不十分）
ls ~/.gemini/antigravity-cli/mcp/github/   # 新サーバーなら gh_command.json のみ（旧 github-mcp-server の多数ツールが消える）
```

## 非自明な地雷（詳細は LESSONS.md の番号付き教訓）

- **バイナリは追跡コミット必須**: `agy plugin install` はビルドせず git 追跡ファイルをコピーするだけ。`main.go` を変えたら両 OS 分を再ビルド＆コミットしないと stale が配布される（#21）。
- **`${extensionPath}` 解決条件**: ソースに `gemini-extension.json` があり `plugin.json` が**無い**時のみ解決（#1）。同梱バイナリ参照プラグインは前者構成。
- **install は wipe しない**: 設計変更時は旧ファイルが残る。再 install 前に `~/.gemini/config/plugins/<name>/` を削除（#24）。
- **検証は MCP キャッシュのツール名で**: mtime 更新だけでなく中身（ツール名）で新サーバーを別人確認（#25）。
- **agy の `rules/` は非機能**（1.0.8／**1.0.9 でも再確認** #35）。プラグインからエージェントへ渡す知識は `skills/` で（#22）。
- **agy の hooks は 1.0.9 で部分的に機能化**: `PostToolUse` payload の `toolCall.args.TargetFile` に編集ファイル絶対パスが入り、2回目以降の編集・`agy -p` でも発火、自前バイナリは PWD 相対で呼べる（#34）。ただし payload は agy 独自スキーマ／`${extensionPath}` 未置換は不変。1.0.8 では全面非機能だった（#18-21）。

## ドキュメント地図

- `LESSONS.md` — 番号付き実装教訓（最重要・着手前に grep）
- `PLAN.md` — タスク進捗 / `implementation-notes.md` — 意思決定ログ / `README.md` — 利用者向け
