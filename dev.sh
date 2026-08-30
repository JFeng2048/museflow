#!/usr/bin/env bash
# ============================================================
#  MuseFlow 开发环境一键热重载（仓库根目录入口）
#
#  用法：
#    ./dev.sh            启动全部后端服务（默认）
#    ./dev.sh gateway    仅 api-gateway（HTTP :5001）
#    ./dev.sh user       仅 user-service（gRPC :5002）
#    ./dev.sh worker     仅 user-service worker（asynq 消费端，无端口）
#    ./dev.sh web        仅前端（pnpm dev）
#    ./dev.sh full       后端全部 + 前端
#    ./dev.sh help       显示帮助
#
#  有 tmux 时每个服务一个窗口（推荐，可随时切换查看日志）；
#  没有 tmux 则退化为后台进程，Ctrl+C 一次性结束全部。
# ============================================================
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SESSION="museflow"

TARGET="${1:-all}"

# 把 Go 工具目录加入 PATH，保证能找到 air
GOBIN="$(go env GOPATH 2>/dev/null || echo "${HOME}/go")/bin"
export PATH="${PATH}:${GOBIN}"

usage() {
  cat <<'EOF'
Usage: ./dev.sh [all|gateway|user|worker|web|full|help]

  all       api-gateway + user-service + user-service worker (default)
  gateway   api-gateway only          (HTTP  :5001)
  user      user-service only         (gRPC  :5002)
  worker    user-service worker only  (asynq consumer)
  web       frontend only             (pnpm dev)
  full      backend all + frontend
  help      show this message

Requires air: go install github.com/air-verse/air@latest
EOF
}

case "${TARGET}" in
  help|-h|--help) usage; exit 0 ;;
  all|gateway|user|worker|web|full) ;;
  *) echo "Unknown target: ${TARGET}" >&2; echo; usage; exit 1 ;;
esac

if [[ "${TARGET}" != "web" ]] && ! command -v air >/dev/null 2>&1; then
  echo "[ERROR] air not found. Install it first:" >&2
  echo "        go install github.com/air-verse/air@latest" >&2
  exit 1
fi

if [[ ! -f "${ROOT}/.env" ]]; then
  echo "[WARN] ${ROOT}/.env not found; services fall back to defaults / system env."
  echo "       Copy .env.example to .env and fill in the values."
fi

echo
echo "  MuseFlow dev launcher"
echo "  root : ${ROOT}"
echo

# ── tmux 模式：一个服务一个窗口 ──────────────────────────────
run_tmux() {
  local name="$1" dir="$2" cfg="$3" desc="$4"
  local air_cmd="air"
  [[ -n "${cfg}" ]] && air_cmd="air -c '${cfg}'"
  local cmd="cd '${dir}' && echo '== ${name} (${desc}) ==' && ${air_cmd}"

  if tmux has-session -t "${SESSION}" 2>/dev/null; then
    tmux new-window -t "${SESSION}" -n "${name}" "${cmd}"
  else
    tmux new-session -d -s "${SESSION}" -n "${name}" "${cmd}"
  fi
  echo "  -> ${name} (${desc})"
}

# ── 无 tmux：退化为后台进程，统一收口 ────────────────────────
PIDS=()
cleanup() {
  echo
  echo "  Stopping services ..."
  for pid in "${PIDS[@]}"; do
    kill "${pid}" 2>/dev/null || true
  done
  exit 0
}

run_bg() {
  local name="$1" dir="$2" cfg="$3" desc="$4"
  local -a cmd=(air)
  [[ -n "${cfg}" ]] && cmd=(air -c "${cfg}")

  ( cd "${dir}" && "${cmd[@]}" ) &
  PIDS+=("$!")
  echo "  -> ${name} (${desc}, pid $!)"
}

run_web() {
  if [[ ! -f "${ROOT}/web/package.json" ]]; then
    echo "  [WARN] web/package.json not found, skip frontend."
    return 0
  fi
  if [[ -n "${USE_TMUX:-}" ]]; then
    run_tmux "web" "${ROOT}/web" "" "Vite dev server"
  else
    ( cd "${ROOT}/web" && pnpm dev ) &
    PIDS+=("$!")
    echo "  -> web (Vite dev server, pid $!)"
  fi
}

# 统一分发：有 tmux 走窗口，没有则走后台进程
start_service() {
  if [[ -n "${USE_TMUX:-}" ]]; then
    run_tmux "$@"
  else
    run_bg "$@"
  fi
}

USE_TMUX=""
if [[ "${TARGET}" != "web" ]] && command -v tmux >/dev/null 2>&1; then
  USE_TMUX=1
fi

case "${TARGET}" in
  all|full)
    start_service "api-gateway"         "${ROOT}/services/api-gateway"  ""                 "HTTP :5001"
    start_service "user-service"        "${ROOT}/services/user-service" ""                 "gRPC :5002"
    start_service "user-service-worker" "${ROOT}/services/user-service" ".air.worker.toml" "asynq consumer"
    [[ "${TARGET}" == "full" ]] && run_web
    ;;
  gateway) start_service "api-gateway"         "${ROOT}/services/api-gateway"  ""                 "HTTP :5001" ;;
  user)    start_service "user-service"        "${ROOT}/services/user-service" ""                 "gRPC :5002" ;;
  worker)  start_service "user-service-worker" "${ROOT}/services/user-service" ".air.worker.toml" "asynq consumer" ;;
  web)     run_web ;;
esac

echo
if [[ -n "${USE_TMUX}" ]]; then
  echo "  tmux session '${SESSION}' is running."
  echo "    attach : tmux attach -t ${SESSION}"
  echo "    switch : Ctrl+b then n / p"
  echo "    stop   : tmux kill-session -t ${SESSION}"
else
  trap cleanup INT TERM
  echo "  Press Ctrl+C to stop all services."
  wait
fi
echo
