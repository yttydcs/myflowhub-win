[CmdletBinding()]
param(
    [switch]$Help,
    [ValidateSet("login", "register")]
    [string]$AuthMode = "login",
    [string]$Endpoint,
    [string]$ConfigDir,
    [string]$DeviceID,
    [string]$DisplayName = "MCP Smoke",
    [uint32]$NodeID,
    [uint32]$DefaultTarget,
    [int]$TimeoutSeconds = 15,
    [switch]$PreferSource
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Show-Usage {
    @"
Usage:
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode register
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\path\to\mcp-config -NodeID 7

Parameters:
  -Endpoint       Hub endpoint, for example 127.0.0.1:9000.
  -AuthMode       register | login. Default: login.
  -ConfigDir      Dedicated MCP config directory. Login mode requires an existing directory with node keys.
  -DeviceID       Optional device ID. Register mode auto-generates one when omitted.
  -DisplayName    Optional display name forwarded to the MCP process. Default: MCP Smoke.
  -NodeID         Optional node ID for login. Required unless the config directory already stores mcp.node_id.
  -DefaultTarget  Optional default target node ID forwarded to the MCP process.
  -TimeoutSeconds MCP request timeout in seconds. Default: 15.
  -PreferSource   Force the launcher script to use go run instead of a built executable.
  -Help           Show this help text.

Notes:
  - The script drives the MCP server over line-delimited JSON-RPC on stdio.
  - Register mode preserves the config directory so the generated node keys can be reused for login.
  - If register returns pending approval, the script fails explicitly and prints the config directory to keep.
"@ | Write-Host
}

function Write-Step {
    param([string]$Message)
    Write-Host ("[mcp-smoke] {0}" -f $Message)
}

function Escape-PowerShellLiteral {
    param([string]$Value)
    if ($null -eq $Value) {
        return ""
    }
    return $Value.Replace("'", "''")
}

function Get-StderrTail {
    param([System.Collections.ArrayList]$Lines)
    if ($null -eq $Lines -or $Lines.Count -eq 0) {
        return @()
    }
    return @($Lines | Select-Object -Last 12)
}

function Get-HostExecutable {
    $powershell = Get-Command powershell.exe -ErrorAction SilentlyContinue
    if ($null -ne $powershell -and -not [string]::IsNullOrWhiteSpace($powershell.Source)) {
        return $powershell.Source
    }
    $pwsh = Get-Command pwsh.exe -ErrorAction SilentlyContinue
    if ($null -ne $pwsh -and -not [string]::IsNullOrWhiteSpace($pwsh.Source)) {
        return $pwsh.Source
    }
    throw "Cannot find powershell.exe or pwsh.exe to launch the MCP process."
}

function Resolve-ConfigDirectory {
    param(
        [string]$AuthMode,
        [string]$ConfigDir
    )

    if ($AuthMode -eq "login") {
        if ([string]::IsNullOrWhiteSpace($ConfigDir)) {
            throw "Login mode requires -ConfigDir so the MCP client can reuse existing node keys."
        }
        $resolved = [System.IO.Path]::GetFullPath($ConfigDir)
        if (-not (Test-Path -LiteralPath $resolved -PathType Container)) {
            throw "Login mode requires an existing -ConfigDir. Missing: $resolved"
        }
        return $resolved
    }

    if ([string]::IsNullOrWhiteSpace($ConfigDir)) {
        $parent = Join-Path ([System.IO.Path]::GetTempPath()) "myflowhub-mcp-smoke"
        New-Item -ItemType Directory -Force -Path $parent | Out-Null
        $resolved = Join-Path $parent ("register-" + (Get-Date -Format "yyyyMMdd-HHmmss") + "-" + [Guid]::NewGuid().ToString("N").Substring(0, 8))
        New-Item -ItemType Directory -Force -Path $resolved | Out-Null
        return $resolved
    }

    $resolved = [System.IO.Path]::GetFullPath($ConfigDir)
    New-Item -ItemType Directory -Force -Path $resolved | Out-Null
    return $resolved
}

function New-LauncherCommand {
    param(
        [string]$StartScriptPath,
        [string]$Endpoint,
        [string]$ConfigDir,
        [string]$DeviceID,
        [string]$DisplayName,
        [uint32]$DefaultTarget,
        [int]$TimeoutSeconds,
        [bool]$PreferSource
    )

    $parts = New-Object System.Collections.Generic.List[string]
    $parts.Add("& '" + (Escape-PowerShellLiteral $StartScriptPath) + "'")
    if ($PreferSource) {
        $parts.Add("-PreferSource")
    }
    $parts.Add("--endpoint '" + (Escape-PowerShellLiteral $Endpoint) + "'")
    $parts.Add("--config-dir '" + (Escape-PowerShellLiteral $ConfigDir) + "'")
    if (-not [string]::IsNullOrWhiteSpace($DeviceID)) {
        $parts.Add("--device-id '" + (Escape-PowerShellLiteral $DeviceID) + "'")
    }
    if (-not [string]::IsNullOrWhiteSpace($DisplayName)) {
        $parts.Add("--display-name '" + (Escape-PowerShellLiteral $DisplayName) + "'")
    }
    if ($DefaultTarget -gt 0) {
        $parts.Add("--default-target $DefaultTarget")
    }
    $parts.Add("--timeout " + $TimeoutSeconds + "s")
    return ($parts -join " ")
}

function Start-McpProcess {
    param(
        [string]$HostExecutable,
        [string]$LauncherCommand
    )

    $encodedCommand = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($LauncherCommand))

    $startInfo = New-Object System.Diagnostics.ProcessStartInfo
    $startInfo.FileName = $HostExecutable
    $startInfo.Arguments = "-NoProfile -ExecutionPolicy Bypass -EncodedCommand $encodedCommand"
    $startInfo.UseShellExecute = $false
    $startInfo.CreateNoWindow = $true
    $startInfo.RedirectStandardInput = $true
    $startInfo.RedirectStandardOutput = $true
    $startInfo.RedirectStandardError = $true

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $startInfo

    $stderrLines = [System.Collections.ArrayList]::Synchronized((New-Object System.Collections.ArrayList))
    $handler = [System.Diagnostics.DataReceivedEventHandler]{
        param($Sender, $EventArgs)
        if ($null -ne $EventArgs.Data) {
            [void]$stderrLines.Add($EventArgs.Data)
        }
    }
    $process.add_ErrorDataReceived($handler)

    if (-not $process.Start()) {
        throw "Failed to start the MCP launcher process."
    }

    $process.StandardInput.AutoFlush = $true
    $process.BeginErrorReadLine()

    return [pscustomobject]@{
        Process     = $process
        StderrLines = $stderrLines
    }
}

