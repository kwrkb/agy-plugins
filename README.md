# agy-plugins

このリポジトリは、AIアシスタント（agy CLI 等）向けの Model Context Protocol (MCP) プラグイン集です。

## 提供するプラグイン

| プラグイン | 説明 |
| :-- | :-- |
| **[github](./github/README.md)** | GitHub CLI (`gh`) を利用して GitHub の Issues / PR / リポジトリ等を操作する MCP サーバー（OS 共通） |
| **[gitlab](./gitlab/README.md)** | GitLab の Issues / MR / プロジェクト等を操作する MCP サーバー |
| **[agy-plugin-kit](./agy-plugin-kit/README.md)** | agy プラグイン開発メタ・ヘルパー（雛形生成・静的検査・Issue #390 パス修正・doc 生成） |
| **[ast-grep](./ast-grep/README.md)** | `ast-grep` (`sg`) を利用してコード構造の検索・安全なリファクタリングを行う MCP サーバー |

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
```

各プラグインの前提条件（PATH に入れるバイナリ / 認証設定）については、各ディレクトリの README を参照してください。

## 同梱スキル

`github` / `gitlab` プラグインは、エージェントが MCP ツールを正しく使うためのガイドを `skills/<name>/SKILL.md` として同梱しています（呼び出し時にロードされる知識。引数フォーマット・プロジェクト指定規約・頻出パターンを記載）。agy 1.0.10 でプロジェクトの `.agents/AGENTS.md` は注入されるようになりましたが、**プラグイン内 `rules/`・`plugin.json "rules"` は依然機能しない**ため（LESSONS #22/#35/#41）、プラグインからエージェントへ知識を渡す手段はこのスキルです。

## 動作要件

| プラグイン | 必要な CLI / バイナリ | 認証 |
| :-- | :-- | :-- |
| github | `gh`（PATH 上） | `gh auth login` 済み |
| gitlab | `glab` >= v1.74.0（PATH 上） | `glab auth login` 済み |
| agy-plugin-kit | （任意）`go` ※validator 再ビルド時のみ。`.exe` 同梱なので通常不要 | 不要 |
| ast-grep | `ast-grep`（CLI, PATH 上） | 不要 |

### 同梱プラットフォームと self-build

Go 製プラグイン（`github` / `ast-grep` / `retro-status` / `agy-plugin-kit` の validator）は **`linux/amd64` / `darwin/arm64`（Apple Silicon）/ `windows/amd64`** のネイティブバイナリを `bin/` に同梱しています。`bin/<name>` の OS 分岐 dispatcher が `uname` から `<name>-<goos>-<goarch>` を算出して exec します（Windows は agy が `.exe` を直接起動）。

**それ以外のプラットフォーム（例: `linux/arm64`〔Raspberry Pi・ARM サーバ〕/ `darwin/amd64`〔Intel Mac〕）は同梱しません**が、その実機上で native ビルドすれば dispatcher が自動で拾います（スクリプト編集不要）:

```sh
cd <plugin>/src && CGO_ENABLED=0 go build -o "../bin/<name>-$(go env GOOS)-$(go env GOARCH)" .
# 例: retro-status を ARM Linux で → retro-status/bin/retro-status-linux-arm64
```

## ライセンス・帰属

各プラグインは公式 MCP 実装のラッパーであり、ラップ対象のライセンスに準拠します。

| プラグイン | ラップ対象 | ライセンス |
| :-- | :-- | :-- |
| github | `gh` CLI | MIT |
| gitlab | [gitlab-org/cli (`glab mcp serve`)](https://gitlab.com/gitlab-org/cli) | MIT |
| ast-grep | [`ast-grep` CLI](https://ast-grep.github.io/) | MIT |

各プラグインは PATH 上のユーザー導入バイナリ / CLI に処理を委譲します（同梱・再配布なし）。
