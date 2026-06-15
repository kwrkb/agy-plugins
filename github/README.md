# GitHub MCP Server プラグイン

公式の [github/github-mcp-server](https://github.com/github/github-mcp-server) を Antigravity CLI (`agy`) プラグインとして使えるようにパッケージしたものです。

Issues / Pull Requests / リポジトリ / Actions / Discussions など、公式がメンテナンスする全ツールセットを AI アシスタントから利用できます。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `plugin.json` | プラグインメタデータ |
| `mcp_config.json` | MCP サーバー設定（PATH 上の `github-mcp-server stdio` を起動） |

## 必要条件

### 1. `github-mcp-server` を PATH に追加

**Go 経由（推奨）:**
```bash
go install github.com/github/github-mcp-server/cmd/github-mcp-server@latest
```

**公式リリースバイナリを使う場合:**  
[github/github-mcp-server Releases](https://github.com/github/github-mcp-server/releases) からお使いの OS / アーキテクチャのバイナリをダウンロードし、PATH の通ったディレクトリに置いてください。

### 2. 認証トークンの設定

環境変数 `GITHUB_PERSONAL_ACCESS_TOKEN` に GitHub の Personal Access Token を設定します。

```bash
export GITHUB_PERSONAL_ACCESS_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
```

**`gh` CLI を使っている場合の Tip:**
```bash
export GITHUB_PERSONAL_ACCESS_TOKEN=$(gh auth token)
```

シェルの設定ファイル（`~/.bashrc` / `~/.zshrc` / `$PROFILE`）に追記しておくと毎回設定不要になります。

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/github
```

## 高度な設定

`--read-only` や `--toolsets` などのオプションを使う場合は `mcp_config.json` の `args` に追加します。

```json
{
  "mcpServers": {
    "github": {
      "command": "github-mcp-server",
      "args": ["stdio", "--read-only"]
    }
  }
}
```

利用可能なオプションの詳細は [公式ドキュメント](https://github.com/github/github-mcp-server) を参照してください。

## ライセンス・帰属

本プラグインは [github/github-mcp-server](https://github.com/github/github-mcp-server)（MIT License）を PATH 経由で実行します。バイナリはリポジトリに同梱せず、ユーザーが導入したものを利用します（再配布なし）。
