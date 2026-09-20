# agy-plugins — プロジェクト固有ガイド

agy (Google Antigravity CLI) 向け MCP プラグイン集。グローバル CLAUDE.md のルールに加え、本リポジトリ固有の事実のみをここに記す。

## 構成（6プラグイン + gitlab / 6 Go モジュール）

Go プラグインは **`src/`（ソース）＋ `bin/`（配布物）** に分離。`bin/` に各 OS のネイティブ
`<name>-linux-amd64` / `<name>-darwin-arm64` / `<name>.exe` と、拡張子なしの **OS 分岐 dispatcher**
`bin/<name>`（shebang sh・`uname` で実機ネイティブを `exec`、+x コミット）を **git にコミットして配布**。
`command` は `${extensionPath}${/}bin${/}<name>`（Windows は agy が `.exe` を補完し `bin/<name>.exe` を直接起動＝dispatcher 非経由）。

- `github/` — `gh` CLI を exec する自作 Go 製 MCP サーバー。module `github.com/kwrkb/agy-plugins/github`（`github/src/`）。
- `ast-grep/` — `ast-grep` CLI を exec。`retro-status/` — リポジトリ解析。`settings-advisor/` — Gemini の settings 助言。`go-lsp/` — `gopls` 経由の Go LSP（definition/references/hover）。いずれも Go 製 MCP サーバーで src/bin 構成、module パスは `github.com/kwrkb/agy-plugins/<name>`。
- `gitlab/` — `glab mcp serve` を呼ぶ薄い設定のみ（`plugin.json` + `mcp_config.json`、Go バイナリ無し＝src/bin 非対象）。
- `agy-plugin-kit/` — プラグイン開発ヘルパー。`validator/`（Go 製・module `agy-plugin-validator`・`validator/src/`＋`validator/bin/`）＋ `skills/` `commands/` `templates/`。hook は `validator/bin/validator --hook`。

## コマンド

```bash
# テスト・静的解析（モジュール別。ソースは <plugin>/src/ 配下）
cd github/src && go vet ./... && go test ./...
cd agy-plugin-kit/validator/src && go vet ./... && go test ./...
# 他プラグイン（ast-grep / retro-status / settings-advisor / go-lsp）も同じ流儀（<name>/src で go vet ./... && go test ./...）
# バイナリ再ビルド（Go 版は .go-version に固定。build.sh が GOTOOLCHAIN で強制するため事前準備不要。Windows は ./build.ps1）
./build.sh                                    # 全プラグイン
./build.sh github                             # github だけ
./build.sh validator                          # validator だけ
# 他ターゲット: ast-grep | retro-status | settings-advisor | go-lsp
```

**ソース変更時は必ず `./build.sh` で再ビルドしてコミット**（`agy plugin install` はビルドせず git 追跡バイナリをコピーするだけ）。決定論フラグは `build.sh` に、Go のパッチ版は `.go-version` に集約され（`build.sh`/`build.ps1`/CI が同じファイルを読む）、bit-identical になる。CI の stale 検出ゲート（`.github/workflows/build-verify.yml`）がこれを前提にする。

## Go バージョンの引き上げ方針

固定版は `.go-version` の1箇所（`build.sh`/`build.ps1`/CI が読む）。**上げるのは以下のいずれかに当たる時だけ**で、「新しい版が出たから」では上げない。1回の引き上げは21バイナリの再ビルド＝約 157MB の新規 blob を伴う。

1. **到達可能な stdlib 脆弱性が、固定中のマイナー内のパッチで修正済み** → そのマイナーの最新パッチへ。言語変更が無く再ビルドのみで済む。**CI の govulncheck がこのケースを fail させる**ので、検知は自動。
2. **固定中のマイナーが EOL**（Go は最新2マイナーのみ patch。1.28 リリース時点で 1.26 が該当）→ サポート内マイナーへ移行。`go vet` の新チェックや挙動変更を見込んでテストを流す。
3. 依存の `go.mod` 下限が固定版を超えた。
4. 必要な言語 / stdlib 機能がある。

