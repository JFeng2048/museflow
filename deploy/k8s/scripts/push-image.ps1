<#
.SYNOPSIS
  离线镜像导入：本地 docker save → 上传服务器 → ctr 导入 K3s containerd

.DESCRIPTION
  用法：
    .\push-image.ps1 <服务器IP> <账号> <镜像> [<镜像>...] [选项]

  示例：
    .\push-image.ps1 192.168.142.121 root museflow/web:latest
    .\push-image.ps1 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest
    .\push-image.ps1 192.168.142.121 ubuntu museflow/user-service:latest -Sudo force

  选项：
    -Port <端口>             SSH 端口，默认 22
    -RemoteDir <目录>        服务器上存放镜像 tar 的目录，默认 /docker/images
    -Sudo auto|force|never   远程执行 ctr 时的提权方式，默认 auto
    -SkipKeySetup            不询问配置公钥，直接用密码认证

  密码与公钥：
    密码不作为参数传递，由 ssh/scp 提示输入（不回显、不落盘）。
    首次运行会询问是否把本机公钥装到服务器：同意后只需输入一次密码，
    之后所有操作（含以后重新运行本脚本）都免密；拒绝则本轮会提示 3 次密码。
    走公钥认证时不会向服务器保存密码，也不会写入任何本地凭据文件。

  服务器要求：
    Linux 节点（K3s）。脚本会自动识别 containerd socket：
    /run/k3s/containerd/containerd.sock（K3s 自带）优先，其次 /run/containerd/containerd.sock。
#>
[CmdletBinding()]
param(
    [Parameter(Position = 0)][string]$Server,
    [Parameter(Position = 1)][string]$User,
    [Parameter(Position = 2)][string[]]$Image,
    [int]$Port = 22,
    [string]$RemoteDir = '/docker/images',
    [ValidateSet('auto', 'force', 'never')][string]$Sudo = 'auto',
    [switch]$SkipKeySetup
)

$ErrorActionPreference = 'Stop'

$UsageText = @'
离线镜像导入：本地 docker save → 上传服务器 → ctr 导入 K3s containerd

用法：
  .\push-image.ps1 <服务器IP> <账号> <镜像> [<镜像>...] [选项]

示例：
  .\push-image.ps1 192.168.142.121 root museflow/web:latest
  .\push-image.ps1 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest
  .\push-image.ps1 192.168.142.121 ubuntu museflow/user-service:latest -Sudo force

选项：
  -Port <端口>             SSH 端口，默认 22
  -RemoteDir <目录>        服务器上存放镜像 tar 的目录，默认 /docker/images
  -Sudo auto|force|never   远程执行 ctr 时的提权方式，默认 auto
  -SkipKeySetup            不询问配置公钥，直接用密码认证

密码与公钥：
  首次运行会询问是否把本机公钥装到服务器：同意后只需输入一次密码，之后免密。

服务器要求：
  Linux 节点（K3s），脚本自动识别 containerd socket（/run/k3s/... 优先）。
'@

# 远程脚本与 push-image.sh 保持一致：内容不出现任何引号字符，
# 参数经白名单校验（无空格与 shell 元字符），因此拼接安全。
$RemotePreflight = @'
set -e
DIR=$1; shift
echo [remote] host: $(hostname) user: $(id -un) uid=$(id -u)
mkdir -p $DIR || { echo [remote] ERROR: cannot create or write $DIR >&2; exit 2; }
echo [remote] target dir: $DIR
echo [remote] disk: $(df -h $DIR | tail -n 1)
if [ -S /run/k3s/containerd/containerd.sock ]; then echo [remote] containerd socket: /run/k3s/containerd/containerd.sock; elif [ -S /run/containerd/containerd.sock ]; then echo [remote] containerd socket: /run/containerd/containerd.sock; else echo [remote] WARN: containerd socket not found, import will fail >&2; fi
if command -v k3s >/dev/null 2>&1; then echo [remote] import via: k3s ctr -n k8s.io; elif command -v ctr >/dev/null 2>&1; then echo [remote] import via: ctr -a SOCKET -n k8s.io; else echo [remote] WARN: neither ctr nor k3s found, import will fail >&2; fi
if [ $(id -u) = 0 ]; then echo [remote] privilege: root; elif sudo -n true >/dev/null 2>&1; then echo [remote] privilege: passwordless sudo; else echo [remote] WARN: not root and no passwordless sudo, import may fail >&2; fi
for f in $@; do if [ -e $DIR/$f ]; then echo [remote] exists, will overwrite: $DIR/$f; else echo [remote] new file: $DIR/$f; fi; done
'@

