# LESSONS（実装知見ログ）

## 2026-06-14: github プラグインを Windows でクロスプラットフォーム化

### 学んだこと

#### 10. MCP `command` に `.sh` を直接置くと Windows で起動不可 → Go ラッパーで解決

`agy plugin install` が生成する `mcp_config.json` の `command` は `.sh` の絶対パスになる。Windows では `.sh` を直接 spawn できないため agy が MCP サーバーを起動できずエラーになる。

**agy は Node.js ホストではない**（`agy.exe` は PE32+ コンパイル済みバイナリ）。Node.js を前提にした解決策（`.mjs` ラッパー）は不要な依存を追加する。

**正しい解決**: Go で認証ラッパーをビルドし `gemini-extension.json` の `command` をフルパス（拡張子なし）で指定する。

```json
"command": "${extensionPath}${/}mcpServers${/}github-mcp-wrapper"
```

**実測確認済み**: Windows で `mcpServers/github-mcp-wrapper.exe` を置き、`command` に `.exe` なしフルパスを指定しても agy は正常に spawn できる（MCP ハンドシェイク・ツール呼び出し両方動作）。

**Go ラッパーの注意点**:
- stdout には書かない（MCP は NDJSON を stdout で流すため）
- `os.Executable()` で自身のパスを取得し、同ディレクトリのバイナリを `runtime.GOOS` で `.exe` 付与して起動
- `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr` で stdio を素通し

## 2026-06-14: gitlab プラグインを `glab mcp serve` へ置き換え

### 学んだこと

#### 7. CLI 内蔵 MCP は wrapper 不要（github と非対称）

GitLab 公式 CLI `glab` は **v1.74.0 頃から `glab mcp serve`（stdio, EXPERIMENTAL）** を内蔵する。github-mcp-server は MCP 専用バイナリで `GITHUB_PERSONAL_ACCESS_TOKEN` のみ読む→ wrapper でトークンをブリッジする必要があったが、`glab mcp serve` は glab 自身のサブコマンドなので **glab 既存 config（`~/.config/glab-cli/config.yml`）をそのまま再利用**する。よってトークン env も wrapper も不要。「公式バイナリ置き換え」でも、対象が汎用 CLI 内蔵か MCP 専用バイナリかで認証グルーの要否が変わる。

#### 8. apt 版 glab は古く mcp 非対応 → go install で最新化

Ubuntu universe の `glab` は 1.53.0（apt 候補も同じ）で `mcp` サブコマンド非対応。`go install gitlab.com/gitlab-org/cli/cmd/glab@latest` で最新化し、`~/go/bin`（PATH 上）に入れる。`gemini-extension.json` の `command` は**ベア名 `glab`**（PATH 解決）で良く、同梱バイナリ不要＝build.sh の gitlab セクションも撤去できる。注意: `go install`（ldflags 未注入）だと `glab --version` は `DEV` 表示になる→バージョン判定は version 文字列でなく `glab mcp serve --help` の成否で行う。

#### 9. バージョン導入時期の二分探索は docs raw を使う

`mcp serve` がどの版で入ったかは、`docs/source/mcp/serve.md` を各タグの raw URL（`/-/raw/<tag>/...`）で取得し有無を二分探索して特定できる（v1.70=なし, v1.74=あり）。grep 判定する際、ファイル先頭は YAML frontmatter の `---` なので `head -1` で `title` を探すと誤判定する（`head -5` でマッチ語を見る）。

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

#### 1. `agy` のプラグイン形式と `${extensionPath}` 置換の仕組み（2026-06-14 更新）

**インストール済みディレクトリの形式（`~/.gemini/config/plugins/<name>/`）:**
- `plugin.json`: 必須マニフェスト（`name` 必須、`description` / `disabled` も可）。`agy plugin validate` が要求する。
- `mcp_config.json`: `mcpServers` を持つ。`agy` が MCP サーバーを起動するときに参照する。
- 上記2ファイルは **`agy plugin install` が生成する**（ソースに置くものではない）。

**`${extensionPath}` 置換の条件（ここが核心）:**
- **ソースに `gemini-extension.json` があり `plugin.json` が無い** → install が `gemini-extension.json` を読んで `${extensionPath}` を解決し、絶対パスの `mcp_config.json` を生成する。
- **ソースに `plugin.json` がある** → install は新形式プラグインと判断し、ソースの `mcp_config.json` をそのままコピーする（`${extensionPath}` は**解決されない**）。

**実用的なルール:**
- `${extensionPath}${/}` を使う必要があるプラグイン（同梱バイナリを参照する github など）は ソースに `gemini-extension.json` を置き `plugin.json` は置かない。
- PATH 上のコマンドを使うプラグイン（gitlab など）はソースに `plugin.json` + `mcp_config.json` を置ける（置換不要なため）。

**置換変数名:** `${extensionPath}` が正しい（`${pluginPath}` ではない）。バイナリの置換トークン集合に
存在するのは `${extensionPath}` / `${workspacePath}` のみ（`convertPluginPath` は Go 内部シンボル、トークンではない）。

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
