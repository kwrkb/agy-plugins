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

> **注意**: インストール後、実行環境のパス解決の仕様によっては、インストール先の `plugin.json` および `mcp_config.json` 内の `command` をバイナリの絶対パスへ変更する必要がある場合があります。

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
