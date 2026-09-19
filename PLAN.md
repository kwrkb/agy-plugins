# PLAN: agy-plugins

## 現在地

フェーズ1〜13 すべて完了。進行中のフェーズなし。次フェーズは未定義。

## 完了フェーズ

経緯と判断の根拠は `LESSONS.md`（番号付き教訓・設計判断ログ）と各 PR を参照。

| # | フェーズ | 要約 |
|---|---|---|
| 1 | github プラグイン gh CLI ラッパー移行（PR #7） | 公式 `github-mcp-server` を廃し `gh` を exec する自前 Go サーバーへ統合。`args: string[]` 化・tmux+agy 実機検証（LESSONS #23-25） |
| 2 | CI 検証ゲート | 決定論ビルドフラグ導入、`build-verify.yml`（vet・test・govulncheck・stale 検出）新設、`CLAUDE.md` 作成 |
| 3 | github / gitlab へスキル追加（PR #9） | `rules/` 非機能のため `skills/` で知識を渡す。glab 実スキーマで裏取り（LESSONS #29-31） |
| 4 | agy 1.0.9 再検証と validator フック再導入 | hooks 部分機能化・rules 継続非機能を確定、validator を agy payload 対応で再同梱（LESSONS #34-36） |
| 5 | retro-status プラグイン追加 | リポジトリ統計を RPG ステータス風 AA で出力する MCP サーバー |
| 6 | agy 1.0.10 での hook / rule 再検証 | project `.agents/AGENTS.md` のみ注入機能化、プラグイン `rules/` は継続非機能、hooks 動的リロード（LESSONS #41-42、upstream #396 更新） |
| 7 | settings-advisor プラグイン追加（PR #16） | レビュー指摘対応込みで squash マージ（LESSONS #45-47） |
| 8 | Codex Security リポジトリスキャン | reportable 2件（`FD-KITVAL-001` / `FD-KITCMD-001`）。成果物はリポジトリ外に保存 |
| 9 | 全プラグインのメンテナンスと安全性向上 | フェーズ8の2件を修正、CI に settings-advisor 追加、日英ドキュメント同期 |
| 10 | worktree-manager プラグイン追加（PR #19） | Git ワークツリー管理 MCP サーバー、ビルド・CI 統合 |
| 11 | PR #19 レビュー指摘の修正 | Codex Review 指摘の修正と回帰テスト、Go 1.26.5 で再ビルド |
| 12 | settings-advisor の検知精度向上（PR #20） | スキャン検知拡充、`models.json` 更新と traits 駆動のモデル選定 |
| 13 | test-runner プラグイン追加（PR #23） | `go test -json` の集計と失敗テストの `rerun` 引数を返す MCP サーバー。CI 3 OS pass で squash マージ |