function Stop-McpProcess {
    param([pscustomobject]$State)

    if ($null -eq $State -or $null -eq $State.Process) {
        return
    }

    $process = $State.Process
    try {
        if (-not $process.HasExited) {
            try {
                $process.StandardInput.WriteLine('{"jsonrpc":"2.0","method":"exit"}')
                $process.StandardInput.Close()
            }
            catch {
            }

            if (-not $process.WaitForExit(3000)) {
                $process.Kill()
                $process.WaitForExit()
            }
        }
    }
    finally {
        $process.Dispose()
    }
}

function Wait-ForResponseLine {
    param(
        [System.Diagnostics.Process]$Process,
        [int]$TimeoutMilliseconds,
        [System.Collections.ArrayList]$StderrLines
    )

    $task = $Process.StandardOutput.ReadLineAsync()
    if (-not $task.Wait($TimeoutMilliseconds)) {
        $tail = Get-StderrTail -Lines $StderrLines
        $tailText = if ($tail.Count -gt 0) { " stderr tail: " + ($tail -join " | ") } else { "" }
        throw "Timed out waiting for MCP stdout response.$tailText"
    }

    $line = $task.Result
    if ($null -eq $line) {
        $tail = Get-StderrTail -Lines $StderrLines
        $tailText = if ($tail.Count -gt 0) { " stderr tail: " + ($tail -join " | ") } else { "" }
        throw "MCP process closed stdout before returning a response.$tailText"
    }
    return $line
}

function ConvertTo-CompactJson {
    param([object]$Value)
    if ($null -eq $Value) {
        return ""
    }
    try {
        return ($Value | ConvertTo-Json -Compress -Depth 20)
    }
    catch {
        return [string]$Value
    }
}

function Get-StringValue {
    param([object]$Value)
    if ($null -eq $Value) {
        return ""
    }
    return [string]$Value
}