マイナー移行は **EOL 直前まで据え置く**（最新マイナーへの追従はしない）。

手順は `.go-version` を書き換えて `./build.sh`（全プラグイン）→ 21バイナリをコミット。`go.mod` の `go` ディレクティブは**言語の下限**であって固定版とは別の軸なので、引き上げに合わせて動かさない。

## 実機検証（tmux + agy）

agy の対話セッションは PTY を要するため tmux 経由で起こす。クリーン install → ツール実行までを実環境で確認する。以下は **agy 1.2.7 で実測した手順**で、1.0.x 当時の書き方から 2 点変わっている（末尾の注記）。

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
ls -l ~/.gemini/config/plugins/test-runner/bin/   # +x が保持されていること

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

## 非自明な地雷（詳細は LESSONS.md の番号付き教訓）

以下のうちバージョンに言及する項目は 1.0.8〜1.0.15 で確かめたもの。**実機の agy は 1.2.7** で、install・`${extensionPath}` 解決・MCP 起動は 1.2.7 でも成立を再確認済みだが、`rules/` と hooks は 1.2.7 で再検証していない。

- **バイナリは追跡コミット必須**: `agy plugin install` はビルドせず git 追跡ファイルをコピーするだけ。`src/main.go` を変えたら全 OS 分（linux-amd64/darwin-arm64/windows）を再ビルド＆コミットしないと stale が配布される（#21）。
- **単一 command を OS 分岐 dispatcher で全 OS 化**: `command` は1本の固定文字列で agy は `${extensionPath}` 以外を置換しない。拡張子なし `bin/<name>`（shebang sh）が `uname` で実機ネイティブを `exec` し、Linux/macOS(arm64)/Windows を単一指定でカバー（実機検証: macOS arm64 で MCP 起動・hook 発火・install コピーで +x 保持を確認 #40）。
- **`${extensionPath}` 解決条件**: ソースに `gemini-extension.json` があり `plugin.json` が**無い**時のみ解決（#1）。同梱バイナリ参照プラグインは前者構成。
- **install は wipe しない**: 設計変更時は旧ファイルが残る。再 install 前に `~/.gemini/config/plugins/<name>/` を削除（#24）。
- **検証は MCP キャッシュのツール名で**: mtime 更新だけでなく中身（ツール名）で新サーバーを別人確認（#25）。キャッシュの場所は `mcp/<plugin>_<server>/`（上記「実機検証」参照）。
- **agy の `rules/` は非機能（プラグイン経路）／プロジェクト `.agents/AGENTS.md` は 1.0.10 で機能化・1.0.15 でも進展なし**（#41 / #48）。1.0.10 でワークスペースルート `.agents/AGENTS.md` は `<user_rules>` に `<RULE[...]>` 注入される（`customizations.agentsCustomization`→`UserRulesSection`、Linux 再現済み）が、**プラグイン内 `rules/*.md`・`plugin.json "rules"`・グローバル `~/.gemini/rules` は依然非注入**（1.0.8/1.0.9 #22/#35、1.0.15 でも #48 で再確認）。**プラグインからエージェントへ渡す知識は引き続き `skills/` で**（`rules/` への移行は不可）。
- **agy の hooks は 1.0.9 で部分機能化・1.0.10 で動的リロード確認**: `PostToolUse` payload の `toolCall.args.TargetFile` に編集ファイル絶対パスが入り、2回目以降の編集・`agy -p` でも発火、自前バイナリは PWD 相対で呼べる（#34）。1.0.10 では install したフックが**親セッションに次ツール実行から動的適用**される（再起動不要 #42）。ただし payload は agy 独自スキーマ／`${extensionPath}` 未置換・`${/}` は `Bad substitution` 継続は不変。1.0.8 では全面非機能だった（#18-21）。

## ドキュメント地図

- `LESSONS.md` — 番号付き実装教訓（最重要・着手前に grep）＋ 末尾に設計判断ログ（意思決定の経緯）
- `PLAN.md` — タスク進捗 / `README.md` — 利用者向け
