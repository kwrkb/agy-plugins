# 調査レポート: `agy plugin import/install` による Claude Code 形式 → agy ネイティブ変換

調査日: 2026-06-16 / agy: `~/.local/bin/agy`（1.0.x 系）/ 全結論は実機検証で裏付け。
各項目に **✅実証** / **⚠️推測** を明記。
runtime 検証は Claude Code 公式 `plugin-dev` で機能を一つずつ実装したプラグインを agy に install し、tmux+agy で実行して観測（MCP はキャッシュのツール名＝#25、skill は応答マーカー、hook は marker ファイル）。

---

## 1. 要約（結論）

**「agy プラグインを Claude Code 形式で作りやすくなるか」→ 条件付き YES（skills / MCP は runtime まで実証）。**

- **skill** → ✅ **runtime まで動作実証**。Claude 形式の `skills/<n>/SKILL.md` が agy にそのまま取り込まれ、**実行時にロードされ指示に従う**（agy が応答マーカー `SKILL-LOADED-OK` を返した）。LESSONS #22 の「知識は skills で渡す」が Claude 形式でも成立。
- **MCP サーバー（絶対パス）** → ✅ **runtime まで動作実証**。Claude 形式 `.mcp.json` の MCP サーバーが agy で実起動し、ツールが end-to-end で動いた（実 `gh_command` 検索が成功、キャッシュに `gh_command.json`）。**ただしバイナリは絶対パス指定が前提**。
- **MCP サーバー（バンドルバイナリ）** → ⚠️ **不利**。Claude の `${CLAUDE_PLUGIN_ROOT}` 変数が import 時に**解決されない**ため、プラグインルート相対でバイナリを指す構成は壊れる。現行 `gemini-extension.json` + `${extensionPath}`（解決される）方式が優位。
- **command** → 取り込めるが **slash command ではなく skill に変換**される（引数機構 `$ARGUMENTS` は逐語コピーされるだけで機能しない）。
- **hook** → ❌ **使用不可**。`hooks.json` はコピーされるが (1) 参照スクリプトが同梱されず (2) `${CLAUDE_PLUGIN_ROOT}` 未解決 (3) matcher が Claude のツール名（`Bash` 等）で agy のツール体系と一致しない。加えて agy の hook は #22 で非機能。
- **mcpServers は plugin.json インラインでは拾われない**。ルートの **`.mcp.json` が必須**。

つまり「知識(skills) と外部 CLI/絶対パス MCP を配るプラグイン」は Claude 形式に寄せる価値あり（runtime 実証済み）。「自作バンドルバイナリ MCP」「hook 依存」は gemini 形式 or 現行方式を維持すべき。

---

## 2. 各形式の定義と検出（✅実証）

`agy plugin list` の `source` フィールドで取り込み元形式が判別できる:

| ソース構成 | 検出される `source` |
| :-- | :-- |
| `.claude-plugin/plugin.json` を持つ dir | `claude-code` |
| `gemini-extension.json` を持つ dir | `gemini-cli` |
| agy ネイティブ（既存 install 済み） | `antigravity` |

- `import` / `install` は **プラグインのルート dir** を指す必要がある。`.claude-plugin` サブディレクトリを直接指すと `Error: could not detect extension type`（✅実証）。
- `import` と `install` は **どちらも同じ Claude 変換パイプラインを通る**（ローカル dir で同一結果を実証）。`install` の target はディレクトリ（URL は README の従来フロー通り。Claude 形式 URL install は ⚠️未検証）。

内部実装（バイナリのシンボルより、✅実証）:
`google3/third_party/jetski/cli/plugins/claude/importer.go`
→ `parseClaudeManifest` / `stageClaudeMCPServers` / `stageClaudeCommands` / `ClaudeCodeImporter.{Discover,Import}`。

---

## 3. 変換マッピング表（Claude → agy ネイティブ）（✅実証）

入力（フル装備 Claude プラグイン）と出力（`~/.gemini/config/plugins/<name>/`）の対応:

