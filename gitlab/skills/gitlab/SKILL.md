---
name: gitlab
description: GitLab の issue/MR/CI/repo を glab_* MCP ツール群で操作する際のルールと頻出パターン。agy 経由で GitLab 操作を行う前に参照する。
---

# GitLab MCP プラグイン（glab_* ツール群）使用ガイド

## ツール概要

このプラグインは `glab mcp serve` を通じて**約220個の MCP ツール**を提供する（glab v1.124.0 で 217。バージョン依存で変動）。

ツール名は `glab_<command>_<subcommand>` 形式:
- `glab_issue_list` / `glab_issue_create` / `glab_issue_view`
- `glab_mr_list` / `glab_mr_create` / `glab_mr_merge`
- `glab_ci_list` / `glab_ci_status` / `glab_ci_trace`
- `glab_repo_view` / `glab_repo_list`

ツール一覧はエージェントのツールリストに既にある。**パラメータは各ツールのスキーマを確認して使う**（バージョンにより変わる可能性があるため）。

## 呼び出し形式（最重要）

各ツールのスキーマは `args` と `flags` の 2 つ:

```
glab_<cmd>_<subcmd>(
  args=[...],      # positional arguments（<id> 等）
  flags={...}      # コマンドフラグ（repo, title, per_page 等）
)
```

### レスポンスサイズとページング

- レスポンスは **50,000 文字で打ち切られる**。打ち切られた JSON は断片になりパースできない
- list 系は `flags` の **`per_page` / `page` / `jq`** で最初から絞る（JSON 出力時の推奨）
- `limit` / `offset`（スキーマには出ない）は**プレーンテキスト出力を続きから読む**ためのもの。打ち切られると `next_offset` 付きで「`offset` を指定して再呼び出し」と返る

```
glab_issue_list(flags={"repo": "myorg/myapp", "per_page": 20})
glab_mr_list(flags={"repo": "myorg/myapp", "jq": ".[] | {iid, title}"})
```

## プロジェクトの指定方法

issue / mr / ci 系は view・write を含め **`flags.repo`（string）でプロジェクトを明示指定できる**（glab v1.117.0 以降、継承フラグがスキーマに載るようになった）。
**常に `flags.repo` を渡す**: 省略すると `glab mcp serve` が起動された CWD の git remote から検出し、agy が GitLab リモートのないディレクトリで起動していると「Could not determine base repository」エラーになる。

```
glab_issue_view(args=["42"], flags={"repo": "myorg/myapp"})
glab_issue_create(flags={"repo": "myorg/myapp", "title": "Bug: login fails", "description": "Steps: ..."})
```

`flags.repo` を**持たない**ツールもある（`glab_repo_*` の大半〈`view` / `list` / `create` / `search` など〉・`glab_api`・`glab_auth_*`・`glab_user_*`・`glab_todo_*` など）。
`glab_repo_view` はプロジェクトを `args=["group/project"]` で渡す。スキーマにないフラグは無視されるので、使う前にスキーマで確認すること。

プロジェクトパスの形式は `group/project`（または `group/sub/project`）。

## カテゴリ別主要ツール

### Issues (`glab_issue_*`)

| ツール | 説明 |
|---|---|
| `glab_issue_list` | イシュー一覧 |
| `glab_issue_create` | イシュー作成 |
| `glab_issue_view` | イシュー詳細（`args=["<id>"]`） |
| `glab_issue_update` | ラベル・マイルストーン等の更新 |
| `glab_issue_close` | イシューをクローズ |
| `glab_issue_reopen` | イシューを再オープン |
| `glab_issue_note` | コメント追加 |

### Merge Requests (`glab_mr_*`)

| ツール | 説明 |
|---|---|
| `glab_mr_list` | MR 一覧 |
| `glab_mr_create` | MR 作成 |
| `glab_mr_view` | MR 詳細（`args=["<id>"]`） |
| `glab_mr_update` | MR 属性更新 |
| `glab_mr_merge` | MR をマージ |
| `glab_mr_approve` | MR を承認 |
| `glab_mr_diff` | MR の変更差分表示 |
| `glab_mr_note` | コメント追加 |

### CI/CD (`glab_ci_*`)

| ツール | 説明 |
|---|---|
| `glab_ci_list` | パイプライン一覧 |
| `glab_ci_get` | パイプライン詳細 |
| `glab_ci_status` | パイプラインステータス |
| `glab_ci_run` | パイプライン手動実行 |
| `glab_ci_retry` | リトライ |
| `glab_ci_trace` | ジョブログ追跡（`args=["<job-id>"]`） |
| `glab_ci_cancel_pipeline` | パイプラインキャンセル |

### Projects (`glab_repo_*`)

| ツール | 説明 |
|---|---|
| `glab_repo_view` | プロジェクト詳細（`args=["group/project"]`） |
| `glab_repo_list` | プロジェクト一覧 |
| `glab_repo_create` | 新規プロジェクト作成 |
| `glab_repo_search` | プロジェクト検索 |

### その他の主要カテゴリ

- `glab_release_*`: リリース管理
- `glab_label_*`: ラベル管理
- `glab_milestone_*`: マイルストーン管理
- `glab_variable_*`: CI/CD 変数管理
- `glab_schedule_*`: パイプラインスケジュール
- `glab_api`: GitLab REST API v4 直接呼び出し

## 頻出パターン

```
# イシュー一覧（オープン、デフォルト）
glab_issue_list(flags={"repo": "myorg/myapp", "per_page": 20})

# イシュー一覧（クローズ済み）
glab_issue_list(flags={"repo": "myorg/myapp", "closed": true, "per_page": 20})

# イシュー詳細
glab_issue_view(args=["42"], flags={"repo": "myorg/myapp"})

# イシュー作成
glab_issue_create(flags={"repo": "myorg/myapp", "title": "Bug: login fails", "description": "Steps: ..."})

# MR 一覧（自分担当）。assignee の型はツールで違う:
#   string = glab_issue_list / array = glab_mr_list・issue_create・mr_create・issue_update・mr_update
glab_mr_list(flags={"repo": "myorg/myapp", "assignee": ["@me"], "per_page": 20})

# MR 作成
glab_mr_create(flags={"repo": "myorg/myapp", "source_branch": "feature/oauth", "target_branch": "main", "title": "Add OAuth support"})

# パイプライン一覧
glab_ci_list(flags={"repo": "myorg/myapp", "ref": "main", "per_page": 10})

# ジョブログ確認（job_id は positional）
glab_ci_trace(args=["12345"], flags={"repo": "myorg/myapp"})

# リポジトリ確認（repo_view に flags.repo はない。group/project は positional）
glab_repo_view(args=["myorg/myapp"])
```

## EXPERIMENTAL / deprecated について

`glab mcp serve` は GitLab により EXPERIMENTAL と位置づけられ、**glab v1.124.0 で deprecated** になった（起動時に stderr へ警告が出るが動作は継続する。後継は GitLab 公式の MCP Server）。ツールの仕様・引数名・ツール数はバージョンアップで変更される可能性がある。パラメータ名は必ずツールスキーマ（エージェントのツールリスト）を参照する。
