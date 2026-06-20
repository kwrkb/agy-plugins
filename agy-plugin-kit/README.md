# agy-plugin-kit — agy プラグイン開発メタ・ヘルパー

agy（Antigravity CLI）プラグインを**正しく・速く量産する**ための「プラグイン開発者のためのプラグイン」。
実機検証で蓄積した落とし穴（リポジトリ `LESSONS.md`、特に Issue #390 = `${extensionPath}` が native
`plugin.json` 形式で解決されない問題）を、雛形生成・静的検査・自動修正・ドキュメント生成として提供します。

## 提供物

| コンポーネント | 内容 |
| :-- | :-- |
| **`validator/`**（Go・`src/`＋`bin/`） | agy 非依存の決定的バリデータ。C1〜C10 のトラップを検出（下表）。`--fix-paths` モードあり。`bin/` に linux-amd64 / darwin-arm64 / windows のネイティブと OS 分岐 dispatcher を同梱 |
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

`validator/bin/` のネイティブバイナリ（`validator-linux-amd64` / `validator-darwin-arm64` / `validator.exe`）は `validator/src/main.go` から `go build` した成果物を**全 OS 分とも同梱**（コミット済み）。再ビルドはリポジトリルートのビルドスクリプトを使います（**Go 1.26.4**。決定論フラグはスクリプトに集約。CI の検証ゲート `.github/workflows/build-verify.yml` がこの結果との bit-identical 一致を要求し、Go のバージョンがずれると fail します）。

```bash
./build.sh validator    # validator のネイティブバイナリ（linux-amd64/darwin-arm64/windows）を再ビルド。Windows は ./build.ps1 validator
# 引数なし（./build.sh / ./build.ps1）で全プラグイン
```

> **ビルドする OS によってスクリプトを使い分ける**: macOS / Linux は `./build.sh`、Windows は `./build.ps1`。
> どちらも `CGO_ENABLED=0` のクロスコンパイルで 3 OS 分を 1 台で一括生成する（各 OS で実機ビルド不要）。

> **対応 OS（Linux / macOS / Windows）**: 拡張子なし `validator/bin/validator` は `uname` で実機判定し
> `validator-{linux-amd64,darwin-arm64}` を `exec` する POSIX sh **dispatcher**（テキスト＝OS 非依存。Windows は
> `validator.exe` 直起動で非経由）。agy は `${extensionPath}` 以外を置換しないため、これが OS 別バイナリを単一
> `command` に畳み込む唯一の手段。dispatcher・ネイティブとも実行ビット（`100755`）を git にコミット（`.gitignore` 対象外）。

## マニフェスト形式（このキット自身）

native `plugin.json` を採用。キットは MCP サーバーを持たないため `${extensionPath}` 置換が不要で、
copy-only で問題ありません（自身が C1 を踏まない）。

## `hooks.json` の自動バリデーション（agy 1.0.9〜、Linux / macOS / Windows）

`hooks.json` を同梱し、**マニフェスト（`plugin.json` / `gemini-extension.json` / `mcp_config.json` /
`hooks.json`）を編集すると validator が自動で走り**、見つかった問題を stderr に助言として出します
（非ブロッキング・常に exit 0）。仕組み:

- `PostToolUse` の `matcher: ".*"`、`command: "validator/bin/validator --hook"`（PWD=install 先の相対パス。
  `${extensionPath}`/`${/}` は使わない＝C10/C2 を踏まない）。`validator/bin/validator` は OS 分岐 dispatcher で、
  hook 経路でも shebang 解釈で実機ネイティブを `exec` する（実機検証: macOS arm64 で発火確認）。
- ツール種別の選別は **validator 側のコードで** 行う（`toolCall.args.TargetFile` が入る編集系ステップだけ
  反応し、マニフェスト basename 以外は無音）。matcher にツール名を書かない（agy のツール名は
  新規作成=`write_to_file` / 既存編集=`replace_file_content` と版で異なるため）。

> **3 OS 対応**: `command` は1本の固定文字列で agy は `${extensionPath}` 以外を置換しないが、
> `validator/bin/validator` を OS 分岐 dispatcher（shebang sh）にすることで Linux / macOS（arm64）/ Windows を
> 単一 command でカバーする（Windows は `.exe` を直接起動）。フックが使えない環境でも `/agy-plugin-kit:validate` /
> `:doctor` を手動で使える。

### 経緯（1.0.8 では非機能 → 1.0.9 で再導入）

当初の `hooks.json` は **agy 1.0.8 で実機検証した結果フックが成立せず**（payload に編集ファイルが無い
＝`toolCall: null`、発火が不安定、`agy -p` 非発火）、一度同梱を取りやめた（agy upstream
[#395](https://github.com/google-antigravity/antigravity-cli/issues/395)）。
**agy 1.0.9 で再検証した結果、payload に `toolCall.args.TargetFile`（編集ファイル絶対パス）が入り、
2 回目以降の編集・`agy -p` でも発火、自前バイナリを PWD 相対で呼べる**ようになったため再導入した
（#395 は解決済みでクローズ／詳細はリポジトリ LESSONS #34、`hook-investigation-report.md` §5）。
**agy 1.0.10 ではフックの動的リロードを確認**（install 後、親セッションに次ツール実行から自動適用＝再起動不要、LESSONS #42／§6.1）。
なお `${extensionPath}` 非置換（[#390](https://github.com/google-antigravity/antigravity-cli/issues/390)）は 1.0.10 でも継続。
`rules/` はプラグイン経路（プラグイン内 `rules/`・`plugin.json "rules"`・グローバル `~/.gemini/rules`）が
[#396](https://github.com/google-antigravity/antigravity-cli/issues/396) で**依然非機能**（知識は `skills/` で渡す）。
ただし **1.0.10 でプロジェクトルートの `.agents/AGENTS.md` のみ注入されるようになった**（LESSONS #41／§6）。

## ライセンス

このキットのコード（validator / commands / templates）は本リポジトリのライセンスに従います。