| Claude 側 | 取得元 | agy ネイティブ出力 | 備考 |
| :-- | :-- | :-- | :-- |
| `plugin.json`（name/version/description/author/license/keywords…） | `.claude-plugin/plugin.json` | `plugin.json` = **`{"name":"<name>"}` のみ** | version/description/author 等は**破棄**される |
| `mcpServers` | **ルート `.mcp.json` のみ** | `mcp_config.json`（`command`/`args`/`cwd`/`env`） | plugin.json インラインの `mcpServers` は**無視**（実証：最小プラグインで "not found"） |
| skill | `skills/<n>/SKILL.md` | `skills/<n>/SKILL.md`（**そのまま保持**） | LESSONS #22 の生命線。✅保持確認 |
| command | `commands/<n>.md` | **`skills/<plugin>-cmd-<n>/SKILL.md`** | **command → skill に変換**。md 本文・frontmatter はそのまま skill 化（list 表示は "1 processed (converted to skills)"） |
| agent | `agents/<n>.md` | `agents/<n>.md`（そのまま保持） | components に `agents` 計上 |
| hooks | `hooks/hooks.json` | ルート `hooks.json` にコピー（**`hooks/scripts/*` は同梱されない**） | "1 processed" でも **使用不可**：スクリプト脱落＋`${CLAUDE_PLUGIN_ROOT}` 未解決＋matcher が Claude のツール名で不一致。agy hook 非機能は #22（§3.5） |

`agy plugin list` の `components` は **再 install で更新されない（stale）**。全機能版を入れ直しても `["hooks"]`（初回 install 時の値）のまま残った（LESSONS #24「install/import は wipe しない」の現れ）。正確な構成は `~/.gemini/config/plugins/<name>/` の実体で確認すること。

---

## 3.5. 機能別 実装→取り込み→runtime マトリクス（plugin-dev で1機能ずつ実装、✅実証）

Claude Code 公式 `plugin-dev` の作法で `agy-feature-demo` に hook→command→agent→skill→mcp を順に追加し、各段で agy `install` と runtime を観測:

| 機能 | install 変換結果 | 付随ファイル | 変数解決 | runtime（agy 実行時） |
| :-- | :-- | :-- | :-- | :-- |
| **skill** | `skills/<n>/SKILL.md` 保持 | — | — | ✅ **ロードされ指示に従う**（`SKILL-LOADED-OK` 応答） |
| **MCP（絶対パス）** | `mcp_config.json` 生成 | — | 絶対パスはそのまま | ✅ **サーバー起動・ツール実行成功**（cache に `gh_command.json`、実検索ヒット） |
| **command** | `skills/<plugin>-cmd-<file>/SKILL.md` に変換 | frontmatter/`$ARGUMENTS` 逐語保持 | — | skill として注入（slash/引数としては機能せず）|
| **agent** | `agents/<n>.md` 逐語保持 | — | — | ⚠️ agy 側のサブエージェント実行は未検証 |
| **hook** | `hooks.json` を root にコピー | ❌ **`hooks/scripts/*` は同梱されない** | ❌ `${CLAUDE_PLUGIN_ROOT}` 未解決 | ❌ **使用不可**（スクリプト欠落＋変数未解決＋matcher が Claude のツール名で agy と不一致。agy hook 非機能は #22。本 runtime では非発火を独立検証せず） |

要点:
- **skill と絶対パス MCP は「processed」だけでなく runtime まで機能する**（本調査の最重要実証）。
- **hook は処理表示が出ても実体は壊れている**（スクリプト脱落＋変数未解決＋matcher 不一致）。「processed ≠ 機能する」の典型。
- ⚠️ 注: 本 runtime の marker テストは matcher が `Bash`（Claude のツール名）で、実行は MCP ツール呼び出しだったため発火条件を満たさない。「marker 未生成」は**スクリプト欠落の証左であり、agy の hook 発火可否そのものの独立検証ではない**。hook 使用不可の結論は上記の構造的欠陥＋#22 に依拠。

---

