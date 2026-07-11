# AGENTS.md

このファイルは、このリポジトリで作業する AI エージェント向けの実務ガイドです。利用者向けの概要は `README.md`、過去の検証結果や agy 固有の罠は `LESSONS.md` と `plugin-gotchas.md` を参照してください。

## Repository Overview

`agy-plugins` は、agy CLI 向け MCP プラグインのモノレポです。

- `github/`: `gh` CLI を呼び出す Go 製 MCP サーバー
- `gitlab/`: `glab mcp serve` を利用する設定中心のプラグイン（Go ソースなし）
- `ast-grep/`: `ast-grep` CLI を呼び出す Go 製 MCP サーバー
- `go-lsp/`: `gopls` を利用する Go 製 MCP サーバー
- `retro-status/`: リポジトリ情報をレトロゲーム風に表示する Go 製 MCP サーバー
- `settings-advisor/`: ワークスペースに適した agy 設定を提案する Go 製 MCP サーバー
- `agy-plugin-kit/`: プラグイン作成用の command、skill、template、および Go 製 validator

Go 製プラグインは原則として `<plugin>/src/` に独立した Go モジュール、`<plugin>/bin/` に配布用バイナリを持ちます。validator のみ `agy-plugin-kit/validator/{src,bin}/` 配下です。

## Before Editing

1. `git status --short` で既存のユーザー変更を確認し、巻き戻さない。
2. 変更対象の `README.md`、`src/main.go`、`src/main_test.go`、manifest、skill を読む。
3. agy の manifest、path、hook、rule、install 動作に触れる場合は、先に `LESSONS.md` と `plugin-gotchas.md` を検索する。
4. `PLAN.md` と `implementation-notes.md` は履歴資料を含むため、現行コードや manifest と矛盾する場合は現行ファイルを優先する。

## Implementation Conventions

- 変更は対象プラグイン内に限定し、他プラグインへの横展開は必要性が確認できた場合だけ行う。
- Go コードは標準ライブラリと既存ヘルパーを優先し、新しい production dependency は追加前に確認する。
- 外部 CLI の引数は文字列結合や手作業の再分割を避け、`exec.CommandContext` へ引数配列として渡す。
- MCP の stdout はプロトコル専用に保つ。診断ログは stderr へ出す。
- OS パスを扱う処理では Linux、macOS、Windows の区切り文字と実行形式を考慮する。
- ユーザー入力からファイルを読む、コマンドを実行する、または destructive な CLI 操作を公開する変更には、境界チェック、タイムアウト、説明を追加する。
- 挙動変更には、正常系だけでなく引数境界、パス、キャンセル、エラー処理など変更リスクに対応するテストを加える。

## Plugin Packaging Rules

- 同梱バイナリを使うプラグインは `gemini-extension.json` と `${extensionPath}${/}bin${/}<name>` を使う。`plugin.json` を併置すると path 展開条件が変わるため、形式を混在させない。
- Linux/macOS では拡張子なしの `bin/<name>` dispatcher が対応する native binary を起動する。dispatcher の executable bit を維持する。
- 配布対象は `bin/<name>-linux-amd64`、`bin/<name>-darwin-arm64`、`bin/<name>.exe`。ソース変更後は対象バイナリも更新する。
- プラグイン固有のエージェント知識は `skills/<name>/SKILL.md` に置く。プラグイン内 `rules/` や `plugin.json` の `rules` が注入される前提にしない。
- skill を新規作成する場合は、対象 AI の接尾辞を skill 名、ディレクトリ名、frontmatter の `name:` で一致させる。Codex 用は `-codex`。
- hook command では `${extensionPath}` や `${/}` の展開を前提にせず、既存の `agy-plugin-kit/hooks.json` の相対パス方式に従う。

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
```

Go ソースを変更したら、リポジトリルートで対象を決定論的に再ビルドします。

```sh
./build.sh <github|validator|ast-grep|go-lsp|retro-status|settings-advisor>
```

Windows PowerShell では同じ target を `./build.ps1` に渡します。決定論ビルドは `go 1.26.5` を前提とします。ローカル Go バージョンが異なる場合、コミット済みバイナリとの差分を正しい更新とみなさず、使用できなかったことを報告してください。

## Verification and Review

- `git diff --check` で whitespace error を確認する。
- `git diff --stat` と `git diff -- <changed-files>` で、意図しない文書・manifest・バイナリ差分がないか確認する。
- Go ソース変更時は、対応する全プラットフォームの配布バイナリが更新されているか確認する。
- manifest や起動経路を変えた場合、可能ならクリーンな一時ディレクトリから install 相当を再現し、MCP の tool list または実ツール呼び出しで確認する。
- ネットワーク接続、依存のインストール、ホームディレクトリの plugin cache 削除、実 GitHub/GitLab 操作は、必要性と影響を示して承認を得てから行う。

## Documentation

- 利用方法や前提条件が変わる場合は、ルートと対象プラグインの README を更新する。
- agy の再利用可能な挙動・罠が判明した場合は `LESSONS.md` または `plugin-gotchas.md` に記録する。
- `PLAN.md` は明示的に進捗更新を求められた場合、または現在の作業が既存 phase に直接対応する場合だけ更新する。

完了報告では、変更したファイル、実行した検証、実行できなかった検証と残るリスクを簡潔に記載してください。
