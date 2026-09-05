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
  helm upgrade --install api-gateway "$K8S_DIR/applications/services/api-gateway" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install user-service "$K8S_DIR/applications/services/user-service" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install crawl4ai-service "$K8S_DIR/applications/services/crawl4ai-service" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
  helm upgrade --install web "$K8S_DIR/applications/web/frontend" \
    --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS" "${KUBECONFIG_ARGS[@]}"
fi
