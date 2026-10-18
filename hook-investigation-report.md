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
            "command": "python3 /home/user/code/hook-test/dump.py ${extensionPath} ${/}",
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
  "artifactDirectoryPath": "/home/user/.gemini/antigravity-cli/brain/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "conversationId": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
  "error": "",
  "stepIdx": 21,
  "toolCall": null,
  "transcriptPath": "/home/user/.gemini/antigravity-cli/brain/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx/.system_generated/logs/transcript_full.jsonl",
  "workspacePaths": [
    "/home/user/code/agy-plugins"
  ]
}
```
**致命的な問題:**  
Claude Code 等とは異なり、`file_path` や `tool_input` が存在しない。それどころか `toolCall` も `null` となっており、**「どのファイルが編集されたのか」を特定する手段が完全に欠落**している。これにより、対象ファイルを絞って実行するバリデーション用途には利用不可能。

### 事実2: パス変数（`${extensionPath}`, `${/}`）が置換されない
引数のダンプ結果は `['/home/user/code/hook-test/dump.py']` となり、コマンド文字列に含めた `${extensionPath}` が消滅していた。
**致命的な問題:**  
agy 側で変数が展開されずにそのままシェルに渡されている。
- `${extensionPath}` は未定義の環境変数としてシェルに評価され、**空文字列**に消滅した。
- `${/}` を含めると Linux の `/bin/sh` が変数置換エラーとみなし、`sh: 1: Bad substitution` で**フックのプロセス自体が起動前にクラッシュ**する。

### 事実3: 実行時のカレントディレクトリ（PWD）がワークスペースではない
Pythonで取得した `os.getcwd()` の結果は以下の通りだった。
```text
/home/user/.gemini/config/plugins/hook-test
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

**再評価トリガー（agy バージョンアップ時）:** agy を更新したら `agy changelog` を確認し、以下が改善されていれば本調査をやり直してフック／`rules` の再導入を判断する。**（2026-06-17 / agy 1.0.9 ＝§5、2026-06-20 / agy 1.0.10 ＝§6、2026-10-18 / agy 1.4.2 ＝§8・§9）**
- [x] フック payload に `toolCall` / `file_path`（編集ファイルパス）が入るようになったか（§2 事実1）→ **✅ 1.0.9 で解消**（`toolCall.args.TargetFile`）／1.4.2 でも維持
- [ ] `hooks.json` 内の `${extensionPath}` / `${PLUGIN_ROOT}` / `${/}` が実行時に置換されるようになったか（§2 事実2）→ **❌ 1.4.2 でも未置換**（`sh -c` が空文字展開、`${/}` は `Bad substitution`。引き続き PWD 相対パスで回避）
- [x] フックが対話セッションで安定発火するようになったか（§2 事実4）→ **✅ 1.0.9 で改善**（n=2）／**1.0.10 で動的リロード確認**（§6.1）
- [x] `rules/`（プラグイン内 / プロジェクト）がシステムプロンプトに注入されるか（§4）→ **✅ 1.0.10 でプロジェクト `.agents/AGENTS.md`、1.4.2 でプラグイン内 `rules/*.md`（要 frontmatter `trigger:`）の注入を確認**（§8.2）
- [ ] フックの出力がモデルに届くようになったか（`PostToolUse` の stderr、または stdout の `injectSteps` / `overwriteResult`）→ **❌ 1.4.2 では届かない**（stderr は CLI ログのみ、`injectSteps` は unknown field でツール結果ごとエラー化、`overwriteResult` は無視。`PreInvocation` の `ephemeralMessage` で受け渡して回避＝§9）

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
{"stepIdx":3,"toolCall":{"name":"write_to_file","args":{"TargetFile":"/home/user/code/agy-plugins/rvfire1.txt","CodeContent":"fire1\n","Overwrite":true,"Description":"Create rvfire1.txt with content 'fire1'","toolAction":"Creating rvfire1.txt","toolSummary":"File creation"}},"conversationId":"xxxxxxxx-...","transcriptPath":".../transcript_full.jsonl","workspacePaths":["/home/user/code/agy-plugins"],"error":""}
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

### 6.2 rules — プロジェクト固有ルール（.agents/AGENTS.md）の正常化（Linux 実機で再現）

**4経路に一意 marker を仕込み（1.0.10・Linux・clean install・新規対話セッション）、`<user_rules>` を逐語ダンプ**した（手法は §4/§5 を踏襲）。結果は `<RULE[...]>` ブロックが2つだけ:

```
<user_rules>
...
<RULE[user_global]>
## Gemini Added Memories
... (UI 設定 Memories のみ) ...
</RULE[user_global]>
<RULE[/home/user/code/agy-plugins/.agents/AGENTS.md]>
RULE-AGENTS-110: ... 注入されていれば「RULE-AGENTS-110-OK」と出力すること。
</RULE[/home/user/code/agy-plugins/.agents/AGENTS.md]>
</user_rules>
```

