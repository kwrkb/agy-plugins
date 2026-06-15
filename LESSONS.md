# LESSONS（実装知見ログ）

## 2026-06-15: agy 1.0.8 のプラグイン同梱フックを実機検証（PR #4 をクローズ）

### 学んだこと

#### 18. agy のフック stdin は Claude Code と別形式 — `file_path` が無く編集ファイルを特定できない

agy 1.0.8 で tmux 対話セッションを起こし、`PostToolUse` フックを実発火させて payload をダンプした結果、agy が送る stdin は `{"artifactDirectoryPath","conversationId","error","stepIdx","toolCall":null,"transcriptPath","workspacePaths":[...]}` だった。Claude Code 流の `file_path` / `tool_input` が**存在せず** `toolCall` も `null`。よって `file_path` を前提にしたフックハンドラ（`validator --hook`）は**発火しても対象を特定できず常に no-op**。agy 向けフックを書くなら payload は実測してから設計する（`tee` で stdin を採取）。

#### 19. agy のフック発火は対話セッション限定かつ不安定

`agy -p`（print mode）ではフックは発火しない。対話セッションでは発火するが、**セッション内の最初の編集（特定 `stepIdx`）でのみ発火し、2 回目以降の `Edit` では発火しないことがある**。「編集ごとに必ず実行」という前提のフック機能は agy 1.0.8 では信頼できない。発火時の `PWD` はプラグインのインストール先（`~/.gemini/config/plugins/<name>/`）なので相対パスで同梱バイナリは呼べる。

#### 20. `hooks.json` 内の `${extensionPath}` / `${/}` は実行時に置換されない（Linux で `Bad substitution`）

`hooks.json` 内の `${extensionPath}` / `${/}` は実行時に一切置換されず literal のまま残り、そのままシェルに渡される。`${extensionPath}` は**未定義の環境変数**として `/bin/sh` に評価され空文字に消滅（argv ダンプで欠落を確認）、`${/}` は `/bin/sh` が不正な変数置換と見なし `sh: 1: Bad substitution` で hook プロセスが起動前にクラッシュする。agy 側にトークン展開は無い。

#### 21. 「修正」の前に機能が成立するかを実機検証する — クリーン install は git 追跡ファイルのみ

PR #4 は `hooks.json` のパス/パースを直したが、上記 #18 のとおり**そもそも agy 下で自動バリデーションは成立しない**機能の表面修正だった。さらに `validator/validator`（拡張子なし Linux バイナリ）を参照する一方そのバイナリは未コミットで、`git archive HEAD` で再現したクリーン URL install には `validator.exe` しか入らず Linux では解決不能だった（ローカル working tree に未追跡ビルドが在ったため「実機検証OK」に見えていた）。**機能の成立性 → クリーン install での再現性 の順で検証してから直す**。表面修正に入る前に「この機能はそもそも動くのか」を実機で確かめる。

#### 22. agy 1.0.8 の `rules/` 機能は完全に非機能 — プラグインの知識は `skills/` で渡す

実機検証（3 パターン: プラグイン内 `rules/*.md` / `plugin.json` の `"rules":[...]` / グローバル `~/.gemini/rules/*.md`）で、いずれもエージェントのシステムプロンプト（`<user_rules>`）に**一切注入されなかった**（未実装ないし不具合）。現状プロンプトへ載るのは UI 側設定（Gemini Added Memories）のグローバルルールのみ。**プラグインからエージェントへ固有の知識・規約を渡すには `rules/` に頼れず、必ず `skills/<name>/SKILL.md` として定義し呼び出させる**設計にする。`hooks.json`（#18-21）と同様、「ドキュメントに載っている機能 ≠ 実機で動く機能」。

## 2026-06-15: github-windows を実装しネイティブ Windows で検証（Issue #1）

### 学んだこと

#### 15. 「検証不能だから follow-up」は環境前提に依存する — 環境が変われば撤回する

判断 4（implementation-notes）で Windows 検証を「WSL2 では不能」と Issue 化したが、後日の作業環境が **Windows ネイティブ**だった。`follow-up 化` の根拠は環境制約であって設計の不確実性ではなかったので、環境が変わった時点で**着手前に前提を再確認**すれば、その場で end-to-end まで完了できた。検証可否は毎回環境を実測して判断する（`go version` が `windows/amd64`、`agy`/`gh` の有無）。