function Get-UInt32Value {
    param([object]$Value)
    if ($null -eq $Value) {
        return [uint32]0
    }
    $text = [string]$Value
    if ([string]::IsNullOrWhiteSpace($text)) {
        return [uint32]0
    }
    return [uint32]$Value
}

function Invoke-Rpc {
    param(
        [pscustomobject]$State,
        [ref]$NextID,
        [string]$Method,
        [object]$Params,
        [int]$TimeoutMilliseconds,
        [switch]$Notification
    )

    $message = [ordered]@{
        jsonrpc = "2.0"
        method  = $Method
    }
    if ($null -ne $Params) {
        $message.params = $Params
    }
    if (-not $Notification) {
        $NextID.Value = [int]$NextID.Value + 1
        $message.id = $NextID.Value
    }

    $payload = $message | ConvertTo-Json -Compress -Depth 20
    try {
        $State.Process.StandardInput.WriteLine($payload)
    }
    catch {
        $tail = Get-StderrTail -Lines $State.StderrLines
        $tailText = if ($tail.Count -gt 0) { " stderr tail: " + ($tail -join " | ") } else { "" }
        throw "Failed to write JSON-RPC request '$Method' to MCP stdin.$tailText"
    }

    if ($Notification) {
        return $null
    }

    while ($true) {
        $line = Wait-ForResponseLine -Process $State.Process -TimeoutMilliseconds $TimeoutMilliseconds -StderrLines $State.StderrLines
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }

        try {
            $response = $line | ConvertFrom-Json
        }
        catch {
            throw "Failed to parse MCP JSON-RPC response line: $line"
        }

        if ([string]$response.id -ne [string]$message.id) {
            throw "Received unexpected JSON-RPC response id '$($response.id)' while waiting for '$($message.id)'."
        }
        if ($null -ne $response.error) {
            throw "RPC '$Method' failed with code $($response.error.code): $($response.error.message)"
        }
        return $response.result
    }
}

function Invoke-McpTool {
    param(
        [pscustomobject]$State,
        [ref]$NextID,
        [string]$Name,
        [hashtable]$Arguments,
        [int]$TimeoutMilliseconds
    )

    if ($null -eq $Arguments) {
        $Arguments = @{}
    }

    $result = Invoke-Rpc -State $State -NextID $NextID -Method "tools/call" -Params @{
        name      = $Name
        arguments = $Arguments
    } -TimeoutMilliseconds $TimeoutMilliseconds

    if ($null -ne $result.isError -and [bool]$result.isError) {
        $payload = $result.structuredContent
        $code = Get-StringValue $payload.code
        $message = Get-StringValue $payload.message
        $hint = Get-StringValue $payload.hint
        $details = ConvertTo-CompactJson $payload.details
        $errorText = "Tool '$Name' failed"
        if (-not [string]::IsNullOrWhiteSpace($code)) {
            $errorText += " [$code]"
        }
        if (-not [string]::IsNullOrWhiteSpace($message)) {
            $errorText += ": $message"
        }
        if (-not [string]::IsNullOrWhiteSpace($hint)) {
            $errorText += " Hint: $hint"
        }
        if (-not [string]::IsNullOrWhiteSpace($details)) {
            $errorText += " Details: $details"
        }
        throw $errorText
    }

    if ($null -ne $result.structuredContent) {
        return $result.structuredContent
    }
    return $result
}

$scriptRoot = Split-Path -Parent $PSCommandPath
$startScriptPath = Join-Path $scriptRoot "start-myflowhub-mcp.ps1"
$mcpState = $null
$resolvedConfigDir = $null

