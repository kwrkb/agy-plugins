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

## フェーズ4: agy 1.0.9 再検証と validator フック再導入（2026-06-17）（完了）

agy 1.0.8 → 1.0.9 で hooks/rules を実機再検証し、hooks の部分機能化を受けて validator フックを再導入。

- [x] `agy changelog` 全件確認（hooks/rules 変更ゼロ・L79 の `rules.json` 矛盾を発見）
- [x] 1.0.9 実機再検証（hooks 4トリガー＋rules 3パターン）。hooks 部分解消（`toolCall.args.TargetFile`・2回目以降/`-p` 発火・PWD 相対）／rules 継続非機能を確定（LESSONS #34/#35）
- [x] L79 を strings 解析で切り分け（`<user_rules>`=Memories 由来 / `rules.json`=customizations 別系統）
- [x] メモリ＋ docs 更新（LESSONS #34/#35、CLAUDE/README/PLAN の version 前提、hook-investigation-report §5 に証拠 payload）
- [x] upstream agy#395（hook payload）を解決済みでクローズ（#390/#396 は未解決で open 維持）
- [x] `agy-plugin-kit/validator` の `runHook` を agy payload対応（`resolveEditedFile`/`isManifestFile` に分離、`toolCall.args.TargetFile`）＋ `main_test.go` 追加
- [x] `agy-plugin-kit/hooks.json` を再同梱（matcher `.*`・相対 `validator/validator --hook`、Linux）
  > 実機で「既存編集は `replace_file_content`」を発見しツール名ガードを撤廃（`TargetFile` 有無で判定）。LESSONS #36。
- [x] バイナリ決定論再ビルド＋実 agy セッションで end-to-end 実証（manifest 編集→C2 検出・非ブロッキング）
- [x] コミット・CI 確認・PR（ユーザー承認後）
  > コミットをプッシュし、既存の PR #13 を更新完了。

## フェーズ5: レトロステータス・プロファイラー (retro-status) プラグインの追加（完了）

TODO数やコード総行数、コミット頻度といったリポジトリのメタデータをRPGのステータス（残りモンスター、ダンジョンの深さ、攻撃力、等）に見立てて、ファミコン風の美しいアスキーアートで出力するMCPプラグインを追加。

- [x] `retro-status/go.mod` の作成と依存関係の定義
- [x] `retro-status/main.go` の実装（スキャナー、RPGステータスマッピング、罫線幅を考慮したAA出力エンジン、デバッグ用スタンドアロン起動 `--standalone`）
- [x] 単体テスト (`retro-status/main_test.go`) の実装と合格確認
- [x] プラグイン設定ファイル (`mcp_config.json`, `gemini-extension.json`, `README.md`) の作成
- [x] エージェント用スキル (`retro-status/skills/retro-status-gemini/SKILL.md`) の作成
- [x] 全体ビルドスクリプト (`build.sh`, `build.ps1`) へのビルド対象追加
- [x] 決定論的ビルド確認（両OSバイナリ生成）と手動動作検証
- [x] コミット・リモートへのプッシュ（PR #13 に統合）

## フェーズ6: agy 1.0.10 での hook と rule の動作確認（2026-06-20）（完了）

agy 1.0.9 から 1.0.10 へのアップデートに伴い、フック（hooks）およびルール（rules）の挙動を実機で再検証する。

- [x] 検証用のダミープラグイン（`hook-test`）の作成とインストール
- [x] 対話セッションにおける hooks の発火検証
  - [x] `${extensionPath}` や `${/}` などのパス変数が置換されるようになったか確認（置換バグ継続、`${/}` はクラッシュ原因）
  - [x] payload に含まれる情報（`toolCall` や `TargetFile`）のスキーマを確認（1.0.9 と同等のスキーマで動作、動的リロードも確認）
- [x] rules 機能の検証（システムプロンプト `<user_rules>` 等への注入状況）
  - [x] プラグイン内 `rules/` フォルダの md ファイル（非機能）
  - [x] プラグイン `plugin.json` 内の `"rules"` 定義（非機能）
  - [x] プロジェクト固有の `rules` / `.agents/AGENTS.md` / グローバルルール等（.agents/AGENTS.md が初めて正常に注入されるようになった）
- [x] 検証結果を `hook-investigation-report.md` に追記
- [x] `PLAN.md` と `LESSONS.md` の更新

### フェーズ6 追補: Linux 実機再現・機構解析・全伝播（2026-06-20）

macOS 予備検証（§6 旧版）を Linux で厳密再現し、機構を確定して全ドキュメントへ伝播した。

- [x] **Linux・4経路 marker・clean install・新規セッション**で `<user_rules>` を逐語ダンプ再現
  > `.agents/AGENTS.md`=✅注入／プラグイン `rules/*.md`・`plugin.json "rules"`・グローバル `~/.gemini/rules`=❌全滅。出力は `RULE-AGENTS-110-OK` のみ。