$RemoteImport = @'
set -e
FILE=$1; WANT=$2; IMAGE=$3; SUDO_MODE=$4
if [ ! -f $FILE ]; then echo [remote] ERROR: file not found: $FILE >&2; exit 3; fi
SIZE=$(stat -c %s $FILE 2>/dev/null || stat -f %z $FILE)
if [ $SIZE != $WANT ]; then echo [remote] ERROR: size mismatch, remote $SIZE bytes vs local $WANT bytes, upload truncated, please retry >&2; exit 4; fi
echo [remote] size check passed: $FILE $SIZE bytes
SUDO=
case $SUDO_MODE in
  force) SUDO=sudo ;;
  never) SUDO= ;;
  auto) if [ $(id -u) = 0 ]; then SUDO=; elif sudo -n true >/dev/null 2>&1; then SUDO=sudo; else echo [remote] WARN: not root and no passwordless sudo, import may fail >&2; fi ;;
  *) echo [remote] ERROR: unknown sudo mode $SUDO_MODE >&2; exit 6 ;;
esac
if [ -S /run/k3s/containerd/containerd.sock ]; then SOCKET=/run/k3s/containerd/containerd.sock
elif [ -S /run/containerd/containerd.sock ]; then SOCKET=/run/containerd/containerd.sock
else echo [remote] ERROR: containerd socket not found: tried /run/k3s/containerd/containerd.sock and /run/containerd/containerd.sock >&2; exit 5
fi
echo [remote] containerd socket: $SOCKET
echo [remote] importing: $IMAGE
if command -v k3s >/dev/null 2>&1; then
  echo [remote] import via: k3s ctr -n k8s.io
  $SUDO k3s ctr -n k8s.io images import $FILE
  LIST=$($SUDO k3s ctr -n k8s.io images ls -q)
else
  echo [remote] import via: ctr -a $SOCKET -n k8s.io
  $SUDO ctr -a $SOCKET -n k8s.io images import $FILE
  LIST=$($SUDO ctr -a $SOCKET -n k8s.io images ls -q)
fi
if echo $LIST | grep -Fq $IMAGE; then echo [remote] image ready: $IMAGE; else echo [remote] WARN: import finished but $IMAGE not found in image list >&2; fi
'@

function Write-Usage { Write-Host $UsageText }
function Write-Step([string]$Text) { Write-Host ''; Write-Host "==> $Text" -ForegroundColor White }
function Write-Info([string]$Text) { Write-Host "[信息] $Text" -ForegroundColor Cyan }
function Write-Ok([string]$Text) { Write-Host "[完成] $Text" -ForegroundColor Green }
function Write-Note([string]$Text) { Write-Host "[注意] $Text" -ForegroundColor Yellow }
function Fail([string]$Text) { Write-Host "[错误] $Text" -ForegroundColor Red; exit 1 }

function Format-Size([long]$Bytes) {
    $units = @('B', 'KB', 'MB', 'GB', 'TB')
    $index = 0
    $value = [double]$Bytes
    while ($value -ge 1024 -and $index -lt 4) { $value = $value / 1024; $index++ }
    return ('{0:N1} {1}' -f $value, $units[$index])
}

# 统一封装外部命令：
#   1) 临时放宽 ErrorActionPreference，避免 PowerShell 把远端 stderr 当成终止性错误中断脚本；
#   2) 用 Out-Host 直接输出，绝不能让它进入成功流 —— 否则命令输出会被当成返回值
#      一起返回给调用方（导致输出不显示、且退出码判断变成「数组 vs 0」而永远为真）。
function Invoke-Native([string]$Command, [string[]]$Arguments, [switch]$Quiet) {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = 'Continue'
    try {
        if ($Quiet) { & $Command @Arguments *> $null } else { & $Command @Arguments | Out-Host }
    } finally {
        $ErrorActionPreference = $previous
    }
    return $LASTEXITCODE
}

if (-not $Server -and -not $User -and -not $Image) { Write-Usage; exit 1 }

