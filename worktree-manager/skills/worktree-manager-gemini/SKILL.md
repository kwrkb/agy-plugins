---
name: worktree-manager-gemini
description: Git ワークツリーの作成、一覧、削除、クリーンアップ（prune）を安全に行うスキル。サブエージェントでの並行作業やブランチ隔離開発で使用する。
---

# worktree-manager MCP プラグイン使用ガイド

このプラグインは、`git worktree` コマンドを安全に操作し、現在の作業ディレクトリを汚さずに複数ブランチの並行作業やサブエージェント用ワークスペースの準備・破棄を行うための MCP サーバーです。

## 提供されるツール

### 1. `worktree_list`
リポジトリ内のすべての worktree 一覧（パス、HEAD、ブランチ名、lock/prune 状態、メインツリー判定）を取得します。

- **引数**:
  - `repo_path` (string, 任意): 対象リポジトリのパス。未指定時はカレントディレクトリ。

### 2. `worktree_add`
新しい worktree を作成します。

- **引数**:
  - `path` (string, 必須): 作成先ディレクトリパス。
  - `branch` (string, 任意): チェックアウトする既存ブランチ。
  - `new_branch` (string, 任意): 新規作成するブランチ名 (`-b <new_branch>`)。
  - `commit` (string, 任意): ベースとするコミット・ブランチ・タグ。
  - `repo_path` (string, 任意): 対象リポジトリのパス。

### 3. `worktree_remove`
指定した linked worktree を安全に削除します。

- **引数**:
  - `path` (string, 必須): 削除する worktree のパス。
  - `force` (boolean, 任意, default=false): 未コミットの変更がある場合でも強制削除するか。
  - `repo_path` (string, 任意): 対象リポジトリのパス。
- **安全保護**: リポジトリのメイン作業ツリー（root）の削除要求は拒絶されます。

### 4. `worktree_prune`
削除済みまたは移動された worktree の管理情報をクリーンアップします。

- **引数**:
  - `dry_run` (boolean, 任意, default=false): 実際には削除せず、対象の確認のみ行うか。
  - `repo_path` (string, 任意): 対象リポジトリのパス。

## 典型的なワークフロー

1. **並行作業の準備**:
   - `worktree_add(path="../worktrees/feature-x", new_branch="feature-x")` で独立した作業ツリーを作成。
2. **作業と確認**:
   - `worktree_list()` で追加されたツリーを確認。
   - サブエージェントまたは別セッションで該当ディレクトリを指定して作業。
3. **作業終了後の後始末**:
   - 変更がコミットまたはマージされた後、`worktree_remove(path="../worktrees/feature-x")` でツリーを安全に破棄。
   - 不要なメタデータが残った場合は `worktree_prune()` を実行。
