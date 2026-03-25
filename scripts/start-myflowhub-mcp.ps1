[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $PSCommandPath
$repoRoot = Split-Path -Parent $scriptRoot
$goCommand = Get-Command go -ErrorAction Stop
$runArgs = @("run", "./cmd/myflowhub-mcp") + $ForwardArgs
$previousGoWork = $env:GOWORK
$exitCode = 0

try {
    $env:GOWORK = "off"
    Push-Location $repoRoot
    try {
        & $goCommand.Source @runArgs
        $exitCode = $LASTEXITCODE
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

exit $exitCode
