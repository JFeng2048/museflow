#!/usr/bin/env bash
set -euo pipefail

SCOPE="all"
KUBECONFIG_ARGS=()

# 解析部署范围和可选的 kubeconfig 参数。
for argument in "$@"; do
  case "$argument" in
    --kubeconfig=*)
      KUBECONFIG_ARGS+=("$argument")
      ;;
    --kubeconfig)
      echo "请使用 --kubeconfig=/path/to/kubeconfig 格式指定 kubeconfig。" >&2
      exit 1
      ;;
    base|app|all)
      SCOPE="$argument"
      ;;
    *)
      echo "用法: $0 [base|app|all] [--kubeconfig=/path/to/kubeconfig]" >&2
      exit 1
      ;;
  esac
done

# 以脚本所在位置为基准，定位仓库和 Kubernetes 配置目录。
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
K8S_DIR="$ROOT_DIR/deploy/k8s"
BASE_VALUES="$K8S_DIR/base/overlays/values.yaml"
BASE_SECRETS="$K8S_DIR/base/overlays/secrets.yaml"
APP_VALUES="$K8S_DIR/applications/overlays/values.yaml"
APP_SECRETS="$K8S_DIR/applications/overlays/secrets.yaml"

# 检查指定部署范围所需的真实密钥文件。
REQUIRED_SECRETS=()
if [[ "$SCOPE" == "base" || "$SCOPE" == "all" ]]; then
  REQUIRED_SECRETS+=("$BASE_SECRETS")
fi
if [[ "$SCOPE" == "app" || "$SCOPE" == "all" ]]; then
  REQUIRED_SECRETS+=("$APP_SECRETS")
fi
for file in "${REQUIRED_SECRETS[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "缺少密钥文件: $file" >&2
    echo "请先复制对应的 example.secrets.yaml 并填写真实密钥。" >&2
    exit 1
  fi
done

# 占位符检查：密钥文件里仍是示例值的键，部署后往往要到运行期才暴露，且错误信息容易误导
# （典型：USER_TURNSTILE_SECRET 留着 change-me 或误填前端 Site Key，
#   人机验证接口会报 invalid-input-secret）。
for file in "${REQUIRED_SECRETS[@]}"; do
  placeholders="$(grep -nE 'change-me|your-smtp-password|0x\.\.\.' "$file" | grep -vE '^[0-9]+:[[:space:]]*#' || true)"
  if [[ -n "$placeholders" ]]; then
    echo "警告：$file 中仍有占位符，请确认已替换为真实值（否则部署后会在运行期报错）：" >&2
    printf '%s\n' "$placeholders" >&2
  fi
done

if [[ "$SCOPE" == "app" || "$SCOPE" == "all" ]]; then
  if ! grep -qE '^[[:space:]]*USER_TURNSTILE_SECRET:' "$APP_SECRETS"; then
    echo "警告：$APP_SECRETS 未配置 USER_TURNSTILE_SECRET，人机验证会降级为「跳过」，接口失去保护。" >&2
  elif grep -qE '^[[:space:]]*USER_TURNSTILE_SECRET:[[:space:]]*("")?[[:space:]]*$' "$APP_SECRETS"; then
    echo "警告：USER_TURNSTILE_SECRET 为空，人机验证会降级为「跳过」（仅适合本地）。" >&2
    echo "      生产环境请填 Cloudflare 控制台的 Secret Key——注意不是前端的 Site Key，" >&2
    echo "      填错会在 user-service 日志里出现 invalid-input-secret。" >&2
  fi
fi

if [[ "$SCOPE" == "base" || "$SCOPE" == "all" ]]; then
  helm upgrade --install postgres "$K8S_DIR/base/charts/postgres" \
    --namespace museflow --create-namespace \
    --values "$BASE_VALUES" --values "$BASE_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install ollama "$K8S_DIR/base/charts/ollama" \
    --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install redis "$K8S_DIR/base/charts/redis" \
    --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install searxng "$K8S_DIR/base/charts/searxng" \
    --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS" "${KUBECONFIG_ARGS[@]}"
fi

if [[ "$SCOPE" == "app" || "$SCOPE" == "all" ]]; then
  helm upgrade --install config-service "$K8S_DIR/applications/services/config-service" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install api-gateway "$K8S_DIR/applications/services/api-gateway" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install user-service "$K8S_DIR/applications/services/user-service" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install crawl4ai-service "$K8S_DIR/applications/services/crawl4ai-service" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install web "$K8S_DIR/applications/web/frontend" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install ingress "$K8S_DIR/edge/ingress" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
fi