#### 16. Go ラッパーは sh ラッパーの `:-` 意味論を厳密移植する（空文字＝未設定）

`${A:-${B:-...}}` は**空文字列も未設定扱い**でフォールバックする。Go で `os.Getenv(name)` の戻り値が空でないか（`v != ""`）で判定しないと、空の `GITHUB_TOKEN` を「設定済み」と誤判定して `gh auth token` に落ちず、`.sh` と挙動が乖離する。`gh auth token` は `cmd.Output()` でバッファ取り込み（stdout 非継承＝NDJSON を汚さない）、子の exit code は `os.Exit` で伝播。`exec.LookPath("github-mcp-server")` は Windows で `.exe` を自動補完する。

#### 17. Windows でのラッパー検証は「MCP キャッシュ mtime 更新＋オーファン無し」で証拠化

LESSONS #12 の証拠法はそのまま Windows でも有効: env トークン無しで `agy -p` を流し、`~/.gemini/antigravity-cli/mcp/github/*.json` の mtime が更新されれば「トークン解決（`gh auth token`）→ server 起動 → introspect」が通った証拠。加えて Windows は exec-replace が無く agy→wrapper→server の親子構造になるが、stdio server は stdin EOF で終了するため**セッション後にオーファン残留しない**ことを `tasklist | grep github-mcp` で確認した（残る場合のみ Job Object 対応）。

## 2026-06-15: binary-on-PATH 化で github が起動不能になった件と差し戻し

### 学んだこと

#### 11. github-mcp-server は無トークンだと**起動拒否で即終了**する（gitlab との非対称）

URL install をクロスプラットフォーム化する過程で github をラッパー廃止＋「PATH の `github-mcp-server stdio` を直接 `command` に置く」binary-on-PATH 方式へ変更したが、**install は成功するのに MCP が起動しない**事象が発生。原因は `github-mcp-server` が `GITHUB_PERSONAL_ACCESS_TOKEN` 未設定だと `Error: GITHUB_PERSONAL_ACCESS_TOKEN not set` で**即 exit する**こと（知見 #5 の通りトークンは env からのみ読む）。直接 `command` 委譲はトークンを供給しないため起動できない。gitlab(`glab mcp serve`) は自前 config を読むので無トークンでも動く（知見 #7）——この非対称性を見落とすと「動かない」を踏む。

→ **差し戻し**: 薄い POSIX sh ラッパー（知見 #5）で `gh auth token` 等を解決してから PATH の `github-mcp-server` を exec する方式に戻した。ただしバイナリは**同梱せず PATH のものを exec**（URL install で clone されるのは軽量スクリプトのみ＝同梱バイナリ gitignore 問題も同時に解消）。

#### 12. 「動く」の検証は実環境（端末から agy）で、キャッシュ更新を証拠にする

`agy -p "..."`（print モード）でも**セッション起動時に MCP サーバーを introspect する**ことを利用し、`~/.gemini/antigravity-cli/mcp/<server-key>/` の mtime 更新を「起動成功」の客観証拠にできる。無トークンの github は stale のまま、`glab` は更新される、という差で切り分けられた。自分の shell で手動 export して直接バイナリを叩くテストは**実環境と一致しない**（agy の env 継承を経由していない）ので、ラッパー経由・env 未設定での確認が必須。

#### 13. `${extensionPath}` の検証より先に LESSONS を引くべきだった（手戻り）

`${extensionPath}` がネイティブ形式で効くかを probe プラグインで実験したが、答えは知見 #1 に既出だった（`gemini-extension.json` 形式でのみ解決、`plugin.json` があるとコピーのみ）。**過去の落とし穴は着手前に LESSONS.md を確認する**こと。probe の結果も #1 と完全一致した。

#### 14. OS 別分割: Windows は `.cmd`/`.sh` を `command` に直接置けない

ユーザー判断で github を OS 別に分割（`github-unix` / `github-windows`）。Windows は `.sh` 直接 spawn 不可（知見 #10）に加え、`.cmd`/`.bat` も `CreateProcess` が直接実行できず `cmd /c` 経由が要る。実測確認済みの方式は Go ラッパー `.exe`（拡張子なしフルパス、知見 #10）。WSL2 では Windows 実機検証不能のため、未検証コードを同梱せず follow-up 化する判断（「検証不能な提案は Issue 化」）。

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
