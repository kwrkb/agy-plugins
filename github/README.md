# GitHub MCP Server

このプロジェクトは、Model Context Protocol (MCP) を使用して、GitHubのリポジトリ情報、イシュー、プルリクエストの操作をAIアシスタント（Antigravityなど）から実行できるようにするためのMCPサーバーです。

Go言語で実装されており、標準入出力（Stdio）を介してMCPクライアントと通信します。

## 提供する機能（MCPツール）

本サーバーは以下のツールをMCPクライアントに公開します。

| ツール名 | 説明 | パラメータ |
| :--- | :--- | :--- |
| `get_repo_info` | リポジトリの基本情報（スター数、フォーク数、オープンイシュー数など）を取得します。 | `owner` (リポジトリ所有者), `repo` (リポジトリ名) |
| `list_issues` | イシュー一覧を取得します（デフォルトはオープンのみ）。 | `owner`, `repo`, `state` (`open`/`closed`/`all`), `per_page` (ページあたりの件数) |
| `create_issue` | 新しいイシューを作成します。 | `owner`, `repo`, `title` (タイトル), `body` (本文) |
| `get_issue` | 指定したイシューの詳細情報およびコメント一覧を取得します。 | `owner`, `repo`, `issue_number` (イシュー番号) |
| `create_issue_comment` | イシューまたはプルリクエストにコメントを追加します。 | `owner`, `repo`, `issue_number`, `body` (コメント本文) |
| `list_prs` | プルリクエスト一覧を取得します。 | `owner`, `repo`, `state` (`open`/`closed`/`all`), `per_page` |
| `create_pr` | 新しいプルリクエストを作成します。 | `owner`, `repo`, `title`, `body`, `head` (ソースブランチ), `base` (ターゲットブランチ) |
| `get_pr` | プルリクエストの詳細（マージ可能ステータスなど）を取得します。 | `owner`, `repo`, `pr_number` |
| `merge_pr` | プルリクエストをマージします。 | `owner`, `repo`, `pr_number`, `commit_title` (コミット詳細), `merge_method` (`merge`/`squash`/`rebase`) |

## 必要条件

* **Go**: 1.20以上
* **GitHub 個人用アクセストークン (PAT)**: 操作対象のリポジトリに応じた適切な権限（`repo` スコープなど）が必要です。

## セットアップと認証

認証用トークンは、サーバー起動時に自動的に以下の優先順位で探索・取得されます。

1. **環境変数**:
   環境変数 `GITHUB_TOKEN` または `GH_TOKEN` が設定されている場合、そのトークンを使用します。
   ```bash
   export GITHUB_TOKEN=ghp_your_personal_access_token
   ```

2. **GitHub CLI (`gh`)**:
   環境変数が空の場合、システムにインストールされている GitHub CLI から認証情報を取得しようと試みます（`gh auth token` コマンドを使用）。

*※認証情報が見つからない場合でも起動はしますが、パブリックリポジトリ以外の操作やAPI制限により正しく機能しない場合があります。*

## ビルド方法

以下のコマンドを実行して実行可能バイナリをビルドします。

```bash
go build -o mcpServers/github-plugin main.go
```

ビルドが完了すると、`mcpServers/github-plugin` にバイナリが出力されます。

## プラグインとしての登録

本リポジトリに含まれる `plugin.json` は、Antigravity などの互換クライアントがこのMCPサーバーをロードするための構成ファイルです。

```json
{
  "name": "github",
  "mcpServers": {
    "github": {
      "command": "./mcpServers/github-plugin"
    }
  }
}
```

この設定に従い、クライアントはビルドされた `./mcpServers/github-plugin` を起動して標準入出力経由で通信を行います。

> **⚠️ 重要: インストール時の絶対パスに関する注意**
> 
> 1. `agy plugin install` コマンドを実行する際は、必ず対象プラグインのソースディレクトリを **絶対パス** で指定してください。相対パスで指定すると正しくインストールされない場合があります。
> 2. インストール完了後、`agy` CLI のパス解決の仕様により、MCPサーバーへの接続（プロセスの起動）に失敗することがあります。その場合は、インストール先の構成ファイル（例: `~/.gemini/config/plugins/github/plugin.json` および `mcp_config.json` 等）を開き、`"command"` の値を `./mcpServers/...` のような相対パスから、**実行バイナリの完全な絶対パス** に直接書き換えてください。
