# Antigravity CLI (agy) 1.0.8 フック機能 実機検証レポート

**調査日:** 2026年6月15日  
**対象環境:** agy 1.0.8  
**調査目的:** プラグインマニフェスト（`hooks.json`）で定義できる `PostToolUse` フックが、自動フォーマットやバリデーション用途（ファイル保存時フックなど）に実用可能かを実機検証データから判断する。

---

## 1. 調査手法

Antigravityエージェント内でダミーのプラグイン（`hook-test`）を生成・インストールし、サブエージェントから意図的にファイル編集ツール（`write_to_file`）を実行してフックを発火させた。
フックで実行するコマンドには、標準入力（stdin）、引数（argv）、環境変数、およびカレントディレクトリ（PWD）を `/tmp/` 下にファイル出力するPythonスクリプトを指定し、生データを採取した。

**使用した `hooks.json` の抜粋:**
```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": ".*",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /home/yugosasaki/code/hook-test/dump.py ${extensionPath} ${/}",
            "timeout": 10000
          }
        ]
      }
    ]
  }
}
```

---

## 2. 検証結果（5つの致命的な事実）

検証で得られた出力ダンプから、以下の事実が判明した。

### 事実1: ペイロードに「編集したファイル」の情報が一切存在しない
フックが標準入力（stdin）から受け取った生 JSON は以下の通りだった。
```json
{
  "artifactDirectoryPath": "/home/yugosasaki/.gemini/antigravity-cli/brain/1c6dffe5-6809-4b21-9064-ed729619275f",
  "conversationId": "1c6dffe5-6809-4b21-9064-ed729619275f",
  "error": "",
  "stepIdx": 21,
  "toolCall": null,
  "transcriptPath": "/home/yugosasaki/.gemini/antigravity-cli/brain/1c6dffe5-6809-4b21-9064-ed729619275f/.system_generated/logs/transcript_full.jsonl",
  "workspacePaths": [
    "/home/yugosasaki/code/agy-plugins"
  ]
}
```
**致命的な問題:**  
Claude Code 等とは異なり、`file_path` や `tool_input` が存在しない。それどころか `toolCall` も `null` となっており、**「どのファイルが編集されたのか」を特定する手段が完全に欠落**している。これにより、対象ファイルを絞って実行するバリデーション用途には利用不可能。

### 事実2: パス変数（`${extensionPath}`, `${/}`）が置換されない
引数のダンプ結果は `['/home/yugosasaki/code/hook-test/dump.py']` となり、コマンド文字列に含めた `${extensionPath}` が消滅していた。
**致命的な問題:**  
agy 側で変数が展開されずにそのままシェルに渡されている。
- `${extensionPath}` は未定義の環境変数としてシェルに評価され、**空文字列**に消滅した。
- `${/}` を含めると Linux の `/bin/sh` が変数置換エラーとみなし、`sh: 1: Bad substitution` で**フックのプロセス自体が起動前にクラッシュ**する。

### 事実3: 実行時のカレントディレクトリ（PWD）がワークスペースではない
Pythonで取得した `os.getcwd()` の結果は以下の通りだった。
```text
/home/yugosasaki/.gemini/config/plugins/hook-test
```
**致命的な問題:**  
コマンドはユーザーのワークスペースではなく、プラグインのインストール先ディレクトリで実行される。相対パスで自前の実行ファイルは呼び出せるが、前述の「ペイロードにファイルパスが含まれない」制約と合わさり、ワークスペース側のファイルにアクセスするパスを組み立てられない。

### 事実4: 発火条件が極めて限定的で不安定
- `agy -p`（プリントモード）では一切発火しない。
- 対話セッションにおいても、セッション内で最初の `Edit`（ファイル編集）ツール使用時のみ発火し、2回目以降の連続編集ではフックが呼び出されない現象が確認された（再起動等がないと状態がリセットされない可能性）。

---

## 3. 結論

現状の **`agy 1.0.8` におけるフック（`hooks.json`）機能は、ファイルと連動する自動処理（Lint、Format、Validation）を組む用途としては完全に破綻している（利用不可能）**。

「動かないからとりあえず同梱をやめる」という消極的選択ではなく、アーキテクチャの制約上「そもそも実装不能」であることが本調査で確定した。

**今後の対応方針:**
- 自動バリデーションの夢は一旦捨て、確実なコマンドトリガー（`/agy-plugin-kit:validate` や `:doctor`）に依存する設計を標準とする。
- 将来のアップデートで、フックのペイロードに `toolCall` や `file_path` が格納されるようになり、変数置換バグが修正されたタイミングで再導入を検討する。