try {
    if ($Help) {
        Show-Usage
        exit 0
    }
    if ([string]::IsNullOrWhiteSpace($Endpoint)) {
        throw "-Endpoint is required. Run with -Help for usage."
    }
    if ($TimeoutSeconds -le 0) {
        throw "-TimeoutSeconds must be greater than 0."
    }
    if (-not (Test-Path -LiteralPath $startScriptPath -PathType Leaf)) {
        throw "Launcher script not found: $startScriptPath"
    }

    $resolvedConfigDir = Resolve-ConfigDirectory -AuthMode $AuthMode -ConfigDir $ConfigDir
    if ([string]::IsNullOrWhiteSpace($DeviceID) -and $AuthMode -eq "register") {
        $DeviceID = "mcp-smoke-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
    }

    Write-Step "config dir: $resolvedConfigDir"
    if (-not [string]::IsNullOrWhiteSpace($DeviceID)) {
        Write-Step "device id: $DeviceID"
    }

    $hostExecutable = Get-HostExecutable
    $launcherCommand = New-LauncherCommand `
        -StartScriptPath $startScriptPath `
        -Endpoint $Endpoint `
        -ConfigDir $resolvedConfigDir `
        -DeviceID $DeviceID `
        -DisplayName $DisplayName `
        -DefaultTarget $DefaultTarget `
        -TimeoutSeconds $TimeoutSeconds `
        -PreferSource ([bool]$PreferSource)

    Write-Step "starting MCP process"
    $mcpState = Start-McpProcess -HostExecutable $hostExecutable -LauncherCommand $launcherCommand
    $rpcID = 0
    $rpcTimeoutMs = ($TimeoutSeconds + 10) * 1000

    Write-Step "initialize"
    $initialize = Invoke-Rpc -State $mcpState -NextID ([ref]$rpcID) -Method "initialize" -Params @{
        protocolVersion = "2025-11-25"
        capabilities    = @{}
        clientInfo      = @{
            name    = "myflowhub-mcp-smoke"
            version = "1"
        }
    } -TimeoutMilliseconds $rpcTimeoutMs
    if ([string]::IsNullOrWhiteSpace((Get-StringValue $initialize.protocolVersion))) {
        throw "Initialize succeeded but the server did not return protocolVersion."
    }

    Invoke-Rpc -State $mcpState -NextID ([ref]$rpcID) -Method "notifications/initialized" -Params @{} -TimeoutMilliseconds $rpcTimeoutMs -Notification

    Write-Step "discover tools"
    $tools = Invoke-Rpc -State $mcpState -NextID ([ref]$rpcID) -Method "tools/list" -Params @{} -TimeoutMilliseconds $rpcTimeoutMs
    $availableTools = @($tools.tools | ForEach-Object { [string]$_.name })
    $requiredTools = @(
        "myflowhub_session_status",
        "myflowhub_session_connect",
        "myflowhub_auth_register",
        "myflowhub_auth_login",
        "myflowhub_auth_get_perms",
        "myflowhub_auth_list_roles",
        "myflowhub_management_list_nodes"
    )
    $missingTools = @($requiredTools | Where-Object { $_ -notin $availableTools })
    if ($missingTools.Count -gt 0) {
        throw "MCP tools/list is missing required tools: $($missingTools -join ', ')"
    }

    Write-Step "connect session"
    $connectStatus = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_session_connect" -Arguments @{ endpoint = $Endpoint } -TimeoutMilliseconds $rpcTimeoutMs
    if ($null -eq $connectStatus.connected -or -not [bool]$connectStatus.connected) {
        throw "Session connect returned without a connected=true status."
    }

    $preAuthStatus = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_session_status" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs

    $effectiveLoginNodeID = $NodeID
    if ($AuthMode -eq "login" -and $effectiveLoginNodeID -eq 0) {
        $effectiveLoginNodeID = Get-UInt32Value $preAuthStatus.auth.node_id
        if ($effectiveLoginNodeID -eq 0) {
            $effectiveLoginNodeID = Get-UInt32Value $preAuthStatus.defaults.node_id
        }
        if ($effectiveLoginNodeID -eq 0) {
            throw "Login mode requires -NodeID or a stored mcp.node_id in $resolvedConfigDir."
        }
    }

    $authTool = if ($AuthMode -eq "login") { "myflowhub_auth_login" } else { "myflowhub_auth_register" }
    $authArgs = @{}
    if ($AuthMode -eq "login" -and $effectiveLoginNodeID -gt 0) {
        $authArgs.node_id = $effectiveLoginNodeID
    }

    Write-Step "authenticate via $authTool"
    $authPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name $authTool -Arguments $authArgs -TimeoutMilliseconds $rpcTimeoutMs
    if ($null -eq $authPayload.response) {
        throw "Auth tool returned without a response payload."
    }

    $authResponse = $authPayload.response
    $authCode = [int]$authResponse.code
    $authStatus = (Get-StringValue $authResponse.status).ToLowerInvariant()
    if ($AuthMode -eq "register" -and ($authCode -eq 202 -or $authStatus -eq "pending")) {
        $requestID = Get-StringValue $authResponse.request_id
        $reason = Get-StringValue $authResponse.reason
        $suffix = if (-not [string]::IsNullOrWhiteSpace($reason)) { " reason=$reason" } else { "" }
        throw "Register entered approval flow (request_id=$requestID$suffix). Keep config dir '$resolvedConfigDir' and rerun with -AuthMode login after approval."
    }
    if ($authCode -ne 1) {
        $msg = Get-StringValue $authResponse.msg
        $reason = Get-StringValue $authResponse.reason
        throw "Auth $AuthMode failed with code $authCode, status '$authStatus', msg '$msg', reason '$reason'."
    }

    $effectiveNodeID = Get-UInt32Value $authResponse.node_id
    if ($effectiveNodeID -eq 0) {
        $effectiveNodeID = Get-UInt32Value $authPayload.status.auth.node_id
    }
    if ($effectiveNodeID -eq 0 -and $AuthMode -eq "login") {
        $effectiveNodeID = $effectiveLoginNodeID
    }
    if ($effectiveNodeID -eq 0) {
        throw "Auth succeeded but no node_id is available for follow-up permission checks."
    }

    $effectiveHubID = Get-UInt32Value $authResponse.hub_id
    if ($effectiveHubID -eq 0) {
        $effectiveHubID = Get-UInt32Value $authPayload.status.auth.hub_id
    }

    $effectiveRole = Get-StringValue $authResponse.role
    if ([string]::IsNullOrWhiteSpace($effectiveRole)) {
        $effectiveRole = Get-StringValue $authPayload.status.auth.role
    }

    Write-Step "query auth_get_perms"
    $permsPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_get_perms" -Arguments @{
        node_id = $effectiveNodeID
    } -TimeoutMilliseconds $rpcTimeoutMs
    if ($null -eq $permsPayload.response -or [int]$permsPayload.response.code -ne 1) {
        throw "auth_get_perms did not return a success response."
    }

    Write-Step "query auth_list_roles"
    $rolesPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_list_roles" -Arguments @{
        limit    = 20
        node_ids = @($effectiveNodeID)
    } -TimeoutMilliseconds $rpcTimeoutMs
    if ($null -eq $rolesPayload.response -or [int]$rolesPayload.response.code -ne 1) {
        throw "auth_list_roles did not return a success response."
    }

    Write-Step "query management_list_nodes"
    $nodesPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_list_nodes" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs
    if ($null -eq $nodesPayload.response -or [int]$nodesPayload.response.code -ne 1) {
        throw "management_list_nodes did not return a success response."
    }

    $permCount = @($permsPayload.response.perms).Count
    $roleCount = @($rolesPayload.response.roles).Count
    $nodeCount = @($nodesPayload.response.nodes).Count

    Write-Step "smoke completed successfully"
    Write-Host ("ConfigDir: {0}" -f $resolvedConfigDir)
    Write-Host ("NodeID: {0}" -f $effectiveNodeID)
    if ($effectiveHubID -gt 0) {
        Write-Host ("HubID: {0}" -f $effectiveHubID)
    }
    if (-not [string]::IsNullOrWhiteSpace($effectiveRole)) {
        Write-Host ("Role: {0}" -f $effectiveRole)
    }
    Write-Host ("Perms: {0}" -f $permCount)
    Write-Host ("RoleEntries: {0}" -f $roleCount)
    Write-Host ("VisibleNodes: {0}" -f $nodeCount)
    exit 0
}
catch {
    $message = $_.Exception.Message
    if ([string]::IsNullOrWhiteSpace($message)) {
        $message = $_.ToString()
    }
    [Console]::Error.WriteLine($message)
    if (-not [string]::IsNullOrWhiteSpace($resolvedConfigDir)) {
        Write-Host ("ConfigDir: {0}" -f $resolvedConfigDir)
    }
    if ($null -ne $mcpState) {
        $tail = Get-StderrTail -Lines $mcpState.StderrLines
        if ($tail.Count -gt 0) {
            Write-Host "MCP stderr tail:"
            foreach ($line in $tail) {
                Write-Host ("  {0}" -f $line)
            }
        }
    }
    exit 1
}
finally {
    Stop-McpProcess -State $mcpState
}
