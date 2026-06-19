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

「動かないからとりあえず同梱をやめる」という消極的選択ではなく、**現行 `agy 1.0.8` のフック実装の制約上**「この用途では実装が成立しない」ことが本調査で確定した。これは agy 1.0.8 時点の挙動についての結論であり、フックの仕様・実装は将来版で変わりうる（恒久的な不可能性を主張するものではない）。

**今後の対応方針:**
- 自動バリデーションの夢は一旦捨て、確実なコマンドトリガー（`/agy-plugin-kit:validate` や `:doctor`）に依存する設計を標準とする。
- 将来のアップデートで、フックのペイロードに `toolCall` や `file_path` が格納されるようになり、変数置換バグが修正されたタイミングで再導入を検討する。

**再評価トリガー（agy バージョンアップ時）:** agy を更新したら `agy changelog` を確認し、以下が改善されていれば本調査をやり直してフック／`rules` の再導入を判断する。**（2026-06-17 / agy 1.0.9 実施結果を反映＝§5）**
- [x] フック payload に `toolCall` / `file_path`（編集ファイルパス）が入るようになったか（§2 事実1）→ **✅ 1.0.9 で解消**（`toolCall.args.TargetFile`）
- [ ] `hooks.json` 内の `${extensionPath}` / `${/}` が実行時に置換されるようになったか（§2 事実2）→ **❌ 1.0.9 でも未置換**（`${/}` は未再検証）
- [x] フックが対話セッションで安定発火するようになったか（§2 事実4）→ **✅ 1.0.9 で改善**（2回目以降の編集・`-p` でも発火、n=2）
- [ ] `rules/`（プラグイン内 / `plugin.json` / グローバル）がシステムプロンプトに注入されるようになったか（§4）→ **❌ 1.0.9 でも3パターン全滅**

---

## 4. 追記: ルール機能（`rules/`）の実機検証

公式ドキュメントで言及されている「ルール機能」についても、フックと同様に実機検証を行った。

### 調査手法
以下の3パターンでルールを定義し、サブエージェントを起動してエージェントのシステムプロンプト（`<user_rules>` セクション）にルールが注入されるかを検証した。
1. プラグイン内に `rules/dummy.md` を配置してインストールするパターン
2. プラグインの `plugin.json` 内に `"rules": ["..."]` 配列を記述するパターン
3. グローバルルールとして `~/.gemini/rules/test.md` を作成するパターン

### 検証結果
**全て全滅（読み込まれない）。**
いずれの手法を用いても、エージェントのシステムプロンプトには設定したルールが一切注入されなかった。現在プロンプトに反映されるのは、UI側の設定（Gemini Added Memories）から渡されるグローバルルールのみである。

### ルール機能の結論
現状の `agy 1.0.8` では、**ファイルベースのルール機能は完全に機能していない（未実装または不具合）**。
プラグインからエージェントに固有の知識や規約を渡したい場合、`rules/` に頼ることはできず、**必ず `skills/`（スキル）として定義して適宜呼び出させる**設計にする必要がある。

---

## 5. 追記: agy 1.0.9 での再検証（2026-06-17）

**対象:** agy 1.0.9。`hook-test` を clean install（旧 `~/.gemini/config/plugins/hook-test` 削除後に `agy plugin install`）し、§3 の再評価トリガー4項目を再実測した。**結論: hooks は部分的に機能化、rules は依然非機能。**

### 5.1 hooks — payload に編集ファイルパスが入った（事実1 解消）

`dump.py` を1呼び出し1レコードの追記式に変更し、対話セッションで2ファイル（`rvfire1.txt`/`rvfire2.txt`）を連続作成させたところ **4回発火**。`write_to_file` のステップ（stepIdx 3/6）で `toolCall` が **populated** になっていた（実証 payload）:

```json
{"stepIdx":3,"toolCall":{"name":"write_to_file","args":{"TargetFile":"/home/yugosasaki/code/agy-plugins/rvfire1.txt","CodeContent":"fire1\n","Overwrite":true,"Description":"Create rvfire1.txt with content 'fire1'","toolAction":"Creating rvfire1.txt","toolSummary":"File creation"}},"conversationId":"4cfd84b2-...","transcriptPath":".../transcript_full.jsonl","workspacePaths":["/home/yugosasaki/code/agy-plugins"],"error":""}
```

- **編集ファイルの絶対パスが `toolCall.args.TargetFile` で取得可能**（1.0.8 の事実1「特定不可」を覆す）。
- 非 write のステップ（stepIdx 1/4）は従来どおり `toolCall:null` → ハンドラは **null / `name!="write_to_file"` を即 no-op** とガードする必要あり。

