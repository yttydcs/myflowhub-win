# Locate a MyFlowHub MCP binary, or run it from source, and forward CLI args.

[CmdletBinding()]
param(
    [switch]$PreferSource,
    [Alias("ensure-running")]
    [switch]$EnsureRunning,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $PSCommandPath
$repoRoot = Split-Path -Parent $scriptRoot
$script:LastLauncherExitCode = 0

function Normalize-McpPath {
    param([string]$Path)

    $trimmed = $Path
    if ($null -eq $trimmed) {
        $trimmed = ""
    }
    $trimmed = $trimmed.Trim()
    if ($trimmed -eq "") {
        return "/mcp"
    }
    if (-not $trimmed.StartsWith("/")) {
        return "/" + $trimmed
    }
    return $trimmed
}

function Get-ForwardArgValue {
    param(
        [string[]]$Items,
        [string[]]$Names
    )

    for ($i = 0; $i -lt $Items.Count; $i++) {
        $arg = $Items[$i]
        foreach ($name in $Names) {
            if ([string]::Equals($arg, $name, [System.StringComparison]::OrdinalIgnoreCase)) {
                if ($i + 1 -ge $Items.Count) {
                    throw "Missing value after '$name'."
                }
                return $Items[$i + 1]
            }
            $prefix = "$name="
            if ($arg.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
                return $arg.Substring($prefix.Length)
            }
        }
    }

    return $null
}

function New-EnsureForwardArgs {
    $argsList = New-Object System.Collections.Generic.List[string]
    foreach ($arg in $ForwardArgs) {
        $argsList.Add($arg)
    }

    $transport = Get-ForwardArgValue -Items $ForwardArgs -Names @("--transport", "-transport")
    if ($null -eq $transport) {
        $argsList.Add("--transport")
        $argsList.Add("http")
        $transport = "http"
    }
    elseif (-not [string]::Equals($transport.Trim(), "http", [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "EnsureRunning requires HTTP transport; got '$transport'."
    }

    $listen = Get-ForwardArgValue -Items $ForwardArgs -Names @("--listen", "-listen")
    if ($null -eq $listen -or $listen.Trim() -eq "") {
        $listen = "127.0.0.1:17688"
        $argsList.Add("--listen")
        $argsList.Add($listen)
    }

    $path = Get-ForwardArgValue -Items $ForwardArgs -Names @("--mcp-path", "-mcp-path")
    if ($null -eq $path -or $path.Trim() -eq "") {
        $path = "/mcp"
        $argsList.Add("--mcp-path")
        $argsList.Add($path)
    }

    return [PSCustomObject]@{
        Args   = [string[]]$argsList.ToArray()
        Listen = $listen.Trim()
        Path   = Normalize-McpPath -Path $path
    }
}

function Get-McpUrl {
    param(
        [string]$Listen,
        [string]$Path
    )

    return "http://$Listen$(Normalize-McpPath -Path $Path)"
}

function Test-McpEndpoint {
    param([string]$Url)

    $request = @{
        jsonrpc = "2.0"
        id = 1
        method = "initialize"
        params = @{
            protocolVersion = "2025-06-18"
            capabilities = @{}
            clientInfo = @{
                name = "myflowhub-mcp-launcher"
                version = "dev"
            }
        }
    }
    $body = ConvertTo-Json $request -Depth 8 -Compress

    try {
        $response = Invoke-WebRequest -UseBasicParsing -Method Post -Uri $Url -ContentType "application/json" -Body $body -TimeoutSec 2
    }
    catch [System.Net.WebException] {
        if ($null -eq $_.Exception.Response) {
            return [PSCustomObject]@{ Status = "unavailable"; Message = $_.Exception.Message }
        }
        return [PSCustomObject]@{ Status = "invalid"; Message = $_.Exception.Message }
    }
    catch {
        return [PSCustomObject]@{ Status = "unavailable"; Message = $_.Exception.Message }
    }

    if ($response.StatusCode -ne 200) {
        return [PSCustomObject]@{ Status = "invalid"; Message = "HTTP status $($response.StatusCode)" }
    }

    try {
        $payload = $response.Content | ConvertFrom-Json
    }
    catch {
        return [PSCustomObject]@{ Status = "invalid"; Message = "response is not JSON" }
    }

    if ($null -ne $payload.result -and $null -ne $payload.result.serverInfo) {
        return [PSCustomObject]@{ Status = "ready"; Message = "MCP endpoint is ready" }
    }

    return [PSCustomObject]@{ Status = "invalid"; Message = "response is not a valid MCP initialize result" }
}

function ConvertTo-ProcessArgument {
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

    $escaped = $Value.Replace('"', '\"')
    if ($escaped.EndsWith("\")) {
        $escaped = $escaped + "\"
    }
    return '"' + $escaped + '"'
}

function Start-BackgroundMcpServer {
    param([string[]]$EffectiveArgs)

    $scriptArgs = New-Object System.Collections.Generic.List[string]
    $scriptArgs.Add("-NoProfile")
    $scriptArgs.Add("-ExecutionPolicy")
    $scriptArgs.Add("Bypass")
    $scriptArgs.Add("-File")
    $scriptArgs.Add($PSCommandPath)
    if ($PreferSource) {
        $scriptArgs.Add("-PreferSource")
    }
    foreach ($arg in $EffectiveArgs) {
        $scriptArgs.Add($arg)
    }

    $argumentText = (($scriptArgs.ToArray() | ForEach-Object { ConvertTo-ProcessArgument -Value $_ }) -join " ")
    return Start-Process -FilePath "powershell.exe" -ArgumentList $argumentText -WorkingDirectory $repoRoot -WindowStyle Hidden -PassThru
}

function Ensure-McpServerRunning {
    $effective = New-EnsureForwardArgs
    $url = Get-McpUrl -Listen $effective.Listen -Path $effective.Path

    $probe = Test-McpEndpoint -Url $url
    if ($probe.Status -eq "ready") {
        Write-Output "MyFlowHub MCP server already running: $url"
        return
    }
    if ($probe.Status -eq "invalid") {
        throw "Endpoint '$url' is reachable but is not a valid MyFlowHub MCP server: $($probe.Message)"
    }

    Write-Output "Starting MyFlowHub MCP server: $url"
    $process = Start-BackgroundMcpServer -EffectiveArgs $effective.Args
    $deadline = (Get-Date).AddSeconds(20)

    while ((Get-Date) -lt $deadline) {
        Start-Sleep -Milliseconds 500

        if ($process.HasExited) {
            throw "MyFlowHub MCP server process exited before becoming ready. ExitCode=$($process.ExitCode)"
        }

        $probe = Test-McpEndpoint -Url $url
        if ($probe.Status -eq "ready") {
            Write-Output "MyFlowHub MCP server ready: $url (pid $($process.Id))"
            return
        }
        if ($probe.Status -eq "invalid") {
            throw "Endpoint '$url' became reachable but is not a valid MyFlowHub MCP server: $($probe.Message)"
        }
    }

    throw "Timed out waiting for MyFlowHub MCP server at '$url'."
}

function Invoke-GoRun {
    # Use go run when no binary is available or source mode is requested.
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
            $script:LastLauncherExitCode = $LASTEXITCODE
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
    # Return binary candidates in priority order.
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
    # Select the first existing MCP binary.
    foreach ($candidate in (Get-ExecutableCandidates)) {
        if ([string]::IsNullOrWhiteSpace($candidate)) {
            continue
        }
        if (Test-Path -LiteralPath $candidate -PathType Leaf) {
            return $candidate
        }
    }
    return $null
}

if ($EnsureRunning) {
    Ensure-McpServerRunning
    exit 0
}

if (-not $PreferSource) {
    $exePath = Find-Executable
    if (-not [string]::IsNullOrWhiteSpace($exePath)) {
        & $exePath @ForwardArgs
        exit $LASTEXITCODE
    }
}

Invoke-GoRun
exit $script:LastLauncherExitCode
