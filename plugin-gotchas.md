# Antigravity CLI (agy) プラグイン開発の既知の問題と回避策まとめ

`agy` におけるプラグイン開発・運用時の既知の仕様・罠、および `agy 1.4.2` 実機検証に基づく最新の解決状況と回避策をまとめました。

## 1. パス変数の展開仕様と `command` フィールドの罠 (Issue #390, #395)
- **Native 形式 (`plugin.json` + `mcp_config.json`)**:
  - `${extensionPath}` および `${/}` は install 時にも実行時にも展開されず、リテラル文字列のまま残ります（Issue #390 継続）。
  - `agy 1.3.2` 以降、MCP サーバーの `args` / `env`（および `1.4.0` で HTTP `headers`）では `${PLUGIN_ROOT}` と `${PLUGIN_DATA}` が実行時に絶対パスへ展開されるようになりましたが、**`mcp_config.json` の `command` フィールド自体では `${PLUGIN_ROOT}` が展開されず起動に失敗します**。
- **`hooks.json`**:
  - `agy 1.4.2` でも `hooks.json` の `command` 内では `${PLUGIN_ROOT}`・`${extensionPath}`・`${/}` のいずれも展開されません（`sh -c` が未定義環境変数として空文字展開し、`${/}` は `Bad substitution` で失敗します）。

> [!TIP]
> **回避策**
> - **同梱バイナリを起動する MCP サーバー**: 3 OS（Linux / macOS / Windows）共通で確実に絶対パス起動するため、ソースは `gemini-extension.json` 形式（`plugin.json` を併置しない）で作成し、`"command": "${extensionPath}${/}bin${/}<name>"` と記述します（`agy plugin install` 時に絶対パス解決済みの `mcp_config.json` と `plugin.json` が生成されます）。
> - **`agy plugin validate` の使い分け（v1.4.0〜）**: `agy plugin validate` は `plugin.json` を直接要求するため、`gemini-extension.json` 形式のソースディレクトリには直接かけず、ソース検査には `agy-plugin-kit` の `validator <src>` を、install 後検査には `agy plugin validate ~/.gemini/config/plugins/<name>` を使います。
> - **`hooks.json`**: 変数を使わず、プラグインルートからの相対パス（例: `"command": "validator/bin/validator --hook"`）で記述します（フック実行時の `pwd` はプラグインルートになります）。

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
