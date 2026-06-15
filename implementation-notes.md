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
