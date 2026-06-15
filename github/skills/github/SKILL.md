---
name: github
description: GitHub の issue/PR/repo/release/search を gh_command MCP ツールで操作する際のルールと頻出パターン。agy 経由で GitHub 操作を行う前に参照する。
---

# GitHub MCP プラグイン（gh_command）使用ガイド

## ツール概要

このプラグインは `gh_command` という**単一の MCP ツール**を提供する。`gh` CLI の任意のサブコマンドを実行できる。

```
gh_command(args=["<subcommand>", "<args...>"])
```

## args 配列のルール（最重要）

- `args` は `["issue", "list", "--limit", "10"]` のようにトークンを**1要素ずつ**渡す配列
- `"gh"` 自体は含めない
- スペースを含む値（タイトル・本文・検索クエリ）は**1要素にまとめる**:
  - 正: `["pr", "create", "--title", "Fix the bug in login flow"]`
  - 誤: `["pr", "create", "--title", "Fix", "the", "bug"]`（gh が複数引数として受け取る）

## CWD 非定常 — リポジトリは必ず明示する

`gh_command` は MCP サーバーが agy に起動されたディレクトリを CWD として継承する。暗黙のカレントリポジトリ検出（`gh issue list` だけ等）は**信頼できない**。

**常に `-R OWNER/REPO` を指定する:**

```
gh_command(args=["issue", "list", "-R", "org/repo", "--limit", "20"])
gh_command(args=["pr", "view", "42", "-R", "org/repo"])
```

## --json フラグの活用

機械処理しやすい JSON 出力には `--json` でフィールド名を明示する（`--json` だけではエラー）:

```
gh_command(args=["issue", "list", "-R", "org/repo", "--json", "number,title,state,labels", "--limit", "30"])
gh_command(args=["pr", "view", "123", "-R", "org/repo", "--json", "number,title,body,state,reviews"])
```

`-q` / `--jq` と組み合わせてフィルタも可能:

```
gh_command(args=["issue", "list", "-R", "org/repo", "--json", "number,title", "-q", ".[].title"])
```

## 頻出パターン

### issue

```
# 一覧（JSON）
gh_command(args=["issue", "list", "-R", "org/repo", "--json", "number,title,state,assignees", "--limit", "20"])

# 作成
gh_command(args=["issue", "create", "-R", "org/repo", "--title", "Bug: login fails on mobile", "--body", "Steps to reproduce: ..."])

# 表示
gh_command(args=["issue", "view", "42", "-R", "org/repo", "--json", "number,title,body,comments"])

# クローズ
gh_command(args=["issue", "close", "42", "-R", "org/repo"])

# 検索（スペース含むクエリは1要素）
gh_command(args=["search", "issues", "auth error is:open", "-R", "org/repo", "--json", "number,title,repository"])
```

### pr

```
# 一覧
gh_command(args=["pr", "list", "-R", "org/repo", "--json", "number,title,state,author,headRefName", "--limit", "20"])

# 作成
gh_command(args=["pr", "create", "-R", "org/repo", "--title", "Add OAuth support", "--body", "Closes #123", "--base", "main"])

# diff 確認
gh_command(args=["pr", "diff", "42", "-R", "org/repo"])

# 承認
gh_command(args=["pr", "review", "42", "-R", "org/repo", "--approve"])

# マージ
gh_command(args=["pr", "merge", "42", "-R", "org/repo", "--squash"])
```

### repo

```
# リポジトリ情報
gh_command(args=["repo", "view", "org/repo", "--json", "name,description,defaultBranchRef,isPrivate"])

# フォーク
gh_command(args=["repo", "fork", "upstream/repo"])
```

### search

```
# リポジトリ検索（クエリは1要素）
gh_command(args=["search", "repos", "mark3labs mcp-go", "--limit", "5", "--json", "name,fullName,description"])

# PR 横断検索
gh_command(args=["search", "prs", "is:open review-requested:@me", "--json", "number,title,repository"])
```

### release

```
# 一覧
gh_command(args=["release", "list", "-R", "org/repo", "--json", "name,tagName,publishedAt", "--limit", "10"])

# 作成
gh_command(args=["release", "create", "v1.2.0", "-R", "org/repo", "--title", "v1.2.0", "--notes", "Release notes here"])
```

### api（任意の GitHub REST / GraphQL 呼び出し）

```
# REST: コントリビューター一覧
gh_command(args=["api", "repos/org/repo/contributors", "--jq", ".[].login"])

# GraphQL
gh_command(args=["api", "graphql", "-f", "query={viewer{login}}"])
```

### Actions

```
# ワークフロー一覧
gh_command(args=["workflow", "list", "-R", "org/repo"])

# 最新 run の確認
gh_command(args=["run", "list", "-R", "org/repo", "--limit", "5", "--json", "status,conclusion,workflowName,createdAt"])
```

## 主要サブコマンド一覧

| カテゴリ | サブコマンド |
|---|---|
| コア | `issue` / `pr` / `repo` / `release` |
| Actions | `run` / `workflow` / `cache` |
| 検索 | `search repos\|issues\|prs\|commits\|code` |
| 低レベル | `api` |
| その他 | `gist` / `org` / `project` / `label` / `secret` / `variable` |

## 注意: 破壊的操作

`gh_command` は `gh` の**あらゆるサブコマンドを実行できる**。`gh pr merge` / `gh issue close` / `gh repo delete` / `gh api`（任意 REST 呼び出し）といった**書き込み・破壊的操作も実行可能**で、`gh auth login` 済みアカウントの権限で動く。実行前に必ず意図を確認する。
