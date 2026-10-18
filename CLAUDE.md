# agy-plugins — プロジェクト固有ガイド

agy (Google Antigravity CLI) 向け MCP プラグイン集。グローバル CLAUDE.md のルールに加え、本リポジトリ固有の事実のみをここに記す。

## 構成（Go 製 7 プラグイン + gitlab / 8 Go モジュール）

Go プラグインは **`src/`（ソース）＋ `bin/`（配布物）** に分離。`bin/` に各 OS のネイティブ
`<name>-linux-amd64` / `<name>-darwin-arm64` / `<name>.exe` と、拡張子なしの **OS 分岐 dispatcher**
`bin/<name>`（shebang sh・`uname` で実機ネイティブを `exec`、+x コミット）を **git にコミットして配布**。
`command` は `${extensionPath}${/}bin${/}<name>`（Windows は agy が `.exe` を補完し `bin/<name>.exe` を直接起動＝dispatcher 非経由）。

- `github/` — `gh` CLI を exec する自作 Go 製 MCP サーバー。module `github.com/kwrkb/agy-plugins/github`（`github/src/`）。
- `ast-grep/` — `ast-grep` CLI を exec。`retro-status/` — リポジトリ解析。`settings-advisor/` — Gemini の settings 助言。`go-lsp/` — `gopls` 経由の Go LSP（definition/references/hover）。`worktree-manager/` — Git worktree 管理。`test-runner/` — Go テスト実行と失敗ログ・再実行引数の返却。いずれも Go 製 MCP サーバーで src/bin 構成、module パスは `github.com/kwrkb/agy-plugins/<name>`。
- `gitlab/` — `glab mcp serve` を呼ぶ薄い設定のみ（`plugin.json` + `mcp_config.json`、Go バイナリ無し＝src/bin 非対象）。
- `agy-plugin-kit/` — プラグイン開発ヘルパー。`validator/`（Go 製・module `agy-plugin-validator`・`validator/src/`＋`validator/bin/`）＋ `skills/` `commands/` `templates/`。hook は `validator/bin/validator --hook`。

## コマンド

```bash
# テスト・静的解析（モジュール別。ソースは <plugin>/src/ 配下）
cd github/src && go vet ./... && go test ./...
cd agy-plugin-kit/validator/src && go vet ./... && go test ./...
# 他プラグイン（ast-grep / retro-status / settings-advisor / go-lsp / worktree-manager / test-runner）も同じ流儀（<name>/src で go vet ./... && go test ./...）
# バイナリ再ビルド（Go 版は .go-version に固定。build.sh が GOTOOLCHAIN で強制するため事前準備不要。Windows は ./build.ps1）
./build.sh                                    # 全プラグイン
./build.sh github                             # github だけ
./build.sh validator                          # validator だけ
# 他ターゲット: ast-grep | retro-status | settings-advisor | go-lsp | worktree-manager | test-runner
```

**ソース変更時は必ず `./build.sh` で再ビルドしてコミット**（`agy plugin install` はビルドせず git 追跡バイナリをコピーするだけ）。決定論フラグは `build.sh` に、Go のパッチ版は `.go-version` に集約され（`build.sh`/`build.ps1`/CI が同じファイルを読む）、bit-identical になる。CI の stale 検出ゲート（`.github/workflows/build-verify.yml`）がこれを前提にする。

## Go バージョンの引き上げ

固定版は `.go-version` の 1 か所。**上げてよい条件は VISION.md「Go バージョンの引き上げ方針」**（「新しい版が出たから」では上げない）。
手順は `.go-version` を書き換えて `./build.sh`（全プラグイン）→ 再ビルドされた全バイナリをコミット。`go.mod` の `go` ディレクティブは動かさない。

## 実機検証（tmux + agy）

agy の対話セッションは PTY を要するため tmux 経由で起こす。クリーン install → ツール実行までを実環境で確認する。以下は **agy 1.2.7 で確立した手順**で、1.4.2 では install・`agy plugin validate`・MCP 起動まで再確認済み（tmux 経由のツール実行の通しは 1.4.2 で未再実施。注意点は末尾の注記）。

