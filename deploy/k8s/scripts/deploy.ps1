[CmdletBinding()]
param(
    [ValidateSet('base', 'app', 'all')]
    [string]$Scope = 'all',
    [string]$Kubeconfig
)

$ErrorActionPreference = 'Stop'
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$k8sDir = (Resolve-Path (Join-Path $scriptDir '..')).Path
$baseValues = Join-Path $k8sDir 'base/overlays/values.yaml'
$baseSecrets = Join-Path $k8sDir 'base/overlays/secrets.yaml'
$appValues = Join-Path $k8sDir 'applications/overlays/values.yaml'
$appSecrets = Join-Path $k8sDir 'applications/overlays/secrets.yaml'

$helmCommon = @('--namespace', 'museflow')
if ($Kubeconfig) {
    $helmCommon += "--kubeconfig=$Kubeconfig"
}

function Assert-SecretFile([string]$Path) {
    if (-not (Test-Path $Path)) {
           throw "Missing secret file: $Path. Copy example.secrets.yaml first."
    }
}

# Placeholder scan: keys still holding sample values only fail at runtime, often with
# misleading messages (e.g. USER_TURNSTILE_SECRET left as change-me, or filled with the
# frontend Site Key -> the gateway/user-service log shows invalid-input-secret).
function Test-SecretPlaceholders([string]$Path) {
    $hits = Select-String -Path $Path -Pattern 'change-me|your-smtp-password|0x\.\.\.' -ErrorAction SilentlyContinue |
        Where-Object { $_.Line -notmatch '^\s*#' }
    if ($hits) {
        Write-Warning "$Path still contains placeholder values; replace them before deploying:"
        foreach ($hit in $hits) {
            Write-Warning ("  line {0}: {1}" -f $hit.LineNumber, $hit.Line.Trim())
        }
    }
}

function Test-TurnstileSecret([string]$Path) {
    $line = Select-String -Path $Path -Pattern '^\s*USER_TURNSTILE_SECRET\s*:' -ErrorAction SilentlyContinue
    if (-not $line) {
        Write-Warning "$Path does not set USER_TURNSTILE_SECRET: captcha verification degrades to 'skipped'."
        return
    }
    if ($line.Line -match '^\s*USER_TURNSTILE_SECRET\s*:\s*("")?\s*$') {
        Write-Warning "USER_TURNSTILE_SECRET is empty: captcha verification degrades to 'skipped' (local use only)."
        Write-Warning "  For production set the Cloudflare Secret Key - NOT the frontend Site Key;"
        Write-Warning "  a wrong value shows up as invalid-input-secret in the user-service log."
    }
}

function Install-Chart([string]$Release, [string]$Chart, [string]$Values, [string]$Secrets, [bool]$CreateNamespace = $false) {
    $arguments = @('upgrade', '--install', $Release, $Chart) + $helmCommon
    if ($CreateNamespace) {
        $arguments += '--create-namespace'
    }
    $arguments += @('--values', $Values, '--values', $Secrets)
    & helm @arguments
}

if ($Scope -eq 'base' -or $Scope -eq 'all') {
    Assert-SecretFile $baseSecrets
    Test-SecretPlaceholders $baseSecrets
    Install-Chart 'postgres' (Join-Path $k8sDir 'base/charts/postgres') $baseValues $baseSecrets $true
    Install-Chart 'ollama' (Join-Path $k8sDir 'base/charts/ollama') $baseValues $baseSecrets
    Install-Chart 'redis' (Join-Path $k8sDir 'base/charts/redis') $baseValues $baseSecrets
    Install-Chart 'searxng' (Join-Path $k8sDir 'base/charts/searxng') $baseValues $baseSecrets
}

if ($Scope -eq 'app' -or $Scope -eq 'all') {
    Assert-SecretFile $appSecrets
    Test-SecretPlaceholders $appSecrets
    Test-TurnstileSecret $appSecrets
    Install-Chart 'config-service' (Join-Path $k8sDir 'applications/services/config-service') $appValues $appSecrets
    Install-Chart 'api-gateway' (Join-Path $k8sDir 'applications/services/api-gateway') $appValues $appSecrets
    Install-Chart 'user-service' (Join-Path $k8sDir 'applications/services/user-service') $appValues $appSecrets
    Install-Chart 'crawl4ai-service' (Join-Path $k8sDir 'applications/services/crawl4ai-service') $appValues $appSecrets
    Install-Chart 'web' (Join-Path $k8sDir 'applications/web/frontend') $appValues $appSecrets
    Install-Chart 'ingress' (Join-Path $k8sDir 'edge/ingress') $appValues $appSecrets
}
