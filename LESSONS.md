# LESSONS（実装知見ログ）

## 2026-06-14: 公式 github-mcp-server への置き換え

### 学んだこと

#### 5. 公式 MCP サーバーは env トークン名が固定 → wrapper でブリッジ

公式 [github/github-mcp-server](https://github.com/github/github-mcp-server) は `GITHUB_PERSONAL_ACCESS_TOKEN` **のみ**を読む（`GITHUB_TOKEN` も `gh auth token` も見ない）。一方このマシンの認証は `gh` CLI のみ（静的 env トークンなし）。

→ 公式バイナリをそのまま `command` にすると認証で動かない。薄い POSIX sh wrapper (`github-mcp-wrapper.sh`) で `GITHUB_PERSONAL_ACCESS_TOKEN` → `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token` の順に解決して `export` → `exec` する。これは回避策ではなく統合グルー。`$(dirname "$0")/github-mcp-server stdio` で同梱バイナリを呼ぶ（agy は `command` を絶対パスで渡すので `$0` は絶対パス）。

検証手順: バイナリ受領後に `--help` と `strings | grep GITHUB_` で実際に読む env を実測（README 要約を鵜呑みにしない）、`gh auth status` でユーザーの認証経路を確定してから config を書く。

#### 6. 公式バイナリは `go install pkg@version` で同梱

`GOBIN=.../mcpServers go install github.com/github/github-mcp-server/cmd/github-mcp-server@v1.3.0` でバージョン固定ビルド。出力バイナリ名は cmd ディレクトリ名 = `github-mcp-server`。`go.mod` 不要（モジュールモードの `pkg@version` 形式はローカルモジュール非依存）なので自作の `main.go`/`go.mod`/`go.sum` は削除できる。`build.sh` 冒頭で `mcpServers/` を `rm -rf` してから作り直すと旧バイナリ残骸を一掃できる。

## 2026-06-14: agy MCP プラグイン再構築

### 学んだこと

#### 1. `${extensionPath}` 変数置換は `gemini-extension.json` 専用

`agy`（Antigravity CLI、gemini-cli フォーク）における MCP サーバーのパス解決変数：

- **`gemini-extension.json`**: `${extensionPath}`（インストール先ディレクトリ）、`${/}`（パス区切り）の置換が**機能する**
- **`mcp_config.json`**: 変数置換が**機能しない**（文字列がそのまま渡される）

→ プラグインの `command` は必ず `gemini-extension.json` の `mcpServers` 内で定義すること。

#### 2. `agy plugin install` の動作

- インストール元ディレクトリ全体を `~/.gemini/config/plugins/<name>/` にコピーする
- `gemini-extension.json` の `${extensionPath}` を絶対パスに解決して `mcp_config.json` を自動生成する
- バイナリも含めてコピーされるため、**ソースを再ビルドしたら `agy plugin install` の再実行が必要**

#### 3. go-sdk MCP の stdio プロトコルテスト

`mcp.StdioTransport{}` は NDJSON（改行区切り JSON）を使う。テストには stdin を開いたまま双方向通信できる Python subprocess が有効：

```python
proc = subprocess.Popen([binary], stdin=PIPE, stdout=PIPE, stderr=PIPE, text=True, bufsize=1)
proc.stdin.write(json.dumps(msg) + '\n')
proc.stdin.flush()
response = json.loads(proc.stdout.readline())
```

`echo ... | binary` や `binary < file` は stdin が即 EOF になるため MCP サーバーがレスポンスを書く前に終了する。

#### 4. API ラッパー型 MCP サーバーへの git 操作は不適

`git` CLI を subprocess で呼ぶツール（`commit_and_push` 等）を API ラッパー MCP サーバーに入れると、以下の問題が複合する：

1. CWD が不定（MCP サーバーのプロセスは呼び出し元の CWD 依存）
2. `git push` 認証が SSH/credential helper 系でトークンと別系統
3. `git commit` に `user.name`/`user.email` が必要（MCP コンテキストでは未設定）
4. `git add .` が広範すぎる

→ git 操作は agy/claude ホストエージェント側の責務。MCP サーバーは API 呼び出しに徹する。
