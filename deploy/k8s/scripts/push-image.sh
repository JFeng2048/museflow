#!/usr/bin/env bash
# ===================================================================
# 离线镜像导入：本地 docker save → 上传服务器 → ctr 导入 K3s containerd
#
# 用法：
#   ./push-image.sh <服务器IP> <账号> <镜像> [<镜像>...] [选项]
#
# 示例：
#   ./push-image.sh 192.168.142.121 root museflow/web:latest
#   ./push-image.sh 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest
#   ./push-image.sh 192.168.142.121 ubuntu museflow/user-service:latest --sudo
#
# 选项：
#   -p, --port <端口>   SSH 端口，默认 22
#   -d, --dir  <目录>   服务器上存放镜像 tar 的目录，默认 /docker/images
#       --sudo          远程执行 ctr 时强制使用 sudo
#       --no-sudo       远程执行 ctr 时不使用 sudo
#   -h, --help          显示本帮助
#
# 密码处理：
#   密码不作为参数传递；由 ssh/scp 自行提示输入，输入时**不回显**、不落盘。
#   本脚本默认开启 SSH 连接复用（ControlMaster），整轮只需输入一次密码；
#   若本机安装了 sshpass，则使用 sshpass（完全不进入交互）。
#   提前配置好 SSH 公钥可完全免密。
#
# 服务器要求：
#   Linux 节点（K3s），具备 ctr（K3s 自带）或 k3s，且账号对 containerd 有权限
#   （root 或配置了免密 sudo）。
# ===================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TMP_DIR="$ROOT_DIR/tmp/images"

# ---------------- 输出helper ----------------
if [ -t 1 ]; then
  C_RESET=$'\033[0m'; C_INFO=$'\033[36m'; C_OK=$'\033[32m'; C_WARN=$'\033[33m'; C_ERR=$'\033[31m'; C_BOLD=$'\033[1m'
else
  C_RESET=; C_INFO=; C_OK=; C_WARN=; C_ERR=; C_BOLD=
fi
step() { printf '\n%s==> %s%s\n' "$C_BOLD" "$*" "$C_RESET"; }
info() { printf '%s[信息]%s %s\n' "$C_INFO" "$C_RESET" "$*"; }
ok()   { printf '%s[完成]%s %s\n' "$C_OK" "$C_RESET" "$*"; }
warn() { printf '%s[注意]%s %s\n' "$C_WARN" "$C_RESET" "$*" >&2; }
die()  { printf '%s[错误]%s %s\n' "$C_ERR" "$C_RESET" "$*" >&2; exit 1; }

usage() {
  cat <<'USAGE'
离线镜像导入：本地 docker save → 上传服务器 → ctr 导入 K3s containerd

用法：
  ./push-image.sh <服务器IP> <账号> <镜像> [<镜像>...] [选项]

示例：
  ./push-image.sh 192.168.142.121 root museflow/web:latest
  ./push-image.sh 192.168.142.121 root museflow/web:latest museflow/api-gateway:latest
  ./push-image.sh 192.168.142.121 ubuntu museflow/user-service:latest --sudo

选项：
  -p, --port <端口>   SSH 端口，默认 22
  -d, --dir  <目录>   服务器上存放镜像 tar 的目录，默认 /docker/images
      --sudo          远程执行 ctr 时强制使用 sudo
      --no-sudo       远程执行 ctr 时不使用 sudo
      --skip-key-setup 不询问配置公钥，直接用密码认证
  -h, --help          显示本帮助

密码处理：
  密码不作为参数传递，由 ssh/scp 提示输入（不回显、不落盘）。
  已有 SSH 公钥时完全免密；没有时会询问是否现场配置一次（只需输一次密码，
  之后永久免密）；拒绝配置则退回密码认证：装有 sshpass 时完全不进入交互，
  否则整轮只输入一次（通过 SSH 连接复用实现）。

服务器要求：
  Linux 节点（K3s）。containerd socket 自动识别：
  /run/k3s/containerd/containerd.sock（K3s 自带）优先，其次 /run/containerd/containerd.sock；
  账号对 containerd 有权限（root 或配置了免密 sudo）。
USAGE
}

