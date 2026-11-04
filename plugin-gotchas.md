# Antigravity CLI (agy) プラグイン開発の既知の問題と回避策まとめ

`agy` におけるプラグイン開発・運用時の既知の仕様・罠、および `agy 1.4.2` 実機検証に基づく最新の解決状況と回避策をまとめました。

## 1. パス変数の展開仕様と `command` フィールドの罠 (Issue #390, #395)
- **Native 形式 (`plugin.json` + `mcp_config.json`)**:
  - `${extensionPath}` および `${/}` は install 時にも実行時にも展開されず、リテラル文字列のまま残ります。upstream の回答（2026-11-03）では legacy の `gemini-extension.json` 専用で、仕様どおりの動作です（Issue #390）。
  - native 形式の MCP サーバーは cwd がプラグインディレクトリ（v1.1.28〜）なので、同梱バイナリは `"command": "./bin/<name>"` と相対パスで書きます。`agy 1.4.12` の Linux / Windows で実機確認済みで、Windows では拡張子なしの `command` に agy が `.exe` を補います。
  - `agy 1.3.2` 以降、MCP サーバーの `args` / `env`（および `1.4.0` で HTTP `headers`）では `${PLUGIN_ROOT}` と `${PLUGIN_DATA}` が実行時に絶対パスへ展開されるようになりましたが、**`mcp_config.json` の `command` フィールド自体では `${PLUGIN_ROOT}` が展開されず起動に失敗します**。
- **`hooks.json`**:
  - `agy 1.4.2` でも `hooks.json` の `command` 内では `${PLUGIN_ROOT}`・`${extensionPath}`・`${/}` のいずれも展開されません（`sh -c` が未定義環境変数として空文字展開し、`${/}` は `Bad substitution` で失敗します）。

> [!TIP]
> **回避策**
> - **同梱バイナリを起動する MCP サーバー**: native 形式（`plugin.json` + `mcp_config.json`）で作成し、`"command": "./bin/<name>"` と記述します。`bin/<name>` を拡張子なしの POSIX sh dispatcher にし、`bin/<name>.exe` を並べれば、Linux / macOS は dispatcher、Windows は agy が補った `.exe` が起動します。2026-11-04 までは `gemini-extension.json` 形式の `${extensionPath}${/}bin${/}<name>` を使っていました。
> - **`agy plugin validate`（v1.4.0〜）**: `plugin.json` を直接要求します。native 形式のソースにはそのままかけられます（legacy の `gemini-extension.json` 形式なら install 先にかけます）。`agy-plugin-kit` の `validator <src>` と併用します。
> - **`hooks.json`**: 変数を使わず、プラグインルートからの相対パスで記述します（フック実行時の `pwd` はプラグインルートになります）。フックは Windows では `cmd /c`、それ以外では `sh -c` で起動され、`cmd` はスラッシュ区切りの `validator/bin/validator` の `/bin` をスイッチとして読むため起動できません。Windows 形と Unix 形を `||` でつなぎます（例: `"command": "validator\\bin\\validator --hook || validator/bin/validator --hook"`。`sh` では前半が not found になり後半が動く）。
> - **`hooks.json` の `timeout` は秒単位**（既定 30）です。Claude Code のミリ秒の感覚で `10000` と書くと約 2.8 時間になります。
> - **`PostToolUse` フックの出力はモデルに届きません**（stderr は CLI ログのみ、stdout の `injectSteps` はツール結果ごとエラー化、`overwriteResult` は無視）。モデルに伝えたい内容は `conversationId` ごとのファイルに書き、`PreInvocation` フックの `injectSteps`（`ephemeralMessage`）で差し込みます。

## 2. プラグイン定義のサブエージェント (Issue #551 — v1.3.2 / v1.4.1〜 で解消)
以前はプラグイン内のサブエージェント（`agent.json`）が `not found or not allowed to be invoked` でブロックされていましたが、**現行仕様では Markdown 形式で正常に動作します**。

> [!IMPORTANT]
> **現在の仕様（v1.4.1〜）**
> - 旧 `agent.json` は CLI に無視され、`agy plugin validate` で警告対象となります。
> - サブエージェントを同梱する場合は `agents/<name>.md` に YAML frontmatter（`name` と `description` 必須）とシステムプロンプト本文を記述してください。親プラグインの MCP サーバーも利用可能です。

## 3. プラグインからの `rules/` 自動注入 (Issue #396 — v1.4.x で解消)
以前はプラグイン内の `rules/*.md` が `<user_rules>` に一切反映されませんでしたが、**現在は YAML frontmatter 付きで自動注入されます**。

> [!NOTE]
> **現在の仕様と `skills/` との使い分け（v1.4.1〜）**
> - `rules/*.md`（推奨: `rules/AGENTS.md`）には YAML frontmatter で `trigger`（`always_on` / `model_decision` / `glob`）の指定が必須です（frontmatter 無しやプラグインルート直下の `AGENTS.md` は `agy plugin validate` でエラー/警告になります）。
> - ただし、`always_on` ルールは全ルール共有の常時コンテキスト予算（20,000 トークン上限、1ファイル 24 KB 上限）を毎ターン消費します。MCP ツールの引数リファレンスやワークフロー手順のように呼び出し時だけ必要な知識は、引き続きオンデマンド読み込みの `skills/<name>/SKILL.md`（Progressive Disclosure）として定義するのが推奨です。

## 4. MCP サーバーの stdout 出力による通信破壊
同梱したバイナリや起動用のディスパッチャスクリプトが少しでも標準出力（`stdout`）にデバッグログなどを吐いてしまうと、MCP が使用する NDJSON プロトコルが破壊され、サーバーが即クラッシュします。

> [!WARNING]
> **回避策**
> ディスパッチャやバイナリの処理経過・エラー等の診断ログは、必ずすべて標準エラー出力（`stderr`）に向けるように実装してください。

## 5. Stateful な MCP サーバーのデータ永続化 (Issue #554 — v1.3.2〜 で解消)
プラグインのアップグレード（`agy plugin install` の再実行）を行うとプラグインのインストールディレクトリ（`~/.gemini/config/plugins/<name>`）は置き換えられますが、**v1.3.2 からプラグイン専用の永続データ領域 `PLUGIN_DATA` が提供されました**。

> [!CAUTION]
> **永続データの保存先**
> - 認証トークンやキャッシュ等の永続データは、プラグインインストール先（`PLUGIN_ROOT`）ではなく、環境変数 `PLUGIN_DATA`（実体は `~/.gemini/antigravity-cli/plugin_data/<name>`）配下に保存してください。
> - `mcp_config.json` の `args` や `env` でも `${PLUGIN_DATA}` マクロとして参照可能です（※ワークスペース `.agents/plugins/` 直置き時は空文字になるため、未設定時のフォールバックを設けると安全です）。

## 6. `bin/` 同梱 CLI ツールの PATH 公開 (v1.3.2〜 新機能)
プラグインの `bin/` ディレクトリに実行可能ファイルを置いて `agy plugin install` すると、`~/.gemini/antigravity-cli/bin` 経由でエージェントの `PATH` 先頭に公開され、`run_command` からコマンド名だけで直接呼び出せます。なお、その子プロセスには `PLUGIN_ROOT` / `PLUGIN_DATA` 環境変数は自動注入されないため、スクリプト内でパス解決が必要な場合は `$0` から辿る設計にしてください。
