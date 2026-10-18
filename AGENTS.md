# AGENTS.md

このファイルは、このリポジトリで作業する AI エージェント向けの実務ガイドです。目的・スコープと現在有効な設計判断は `VISION.md`（正典）、利用者向けの概要は `README.md`、過去の検証結果や agy 固有の罠は `LESSONS.md` と `plugin-gotchas.md` を参照してください。

ドキュメントが矛盾した時の優先順は **VISION.md > PLAN.md > コード > NOTES.md** です（`LESSONS.md` は序列外の記録）。コードが `VISION.md` と矛盾していたら、どちらかに寄せる前に作業を止めて確認してください。

## Repository Overview

`agy-plugins` は、agy CLI 向け MCP プラグインのモノレポです。

- `github/`: `gh` CLI を呼び出す Go 製 MCP サーバー
- `gitlab/`: `glab mcp serve` を利用する設定中心のプラグイン（Go ソースなし）
- `ast-grep/`: `ast-grep` CLI を呼び出す Go 製 MCP サーバー
- `go-lsp/`: `gopls` を利用する Go 製 MCP サーバー
- `retro-status/`: リポジトリ情報をレトロゲーム風に表示する Go 製 MCP サーバー
- `settings-advisor/`: ワークスペースに適した agy 設定を提案する Go 製 MCP サーバー
- `worktree-manager/`: Git worktree の作成・削除・一覧・prune を行う Go 製 MCP サーバー
- `test-runner/`: Go テストを実行し、失敗ログと再実行引数を返す Go 製 MCP サーバー
- `agy-plugin-kit/`: プラグイン作成用の command、skill、template、および Go 製 validator

Go 製プラグインは原則として `<plugin>/src/` に独立した Go モジュール、`<plugin>/bin/` に配布用バイナリを持ちます。validator のみ `agy-plugin-kit/validator/{src,bin}/` 配下です。

## Permissions

- `.gemini/settings.json`（`proceed-in-sandbox`）により、サンドボックス内の開発操作は自動承認。
- **自動実行可**: サンドボックス内でのファイル編集、ビルド（`./build.sh`）、テスト・検証（`go test`, `go vet`, `gofmt`）、ローカル Git 参照（`status`, `diff`, `log`）、同梱 MCP ツールのローカル解析。
- **要ユーザー承認**: 外部通信・リモート操作（`git push`, `gh`/`glab` による PR/MR 操作）、破壊的操作（`git reset --hard`, ワークツリー強制削除等）、ホームディレクトリ（`~/.gemini/` 等）への変更・キャッシュ操作、依存の新規追加。

## Before Editing

1. `git status --short` で既存のユーザー変更を確認し、巻き戻さない。
2. 変更対象の `README.md`、`src/main.go`、`src/main_test.go`、manifest、skill を読む。
3. agy の manifest、path、hook、rule、install 動作に触れる場合は、先に `LESSONS.md` と `plugin-gotchas.md` を検索する。
4. 構成の当たりを付ける時は `NOTES.md`（内部構造の索引。鮮度非保証なのでコードで裏を取る）を見る。`LESSONS.md` は履歴の記録で仕様ではない。

## Implementation Conventions

- 変更は対象プラグイン内に限定し、他プラグインへの横展開は必要性が確認できた場合だけ行う。
- Go コードは標準ライブラリと既存ヘルパーを優先し、新しい production dependency は追加前に確認する。
- 外部 CLI の引数は文字列結合や手作業の再分割を避け、`exec.CommandContext` へ引数配列として渡す。ユーザー入力パスや引数が `-` で始まる場合のオプション誤認を防ぐため、必要に応じて `--` デリミタを挟む。
- 外部 CLI 呼び出しでは、正常終了時でも stderr にレポートや診断メッセージを出力するツール（例: `git worktree prune -v`）があるため、必要に応じて stderr も捕捉して利用者に返す。
- MCP の stdout はプロトコル専用に保つ。診断ログは stderr へ出す。
- OS パスを扱う処理では Linux、macOS、Windows の区切り文字と実行形式を考慮する。パス比較で不用意に大文字小文字を無視（case-insensitive）せず、プラットフォームに応じた比較を行う。
- ユーザー入力からファイルを読む、コマンドを実行する、または destructive な CLI 操作を公開する変更には、境界チェック、タイムアウト、説明を追加する。
- 挙動変更には、正常系だけでなく引数境界、パス、キャンセル、エラー処理など変更リスクに対応するテストを加える。