# ---------------- 工具函数 ----------------
file_size() {
  if stat -c %s "$1" >/dev/null 2>&1; then stat -c %s "$1"; else stat -f %z "$1"; fi
}

human_size() {
  awk -v bytes="$1" 'BEGIN {
    split("B KB MB GB TB", unit, " ");
    i = 1;
    while (bytes >= 1024 && i < 5) { bytes /= 1024; i++ }
    printf "%.1f %s", bytes, unit[i]
  }'
}

# 把本机公钥追加到服务器 authorized_keys（幂等），装好并能免密登录时返回 0。
# 这样只需在首次输入一次密码，之后（含以后重新运行本脚本）都不必再输。
install_public_key() {
  local key="$HOME/.ssh/id_ed25519" pub="$key.pub" pubkey
  if [ ! -f "$key" ]; then
    info "未找到密钥，正在生成：$key"
    mkdir -p "$HOME/.ssh" && chmod 700 "$HOME/.ssh"
    ssh-keygen -t ed25519 -N '' -f "$key" -C museflow-push-image >/dev/null || { warn "ssh-keygen 执行失败"; return 1; }
  fi
  [ -f "$pub" ] || { warn "找不到公钥文件：$pub"; return 1; }
  pubkey="$(cat "$pub")"
  case "$pubkey" in
    ssh-*) ;;
    *) warn "公钥内容异常：$pub"; return 1 ;;
  esac
  cat "$pub" | ssh "${SSH_OPTS[@]}" "$SSH_TARGET" \
    "mkdir -p ~/.ssh && chmod 700 ~/.ssh && (grep -qxF '$pubkey' ~/.ssh/authorized_keys 2>/dev/null || cat >> ~/.ssh/authorized_keys) && chmod 600 ~/.ssh/authorized_keys && echo '[remote] public key installed'" \
    || return 1
  ssh "${SSH_OPTS[@]}" -o BatchMode=yes -o ConnectTimeout=8 "$SSH_TARGET" true >/dev/null 2>&1
}

# 远程执行 POSIX sh 脚本：脚本本身单引号包裹（内容不得含单引号），
# 动态值一律以位置参数传入（sh -c 'script' $0 $1 $2 ...），避免拼接注入。
run_remote() {
  local script="$1"; shift
  local args="" arg
  for arg in "$@"; do args="$args '$arg'"; done
  "${SSH_CMD[@]}" "${SSH_OPTS[@]}" "$SSH_TARGET" "sh -c '$script' museflow$args"
}

# ---------------- 参数解析 ----------------
PORT=22
REMOTE_DIR=/docker/images
SUDO_MODE=auto
SKIP_KEY_SETUP=0
POSITIONAL=()

while [ $# -gt 0 ]; do
  case "$1" in
    -p|--port)  [ $# -ge 2 ] || die "--port 需要一个端口号"; PORT="$2"; shift 2 ;;
    --port=*)   PORT="${1#*=}"; shift ;;
    -d|--dir)   [ $# -ge 2 ] || die "--dir 需要一个目录"; REMOTE_DIR="$2"; shift 2 ;;
    --dir=*)    REMOTE_DIR="${1#*=}"; shift ;;
    --sudo)     SUDO_MODE=force; shift ;;
    --no-sudo)  SUDO_MODE=never; shift ;;
    --skip-key-setup) SKIP_KEY_SETUP=1; shift ;;
    -h|--help)  usage; exit 0 ;;
    --)         shift; while [ $# -gt 0 ]; do POSITIONAL+=("$1"); shift; done ;;
    -*)         printf '%s\n' "未知选项：$1" >&2; usage >&2; exit 1 ;;
    *)          POSITIONAL+=("$1"); shift ;;
  esac
done

