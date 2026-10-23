# GitLab MCP Server プラグイン (Cross-Platform)

GitLab 公式 CLI [`glab`](https://gitlab.com/gitlab-org/cli) を利用して、GitLab の各種操作（Issues・Merge Requests・CI/CD・Releases など）を AI アシスタントから実行できるようにする MCP サーバープラグインです。

> **対応 OS**: Linux / macOS / Windows 共通

## 概要

このプラグインは、**システムにインストールされた `glab` コマンド** を内部で呼び出す独自の MCP サーバー（Go 言語実装）として動作します。[github](../github/README_ja.md) プラグインが `gh` を呼ぶのと同じ方式です。

- `glab` の既存設定と `glab auth login` の認証をそのまま利用するため、追加のトークン設定は不要。
- OS を問わず単一のプラグインとして動作。

> **`glab mcp serve` を使わない理由**: 以前のこのプラグインは `glab` 内蔵の `glab mcp serve` を設定だけで使っていました。これは EXPERIMENTAL で、**glab v1.124.0 で deprecated** になり、後継として [GitLab MCP Server](https://docs.gitlab.com/user/model_context_protocol/mcp_server/) が案内されています。後継は HTTP + OAuth 方式で、agy で動くかは未確認のため、`glab` 自体を呼ぶラッパーに移りました。

## 構成

| ファイル | 役割 |
| :--- | :--- |
| `gemini-extension.json` | プラグインマニフェスト。`command` は `${extensionPath}${/}bin${/}gitlab` を絶対パスへ解決する |
| `src/main.go` / `src/go.mod` / `src/go.sum` | `glab` コマンドを呼び出す MCP サーバーのソースコード |
| `bin/gitlab-linux-amd64` / `bin/gitlab-darwin-arm64` / `bin/gitlab.exe` | コンパイル済みの MCP サーバーネイティブバイナリ（各 OS×arch） |
| `bin/gitlab` | OS 分岐 dispatcher（shebang sh）。`uname` で実機を判定し対応ネイティブを `exec`（Windows は agy が `bin/gitlab.exe` を直接起動） |
| `skills/gitlab/SKILL.md` | エージェント向け使用ガイド（呼び出し時ロード）。`glab_command` の引数規則・`-R` 必須とその例外・`--output json`・使わないコマンド・頻出パターン |

## 必要条件

### 1. `glab` コマンドを PATH に追加
[`glab`](https://gitlab.com/gitlab-org/cli) をインストールし、ターミナルで `glab` が実行できる状態にしてください。**v1.124.0** で動作確認済みです。

- apt 版（Ubuntu universe）の `glab` は古いので、最新版を導入してください:
  ```bash
  # 必要なら apt 版を削除
  sudo apt remove glab
  # 最新の glab を導入（~/go/bin が PATH 上にあること）
  go install gitlab.com/gitlab-org/cli/cmd/glab@latest
  ```
- `go install` でビルドした場合 `glab --version` は `DEV` と表示されます。実際の版は次で確認できます:
  ```bash
  go version -m "$(command -v glab)" | awk '$1 == "mod" { print $3 }'
  ```

### 2. GitLab 認証

```bash
glab auth login        # 未認証の場合
glab auth status       # 認証済みか確認
```

MCP サーバーは `glab` 自身の設定（`~/.config/glab-cli/config.yml`）をそのまま利用します。OAuth / 個人用アクセストークン（PAT）いずれも、`glab auth login` で構成済みであれば利用されます。

## 提供されるツール

- **`glab_command`**: 任意の `glab` サブコマンドを実行します。引数は**トークンごとに分割した文字列の配列** `args` で渡します（例: `["issue", "list", "-R", "group/project"]`、`["mr", "view", "17", "-R", "group/project"]`）。
  - スペースを含む値（タイトル・説明・検索クエリなど）は**1要素**にまとめます。文字列を空白分割する方式ではないため、クオートで囲む必要はありません。
  - プロジェクトは常に `-R group/project` で指定します。MCP サーバーの作業ディレクトリはワークスペースとは限らず、`glab` のカレントリポジトリ検出は当てになりません。
  - 対話プロンプトは無効です（`GLAB_NO_PROMPT=1`・stdin なし）。必須フラグが足りないコマンドは、入力を待たずにエラーで返ります。

> ⚠️ **注意（任意コマンド実行）**: `glab_command` は `glab` の**あらゆるサブコマンドを実行できます**。読み取りだけでなく、`glab mr merge` / `glab issue close` / `glab repo delete` / `glab api`（任意の REST 呼び出し）といった**書き込み・破壊的操作も実行可能**で、`glab auth login` 済みアカウントの権限で動きます。エージェントに渡すタスクの範囲に注意し、信頼できる文脈でのみ利用してください。

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

### `glab mcp serve` 版からの更新

`agy plugin install` は以前の install で置かれたファイルを消さないため、旧版の `plugin.json` / `mcp_config.json` が新しいマニフェストの横に残ります。install 済みのプラグインを削除してから入れ直してください:

```bash
rm -rf ~/.gemini/config/plugins/gitlab
agy plugin install https://github.com/kwrkb/agy-plugins/gitlab
```

> **注意**: ソースコード (`src/main.go`) を変更したら、`./build.sh gitlab`（Windows は `./build.ps1 gitlab`）で全 OS 分のバイナリを再ビルドしてコミットしてください（`agy plugin install` はビルドせず**コミット済みバイナリをコピー**するため、再ビルドを忘れると stale バイナリが配布されます）。Go のパッチ版は `.go-version` に固定され、CI の検証ゲート `.github/workflows/build-verify.yml` がビルド結果との bit-identical 一致を要求します。

## ライセンス・帰属

本プラグインは、利用者が別途インストールした [gitlab-org/cli (`glab`)](https://gitlab.com/gitlab-org/cli)（MIT License）を呼び出すラッパーです。`glab` バイナリの再配布はしません。本プラグインのコードは公式実装に準じ MIT として扱います。
