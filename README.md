# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

| プラグイン | 説明 |
| :-- | :-- |
| **[github](./github/README.md)** | GitHub の Issues / PR / リポジトリ等を AI アシスタントから操作する MCP サーバー |
| **[gitlab](./gitlab/README.md)** | GitLab の Issues / MR / プロジェクト等を AI アシスタントから操作する MCP サーバー |

## インストール方法

```bash
# GitHub プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/github

# GitLab プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

各プラグインの前提条件（PATH に入れるバイナリ / 認証設定）については、各ディレクトリの README を参照してください。

## 動作要件

| プラグイン | 必要な CLI / バイナリ | 認証 |
| :-- | :-- | :-- |
| github | `github-mcp-server`（PATH 上） | 環境変数 `GITHUB_PERSONAL_ACCESS_TOKEN` |
| gitlab | `glab` >= v1.74.0（PATH 上） | `glab auth login` 済み |

## ライセンス・帰属

各プラグインは公式 MCP 実装のラッパーであり、ラップ対象のライセンスに準拠します。

| プラグイン | ラップ対象 | ライセンス |
| :-- | :-- | :-- |
| github | [github/github-mcp-server](https://github.com/github/github-mcp-server) | MIT |
| gitlab | [gitlab-org/cli (`glab mcp serve`)](https://gitlab.com/gitlab-org/cli) | MIT |

各プラグインは PATH 上のユーザー導入バイナリ / CLI に処理を委譲します（同梱・再配布なし）。
