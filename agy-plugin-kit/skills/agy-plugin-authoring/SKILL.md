---
name: agy-plugin-authoring
description: agy（Antigravity CLI）プラグインを正しく作るための実証済みオーサリング規則。プラグインの新規作成・マニフェスト記述・mcp_config/hooks 設計・Windows 対応・URL install を行う際に必ず参照する。
---

# agy プラグイン オーサリング規則（実機検証済み）

agy プラグイン開発で繰り返し踏んできた落とし穴を規則化したもの。新規プラグイン作成・
`plugin.json`/`mcp_config.json`/`hooks.json` 記述・Windows 対応・URL install の前に参照する。
詳細な背景はリポジトリ `LESSONS.md`、確定事実はメモ `agy-plugin-format-facts` を参照。

## マニフェスト形式の選択（最重要）

- **`plugin.json` 形式（native。これを使う）**: ルート直下または `.antigravity-plugin/plugin.json` / `.gemini-plugin/plugin.json`。
  `mcp_config.json` はそのままコピーされる（変数の書き換えは起きない）。
  - **同梱バイナリは `"command": "./bin/<name>"`**。native 形式の MCP サーバーは cwd がプラグインディレクトリ（agy 1.1.28〜）なので、
    プラグインルート相対のパスがそのまま正式な書き方（upstream #390 のコラボレーター回答）。agy 1.4.12 の Linux / Windows で実機確認、
    Windows は拡張子なしの `command` に agy が `.exe` を補い `bin/<name>.exe` を起動する（LESSONS 2026-11-04）。
  - マニフェストをサブフォルダに置いても `mcp_config.json`・`hooks.json`・`skills/` 等のコンポーネントは**プラグインルート直下**に置く
    （`.antigravity-plugin/` 内に置くと読まれない。1.4.2 で `agy plugin validate` / `install` により確認、出典は同梱 changelog）。
- **`gemini-extension.json` 形式（legacy）**（`plugin.json` を置かない）: `agy plugin install` 時に `${extensionPath}`/`${/}` を
  絶対パスへ解決し、install 先に `plugin.json`+`mcp_config.json` を生成する。upstream は legacy の取り込み用と位置づけている。
  新規には使わない（本リポジトリも 2026-11-04 に native 形式へ移した）。
- **両方置かない**（C1 警告対象）。`plugin.json` 側が優先され copy-only 化し、`gemini-extension.json` の内容は使われない。

## 変数展開仕様と `command` フィールドの罠（Issue #390 / `${PLUGIN_ROOT}` / `${PLUGIN_DATA}`）

- **`${extensionPath}` / `${/}`**: `gemini-extension.json` の install 時変換でのみ展開される（legacy 専用で仕様どおり。Issue #390）。
  native `plugin.json` 形式の `mcp_config.json` に書くと **literal のまま**残り起動不能になる（C2 エラー）。
- **`${PLUGIN_ROOT}` / `${PLUGIN_DATA}`（agy 1.3.2〜）**:
  - MCP サーバーのプロセス環境変数 `PLUGIN_ROOT`（`~/.gemini/config/plugins/<name>`）と `PLUGIN_DATA`
    （`~/.gemini/antigravity-cli/plugin_data/<name>`、install 更新でも消えない永続領域＝Issue #554 解消。ただし `.agents/plugins/` 直置き時は空文字）が自動注入される。
  - native `mcp_config.json` の **`args` / `env`（および 1.4.0〜 の HTTP `headers`）では実行時に絶対パスへ展開される**。
  - **罠**: native `mcp_config.json` の **`command` フィールド自体では `${PLUGIN_ROOT}` が展開されず起動に失敗する**（1.4.2 実機検証済み、C2 エラー）。
    `gemini-extension.json` の `command` に書いても install は `${PLUGIN_ROOT}` を書き換えず、生成先 `mcp_config.json` に literal のまま残る（1.4.2 実測。マニフェスト形式を問わず C2 エラー）。
- **対処**: (a) 同梱バイナリは native `mcp_config.json` に `"command": "./bin/<name>"` と書く（`command` の
  `${extensionPath}`/`${PLUGIN_ROOT}`/`${PLUGIN_DATA}` はすべてこれで置き換える） / (b) `${extensionPath}${/}` が残っている
  `mcp_config.json` は `/agy-plugin-kit:doctor` の path-fix（validator `--fix-paths`）で `./` 相対へ書き換えられる（ソースに実行してよい。
  `${PLUGIN_ROOT}`/`${PLUGIN_DATA}` は置換しないので手で直す） / (c) PATH 上のコマンドにして同梱をやめる。

## マルチプラットフォーム対応（同梱 Go バイナリ・`src/`＋`bin/`）

