# agy-plugin-kit — agy プラグイン開発メタ・ヘルパー

agy（Antigravity CLI）プラグインを**正しく・速く量産する**ための「プラグイン開発者のためのプラグイン」。
実機検証で蓄積した落とし穴（リポジトリ `LESSONS.md`、特に Issue #390 = `${extensionPath}` が native
`plugin.json` 形式で解決されない問題）を、雛形生成・静的検査・自動修正・ドキュメント生成として提供します。

## 提供物

| コンポーネント | 内容 |
| :-- | :-- |
| **`validator/`**（Go `.exe`） | agy 非依存の決定的バリデータ。C1〜C10 のトラップを検出（下表）。`--fix-paths` モードあり |
| **コマンド** `/agy-plugin-kit:new` | マニフェスト形式を自動選択して正しい雛形を生成（量産エンジン） |
| **コマンド** `/agy-plugin-kit:validate` | 対象プラグインを静的検査して要約 |
| **コマンド** `/agy-plugin-kit:doctor` | `agy plugin validate` ＋ キット検査 ＋ 修正（Issue #390 の絶対パス自動埋め込み含む） |
| **コマンド** `/agy-plugin-kit:doc` | 既存プラグインから README / SKILL.md を生成 |
| **スキル** `agy-plugin-authoring` | 12 痛点→オーサリング規則（エージェントが正しい流儀を引ける知識） |
| **`templates/`** | `/new` が複製する雛形群（plugin.json / gemini-extension.json / Go ラッパー 等） |

## バリデータのチェック

| # | 検出内容 | 重大度 |
| :-- | :-- | :-- |
| C1 | `plugin.json` と `gemini-extension.json` の二重マニフェスト | WARN |
| C2 | native `plugin.json` 形式で `${extensionPath}` 使用（**Issue #390**: 解決されず壊れる） | ERROR |
| C3 | `.sh`/`.cmd`/`.bat` を `command` に直接指定（Windows で spawn 不可） | ERROR |
| C4 | マニフェスト参照バイナリが `.gitignore` 対象（URL install で消える。`git check-ignore` 判定） | ERROR |
| C5 | `plugin.json`/`gemini-extension.json` が無い・不正 JSON | ERROR |
| C6 | `${extensionPath}` 形式 command に `.exe` 拡張子 | WARN |
| C7 | トークン専用 MCP サーバーを wrapper 無しで直叩き（heuristic） | WARN |
| C8 | `${CLAUDE_PLUGIN_ROOT}`（agy に存在しないトークン） | ERROR |
| C9 | 同梱 Go ラッパーが自前で stdout に書く（NDJSON 破壊。heuristic） | WARN |
| C10 | native `plugin.json` 形式の `hooks.json` で `${extensionPath}` 使用（実機検証: 実行時に置換されず literal のまま残り、`${/}` は `/bin/sh` で `Bad substitution` になり hook 起動失敗） | WARN |

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit
```

`validator`（Linux）/ `validator.exe`（Windows）は `validator/main.go` から `go build` した成果物を**両 OS 分とも同梱**（コミット済み）。**Go 1.26.4** を使い、決定論フラグ付きで再ビルドします（CI の検証ゲート `.github/workflows/build-verify.yml` がこの結果との bit-identical 一致を要求します。Go のバージョンやフラグがずれると CI が fail します）。

```bash
cd agy-plugin-kit/validator
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags=-buildid= -o validator     .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags=-buildid= -o validator.exe .
```

> **対応 OS**: validator は Linux / Windows 両方のバイナリを同梱します。`command` の参照先は OS に応じて `validator`（Linux/macOS）/ `validator.exe`（Windows）を使い分けてください。同梱バイナリは `git` に明示コミットしています（URL install で確実に clone されるよう `.gitignore` 対象外）。

## マニフェスト形式（このキット自身）

native `plugin.json` を採用。キットは MCP サーバーを持たないため `${extensionPath}` 置換が不要で、
copy-only で問題ありません（自身が C1 を踏まない）。

## なぜ `hooks.json` を同梱しないか（agy 1.0.8 実機検証）

当初は「マニフェスト編集時に validator を自動実行する」`hooks.json` を同梱する構想でしたが、
**agy 1.0.8 で対話セッションを起こして実機検証した結果、フックによる自動バリデーションは成立しない**
と判明したため、同梱を取りやめました（確実な検査は `/agy-plugin-kit:validate` / `:doctor` コマンドを使用）。

実機検証で確認した事実:

- **発火は対話セッションのみ**。`agy -p`（print mode）では発火しない。
- **発火が不安定**。セッション内の最初の編集（特定の `stepIdx`）でのみ発火し、2 回目以降の `Edit` では
  発火しないことがある。「編集ごとに必ず検査」という用途には信頼できない。
- **agy のフック stdin payload に編集ファイル情報が無い**。実際に送られるのは
  `{"artifactDirectoryPath": "...", "conversationId": "...", "error": null, "stepIdx": 0, "toolCall": null, "transcriptPath": "...", "workspacePaths": []}`
  で、Claude Code 流の `file_path` / `tool_input` が存在せず `toolCall` も `null`。validator の `--hook` は
  `file_path` を前提に編集対象を特定するため、**発火しても対象を特定できず常に無音 no-op** になる。
- 発火時の `PWD` は**プラグインのインストール先**（`~/.gemini/config/plugins/<name>/`）。相対パスで同梱
  バイナリは呼べるが、上記のとおり対象ファイルを取得できないため意味を成さない。
- `hooks.json` 内の `${extensionPath}` / `${/}` は**実行時に一切置換されず literal のまま残る**。Linux では
  `/bin/sh` が `${/}` を `Bad substitution` と見なし hook 起動に失敗する（C10 で検出。トークン非展開は
  MCP 側と同根で agy upstream [#390](https://github.com/google-antigravity/antigravity-cli/issues/390)）。

> **upstream tracking**: 上記のうち payload に編集ファイル情報が無い件・発火が不安定な件は agy 本体へ
> [#395](https://github.com/google-antigravity/antigravity-cli/issues/395) として報告済み。`rules/` が
> 機能しない件（プラグインの知識は `skills/` で渡す）は [#396](https://github.com/google-antigravity/antigravity-cli/issues/396)。

将来 agy のフック payload が編集ファイルを露出するようになれば（#395 の解消）、`validator/main.go` の
`runHook` をその実 payload（`transcriptPath` 解析等）に対応させたうえで再導入を検討する。

## ライセンス

このキットのコード（validator / commands / templates）は本リポジトリのライセンスに従います。
