# settings-advisor — 設定アドバイザー

カレントワークスペースのコードベース規模・使用言語・機密/CI/本番設定ファイルの有無をスキャンし、
agy（Antigravity CLI）の設定（モデル選定・サンドボックス・ツール許可モード）を**控えめに提案**する MCP サーバーです。
設定を自動変更はせず、ユーザーが `/model` や `/settings` から手動適用するための推奨だけを返します。

## 提供するMCPツール

### `settings_advisor`

指定パスを走査し、規模とタスク内容に応じた推奨設定を出力します。

#### パラメータ

- `path` (string, 任意): 解析するリポジトリのパス。指定がない場合はカレントディレクトリ。
- `task_hint` (string, 任意): 予定タスクの概要（例: `大規模リファクタリング`）。推奨の精緻化に使用。
- `format` (string, 任意): 出力形式。`text`（コンパクト数行）または `json`。デフォルトは `text`。

## インストール方法

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/settings-advisor
```

> **対応 OS（Linux / macOS / Windows）**: ソースは `src/`、配布物は `bin/` に分離。`bin/` に
> `settings-advisor-linux-amd64` / `settings-advisor-darwin-arm64` / `settings-advisor.exe`（ネイティブ）と、拡張子なしの
> OS 分岐 dispatcher `bin/settings-advisor`（shebang sh・`uname` で実機ネイティブを `exec`）を同梱。`command` は
> `${extensionPath}${/}bin${/}settings-advisor` で 3 OS を単一指定でカバーする（Windows は agy が `.exe` を直接起動）。

## 推奨ロジック

- **モデルティア**: 総行数と言語数から `light` / `mid` / `heavy` を判定。Go, TS, JS, Python, Rust, Dart, Swift, Vue, Svelte, PHP, Scala, C/C++, C#, Java, Kotlin, Ruby, Zig, Lua, SQL, Shell, Terraform などの主要開発言語を幅広く走査。ビルド生成物やパッケージキャッシュ（`.next`, `.nuxt`, `out`, `target`, `.dart_tool`, `Pods`, `node_modules`, `dist`, `build` 等）はスキップ。`task_hint` に「リファクタ/設計/アーキ/migrate」等が含まれる場合は `heavy` に格上げ。
- **モデル名**: `models.json` の定義および `traits`（`instruction-following` による Claude Sonnet、`quota-independent` による GPT-OSS など）に基づき動的に推奨モデルをマッチング。
- **サンドボックス**: 実機密設定の `.env`（系）を検出したら `enableTerminalSandbox: true` を提案（`.env.example` や `.env.sample` などのテンプレートは誤検知防止のため除外）。
- **ツール許可モード（`toolPermission`）**: CI/CD 設定（GitHub Actions, GitLab CI, CircleCI, Bitbucket, Azure Pipelines）検出時は `proceed-in-sandbox`、本番設定ファイルまたはパス（ファイル名や `config/prod/` などのディレクトリ名に `prod`/`production` トークンを含む場合）検出時は最も厳格な `strict` を提案。値はいずれも agy 本体の有効列挙値（`always-proceed` / `request-review` / `strict` / `proceed-in-sandbox`）。

## 同梱スキル

`skills/settings-advisor-gemini/SKILL.md` を同梱し、「どのモデルを使うべき？」「セキュリティ設定はどうすべき？」といった問いや大規模タスク着手前に本ツールを呼ぶようエージェントへ案内します。
