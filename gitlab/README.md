# GitLab MCP Server

このプロジェクトは、Model Context Protocol (MCP) を使用して、GitLabのプロジェクト（リポジトリ）情報、イシュー、マージリクエスト（MR）の操作をAIアシスタントから実行できるようにするためのMCPサーバーです。

Go言語で実装されており、標準入出力（Stdio）を介してMCPクライアントと通信します。

## 提供する機能（MCPツール）

本サーバーは以下のツールをMCPクライアントに公開します。

| ツール名 | 説明 | パラメータ |
| :--- | :--- | :--- |
| `get_project_info` | プロジェクトの基本情報（スター数、フォーク数、オープンイシュー数など）を取得します。 | `owner` (ユーザー名またはグループ名), `repo` (プロジェクト名) |
| `list_issues` | イシュー一覧を取得します（デフォルトは open/opened のみ）。 | `owner`, `repo`, `state` (`opened`/`closed`/`all`), `per_page` (ページあたりの件数) |
| `create_issue` | 新しいイシューを作成します。 | `owner`, `repo`, `title` (タイトル), `body` (本文) |
| `get_issue` | 指定したイシューの詳細情報およびコメント一覧を取得します。 | `owner`, `repo`, `issue_number` (イシュー IID) |
| `create_issue_comment` | イシューまたはマージリクエストにコメントを追加します。 | `owner`, `repo`, `issue_number` (イシューまたはMRのIID), `body` (コメント本文) |
| `list_mrs` | マージリクエスト一覧を取得します。 | `owner`, `repo`, `state` (`opened`/`closed`/`locked`/`merged`/`all`), `per_page` |
| `create_mr` | 新しいマージリクエストを作成します。 | `owner`, `repo`, `title`, `body`, `head` (ソースブランチ), `base` (ターゲットブランチ) |
| `get_mr` | マージリクエストの詳細（コンフリクト有無など）を取得します。 | `owner`, `repo`, `pr_number` (MRのIID) |
| `merge_mr` | マージリクエストをマージします。 | `owner`, `repo`, `pr_number` (MRのIID), `commit_title` (コミット詳細) |

## 必要条件

* **Go**: 1.26以上
* **GitLab 個人用アクセストークン (PAT) または OAuth トークン**: 操作対象のプロジェクトに応じた適切な権限（`api` スコープなど）が必要です。

## セットアップと認証

認証用トークンとAPIエンドポイントは、サーバー起動時に自動的に以下の優先順位で探索・取得されます。

1. **環境変数**:
   環境変数 `GITLAB_TOKEN` (または `GL_TOKEN`) が設定されている場合、そのトークンを使用します。
   また、オンプレミス版などの場合は `GITLAB_BASE_URL` (または `GL_BASE_URL`) でAPIのエンドポイントを指定可能です。
   ```bash
   export GITLAB_TOKEN=glpat-your-personal-access-token
   ```

2. **GitLab CLI (`glab`)**:
   環境変数が空の場合、システムにインストールされている GitLab CLI から認証情報を取得しようと試みます（`glab auth status -t` コマンドを使用）。
   OAuthトークンにも対応しており、CLI経由でログインしている場合は設定不要で自動的に動作します。

## ビルド方法

以下のコマンドを実行して依存関係を解決し、実行可能バイナリをビルドします。

```bash
go mod tidy
go build -o mcpServers/gitlab-plugin main.go
```

ビルドが完了すると、`mcpServers/gitlab-plugin` にバイナリが出力されます。

## プラグインとしての登録

本ディレクトリに含まれる `gemini-extension.json` が、Antigravity (`agy`) がこのMCPサーバーをロードするための構成ファイルです。

```bash
# ビルド後にインストール
agy plugin install /path/to/agy-plugins/gitlab
```

インストールは絶対パスで指定してください。`gemini-extension.json` 内の `${extensionPath}` 変数が `agy` によってインストール先ディレクトリに自動解決されるため、手動でのパス編集は不要です。
