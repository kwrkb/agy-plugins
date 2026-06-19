# retro-status — レトロステータス・プロファイラー

リポジトリを解析し、その規模や開発の活発さをRPGのステータス（レベル、職業、HP/MP、攻撃力/防御力、装備、スキル）に見立てて出力する、遊び心にあふれたリポジトリ分析ツールです。

## 提供するMCPツール

### `retro_status`

指定したパスのリポジトリをスキャンし、RPGのレトロステータス画面風のAA（アスキーアート）で出力します。

#### パラメータ

- `path` (string, 任意): スキャンするリポジトリのパス。指定がない場合はカレントディレクトリ。
- `format` (string, 任意): 出力形式。`text` (ファミコン風のAA) または `json`。デフォルトは `text`。

## インストール方法

```bash
agy plugin install https://github.com/kwrkb/agy-plugins/retro-status
```

> **対応 OS（Linux / macOS / Windows）**: ソースは `src/`、配布物は `bin/` に分離。`bin/` に
> `retro-status-linux-amd64` / `retro-status-darwin-arm64` / `retro-status.exe`（ネイティブ）と、拡張子なしの
> OS 分岐 dispatcher `bin/retro-status`（shebang sh・`uname` で実機ネイティブを `exec`）を同梱。`command` は
> `${extensionPath}${/}bin${/}retro-status` で 3 OS を単一指定でカバーする（Windows は agy が `.exe` を直接起動）。

## ステータスマッピングの仕組み

- **レベル (LV)**: 総コミット数とコード行数 (LOC) に基づいて上昇します。
- **ダンジョンの深さ (DEPTH)**: リポジトリの総行数 (LOC) です。行数が増えるほど深くなります（B15F など）。
- **残りモンスター数 (MONSTERS)**: リポジトリ内に残っている `TODO` コメントの数です。
- **攻撃力 (ATK)**: 直近30日間のコミット頻度（開発の「攻め」の勢い）です。
- **防御力 (DEF)**: テストファイルの行数、Linter、CI/CDの設定状況（バグに対する「守り」）です。
- **MP (魔力)**: `go.mod` や `package.json` の依存パッケージの数です。
- **装備 (Equipment)**:
  - 武器: 主要開発言語（Go -> `Goの鋭いメス`、TS -> `TSの魔導書` など）
  - 盾: パッケージロックファイル (`go.sum`, `package-lock.json` など)
  - 鎧: Linter設定やテストコードの有無
  - 兜: CI/CD設定 (`.github/workflows`, `.gitlab-ci.yml`)
  - アクセサリー: コンテナ設定 (`Dockerfile`)
- **スキル/呪文**: テストがあると `ベホマ (自動テスト)`、CI/CDがあると `ルーラ (自動デプロイ)` などを習得します。
- **持ち物 (INVENTORY)**: サーバーを動かしているマシンの `PATH` 上にあるモダン開発 CLI（`rg`/`fd`/`jq`/`gh`/`docker` など）を検出し、`鷹の目 (rg)` のような秘宝として並べます。リポジトリではなく実行環境を見るため、同じリポでもマシンが変われば中身が変わります。
