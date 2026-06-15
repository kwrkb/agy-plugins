---
name: gitlab
description: GitLab の issue/MR/CI/repo を glab_* MCP ツール群で操作する際のルールと頻出パターン。agy 経由で GitLab 操作を行う前に参照する。
---

# GitLab MCP プラグイン（glab_* ツール群）使用ガイド

## ツール概要

このプラグインは `glab mcp serve` を通じて**約190個の MCP ツール**を提供する（バージョン依存で変動）。

ツール名は `glab_<command>_<subcommand>` 形式:
- `glab_issue_list` / `glab_issue_create` / `glab_issue_view`
- `glab_mr_list` / `glab_mr_create` / `glab_mr_merge`
- `glab_ci_list` / `glab_ci_status` / `glab_ci_trace`
- `glab_repo_view` / `glab_repo_list`

ツール一覧はエージェントのツールリストに既にある。**パラメータは各ツールのスキーマを確認して使う**（バージョンにより変わる可能性があるため）。

## 呼び出し形式（最重要）

各ツールは以下の共通パラメータを持つ:

```
glab_<cmd>_<subcmd>(
  args=[...],      # positional arguments（<id> 等）
  flags={...},     # コマンドフラグ（--repo, --title 等）
  limit=50,        # レスポンスサイズ上限（オプション）
  offset=0         # ページネーションオフセット（オプション）
)
```

## プロジェクトの指定方法

`repo` フラグの有無は**ツール単位**で異なる（「list 系なら一律あり」ではない）。使う前にスキーマで確認すること:

| ツール | `flags.repo` | 動作 |
|---|---|---|
| `glab_issue_list`, `glab_mr_list` | あり（string） | `flags.repo` で明示指定できる |
| `glab_ci_list` | **なし** | CWD 依存（※）。`ref` 等で絞り込むがプロジェクト指定はできない |
| view 系（`glab_issue_view`, `glab_mr_view`, `glab_mr_diff`） | なし | CWD 依存（※） |
| write 系（`glab_issue_create`, `glab_mr_create` 等） | なし | CWD 依存（※） |

**※ CWD 依存の制限**: `repo` フラグがないツールは、`glab mcp serve` が起動された CWD の git remote からプロジェクトを検出する。agy が GitLab リモートのないディレクトリで起動した場合は「Could not determine base repository」エラーになる。`flags.repo` を渡してもスキーマにないツールでは無視される。

```
# list 系は flags.repo で明示指定（推奨）
glab_issue_list(flags={"repo": "myorg/myapp"}, limit=20)

# view/write 系は repo 指定不可（CWD の git remote に依存）
glab_issue_view(args=["42"])
glab_issue_create(flags={"title": "Bug: login fails", "description": "Steps: ..."})
```

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
glab_issue_list(flags={"repo": "myorg/myapp"}, limit=20)

# イシュー一覧（クローズ済み）
glab_issue_list(flags={"repo": "myorg/myapp", "closed": true}, limit=20)

# イシュー詳細（repo 指定不可 → CWD の git remote から検出）
glab_issue_view(args=["42"])

# イシュー作成（repo 指定不可 → CWD 依存）
glab_issue_create(flags={"title": "Bug: login fails", "description": "Steps: ..."})

# MR 一覧（自分担当）。mr_list の assignee は配列（issue_list は string）
glab_mr_list(flags={"repo": "myorg/myapp", "assignee": ["@me"]}, limit=20)

# MR 作成（repo 指定不可 → CWD 依存）
glab_mr_create(flags={"source_branch": "feature/oauth", "target_branch": "main", "title": "Add OAuth support"})

# パイプライン一覧（ci_list に repo フラグはない → CWD 依存）
glab_ci_list(flags={"ref": "main"}, limit=10)

# ジョブログ確認（job_id は positional）
glab_ci_trace(args=["12345"])

# リポジトリ確認（group/project は positional）
glab_repo_view(args=["myorg/myapp"])
```

## EXPERIMENTAL について

`glab mcp serve` は GitLab により EXPERIMENTAL と位置づけられている。ツールの仕様・引数名・ツール数はバージョンアップで変更される可能性がある。パラメータ名は必ずツールスキーマ（エージェントのツールリスト）を参照する。
