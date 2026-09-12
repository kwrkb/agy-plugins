#!/usr/bin/env pwsh
# build.sh の Windows(PowerShell) 版。決定論フラグ・対象は build.sh と完全に揃える。
# 同一 Go バージョン + 同一フラグ + CGO 無効のため、build.sh と bit-identical な
# バイナリを生成する（CI の検証ゲートはどちらでビルドしても通る）。
#
# 使い方:
#   ./build.ps1            # 全モジュール（= all）
#   ./build.ps1 github     # github プラグインのみ
#   ./build.ps1 validator  # agy-plugin-kit の validator のみ
#
# 注意: 決定論ビルドは Go ツールチェーンのパッチ版まで一致が前提。版は .go-version が
#       唯一の定義箇所（build.sh・CI も同じファイルを読む）。
param([string]$Target = 'all')
$ErrorActionPreference = 'Stop'

# 決定論フラグ（build.sh の FLAGS と一致させること）
$Flags = @('-trimpath', '-buildvcs=false', '-ldflags=-buildid=')

# Go ツールチェーン固定版（build.sh の GOTOOLCHAIN と同じ .go-version を読む）。
$GoToolchain = 'go' + (Get-Content (Join-Path $PSScriptRoot '.go-version') -Raw).Trim()

# Build <plugin-dir> <output-basename>
# <plugin-dir>/src/ のソースから、ネイティブバイナリを <plugin-dir>/bin/ に生成する。
#   <base>-linux-amd64   <base>-darwin-arm64   <base>.exe
# 拡張子なしの <base>（OS 分岐 dispatcher）は build.sh / build.ps1 では触らない
# （git 追跡のテキストスクリプト）。Windows の agy は <base>.exe を直接起動する。
function Build([string]$dir, [string]$base) {
    # $env: はプロセス環境を書き換えるため、対話セッションで ./build.ps1 を
    # 実行すると呼び出し元シェルを汚染する。退避し finally で必ず復元する
    # （build.sh はコマンド単位 env + subshell なので汚染しない。それと挙動を揃える）。
    # GOTOOLCHAIN も同じ扱い（build.sh 側は export だが、あちらは子プロセスで完結する）。
    $oldCgo = $env:CGO_ENABLED
    $oldArch = $env:GOARCH
    $oldOs = $env:GOOS
    $oldToolchain = $env:GOTOOLCHAIN
    $pushed = $false
    try {
        # 固定は go version の表示より先に行う。表示された版が「固定が効いた証拠」になる。
        $env:GOTOOLCHAIN = $GoToolchain
        $ver = (go version) -split ' ' | Select-Object -Index 2
        Write-Host "==> building $base (linux-amd64, darwin-arm64, windows) from $dir/src/  [$ver]"
        Push-Location "$dir/src"
        $pushed = $true
        $env:CGO_ENABLED = '0'
        $env:GOARCH = 'amd64'
        $env:GOOS = 'linux'
        go build @Flags -o "../bin/$base-linux-amd64" .
        if ($LASTEXITCODE -ne 0) { throw "go build failed ($dir, linux-amd64)" }
        $env:GOARCH = 'arm64'
        $env:GOOS = 'darwin'
        go build @Flags -o "../bin/$base-darwin-arm64" .
        if ($LASTEXITCODE -ne 0) { throw "go build failed ($dir, darwin-arm64)" }
        $env:GOARCH = 'amd64'
        $env:GOOS = 'windows'
        go build @Flags -o "../bin/$base.exe" .
        if ($LASTEXITCODE -ne 0) { throw "go build failed ($dir, windows)" }
    }
    finally {
        $env:CGO_ENABLED = $oldCgo
        $env:GOARCH = $oldArch
        $env:GOOS = $oldOs
        $env:GOTOOLCHAIN = $oldToolchain
        if ($pushed) { Pop-Location }
    }
}

# スクリプトの位置をリポジトリルートとして扱う
Set-Location $PSScriptRoot

switch ($Target) {
    'github' { Build 'github' 'github' }
    'validator' { Build 'agy-plugin-kit/validator' 'validator' }
    'ast-grep' { Build 'ast-grep' 'ast-grep' }
    'retro-status' { Build 'retro-status' 'retro-status' }
    'settings-advisor' { Build 'settings-advisor' 'settings-advisor' }
    'go-lsp' { Build 'go-lsp' 'go-lsp' }
    'worktree-manager' { Build 'worktree-manager' 'worktree-manager' }
    'all' {
        Build 'github' 'github'
        Build 'agy-plugin-kit/validator' 'validator'
        Build 'ast-grep' 'ast-grep'
        Build 'retro-status' 'retro-status'
        Build 'settings-advisor' 'settings-advisor'
        Build 'go-lsp' 'go-lsp'
        Build 'worktree-manager' 'worktree-manager'
    }
    default {
        Write-Error "unknown target: $Target (expected: github | validator | ast-grep | retro-status | settings-advisor | go-lsp | worktree-manager | all)"
        exit 2
    }
}
