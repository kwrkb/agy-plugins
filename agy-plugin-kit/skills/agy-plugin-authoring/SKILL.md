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

## マルチプラットフォーム対応（同梱 Go バイナリ・`src/`＋`bin/`）

- `command` は **1 本の固定文字列**で agy は `${extensionPath}` 以外を置換せず OS 別に切り替えられない。Go の
  ネイティブは OS×arch ごとに別物なので、**dispatcher 方式**で単一 command に畳み込む（実機検証済み）:
  - ソースは `<plugin>/src/`、配布物は `<plugin>/bin/`。`bin/` に `<name>-linux-amd64` / `<name>-darwin-arm64` /
    `<name>.exe`（ネイティブ）＋拡張子なし `bin/<name>`（**POSIX sh dispatcher**＝OS 非依存テキスト。`uname` で実機
    判定し対応ネイティブを `exec`）。`command` は `${extensionPath}${/}bin${/}<name>`。Windows は agy が `.exe` を
    補完して `bin/<name>.exe` を直接起動＝dispatcher 非経由。
  - dispatcher はエラーのみ stderr（stdout は NDJSON 専用）。実行ビット（`100755`）をコミット
    （`git update-index --chmod=+x`。install コピーで +x は保持＝実機検証済み）。`.sh`/`.cmd`/`.bat` の直 `command`
    は Windows で spawn 不可（C3）だが、dispatcher は拡張子なし＋Windows は `.exe` 直起動なので C3 を踏まない。
- `${CLAUDE_PLUGIN_ROOT}` は **agy に存在しない**（Claude Code 専用）。使うと解決されない（C8 エラー）。
  agy のトークンは `${extensionPath}` と `${workspacePath}` のみ。

## 同梱バイナリ

- 同梱するネイティブ/スクリプトが **`.gitignore` にマッチすると URL install で clone されず**起動不能になる
  （C4 エラー）。**明示的にコミット**すること。Go プラグインはソース（`src/`）と `bin/` のネイティブ・dispatcher を
  すべてコミット。再ビルドはルートの `./build.sh <name>`（Windows は `./build.ps1`、Go バージョン固定で決定論）。
- MCP サーバー（および Go ラッパー）は **stdout に一切書かない**（MCP は stdout で NDJSON を流すため）。
  診断は stderr へ（C9）。トークン解決は shell の `${A:-${B:-}}` 同様、空文字も未設定扱いでフォールバックする。

## コンポーネント構造

- `commands/<group>/<name>.toml`（`description=` / `prompt="""..."""`）→ `/group:name`。install 時に
  **skills に変換**される。`skills/<name>/SKILL.md` も同様にスラッシュコマンド化される。
- **`hooks.json` は agy 1.0.9〜で利用可（1.0.8 では非機能だった）**: 1.0.8 ではフック payload に編集対象ファイルパスが無く（`toolCall: null`）発火も不安定で利用不可能だったが、**1.0.9 で payload に `toolCall.args.TargetFile`（編集ファイル絶対パス）が入り発火も安定化、1.0.10 で動的リロードも確認**された（本キットは validator フックを同梱＝`hooks.json` の自動バリデーション参照）。ただし payload は agy 独自スキーマで、変数（`${extensionPath}`/`${/}`）は依然未置換のため、フック command は **PWD 相対**（例: `validator/bin/validator --hook`）で書く。フックが使えない環境向けに明示コマンド（`/agy-plugin-kit:validate`）も併せて提供する。
- **`rules/` は使わない（プラグイン経路は 1.0.10 でも非機能）**: プラグイン内 `rules/*.md`・`plugin.json` の `"rules": [...]`・グローバル `~/.gemini/rules/*.md` のいずれも、agy 1.0.8/1.0.9/1.0.10 の実機検証でシステムプロンプト（`<user_rules>`）に**一切注入されなかった**。プラグインからエージェントへ固有の知識・規約を渡すには、必ず `skills/<name>/SKILL.md`（呼び出し時ロード）として定義する。**※ 1.0.10 でプロジェクトルートの `.agents/AGENTS.md`（Workspace Customizations Root）のみ注入されるようになったが、これはプロジェクト全体の規約用でありプラグインからは配置できない**（プラグイン知識の配布手段にはならない）。

## 検証

- 健全性チェックは **`agy plugin validate <path>`**（`agy doctor` は存在しない）。
- 「動く」の証明は **実環境（端末から agy）** で行う。`agy -p` 実行後に
  `~/.gemini/antigravity-cli/mcp/<server>/` の mtime 更新で MCP 起動成功を客観確認できる（手動 export の直叩きは不可）。
- ソースを再ビルドしたら **`agy plugin install` の再実行**が必要。

## このキットのツール

- `/agy-plugin-kit:new <name>` 正しい雛形を生成 / `/agy-plugin-kit:validate <path>` 上記トラップを機械検出 /
  `/agy-plugin-kit:doctor <path>` 検出＋修正提案（#390 の path-fix 含む） / `/agy-plugin-kit:doc <path>` README/SKILL 生成。
