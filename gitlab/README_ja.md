# GitLab MCP プラグイン（`glab mcp serve`）

このプラグインは、GitLab 公式 CLI [`glab`](https://gitlab.com/gitlab-org/cli) に内蔵された **`glab mcp serve`**（Model Context Protocol サーバー）を、Antigravity (`agy`) から利用できるようにするものです。

以前は自作の Go 製 MCP サーバーでしたが、`glab` 自身が MCP サーバー機能を持つようになったため、公式実装に置き換えました。自作サーバーの全機能（プロジェクト情報・イシュー・MR）を包含し、さらに CI/CD パイプライン・ジョブの操作が加わります。

> **Note**: `glab mcp serve` は GitLab により **EXPERIMENTAL** と位置づけられています。仕様が変更・削除される可能性があります。
> さらに **glab v1.124.0 で deprecated** となり、後継として [GitLab MCP Server](https://docs.gitlab.com/user/model_context_protocol/mcp_server/) が案内されています。現時点では動作します（agy 1.4.2 でサーバー起動と全 217 ツールの探索を確認済み。起動時に stderr へ deprecated 警告が出るのみ）。

## 提供する機能（MCPツール）

`glab mcp serve` は `glab` の各サブコマンドを MCP ツールとして公開します（glab v1.124.0 の実測で 217 ツール。`glab_<command>_<subcommand>` 形式。具体的なツール名・引数は `glab` のバージョンに依存します）。主なカテゴリ:

- **Issues** (`glab_issue_*`): 一覧・作成・更新・クローズ・ノート追加・view など
- **Merge Requests** (`glab_mr_*`): 一覧・作成・更新・マージ・diff・approve・ノート操作 など
- **Projects / Repo** (`glab_repo_*`): 一覧・view・create・clone・search など
- **CI/CD** (`glab_ci_*`, `glab_job_*`): パイプライン実行・status・trace・job artifact など
- その他: releases / labels / milestones / variables / schedules など glab の広範なコマンド群

> **引数の差分（旧自作サーバーから）**: 旧サーバーは `owner` / `repo` を引数に取っていましたが、`glab mcp serve` のツールは glab 流のプロジェクトパス（例: `group/project`）系の指定になります。

## 同梱スキル

`skills/gitlab/SKILL.md` にエージェント向けの使用ガイドを同梱しています（呼び出し時にオンデマンドでロードされる知識）。`glab_*` ツール共通の呼び出し形式（`args` / `flags`）、レスポンスサイズ上限とページング（`per_page` / `page` / `jq`）、プロジェクト指定（`flags.repo` を常に渡す・持たないツール・`assignee` 型がツール単位で異なる点）、カテゴリ別ツールマップ、頻出パターンを記載しています。常時コンテキスト予算を消費せず必要時のみ読み込ませるため、スキルとして提供しています。

## 必要条件

- **`glab` >= v1.74.0**（`mcp serve` サブコマンドを含むバージョン。**推奨は v1.117.0 以降**: issue / MR / CI の view・write 系ツールでも `flags.repo` が使えるようになり、同梱スキルはこれを前提にしています。v1.124.0 で動作確認済み）
  - apt 版（Ubuntu universe）の `glab` は古く（例: 1.53.0）`mcp` 非対応です。次のように最新版を導入してください:
    ```bash
    # 必要なら apt 版を削除
    sudo apt remove glab
    # 最新の glab を導入（~/go/bin が PATH 上にあること）
    go install gitlab.com/gitlab-org/cli/cmd/glab@latest
    ```
  - 導入後の確認:
    ```bash
    which glab            # ~/go/bin/glab を指すこと
    glab mcp serve --help # ヘルプが表示されること（mcp serve 対応の確認）
    ```
    `go install`（ldflags 未注入）でビルドした場合 `glab --version` は `DEV` と表示されますが、`mcp serve` 機能には影響しません。

## セットアップと認証

認証は `glab` 自身の設定（`~/.config/glab-cli/config.yml`）をそのまま再利用します。専用のトークン環境変数や wrapper は不要です。

```bash
glab auth login        # 未認証の場合
glab auth status       # 認証済みか確認
```

OAuth / 個人用アクセストークン（PAT）いずれも、`glab auth login` で構成済みであればそのまま利用されます。

## ビルド方法

ビルド不要です。システムにインストールされた `glab` を直接利用するため、`mcp_config.json` は PATH 上の `glab` を `glab mcp serve` で起動します。

## プラグインとしての登録

```bash
agy plugin install /path/to/agy-plugins/gitlab
```

インストールは絶対パスで指定してください。`mcp_config.json` の `mcpServers.glab` が `command: "glab"`, `args: ["mcp", "serve"]` を定義しており、`agy` は PATH 上の `glab` を解決して MCP サーバーを起動します。

## ライセンス・帰属

本プラグインは [gitlab-org/cli (`glab`)](https://gitlab.com/gitlab-org/cli)（MIT License）に内蔵された `glab mcp serve` を呼び出すラッパーです。`glab` バイナリはシステムに別途インストールされたものを使用し、再配布はしません。本プラグインのコードは公式実装に準じ MIT として扱います。