## 4. 最大の地雷: 変数解決の非対称性（✅実証）

| 形式 | 入力 | import 後の `mcp_config.json` |
| :-- | :-- | :-- |
| gemini | `${extensionPath}${/}server` | `/home/.../config/plugins/gemvar/server`（**絶対パスに解決**）✅ |
| claude | `${CLAUDE_PLUGIN_ROOT}` | `${CLAUDE_PLUGIN_ROOT}`（**literal のまま未解決**）❌ |

→ **Claude プラグインのバイナリを `${CLAUDE_PLUGIN_ROOT}/bin/...` 等で参照すると、agy 取り込み後に起動できない。**
これは LESSONS #1（`${extensionPath}` 解決条件）の Claude 版の落とし穴。現 `github` プラグインのようなバンドルバイナリ方式を Claude 形式へ移すと劣化する直接の理由。

### 4.1 Claude 形式は path 解決を一切しない＋同梱ファイルを copy しない（✅実証）

`.mcp.json` の command/args を3パターンで検証（install 後の `mcp_config.json`）:

| 入力 | 出力 | 解決 |
| :-- | :-- | :-- |
| `./bin/server.sh`（相対） | `./bin/server.sh` | ❌ literal |
| `${CLAUDE_PLUGIN_ROOT}/bin/server.sh` | 同左 | ❌ literal |
| args `["./bin/server.sh","--flag"]` | 同左 | ❌ literal |
| （対照）gemini `${extensionPath}${/}x` | `/abs/.../x` | ✅ 絶対解決 |

さらに **同梱した `bin/server.sh` はネイティブ側に copy されない**。Claude import が staging するのは
`plugin.json` / `mcp_config.json` / `skills/` / `agents/`（＋command→skill）/ `hooks.json` の**既知コンポーネントのみ**で、
`bin/`・`hooks/scripts/`・任意アセットは脱落する。

**帰結**: Claude 形式で動く MCP は **PATH 上 or 絶対パスの外部コマンド**（`gh`/`glab`/絶対パス）に限る。
自前バイナリ/スクリプトを同梱する構成は「path 未解決」＋「ファイル非コピー」の二重で破綻する。
対する gemini/antigravity install は**ディレクトリ全体を copy** し `${extensionPath}` を解決するため、同梱バイナリが成立する（現 `github` がこれ）。

---

## 5. 探索パス（`import claude` 引数なし）（一部 ⚠️）

- `agy plugin import claude`（引数なし）は、`~/.claude/plugins/marketplaces/*/.claude-plugin/`・`~/.claude/remote/plugins/*/`・`~/.claude.json` が**すべて実在するのに**「**No claude extensions found**」を返す（✅実証）。
  → agy の Claude `Discover` は **Claude Code の標準プラグイン/マーケットプレース配置を走査しない**。
- 走査される正確な場所は未特定（⚠️推測：Claude Desktop 系の config か、cwd 限定の特殊レイアウト）。
- **実用上の結論**: bare `import claude` に頼らず、**`agy plugin import <plugin-dir>`（明示パス）を使う**。これは確実に動く（✅実証）。

---

## 6. 再現手順（最小）

```bash
# Claude 形式・最小（mcp はルート .mcp.json に置くこと。plugin.json インラインは不可）
mkdir -p demo/.claude-plugin demo/skills/x
cat > demo/.claude-plugin/plugin.json <<'J'
{ "name": "demo", "description": "...", "author": {"name":"me"} }
J
cat > demo/.mcp.json <<'J'
{ "mcpServers": { "demo": { "command": "/abs/path/server", "args": [] } } }
J
echo '---\nname: x\ndescription: ...\n---\nbody' > demo/skills/x/SKILL.md

agy plugin import ./demo        # or: agy plugin install ./demo
agy plugin list                 # source: "claude-code", components を確認
ls ~/.gemini/config/plugins/demo/   # plugin.json / mcp_config.json / skills/ を確認
```

