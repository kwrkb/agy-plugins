# go-lsp プラグイン

このプラグインは、Goの言語サーバーである [gopls](https://pkg.go.dev/golang.org/x/tools/gopls) を利用した MCP サーバーを提供します。
Goコードベースでの定義元ジャンプ、参照元の検索、ホバーによるドキュメントや型情報の表示を正確に行うことができます。

## 必要な前提条件

このプラグインを実行するには、システムの `PATH` に `gopls` バイナリがインストールされている必要があります。

**インストール例:**
```bash
go install golang.org/x/tools/gopls@latest
```

## インストール方法

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/go-lsp
```

## 提供されるツール

* **`go_definition`**: 指定したGoソースファイルの位置にあるシンボルの定義元を検索します（1-indexed）。
* **`go_references`**: 指定したGoソースファイルの位置にあるシンボルの参照元をすべて検索します（1-indexed）。
* **`go_hover`**: 指定したGoソースファイルの位置にあるシンボルの型情報やホバー用のドキュメントを取得します（1-indexed）。

## スキル (`SKILL.md`)

このプラグインには、AIエージェントが Go LSP を使ってコード定義やドキュメントを検索し、Goコードの理解とナビゲーションを効率的に行うための `SKILL.md` が同梱されています。
