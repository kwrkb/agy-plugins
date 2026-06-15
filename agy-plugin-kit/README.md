# agy-plugin-kit — agy プラグイン開発メタ・ヘルパー

agy（Antigravity CLI）プラグインを**正しく・速く量産する**ための「プラグイン開発者のためのプラグイン」。
実機検証で蓄積した落とし穴（リポジトリ `LESSONS.md`、特に Issue #390 = `${extensionPath}` が native
`plugin.json` 形式で解決されない問題）を、雛形生成・静的検査・自動修正・ドキュメント生成として提供します。

## 提供物

| コンポーネント | 内容 |
| :-- | :-- |
| **`validator/`**（Go `.exe`） | agy 非依存の決定的バリデータ。C1〜C9 のトラップを検出（下表）。`--hook` / `--fix-paths` モードあり |
| **コマンド** `/agy-plugin-kit:new` | マニフェスト形式を自動選択して正しい雛形を生成（量産エンジン） |
| **コマンド** `/agy-plugin-kit:validate` | 対象プラグインを静的検査して要約 |
| **コマンド** `/agy-plugin-kit:doctor` | `agy plugin validate` ＋ キット検査 ＋ 修正（Issue #390 の絶対パス自動埋め込み含む） |
| **コマンド** `/agy-plugin-kit:doc` | 既存プラグインから README / SKILL.md を生成 |
| **スキル** `agy-plugin-authoring` | 12 痛点→オーサリング規則（エージェントが正しい流儀を引ける知識） |
| **`templates/`** | `/new` が複製する雛形群（plugin.json / gemini-extension.json / Go ラッパー 等） |
| **`hooks.json`** | マニフェスト編集時に validator を自動実行（**実験的**: 後述） |

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
| C10 | native `plugin.json` 形式の `hooks.json` で `${extensionPath}` 使用（実行時解決が未確認） | WARN |

## インストール

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/agy-plugin-kit
```

`validator/validator.exe` は `validator/main.go` から `go build` した成果物を同梱（コミット済み）。再ビルド/再現:

```powershell
cd agy-plugin-kit/validator
go build -o validator.exe .
```

> **対応 OS**: validator は Windows `.exe` を同梱するため**当面 Windows ターゲット**です（本リポジトリの Windows ネイティブ運用に合わせる）。Linux / macOS で使う場合は上記 `go build` でその OS 向けバイナリを作り、コマンドの参照先（`validator.exe`→`validator`）を読み替えてください。同梱 `.exe` は `validator/main.go` からビルドされ、`git` に明示コミットしています（URL install で確実に clone されるよう `.gitignore` 対象外）。

## マニフェスト形式（このキット自身）

native `plugin.json` を採用。キットは MCP サーバーを持たないため `${extensionPath}` 置換が不要で、
copy-only で問題ありません（自身が C1 を踏まない）。

## hooks.json の注意（実験的）

プラグイン同梱 `hooks.json` は install 時に **parse/processed** されることは確認済みですが、
**`agy -p`（print mode）では実行時に発火しませんでした**（PostToolUse・UserPromptSubmit とも、
catch-all matcher でも未発火）。対話セッションや `/hooks` での有効化が必要な可能性があります。
さらに hook 内の `${extensionPath}` が実行時に解決されるかも未確認です。よって自動バリデーションは
**実験的機能**とし、確実な検査は `/agy-plugin-kit:validate`（コマンド経由）を使ってください。
対話セッションで発火を確認できたら本 README を更新します。

> このキット自身を validator にかけると **C10 WARN**（native `plugin.json` 形式の `hooks.json` で
> `${extensionPath}` を使用）が出ます。これは上記の「実験的 hook」を**意図的に同梱しているため**で、
> ERROR ではありません。hook 経路が実機確認できれば C10 は解消します。

## ライセンス

このキットのコード（validator / commands / templates）は本リポジトリのライセンスに従います。
