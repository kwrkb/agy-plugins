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

## フェーズ2: CI 検証ゲート（完了）

- [x] 全 Go バイナリを決定論フラグ（`-trimpath -buildvcs=false -ldflags=-buildid=`）+ Go 1.26.4 で baseline 再ビルド
  > 2回ビルドで bit-identical を実測確認。validator は Linux 版（`validator`）も新規同梱に方針変更。
- [x] `.github/workflows/build-verify.yml` 新設（github / validator の2ジョブ：vet・test・govulncheck・決定論ビルド・`git diff --exit-code` ゲート）
- [x] `github/README.md` / `agy-plugin-kit/README.md` を決定論フラグ＋Go 1.26.4 固定・validator 両 OS 同梱に更新
- [x] `CLAUDE.md` 新規作成（構成・コマンド・tmux 実機検証・地雷の入口）

## フェーズ3: github / gitlab プラグインへスキル追加（PR #9・完了）

agy 1.0.8／1.0.9 では `rules/` が非機能（LESSONS #22/#35）なため、プラグインからエージェントへ知識を渡す唯一の手段が `skills/<name>/SKILL.md`。両プラグインにスキルが無く、エージェントが引数フォーマット・プロジェクトパス規約を知らずに操作する懸念があった。

- [x] `github/skills/github/SKILL.md` 作成（`gh_command` の args 配列規則・CWD 非定常で `-R` 必須・`--json` フィールド必須・頻出パターン）
- [x] `gitlab/skills/gitlab/SKILL.md` 作成（`glab_*` の args/flags/limit/offset 形式・プロジェクト指定の制約・カテゴリ別ツールマップ）
- [x] `glab mcp serve` の実スキーマを `tools/list` / `tools/call` で検証し記述を裏取り
  > `flags.repo` の有無・`assignee` 型はツール単位で異なることを実測（LESSONS #29-30）。
- [x] Codex レビュー対応（種別での過度な一般化バグを修正）
  > `glab_ci_list` は repo フラグなし、`glab_mr_list.assignee` は配列。実スキーマで確認のうえ訂正。
- [x] Gemini レビュー対応（表の区切りを `/` に統一）
  > SKILL.md は生テキストでエージェントに注入されるため `&#124;` は不適。LESSONS #31。
- [x] LESSONS.md 更新（#29 ツール単位の repo / #30 ツール単位の型 / #31 生テキスト consumer）
- [x] CI（build-verify: github / validator）pass を確認のうえ PR #9 を master へ squash マージ
- [x] PLAN.md / README 群更新
