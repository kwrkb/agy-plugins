# GitHub MCP Server プラグイン (Cross-Platform)

GitHub CLI (`gh`) を利用して、GitHub の各種操作（Issues, Pull Requests など）を AI アシスタントから実行できるようにする MCP サーバープラグインです。

> **対応 OS**: Linux / macOS / Windows 共通

## 概要

このプラグインは公式の `github/github-mcp-server` を使わず、**システムにインストールされた `gh` コマンド** を内部で呼び出す独自の MCP サーバー（Go 言語実装）として動作します。
これにより、以下のメリットがあります：

- 既に `gh` コマンドを使用している環境であれば、追加のトークン設定やラッパースクリプトが不要。
- `gh auth login` による認証セッションをそのまま引き継いで動作するため、面倒な PAT (Personal Access Token) 管理が不要。
- OS を問わず単一のプラグインとして動作。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `gemini-extension.json` | プラグインマニフェスト。`${extensionPath}` でビルドされたバイナリの絶対パスを解決する |
| `main.go` / `go.mod` / `go.sum` | `gh` コマンドを呼び出す MCP サーバーのソースコード |
| `github` / `github.exe` | コンパイル済みの MCP サーバーバイナリ |
| `skills/github/SKILL.md` | エージェント向け使用ガイド（呼び出し時ロード）。`gh_command` の引数規則・`-R` 必須・`--json` フィールド指定・頻出パターン |

## 必要条件

### 1. `gh` コマンドを PATH に追加
[GitHub CLI](https://cli.github.com/) をインストールし、コマンドプロンプトやターミナルで `gh` コマンドが実行できる状態にしてください。

### 2. GitHub 認証
ターミナル上で以下を実行し、GitHub へのログインを済ませておいてください。

```bash
gh auth login
```

これだけで設定は完了です。MCP サーバーは `gh` コマンドの認証をそのまま利用します。

## 提供されるツール

- **`gh_command`**: 任意の `gh` サブコマンドを実行します。引数は**トークンごとに分割した文字列の配列** `args` で渡します（例: `["issue", "list", "--limit", "10"]`、`["pr", "view", "123"]`）。
  - スペースを含む値（タイトル・本文・検索クエリなど）は**1要素**にまとめます（例: `["pr", "create", "--title", "My Title"]`）。文字列を空白分割する方式ではないため、クオートで囲む必要はありません。

> ⚠️ **注意（任意コマンド実行）**: `gh_command` は `gh` の**あらゆるサブコマンドを実行できます**。読み取りだけでなく、`gh pr merge` / `gh issue close` / `gh repo delete` / `gh api`（任意の REST 呼び出し）といった**書き込み・破壊的操作も実行可能**で、`gh auth login` 済みアカウントの権限で動きます。エージェントに渡すタスクの範囲に注意し、信頼できる文脈でのみ利用してください。

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/github
```

> **注意**: ソースコード (`main.go`) を変更したら、下記コマンドで両 OS 分のバイナリを再ビルドしてコミットしてください（`agy plugin install` はビルドせず**コミット済みバイナリをコピー**するため、再ビルドを忘れると stale バイナリが配布されます）。

### バイナリの再ビルド

リポジトリルートのビルドスクリプトを使います（**Go 1.26.4**。決定論フラグはスクリプトに集約。CI の検証ゲート `.github/workflows/build-verify.yml` がこの結果との bit-identical 一致を要求し、Go のバージョンがずれると fail します）。

```bash
./build.sh github    # github のバイナリ（linux/windows）を再ビルド。Windows は ./build.ps1 github
# 引数なし（./build.sh / ./build.ps1）で全プラグイン
```
