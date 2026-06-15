# GitHub MCP Server プラグイン（Linux / macOS）

公式の [github/github-mcp-server](https://github.com/github/github-mcp-server) を Antigravity CLI (`agy`) プラグインとして使えるようにパッケージしたものです。

Issues / Pull Requests / リポジトリ / Actions / Discussions など、公式がメンテナンスする全ツールセットを AI アシスタントから利用できます。

> **対応 OS**: Linux / macOS（POSIX シェル）。Windows 版は検証中です（リポジトリの Issue で追跡）。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `gemini-extension.json` | プラグインマニフェスト。`${extensionPath}` でラッパーの絶対パスを解決する |
| `github-mcp-wrapper.sh` | トークンを解決し PATH 上の `github-mcp-server stdio` を起動する薄いラッパー |

`agy plugin install` 時に `gemini-extension.json` の `${extensionPath}` が install 先の絶対パスへ解決され、`mcp_config.json` が自動生成されます（ソースに `mcp_config.json` / `plugin.json` は置きません）。

## 必要条件

### 1. `github-mcp-server` を PATH に追加

**Go 経由（推奨）:**
```bash
go install github.com/github/github-mcp-server/cmd/github-mcp-server@latest
```

**公式リリースバイナリを使う場合:**
[github/github-mcp-server Releases](https://github.com/github/github-mcp-server/releases) からお使いの OS / アーキテクチャのバイナリをダウンロードし、PATH の通ったディレクトリに置いてください。

### 2. GitHub 認証

ラッパーが次の順でトークンを解決します。**いずれか一つ**を満たせば手動の `export` は不要です。

1. 環境変数 `GITHUB_PERSONAL_ACCESS_TOKEN`
2. 環境変数 `GITHUB_TOKEN`
3. 環境変数 `GH_TOKEN`
4. `gh auth token`（[GitHub CLI](https://cli.github.com/) で `gh auth login` 済みの場合）

最も手軽なのは `gh auth login` です:
```bash
gh auth login
```

> 端末から起動した `agy` は shell の PATH と `gh` の認証を継承するため、`gh auth login` 済みなら追加設定なしで動作します。

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/github-unix
```

## 高度な設定

`--read-only` や `--toolsets` などのオプションは `github-mcp-wrapper.sh` 末尾の `exec` 行に追加します。

```sh
exec github-mcp-server stdio --read-only "$@"
```

利用可能なオプションの詳細は [公式ドキュメント](https://github.com/github/github-mcp-server) を参照してください。

## ライセンス・帰属

本プラグインは [github/github-mcp-server](https://github.com/github/github-mcp-server)（MIT License）を PATH 経由で実行します。バイナリはリポジトリに同梱せず、ユーザーが導入したものを利用します（再配布なし）。同梱するのはトークンを解決するための薄いラッパースクリプトのみです。
