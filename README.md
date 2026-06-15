# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

| プラグイン | 説明 |
| :-- | :-- |
| **[github-unix](./github-unix/README.md)** | GitHub の Issues / PR / リポジトリ等を操作する MCP サーバー（Linux / macOS） |
| **[github-windows](./github-windows/README.md)** | 同上の Windows 版（Go 製トークン解決ラッパーを `.exe` で同梱） |
| **[gitlab](./gitlab/README.md)** | GitLab の Issues / MR / プロジェクト等を操作する MCP サーバー |
| **[agy-plugin-kit](./agy-plugin-kit/README.md)** | agy プラグイン開発メタ・ヘルパー（雛形生成・静的検査・Issue #390 パス修正・doc 生成） |

> **github を OS 別に分けている理由**: 公式 `github-mcp-server` はトークンを環境変数からしか読まず、起動グルー（トークン解決ラッパー）が OS のシェルに依存するためです。`gh auth login` 済みなら手動のトークン設定なしで動きます。Linux / macOS は POSIX sh ラッパー、Windows は `.sh` を `command` に直接置けないため Go 製 `.exe` ラッパーを使います。

## インストール方法

```bash
# GitHub プラグイン（Linux / macOS）
agy plugin install https://github.com/kwrkb/agy-plugins/github-unix

# GitHub プラグイン（Windows）
agy plugin install https://github.com/kwrkb/agy-plugins/github-windows

# GitLab プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab

# agy プラグイン開発メタ・ヘルパー
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit
```

各プラグインの前提条件（PATH に入れるバイナリ / 認証設定）については、各ディレクトリの README を参照してください。

## 動作要件

| プラグイン | 必要な CLI / バイナリ | 認証 |
| :-- | :-- | :-- |
| github-unix | `github-mcp-server`（PATH 上） | `gh auth login` または `GITHUB_PERSONAL_ACCESS_TOKEN` 等の環境変数 |
| github-windows | `github-mcp-server`（PATH 上） | `gh auth login` または `GITHUB_PERSONAL_ACCESS_TOKEN` 等の環境変数 |
| gitlab | `glab` >= v1.74.0（PATH 上） | `glab auth login` 済み |
| agy-plugin-kit | （任意）`go` ※validator 再ビルド時のみ。`.exe` 同梱なので通常不要 | 不要 |

## ライセンス・帰属

各プラグインは公式 MCP 実装のラッパーであり、ラップ対象のライセンスに準拠します。

| プラグイン | ラップ対象 | ライセンス |
| :-- | :-- | :-- |
| github-unix | [github/github-mcp-server](https://github.com/github/github-mcp-server) | MIT |
| github-windows | [github/github-mcp-server](https://github.com/github/github-mcp-server) | MIT |
| gitlab | [gitlab-org/cli (`glab mcp serve`)](https://gitlab.com/gitlab-org/cli) | MIT |

各プラグインは PATH 上のユーザー導入バイナリ / CLI に処理を委譲します（同梱・再配布なし）。github プラグインはトークン解決用の薄いラッパースクリプトのみを同梱します。
