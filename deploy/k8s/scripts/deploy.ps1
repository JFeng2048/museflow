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
    Install-Chart 'postgres' (Join-Path $k8sDir 'base/charts/postgres') $baseValues $baseSecrets $true
    Install-Chart 'ollama' (Join-Path $k8sDir 'base/charts/ollama') $baseValues $baseSecrets
    Install-Chart 'redis' (Join-Path $k8sDir 'base/charts/redis') $baseValues $baseSecrets
    Install-Chart 'searxng' (Join-Path $k8sDir 'base/charts/searxng') $baseValues $baseSecrets
}

if ($Scope -eq 'app' -or $Scope -eq 'all') {
    Assert-SecretFile $appSecrets
    Install-Chart 'api-gateway' (Join-Path $k8sDir 'applications/services/api-gateway') $appValues $appSecrets
    Install-Chart 'user-service' (Join-Path $k8sDir 'applications/services/user-service') $appValues $appSecrets
    Install-Chart 'crawl4ai-service' (Join-Path $k8sDir 'applications/services/crawl4ai-service') $appValues $appSecrets
    Install-Chart 'web' (Join-Path $k8sDir 'applications/web/frontend') $appValues $appSecrets
}
