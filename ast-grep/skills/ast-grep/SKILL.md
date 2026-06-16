---
name: ast-grep
description: ast-grep (sg) による高速なASTベースの検索・置換（リファクタリング）操作スキル
---

# ast-grep スキル

このスキルは、`ast-grep` プラグインを使用して、コードベースの構造的検索やリファクタリングを安全かつ正確に行うための知識を提供します。

## ワークフローの鉄則

1. **いきなり置換 (`ast_replace`) しない**:
   必ず先に `ast_search` を使ってパターンが正しく意図したノードにマッチするかを確認すること。
2. **言語指定 (`language`) は必須**:
   `sg` は拡張子に基づく自動推論も行いますが、意図せぬファイルへの適用を防ぐため、`language` 引数（例: `go`, `typescript`, `python` 等）は必ず明示的に指定してください。

## 検索・置換パターンの基本記法（メタ変数）

`ast-grep` では、特定の構文ノードをキャプチャするために独自の「メタ変数」を使用します。

### 1. 単一ノードのキャプチャ: `$A`
`$` の後に大文字アルファベットの識別子（例: `$A`, `$VAR`, `$FUNC`）を付けると、単一のASTノード（変数、式、ブロックなど）にマッチします。

* **Search**: `fmt.Println($A)`
* **Rewrite**: `log.Println($A)`
* **マッチ例**: `fmt.Println("hello")`, `fmt.Println(x + y)` にマッチし、`$A` に引数の式がキャプチャされます。

### 2. 複数ノード（可変長）のキャプチャ: `$$$ARGS`
`$$$` の後に大文字アルファベットを付けると、複数の連続するノード（引数のリストや、関数内の複数行のステートメントなど）にマッチします。

* **Search**: `log.Printf($MSG, $$$ARGS)`
* **Rewrite**: `logger.Info($MSG, $$$ARGS)`
* **マッチ例**: `log.Printf("error: %v, code: %d", err, code)` の場合、`$MSG` に `"error: %v, code: %d"` が入り、`$$$ARGS` に `err, code` が入ります。

## 頻出パターン例

### 引数の順番を入れ替える
* **Search**: `assert.Equal($ACTUAL, $EXPECTED)`
* **Rewrite**: `assert.Equal($EXPECTED, $ACTUAL)`

### エラーハンドリングの一括修正 (Go)
* **Search**:
```go
if $ERR != nil {
  return $ERR
}
```
* **Rewrite**:
```go
if $ERR != nil {
  return fmt.Errorf("wrapped error: %w", $ERR)
}
```

### 不要なラッパーの削除
* **Search**: `Promise.resolve($VAL)`
* **Rewrite**: `$VAL`

## トラブルシューティング

* **パターンがマッチしない場合**: コードのフォーマットやセミコロンの有無などの表面的な違いは `ast-grep` が自動で吸収しますが、言語の構文としてパースできない不完全なスニペットは検索できません。パターンは「有効な構文の断片」である必要があります。
* **複雑すぎるリファクタリング**: `ast_replace` だけで完結させるのが難しい複雑な条件（名前が `test` で始まる変数だけ置換したい、など）がある場合は、`ast_search` でマッチした結果の JSON からファイルパスと行番号を抽出し、`agy` 本体が持つ `multi_replace_file_content` 等を組み合わせて慎重に編集してください。
