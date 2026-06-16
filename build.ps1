#!/usr/bin/env pwsh
# build.sh の Windows(PowerShell) 版。決定論フラグ・対象は build.sh と完全に揃える。
# 同一 Go バージョン(1.26.4) + 同一フラグ + CGO 無効のため、build.sh と bit-identical な
# バイナリを生成する（CI の検証ゲートはどちらでビルドしても通る）。
#
# 使い方:
#   ./build.ps1            # 全モジュール（= all）
#   ./build.ps1 github     # github プラグインのみ
#   ./build.ps1 validator  # agy-plugin-kit の validator のみ
#
# 注意: 決定論ビルドは Go ツールチェーンのバージョン一致が前提（現状 go 1.26.4）。
param([string]$Target = 'all')
$ErrorActionPreference = 'Stop'

# 決定論フラグ（build.sh の FLAGS と一致させること）
$Flags = @('-trimpath', '-buildvcs=false', '-ldflags=-buildid=')

# Build <module-dir> <output-basename>
function Build([string]$dir, [string]$base) {
    $ver = (go version) -split ' ' | Select-Object -Index 2
    Write-Host "==> building $base (linux, windows) in $dir/  [$ver]"
    Push-Location $dir
    try {
        $env:CGO_ENABLED = '0'
        $env:GOARCH = 'amd64'
        $env:GOOS = 'linux'
        go build @Flags -o $base .
        if ($LASTEXITCODE -ne 0) { throw "go build failed ($dir, linux)" }
        $env:GOOS = 'windows'
        go build @Flags -o "$base.exe" .
        if ($LASTEXITCODE -ne 0) { throw "go build failed ($dir, windows)" }
    }
    finally { Pop-Location }
}

# スクリプトの位置をリポジトリルートとして扱う
Set-Location $PSScriptRoot

switch ($Target) {
    'github' { Build 'github' 'github' }
    'validator' { Build 'agy-plugin-kit/validator' 'validator' }
    'ast-grep' { Build 'ast-grep' 'ast-grep' }
    'all' {
        Build 'github' 'github'
        Build 'agy-plugin-kit/validator' 'validator'
        Build 'ast-grep' 'ast-grep'
    }
    default {
        Write-Error "unknown target: $Target (expected: github | validator | ast-grep | all)"
        exit 2
    }
}
