# LESSONS（実装知見ログ）

## 2026-06-19: retro-status 追加・validator フック再導入の Codex レビュー対応（PR #13）

### 学んだこと

#### 37. 同梱バイナリは manifest の単一 `command` パスで OS 別に切り替えられない＝darwin 非対応。README に対応 OS を明記する

`gemini-extension.json` / `hooks.json` の `command` は **1本の文字列パス**（`${extensionPath}${/}retro-status` や相対 `validator/validator`）で、agy は `${extensionPath}` も置換しないため（#20/#34）、**OS ごとに別バイナリ名を選ぶ術がない**。`build.sh` は `GOOS=linux`（拡張子なし＝ELF）と `GOOS=windows`（`.exe`）のみを生成し、拡張子なしファイルが Linux ELF 固定なので **macOS では exec フォーマットエラーで起動失敗**する。Codex が「README が Linux/macOS を謳うが darwin バイナリが無い」と指摘し、実態（Linux 対応・Windows は `.exe`・macOS 非対応）に doc を訂正した。
- **ルール**: 同梱バイナリ参照プラグインの README には**対応 OS を明記**し、darwin バイナリを `build.sh` で生成・コミットしていない限り **macOS 対応を謳わない**。「Linux/Windows」など実際に同梱した GOOS のみ書く。

#### 38. 新規プラグイン追加は `build-verify.yml` の stale ゲート更新とセットにする

retro-status を `build.sh`/`build.ps1` にターゲット追加しただけでは、CI の stale 検出ゲート（`.github/workflows/build-verify.yml`）の `paths` トリガにも per-plugin ジョブにも入らず、**`retro-status/**` の変更が `go vet`/`go test`/`govulncheck`/バイナリ diff チェック無しでマージできてしまう**（Codex 指摘）。`main.go` とコミット済みバイナリの乖離（#21）を検知できなくなる。
- **ルール**: バイナリ同梱プラグインを新設したら、同 PR 内で `build-verify.yml` に **(1) `paths` トリガ（`<plugin>/**`）** と **(2) 既存ジョブを雛形にした per-plugin ジョブ（vet/test/govulncheck/`./build.sh <plugin>` 後の `git diff --exit-code`）** を必ず追加する。`build.sh` のターゲット追加だけで満足しない。

#### 39. bot レビュー（Codex）は全件コードでトレースして真偽を分類する — 実バグ・doc 不一致・誤検知が混在する

