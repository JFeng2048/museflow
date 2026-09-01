[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Token,

    [string]$Secret = $env:USER_TURNSTILE_SECRET,
    [string]$Endpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify",
    [string]$Action = "",
    [string]$RemoteIp = "",
    [uri]$Proxy = $null
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($Secret)) {
    throw "USER_TURNSTILE_SECRET is empty. Set it in the shell or pass -Secret explicitly."
}

if ([string]::IsNullOrWhiteSpace($Token)) {
    throw "Token is empty. Copy a fresh token from the frontend Turnstile callback."
}

$body = @{
    secret   = $Secret
    response = $Token
}
if (-not [string]::IsNullOrWhiteSpace($RemoteIp)) {
    $body.remoteip = $RemoteIp
}

$stopwatch = [Diagnostics.Stopwatch]::StartNew()
try {
    $request = @{
        Uri         = $Endpoint
        Method      = "Post"
        Body        = $body
        ContentType = "application/x-www-form-urlencoded"
        Headers     = @{ Accept = "application/json" }
    }
    if ($Proxy) {
        $request.Proxy = $Proxy
    }

    $response = Invoke-WebRequest @request
    $stopwatch.Stop()
    $result = $response.Content | ConvertFrom-Json

    [pscustomobject]@{
        Success      = $result.success
        HttpStatus   = [int]$response.StatusCode
        Action       = $result.action
        Hostname     = $result.hostname
        ErrorCodes   = ($result.'error-codes' -join ",")
        ElapsedMs    = $stopwatch.ElapsedMilliseconds
        Endpoint     = $Endpoint
        Proxy        = if ($Proxy) { $Proxy.AbsoluteUri } else { "<system/default>" }
    } | Format-List

    if (-not $result.success) {
        exit 2
    }
}
catch {
    $stopwatch.Stop()
    Write-Error ("Request failed after {0} ms: {1}" -f $stopwatch.ElapsedMilliseconds, $_.Exception.Message)
    Write-Host "Check DNS/TLS/proxy connectivity. The token is single-use; obtain a fresh frontend token before retrying."
    exit 1
}
