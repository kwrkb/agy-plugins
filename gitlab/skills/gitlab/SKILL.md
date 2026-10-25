---
name: gitlab
description: GitLab の issue/MR/CI/release/project を glab_command MCP ツールで操作する際のルールと頻出パターン。agy 経由で GitLab 操作を行う前に参照する。
---

# GitLab MCP プラグイン（glab_command）使用ガイド

## ツール概要

このプラグインは `glab_command` という**単一の MCP ツール**を提供する。`glab` CLI の任意のサブコマンドを実行できる。

```
glab_command(args=["<subcommand>", "<args...>"])
```

成功時は glab の stdout が、失敗時は終了コードと stderr（` ERROR ` の見出し付き）がエラーとして返る。

## args 配列のルール（最重要）

- `args` は `["issue", "list", "-R", "group/project"]` のようにトークンを**1要素ずつ**渡す配列
- `"glab"` 自体は含めない
- スペースを含む値（タイトル・説明・検索クエリ）は**1要素にまとめる**:
  - 正: `["mr", "create", "-R", "group/project", "--title", "Fix the bug in login flow", ...]`
  - 誤: `["mr", "create", "--title", "Fix", "the", "bug"]`（glab が複数引数として受け取る）

## 出力上限とタイムアウト

- **出力の上限**: stdout は先頭 64 KiB、stderr は先頭 16 KiB だけを返す。打ち切ったときは末尾に
  `[output truncated: kept the first N of M bytes]` の行が付く（N は返した分、M は glab が出した全体）。
  この行があれば結果は**途中まで**なので、`--per-page` / `--jq` / `api` の `per_page` などで絞って呼び直す
- **タイムアウト**: 既定で 300 秒たつと glab を（子プロセスごと）止め、エラーで返す。エラーには
  `timed out after ...` と、それまでの stdout / stderr が入る
- **流し続けるコマンド**（実行中のジョブへの `ci trace` など）はジョブが終わるまで返らない。`timeout_seconds`（1〜3600 の整数）で期限を延ばす:

```
glab_command(args=["ci", "trace", "987654", "-R", "myorg/myapp"], timeout_seconds=1800)
```

## CWD 非定常 — プロジェクトは必ず明示する

`glab_command` は MCP サーバーが agy に起動されたディレクトリを CWD として継承する。暗黙のカレントリポジトリ検出は**信頼できない**
（GitLab リモートのないディレクトリでは `not a git repository` などで失敗し、別のリポジトリにいれば**そちらが操作対象になる**）。

**常に `-R group/project` を指定する**（`group/sub/project`、`https://gitlab.com/group/project` も可）:

```
glab_command(args=["issue", "list", "-R", "myorg/myapp", "--output", "json"])
glab_command(args=["mr", "view", "42", "-R", "myorg/myapp"])
```

`-R` が使えるのは issue / mr / ci / release / label / variable / schedule など**プロジェクト単位のコマンド**。例外:

| コマンド | プロジェクトの渡し方 |
|---|---|
| `repo view` / `repo list` / `repo search` など `repo` の大半 | `-R` なし。`repo view` は位置引数 `["repo", "view", "myorg/myapp"]` |
| `milestone list` | `-R` は効かず `--project myorg/myapp`（グループなら `--group myorg`） |
| `api` | `-R` なし。パスにプロジェクトを直接書く（下記） |
| `auth` / `user` / `todo` / `alias` / `runner-controller` / `skills` | プロジェクトに属さない（`user events` は `--all` を付けないと CWD のリポジトリを要求する） |

## 使わないコマンド

- **ローカルの git 作業ツリーを前提にするもの**: `mr checkout` / `repo clone` / `mr create --fill`（`--push`）/ `stack` など。
  `-R` を付けても CWD の git を見て失敗する。git 操作はホストエージェントの責務で、このツールの範囲外。
- **ID を省くと現在のブランチから探すもの**: `ci view` / `ci trace` / `ci status` は、パイプライン ID・ジョブ ID か `--branch` を必ず渡す
  （省くと CWD のブランチを使う。`ci view` は TUI なので使わず、`ci get` / `ci list` を使う）。
- **`duo` / `orbit`**: 初回に外部バイナリのダウンロード確認を求める。このツールでは応答できないので使わない。
- **エディタを開く値**: `--description "-"` などは使わない。値は引数で直接渡す。

対話プロンプトは無効にしてある（`GLAB_NO_PROMPT=1`・stdin なし）。必須フラグが足りないと、待たずに
`required for non-interactive mode` のエラーで返るので、足りないフラグを追加して呼び直す。