PR #13 で Codex の6指摘を全件トレースした結果、**実バグ3 / doc 不一致1 / 誤検知1 / 対応済み1** に分かれた。鵜呑みも全棄却もせず、1件ずつコードと突き合わせて初めて分類できた（#32 と同根）。具体例と再利用ルール:
- **実バグ（採用）**: ①SSH remote `git@host:owner/repo.git` を `/` だけで split すると owner/repo にならず repo 名が壊れる→`:` を `/` に正規化してから末尾2要素を取る（自前の standalone 出力で再現確認できた＝**出力を実際に見る**のが効く）。②`filepath.WalkDir` のパスは Windows で `\` 区切り→`/.github/workflows/` の固定文字列一致が効かない→**`filepath.ToSlash(path)` で正規化**してから比較。③全任意引数の MCP ツールは引数省略時 `request.Params.Arguments == nil` で型アサーションが失敗→**nil を空マップ扱い**にして既定値へフォールバック。
- **doc 不一致（コードを doc に合わせた）**: README が「MP=依存パッケージ数」と明記しているのに実装は Docker/CI フラグ依存だった→`go.mod`/`package.json` 等を数える実装に直した（doc を曲げるより、明記済みの仕様にコードを寄せる）。
- **誤検知（棄却・ただし整理）**: 「`WalkDir` コールバック内の `defer file.Close()` で fd がリーク」は誤り。`defer` は**コールバック関数スコープ**で発火するのでファイルごとに閉じている。ただし再指摘ノイズを避けるため**明示 `file.Close()` に整理**した（機能不変）。

## 2026-06-17: agy 1.0.9 で hooks/rules を再検証（#18/#19/#22 の追跡更新）

agy 1.0.8 → **1.0.9** で `agy changelog` に hooks/rules の変更記述はゼロだが、`reference_agy_rules_nonfunctional` メモリと changelog 1.0.4「`rules.json` の allowlist 無視を修正、`.md` rule を無条件 load」の矛盾を解消するため、`hook-test` を 1.0.9 に clean install して4トリガー＋rules 3パターンを実機再実測した。**hooks は部分的に機能するようになった一方、rules は依然非機能。**「changelog に記述が無くても挙動は変わりうる」＝バージョンアップ時は実機再検証する。

### 学んだこと

#### 34. agy 1.0.9 でフック payload に編集ファイルの絶対パスが入るようになった（#18/#19 の部分的覆し）— ただし agy 独自スキーマ＆`${extensionPath}` は依然未解決

1.0.9 で `write_to_file` の `PostToolUse` 発火時、stdin payload の `toolCall` が **populated** になった: `{"toolCall":{"name":"write_to_file","args":{"TargetFile":"/abs/path","CodeContent":"...","Overwrite":true,...}},"stepIdx":3,...}`。**編集ファイルの絶対パスが `toolCall.args.TargetFile` で取れる**（#18 の「特定不可」が解消）。さらに **(a)** 同一セッションの2回目以降の編集でも発火（#19 の「最初の編集のみ」が改善、実測 n=2）、**(b)** `agy -p`（print mode）でも発火（#19 の「-p 非発火」も解消）、**(c)** `command:"python3 dummy.py REL"` の相対指定が `cwd`=プラグイン install 先で実行され自前同梱バイナリを相対パスで呼べることを確認（argv=`['dump.py','REL']`, PWD=`~/.gemini/config/plugins/<name>`）。

ただし制約は残る:
- **payload は agy 独自スキーマ**（`toolCall.args.TargetFile` / `name:"write_to_file"`）。Claude Code 形式（`tool_input.file_path`）とは別物で、Claude 流の validator は**そのままでは動かず agy アダプタが要る**（#18 の「別形式」は依然真）。
- **`${extensionPath}` は依然 literal/空のまま未置換**（#20 不変、argv に展開後の値が出ない）。同梱バイナリは変数でなく **PWD（=install 先）相対**で呼ぶ。
- PostToolUse は **非ファイルツールのステップでも発火**し、その回は `toolCall:null`（実測 stepIdx 1/4）。ハンドラは **`toolCall` が null / `args.TargetFile` が空なら即 no-op** とガードする。
- **ツール名で絞ってはいけない**: 新規作成は `write_to_file` だが**既存ファイルの編集は `replace_file_content`**（agy plugin-kit の実機検証で判明。`name=="write_to_file"` 限定にすると編集を取りこぼす）。`args.TargetFile` の有無で判定し、最終フィルタは basename で行う。
- `${/}` の `Bad substitution`（#20）は今回の `hooks.json` に `${/}` を含めず**未再検証**。PWD 回避で不要。

帰結: **「保存した編集ファイルを対象に validator を回す」フック用途は 1.0.9 で成立**（編集ファイルパス＋自前バイナリ相対呼び出し＋発火安定性が揃った）。**本リポジトリでは agy-plugin-kit に `hooks.json`＋`validator --hook`（agy payload 対応）として再導入し、実 agy セッションで end-to-end 実証済み（Linux）**。証拠 payload は `hook-investigation-report.md` の 1.0.9 追記に保存。

#### 35. agy 1.0.9 でも `rules/` は全パターン非機能（#22 を再確認）— `rules.json` の discover と `<user_rules>` 注入は別系統

1.0.9 で3パターン（プラグイン内 `rules/*.md` / `plugin.json` の `"rules":[...]` / グローバル `~/.gemini/rules/*.md`）に一意 marker を仕込み、対話セッションでエージェントに `<user_rules>` を逐語出力させたところ、セクション（`<RULE[user_global]>`）に載るのは **"Gemini Added Memories"（UI 設定）のみ**で、3 marker は**いずれも未注入**＝ #22 の結論不変。

changelog 1.0.4「`rules.json` の allowlist 無視を修正し `.md` rule を load」と #22 の矛盾は解消した: agy バイナリの strings で **`<user_rules>` は `UserRulesSection.formatMemoriesAsPrompt`（Memories 由来）**、一方 `rules.json` は `agents.txt`/`skills.txt` と並ぶ **`customizations` サブシステムのディスカバリ・マニフェスト**で別系統。**「rule の discover/load ≠ エージェント `<user_rules>` への注入」**。プラグインの知識は引き続き `skills/<name>/SKILL.md` で渡す。

#### 36. agy hook の検証は payload を手で流す pipe test で終えず、実 agy で「新規作成」と「既存編集」の両方を発火させる

validator フックを 1.0.9 向けに再導入した際、`validator --hook` に payload を手で流す pipe test（`write_to_file` の例）は green だったが、**実 agy セッションで既存マニフェストを編集すると別ツール名 `replace_file_content` が来て取りこぼした**（新規作成のみ `write_to_file`）。pipe test は自分が用意した payload しか試せないため、agy 固有のツール名差を見落とす。
- **ルール**: agy フックハンドラは (1) `hooks.json` の matcher を `.*` にしてツール種別の選別は**コード側**で行う、(2) 編集ファイルは **`toolCall.args.TargetFile` の有無**で判定し**ツール名でガードしない**（新規=`write_to_file`／編集=`replace_file_content`、版で増えうる）、(3) 検証は pipe test に加え**実 agy で新規作成と既存編集の両方を発火**させ、`cwd`=install 先・stdin=実 payload で end-to-end を確認する（#34 の payload 形式・#21 の「実機で機能成立を確認」と同根）。

## 2026-06-16: ast-grep プラグインのレビュー対応・実機検証（PR #10）

### 学んだこと

#### 32. ast-grep の終了コードは grep 互換（マッチなし=1）— bot の「no-match=0」主張は逆、CLI ラッパーの error 判定は exit code 単独で決めない

ast-grep プラグインのレビューで Gemini bot が「最近の ast-grep は**マッチなしでも exit 0**、exit 1 はパースエラー」と critical 指摘し、`if err != nil { return error }` への単純化を勧めた。実機検証（`npm i -g @ast-grep/cli`、ast-grep 0.43.0）すると**真逆**だった: `ast-grep run` は**マッチあり→exit 0 / マッチなし→exit 1（grep 互換）/ 本当のエラー（不正な `--lang` 等）→exit 2 + stderr にエラー文**。つまり bot 提案どおり `err != nil` でエラーにすると、**マッチ0件の正常検索を全てエラー報告するバグ**になる。正しい判定は「**非0終了でも stderr が空なら no-match（無害）、stderr に出力がある時だけ real error**」＋「起動失敗（`*exec.Error`、command-not-found）は常にエラー」。exit code は CLI/版で grep 互換だったり逆だったりするので、ラッパーの error 判定を **exit code 単独に依存させず stderr の有無で見る**のが堅牢（bot 指摘を鵜呑みにせず実機で exit code を取って初めて確定できた典型例）。

補足の実機事実: ①Linux では npm 版が `ast-grep` と `sg` 両方を入れるが、`sg` は `setgroups`(/usr/bin/sg) と衝突するためラッパーは**フルネーム `ast-grep` を hard-code** する。②`--json[=<STYLE>]` は値指定に `=` 必須（`--json .` は `.` を style として食わず positional path のまま）＝ラッパーの `--json` 直後に dir を置く引数構築は安全。③`-r ''`（空 rewrite）は**マッチノードの削除**として正常動作（exit 0, stderr=`Applied N changes`）＝必須引数チェックは「存在＋string 型」だけにし空文字を弾かない。④`fmt.Println($A)` は Go の**型変換式**と解釈されマッチしない（`$$$` 可変長や具体リテラルなら可）＝マッチしない時はパターン解釈を `--debug-query` で疑う（プラグインのバグではない）。

#### 33. 決定論ビルドでも「コミット済みバイナリ ↔ 再ビルド」はセッション中にドリフトしうる — マージ直前に `./build.sh` 再ビルドで揃え直す（#26 の運用面）

PR #10 のマージ直前、ast-grep の stale 検出ゲート（#26）だけが fail した。調査すると**ビルド自体は決定論的**（同一マシンで連続2回ビルドが bit-identical）なのに、**コミット済みバイナリと現在の再ビルド結果の sha が不一致**だった（github/validator ジョブは pass＝クロス環境決定論そのものは健在）。原因は、レビュー対応の過程で**バイナリをコミットした時点と最終状態とで作業環境の状態が変わり**（このセッションでは作業途中に `npm i -g @ast-grep/cli` 等で PATH/ツール状態が変化）、コミット済みバイナリが「過去の環境で焼いた版」のまま取り残されたこと。決定論フラグ（`-trimpath -buildvcs=false -ldflags=-buildid=`）は**同一環境内**の再現性は保証するが、ソース無変更でもコメント修正など複数コミットを跨ぐ間に古いバイナリを commit し続けると、最終ソースから焼き直した版とズレる。教訓: **(a) バイナリ同梱プラグインは「最後にソースを触ったら最後に1回 `./build.sh <target>` して、その出力をコミットする」を徹底**し、途中コミットのバイナリを信用しない。**(b) マージ前にゲートが落ちたら、原因究明より先にまず `./build.sh` 再ビルド→`sha256sum` で再現性（連続2回一致）を確認→コミットすれば直る**（ビルドの非決定性とドリフトはこれで切り分く）。**(c) CI が同一 sha を再現できれば pass する**＝クロス環境決定論は前提どおり機能する（PR #10 で再実証）。

## 2026-06-15: github/gitlab プラグインへのスキル追加

### 学んだこと

#### 29. glab MCP の `flags.repo` はスキーマに存在するツールのみ有効 — 「list 系なら一律あり」と一般化してはいけない

`glab mcp serve` の MCP ツールで `flags.repo` を持つかは**ツール単位**で異なる。`glab_issue_list`・`glab_mr_list` は `repo`（string）を持ち `flags={"repo": "group/project"}` で明示指定できるが、**同じ list 系でも `glab_ci_list` は `repo` を持たない**（CWD 依存）。view 系（`glab_issue_view`、`glab_mr_view`）・write 系（`glab_issue_create`、`glab_mr_create` 等）も `repo` がない。スキーマにないツールへ `flags.repo` を渡しても**サーバー側でフィルタリングされ無視される**（スキーマ外フラグは転送されない）。これらは起動時の CWD の git remote からプロジェクトを検出し、GitLab リモートのないディレクトリでは「Could not determine base repository」エラーになる。**「list / view / write」のような種別でまとめて一般化せず、ツールごとに `tools/list` で実スキーマを確認すること**（種別での一般化が誤りを生んだ実例: Codex レビューで `glab_ci_list` の repo 誤記を指摘された）。

#### 30. glab MCP の `assignee` 型はツールで異なる — `glab_issue_list` は string、`glab_mr_list` は array

`flags.assignee` の型が同じ「list 系」でも**ツールによって違う**。`glab_issue_list.assignee` は `{"type": "string"}`（`"assignee": "@me"`）だが、`glab_mr_list.assignee` は `{"items": {"type": "string"}, "type": "array"}`（`"assignee": ["@me"]`）。issue_list で確認した型を mr_list に流用すると誤る。`@me` 自体は `glab issue list --assignee=@me` として公式サポートされている。**#29 と同根の教訓: ツール種別で型を一般化せず、各ツールのスキーマを個別に確認する**。

#### 31. SKILL.md は生テキストでエージェントに注入される — HTML レンダリング前提のレビュー指摘を鵜呑みにしない

PR #9 で Gemini bot が「Markdown テーブル内の `\|` は GitHub で backslash がそのまま表示されうるので HTML エンティティ `&#124;` を使え」と指摘した。しかし **SKILL.md は HTML レンダリングされず、エージェントへ生テキストとして注入される**成果物であり、`&#124;` にすると逆にエージェントが `repos&#124;issues` というエンティティ文字列をそのまま読んで悪化する。最適解はどちらでもなく、表の他行と同じ `/` 区切りに統一すること（生テキスト可読性・GitHub レンダリング・表内スタイルの全てを満たす）。**ルール: agy の `skills/`・`commands/` 等「LLM が生テキストで読む」成果物では、HTML 表示を前提にした bot 指摘（エンティティ化・装飾）は consumer を取り違えている可能性を疑い、人間向けレンダリングではなく LLM の生テキスト読解を最優先に判断する**。

## 2026-06-15: コミット済みバイナリの stale 検出 CI ゲート（PR #8）

### 学んだこと

#### 26. 「再ビルド → `git diff --exit-code`」でバイナリの stale を検出するには決定論ビルドが前提（BuildID 固定が肝）

`agy plugin install` はビルドせず git 追跡バイナリをコピーするだけなので、`main.go` 変更後の再ビルド忘れで stale バイナリが配布される。これを CI で防ぐ素直な方法は「CI が再ビルド → コミット済みと `git diff --exit-code`」だが、**素の `go build` は BuildID がランダムで毎回バイナリが変わり**、同一ソースでも sha 不一致＝誤検出する（実測: `-trimpath -buildvcs=false` だけでは不十分）。`CGO_ENABLED=0 ... go build -trimpath -buildvcs=false -ldflags=-buildid=` まで付けると **bit-identical** になり、ローカル(WSL2) ↔ CI(ubuntu-latest + setup-go) でも一致した（実走で実証）。決定論は **Go ツールチェーンのパッチ版まで一致が条件**なので、CI は `go-version: '1.26.4'` 固定、開発者も同じ版を使う。ゲート導入時は**既存のコミット済みバイナリも決定論フラグで baseline 再ビルドして揃える**（揃えないと初回から fail）。CI は書き込みせず fail で知らせるだけ（auto-commit はしない選択）。

#### 27. ビルド設定は単一スクリプトに集約し、その**スクリプト自身も** CI の `paths` フィルタに入れる

決定論フラグを README・workflow・CLAUDE.md に直書きすると drift する。`build.sh`/`build.ps1`（引数 `github|validator|all`）に集約し、**CI もこのスクリプトを呼ぶ**ことでフラグの真実が1箇所になる。ただし落とし穴: workflow の `on.paths` フィルタが `build.sh`/`build.ps1` を含まないと、**スクリプトだけ変更した PR でゲートが起動せず**、壊れたビルド設定が素通りする（ゲートがゲート自身の依存を守れない）。集約したスクリプトは必ず `pull_request`/`push` 両方の `paths` に加える。

#### 28. bot レビューの「既定ブランチ」前提は鵜呑みにせず事実確認する

Codex の GitHub bot が「push トリガが `master` だが実際の既定ブランチは `main`」と P2 指摘してきたが、`gh repo view --json defaultBranchRef` は `master`、`git ls-remote --heads` に `main` は無く、過去 PR も `master` にマージ済み——**bot の前提が誤り**だった。鵜呑みに `master`→`main` すると実ブランチでゲートが動かなくなる。bot 指摘（特にリポジトリ状態に依存する主張）はトレース検証してから採否を決め、却下時は根拠を PR コメント・コミットに残す。

## 2026-06-15: github プラグインを gh CLI ラッパーへ移行・実機検証（PR #7）

### 学んだこと

#### 23. MCP ツールに「スペース込みの値」を渡すなら入力は配列にする — 文字列＋空白分割は壊れる

`gh_command` の入力を単一 `command` 文字列にして `strings.Fields(cmdStr)` で空白分割していたが、`pr create --title "My Title"` は `["--title", "\"My", "Title\""]` に砕け、`exec.Command`（シェル非経由）なのでリテラルのクオートも剥がれず `gh` に渡る。`issue list --limit 10` 等の**スペース無し読み取り系はたまたま通る**ためスモークテストをすり抜け、書き込み系（title/body/検索クエリ）で破綻する。**修正は shell-words パーサ追加よりツール入力を `args: string[]` 配列にするのが筋**（依存ゼロ・クオート曖昧性が原理消滅）。mcp-go なら `mcp.WithArray`+`mcp.Items` で定義し `request.RequireStringSlice("args")` で受ける。description に「スペースを含む値は1要素」と明示する。

#### 24. `agy plugin install` はインストール先を事前に wipe しない — 設計変更時は旧ファイルが残る

github-mcp-server ラッパー設計 → gh CLI ラッパー設計へ作り替えて再 install したら、`~/.gemini/config/plugins/github/` に**旧設計の残骸**（`github-mcp-wrapper.sh` / 旧 `plugin.json` / 旧 `mcp_config.json`）が同居していた。install はソースを上書きコピーするが**消えたファイルは消さない**。残った旧 `plugin.json` があると `${extensionPath}` 解決条件（知見 #1）も崩れうる。**プラグインの構成ファイルを増減させた時は、再 install 前にインストール先ディレクトリを `rm -rf` してから入れる**。MCP ツールキャッシュ（`~/.gemini/antigravity-cli/mcp/<name>/`）も旧ツール名が残るので併せて消す。

#### 25. agy 実機での「動いた」証拠は MCP キャッシュの中身（ツール名）まで見る

知見 #12 はキャッシュ mtime 更新を起動成功の証拠としたが、設計を作り替えた今回は mtime だけでは不十分だった。クリーン install（`git archive HEAD`→`agy plugin install`、知見 #21）後に tmux で `agy -p` を流し、`~/.gemini/antigravity-cli/mcp/github/` が**新サーバーの単一ツール `gh_command.json` のみ**に置き換わった（旧 github-mcp-server の 40+ ツールが消えた）ことで「新サーバーが introspect された」と確証できた。さらに**スペース込みクエリ `"mark3labs mcp-go"` を含む読み取り操作**を agy 経由で1回実行し、正しい検索結果が返ることでエンドツーエンド（agy→MCP→gh_command→gh）と知見 #23 の修正効果を同時に実証した。**設計変更時は mtime ではなくキャッシュ内のツール名で別人確認する**。

## 2026-06-15: agy 1.0.8 のプラグイン同梱フックを実機検証（PR #4 をクローズ）

### 学んだこと

#### 18. agy のフック stdin は Claude Code と別形式 — `file_path` が無く編集ファイルを特定できない

agy 1.0.8 で tmux 対話セッションを起こし、`PostToolUse` フックを実発火させて payload をダンプした結果、agy が送る stdin は `{"artifactDirectoryPath": "...", "conversationId": "...", "error": null, "stepIdx": 0, "toolCall": null, "transcriptPath": "...", "workspacePaths": []}` だった。Claude Code 流の `file_path` / `tool_input` が**存在せず** `toolCall` も `null`。よって `file_path` を前提にしたフックハンドラ（`validator --hook`）は**発火しても対象を特定できず常に no-op**。agy 向けフックを書くなら payload は実測してから設計する（`tee` で stdin を採取）。upstream tracking: [agy#395](https://github.com/google-antigravity/antigravity-cli/issues/395)。**【1.0.9 更新】`toolCall` に編集ファイル絶対パスが入るようになった＝本項の「特定不可」は解消（#34）。ただし payload は agy 独自スキーマで Claude 形式ではない点は不変。**

#### 19. agy のフック発火は対話セッション限定かつ不安定

`agy -p`（print mode）ではフックは発火しない。対話セッションでは発火するが、**セッション内の最初の編集（特定 `stepIdx`）でのみ発火し、2 回目以降の `Edit` では発火しないことがある**。「編集ごとに必ず実行」という前提のフック機能は agy 1.0.8 では信頼できない。発火時の `PWD` はプラグインのインストール先（`~/.gemini/config/plugins/<name>/`）なので相対パスで同梱バイナリは呼べる。upstream tracking: [agy#395](https://github.com/google-antigravity/antigravity-cli/issues/395)。**【1.0.9 更新】2回目以降の編集でも発火し、`agy -p` でも発火するよう改善（#34、実測 n=2）。**

#### 20. `hooks.json` 内の `${extensionPath}` / `${/}` は実行時に置換されない（Linux で `Bad substitution`）

`hooks.json` 内の `${extensionPath}` / `${/}` は実行時に一切置換されず literal のまま残り、そのままシェルに渡される。`${extensionPath}` は**未定義の環境変数**として `/bin/sh` に評価され空文字に消滅（argv ダンプで欠落を確認）、`${/}` は `/bin/sh` が不正な変数置換と見なし `sh: 1: Bad substitution` で hook プロセスが起動前にクラッシュする。agy 側にトークン展開は無い。MCP 側（mcp_config.json）と同根で upstream tracking: [agy#390](https://github.com/google-antigravity/antigravity-cli/issues/390)。

#### 21. 「修正」の前に機能が成立するかを実機検証する — クリーン install は git 追跡ファイルのみ

PR #4 は `hooks.json` のパス/パースを直したが、上記 #18 のとおり**そもそも agy 下で自動バリデーションは成立しない**機能の表面修正だった。さらに `validator/validator`（拡張子なし Linux バイナリ）を参照する一方そのバイナリは未コミットで、`git archive HEAD` で再現したクリーン URL install には `validator.exe` しか入らず Linux では解決不能だった（ローカル working tree に未追跡ビルドが在ったため「実機検証OK」に見えていた）。**機能の成立性 → クリーン install での再現性 の順で検証してから直す**。表面修正に入る前に「この機能はそもそも動くのか」を実機で確かめる。

#### 22. agy 1.0.8 の `rules/` 機能は完全に非機能 — プラグインの知識は `skills/` で渡す

実機検証（3 パターン: プラグイン内 `rules/*.md` / `plugin.json` の `"rules":[...]` / グローバル `~/.gemini/rules/*.md`）で、いずれもエージェントのシステムプロンプト（`<user_rules>`）に**一切注入されなかった**（未実装ないし不具合）。現状プロンプトへ載るのは UI 側設定（Gemini Added Memories）のグローバルルールのみ。**プラグインからエージェントへ固有の知識・規約を渡すには `rules/` に頼れず、必ず `skills/<name>/SKILL.md` として定義し呼び出させる**設計にする。`hooks.json`（#18-21）と同様、「ドキュメントに載っている機能 ≠ 実機で動く機能」。upstream tracking: [agy#396](https://github.com/google-antigravity/antigravity-cli/issues/396)。**【1.0.9 更新】3パターンとも依然非機能を再確認（#35）。`rules.json` の discover と `<user_rules>` 注入は別系統。**

## 2026-06-15: github-windows を実装しネイティブ Windows で検証（Issue #1）

### 学んだこと

#### 15. 「検証不能だから follow-up」は環境前提に依存する — 環境が変われば撤回する

判断 4（implementation-notes）で Windows 検証を「WSL2 では不能」と Issue 化したが、後日の作業環境が **Windows ネイティブ**だった。`follow-up 化` の根拠は環境制約であって設計の不確実性ではなかったので、環境が変わった時点で**着手前に前提を再確認**すれば、その場で end-to-end まで完了できた。検証可否は毎回環境を実測して判断する（`go version` が `windows/amd64`、`agy`/`gh` の有無）。

#### 16. Go ラッパーは sh ラッパーの `:-` 意味論を厳密移植する（空文字＝未設定）

`${A:-${B:-...}}` は**空文字列も未設定扱い**でフォールバックする。Go で `os.Getenv(name)` の戻り値が空でないか（`v != ""`）で判定しないと、空の `GITHUB_TOKEN` を「設定済み」と誤判定して `gh auth token` に落ちず、`.sh` と挙動が乖離する。`gh auth token` は `cmd.Output()` でバッファ取り込み（stdout 非継承＝NDJSON を汚さない）、子の exit code は `os.Exit` で伝播。`exec.LookPath("github-mcp-server")` は Windows で `.exe` を自動補完する。

#### 17. Windows でのラッパー検証は「MCP キャッシュ mtime 更新＋オーファン無し」で証拠化

LESSONS #12 の証拠法はそのまま Windows でも有効: env トークン無しで `agy -p` を流し、`~/.gemini/antigravity-cli/mcp/github/*.json` の mtime が更新されれば「トークン解決（`gh auth token`）→ server 起動 → introspect」が通った証拠。加えて Windows は exec-replace が無く agy→wrapper→server の親子構造になるが、stdio server は stdin EOF で終了するため**セッション後にオーファン残留しない**ことを `tasklist | grep github-mcp` で確認した（残る場合のみ Job Object 対応）。

## 2026-06-15: binary-on-PATH 化で github が起動不能になった件と差し戻し

### 学んだこと

#### 11. github-mcp-server は無トークンだと**起動拒否で即終了**する（gitlab との非対称）

URL install をクロスプラットフォーム化する過程で github をラッパー廃止＋「PATH の `github-mcp-server stdio` を直接 `command` に置く」binary-on-PATH 方式へ変更したが、**install は成功するのに MCP が起動しない**事象が発生。原因は `github-mcp-server` が `GITHUB_PERSONAL_ACCESS_TOKEN` 未設定だと `Error: GITHUB_PERSONAL_ACCESS_TOKEN not set` で**即 exit する**こと（知見 #5 の通りトークンは env からのみ読む）。直接 `command` 委譲はトークンを供給しないため起動できない。gitlab(`glab mcp serve`) は自前 config を読むので無トークンでも動く（知見 #7）——この非対称性を見落とすと「動かない」を踏む。

→ **差し戻し**: 薄い POSIX sh ラッパー（知見 #5）で `gh auth token` 等を解決してから PATH の `github-mcp-server` を exec する方式に戻した。ただしバイナリは**同梱せず PATH のものを exec**（URL install で clone されるのは軽量スクリプトのみ＝同梱バイナリ gitignore 問題も同時に解消）。

#### 12. 「動く」の検証は実環境（端末から agy）で、キャッシュ更新を証拠にする

`agy -p "..."`（print モード）でも**セッション起動時に MCP サーバーを introspect する**ことを利用し、`~/.gemini/antigravity-cli/mcp/<server-key>/` の mtime 更新を「起動成功」の客観証拠にできる。無トークンの github は stale のまま、`glab` は更新される、という差で切り分けられた。自分の shell で手動 export して直接バイナリを叩くテストは**実環境と一致しない**（agy の env 継承を経由していない）ので、ラッパー経由・env 未設定での確認が必須。

#### 13. `${extensionPath}` の検証より先に LESSONS を引くべきだった（手戻り）

`${extensionPath}` がネイティブ形式で効くかを probe プラグインで実験したが、答えは知見 #1 に既出だった（`gemini-extension.json` 形式でのみ解決、`plugin.json` があるとコピーのみ）。**過去の落とし穴は着手前に LESSONS.md を確認する**こと。probe の結果も #1 と完全一致した。

#### 14. OS 別分割: Windows は `.cmd`/`.sh` を `command` に直接置けない

ユーザー判断で github を OS 別に分割（`github-unix` / `github-windows`）。Windows は `.sh` 直接 spawn 不可（知見 #10）に加え、`.cmd`/`.bat` も `CreateProcess` が直接実行できず `cmd /c` 経由が要る。実測確認済みの方式は Go ラッパー `.exe`（拡張子なしフルパス、知見 #10）。WSL2 では Windows 実機検証不能のため、未検証コードを同梱せず follow-up 化する判断（「検証不能な提案は Issue 化」）。

## 2026-06-14: github プラグインを Windows でクロスプラットフォーム化

### 学んだこと

#### 10. MCP `command` に `.sh` を直接置くと Windows で起動不可 → Go ラッパーで解決

`agy plugin install` が生成する `mcp_config.json` の `command` は `.sh` の絶対パスになる。Windows では `.sh` を直接 spawn できないため agy が MCP サーバーを起動できずエラーになる。

**agy は Node.js ホストではない**（`agy.exe` は PE32+ コンパイル済みバイナリ）。Node.js を前提にした解決策（`.mjs` ラッパー）は不要な依存を追加する。

**正しい解決**: Go で認証ラッパーをビルドし `gemini-extension.json` の `command` をフルパス（拡張子なし）で指定する。

```json
"command": "${extensionPath}${/}mcpServers${/}github-mcp-wrapper"
```

**実測確認済み**: Windows で `mcpServers/github-mcp-wrapper.exe` を置き、`command` に `.exe` なしフルパスを指定しても agy は正常に spawn できる（MCP ハンドシェイク・ツール呼び出し両方動作）。

**Go ラッパーの注意点**:
- stdout には書かない（MCP は NDJSON を stdout で流すため）
- `os.Executable()` で自身のパスを取得し、同ディレクトリのバイナリを `runtime.GOOS` で `.exe` 付与して起動
- `cmd.Stdin/Stdout/Stderr = os.Stdin/Stdout/Stderr` で stdio を素通し

## 2026-06-14: gitlab プラグインを `glab mcp serve` へ置き換え

### 学んだこと

#### 7. CLI 内蔵 MCP は wrapper 不要（github と非対称）

GitLab 公式 CLI `glab` は **v1.74.0 頃から `glab mcp serve`（stdio, EXPERIMENTAL）** を内蔵する。github-mcp-server は MCP 専用バイナリで `GITHUB_PERSONAL_ACCESS_TOKEN` のみ読む→ wrapper でトークンをブリッジする必要があったが、`glab mcp serve` は glab 自身のサブコマンドなので **glab 既存 config（`~/.config/glab-cli/config.yml`）をそのまま再利用**する。よってトークン env も wrapper も不要。「公式バイナリ置き換え」でも、対象が汎用 CLI 内蔵か MCP 専用バイナリかで認証グルーの要否が変わる。

#### 8. apt 版 glab は古く mcp 非対応 → go install で最新化

Ubuntu universe の `glab` は 1.53.0（apt 候補も同じ）で `mcp` サブコマンド非対応。`go install gitlab.com/gitlab-org/cli/cmd/glab@latest` で最新化し、`~/go/bin`（PATH 上）に入れる。`gemini-extension.json` の `command` は**ベア名 `glab`**（PATH 解決）で良く、同梱バイナリ不要＝build.sh の gitlab セクションも撤去できる。注意: `go install`（ldflags 未注入）だと `glab --version` は `DEV` 表示になる→バージョン判定は version 文字列でなく `glab mcp serve --help` の成否で行う。

#### 9. バージョン導入時期の二分探索は docs raw を使う

`mcp serve` がどの版で入ったかは、`docs/source/mcp/serve.md` を各タグの raw URL（`/-/raw/<tag>/...`）で取得し有無を二分探索して特定できる（v1.70=なし, v1.74=あり）。grep 判定する際、ファイル先頭は YAML frontmatter の `---` なので `head -1` で `title` を探すと誤判定する（`head -5` でマッチ語を見る）。

## 2026-06-14: 公式 github-mcp-server への置き換え

### 学んだこと

#### 5. 公式 MCP サーバーは env トークン名が固定 → wrapper でブリッジ

公式 [github/github-mcp-server](https://github.com/github/github-mcp-server) は `GITHUB_PERSONAL_ACCESS_TOKEN` **のみ**を読む（`GITHUB_TOKEN` も `gh auth token` も見ない）。一方このマシンの認証は `gh` CLI のみ（静的 env トークンなし）。

→ 公式バイナリをそのまま `command` にすると認証で動かない。薄い POSIX sh wrapper (`github-mcp-wrapper.sh`) で `GITHUB_PERSONAL_ACCESS_TOKEN` → `GITHUB_TOKEN` → `GH_TOKEN` → `gh auth token` の順に解決して `export` → `exec` する。これは回避策ではなく統合グルー。`$(dirname "$0")/github-mcp-server stdio` で同梱バイナリを呼ぶ（agy は `command` を絶対パスで渡すので `$0` は絶対パス）。

検証手順: バイナリ受領後に `--help` と `strings | grep GITHUB_` で実際に読む env を実測（README 要約を鵜呑みにしない）、`gh auth status` でユーザーの認証経路を確定してから config を書く。

#### 6. 公式バイナリは `go install pkg@version` で同梱

`GOBIN=.../mcpServers go install github.com/github/github-mcp-server/cmd/github-mcp-server@v1.3.0` でバージョン固定ビルド。出力バイナリ名は cmd ディレクトリ名 = `github-mcp-server`。`go.mod` 不要（モジュールモードの `pkg@version` 形式はローカルモジュール非依存）なので自作の `main.go`/`go.mod`/`go.sum` は削除できる。`build.sh` 冒頭で `mcpServers/` を `rm -rf` してから作り直すと旧バイナリ残骸を一掃できる。

## 2026-06-14: agy MCP プラグイン再構築

### 学んだこと

#### 1. `agy` のプラグイン形式と `${extensionPath}` 置換の仕組み（2026-06-14 更新）

**インストール済みディレクトリの形式（`~/.gemini/config/plugins/<name>/`）:**
- `plugin.json`: 必須マニフェスト（`name` 必須、`description` / `disabled` も可）。`agy plugin validate` が要求する。
- `mcp_config.json`: `mcpServers` を持つ。`agy` が MCP サーバーを起動するときに参照する。
- 上記2ファイルは **`agy plugin install` が生成する**（ソースに置くものではない）。

**`${extensionPath}` 置換の条件（ここが核心）:**
- **ソースに `gemini-extension.json` があり `plugin.json` が無い** → install が `gemini-extension.json` を読んで `${extensionPath}` を解決し、絶対パスの `mcp_config.json` を生成する。
- **ソースに `plugin.json` がある** → install は新形式プラグインと判断し、ソースの `mcp_config.json` をそのままコピーする（`${extensionPath}` は**解決されない**）。

**実用的なルール:**
- `${extensionPath}${/}` を使う必要があるプラグイン（同梱バイナリを参照する github など）は ソースに `gemini-extension.json` を置き `plugin.json` は置かない。
- PATH 上のコマンドを使うプラグイン（gitlab など）はソースに `plugin.json` + `mcp_config.json` を置ける（置換不要なため）。

**置換変数名:** `${extensionPath}` が正しい（`${pluginPath}` ではない）。バイナリの置換トークン集合に
存在するのは `${extensionPath}` / `${workspacePath}` のみ（`convertPluginPath` は Go 内部シンボル、トークンではない）。

#### 2. `agy plugin install` の動作

- インストール元ディレクトリ全体を `~/.gemini/config/plugins/<name>/` にコピーする
- `gemini-extension.json` の `${extensionPath}` を絶対パスに解決して `mcp_config.json` を自動生成する
- バイナリも含めてコピーされるため、**ソースを再ビルドしたら `agy plugin install` の再実行が必要**

#### 3. go-sdk MCP の stdio プロトコルテスト

`mcp.StdioTransport{}` は NDJSON（改行区切り JSON）を使う。テストには stdin を開いたまま双方向通信できる Python subprocess が有効：

```python
proc = subprocess.Popen([binary], stdin=PIPE, stdout=PIPE, stderr=PIPE, text=True, bufsize=1)
proc.stdin.write(json.dumps(msg) + '\n')
proc.stdin.flush()
response = json.loads(proc.stdout.readline())
```

`echo ... | binary` や `binary < file` は stdin が即 EOF になるため MCP サーバーがレスポンスを書く前に終了する。

#### 4. API ラッパー型 MCP サーバーへの git 操作は不適

`git` CLI を subprocess で呼ぶツール（`commit_and_push` 等）を API ラッパー MCP サーバーに入れると、以下の問題が複合する：

1. CWD が不定（MCP サーバーのプロセスは呼び出し元の CWD 依存）
2. `git push` 認証が SSH/credential helper 系でトークンと別系統
3. `git commit` に `user.name`/`user.email` が必要（MCP コンテキストでは未設定）
4. `git add .` が広範すぎる

→ git 操作は agy/claude ホストエージェント側の責務。MCP サーバーは API 呼び出しに徹する。
