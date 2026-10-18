# VISION: agy-plugins

本リポジトリの正典。目的・スコープ・非スコープと、現在有効な設計判断だけを書く。
ドキュメントの優先順は **VISION.md > PLAN.md > コード > NOTES.md**（`LESSONS.md` は序列外の記録）。
コードがここと矛盾していたら、どちらかに寄せる前に作業を止めて確認する。更新はユーザー承認が必要。

## 目的

agy（Google Antigravity CLI）で日常的に使う MCP プラグインを、**`agy plugin install <URL>` だけで
Linux / macOS / Windows のどれでも動く形**で配布する。あわせて、agy プラグイン開発で繰り返し踏んだ罠を
機械的に検出・回避するための開発キット（agy-plugin-kit）を提供する。

## スコープ

- 外部 CLI（`gh` / `ast-grep` / `gopls` / `git` / `go`）を呼ぶ、またはリポジトリを解析する Go 製 MCP サーバー:
  github / ast-grep / go-lsp / worktree-manager / test-runner / retro-status / settings-advisor
- 公式 MCP サーバーを設定だけで使うプラグイン: gitlab（`glab mcp serve`）
- agy-plugin-kit: 雛形・スラッシュコマンド・オーサリング skill と、agy 非依存の決定的な validator。
  マニフェスト編集時に validator を走らせ、結果をエージェントへ届けるフックを同梱する
- 配布対象の OS/arch: linux-amd64 / darwin-arm64 / windows-amd64

## 非スコープ

- 上記 3 つ以外の OS/arch 向けバイナリ（darwin-amd64、linux-arm64 など）
- 公式の MCP 機能を持つ CLI のラッパーを自前で再実装すること（gitlab は `glab mcp serve` をそのまま使う）
- API ラッパー型 MCP サーバー（github / gitlab など）の中で commit / push などの git 操作を行うこと（ホストエージェントの責務。
  worktree-manager の worktree 操作はこれに当たらない）
- プラグイン自身の MCP サーバーバイナリを利用者に PATH へ置かせる、またはトークンを手動で export させる配布方法
  （`gh` / `glab` / `ast-grep` / `gopls` などの外部 CLI への依存は、各 README の前提条件として許容する）
- 実機で確かめていない方式への一斉移行・未検証コードの同梱

## 現在有効な設計判断

各判断の根拠と「覆す条件」は `LESSONS.md` の該当エントリにある（`#N` は旧形式の番号付き教訓）。

### 配布・パッケージング

- **同梱バイナリを使うプラグインは `gemini-extension.json` 形式にし、`plugin.json` を置かない。**
  `command` は `${extensionPath}${/}bin${/}<name>`。`${extensionPath}` が install 時に解決されるのはこの形式だけで、
  native `mcp_config.json` の `command` では `${PLUGIN_ROOT}` も展開されない
  （#1、LESSONS 2026-10-18「同梱バイナリ参照プラグインは gemini-extension.json 形式を維持」）。
- **OS 別のネイティブバイナリ 3 本と、拡張子なしの POSIX sh dispatcher `bin/<name>` を同梱する。**
  dispatcher が `uname` で実機向けを `exec` し、Windows は agy が `.exe` を補完して直接起動する（#40）。
- **バイナリは git にコミットして配布する。** `agy plugin install` はビルドせず追跡ファイルをコピーするだけ（#21）。
  CI は再ビルドして差分を検出する検証ゲートで、CI から自動コミットはしない（判断 10）。
- **ビルドは決定論的にし、Go のパッチ版を `.go-version` の 1 か所に固定する。** `build.sh` / `build.ps1` / CI が
  `GOTOOLCHAIN` で強制し、bit-identical なバイナリを作る（LESSONS 2026-09-12「`.go-version` に集約」）。

### Go バージョンの引き上げ方針

固定版を上げるのは次のいずれかに当たる時だけ（LESSONS 2026-09-12「固定 Go 版の引き上げ条件」）。
1 回の引き上げは全ネイティブバイナリの再ビルド（大きな新規 blob）を伴う。

1. 到達可能な stdlib 脆弱性が、固定中のマイナー内のパッチで修正済み → そのマイナーの最新パッチへ（CI の govulncheck が fail で検知する）
2. 固定中のマイナーが EOL → サポート内マイナーへ
3. 依存の `go.mod` 下限が固定版を超えた
4. 必要な言語 / stdlib 機能がある

マイナー移行は EOL 直前まで据え置き、最新マイナーへの追従はしない。`go.mod` の `go` ディレクティブは言語の下限で、
固定版とは別の軸なので引き上げに合わせて動かさない。

### エージェントへの知識の渡し方

- **プラグイン固有のツール知識・手順は、オンデマンドで読み込まれる `skills/<name>/SKILL.md` で渡す。**
  agy 1.4.2 で frontmatter 付き `rules/*.md` も注入されるようになったが、`always_on` は全ルール共有の常時予算を
  消費するため、`rules/` は常時強制したい短いガードレールに限る（LESSONS 2026-10-18「rules/ 注入が機能化したが…」）。

### MCP サーバーの実装

- **MCP の stdout はプロトコル専用。** 診断は stderr へ出す（validator の C9 で検出）。
- **外部 CLI の引数は配列で受け取り、そのまま `exec` に渡す。** 自前のシェル風パーサは持たない（判断 9）。

### フック（agy-plugin-kit）

- **hooks.json の command は変数を使わず、プラグインルートからの相対パスで書く。** `${extensionPath}` /
  `${PLUGIN_ROOT}` / `${/}` はフックでは展開されない（#20、C10）。
- **command は Windows 形と Unix 形を `||` でつなぐ。** スラッシュ区切りの相対パスは Windows の `cmd /c` で起動しない
  （LESSONS 2026-10-18「フックの command は Windows 形と Unix 形を `||` でつなぐ」）。
- **`timeout` は秒単位で書く**（C11）。
- **フックの結果をエージェントへ届けるには、PostToolUse がファイルに書き、PreInvocation の `injectSteps` で差し込む。**
  PostToolUse の stderr / stdout はモデルに届かない。受け渡しファイルはユーザーごとのキャッシュディレクトリ（0700）に置く
  （LESSONS 2026-10-18「PostToolUse → PreInvocation の受け渡しを使う」）。差し込みを `ephemeralMessage` のままにするかは #33 で検討中。

### 検証

- **「動く」の証明は実機の agy で取る。** 起動の証拠は MCP キャッシュ `mcp/<plugin>_<server>/` のツール名、
  対話でのツール実行の証拠は TUI の `Called N MCP tool (...)` 表示、フックの差し込みの証拠は transcript の
  `EPHEMERAL_MESSAGE`（#25、LESSONS 2026-09-20 / 2026-10-18 の各エントリ）。
- **ソースの検査はキットの validator、install 後の検査は `agy plugin validate`。** `agy plugin validate` は
  `plugin.json` を直接要求するので、`gemini-extension.json` 形式のソースには使えない。