### 5.2 hooks — 発火安定性の改善（事実4 改善）

- 同一セッションの **2回目の編集（rvfire2）でも発火**（1.0.8 の「最初の編集のみ」を覆す、実測 n=2）。
- **`agy -p`（print mode）でも発火**（1.0.8 の「-p 非発火」を覆す。print mode でも `toolCall.args.TargetFile` が入る）。
- `command:"python3 dump.py REL"`（相対）が `cwd`=`~/.gemini/config/plugins/hook-test` で実行され、argv=`['dump.py','REL']` を確認 → **自前同梱バイナリを `${extensionPath}` 無しで PWD 相対に呼べる**。

### 5.3 hooks — 未解決のまま（事実2）

`${extensionPath}` は依然 literal/空で未置換（argv に展開値が出ない）。`${/}` の `Bad substitution` は今回の `hooks.json` に含めず**未再検証**だが、PWD 相対呼び出しで回避可能なため実害なし。payload も **agy 独自スキーマ**（`toolCall.args.TargetFile`）で Claude Code 形式（`tool_input.file_path`）ではない＝Claude 流 validator はそのままでは動かず agy アダプタが要る。

### 5.4 rules — 依然全パターン非機能（§4 再確認）＋ changelog 矛盾の解消

3パターンに一意 marker を仕込み、対話セッションで `<user_rules>` を逐語出力させた結果、セクション（`<RULE[user_global]>`）に載るのは **"Gemini Added Memories"（UI 設定）のみ**で marker は未注入＝ §4 の結論不変。

changelog 1.0.4「`rules.json` の allowlist 無視を修正し `.md` rule を load」と本結論の矛盾は **agy バイナリの strings 解析で解消**した: `<user_rules>` は `UserRulesSection.formatMemoriesAsPrompt`（Memories 由来）、`rules.json` は `agents.txt`/`skills.txt` と並ぶ `customizations` サブシステムのディスカバリ・マニフェストで**別系統**。「rule の discover/load ≠ `<user_rules>` 注入」。

### 5.5 帰結と今後

- **「保存した編集ファイルを validator にかける」フック用途は 1.0.9 で初めて成立しうる**（編集ファイル絶対パス＋自前バイナリ PWD 相対呼び出し＋発火安定性が揃った）。
- ただし再導入は **agy 独自 payload スキーマ用のハンドラ実装＝設計変更**であり、本検証のスコープ外。**follow-up Issue として切り出す**。
- rules は引き続き非機能のため、知識は `skills/` で渡す方針を維持。

---

## 6. 追記: agy 1.0.10 での再検証（2026-06-20）

**対象:** agy 1.0.10。`hook-test` を再ビルド・再インストールして検証。
**結論: プロジェクトルール（.agents/AGENTS.md）が初めて正常に動作（注入）するようになった。プラグインルールは依然非機能。hooksは1.0.9同等だが、動的リロードを確認。**

### 6.1 hooks — 動的リロードの確認と変数置換の継続バグ

- **動的リロード**: `agy plugin install hook-test` を実行した直後、**すでに立ち上がっている同一の対話セッション（親セッション）においても、次のツール実行からフックが動的にロードされて発火する**ことが実証された（セッションの再起動が不要）。
- **変数置換バグの継続**: `${extensionPath}` は依然として literal として置換されず（空文字になる）。また、`${/}` をコマンドに含めると `Bad substitution` エラーによりフックプロセス自体が起動前にクラッシュする問題も継続。
  - **対策方針**: 同梱スクリプトやバイナリの呼び出しは、変数は使わず `command: "python3 dump.py"` や `command: "validator/bin/validator --hook"` のように **PWD 相対（プラグインディレクトリ相対）** での記述を維持する。

### 6.2 rules — プロジェクト固有ルール（.agents/AGENTS.md）の正常化

- **Workspace Customizations Rule の動作確認**: 
  `.agents/AGENTS.md` に記述したルールが、新規に立ち上げたセッションのシステムプロンプト `<user_rules>` セクションに、`<RULE[/Users/kwrkb/code/agy-plugins/.agents/AGENTS.md]>` の形式で**正しく自動注入されていること**を実証した。
- **プラグイン固有ルールの非機能（継続）**:
  プラグイン内の `rules/` フォルダの md ファイル、および `plugin.json` 内の `"rules"` 配列定義は、依然としてシステムプロンプトに注入されない。
  - **対策方針**: プラグインからエージェントへルールや知識を渡したい場合は、引き続き `skills/<name>/SKILL.md` に定義してエージェントに読み込ませる方針を維持する。
