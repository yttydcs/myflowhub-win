# 本脚本负责定位或构建 MCP 二进制，并转发本地启动或冒烟测试所需的 CLI 参数。

[CmdletBinding()]
param(
    [switch]$PreferSource,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $PSCommandPath
$repoRoot = Split-Path -Parent $scriptRoot

function Invoke-GoRun {
    # Invoke-GoRun 在找不到可执行文件或显式要求源码模式时，统一走 go run 启动 MCP。
    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if ($null -eq $goCommand) {
        $searched = Get-ExecutableCandidates
        $searchedText = ($searched | ForEach-Object { "'$_'" }) -join ", "
        throw "myflowhub-mcp executable not found and 'go' is unavailable. Checked: $searchedText"
    }

    $runArgs = @("run", "./cmd/myflowhub-mcp") + $ForwardArgs
    $previousGoWork = $env:GOWORK

    try {
        $env:GOWORK = "off"
        Push-Location $repoRoot
        try {
            & $goCommand.Source @runArgs
            return $LASTEXITCODE
        }
        finally {
            Pop-Location
        }
    }
    finally {
        if ($null -eq $previousGoWork) {
            Remove-Item Env:GOWORK -ErrorAction SilentlyContinue
        }
        else {
            $env:GOWORK = $previousGoWork
        }
    }
}

function Get-ExecutableCandidates {
    # Get-ExecutableCandidates 汇总环境变量和仓库常见输出位置，按优先级提供候选路径。
    $candidates = New-Object System.Collections.Generic.List[string]

    if (-not [string]::IsNullOrWhiteSpace($env:MYFLOWHUB_MCP_EXE)) {
        $candidates.Add($env:MYFLOWHUB_MCP_EXE)
    }

    $candidates.Add((Join-Path $repoRoot "build\bin\myflowhub-mcp.exe"))
    $candidates.Add((Join-Path $repoRoot "myflowhub-mcp.exe"))
    $candidates.Add((Join-Path $repoRoot "bin\myflowhub-mcp.exe"))

    return $candidates
}

function Find-Executable {
    # Find-Executable 选择首个真实存在的 MCP 二进制，避免后续重复探测。
    foreach ($candidate in Get-ExecutableCandidates) {
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            return $candidate
        }
    }
    return $null
}

if (-not $PreferSource) {
    $exePath = Find-Executable
    if (-not [string]::IsNullOrWhiteSpace($exePath)) {
        & $exePath @ForwardArgs
        exit $LASTEXITCODE
    }
}

$exitCode = Invoke-GoRun
exit $exitCode