- [x] **strings 機構解析**: `customizations.agentsCustomization`（Global / Workspace=`.agents/` の2 Customization Root から `AGENTS.md` discover）→ `mixins.UserRulesSection` が `<RULE[%s]>` 整形。1.0.9 の「discover ≠ inject」が `AGENTS.md` 経路のみ配線された（`hook-investigation-report.md` §6.3）。
- [x] 確定見出しを全 doc/memory へ伝播（CLAUDE.md / README.md / agy-plugin-kit/README.md / LESSONS #41-42 / メモリ3件 + MEMORY.md）
  > 見出し: **rules が使えるのは project `.agents/AGENTS.md` だけ。プラグイン rules/ は依然不可＝skills/ 維持。**
- [x] 検証フィクスチャ（使い捨てプラグイン・グローバル rule・ルート `.agents/AGENTS.md`）を撤去（リポジトリ常設せず）
- [x] upstream **#396** をスコープ縮小コメントで更新（project `.agents/AGENTS.md` 解消・plugin/global 3経路は継続非注入／[comment](https://github.com/google-antigravity/antigravity-cli/issues/396#issuecomment-4755551315)）
  > #390（`${extensionPath}`/`${/}` 置換）は今回見送り。1.0.10 でも再現する事実は §6.1／LESSONS #42 に記録済み。

## フェーズ7: 設定アドバイザー (settings-advisor) プラグインの追加（完了）

- [x] `settings-advisor/gemini-extension.json` の作成
- [x] `settings-advisor/models.json` によるモデル仕様の定義
- [x] `settings-advisor/skills/settings-advisor-gemini/SKILL.md` の作成（AI別接尾辞 `-gemini` ルール準拠）
- [x] `settings-advisor/src/main.go` / `main_test.go` / `go.mod` / `go.sum` 実装およびテストパス確認
- [x] dispatcher `settings-advisor/bin/settings-advisor` 作成および実行ビット `100755` 設定
- [x] 全体ビルドスクリプト (`build.sh`, `build.ps1`) への追加と決定論ビルド成功確認
- [x] PR #16 レビュー指摘対応（行カウント/tier判定/Windowsパス/権限優先）・squash マージ完了
  > bot（gemini/codex）10指摘を現コードへトレースし採用6・stale 2に分類。tier の `||`→`&&` バグと Windows パス検知失敗（実機で再現→解消）を修正、回帰テスト2件追加。Go 1.26.4 で全 OS 再ビルドし CI 4チェック pass。LESSONS #45（テスト緑≠カバー）/#46（Scanner→ReadSlice の末尾行）/#47（単一値キーの優先順位）。

## フェーズ8: Codex Security リポジトリスキャン（完了）

### 目的

- `C:\Users\kiwar\Code\agy-plugins` 全体を対象に Codex Security の repository-wide scan を実施し、脅威モデル、発見、検証、attack-path 分析、最終レポートを成果物として残す。

### 変更対象ファイル

- `PLAN.md`
- スキャン成果物: `C:\tmp\codex-security-scans\agy-plugins\085d813_20260617-002836\`

### 主要ステップ

- [x] スキャンスキル、成果物パス規約、hard rules を確認
- [x] サブエージェント利用承認を取得
- [x] Codex goal を作成
- [x] threat-model フェーズ
- [x] finding-discovery フェーズ
- [x] validation フェーズ
- [x] attack-path-analysis フェーズ
- [x] final markdown / HTML report 作成

### 確認方法

- 各フェーズの成果物が規定パスに存在すること
- coverage ledger が対象ファイルまたは worklist row を `reportable` / `suppressed` / `not_applicable` / `deferred` のいずれかで閉じていること
- candidate ledger が discovery / validation / attack-path receipt を持つこと
- `report.md` と `report.html` が作成されていること

### 想定リスク（影響範囲）

- 既存未追跡ファイル `github/mcpServers/` はユーザー作業として扱い、巻き戻さない。
- スキャン成果物はリポジトリ外 `C:\tmp\codex-security-scans\agy-plugins\...` に保存し、ソースツリーへの変更は `PLAN.md` の進捗更新に限定する。
- repository-wide scan はファイル数に応じて時間がかかるため、coverage ledger に明示的な deferred closure が必要になる可能性がある。

### 結果

- repository-wide Codex Security scan 完了。
- 最終レポート:
  - `C:\tmp\codex-security-scans\agy-plugins\085d813_20260617-002836\report.md`
  - `C:\tmp\codex-security-scans\agy-plugins\085d813_20260617-002836\report.html`
- Reportable findings:
  - `FD-KITVAL-001`: validator `--fix-paths` が symlinked `mcp_config.json` を辿り、選択 plugin 外の config を書き換え可能（medium / P2）。
  - `FD-KITCMD-001`: `/agy-plugin-kit:doc` が対象 plugin の CLI `--help` 実行を trust/confirmation gate なしで指示する（medium / P2）。
- `rank_input.csv` / `deep_review_input.csv` は 15 行、`work_ledger.jsonl` は 15 receipt。
- `report.md` は Codex Security report validator 通過済み。`report.html` 生成済み。

### 未解決事項

- `github/mcpServers/` はスキャン前から未追跡の既存作業として残した。
- スキャンは report 生成まで完了。修正実装は未実施。
