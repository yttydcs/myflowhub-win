# Install the MyFlowHub MCP server into a Codex config file.

[CmdletBinding(SupportsShouldProcess = $true)]
param(
    [string]$ConfigPath = (Join-Path $HOME ".codex\config.toml"),
    [string]$ServerName = "myflowhub",
    [ValidateSet("stdio", "http")]
    [string]$Transport = "stdio",
    [string]$Url = "",
    [string]$Listen = "127.0.0.1:17688",
    [string]$McpPath = "/mcp",
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
    # Convert one value to a TOML string literal.
    param([string]$Value)

    if ($null -eq $Value) {
        $Value = ""
    }

    $escaped = $Value.Replace("\", "\\").Replace('"', '\"')
    return '"' + $escaped + '"'
}

function ConvertTo-CommandLineArgument {
    # Quote values for display in a copyable PowerShell command.
    param([string]$Value)

    if ($null -eq $Value) {
        return '""'
    }
    if ($Value -eq "") {
        return '""'
    }
    if ($Value -notmatch '[\s"]') {
        return $Value
    }

    return '"' + $Value.Replace('"', '\"') + '"'
}

function New-EnsureRunningCommand {
    # Build the suggested launcher command for HTTP transport users.
    $command = New-Object System.Collections.Generic.List[string]
    $command.Add("powershell.exe")
    $command.Add("-ExecutionPolicy")
    $command.Add("Bypass")
    $command.Add("-File")
    $command.Add($startScriptPath)
    $command.Add("-EnsureRunning")
    $command.Add("--listen")
    $command.Add($Listen)
    $command.Add("--mcp-path")
    $command.Add((Normalize-McpPath -Path $McpPath))
    $command.Add("--config-dir")
    $command.Add($ConfigDir)
    $command.Add("--device-id")
    $command.Add($DeviceID)
    $command.Add("--display-name")
    $command.Add($DisplayName)

    $trimmedEndpoint = $Endpoint.Trim()
    if ($trimmedEndpoint -ne "") {
        $command.Add("--endpoint")
        $command.Add($trimmedEndpoint)
    }
    if ($AllowWrite) {
        $command.Add("--allow-write")
    }

    return (($command.ToArray() | ForEach-Object { ConvertTo-CommandLineArgument -Value $_ }) -join " ")
}

function New-McpServerBlock {
    # Build a full mcp_servers.<name> block from the current parameters.
    if ($Transport -eq "http") {
        $trimmedUrl = $Url.Trim()
        if ($trimmedUrl -eq "") {
            $path = Normalize-McpPath -Path $McpPath
            $trimmedUrl = "http://$Listen$path"
        }
        return @(
            "[mcp_servers.$ServerName]"
            'type = "http"'
            "url = $(ConvertTo-TomlString $trimmedUrl)"
        ) -join "`r`n"
    }

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

function Normalize-McpPath {
    param([string]$Path)

    $trimmed = $Path.Trim()
    if ($trimmed -eq "") {
        return "/mcp"
    }
    if (-not $trimmed.StartsWith("/")) {
        return "/" + $trimmed
    }
    return $trimmed
}

function Set-McpServerBlock {
    # Replace an existing block first, then append when the block does not exist.
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
Write-Output "Transport: $Transport"
Write-Output "ConfigPath: $ConfigPath"
Write-Output "StartScript: $startScriptPath"
Write-Output "ConfigDir: $ConfigDir"
if ($Transport -eq "http") {
    Write-Output "Url: $($Url.Trim())"
    if ($Url.Trim() -eq "") {
        Write-Output "DerivedUrl: http://$Listen$(Normalize-McpPath -Path $McpPath)"
    }
    Write-Output "EnsureRunning: $(New-EnsureRunningCommand)"
}
Write-Output ""
Write-Output $block
