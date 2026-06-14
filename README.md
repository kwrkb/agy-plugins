# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

現在、以下のプラグインが含まれています。

* **[github](./github/README.md)**
  GitHubのリポジトリ情報、イシュー、プルリクエストの操作をAIアシスタントから実行できるようにするMCPサーバーです。GitHub CLI (`gh`) の認証情報を自動的に読み込む機能を備えています。
* **[gitlab](./gitlab/README.md)**
  GitLabのプロジェクト情報、イシュー、マージリクエストの操作をAIアシスタントから実行できるようにするMCPサーバーです。GitLab CLI (`glab`) の認証情報（OAuthトークン対応）を自動的に読み込む機能を備えています。

## インストール方法

`agy` CLI などの対応クライアントを使用している場合、以下のコマンドでプラグインを直接インストールできます。

```bash
# GitHub プラグインのインストール
agy plugin install /path/to/agy-plugins/github

# GitLab プラグインのインストール
agy plugin install /path/to/agy-plugins/gitlab
```

> **⚠️ 重要: インストール時の絶対パスに関する注意**
> 
> 1. `agy plugin install` コマンドを実行する際は、必ず対象プラグインのソースディレクトリを **絶対パス** で指定してください。相対パスで指定すると正しくインストールされない場合があります。
> 2. インストール完了後、`agy` CLI のパス解決の仕様により、MCPサーバーへの接続（プロセスの起動）に失敗することがあります。その場合は、インストール先の構成ファイル（例: `~/.gemini/config/plugins/github/plugin.json` および `mcp_config.json` 等）を開き、`"command"` の値を `./mcpServers/...` のような相対パスから、**実行バイナリの完全な絶対パス** に直接書き換えてください。

## 開発とビルド

各プラグインディレクトリに移動し、Goコマンドで依存関係の解決とビルドを行います。

```bash
cd gitlab
go mod tidy
go build -o mcpServers/gitlab-plugin main.go
```

## 動作要件
* Go 1.20以上
* 各プラットフォームに対応した CLI ツール（`gh` や `glab`）またはアクセストークンの環境変数設定
