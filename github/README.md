# GitHub MCP Server (公式版プラグイン)

公式の [github/github-mcp-server](https://github.com/github/github-mcp-server) を Antigravity CLI (`agy`) プラグインとして使えるようにパッケージしたものです。

以前は自作の Go 製 MCP サーバーでしたが、公式サーバーへ置き換えました。issues / pull_requests / repos / actions / code_security / discussions など、公式がメンテナンスする全ツールセットを利用できます。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `gemini-extension.json` | `agy` がプラグインをロードするための構成ファイル |
| `github-mcp-wrapper.mjs` | 公式バイナリ起動前に認証トークンを解決するラッパー（Node.js 製・クロスプラットフォーム） |
| `build.sh`（リポジトリルート） | 公式バイナリを `go install` で取得し、ラッパーと共に `mcpServers/` へ配置 |

ビルド成果物（`mcpServers/` 配下の公式バイナリとラッパー）は `.gitignore` で除外され、`build.sh` で生成します。

## 必要条件

* **Go**: 1.26 以上（`go install` で公式バイナリをビルドするため）
* **Node.js**: 18 以上（`agy` 本体が Node なので、通常は既に利用可能）
* **GitHub 認証**: 以下のいずれか
  * 環境変数 `GITHUB_PERSONAL_ACCESS_TOKEN` / `GITHUB_TOKEN` / `GH_TOKEN`
  * GitHub CLI (`gh auth login` 済み)

## 認証の仕組み

公式バイナリは `GITHUB_PERSONAL_ACCESS_TOKEN` のみを参照します。
`github-mcp-wrapper.mjs` が起動時に次の優先順位でトークンを解決し、`GITHUB_PERSONAL_ACCESS_TOKEN` として公式バイナリに渡します。

1. `GITHUB_PERSONAL_ACCESS_TOKEN`（既に設定済みならそのまま）
2. `GITHUB_TOKEN`
3. `GH_TOKEN`
4. `gh auth token`（GitHub CLI の認証情報）

これにより、静的な PAT を環境変数に置かなくても `gh` 認証だけで動作します。

## ビルド方法

リポジトリルートの `build.sh` を実行します（github / gitlab 両方をビルド）。

```bash
./build.sh
```

完了すると以下が生成されます。

* `mcpServers/github-mcp-server` （Linux/macOS）または `mcpServers/github-mcp-server.exe`（Windows） … 公式バイナリ (v1.3.0)
* `mcpServers/github-mcp-wrapper.mjs` … 認証ラッパー

公式バージョンを変更する場合は `build.sh` の `GITHUB_MCP_VERSION` を編集してください。

## プラグインとしての登録

```bash
# ビルド後にインストール（絶対パスで指定）
agy plugin install /path/to/agy-plugins/github
```

`gemini-extension.json` 内の `${extensionPath}` 変数が `agy` によってインストール先ディレクトリに自動解決されるため、手動でのパス編集は不要です。

## 提供する機能

公式サーバーのデフォルトツールセット（context / copilot / issues / pull_requests / repos / users）が有効になります。`--toolsets` や `--read-only` 等のオプションを使いたい場合は `gemini-extension.json` の `args` に追加してください（例: `"args": ["${extensionPath}${/}mcpServers${/}github-mcp-wrapper.mjs", "--read-only"]`）。

利用可能なツールの詳細は公式ドキュメントを参照してください: <https://github.com/github/github-mcp-server>
