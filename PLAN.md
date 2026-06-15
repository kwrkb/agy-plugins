# PLAN: github プラグイン gh CLI ラッパー移行（PR #7）

## 現状の把握

- 旧 `github-unix` / `github-windows`（公式 `github-mcp-server` + OS 別トークン解決ラッパー）を廃止。
- 単一クロスプラットフォーム `github` プラグインに統合。**システムの `gh` CLI を直接 exec する自前 Go 製 MCP サーバー**方式。
- `gh auth login` 済みなら追加のトークン設定・ラッパー不要（`gh` の認証をそのまま継承）。
- バイナリ（`github` / `github.exe`）は手動 `go build` して git にコミット。`agy plugin install` はビルドせずコミット済みバイナリをコピーする（CI / build script なし）。

## フェーズ1: 移行実装（完了）

- [x] `github-mcp-server` 依存を廃し `gh` CLI ラッパー（`main.go`）へ移行
- [x] `github-unix` / `github-windows` を削除し単一 `github` に統合
- [x] root README / `github/README.md` 更新
- [x] `gh_command` の引数を単一文字列＋`strings.Fields` から `args: string[]` 配列へ変更
  > スペースを含む値（title/body/検索クエリ）が空白分割で破損するブロッカー修正。`mcp.WithArray`+`request.RequireStringSlice` で受け、依存追加ゼロ。LESSONS #23。
- [x] 回帰テスト追加（`main_test.go`、spaced 値が exec 境界をそのまま越えることを検証）
- [x] tmux + agy 実機検証
  > クリーン install（`git archive HEAD`）→ MCP キャッシュが新サーバーの `gh_command.json` のみに置換 → `gh search repos "mark3labs mcp-go"` がエンドツーエンドで正しく返却。LESSONS #24/#25。
- [x] スコープ外の Linux `validator` バイナリを除去（`.exe` 正規同梱物は保持）
- [x] LESSONS.md 更新（#23 配列入力 / #24 install は wipe しない / #25 キャッシュのツール名で確認）
- [x] PR #7 を origin に push・更新
- [x] Gemini Code Assist レビュー対応
  > Medium 指摘の `exec.CommandContext`（取消・タイムアウト伝播）を採用。High 指摘の `splitCommand`（手書きシェルパーサ）は配列化で根本解決済みのため見送り。

- [x] destructive な `gh` 操作（`gh repo delete` / `gh api` 等）への README 注意書き
  > `github/README.md` の「提供されるツール」に任意コマンド実行の警告を追記。あわせて `gh_command` を配列引数 `args` 方式に記述更新。`destructiveHint` アノテーションは単一汎用ツールのため見送り。
- [x] PR #7 マージ（ユーザー承認のうえ master へマージ）

## フォローアップ Issue 候補

- [ ] バイナリビルドの CI 自動化（`main.go` 変更時の再ビルド忘れ＝stale バイナリ配布を防止）
