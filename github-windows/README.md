# GitHub MCP Server プラグイン（Windows）

公式の [github/github-mcp-server](https://github.com/github/github-mcp-server) を Antigravity CLI (`agy`) プラグインとして使えるようにパッケージしたものです。

Issues / Pull Requests / リポジトリ / Actions / Discussions など、公式がメンテナンスする全ツールセットを AI アシスタントから利用できます。

> **対応 OS**: Windows。Linux / macOS は [`github-unix`](../github-unix/README.md) を使ってください。
>
> Windows では `.sh` / `.cmd` / `.bat` を MCP の `command` に直接置けない（`CreateProcess` が直接実行できない）ため、トークン解決ラッパーを Go でビルドした `.exe` として同梱しています。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `gemini-extension.json` | プラグインマニフェスト。`${extensionPath}` でラッパーの絶対パス（拡張子なし）を解決する |
| `github-mcp-wrapper.exe` | トークンを解決し PATH 上の `github-mcp-server stdio` を起動する薄いラッパー |
| `github-mcp-wrapper.go` | 上記 `.exe` の Go ソース（`github-unix` の `.sh` ラッパーと等価） |

`agy plugin install` 時に `gemini-extension.json` の `${extensionPath}` が install 先の絶対パスへ解決され、`mcp_config.json` が自動生成されます（ソースに `mcp_config.json` / `plugin.json` は置きません）。`command` は拡張子なしのフルパス（`${extensionPath}${/}github-mcp-wrapper`）で、`agy` が `.exe` を補完して起動します。

## 必要条件

### 1. `github-mcp-server` を PATH に追加

**Go 経由（推奨）:**
```powershell
go install github.com/github/github-mcp-server/cmd/github-mcp-server@latest
```
`go install` の出力先（`%USERPROFILE%\go\bin`）が PATH に入っていることを確認してください。

**公式リリースバイナリを使う場合:**
[github/github-mcp-server Releases](https://github.com/github/github-mcp-server/releases) から Windows / お使いのアーキテクチャ（通常 `amd64`）のバイナリをダウンロードし、PATH の通ったディレクトリに置いてください。

### 2. GitHub 認証

ラッパーが次の順でトークンを解決します。**いずれか一つ**を満たせば手動の設定は不要です。

1. 環境変数 `GITHUB_PERSONAL_ACCESS_TOKEN`
2. 環境変数 `GITHUB_TOKEN`
3. 環境変数 `GH_TOKEN`
4. `gh auth token`（[GitHub CLI](https://cli.github.com/) で `gh auth login` 済みの場合）

最も手軽なのは `gh auth login` です:
```powershell
gh auth login
```

> 端末から起動した `agy` は shell の PATH と `gh` の認証を継承するため、`gh auth login` 済みなら追加設定なしで動作します。

## インストール

```powershell
agy plugin install https://github.com/kwrkb/agy-plugins/github-windows
```

## 高度な設定

`--read-only` や `--toolsets` などのオプションを使う場合は、`github-mcp-wrapper.go` の `args` を組み立てている箇所（`stdio` 引数）を編集し、再ビルドします。

```powershell
go build -o github-mcp-wrapper.exe ./github-mcp-wrapper.go
```

利用可能なオプションの詳細は [公式ドキュメント](https://github.com/github/github-mcp-server) を参照してください。

## ライセンス・帰属

本プラグインは [github/github-mcp-server](https://github.com/github/github-mcp-server)（MIT License）を PATH 経由で実行します。バイナリはリポジトリに同梱せず、ユーザーが導入したものを利用します（再配布なし）。同梱するのはトークンを解決するための薄いラッパー（`github-mcp-wrapper.exe` とそのソース `github-mcp-wrapper.go`）のみです。