# ---------------- 参数校验（同时保证远程拼串安全） ----------------
if ($Port -lt 1 -or $Port -gt 65535) { Fail "端口必须在 1-65535 之间：$Port" }
if ($Server -notmatch '^[A-Za-z0-9.:_-]+$') { Fail "服务器地址含非法字符：$Server" }
if ($User -notmatch '^[A-Za-z0-9._-]+$') { Fail "账号含非法字符：$User" }
if (-not $RemoteDir.StartsWith('/')) { Fail "远程目录必须是绝对路径：$RemoteDir" }
if ($RemoteDir -notmatch '^[A-Za-z0-9._/-]+$') { Fail "远程目录含非法字符：$RemoteDir" }
if (-not $Image -or $Image.Count -eq 0) { Fail '至少需要一个镜像，例如 museflow/web:latest' }
foreach ($img in $Image) {
    if ($img -notmatch '^[A-Za-z0-9._/:@-]+$') { Fail "镜像名含非法字符：$img" }
}

# ---------------- 本地环境检查 ----------------
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) { Fail '未找到 docker 命令，请先安装并启动 Docker Desktop' }
if ((Invoke-Native 'docker' @('info') -Quiet) -ne 0) { Fail 'docker 未运行（docker info 失败），请先启动 Docker Desktop' }
if (-not (Get-Command ssh -ErrorAction SilentlyContinue)) { Fail '未找到 ssh 命令，请启用 Windows 可选功能 OpenSSH 客户端' }
if (-not (Get-Command scp -ErrorAction SilentlyContinue)) { Fail '未找到 scp 命令，请启用 Windows 可选功能 OpenSSH 客户端' }

$ScriptDir = $PSScriptRoot
$RootDir = (Resolve-Path (Join-Path $ScriptDir '..\..\..')).Path
$TmpDir = Join-Path $RootDir 'tmp\images'

$target = "$User@$Server"
$sshArgs = @('-o', 'StrictHostKeyChecking=accept-new', '-o', 'ConnectTimeout=10', '-o', 'LogLevel=ERROR', '-p', "$Port")
$scpArgs = @('-o', 'StrictHostKeyChecking=accept-new', '-o', 'ConnectTimeout=10', '-o', 'LogLevel=ERROR', '-P', "$Port")

function Invoke-Remote([string]$Script, [string[]]$Arguments) {
    $argText = ''
    if ($Arguments) { foreach ($item in $Arguments) { $argText += " '$item'" } }
    $body = $Script.Replace([string][char]13, '')
    $remote = "sh -c '$body' museflow$argText"
    $callArgs = $sshArgs + @($target, $remote)
    return (Invoke-Native 'ssh' $callArgs)
}

function Test-SshKeyAuth {
    $keyArgs = @('-o', 'BatchMode=yes', '-o', 'StrictHostKeyChecking=accept-new', '-o', 'ConnectTimeout=8', '-o', 'LogLevel=ERROR', '-p', "$Port", $target, 'true')
    return ((Invoke-Native 'ssh' $keyArgs -Quiet) -eq 0)
}

# 把本机公钥追加到服务器 authorized_keys（幂等）。
# 公钥内容作为参数传给 ssh，不使用管道，避免 PowerShell 文本管道破坏内容。
function Install-SshPublicKey {
    $keyPath = Join-Path $HOME '.ssh\id_ed25519'
    if (-not (Test-Path $keyPath)) {
        Write-Info "未找到密钥，正在生成：$keyPath"
        New-Item -ItemType Directory -Path (Join-Path $HOME '.ssh') -Force | Out-Null
        if ((Invoke-Native 'ssh-keygen' @('-t', 'ed25519', '-N', '', '-f', $keyPath, '-C', 'museflow-push-image')) -ne 0) {
            return $false
        }
    }
    $pubPath = "$keyPath.pub"
    if (-not (Test-Path $pubPath)) { Write-Note "找不到公钥文件：$pubPath"; return $false }
    $pubKey = (Get-Content -Raw $pubPath).Trim()
    if ($pubKey -notmatch '^ssh-') { Write-Note "公钥内容异常：$pubPath"; return $false }

    $script = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && (grep -qxF '$pubKey' ~/.ssh/authorized_keys 2>/dev/null || echo '$pubKey' >> ~/.ssh/authorized_keys) && chmod 600 ~/.ssh/authorized_keys && echo [remote] public key installed"
    $keyInstallArgs = $sshArgs + @($target, $script)
    if ((Invoke-Native 'ssh' $keyInstallArgs) -ne 0) { return $false }
    return (Test-SshKeyAuth)
}