- `command` は **1 本の固定文字列**で OS 別に切り替えられない。Go のネイティブは OS×arch ごとに別物なので、
  **dispatcher 方式**で単一 command に畳み込む（実機検証済み）:
  - ソースは `<plugin>/src/`、配布物は `<plugin>/bin/`。`bin/` に `<name>-linux-amd64` / `<name>-darwin-arm64` /
    `<name>.exe`（ネイティブ）＋拡張子なし `bin/<name>`（**POSIX sh dispatcher**＝OS 非依存テキスト。`uname` で実機
    判定し対応ネイティブを `exec`）。`command` は `./bin/<name>`（拡張子なし。`.exe` を付けると C6）。Windows は agy が `.exe` を
    補完して `bin/<name>.exe` を直接起動＝dispatcher 非経由。
  - dispatcher はエラーのみ stderr（stdout は NDJSON 専用）。実行ビット（`100755`）をコミット
    （`git update-index --chmod=+x`。install コピーで +x は保持＝実機検証済み）。`.sh`/`.cmd`/`.bat` の直 `command`
    は Windows で spawn 不可（C3）だが、dispatcher は拡張子なし＋Windows は `.exe` 直起動なので C3 を踏まない。
- **`bin/` の PATH 自動公開（agy 1.3.2〜）**: `agy plugin install` されたプラグインの `bin/<tool>` は `~/.gemini/antigravity-cli/bin`
  経由でエージェントの `PATH` 先頭に入り、`run_command` からコマンド名だけで直接実行できる（ただしその子プロセスには
  `PLUGIN_ROOT`/`PLUGIN_DATA` 環境変数は自動注入されないため、パス解決は `$0` 基準で行う）。
- `${CLAUDE_PLUGIN_ROOT}` は **agy に存在しない**（Claude Code 専用）。使うと解決されない（C8 エラー）。

## 同梱バイナリ

- 同梱するネイティブ/スクリプトが **`.gitignore` にマッチすると URL install で clone されず**起動不能になる
  （C4 エラー）。**明示的にコミット**すること。Go プラグインはソース（`src/`）と `bin/` のネイティブ・dispatcher を
  すべてコミット。再ビルドはルートの `./build.sh <name>`（Windows は `./build.ps1`、Go バージョン固定で決定論）。
- MCP サーバー（および Go ラッパー）は **stdout に一切書かない**（MCP は stdout で NDJSON を流すため）。
  診断は stderr へ（C9）。トークン解決は shell の `${A:-${B:-}}` 同様、空文字も未設定扱いでフォールバックする。

## コンポーネント構造

- **`skills/<name>/SKILL.md`（主力）**: 呼び出し時（オンデマンド）にロードされる知識。YAML frontmatter の `name` と
  `description` が必須で、**`name` はディレクトリ名と完全一致**させる（1.4.1〜 `agy plugin validate` で検査、本文 24KB 上限）。
  `commands/<group>/<name>.toml` も install 時に skills に変換される。
- **`hooks.json`（agy 1.0.9〜で利用可）**: `PostToolUse` payload の `toolCall.args.TargetFile` で編集ファイルの絶対パスを取得可能。
  ただし **1.4.2 でも `hooks.json` の `command` 内では `${extensionPath}`・`${PLUGIN_ROOT}`・`${/}` は一切展開されない**
  （`sh -c` が未定義変数として空文字展開し `/hook.sh: No such file` になるか、`${/}` で `Bad substitution` になる＝C10 警告）。
  フック実行時の `pwd` はプラグインルートになるため、command は必ず **PWD 相対パス**で書く。
  - **Windows は `cmd /c`、それ以外は `sh -c` で起動される**ので、同梱バイナリは Windows 形と Unix 形を `||` でつなぐ
    （JSON では `"validator\\bin\\validator --hook || validator/bin/validator --hook"`）。`cmd` は `validator/bin/...` の `/bin` を
    スイッチとして読み `validator` を PATH から探すため、スラッシュ区切りだけだと Windows で起動しない（PATH に無ければ
    `PostToolUse` が失敗して全ツール結果がエラー化、同名コマンドがあればそれが黙って動く）。`sh` では前半のバックスラッシュが
    エスケープになって not found で落ち、後半が動く。パスをクォートで囲む案は agy が `\"` にエスケープするので不可。
    プラグインの `bin/` はフック実行時の PATH に入らない（1.4.2 Windows 実測・Issue #30）。
  - **`timeout` は秒単位**（既定 30）。Claude Code のミリ秒の値（例: `10000`）を書くと約 2.8 時間の意味になる（1.4.2 実測）。
  - **フックの出力は agy が JSON として厳格に解釈する**。`PostToolUse` の stdout は `{}` 以外を書かない。未知の項目を返したり
    異常終了・timeout したりすると、ツール結果全体がモデルからはエラーに見える（書き込み自体は成功していても）。
  - **`PostToolUse` からはモデルに何も届かない**（1.4.2 実測）: stderr は CLI ログにしか出ず、stdout の `injectSteps` は
    unknown field でエラー、`overwriteResult` は無視される。モデルに届けたい時は、`PostToolUse` が `conversationId` ごとの
    ファイルに書き、`PreInvocation` が読んで `{"injectSteps":[{"ephemeralMessage":"..."}]}` を返す（同じターン内に届く。本キットの validator がこの方式）。
    `PreInvocation` のフックが失敗（非 0 終了・JSON でない出力・コマンド不在）してもモデル呼び出しは止まらず、CLI ログに残るだけ（1.4.2 実測）。
  - 構造は agy の名前付き形式 `{"<hook-name>": {"PostToolUse": [{"matcher": ".*", "hooks": [...]}], "PreInvocation": [{...}]}}`。
    `PreToolUse`/`PostToolUse` は `matcher`＋`hooks` のグループ、`PreInvocation`/`PostInvocation`/`SessionStart`/`Stop` はハンドラを直接並べる。