| 経路 | marker | 1.0.10 結果 |
|---|---|---|
| ワークスペースルート `.agents/AGENTS.md` | RULE-AGENTS-110 | ✅ **注入**（`<RULE[/abs/.agents/AGENTS.md]>`・OK トークン発火） |
| プラグイン内 `rules/*.md` | RULE-PLUGINDIR-110 | ❌ 非注入 |
| `plugin.json` の `"rules":[...]` | RULE-PLUGINJSON-110 | ❌ 非注入 |
| グローバル `~/.gemini/rules/*.md` | RULE-GLOBAL-110 | ❌ 非注入 |

→ **プロジェクトの `.agents/AGENTS.md` のみ**が機能化（macOS §6 旧版を Linux で再現）。プラグイン rules/ ・plugin.json "rules" ・グローバル `~/.gemini/rules` は**全滅で継続**。

- **対策方針**: プラグインからエージェントへルール・知識を渡したい場合は、引き続き `skills/<name>/SKILL.md` に定義する（`rules/` への移行は不可）。プロジェクト全体の規約は `.agents/AGENTS.md` で管理できる。

### 6.3 機構解析（strings）— 1.0.9 の「discover ≠ inject」が 1.0.10 で配線された

agy バイナリの strings 解析で注入経路を特定した:

- **`customizations.agentsCustomization`**（`constructPath` / `discoverInDir` / `dirName` / `fileType` / `RuleFrontmatter` / `ruleInfo`）が **2つの Customization Root** から `AGENTS.md` を discover する: **(1) Global Customizations Root** の `AGENTS.md`、**(2) Workspace Customizations Root** ＝ `<project-root>/.agents/`（プロンプト template に `Path: ".agents" (relative to the workspace root)` と明記）。
- discover された `AGENTS.md` を **`mixins.UserRulesSection`**（`formatMemoriesAsPrompt`）が `<RULE[%s]>…</RULE[%s]>` 形式で `<user_rules>` に整形する。`<RULE[user_global]>`（Memories）と `<RULE[/abs/.agents/AGENTS.md]>` は同じ `UserRulesSection` 経由。
- **1.0.9 との差**: 1.0.9 の strings 解析では `<user_rules>` は Memories 専用で、`rules.json` 等の customizations discovery は別マニフェスト系統で**プロンプトに到達しなかった**（§5.4「discover ≠ inject」）。1.0.10 は `agentsCustomization` の `AGENTS.md` が初めて `UserRulesSection` に**配線**された。
- **プラグイン rules/ 等が依然非注入の理由**: それらは Customization Root の `AGENTS.md` ではない（プラグイン `rules/*.md` / `plugin.json "rules"` / `~/.gemini/rules/*.md` はいずれも customization root の `AGENTS.md` に該当しない）ため `agentsCustomization` の discover 対象外。
- **changelog 上の記述はゼロ**: 1.0.10 changelog に rules/`.agents` の明示なし（#35「changelog に記述が無くても挙動は変わりうる＝実機再検証」を再々確認）。

---

## 7. 追記: agy 1.0.15 での再検証（2026-07-02）

**対象:** agy 1.0.15
**結論: 1.0.10時点からの進展なし。変数置換バグ、ルール未注入バグともに継続。**

### 7.1 hooks — 変数置換バグの継続
- `${extensionPath}` は依然として空文字に置換（または未置換）されます。
- `${/}` を含めると `Bad substitution` となりフックプロセスが起動前にクラッシュする問題も継続しています。
- 対話セッションでのフックの動的発火は確認できましたが、コマンドのパス解決には依然として **PWD 相対パス**（例: `python3 dump.py`）を使用する必要があります。

### 7.2 rules — プロンプト注入バグの継続
- サブエージェントのシステムプロンプト内の `<user_rules>` セクションをダンプし、ルールの注入状況を調査しました。
- 1.0.10と同様に、**プロジェクトルール（`.agents/AGENTS.md`）のみ**が注入され、プラグイン内のルール（`rules/*.md`）やグローバルルール（`~/.gemini/rules/*.md`）は一切読み込まれない状態が継続しています。

---

## 8. 追記: agy 1.4.2 での再検証（2026-10-18）

**対象:** agy 1.4.2
**結論: プラグイン内 `rules/*.md`（YAML frontmatter `trigger:` 付き）が `<user_rules>` に注入されるよう解消された。一方、`hooks.json` 内の変数置換は `${PLUGIN_ROOT}` を含め依然として未置換のため、PWD 相対パスを維持する。**

### 8.1 hooks — `${PLUGIN_ROOT}` / `${extensionPath}` ともに未置換で継続
- `hooks.json` の `PreInvocation` に (1) `sh ./hook.sh relative`、(2) `sh ${PLUGIN_ROOT}/hook.sh pluginroot`、(3) `sh ${extensionPath}/hook.sh extpath` の3ハンドラを設定して実測。
- (1) の相対パスのみ `pwd=/home/.../.gemini/config/plugins/<name>` で正常実行され、(2) と (3) は `sh -c` が未定義環境変数として空文字展開したため `sh: 0: cannot open /hook.sh: No such file`（exit status 2）で失敗した。
- **対策方針**: `hooks.json` では `${PLUGIN_ROOT}` も使用不可。引き続き PWD 相対パス（例: `validator/bin/validator --hook`）で記述する。

