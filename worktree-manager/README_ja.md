# worktree-manager — Git ワークツリー管理プラグイン

agy 向けに `git worktree` の操作（一覧取得、作成、削除、クリーンアップ）を安全かつ構造化して提供する MCP サーバーです。現在の作業ツリーで変更を stash することなく、並行ブランチ開発やサブエージェント用の隔離作業環境をスムーズに構築できます。

## 提供されるツール

### `worktree_list`
リポジトリ内のすべての worktree 一覧（パス、HEAD コミット、ブランチ名、lock 状態、メインツリー識別など）を取得します。

#### パラメータ
- `repo_path` (string, 任意): 対象リポジトリのパス（デフォルト: `.`）。

### `worktree_add`
新しい worktree を作成します。

#### パラメータ
- `path` (string, 必須): 作成先ディレクトリパス。
- `branch` (string, 任意): チェックアウトする既存ブランチ。
- `new_branch` (string, 任意): 新規作成するブランチ名 (`-b <new_branch>`)。
- `commit` (string, 任意): 分岐元のコミット・ブランチ・タグ。
- `repo_path` (string, 任意): 対象リポジトリのパス。

### `worktree_remove`
linked worktree を安全に削除します。誤ってメインの作業ツリーを削除しないよう保護されています。

#### パラメータ
- `path` (string, 必須): 削除対象の worktree パス。
- `force` (boolean, 任意, デフォルト: false): 未コミットの変更があっても強制削除するか。
- `repo_path` (string, 任意): 対象リポジトリのパス。

### `worktree_prune`
削除・移動された worktree の管理メタデータを整理（prune）します。

#### パラメータ
- `dry_run` (boolean, 任意, デフォルト: false): 実際には削除せず、対象の確認のみ行う。
- `repo_path` (string, 任意): 対象リポジトリのパス。

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/worktree-manager
```

## 同梱スキル

並行作業やサブエージェント委譲時の worktree 運用を案内する `skills/worktree-manager-gemini/SKILL.md` を同梱しています。