```bash
# 1) クリーン install を再現（git 追跡ファイルのみ＝URL install と等価）
D=$(mktemp -d) && git archive HEAD test-runner/ | tar -x -C "$D" && \
  rm -rf ~/.gemini/config/plugins/test-runner \
         ~/.gemini/antigravity-cli/mcp/test-runner_test-runner && \
  agy plugin install "$D/test-runner"
# mcp_config.json の command が ${extensionPath} 解決済み絶対パスになっていること
# 配布物は mtime でなく中身で確認する（目視で突き合わせず、file ごとに判定させる）
for f in test-runner/bin/*; do
  cmp -s "$f" ~/.gemini/config/plugins/test-runner/bin/"$(basename "$f")" \
    && echo "OK $(basename "$f")" || echo "DIFFER $(basename "$f")"
done
# 中身が同じでも install が実行ビットを落とすことがあり、内容比較では検出できない。
# 起動するのは実行ホストの分だけなので、他 OS 向けも含めて明示的に判定させる
for f in ~/.gemini/config/plugins/test-runner/bin/*; do
  [ -x "$f" ] && echo "+x $(basename "$f")" || echo "NOT-EXECUTABLE $(basename "$f")"
done

# 2) tmux で agy を「対話モードで」起こす（`agy -p` は下記のとおり使えない）
tmux new-session -d -s v -x 220 -y 50 -c <検証用モジュールのディレクトリ>
tmux send-keys -t v 'agy' Enter
#   初回のみ「Do you trust this folder?」→ Enter で承認
#   本文と Enter は別コール（TUI が本文を受け取ってから改行が届く。1 コールでも
#   クォート自体は壊れないことは確認済み）
tmux send-keys -t v 'Use the go_test tool with module_path "<絶対パス>" and packages ["./..."]'
tmux send-keys -t v Enter
#   「Allow calling this tool?」→ 1（Yes, allow tool call）。3 と 6 は settings.json へ
#   永続化するので検証では選ばない。1 は毎回聞かれる＝何も残らない
tmux capture-pane -t v -p -S -80   # 画面外に出るのでスクロールバックごと取る

# 3) 起動成功の証拠は MCP キャッシュの「ツール名」で確認（mtime だけでは不十分）
ls ~/.gemini/antigravity-cli/mcp/test-runner_test-runner/   # 新サーバーなら go_test.json のみ
```

- **キャッシュのディレクトリ名は `<plugin>_<server>`**（例 `test-runner_test-runner`）。`mcp/<plugin>` を消しても**何も消えず**、残った古いキャッシュを新サーバーと誤認する。
- **`agy -p`（headless）では MCP ツールが自動拒否される**（`a tool required the "mcp" permission that headless mode cannot prompt for`）。ツール実行の検証は対話モードで行う。サーバー起動とツール探索だけなら headless でもキャッシュに現れる。
- **引数のバイト列を問う検証は tmux 経由でやらない**。非印字文字や数十 KiB の値はシェル・プロンプト・モデルのどこかで壊れ、「通った」が「そもそも届いていない」を意味しうる。`bin/<name>` へ JSON-RPC を直接流す（`initialize` → `notifications/initialized` → `tools/call`）。
- **相対パスはサーバーの cwd 基準**（`mcp_config.json` は `"cwd": ""`）。ワークスペース相対ではないので、引数のパスは絶対で渡す。

## 非自明な地雷

現在有効な設計判断（マニフェスト形式・dispatcher・バイナリのコミット配布・skills 主体・フックの書き方と受け渡し）は **VISION.md** が正典。
ここには作業中に踏みやすい手順上の注意だけを置く。根拠は LESSONS.md（`#N` は旧形式の番号付き教訓）。

**実機の agy は 1.4.2**（2026-10-18 に install・MCP 起動・`rules/`・`agents/`・hooks・`PLUGIN_DATA`・`bin/` PATH を再検証）。

- **ソースを変えたら全 OS 分のバイナリを再ビルドしてコミット**。忘れると stale が配布される（#21。CI の stale ゲートでも落ちる）。
- **install は wipe しない**: 設計変更時は旧ファイルが残る。再 install 前に `~/.gemini/config/plugins/<name>/` を削除（#24）。
  install 先の名前はマニフェストの `name` で決まるので、削除はその名前で行う。
- **ソース検査は `validator <src>`、install 後の検査は `agy plugin validate`**（後者は `plugin.json` を直接要求する）。
- **起動の証拠は MCP キャッシュのツール名**: mtime だけでなく中身で新サーバーを確認する（#25）。場所は `mcp/<plugin>_<server>/`。
- **フックの差し込みの確認**: `ephemeralMessage` は直後の 1 回のモデル呼び出しにしか見えない。検証プロンプトは「書いた直後に報告」させ、
  届いたかは transcript の `EPHEMERAL_MESSAGE` で判定する（LESSONS 2026-10-18）。

## ドキュメント地図

矛盾時の優先順は **VISION.md > PLAN.md > コード > NOTES.md**（`LESSONS.md` は序列外＝仕様ではなく記録）。

- `VISION.md` — 正典（目的・スコープ・非スコープ・現在有効な設計判断）。更新はユーザー承認が必要
- `PLAN.md` — フェーズ境界と現在地だけ
- `NOTES.md` — 内部構造の索引（揮発・鮮度非保証）。agy 側のログ・キャッシュ・transcript の場所もここ
- `LESSONS.md` — 追記専用の判断記録（冒頭に区画の説明。新エントリは末尾へ）。着手前に grep する
- `hook-investigation-report.md` / `plugin-gotchas.md` / `claude-import-investigation.md` — 調査資料（NOTES.md に索引）
- `README.md` — 利用者向け