- **`rules/*.md`（agy 1.4.2 で機能化確認・用途を絞って使う）**: 1.0.8〜1.0.15 ではプラグイン `rules/` が `<user_rules>` に
  注入されなかったが、**1.4.2 実機検証で YAML frontmatter（`trigger: always_on | model_decision | glob` 必須、24KB 上限）付きの
  `rules/AGENTS.md` が `<user_rules>` に自動注入されることを確認**（ルート直下の `AGENTS.md` は validate 警告となるため `rules/` 配下に置く）。
  ただし `always_on` ルールは全ルール共有の常時 20,000 トークン予算を毎ターン消費するため、**MCP ツールの詳細な引数仕様や手順は
  引き続きオンデマンドの `skills/<name>/SKILL.md` に置き**、`rules/` は常時強制したい短いガードレールに限定する。
- **`agents/<name>.md`（サブエージェント・agy 1.3.2 / 1.4.1〜で機能化確認）**: 旧 `agent.json` は CLI に無視され validate 警告となる。
  `agents/<name>.md` に YAML frontmatter（`name`, `description` 必須）と本文を書くと `invoke_subagent` で呼び出せ、親プラグインの MCP サーバーも利用できる。

## 検証

- **ソースには本キットの `validator <src-dir>` と `agy plugin validate <src-dir>` の両方をかける**:
  - `agy plugin validate <path>`（`--strict` / `--json` 対応）は 1.4.0 から **`plugin.json` の存在を直接要求する**。native 形式のソースならそのままかけられる。
    legacy の `gemini-extension.json` 形式のソースでは失敗するので、install 後の `~/.gemini/config/plugins/<name>` にかける。
  - 本キットの validator は `agy plugin validate` が見ない罠（#390・`command` の変数・hooks の変数・gitignore 等）を見る。
- 「動く」の証明は **実環境（端末から agy）** で行う（手動 export の直叩きは不可）。証拠は
  **`~/.gemini/antigravity-cli/mcp/<plugin>_<server>/` 内のツール名**で取る。
  - **ディレクトリ名は `<plugin>_<server>`**（例 `github_github`）。`mcp/<plugin>/` という旧名の
    ディレクトリが**古い内容のまま残っている**ことがあり、そちらを見ると mtime もツール名も「それらしく」通ってしまう。消す時・見る時とも
    `<plugin>_<server>` を指すこと。
  - 起動確認の前に **現行の `mcp/<plugin>_<server>/` も削除する**。残したままでは、今回のサーバーが
    起動に失敗しても前回のツール JSON を起動成功の証拠と誤認する。ただしこれはユーザーの
    `~/.gemini/` を変更する操作なので、**消すディレクトリの絶対パスと理由を示して承認を得てから**行う。
    承認が得られなければ削除せず、キャッシュのツール JSON は起動成功の証拠に使えない（前回の残り
    かもしれない）ことを検証結果に明記する。
  - **mtime だけでは不十分**。設計を作り替えた時は中身のツール名で新サーバーを別人確認する。
  - **`agy -p`（headless）で確認できるのは起動とツール探索まで**。headless では MCP
    ツールの実行を自動拒否する（権限プロンプトを出せないため）ので、ツール実行まで確かめるなら
    対話セッションで起こして許可プロンプトに応答する。
- ソースを再ビルドしたら **`agy plugin install` の再実行**が必要。

## このキットのツール

- `/agy-plugin-kit:new <name>` 正しい雛形を生成 / `/agy-plugin-kit:validate <path>` 上記トラップを機械検出 /
  `/agy-plugin-kit:doctor <path>` 検出＋修正提案（#390 の path-fix 含む） / `/agy-plugin-kit:doc <path>` README/SKILL 生成。
