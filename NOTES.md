# NOTES: agy-plugins（内部構造の索引）

どこに何があるかの索引。**揮発・鮮度非保証**で、コードと食い違っていたらコードが正しい
（優先順は VISION.md > PLAN.md > コード > NOTES.md）。判断の理由はここに書かず `LESSONS.md` に、
現在有効な設計判断は `VISION.md` に書く。

## リポジトリの構成

| パス | 中身 |
|---|---|
| `<plugin>/src/` | Go 製プラグインのソース（独立した Go モジュール。module パスは `github.com/kwrkb/agy-plugins/<plugin>`） |
| `<plugin>/bin/` | 配布物: `<name>-linux-amd64` / `<name>-darwin-arm64` / `<name>.exe` と、拡張子なしの dispatcher `<name>`（100755） |
| `<plugin>/gemini-extension.json` | マニフェスト兼 MCP 定義（`command` は `${extensionPath}${/}bin${/}<name>`） |
| `<plugin>/skills/<name>/SKILL.md` | エージェント向けのツールリファレンス |
| `gitlab/` | `plugin.json` + `mcp_config.json`（`glab mcp serve`）と skill だけ。Go ソースなし |
| `agy-plugin-kit/` | 開発キット（下記） |
| `build.sh` / `build.ps1` | 決定論ビルド。引数でターゲットを選ぶ（`github` / `validator` / … / 省略で全部） |
| `.go-version` | 固定する Go のパッチ版（build スクリプトと CI が読む） |
| `.github/workflows/build-verify.yml` | モジュールごとに vet・test・govulncheck・再ビルド後の `git diff --exit-code`（stale 検出） |

Go 製プラグイン: github / ast-grep / go-lsp / retro-status / settings-advisor / worktree-manager / test-runner
（`mark3labs/mcp-go` v1 系に依存）。validator は依存なし。

## agy-plugin-kit

| パス | 中身 |
|---|---|
| `validator/src/main.go` | validator 本体（1 ファイル）。モードは `<dir>` / `--hook` / `--inject` / `--fix-paths` |
| `validator/bin/` | 配布物（他プラグインと同じ 3 本＋dispatcher） |
| `hooks.json` | 名前付きフック `agy-plugin-kit-validator`: PostToolUse → `--hook`、PreInvocation → `--inject`（各 timeout 10 秒） |
| `commands/agy-plugin-kit/` | `/agy-plugin-kit:new` / `:validate` / `:doctor` / `:doc` |
| `skills/agy-plugin-authoring/` | オーサリング規則の skill |
| `templates/` | `:new` が使う雛形 |
| `fixtures/` | validator の動作確認用の壊れたプラグイン（`broken-plugin`） |

### validator の検査（`validate()`）

| コード | 重大度 | 内容 |
|---|---|---|
| C1 | WARN | `plugin.json` と `gemini-extension.json` の二重マニフェスト |
| C2 | ERROR | native 形式で `${extensionPath}`/`${/}`、または MCP `command` で `${PLUGIN_ROOT}`/`${PLUGIN_DATA}` |
| C3 | ERROR | MCP `command` に `.sh`/`.cmd`/`.bat` |
| C4 | ERROR | 参照する実行ファイルが `.gitignore` 対象 |
| C5 | ERROR | マニフェストが無い・空・不正 JSON（`.antigravity-plugin/` / `.gemini-plugin/` 配下も対象） |
| C6 | WARN | `${extensionPath}` 形式の `command` に `.exe` |
| C7 | WARN | トークン専用 MCP サーバーを wrapper なしで起動（heuristic） |
| C8 | ERROR | `${CLAUDE_PLUGIN_ROOT}` |
| C9 | WARN | 同梱 Go ラッパーが自前で stdout に書く（heuristic） |
| C10 | WARN | hooks.json で `${extensionPath}`/`${PLUGIN_ROOT}`/`${PLUGIN_DATA}`/`${/}` |
| C11 | WARN | hooks.json の `timeout` が 600 超（ミリ秒の持ち込み） |

### フックの受け渡し

- `handleHook`: マニフェスト系（`isManifestFile`）の編集だけ検査し、結果を stderr と受け渡しファイルに書く。stdout は常に `{}`
- `injectOutput`: 受け渡しファイルを `takeRelay` で消費し、`{"injectSteps":[{"ephemeralMessage":...}]}` を返す
- 受け渡しファイル: `os.UserCacheDir()/agy-plugin-kit/relay/<conversationId>.txt`（`safeRelayDir` が 0700・非 symlink を確認）
  - Linux `~/.cache/…` / macOS `~/Library/Caches/…` / Windows `%LocalAppData%\…`

## agy 側の場所（実機検証で見る所）

| パス | 中身 |
|---|---|
| `~/.gemini/config/plugins/<name>/` | install 先（install は wipe しない） |
| `~/.gemini/antigravity-cli/mcp/<plugin>_<server>/` | MCP のツール定義キャッシュ（起動の証拠） |
| `~/.gemini/antigravity-cli/log/cli-*.log` | CLI ログ（フックの stderr・失敗、MCP の起動・停止） |
| `~/.gemini/antigravity-cli/brain/<conversationId>/.system_generated/logs/transcript.jsonl` | 会話のステップ列（`EPHEMERAL_MESSAGE` などの確認） |
| `~/.gemini/antigravity-cli/plugin_data/<name>/` | プラグインの永続データ（`PLUGIN_DATA`） |

## 調査資料

| ファイル | 中身 |
|---|---|
| `hook-investigation-report.md` | agy のフックと rules の実機調査（1.0.8〜1.4.2）。§9 出力経路・timeout、§10 Windows の起動経路 |
| `plugin-gotchas.md` | agy プラグイン開発の既知の罠と回避策の要約 |
| `claude-import-investigation.md` | `agy plugin import/install` で Claude Code 形式のプラグインを agy 形式へ変換できるかの調査（1.0.x） |
| `CHANGELOG.md` | 利用者向けの変更履歴 |
