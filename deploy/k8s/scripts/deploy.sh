#!/usr/bin/env bash
set -euo pipefail

ENVIRONMENT="${1:-dev}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
K8S_DIR="$ROOT_DIR/deploy/k8s"
BASE_VALUES="$K8S_DIR/base/overlays/values.yaml"
BASE_SECRETS="$K8S_DIR/base/overlays/secrets.yaml"
APP_VALUES="$K8S_DIR/applications/overlays/values.yaml"
APP_SECRETS="$K8S_DIR/applications/overlays/secrets.yaml"

if [[ "$ENVIRONMENT" != "dev" ]]; then
  echo "Only the dev overlay is currently available: $ENVIRONMENT" >&2
  exit 1
fi

for file in "$BASE_SECRETS" "$APP_SECRETS"; do
  if [[ ! -f "$file" ]]; then
    echo "Missing secret file: $file" >&2
    echo "Copy the corresponding example.secrets.yaml first." >&2
    exit 1
  fi
done

helm upgrade --install postgres "$K8S_DIR/base/charts/postgres" \
  --namespace museflow --create-namespace \
  --values "$BASE_VALUES" --values "$BASE_SECRETS"
helm upgrade --install ollama "$K8S_DIR/base/charts/ollama" \
  --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS"
helm upgrade --install redis "$K8S_DIR/base/charts/redis" \
  --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS"
helm upgrade --install searxng "$K8S_DIR/base/charts/searxng" \
  --namespace museflow --values "$BASE_VALUES" --values "$BASE_SECRETS"

helm upgrade --install api-gateway "$K8S_DIR/applications/services/api-gateway" \
  --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS"
helm upgrade --install user-service "$K8S_DIR/applications/services/user-service" \
  --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS"
helm upgrade --install crawl4ai-service "$K8S_DIR/applications/services/crawl4ai-service" \
  --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS"
helm upgrade --install web "$K8S_DIR/applications/web/frontend" \
  --namespace museflow --values "$APP_VALUES" --values "$APP_SECRETS"
