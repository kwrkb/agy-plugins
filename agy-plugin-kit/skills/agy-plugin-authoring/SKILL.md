---
name: agy-plugin-authoring
description: agy（Antigravity CLI）プラグインを正しく作るための実証済みオーサリング規則。プラグインの新規作成・マニフェスト記述・mcp_config/hooks 設計・Windows 対応・URL install を行う際に必ず参照する。
---

# agy プラグイン オーサリング規則（実機検証済み）

agy プラグイン開発で繰り返し踏んできた落とし穴を規則化したもの。新規プラグイン作成・
`plugin.json`/`mcp_config.json`/`hooks.json` 記述・Windows 対応・URL install の前に参照する。
詳細な背景はリポジトリ `LESSONS.md`、確定事実はメモ `agy-plugin-format-facts` を参照。

## マニフェスト形式の選択（最重要）

- **`gemini-extension.json` 形式**（`plugin.json` を置かない）: install 時に `${extensionPath}`/`${/}` を
  **絶対パスへ解決**し `plugin.json`+`mcp_config.json` を生成する。**同梱バイナリを参照する場合はこれ**。
- **`plugin.json` 形式**（native）: `mcp_config.json` はそのままコピーされ **`${extensionPath}` は解決されない**
  （= Issue #390。実機確認済み）。PATH 上のコマンドを使う・置換が不要な場合のみこれ。
- **両方置かない**（C1 警告対象）。曖昧になり `plugin.json` 側が優先され copy-only 化する。

## Issue #390（`${extensionPath}` が native 形式で解決されない）

- native `plugin.json` 形式の `mcp_config.json` に `${extensionPath}` を書いても **literal のまま**残り、
  MCP の `command` パスが壊れて起動不能になる。
- 対処: (a) 同梱バイナリ参照なら `gemini-extension.json` 形式にする / (b) `/agy-plugin-kit:doctor` の
  path-fix（validator `--fix-paths`）で絶対パスへ書き換える / (c) PATH 上のコマンドにして同梱をやめる。

## Windows 対応

- `.sh`/`.cmd`/`.bat` を `command` に**直接置くと Windows で spawn 不可**（C3 エラー）。
  → トークン解決などのグルーは **Go で `.exe` にビルド**し、`gemini-extension.json` の `command` に
  **拡張子なしフルパス** `${extensionPath}${/}name` を指定する（agy が `.exe` を補完。実機確認済み）。
- `${CLAUDE_PLUGIN_ROOT}` は **agy に存在しない**（Claude Code 専用）。使うと解決されない（C8 エラー）。
  agy のトークンは `${extensionPath}` と `${workspacePath}` のみ。

## 同梱バイナリ

- 同梱する `.exe`/スクリプトが **`.gitignore` にマッチすると URL install で clone されず**起動不能になる
  （C4 エラー）。**明示的にコミット**すること。Go ラッパーはソース（`.go`+`go.mod`）と `.exe` を両方コミット。
- Go ラッパーは **stdout に一切書かない**（MCP は stdout で NDJSON を流すため）。診断は stderr へ（C9）。
  トークン解決は shell の `${A:-${B:-}}` 同様、空文字も未設定扱いでフォールバックする。

## コンポーネント構造

- `commands/<group>/<name>.toml`（`description=` / `prompt="""..."""`）→ `/group:name`。install 時に
  **skills に変換**される。`skills/<name>/SKILL.md` も同様にスラッシュコマンド化される。
- `hooks.json` は**ルート直下**（`hooks/` ではない）。`{"hooks":{"PostToolUse":[{"matcher":"...","hooks":[{"type":"command","command":"...","timeout":ms}]}]}}`。
  ※ プラグイン同梱 hook の**実行時発火は print mode では未確認**。対話セッション / `/hooks` 有効化が要る可能性あり。
- `rules/` の `.md` は常時システムプロンプトに載る（トークン高）。長い知識は skill（呼び出し時ロード）に置く。

## 検証

- 健全性チェックは **`agy plugin validate <path>`**（`agy doctor` は存在しない）。
- 「動く」の証明は **実環境（端末から agy）** で行う。`agy -p` 実行後に
  `~/.gemini/antigravity-cli/mcp/<server>/` の mtime 更新で MCP 起動成功を客観確認できる（手動 export の直叩きは不可）。
- ソースを再ビルドしたら **`agy plugin install` の再実行**が必要。

## このキットのツール

- `/agy-plugin-kit:new <name>` 正しい雛形を生成 / `/agy-plugin-kit:validate <path>` 上記トラップを機械検出 /
  `/agy-plugin-kit:doctor <path>` 検出＋修正提案（#390 の path-fix 含む） / `/agy-plugin-kit:doc <path>` README/SKILL 生成。
