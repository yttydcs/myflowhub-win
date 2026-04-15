# 本脚本负责通过 stdio 驱动分阶段 JSON-RPC 冒烟测试，用来验证 Win MCP 客户端。

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
    [switch]$PreferSource,
    [switch]$EnableExtendedRead,
    [switch]$EnableAuthoritySmoke,
    [switch]$EnableWriteSmoke,
    [string]$ConfigKey = "authority.node_id",
    [string]$NodeEchoMessage,
    [uint32]$AuthorityID,
    [string]$PermitDeviceID,
    [string]$PermitRole,
    [string]$PendingRequestID,
    [ValidateSet("none", "approve", "reject")]
    [string]$PendingAction = "none",
    [string]$ApprovalRole,
    [string]$RejectReason = "mcp smoke reject",
    [uint32]$ExecutorNode,
    [string]$FlowID,
    [string]$FlowName,
    [string]$FlowMethod,
    [string]$VarName,
    [string]$VarValue = "mcp smoke value",
    [uint32]$VarOwner
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Show-Usage {
    # Show-Usage 汇总 staged smoke 的入口参数和各阶段语义，便于单次命令自解释。
    @"
Usage:
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode register
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\path\to\mcp-config -NodeID 7
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\path\to\mcp-config -EnableExtendedRead
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\path\to\mcp-config -EnableAuthoritySmoke -PendingAction approve -PendingRequestID req-1
  powershell -ExecutionPolicy Bypass -File .\scripts\test-myflowhub-mcp-smoke.ps1 -Endpoint 127.0.0.1:9000 -AuthMode login -ConfigDir C:\path\to\mcp-config -EnableWriteSmoke -ExecutorNode 42 -FlowMethod demo::run

Parameters:
  -Endpoint              Hub endpoint, for example 127.0.0.1:9000.
  -AuthMode              register | login. Default: login.
  -ConfigDir             Dedicated MCP config directory. Login mode requires an existing directory with node keys.
  -DeviceID              Optional device ID. Register mode auto-generates one when omitted.
  -DisplayName           Optional display name forwarded to the MCP process. Default: MCP Smoke.
  -NodeID                Optional node ID for login. Required unless the config directory already stores mcp.node_id.
  -DefaultTarget         Optional default target node ID forwarded to the MCP process.
  -TimeoutSeconds        MCP request timeout in seconds. Default: 15.
  -PreferSource          Force the launcher script to use go run instead of a built executable.
  -EnableExtendedRead    Run read-only management, config, exec, and flow-read checks after the base chain.
  -EnableAuthoritySmoke  Run authority approval-flow checks. Always lists pending requests first.
  -EnableWriteSmoke      Run flow/varstore write checks with temporary resources. This automatically starts MCP with --allow-write.
  -ConfigKey             Preferred management config key for config_get. Default: authority.node_id.
  -NodeEchoMessage       Message for management_node_echo. Auto-generated when omitted.
  -AuthorityID           Optional authority node ID for authority smoke actions.
  -PermitDeviceID        Device ID for issue_register_permit.
  -PermitRole            Role for issue_register_permit. Requires -PermitDeviceID.
  -PendingRequestID      Request ID for approve/reject actions.
  -PendingAction         none | approve | reject. Default: none.
  -ApprovalRole          Optional role for approve_register.
  -RejectReason          Optional reason for reject_register. Default: mcp smoke reject.
  -ExecutorNode          Flow executor node for read/write flow checks. Required for write smoke.
  -FlowID                Flow ID for flow_get/status, or for the temporary write-smoke flow when write smoke is enabled.
  -FlowName              Optional name for the temporary write-smoke flow.
  -FlowMethod            Required flow call method for write smoke.
  -VarName               Variable name for write smoke. Auto-generated when omitted.
  -VarValue              Variable value for write smoke. Default: mcp smoke value.
  -VarOwner              Variable owner for write smoke. Defaults to the authenticated node.
  -Help                  Show this help text.

Stages:
  - Base: initialize -> tools/list -> connect -> register/login -> auth_get_perms -> auth_list_roles -> management_list_nodes
  - Extended read: management_node_info -> management_node_echo -> management_list_subtree -> management_config_list/get -> exec_cap_query -> flow_list/get/status
  - Authority: auth_list_pending_registers and optional permit / approve / reject actions
  - Write: varstore list/set/get/revoke plus flow set/list/get/run/status/delete with explicit cleanup

Notes:
  - The script drives the MCP server over line-delimited JSON-RPC on stdio.
  - Register mode preserves the config directory so the generated node keys can be reused for login.
  - If register returns pending approval, the script fails explicitly and prints the config directory to keep.
  - Authority and write stages mutate real Hub state. They never run unless explicitly enabled.
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
    # Resolve-ConfigDirectory 根据 auth 模式决定是复用现有目录还是创建临时 smoke 目录。
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
    # New-LauncherCommand 把 smoke 参数收敛成一条可复现的 MCP 启动命令。
    param(
        [string]$StartScriptPath,
        [string]$Endpoint,
        [string]$ConfigDir,
        [string]$DeviceID,
        [string]$DisplayName,
        [uint32]$DefaultTarget,
        [int]$TimeoutSeconds,
        [bool]$PreferSource,
        [bool]$AllowWrite
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
    if ($AllowWrite) {
        $parts.Add("--allow-write")
    }
    $parts.Add("--timeout " + $TimeoutSeconds + "s")
    return ($parts -join " ")
}

function Start-McpProcess {
    # Start-McpProcess 以后台 stdio 进程拉起 MCP，并同步收集 stderr 方便失败诊断。
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
    # Wait-ForResponseLine 在限定时间内读取一行 stdout，并在超时时带上 stderr 尾部证据。
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
    $text = Get-StringValue $Value
    if ([string]::IsNullOrWhiteSpace($text)) {
        return [uint32]0
    }
    return [uint32]$Value
}

function Get-IntValue {
    param([object]$Value)
    $text = Get-StringValue $Value
    if ([string]::IsNullOrWhiteSpace($text)) {
        return 0
    }
    return [int]$Value
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

function Assert-ResponseSuccess {
    param(
        [string]$ToolName,
        [object]$Payload
    )

    if ($null -eq $Payload) {
        throw "$ToolName returned an empty payload."
    }
    if ($null -eq $Payload.response) {
        throw "$ToolName returned without a response payload."
    }

    $response = $Payload.response
    $code = Get-IntValue $response.code
    if ($code -ne 1) {
        $msg = Get-StringValue $response.msg
        throw "$ToolName returned code $code msg '$msg'."
    }
    return $response
}

function Assert-ToolsAvailable {
    param(
        [string[]]$AvailableTools,
        [string[]]$RequiredTools
    )

    $missingTools = @($RequiredTools | Where-Object { $_ -notin $AvailableTools })
    if ($missingTools.Count -gt 0) {
        throw "MCP tools/list is missing required tools: $($missingTools -join ', ')"
    }
}

function Get-RequiredToolNames {
    param(
        [bool]$ExtendedRead,
        [bool]$AuthoritySmoke,
        [bool]$WriteSmoke
    )

    $names = New-Object System.Collections.Generic.List[string]
    foreach ($name in @(
        "myflowhub_session_status",
        "myflowhub_session_connect",
        "myflowhub_auth_register",
        "myflowhub_auth_login",
        "myflowhub_auth_get_perms",
        "myflowhub_auth_list_roles",
        "myflowhub_management_list_nodes"
    )) {
        $names.Add($name)
    }

    if ($ExtendedRead) {
        foreach ($name in @(
            "myflowhub_management_node_info",
            "myflowhub_management_node_echo",
            "myflowhub_management_list_subtree",
            "myflowhub_management_config_get",
            "myflowhub_management_config_list",
            "myflowhub_exec_cap_query",
            "myflowhub_flow_list",
            "myflowhub_flow_get",
            "myflowhub_flow_status"
        )) {
            $names.Add($name)
        }
    }

    if ($AuthoritySmoke) {
        foreach ($name in @(
            "myflowhub_auth_list_pending_registers",
            "myflowhub_auth_approve_register",
            "myflowhub_auth_reject_register",
            "myflowhub_auth_issue_register_permit",
            "myflowhub_auth_revoke_register_permit"
        )) {
            $names.Add($name)
        }
    }

    if ($WriteSmoke) {
        foreach ($name in @(
            "myflowhub_flow_list",
            "myflowhub_flow_get",
            "myflowhub_flow_set",
            "myflowhub_flow_run",
            "myflowhub_flow_status",
            "myflowhub_flow_delete",
            "myflowhub_varstore_list",
            "myflowhub_varstore_get",
            "myflowhub_varstore_set",
            "myflowhub_varstore_revoke"
        )) {
            $names.Add($name)
        }
    }

    return @($names | Select-Object -Unique)
}

function New-StageSummary {
    return [ordered]@{
        base          = "pending"
        extended_read = if ($EnableExtendedRead) { "pending" } else { "skipped" }
        authority     = if ($EnableAuthoritySmoke) { "pending" } else { "skipped" }
        write         = if ($EnableWriteSmoke) { "pending" } else { "skipped" }
    }
}

function Write-StageSummary {
    param([hashtable]$Stages)
    Write-Host ("Stages: base={0}; extended_read={1}; authority={2}; write={3}" -f $Stages.base, $Stages.extended_read, $Stages.authority, $Stages.write)
}

function New-TempResourceName {
    param([string]$Prefix)
    return "{0}-{1}-{2}" -f $Prefix, (Get-Date -Format "yyyyMMddHHmmss"), [Guid]::NewGuid().ToString("N").Substring(0, 8)
}

function Select-ConfigGetKey {
    param(
        [object[]]$Keys,
        [string]$PreferredKey,
        [bool]$PreferredKeyExplicit
    )

    $normalized = @()
    foreach ($item in @($Keys)) {
        $value = Get-StringValue $item
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            $normalized += $value
        }
    }

    $preferred = Get-StringValue $PreferredKey
    if (-not [string]::IsNullOrWhiteSpace($preferred)) {
        if ($PreferredKeyExplicit -or $preferred -in $normalized) {
            return $preferred
        }
    }

    if ($normalized.Count -gt 0) {
        return $normalized[0]
    }
    return ""
}

function Get-FirstFlowID {
    param([object[]]$Flows)
    foreach ($flow in @($Flows)) {
        $flowID = Get-StringValue $flow.flow_id
        if (-not [string]::IsNullOrWhiteSpace($flowID)) {
            return $flowID
        }
    }
    return ""
}

function New-TempFlowGraph {
    param([string]$Method)
    return @{
        nodes = @(
            @{
                id   = "call1"
                kind = "call"
                spec = @{
                    method        = $Method
                    args_template = @{}
                }
            }
        )
        edges = @()
    }
}

function Add-OptionalStringArg {
    param(
        [hashtable]$Arguments,
        [string]$Name,
        [string]$Value
    )

    $text = Get-StringValue $Value
    if (-not [string]::IsNullOrWhiteSpace($text)) {
        $Arguments[$Name] = $text
    }
}

function Add-OptionalUInt32Arg {
    param(
        [hashtable]$Arguments,
        [string]$Name,
        [uint32]$Value
    )

    if ($Value -gt 0) {
        $Arguments[$Name] = $Value
    }
}

$scriptRoot = Split-Path -Parent $PSCommandPath
$startScriptPath = Join-Path $scriptRoot "start-myflowhub-mcp.ps1"
$mcpState = $null
$resolvedConfigDir = $null
$stages = New-StageSummary
$cleanupIssues = New-Object "System.Collections.Generic.List[string]"
$selectedConfigKey = ""
$selectedFlowID = ""
$writeFlowID = ""
$writeVarName = ""

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

    $preferredConfigKeyExplicit = $PSBoundParameters.ContainsKey("ConfigKey")
    $pendingRequestIDText = (Get-StringValue $PendingRequestID).Trim()
    $permitDeviceIDText = (Get-StringValue $PermitDeviceID).Trim()
    $permitRoleText = (Get-StringValue $PermitRole).Trim()
    $flowMethodText = (Get-StringValue $FlowMethod).Trim()

    if ([string]::IsNullOrWhiteSpace($NodeEchoMessage)) {
        $NodeEchoMessage = "mcp-smoke-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
    }

    if ($EnableAuthoritySmoke) {
        if (($permitDeviceIDText -eq "") -xor ($permitRoleText -eq "")) {
            $stages.authority = "failed"
            throw "Permit smoke requires both -PermitDeviceID and -PermitRole."
        }
        if ($PendingAction -ne "none" -and [string]::IsNullOrWhiteSpace($pendingRequestIDText)) {
            $stages.authority = "failed"
            throw "Pending approval smoke requires -PendingRequestID when -PendingAction is approve or reject."
        }
    }

    if ($EnableWriteSmoke) {
        if ($ExecutorNode -eq 0) {
            $stages.write = "failed"
            throw "Write smoke requires -ExecutorNode so flow operations use an explicit executor."
        }
        if ([string]::IsNullOrWhiteSpace($flowMethodText)) {
            $stages.write = "failed"
            throw "Write smoke requires -FlowMethod so the temporary flow uses an explicit call method."
        }
    }

    try {
        $resolvedConfigDir = Resolve-ConfigDirectory -AuthMode $AuthMode -ConfigDir $ConfigDir
    }
    catch {
        $stages.base = "failed"
        throw
    }
    if ([string]::IsNullOrWhiteSpace($DeviceID) -and $AuthMode -eq "register") {
        $DeviceID = "mcp-smoke-" + [Guid]::NewGuid().ToString("N").Substring(0, 8)
    }

    Write-Step "config dir: $resolvedConfigDir"
    if (-not [string]::IsNullOrWhiteSpace($DeviceID)) {
        Write-Step "device id: $DeviceID"
    }
    if ($EnableWriteSmoke) {
        Write-Step "write smoke enabled: launcher will add --allow-write"
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
        -PreferSource ([bool]$PreferSource) `
        -AllowWrite ([bool]$EnableWriteSmoke)

    Write-Step "starting MCP process"
    $mcpState = Start-McpProcess -HostExecutable $hostExecutable -LauncherCommand $launcherCommand
    $rpcID = 0
    $rpcTimeoutMs = ($TimeoutSeconds + 10) * 1000

    $effectiveNodeID = [uint32]0
    $effectiveHubID = [uint32]0
    $effectiveRole = ""
    $permCount = 0
    $roleCount = 0
    $nodeCount = 0

    try {
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
        $requiredTools = Get-RequiredToolNames -ExtendedRead ([bool]$EnableExtendedRead) -AuthoritySmoke ([bool]$EnableAuthoritySmoke) -WriteSmoke ([bool]$EnableWriteSmoke)
        Assert-ToolsAvailable -AvailableTools $availableTools -RequiredTools $requiredTools

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
        $authCode = Get-IntValue $authResponse.code
        $authStatus = (Get-StringValue $authResponse.status).ToLowerInvariant()
        if ($AuthMode -eq "register" -and ($authCode -eq 202 -or $authStatus -eq "pending")) {
            $requestID = Get-StringValue $authResponse.request_id
            $reason = Get-StringValue $authResponse.reason
            $suffix = if (-not [string]::IsNullOrWhiteSpace($reason)) { " reason=$reason" } else { "" }
            throw "Register entered approval flow (request_id=$requestID$suffix). Keep config dir '$resolvedConfigDir' and rerun with -AuthMode login after approval."
        }
        if ($AuthMode -eq "register" -and $authStatus -eq "rejected") {
            $reason = Get-StringValue $authResponse.reason
            $suffix = if (-not [string]::IsNullOrWhiteSpace($reason)) { " reason=$reason" } else { "" }
            throw "Register was rejected$suffix. Keep config dir '$resolvedConfigDir' for inspection."
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
        $permsResponse = Assert-ResponseSuccess -ToolName "myflowhub_auth_get_perms" -Payload $permsPayload

        Write-Step "query auth_list_roles"
        $rolesPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_list_roles" -Arguments @{
            limit    = 20
            node_ids = @($effectiveNodeID)
        } -TimeoutMilliseconds $rpcTimeoutMs
        $rolesResponse = Assert-ResponseSuccess -ToolName "myflowhub_auth_list_roles" -Payload $rolesPayload

        Write-Step "query management_list_nodes"
        $nodesPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_list_nodes" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs
        $nodesResponse = Assert-ResponseSuccess -ToolName "myflowhub_management_list_nodes" -Payload $nodesPayload

        $permCount = @($permsResponse.perms).Count
        $roleCount = @($rolesResponse.roles).Count
        $nodeCount = @($nodesResponse.nodes).Count
        $stages.base = "passed"
    }
    catch {
        $stages.base = "failed"
        throw
    }

    if ($EnableExtendedRead) {
        try {
            Write-Step "stage: extended read"

            Write-Step "query management_node_info"
            $nodeInfoPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_node_info" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs
            [void](Assert-ResponseSuccess -ToolName "myflowhub_management_node_info" -Payload $nodeInfoPayload)

            Write-Step "query management_node_echo"
            $nodeEchoPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_node_echo" -Arguments @{
                message = $NodeEchoMessage
            } -TimeoutMilliseconds $rpcTimeoutMs
            [void](Assert-ResponseSuccess -ToolName "myflowhub_management_node_echo" -Payload $nodeEchoPayload)

            Write-Step "query management_list_subtree"
            $subtreePayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_list_subtree" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs
            [void](Assert-ResponseSuccess -ToolName "myflowhub_management_list_subtree" -Payload $subtreePayload)

            Write-Step "query management_config_list"
            $configListPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_config_list" -Arguments @{} -TimeoutMilliseconds $rpcTimeoutMs
            $configListResponse = Assert-ResponseSuccess -ToolName "myflowhub_management_config_list" -Payload $configListPayload

            $selectedConfigKey = Select-ConfigGetKey -Keys @($configListResponse.keys) -PreferredKey $ConfigKey -PreferredKeyExplicit $preferredConfigKeyExplicit
            if (-not [string]::IsNullOrWhiteSpace($selectedConfigKey)) {
                Write-Step "query management_config_get ($selectedConfigKey)"
                $configGetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_management_config_get" -Arguments @{
                    key = $selectedConfigKey
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_management_config_get" -Payload $configGetPayload)
            }
            else {
                Write-Step "management_config_get skipped: no readable config key available"
            }

            $execArgs = @{
                limit          = 20
                include_schema = $true
            }
            Add-OptionalStringArg -Arguments $execArgs -Name "method" -Value $FlowMethod

            Write-Step "query exec_cap_query"
            $execPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_exec_cap_query" -Arguments $execArgs -TimeoutMilliseconds $rpcTimeoutMs
            [void](Assert-ResponseSuccess -ToolName "myflowhub_exec_cap_query" -Payload $execPayload)

            $flowReadArgs = @{}
            Add-OptionalUInt32Arg -Arguments $flowReadArgs -Name "executor_node" -Value $ExecutorNode

            Write-Step "query flow_list"
            $flowListPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_list" -Arguments $flowReadArgs -TimeoutMilliseconds $rpcTimeoutMs
            $flowListResponse = Assert-ResponseSuccess -ToolName "myflowhub_flow_list" -Payload $flowListPayload

            $selectedFlowID = (Get-StringValue $FlowID).Trim()
            if ([string]::IsNullOrWhiteSpace($selectedFlowID)) {
                $selectedFlowID = Get-FirstFlowID -Flows @($flowListResponse.flows)
            }

            if (-not [string]::IsNullOrWhiteSpace($selectedFlowID)) {
                $flowGetArgs = @{ flow_id = $selectedFlowID }
                Add-OptionalUInt32Arg -Arguments $flowGetArgs -Name "executor_node" -Value $ExecutorNode

                Write-Step "query flow_get ($selectedFlowID)"
                $flowGetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_get" -Arguments $flowGetArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_get" -Payload $flowGetPayload)

                $flowStatusArgs = @{ flow_id = $selectedFlowID }
                Add-OptionalUInt32Arg -Arguments $flowStatusArgs -Name "executor_node" -Value $ExecutorNode

                Write-Step "query flow_status ($selectedFlowID)"
                $flowStatusPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_status" -Arguments $flowStatusArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_status" -Payload $flowStatusPayload)
            }
            else {
                Write-Step "flow_get/status skipped: no flow_id was provided and flow_list returned no flows"
            }

            $stages.extended_read = "passed"
        }
        catch {
            $stages.extended_read = "failed"
            throw
        }
    }

    if ($EnableAuthoritySmoke) {
        try {
            Write-Step "stage: authority smoke"

            $pendingArgs = @{ limit = 20 }
            Add-OptionalUInt32Arg -Arguments $pendingArgs -Name "authority_id" -Value $AuthorityID
            Add-OptionalStringArg -Arguments $pendingArgs -Name "device_id" -Value $PermitDeviceID

            Write-Step "query auth_list_pending_registers"
            $pendingPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_list_pending_registers" -Arguments $pendingArgs -TimeoutMilliseconds $rpcTimeoutMs
            [void](Assert-ResponseSuccess -ToolName "myflowhub_auth_list_pending_registers" -Payload $pendingPayload)

            if (-not [string]::IsNullOrWhiteSpace($permitDeviceIDText) -and -not [string]::IsNullOrWhiteSpace($permitRoleText)) {
                $permitArgs = @{
                    device_id = $permitDeviceIDText
                    role      = $permitRoleText
                }
                Add-OptionalUInt32Arg -Arguments $permitArgs -Name "authority_id" -Value $AuthorityID

                Write-Step "issue register permit"
                $permitPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_issue_register_permit" -Arguments $permitArgs -TimeoutMilliseconds $rpcTimeoutMs
                $permitResponse = Assert-ResponseSuccess -ToolName "myflowhub_auth_issue_register_permit" -Payload $permitPayload
                $permitToken = Get-StringValue $permitResponse.permit
                if ([string]::IsNullOrWhiteSpace($permitToken)) {
                    throw "auth_issue_register_permit succeeded but did not return a permit token."
                }

                Write-Step "revoke register permit"
                $revokeArgs = @{ permit = $permitToken }
                Add-OptionalUInt32Arg -Arguments $revokeArgs -Name "authority_id" -Value $AuthorityID
                $revokePayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_revoke_register_permit" -Arguments $revokeArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_auth_revoke_register_permit" -Payload $revokePayload)
            }

            if ($PendingAction -eq "approve") {
                $approveArgs = @{ request_id = $pendingRequestIDText }
                Add-OptionalUInt32Arg -Arguments $approveArgs -Name "authority_id" -Value $AuthorityID
                Add-OptionalStringArg -Arguments $approveArgs -Name "role" -Value $ApprovalRole

                Write-Step "approve register request"
                $approvePayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_approve_register" -Arguments $approveArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_auth_approve_register" -Payload $approvePayload)
            }
            elseif ($PendingAction -eq "reject") {
                $rejectArgs = @{
                    request_id = $pendingRequestIDText
                    reason     = $RejectReason
                }
                Add-OptionalUInt32Arg -Arguments $rejectArgs -Name "authority_id" -Value $AuthorityID

                Write-Step "reject register request"
                $rejectPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_auth_reject_register" -Arguments $rejectArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_auth_reject_register" -Payload $rejectPayload)
            }

            $stages.authority = "passed"
        }
        catch {
            $stages.authority = "failed"
            throw
        }
    }

    if ($EnableWriteSmoke) {
        try {
            Write-Step "stage: write smoke"

            $writeFlowID = (Get-StringValue $FlowID).Trim()
            if ([string]::IsNullOrWhiteSpace($writeFlowID)) {
                $writeFlowID = New-TempResourceName -Prefix "mcp-smoke-flow"
            }

            $writeFlowName = (Get-StringValue $FlowName).Trim()
            if ([string]::IsNullOrWhiteSpace($writeFlowName)) {
                $writeFlowName = "MCP Smoke $writeFlowID"
            }

            $writeVarName = (Get-StringValue $VarName).Trim()
            if ([string]::IsNullOrWhiteSpace($writeVarName)) {
                $writeVarName = New-TempResourceName -Prefix "mcp_smoke_var"
            }

            $writeVarOwner = if ($VarOwner -gt 0) { $VarOwner } else { $effectiveNodeID }
            $writeFailureMessage = ""
            $flowCreated = $false
            $varCreated = $false

            try {
                Write-Step "query varstore_list"
                $varListPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_varstore_list" -Arguments @{
                    owner = $writeVarOwner
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_varstore_list" -Payload $varListPayload)

                Write-Step "set temp variable $writeVarName"
                $varSetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_varstore_set" -Arguments @{
                    name       = $writeVarName
                    value      = $VarValue
                    owner      = $writeVarOwner
                    visibility = "private"
                    type       = "string"
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_varstore_set" -Payload $varSetPayload)
                $varCreated = $true

                Write-Step "get temp variable $writeVarName"
                $varGetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_varstore_get" -Arguments @{
                    name  = $writeVarName
                    owner = $writeVarOwner
                } -TimeoutMilliseconds $rpcTimeoutMs
                $varGetResponse = Assert-ResponseSuccess -ToolName "myflowhub_varstore_get" -Payload $varGetPayload
                if ((Get-StringValue $varGetResponse.value) -ne $VarValue) {
                    throw "varstore_get returned an unexpected value for '$writeVarName'."
                }

                Write-Step "set temp flow $writeFlowID"
                $flowSetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_set" -Arguments @{
                    flow_id       = $writeFlowID
                    name          = $writeFlowName
                    executor_node = $ExecutorNode
                    trigger       = @{
                        type     = "interval"
                        every_ms = 60000
                    }
                    graph         = New-TempFlowGraph -Method $flowMethodText
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_set" -Payload $flowSetPayload)
                $flowCreated = $true

                Write-Step "query flow_list"
                $writeFlowListPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_list" -Arguments @{
                    executor_node = $ExecutorNode
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_list" -Payload $writeFlowListPayload)

                Write-Step "query flow_get ($writeFlowID)"
                $writeFlowGetPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_get" -Arguments @{
                    flow_id       = $writeFlowID
                    executor_node = $ExecutorNode
                } -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_get" -Payload $writeFlowGetPayload)

                Write-Step "run flow ($writeFlowID)"
                $flowRunPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_run" -Arguments @{
                    flow_id       = $writeFlowID
                    executor_node = $ExecutorNode
                } -TimeoutMilliseconds $rpcTimeoutMs
                $flowRunResponse = Assert-ResponseSuccess -ToolName "myflowhub_flow_run" -Payload $flowRunPayload

                $flowStatusArgs = @{
                    flow_id       = $writeFlowID
                    executor_node = $ExecutorNode
                }
                Add-OptionalStringArg -Arguments $flowStatusArgs -Name "run_id" -Value (Get-StringValue $flowRunResponse.run_id)

                Write-Step "query flow_status ($writeFlowID)"
                $flowStatusPayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_status" -Arguments $flowStatusArgs -TimeoutMilliseconds $rpcTimeoutMs
                [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_status" -Payload $flowStatusPayload)
            }
            catch {
                $writeFailureMessage = $_.Exception.Message
                if ([string]::IsNullOrWhiteSpace($writeFailureMessage)) {
                    $writeFailureMessage = $_.ToString()
                }
            }
            finally {
                if ($flowCreated) {
                    try {
                        Write-Step "cleanup flow_delete ($writeFlowID)"
                        $flowDeletePayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_flow_delete" -Arguments @{
                            flow_id       = $writeFlowID
                            executor_node = $ExecutorNode
                        } -TimeoutMilliseconds $rpcTimeoutMs
                        [void](Assert-ResponseSuccess -ToolName "myflowhub_flow_delete" -Payload $flowDeletePayload)
                    }
                    catch {
                        $cleanupMessage = $_.Exception.Message
                        if ([string]::IsNullOrWhiteSpace($cleanupMessage)) {
                            $cleanupMessage = $_.ToString()
                        }
                        $cleanupIssues.Add("flow_delete($writeFlowID): $cleanupMessage") | Out-Null
                    }
                }

                if ($varCreated) {
                    try {
                        Write-Step "cleanup varstore_revoke ($writeVarName)"
                        $varRevokePayload = Invoke-McpTool -State $mcpState -NextID ([ref]$rpcID) -Name "myflowhub_varstore_revoke" -Arguments @{
                            name  = $writeVarName
                            owner = $writeVarOwner
                        } -TimeoutMilliseconds $rpcTimeoutMs
                        [void](Assert-ResponseSuccess -ToolName "myflowhub_varstore_revoke" -Payload $varRevokePayload)
                    }
                    catch {
                        $cleanupMessage = $_.Exception.Message
                        if ([string]::IsNullOrWhiteSpace($cleanupMessage)) {
                            $cleanupMessage = $_.ToString()
                        }
                        $cleanupIssues.Add("varstore_revoke($writeVarName): $cleanupMessage") | Out-Null
                    }
                }
            }

            if (-not [string]::IsNullOrWhiteSpace($writeFailureMessage)) {
                if ($cleanupIssues.Count -gt 0) {
                    throw "$writeFailureMessage Cleanup: $($cleanupIssues -join '; ')"
                }
                throw $writeFailureMessage
            }

            if ($cleanupIssues.Count -gt 0) {
                throw "Write smoke cleanup failed: $($cleanupIssues -join '; ')"
            }

            $stages.write = "passed"
        }
        catch {
            if ($stages.write -ne "passed") {
                $stages.write = "failed"
            }
            throw
        }
    }

    Write-Step "smoke completed successfully"
    Write-StageSummary -Stages $stages
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
    if (-not [string]::IsNullOrWhiteSpace($selectedConfigKey)) {
        Write-Host ("ConfigGetKey: {0}" -f $selectedConfigKey)
    }
    if (-not [string]::IsNullOrWhiteSpace($selectedFlowID)) {
        Write-Host ("FlowReadID: {0}" -f $selectedFlowID)
    }
    if (-not [string]::IsNullOrWhiteSpace($writeFlowID)) {
        Write-Host ("WriteFlowID: {0}" -f $writeFlowID)
    }
    if (-not [string]::IsNullOrWhiteSpace($writeVarName)) {
        Write-Host ("WriteVarName: {0}" -f $writeVarName)
    }
    if ($EnableWriteSmoke) {
        Write-Host "Cleanup: ok"
    }
    exit 0
}
catch {
    $message = $_.Exception.Message
    if ([string]::IsNullOrWhiteSpace($message)) {
        $message = $_.ToString()
    }
    [Console]::Error.WriteLine($message)
    Write-StageSummary -Stages $stages
    if (-not [string]::IsNullOrWhiteSpace($resolvedConfigDir)) {
        Write-Host ("ConfigDir: {0}" -f $resolvedConfigDir)
    }
    if (-not [string]::IsNullOrWhiteSpace($selectedConfigKey)) {
        Write-Host ("ConfigGetKey: {0}" -f $selectedConfigKey)
    }
    if (-not [string]::IsNullOrWhiteSpace($selectedFlowID)) {
        Write-Host ("FlowReadID: {0}" -f $selectedFlowID)
    }
    if (-not [string]::IsNullOrWhiteSpace($writeFlowID)) {
        Write-Host ("WriteFlowID: {0}" -f $writeFlowID)
    }
    if (-not [string]::IsNullOrWhiteSpace($writeVarName)) {
        Write-Host ("WriteVarName: {0}" -f $writeVarName)
    }
    if ($cleanupIssues.Count -gt 0) {
        Write-Host "CleanupIssues:"
        foreach ($issue in $cleanupIssues) {
            Write-Host ("  {0}" -f $issue)
        }
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
