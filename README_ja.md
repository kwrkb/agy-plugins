# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

| プラグイン | 説明 |
| :-- | :-- |
| **[github](./github/README_ja.md)** | GitHub CLI (`gh`) を利用して GitHub の Issues / PR / リポジトリ等を操作する MCP サーバー（OS 共通） |
| **[gitlab](./gitlab/README_ja.md)** | GitLab の Issues / MR / プロジェクト等を操作する MCP サーバー |
| **[agy-plugin-kit](./agy-plugin-kit/README_ja.md)** | agy プラグイン開発メタ・ヘルパー（雛形生成・静的検査・Issue #390 パス修正・doc 生成） |
| **[ast-grep](./ast-grep/README_ja.md)** | `ast-grep` (`sg`) を利用してコード構造の検索・安全なリファクタリングを行う MCP サーバー |
| **[go-lsp](./go-lsp/README_ja.md)** | `gopls` を利用して Go 言語の定義ジャンプ・参照検索・ホバー情報を取得する MCP サーバー |
| **[retro-status](./retro-status/README_ja.md)** | リポジトリをスキャンし、RPGのレトロステータス画面風のAA（アスキーアート）で出力する MCP サーバー |
| **[settings-advisor](./settings-advisor/README_ja.md)** | ワークスペースの規模・言語・機密/CI/本番設定を解析し、最適なモデル・サンドボックス・許可モードを控えめに提案する MCP サーバー |
| **[worktree-manager](./worktree-manager/README_ja.md)** | `git worktree` の安全な操作（一覧・作成・削除・クリーンアップ）を提供し、ブランチ隔離や並行作業を支援する MCP サーバー |

## インストール方法

```bash
# GitHub プラグイン (Cross-Platform)
agy plugin install https://github.com/kwrkb/agy-plugins/github

# GitLab プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab

# agy プラグイン開発メタ・ヘルパー
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit

# ast-grep プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/ast-grep

# go-lsp プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/go-lsp

# retro-status プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/retro-status

# settings-advisor プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/settings-advisor

# worktree-manager プラグイン
agy plugin install https://github.com/kwrkb/agy-plugins/worktree-manager
```

各プラグインの前提条件（PATH に入れるバイナリ / 認証設定）については、各ディレクトリの README を参照してください。

## 同梱スキル

各プラグインは、エージェントが MCP ツールを正しく使うためのガイドを `skills/<name>/SKILL.md` として同梱しています（呼び出し時にロードされる知識。引数フォーマット・プロジェクト指定規約・頻出パターンを記載）。agy 1.0.10 でプロジェクトの `.agents/AGENTS.md` は注入されるようになりましたが、**プラグイン内 `rules/`・`plugin.json "rules"` は依然機能しない**ため（LESSONS #22/#35/#41）、プラグインからエージェントへ知識を渡す手段はこのスキルです。

## 動作要件

| プラグイン | 必要な CLI / バイナリ | 認証 |
| :-- | :-- | :-- |
| github | `gh`（PATH 上） | `gh auth login` 済み |
| gitlab | `glab` >= v1.74.0（PATH 上） | `glab auth login` 済み |
| agy-plugin-kit | （任意）`go` ※validator 再ビルド時のみ。バイナリ同梱のため通常不要 | 不要 |
| ast-grep | `ast-grep`（CLI, PATH 上） | 不要 |
| go-lsp | `gopls`（PATH 上） | 不要 |
| retro-status | `git`（推奨）、`rg`（任意） | 不要 |
| settings-advisor | 不要（バイナリ同梱） | 不要 |
| worktree-manager | `git`（PATH 上） | 不要 |

### 同梱プラットフォームと self-build

✅ **全プラグインで Windows ネイティブ動作確認済み**

Go 製プラグイン（`github`、`ast-grep`、`go-lsp`、`retro-status`、`settings-advisor`、`worktree-manager`、および `agy-plugin-kit` の validator）は、**`linux/amd64`**、**`darwin/arm64`（Apple Silicon）**、**`windows/amd64`** のネイティブバイナリを `bin/` に同梱しています。拡張子なしの `bin/<name>` ディスパッチャスクリプトが `uname` で実行環境を判別し、`<name>-<goos>-<goarch>` を起動します（Windows では agy が `<name>.exe` を直接起動）。

**それ以外の環境（`linux/arm64` や `darwin/amd64` など）は標準同梱していません**が、スクリプトの書き換えなしで手元でビルドして動かせます：

```sh
cd <plugin>/src && CGO_ENABLED=0 go build -o "../bin/<name>-$(go env GOOS)-$(go env GOARCH)" .
# 例: ARM Linux 向けに retro-status をビルド -> retro-status/bin/retro-status-linux-arm64
```

## ライセンス・帰属

各プラグインは既存のツール・サーバーのラッパーとして動作し、それぞれのライセンスに従います：

| プラグイン | ラップ対象 | ライセンス |
| :-- | :-- | :-- |
| github | `gh` CLI | MIT |
| gitlab | [gitlab-org/cli (`glab mcp serve`)](https://gitlab.com/gitlab-org/cli) | MIT |
| ast-grep | [`ast-grep` CLI](https://ast-grep.github.io/) | MIT |
| go-lsp | [`gopls` (Go Language Server)](https://pkg.go.dev/golang.org/x/tools/gopls) | BSD-3-Clause |
| retro-status | 独自 Go 実装 | MIT |
| settings-advisor | 独自 Go 実装 | MIT |
| agy-plugin-kit | 独自 Go 実装 | MIT |
| worktree-manager | `git` CLI | MIT |

各プラグインはユーザーの PATH 上にあるインストール済みバイナリ/CLI に処理を委譲します（外部バイナリの再配布は行っていません）。