## JSON 出力

JSON を出せるのは**一部のコマンドだけ**（list / view / get 系が中心）。対応コマンドでは長い形の `--output json` を明示する
（短い形は `-F` と `-O` がコマンドで違う）。`--jq` でフィルタもできる。対応していないコマンド（`mr diff`・`ci trace`・`issue note` など）はテキストで返る。
付けられるかは `glab_command(args=["<cmd>", "<sub>", "--help"])` の FLAGS に `--output` があるかで確かめる。

```
glab_command(args=["issue", "list", "-R", "myorg/myapp", "--output", "json", "--per-page", "20"])
glab_command(args=["mr", "list", "-R", "myorg/myapp", "--output", "json", "--jq", ".[] | {iid, title}"])
```

## 頻出パターン

### issue

```
# 一覧（既定はオープンのみ。--all / --closed で切り替え）
glab_command(args=["issue", "list", "-R", "myorg/myapp", "--output", "json", "--per-page", "20"])

# 検索（スペースを含むクエリは1要素）
glab_command(args=["issue", "list", "-R", "myorg/myapp", "--search", "login error", "--output", "json"])

# 表示（コメント込み）
glab_command(args=["issue", "view", "42", "-R", "myorg/myapp", "--comments"])

# 作成
glab_command(args=["issue", "create", "-R", "myorg/myapp", "--title", "Bug: login fails on mobile", "--description", "Steps to reproduce: ...", "--yes"])

# コメント
glab_command(args=["issue", "note", "42", "-R", "myorg/myapp", "--message", "Fixed in !17"])

# クローズ
glab_command(args=["issue", "close", "42", "-R", "myorg/myapp"])
```

### mr

```
# 一覧（自分担当）
glab_command(args=["mr", "list", "-R", "myorg/myapp", "--assignee=@me", "--output", "json"])

# 表示
glab_command(args=["mr", "view", "17", "-R", "myorg/myapp", "--output", "json"])

# diff
glab_command(args=["mr", "diff", "17", "-R", "myorg/myapp"])

# 作成（ブランチは push 済みであること。ローカルクローンなしで作るので全項目をフラグで渡す）
glab_command(args=["mr", "create", "-R", "myorg/myapp", "--source-branch", "feature/oauth", "--target-branch", "main", "--title", "Add OAuth support", "--description", "Closes #42", "--yes"])

# 承認・マージ
glab_command(args=["mr", "approve", "17", "-R", "myorg/myapp"])
glab_command(args=["mr", "merge", "17", "-R", "myorg/myapp", "--squash", "--yes"])
```

### CI/CD

```
# パイプライン一覧
glab_command(args=["ci", "list", "-R", "myorg/myapp", "--output", "json", "--per-page", "10"])

# パイプライン詳細（ID を明示）
glab_command(args=["ci", "get", "-R", "myorg/myapp", "--pipeline-id", "123456", "--output", "json"])

# ジョブログ（ジョブ ID を明示）
glab_command(args=["ci", "trace", "987654", "-R", "myorg/myapp"])
```

### release / repo

```
glab_command(args=["release", "list", "-R", "myorg/myapp", "--output", "json"])
glab_command(args=["repo", "view", "myorg/myapp", "--output", "json"])
```

### api（任意の GitLab REST API v4 呼び出し）

`:fullpath` / `:id` などのプレースホルダは CWD のリポジトリから解決されるので使わない。
プロジェクトは **URL エンコードしたパス**（`/` → `%2F`）か数値 ID で直接書く。

```
glab_command(args=["api", "projects/myorg%2Fmyapp"])
glab_command(args=["api", "projects/myorg%2Fmyapp/issues?state=opened&per_page=5"])
```

## 主要サブコマンド一覧

| カテゴリ | サブコマンド |
|---|---|
| コア | `issue` / `mr` / `release` / `repo` |
| CI/CD | `ci` / `job` / `schedule` / `variable` |
| プロジェクト管理 | `label` / `milestone` / `iteration` / `incident` / `work-items` |
| 低レベル | `api` |

## 注意: 破壊的操作

`glab_command` は `glab` の**あらゆるサブコマンドを実行できる**。`mr merge` / `issue close` / `repo delete` / `variable set` /
`api`（任意の REST 呼び出し）といった**書き込み・破壊的操作も実行可能**で、`glab auth login` 済みアカウントの権限で動く。
`--yes` は確認プロンプトを飛ばすためのもので、実行前の意図の確認は呼び出し側で行う。