### 8.2 rules — プラグイン内 `rules/*.md` の注入解消（frontmatter `trigger:` 必須）
- プラグイン内に `rules/AGENTS.md`（先頭に YAML frontmatter `trigger: always_on`）を配置し、ワークスペース配置（`.agents/plugins/`）とグローバル install（`agy plugin install`）の両方で検証した結果、エージェントの `<user_rules>` にマーカー（`CANARY-RULE-7731`）が注入されることを確認した。
- なお `agy 1.4.1` のバリデーション仕様により、プラグインの `rules/*.md` には有効な `trigger`（`always_on` / `model_decision` / `glob`）の frontmatter が必須となった（1ファイル 24KB 上限、常時ルール合計 20,000 トークン予算）。
- **設計方針**: 常時ルールはコンテキスト予算を毎ターン消費するため、MCP ツールの引数リファレンスや手順書は引き続きオンデマンドの `skills/<name>/SKILL.md` で配布する。

---

## 9. 追記: agy 1.4.2 でのフック出力経路と timeout の実測（2026-10-18）

**対象:** agy 1.4.2（Linux）。きっかけは、tmux の対話検証で validator フックが起動して C2 を出しているのに、結果がモデルにも TUI にも出なかったこと。使い捨てのプラグインを install し、目印の文字列でモデルの返答と transcript の両方を確かめた。
**結論: `PostToolUse` からはモデルに何も届かない。`PostToolUse` がファイルに書き、`PreInvocation` の `ephemeralMessage` で差し込めば同じターン内に届く。`timeout` は秒単位。**

### 9.1 フックの出力経路

| 経路 | 結果 |
|---|---|
| `PostToolUse` の stderr | CLI ログ（`~/.gemini/antigravity-cli/log/cli-*.log`）に `JSON hook command stderr:` として残るだけ。TUI にもモデルの文脈にも出ない |
| `PostToolUse` の stdout に `{"injectSteps":[...]}` | `failed to unmarshal result from hook ... via protojson: ... unknown field "injectSteps"`。ファイルは書けているのに、ツール結果全体が `Encountered error in tool execution` に置き換わる |
| `PostToolUse` の stdout に `{"overwriteResult":"..."}` | エラーにはならないが無視される（`write_to_file` / `view_file` で確認）。フックへの入力にも `result` は入っていない |
| `PreInvocation` の stdout に `{"injectSteps":[{"ephemeralMessage":"..."}]}` | headless・対話のどちらでもモデルに届く |
| `PostToolUse` → `conversationId` ごとのファイル → `PreInvocation` | 書き込み直後、同じターン内のモデル呼び出しに届く。transcript ではツールのステップ直後に `EPHEMERAL_MESSAGE`、その次がモデルの応答。`PreInvocation` は `PostToolUse` の約 1ms 後に起動した |

- 本体に埋め込まれたフック仕様（strings で確認）でも、`PostToolUse` の出力は「`{}` を期待」とされている。`overwriteResult` は出力の proto には定義されているが、仕様書には無い（未対応の `type: "prompt"` フック向けの定義と見られる）。
- 本キットの validator はこの結果を受けて、`--hook`（`PostToolUse`）と `--inject`（`PreInvocation`）の受け渡し方式に改めた。受け渡しファイルは `os.UserCacheDir()/agy-plugin-kit/relay/<conversationId>.txt`（0700）に置く。

### 9.2 フックの失敗の扱い

- **`PostToolUse`**: 1 つでも失敗（打ち切り・出力の解析エラー）すると、ツールの呼び出し全体がモデルからはエラーに見える（書き込み自体は成功していても）。
- **`PreInvocation`**: 非 0 終了・JSON でない出力・コマンドが見つからない（127）のいずれでも、モデルの呼び出しは止まらず、返答が届いた。CLI ログに `failed to call custom pre-invocation hook` が残るだけ。

### 9.3 timeout の単位

- `timeout: 2` で `sleep 1` するハンドラはエラーなく完了し、`timeout: 1` で `sleep 3` するハンドラは起動から約 1〜2 秒後に `signal: killed` になった。**秒単位**（埋め込み仕様も「seconds, default 30」）。キットの旧 `"timeout": 10000` は約 2.8 時間を意味していた（validator の C11 で検出するようにした）。
- 打ち切られるのは `sh -c` の外側のプロセスだけで、その中から起動した子のスクリプトは残って最後まで走った。

### 9.4 未検証

- Windows（`cmd /c` で実行される）と macOS の実機で、`--hook` / `--inject` が起動するか。
- 1.4.2 より前の agy で `PreInvocation` キーがどう扱われるか。

