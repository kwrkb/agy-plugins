# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

現在、以下のプラグインが含まれています。

* **[github](./github/README.md)**
  GitHubのリポジトリ情報、イシュー、プルリクエストの操作をAIアシスタントから実行できるようにするMCPサーバーです。GitHub CLI (`gh`) の認証情報を自動的に読み込む機能を備えています。
* **[gitlab](./gitlab/README.md)**
  GitLabのプロジェクト情報、イシュー、マージリクエストの操作をAIアシスタントから実行できるようにするMCPサーバーです。GitLab CLI (`glab`) の認証情報（OAuthトークン対応）を自動的に読み込む機能を備えています。

## インストール方法

リポジトリルートで `build.sh` を実行してバイナリをビルドしてから、`agy plugin install` でインストールします。

```bash
# ビルド（両プラグイン）
./build.sh

# GitHub プラグインのインストール
agy plugin install /path/to/agy-plugins/github

# GitLab プラグインのインストール
agy plugin install /path/to/agy-plugins/gitlab
```

インストールは絶対パスで指定してください。`gemini-extension.json` 内の `${extensionPath}` 変数が `agy` によってインストール先ディレクトリに自動解決されるため、手動でのパス編集は不要です。

## 開発とビルド

```bash
# 両プラグインを一括ビルド
./build.sh
```

GitLab プラグインはビルド不要（システムの `glab` を直接利用）。

## 動作要件

* **Go** 1.26 以上（github プラグインのバイナリビルドに必要）
* 各プラットフォームの CLI ツール（`gh` / `glab`）またはアクセストークンの環境変数設定