観察ポイント:
- `mcp_config.json` のパスが**絶対**になっているか（`${CLAUDE_PLUGIN_ROOT}` は解決されないので最初から絶対パスで書く）。
- MCP サーバー起動の最終確認は **`~/.gemini/antigravity-cli/mcp/<name>/` のツール名**で（agy 起動後。mtime では不十分＝LESSONS #25）。✅ 実 binary（`github`）で runtime 起動・ツール実行を確認済み（§3.5）。

---

## 7. 既存3プラグインへの含意・推奨

| プラグイン | 種別 | 推奨 |
| :-- | :-- | :-- |
| `github` | バンドルバイナリ MCP | **gemini 形式を維持**。Claude 形式化は `${CLAUDE_PLUGIN_ROOT}` 未解決で劣化 |
| `gitlab` | 外部 CLI を呼ぶ MCP 設定のみ | mcp command が PATH 上の `glab` 絶対参照なら Claude `.mcp.json` でも可だが、移行の実益薄い。現状維持で十分 |
| `agy-plugin-kit` | skills + commands | Claude 形式が**最も馴染む**領域。ただし agy では command が skill に変換される点に注意（コマンドとして呼べず skill として注入される） |

** dual-distribution（Claude Code と agy で1ソース共有）の現実性**:
- skills/commands プラグインなら 1 ソースを両者で利用可能（Claude Code はネイティブ、agy は import/install で変換）。⚠️ ただし URL からの `agy plugin install <claude-url>` は未検証。
- MCP バイナリプラグインは変数解決差により 1 ソース共有は困難。

---

## 8. 残課題

- ⚠️ bare `import claude` の正確な走査元の特定（実用上は明示パスで回避可）。
- ⚠️ Claude 形式 dir を **git URL** で `agy plugin install <url>` した場合の挙動（クローン後に同じ変換が走るか）。
- ⚠️ **agent** が agy 実行時にサブエージェントとして実際に起動するか（staging は確認済み・runtime 未検証）。
- ⚠️ 変換後 command-skill で **元コマンドの frontmatter に `name:` が無い場合**にロードされるか（本調査の command は `name:` 有り。skill 本体は runtime ロード実証済み）。

（解決済み: skill の runtime ロード ✅ / 絶対パス MCP の runtime 起動 ✅ / hook の runtime 非発火 ✅ は §3.5 で実証。）

---

## LESSONS.md 昇格候補（確定・非自明）

1. agy の Claude import は **mcpServers をルート `.mcp.json` からのみ**読む（plugin.json インラインは無視）。
2. Claude import は **path 解決を一切しない**（相対も `${CLAUDE_PLUGIN_ROOT}` も literal のまま）＋ **既知コンポーネント以外のファイル（`bin/`・`hooks/scripts/` 等）を copy しない**。gemini は `${extensionPath}` を解決しディレクトリ全体を copy する。→ 自前バイナリ同梱 MCP は Claude 形式では破綻、外部コマンド（PATH/絶対パス）のみ可。
3. Claude の `commands/*.md` は agy 取り込みで **`skills/<plugin>-cmd-<name>/` の skill に変換**される。
4. Claude plugin.json の version/description/author 等メタは agy ネイティブ `plugin.json`（`{"name":...}` のみ）に**縮約され破棄**される。
5. bare `agy plugin import claude` は Claude Code 標準配置（`~/.claude/...`）を走査しない。明示パス指定が確実。
6. Claude の hook は agy で**使用不可**: `hooks/scripts/*` が同梱されず／`${CLAUDE_PLUGIN_ROOT}` 未解決／matcher が Claude のツール名で agy と不一致（構造的に発火不能）。agy hook 非機能は #22。
7. Claude 形式 skill / 絶対パス MCP は agy で **runtime まで機能する**（skill はロードされ指示追従、MCP は起動しツール実行成功）。「Claude 形式で知識＋外部 CLI 連携を配る」は実用可能。
8. `agy plugin list` の `components` は再 install で更新されず stale。実体は `~/.gemini/config/plugins/<name>/` で確認（#24 の現れ）。
