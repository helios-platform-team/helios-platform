#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "$0")" && pwd)"
# shellcheck source=../../scripts/lib/env_helpers.sh
source "$SCRIPT_DIR/../../scripts/lib/env_helpers.sh"

decode_base64() {
  if printf 'Zg==' | base64 --decode >/dev/null 2>&1; then
    base64 --decode
    return
  fi
  if printf 'Zg==' | base64 -d >/dev/null 2>&1; then
    base64 -d
    return
  fi
  if printf 'Zg==' | base64 -D >/dev/null 2>&1; then
    base64 -D
    return
  fi
  if command -v openssl >/dev/null 2>&1; then
    openssl base64 -d -A
    return
  fi

  echo "No compatible base64 decoder found" >&2
  return 1
}

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Track background PIDs for cleanup
PIDS=()

wait_for_tcp() {
  local host="$1"
  local port="$2"
  local tries="${3:-25}"
  local delay_s="${4:-0.2}"
  local i=1
  while [ "$i" -le "$tries" ]; do
    if command -v nc >/dev/null 2>&1; then
      if nc -z "$host" "$port" >/dev/null 2>&1; then
        return 0
      fi
    else
      # Fallback: bash TCP check
      if (echo >/dev/tcp/"$host"/"$port") >/dev/null 2>&1; then
        return 0
      fi
    fi
    sleep "$delay_s"
    i=$((i + 1))
  done
  return 1
}

ensure_kubectl_access() {
  if ! kubectl version --client >/dev/null 2>&1; then
    echo -e "${RED}❌  kubectl not found (required for local dev).${NC}" >&2
    return 1
  fi
  if ! kubectl cluster-info >/dev/null 2>&1; then
    echo -e "${RED}❌  kubectl cannot reach the current cluster context.${NC}" >&2
    echo -e "${YELLOW}   Fix: start your cluster (kind/minikube/k3d) or switch context, then retry.${NC}" >&2
    echo -e "${YELLOW}   Debug: kubectl config current-context && kubectl cluster-info${NC}" >&2
    return 1
  fi
}

# 1. ArgoCD Token Automation
# core-install doesn't ship argocd-server, so the admin secret may not exist.
ARGOCD_AUTH_TOKEN=""
if kubectl -n argocd get secret argocd-initial-admin-secret &>/dev/null; then
  echo -e "${YELLOW}🔑  Fetching ArgoCD Admin Password...${NC}"
  ARGOCD_PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | decode_base64)

  echo -e "${YELLOW}🚀  Starting ArgoCD Port-Forward (localhost:8080)...${NC}"
  kubectl port-forward -n argocd svc/argocd-server 8080:443 > /dev/null 2>&1 &
  PIDS+=($!)
  sleep 2

  TOKEN_JSON=$(curl -k -s -X POST -H "Content-Type: application/json" \
    -d "{\"username\":\"admin\",\"password\":\"$ARGOCD_PASS\"}" \
    https://127.0.0.1:8080/api/v1/session)

  ARGOCD_AUTH_TOKEN=$(echo "$TOKEN_JSON" | sed 's/.*"token":"\([^"]*\)".*/\1/')

  if [ -n "$ARGOCD_AUTH_TOKEN" ] && [ "${#ARGOCD_AUTH_TOKEN}" -ge 20 ]; then
    echo -e "${GREEN}✅  ArgoCD Token Generated!${NC}"
  else
    echo -e "${YELLOW}⚠️   Could not generate ArgoCD token (ArgoCD proxy features may be unavailable)${NC}"
    ARGOCD_AUTH_TOKEN=""
  fi
else
  echo -e "${YELLOW}ℹ️   ArgoCD core-install detected (no admin secret). Skipping token generation.${NC}"
fi
export ARGOCD_AUTH_TOKEN

# Preflight: ensure kubectl can talk to the cluster before port-forwards.
ensure_kubectl_access

# 2. Gitea Port-Forward
echo -e "${YELLOW}🚀  Starting Gitea Port-Forward (localhost:3030)...${NC}"
kubectl port-forward -n gitea svc/gitea-http 3030:3000 >/dev/null 2>&1 &
PIDS+=($!)
if ! wait_for_tcp 127.0.0.1 3030 25 0.2; then
  echo -e "${RED}❌  Gitea port-forward did not become ready on 127.0.0.1:3030.${NC}" >&2
  echo -e "${YELLOW}   Debug: kubectl -n gitea get svc,pods && kubectl -n gitea port-forward svc/gitea-http 3030:3000${NC}" >&2
  exit 1
fi

# 3. Kubectl Proxy
echo -e "${YELLOW}🚀  Starting Kubectl Proxy (localhost:8001)...${NC}"
kubectl proxy --port=8001 >/dev/null 2>&1 &
PIDS+=($!)
if ! wait_for_tcp 127.0.0.1 8001 25 0.2; then
  echo -e "${RED}❌  kubectl proxy did not become ready on 127.0.0.1:8001.${NC}" >&2
  echo -e "${YELLOW}   Debug: kubectl proxy --port=8001${NC}" >&2
  exit 1
fi

APP_PID=0

# Cleanup function
cleanup() {
  echo -e "\n${YELLOW}🛑  Stopping background processes...${NC}"
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  if [ $APP_PID -ne 0 ]; then
      kill $APP_PID 2>/dev/null || true
  fi
  echo -e "${GREEN}Bye! 👋${NC}"
}
trap cleanup EXIT INT TERM

# 4. Start Backstage
echo -e "${GREEN}🌟  Starting Backstage Portal...${NC}"
# Load existing .env if present.
# Note: the repo-root `.env` is the source of truth, but `start-dev.sh` runs from `apps/portal/`.
ENV_FILE_CWD="./.env"
ENV_FILE_REPO_ROOT="$SCRIPT_DIR/../../.env"
if [ -f "$ENV_FILE_CWD" ]; then
  ENV_FILE="$ENV_FILE_CWD"
elif [ -f "$ENV_FILE_REPO_ROOT" ]; then
  ENV_FILE="$ENV_FILE_REPO_ROOT"
else
  ENV_FILE=""
fi

if [ -n "$ENV_FILE" ]; then
  SAFE_ENV_KEYS=(GITEA_TOKEN GITEA_USER GITEA_URL GITEA_INTERNAL_URL AUTH_GITHUB_CLIENT_ID AUTH_GITHUB_CLIENT_SECRET)
  for key in "${SAFE_ENV_KEYS[@]}"; do
    value="$(read_env_value "$ENV_FILE" "$key" || true)"
    if [ -n "$value" ]; then
      export "$key=$value"
    fi
  done
else
  echo -e "${YELLOW}⚠️  No .env found (checked apps/portal/.env and repo root).${NC}"
fi
export ARGOCD_AUTH_TOKEN # Re-export to ensure override

yarn start &
APP_PID=$!

wait $APP_PID
