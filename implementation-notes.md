# Implementation Notes（意思決定ログ）

## 2026-06-15: github プラグインを wrapper 方式へ差し戻し（OS 別分割）

### 判断 1: binary-on-PATH をやめ、ラッパー方式へ戻す
- **背景**: 直前に「PATH の `github-mcp-server stdio` を `command` に直接置く」binary-on-PATH 方式へ変更したが、`github-mcp-server` が無トークンで即 exit するため起動不能だった。
- **選択肢**: (a) README で `export GITHUB_PERSONAL_ACCESS_TOKEN=...` を手動案内 / (b) mcp_config に env 注入（agy の env 展開サポート不明）/ (c) ラッパーでトークン自動解決。
- **決定**: (c)。ユーザーは端末から agy を起動し env/PATH/`gh` を継承するため、ラッパーの `gh auth token` フォールバックが「手動設定なしで動く」を実現できる。(a) は UX が悪く、(b) は agy の挙動が未検証。

### 判断 2: バイナリは同梱せず PATH の `github-mcp-server` を exec
- 旧設計は build.sh で公式バイナリを `mcpServers/` に同梱していたが gitignore 済み → URL install で clone されず起動不能、というのが元の発端。ラッパーは PATH のバイナリを exec するだけにし、同梱は軽量スクリプトのみ。これで URL install が成立し、ライセンス上の再配布も発生しない。

### 判断 3: 形式は gemini-extension.json（plugin.json ではない）
- ラッパーを `${extensionPath}` で参照する必要があるが、`${extensionPath}` は `gemini-extension.json` 形式でのみ解決される（`plugin.json` があるとソースの mcp_config がコピーされるだけ。LESSONS #1）。よって native 形式ではなく gemini-extension.json を採用。install 時に plugin.json と解決済み mcp_config.json が自動生成されることを実機確認。

### 判断 4: OS 別分割で Windows は未同梱（follow-up）
- ユーザー判断で `github-unix` / `github-windows` に分割。Windows は `.sh`/`.cmd` を `command` に直接置けず、実測確認済みは Go `.exe` ラッパー（LESSONS #10）。WSL2 では Windows 実機検証ができないため、**未検証コードを同梱せず**フォローアップ（Issue）に回す。`github-unix`（実環境で end-to-end 検証済み）を先行リリース。

## 2026-06-15: github-windows を実装・ネイティブ Windows で end-to-end 検証（Issue #1）

### 判断 5: 検証 follow-up を撤回し、本タスクで検証まで完了させた
- **背景**: 判断 4 では WSL2 制約で Windows 検証不能 → Issue 化していた。が、今回の作業環境が **Windows 11 ネイティブ**（Go 1.26.4 windows/amd64、`gh` 認証済み、`agy` インストール済み）と判明。制約が消えたため「未検証コードを同梱しない」原則の適用対象外になった。
- **決定**: ラッパーを実機ビルド＋`agy plugin install`＋`agy -p` で end-to-end 検証してからコミット。LESSONS #12 の証拠法（MCP キャッシュ mtime 更新）で起動成功を客観確認した。

### 判断 6: Go ラッパーは sh の `:-` 意味論を厳密移植
- 空文字列の env を未設定扱い（`v != ""`）にし、空 `GITHUB_TOKEN` でも `gh auth token` へフォールバックさせた。`os.Getenv` の存在チェックだけだと空文字を「設定済み」と誤判定し `.sh` と挙動が乖離するため。
- `gh auth token` は `cmd.Output()` でバッファ取り込み（stdout 非継承）。1 バイトでも漏れると NDJSON が壊れる。子の exit code は `os.Exit` で伝播。

### 判断 7: Codex レビュー指摘の取捨（依存追加を避ける範囲で反映）
- **反映**: (#2) 空白のみ env を `strings.TrimSpace` で未設定扱いにしフォールバック継続。(#5) `LookPath` 失敗時に `github-mcp-server.exe` を明示再試行（PATHEXT 非標準対策）。(#8) README に「PATH は信頼できるディレクトリのみで構成」のセキュリティ注記。
- **見送り**: (#6 Job Object によるオーファン kill) — 実測でオーファン残留が無く（stdin EOF で stdio server が終了）、`golang.org/x/sys/windows` 依存＋`go.mod` 追加というコストに見合わない。advisor も「残留した場合のみ Job Object、先回り実装はしない」と助言済み。残留を観測したら導入する方針を README/notes に残す。

### 判断 8: PR レビュー（gemini-code-assist）指摘で go.mod を追加
- **指摘**: クリーン環境/CI で `go build` が `go.mod not found` で失敗しうるので `github-windows/go.mod` を追加すべき。
- **検証**: 親ツリーに go.mod 無し＋完全クリーンな一時ディレクトリで**単一ファイル `go build x.go` は成功**（stdlib のみはモジュール不要）。よって指摘の失敗前提は単一ファイルビルドには当てはまらない。一方 `go build .`（パッケージモード）は go.mod 無しだと失敗する。
- **決定**: 堅牢性・慣習に従い最小 `go.mod`（`module github-mcp-wrapper` / `go 1.21`）を追加。パッケージモードビルドも可能になり、ビルド/vet はモジュール内で実行する形に統一（README の再ビルド手順も `cd github-windows && go build -o github-mcp-wrapper.exe .` に更新）。追加後に両ビルドモード・`go vet`・agy end-to-end を再検証済み。

### 検証結果（全 6 点パス / ネイティブ Windows）
- ビルド: `go build` 単一 stdlib ファイル、go.mod 不要。
- install: `mcp_config.json` の `command` が `...\github\github-mcp-wrapper`（拡張子なし絶対パス）に解決。
- 起動証拠: env トークン無し → `gh auth token` フォールバックで解決 → `agy -p` 実行後にキャッシュ（`get_me.json` 等）mtime が `08:10`→`12:53` に更新、40+ ツールが introspect された。
- オーファン: セッション終了後に `github-mcp-server.exe` の残留なし（stdin EOF で子が終了）。
- go.mod / arm64 は不要（amd64 のみ。arm64 は必要時 follow-up）。