if [ ${#POSITIONAL[@]} -lt 3 ]; then
  printf '%s\n' "参数不足：至少需要 <服务器IP> <账号> <镜像>" >&2
  usage >&2
  exit 1
fi

SERVER_IP="${POSITIONAL[0]}"
SSH_USER="${POSITIONAL[1]}"
IMAGES=("${POSITIONAL[@]:2}")

# ---------------- 参数校验（同时保证远程拼串安全） ----------------
case "$PORT" in
  ''|*[!0-9]*) die "端口必须是数字：$PORT" ;;
esac
case "$SERVER_IP" in
  ''|*[!A-Za-z0-9.:_-]*) die "服务器地址含非法字符：$SERVER_IP" ;;
esac
case "$SSH_USER" in
  ''|*[!A-Za-z0-9._-]*) die "账号含非法字符：$SSH_USER" ;;
esac
case "$REMOTE_DIR" in
  /*) ;;
  *) die "远程目录必须是绝对路径：$REMOTE_DIR" ;;
esac
case "$REMOTE_DIR" in
  *[!A-Za-z0-9._/-]*) die "远程目录含非法字符：$REMOTE_DIR" ;;
esac
for image in "${IMAGES[@]}"; do
  case "$image" in
    ''|*[!A-Za-z0-9._/:@-]*) die "镜像名含非法字符：$image" ;;
  esac
done

# ---------------- 本地环境检查 ----------------
command -v docker >/dev/null 2>&1 || die "未找到 docker 命令，请先安装并启动 Docker"
docker info >/dev/null 2>&1 || die "docker 未运行（docker info 失败），请先启动 Docker Desktop"

SSH_TARGET="$SSH_USER@$SERVER_IP"
SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -o LogLevel=ERROR -p "$PORT")
SCP_OPTS=(-o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 -o LogLevel=ERROR -P "$PORT")

SSH_CMD=(ssh)
SCP_CMD=(scp)
CTL_SOCKET=""
SSH_KEY_OK=0

# 认证优先级：已有公钥（免密）> 现场配置一次公钥（之后永久免密）> sshpass > 连接复用
if ssh "${SSH_OPTS[@]}" -o BatchMode=yes -o ConnectTimeout=8 "$SSH_TARGET" true >/dev/null 2>&1; then
  SSH_KEY_OK=1
  ok "SSH 公钥认证可用，本轮无需输入密码"
elif [ "$SKIP_KEY_SETUP" = 0 ] && [ -t 0 ]; then
  warn "未检测到可用的公钥认证（否则本轮要输入密码）"
  printf '%s是否现在配置公钥（只需输入一次密码，之后永久免密）？[Y/n] %s' "$C_INFO" "$C_RESET"
  read -r answer
  case "$answer" in
    ''|y|Y)
      info "正在配置公钥，请按提示输入一次密码"
      if install_public_key; then
        SSH_KEY_OK=1
        ok "公钥配置完成，后续免密"
      else
        warn "公钥配置失败，改用密码认证"
      fi
      ;;
    *) info "已跳过公钥配置" ;;
  esac
fi

if [ "$SSH_KEY_OK" = 1 ]; then
  SSH_OPTS+=(-o BatchMode=yes)
  SCP_OPTS+=(-o BatchMode=yes)
elif command -v sshpass >/dev/null 2>&1; then
  printf '%s请输入 %s 的登录密码（不回显）：%s' "$C_INFO" "$SSH_TARGET" "$C_RESET"
  read -r -s SSHPASS
  printf '\n'
  [ -n "${SSHPASS:-}" ] || die "密码不能为空"
  export SSHPASS
  SSH_CMD=(sshpass -e ssh)
  SCP_CMD=(sshpass -e scp)
else
  CTL_SOCKET="${TMPDIR:-/tmp}/museflow-image-$$.sock"
  SSH_OPTS+=(-o ControlMaster=auto -o "ControlPath=$CTL_SOCKET" -o ControlPersist=120)
  SCP_OPTS+=(-o ControlMaster=auto -o "ControlPath=$CTL_SOCKET" -o ControlPersist=120)
  info "未检测到 sshpass：由 ssh 提示密码并复用连接（整轮只输入一次，不回显）"
fi

cleanup() {
  if [ -n "$CTL_SOCKET" ] && [ -S "$CTL_SOCKET" ]; then
    ssh -O exit -o "ControlPath=$CTL_SOCKET" "$SSH_TARGET" >/dev/null 2>&1 || true
  fi
  unset SSHPASS 2>/dev/null || true
}
trap cleanup EXIT

# ---------------- 远程脚本 ----------------
# 注意：远程脚本内不出现任何引号字符（单双引号都不用）。
# 参数均已通过白名单校验（无空格与 shell 元字符），因此照常拼接也安全；
# 这也让 Windows(PowerShell) 版可以直接复用同一段文本。
#
# 预检：创建目录、报告主机/权限/磁盘/已有文件（同名文件将被覆盖）
REMOTE_PREFLIGHT='
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
'

# 导入：校验文件大小 → ctr -n k8s.io images import → 回查镜像列表
REMOTE_IMPORT='
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
'

# ---------------- 执行 ----------------
step "预检：连接 $SSH_TARGET:$PORT"
info "目标目录：$REMOTE_DIR；本次共 ${#IMAGES[@]} 个镜像：${IMAGES[*]}"
FILE_NAMES=()
for image in "${IMAGES[@]}"; do
  FILE_NAMES+=("$(printf '%s' "$image" | tr '/:' '--').tar")
done
run_remote "$REMOTE_PREFLIGHT" "$REMOTE_DIR" "${FILE_NAMES[@]}" \
  || die "无法连接服务器或目标目录不可用：请检查 IP/端口/账号/密码以及目录权限"

mkdir -p "$TMP_DIR"
IMPORTED=()

for index in "${!IMAGES[@]}"; do
  image="${IMAGES[$index]}"
  file_name="${FILE_NAMES[$index]}"
  local_file="$TMP_DIR/$file_name"
  remote_file="$REMOTE_DIR/$file_name"

  step "镜像 $((index + 1))/${#IMAGES[@]}：$image"

  if ! docker image inspect "$image" >/dev/null 2>&1; then
    die "本地不存在镜像 $image；请先构建或拉取（当前镜像列表：docker images）"
  fi

  info "导出镜像（docker save → $local_file）"
  start_time=$(date +%s)
  docker save -o "$local_file" "$image" || die "docker save 失败：$image"
  end_time=$(date +%s)
  size=$(file_size "$local_file")
  ok "导出完成：$(human_size "$size")，耗时 $((end_time - start_time)) 秒"

  info "上传到 $SSH_TARGET:$remote_file（同名文件将覆盖）"
  "${SCP_CMD[@]}" "${SCP_OPTS[@]}" "$local_file" "$SSH_TARGET:$remote_file" \
    || die "上传失败：请检查网络、服务器磁盘空间与目录写权限"
  ok "上传完成"

  info "导入 containerd（ctr -n k8s.io images import）"
  run_remote "$REMOTE_IMPORT" "$remote_file" "$size" "$image" "$SUDO_MODE" \
    || die "导入失败：详见上方服务器输出（常见原因：权限不足、tar 损坏、磁盘空间不够）"
  ok "镜像就绪：$image"
  IMPORTED+=("$image")
done

# ---------------- 汇总 ----------------
step "全部完成"
printf '%s%d 个镜像已导入 %s%s：\n' "$C_OK" "${#IMPORTED[@]}" "$SSH_TARGET" "$C_RESET"
for image in "${IMPORTED[@]}"; do printf '  - %s\n' "$image"; done
printf '\n'
info "镜像文件位置：服务器 $REMOTE_DIR；本地 $TMP_DIR（已加入 .gitignore，可随时删除）"
info "Chart 中 imagePullPolicy=IfNotPresent，导入新镜像后需重启工作负载才会生效："
printf '    kubectl -n museflow rollout restart deployment/<名称>\n'
printf '    kubectl -n museflow get deploy -o wide        # 查看工作负载与镜像\n'
