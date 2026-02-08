#!/bin/bash
set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. ArgoCD Token Automation
echo -e "${YELLOW}🔑  Fetching ArgoCD Admin Password...${NC}"
ARGOCD_PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)

echo -e "${YELLOW}🔄  Generating ArgoCD Token...${NC}"
# Use temporary port-forward to get token if not running
# We'll start port-forward first!

echo -e "${YELLOW}🚀  Starting ArgoCD Port-Forward (localhost:8080)...${NC}"
kubectl port-forward -n argocd svc/argocd-server 8080:443 > /dev/null 2>&1 &
PF_PID=$!
sleep 2 # Wait for it to be ready

# Get Token via Curl
TOKEN_JSON=$(curl -k -s -X POST -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"password\":\"$ARGOCD_PASS\"}" https://127.0.0.1:8080/api/v1/session)

# Extract token (simple string extraction since jq might be missing)
ARGOCD_AUTH_TOKEN=$(echo $TOKEN_JSON | sed 's/.*"token":"\([^"]*\)".*/\1/')

if [ -z "$ARGOCD_AUTH_TOKEN" ] || [ "${#ARGOCD_AUTH_TOKEN}" -lt 20 ]; then
  echo -e "${RED}❌  Failed to generate ArgoCD Token! Check if ArgoCD is running.${NC}"
  kill $PF_PID
  exit 1
fi

echo -e "${GREEN}✅  ArgoCD Token Generated!${NC}"
export ARGOCD_AUTH_TOKEN

# 2. Kubectl Proxy
echo -e "${YELLOW}🚀  Starting Kubectl Proxy (localhost:8001)...${NC}"
kubectl proxy --port=8001 > /dev/null 2>&1 &
PROXY_PID=$!

APP_PID=0

# Cleanup function
cleanup() {
  echo -e "\n${YELLOW}🛑  Stopping background processes...${NC}"
  kill $PF_PID
  kill $PROXY_PID
  if [ $APP_PID -ne 0 ]; then
      kill $APP_PID
  fi
  echo -e "${GREEN}Bye! 👋${NC}"
}
trap cleanup EXIT INT TERM

# 3. Start Backstage
echo -e "${GREEN}🌟  Starting Backstage Portal...${NC}"
# Load existing .env if present (but override TOKEN)
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi
export ARGOCD_AUTH_TOKEN # Re-export to ensure override

yarn start &
APP_PID=$!

wait $APP_PID
