---
name: doc
description: "既存の agy プラグインの構成を読み取り、README.md / SKILL.md を生成・更新する。agy プラグインのドキュメントを書くとき、`/agy-plugin-kit:doc <パス>` で呼ばれたときに使う。"
---

# agy プラグインのドキュメントを生成・更新する

引数は対象ディレクトリのパス。`/agy-plugin-kit:doc <path>` のように呼ばれたときは、その後ろに続く文字列を引数とみなす。
引数が無ければ会話から読み取り、確信が持てなければユーザーに尋ねる。

手順:
1. 対象の構成を読み取る: `plugin.json`/`gemini-extension.json`（name・description）、`mcp_config.json`
   （提供する MCP サーバーと command）、`skills/`（提供する skill。`/<plugin>:<skill>` で呼べる）、`hooks.json`。
   native 形式で `commands/` がある場合、それは読み込まれないので機能として書かず、skill への移行を提案する。
2. その情報から README.md を生成/更新する。含める節:
   - 概要 / 対応 OS / 構成（ファイル表）/ 必要条件（PATH に要るバイナリ・認証）/ インストール
     （`agy plugin install https://github.com/<owner>/<repo>/<plugin>`）/ ライセンス・帰属。
   - リポジトリ既存プラグイン（`github/README.md` 等）のトーンに合わせる。
3. プラグインが既存 CLI をラップする場合は、対象 CLI が信頼できる正規バイナリであることを確認した上で（または静的ドキュメント・ソースコードを参照して）`--help` を読み、SKILL.md を生成する。未知・未検証のバイナリに対して無条件にコマンドを実行しないこと。
4. 生成物は事実に基づくこと（推測でコマンドや認証を書かない）。不明点はソースを読むか不明と明示する。
5. 仕上げに `~/.gemini/config/plugins/agy-plugin-kit/validator/bin/validator <対象>` を実行し、ドキュメント追加で
   構造が壊れていないか確認する。
