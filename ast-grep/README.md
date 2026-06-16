# ast-grep プラグイン

このプラグインは、[ast-grep (`sg`)](https://ast-grep.github.io/) CLI を利用した MCP サーバーを提供します。
テキストベースの検索（正規表現など）では難しい、**抽象構文木（AST）に基づいた正確なコード構造の検索とリファクタリング**を可能にします。

## 必要な前提条件

このプラグインを実行するには、システムの `PATH` に `ast-grep` バイナリがインストールされている必要があります（Linux では `sg` は `setgroups` コマンドと衝突するため、フルネームの `ast-grep` を使用します）。

**インストール例（macOS / Linux）:**
```bash
brew install ast-grep
# または
npm install -g @ast-grep/cli
```

## インストール方法

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/ast-grep
```

## 提供されるツール

* **`ast_search`**: 指定したディレクトリ内のファイルを対象に、ASTパターン検索を行います。マッチした結果（ファイル名や行番号、キャプチャされた変数）を JSON で返します。
* **`ast_replace`**: 指定したディレクトリ内のファイルを対象に、ASTパターン検索と構造的置換を一括で行います（ファイルは直接上書き更新されます）。

## スキル (`SKILL.md`)

このプラグインには、AIエージェントが `ast-grep` の独自のメタ変数（`$A` や `$$$ARGS` など）を正しく扱い、いきなり置換せずに検索から入るというワークフローを教え込むための `SKILL.md` が同梱されています。
