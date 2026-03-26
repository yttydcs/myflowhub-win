[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [string]$ConfigPath = (Join-Path $HOME ".codex\config.toml"),
    [string]$ServerName = "myflowhub",
    [string]$ConfigDir = (Join-Path ([Environment]::GetFolderPath("ApplicationData")) "myflowhub\mcp-codex"),
    [string]$DeviceID = "ai-node",
    [string]$DisplayName = "AI MCP",
    [string]$Endpoint = "",
    [switch]$AllowWrite
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if ($ServerName -notmatch '^[A-Za-z0-9_-]+$') {
    throw "ServerName '$ServerName' is invalid. Use letters, digits, '_' or '-'."
}

$scriptRoot = Split-Path -Parent $PSCommandPath
$startScriptPath = Join-Path $scriptRoot "start-myflowhub-mcp.ps1"
if (-not (Test-Path -LiteralPath $startScriptPath -PathType Leaf)) {
    throw "Start script not found: $startScriptPath"
}

function ConvertTo-TomlString {
    param([string]$Value)

    if ($null -eq $Value) {
        $Value = ""
    }

    $escaped = $Value.Replace("\", "\\").Replace('"', '\"')
    return '"' + $escaped + '"'
}

function New-McpServerBlock {
    $argsList = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $startScriptPath,
        "--config-dir", $ConfigDir,
        "--device-id", $DeviceID,
        "--display-name", $DisplayName
    )

    $trimmedEndpoint = $Endpoint.Trim()
    if ($trimmedEndpoint -ne "") {
        $argsList += @("--endpoint", $trimmedEndpoint)
    }
    if ($AllowWrite) {
        $argsList += "--allow-write"
    }

    $argsLines = foreach ($arg in $argsList) {
        "  $(ConvertTo-TomlString $arg)"
    }

    return @(
        "[mcp_servers.$ServerName]"
        'command = "powershell.exe"'
        "args = ["
        ($argsLines -join ",`r`n")
        "]"
    ) -join "`r`n"
}

function Set-McpServerBlock {
    param(
        [string]$Path,
        [string]$Block
    )

    $existing = ""
    if (Test-Path -LiteralPath $Path -PathType Leaf) {
        $existing = [System.IO.File]::ReadAllText($Path)
    }

    $pattern = "(?ms)^\[mcp_servers\." + [regex]::Escape($ServerName) + "\]\r?\n.*?(?=^\[|\z)"
    if ([regex]::IsMatch($existing, $pattern)) {
        return [regex]::Replace($existing, $pattern, $Block + "`r`n")
    }

    if ([string]::IsNullOrWhiteSpace($existing)) {
        return "[mcp_servers]`r`n`r`n$Block`r`n"
    }

    return $existing.TrimEnd() + "`r`n`r`n" + $Block + "`r`n"
}

$block = New-McpServerBlock
$newContent = Set-McpServerBlock -Path $ConfigPath -Block $block

if ($PSCmdlet.ShouldProcess($ConfigPath, "Install MCP server '$ServerName'")) {
    $configDirPath = Split-Path -Parent $ConfigPath
    if (-not [string]::IsNullOrWhiteSpace($configDirPath)) {
        [System.IO.Directory]::CreateDirectory($configDirPath) | Out-Null
    }
    if (Test-Path -LiteralPath $ConfigPath -PathType Leaf) {
        $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
        $backupPath = "$ConfigPath.bak_$timestamp"
        Copy-Item -LiteralPath $ConfigPath -Destination $backupPath -Force
    }
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($ConfigPath, $newContent, $utf8NoBom)
}

Write-Output "ServerName: $ServerName"
Write-Output "ConfigPath: $ConfigPath"
Write-Output "StartScript: $startScriptPath"
Write-Output "ConfigDir: $ConfigDir"
Write-Output ""
Write-Output $block
