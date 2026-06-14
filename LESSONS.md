# LESSONS（実装知見ログ）

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