## Plugin Packaging Rules

マニフェスト形式・dispatcher・バイナリのコミット配布・skills 主体・フックの書き方は `VISION.md`「現在有効な設計判断」に従う。作業時の注意:

- dispatcher（拡張子なし `bin/<name>`）の executable bit を維持する。
- Go ソースを変えたら、配布対象 `bin/<name>-linux-amd64`・`bin/<name>-darwin-arm64`・`bin/<name>.exe` を再ビルドしてコミットする。
- skill を新規作成する場合は、対象 AI の接尾辞を skill 名、ディレクトリ名、frontmatter の `name:` で一致させる。Codex 用は `-codex`。
- hook command は既存の `agy-plugin-kit/hooks.json` の書き方（変数を使わず、Windows 形 `||` Unix 形の相対パス）に従う。

## Build and Test

対象 Go モジュールのディレクトリで、最低限次を実行します。

```sh
cd <plugin>/src
gofmt -w <changed-go-files>
go vet ./...
go test ./...
```

validator の場合は `agy-plugin-kit/validator/src` を使います。全 Go モジュールを確認する場合は、以下の各ディレクトリで `go vet ./...` と `go test ./...` を実行します。

```text
agy-plugin-kit/validator/src
ast-grep/src
github/src
go-lsp/src
retro-status/src
settings-advisor/src
worktree-manager/src
test-runner/src
```

Go ソースを変更したら、リポジトリルートで対象を決定論的に再ビルドします。

```sh
./build.sh <github|validator|ast-grep|go-lsp|retro-status|settings-advisor|worktree-manager|test-runner>
```

Windows PowerShell では同じ target を `./build.ps1` に渡します。決定論ビルドが前提とする Go のパッチ版は `.go-version` に定義され、`build.sh`/`build.ps1` が `GOTOOLCHAIN` で強制します（未取得なら Go が自動ダウンロード）。素の `go build` で焼いたバイナリはローカル Go 版によって別物になるため、必ずビルドスクリプト経由で再ビルドしてください。`.go-version` を独断で書き換えないこと。引き上げ条件は VISION.md「Go バージョンの引き上げ方針」に定義され、条件1（固定中のマイナー内のパッチで直る stdlib 脆弱性）は CI の govulncheck が fail で通知します。

## Verification and Review

- `git diff --check` で whitespace error を確認する。
- `git diff --stat` と `git diff -- <changed-files>` で、意図しない文書・manifest・バイナリ差分がないか確認する。
- Go ソース変更時は、対応する全プラットフォームの配布バイナリが更新されているか確認する。
- manifest や起動経路を変えた場合、可能ならクリーンな一時ディレクトリから install 相当を再現し、MCP の tool list または実ツール呼び出しで確認する。
- ネットワーク接続、依存のインストール、ホームディレクトリの plugin cache 削除、実 GitHub/GitLab 操作は、必要性と影響を示して承認を得てから行う。

## Documentation

- 利用方法や前提条件が変わる場合は、ルートと対象プラグインの README を更新する。
- 明白な代替案を却下する判断をしたら、その場で `LESSONS.md` の末尾に 1 エントリ追記する（「却下した案 / 決め手（観測事実のみ） / 覆す条件」。既存エントリは編集しない）。agy の罠の要約は `plugin-gotchas.md` にも反映する。
- 構成（ファイルの置き場所・検査コード・agy 側のパス）が変わったら `NOTES.md` を更新する。
- `VISION.md` の変更（スコープ・設計判断）はユーザーの承認を得てから行う。
- `PLAN.md` は明示的に進捗更新を求められた場合、または現在の作業が既存 phase に直接対応する場合だけ更新する。

完了報告では、変更したファイル、実行した検証、実行できなかった検証と残るリスクを簡潔に記載してください。