# ---------------- 执行 ----------------
Write-Step "预检：连接 $target`:$Port"
Write-Info "目标目录：$RemoteDir；本次共 $($Image.Count) 个镜像：$($Image -join ', ')"

if (Test-SshKeyAuth) {
    Write-Ok 'SSH 公钥认证可用，本轮不会提示输入密码'
} elseif ($SkipKeySetup) {
    Write-Note '本轮使用密码认证：预检/上传/导入会各提示一次（均不回显）'
} else {
    Write-Note '未检测到可用的 SSH 公钥认证，否则本轮要输入 3 次密码'
    $answer = Read-Host '是否现在配置一次公钥（输入一次密码后永久免密）？[Y/n]'
    if ($answer -eq '' -or $answer -match '^[Yy]') {
        Write-Info '正在配置公钥，请输入一次密码'
        if (Install-SshPublicKey) { Write-Ok '公钥配置完成，后续操作免密' } else { Write-Note '公钥配置失败，本轮改用密码认证（3 次提示）' }
    } else {
        Write-Note '已跳过公钥配置，本轮使用密码认证（3 次提示）'
    }
}

$fileNames = @()
foreach ($img in $Image) {
    $fileNames += (($img -replace '[/:]', '-') + '.tar')
}

$preflightCode = Invoke-Remote $RemotePreflight (@($RemoteDir) + $fileNames)
if ($preflightCode -ne 0) { Fail '无法连接服务器或目标目录不可用：请检查 IP/端口/账号/密码以及目录权限' }

if (-not (Test-Path $TmpDir)) { New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null }
$imported = @()

for ($i = 0; $i -lt $Image.Count; $i++) {
    $img = $Image[$i]
    $fileName = $fileNames[$i]
    $localFile = Join-Path $TmpDir $fileName
    $remoteFile = "$RemoteDir/$fileName"

    Write-Step "镜像 $($i + 1)/$($Image.Count)：$img"

    if ((Invoke-Native 'docker' @('image', 'inspect', $img) -Quiet) -ne 0) {
        Fail "本地不存在镜像 $img；请先构建或拉取（docker images 查看）"
    }

    Write-Info "导出镜像（docker save → $localFile）"
    $watch = [Diagnostics.Stopwatch]::StartNew()
    if ((Invoke-Native 'docker' @('save', '-o', $localFile, $img)) -ne 0) { Fail "docker save 失败：$img" }
    $watch.Stop()
    $size = (Get-Item $localFile).Length
    Write-Ok "导出完成：$(Format-Size $size)，耗时 $([int]$watch.Elapsed.TotalSeconds) 秒"

    Write-Info "上传到 $target`:$remoteFile（同名文件将覆盖）"
    $scpCallArgs = $scpArgs + @($localFile, "${target}:$remoteFile")
    if ((Invoke-Native 'scp' $scpCallArgs) -ne 0) {
        Fail '上传失败：请检查网络、服务器磁盘空间与目录写权限'
    }
    Write-Ok '上传完成'

    Write-Info '导入 containerd（ctr -n k8s.io images import）'
    if ((Invoke-Remote $RemoteImport @($remoteFile, "$size", $img, $Sudo)) -ne 0) {
        Fail '导入失败：详见上方服务器输出（常见原因：权限不足、containerd socket 不对、tar 损坏、磁盘空间不够）'
    }
    Write-Ok "镜像就绪：$img"
    $imported += $img
}

# ---------------- 汇总 ----------------
Write-Step '全部完成'
Write-Host "$($imported.Count) 个镜像已导入 ${target}：" -ForegroundColor Green
foreach ($img in $imported) { Write-Host "  - $img" }
Write-Host ''
Write-Info "镜像文件位置：服务器 $RemoteDir；本地 $TmpDir（已被 .gitignore 忽略，可随时删除）"
Write-Info 'Chart 中 imagePullPolicy=IfNotPresent，导入新镜像后需重启工作负载才会生效：'
Write-Host '    kubectl -n museflow rollout restart deployment/<名称>'
Write-Host '    kubectl -n museflow get deploy -o wide        # 查看工作负载与镜像'
exit 0
