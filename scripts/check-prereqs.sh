#!/usr/bin/env bash
# =============================================================================
# Helios Platform - Prerequisite Checker
# =============================================================================
# Verifies that all required tools are installed and properly configured.
# Can be run standalone or via `task check`.
#
# Usage: ./scripts/check-prereqs.sh [--env]
#   --env   Also validate that .env exists and required variables are set
# =============================================================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

ERRORS=0
WARNINGS=0

pass()  { echo -e "  ${GREEN}[OK]${NC}  $1"; }
warn()  { echo -e "  ${YELLOW}[WARN]${NC} $1"; WARNINGS=$((WARNINGS + 1)); }
fail()  { echo -e "  ${RED}[FAIL]${NC} $1"; ERRORS=$((ERRORS + 1)); }

# ---------------------------------------------------------------------------
# Version comparison: returns 0 if $1 >= $2
# ---------------------------------------------------------------------------
version_gte() {
  printf '%s\n%s' "$2" "$1" | sort -t. -k1,1n -k2,2n -k3,3n -C
}

# ---------------------------------------------------------------------------
# Check a single binary
# $1 = binary name
# $2 = minimum version (empty string = any)
# $3 = version extraction command
# $4 = install hint
# ---------------------------------------------------------------------------
check_tool() {
  local name="$1" min_ver="$2" ver_cmd="$3" hint="$4"

  if ! command -v "$name" &>/dev/null; then
    fail "$name not found. Install: $hint"
    return
  fi

  if [[ -n "$min_ver" ]]; then
    local actual_ver
    actual_ver=$(eval "$ver_cmd" 2>/dev/null || echo "unknown")
    if [[ "$actual_ver" == "unknown" ]]; then
      warn "$name installed but could not determine version"
    elif version_gte "$actual_ver" "$min_ver"; then
      pass "$name $actual_ver (>= $min_ver)"
    else
      fail "$name $actual_ver is below minimum $min_ver. $hint"
    fi
  else
    pass "$name $(command -v "$name")"
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo -e "\n${BOLD}Helios Platform - Prerequisite Check${NC}\n"
echo "----------------------------------------------"

echo -e "\n${BOLD}Core Tools${NC}"
check_tool "go" "1.24" \
  "go version | grep -oP 'go\K[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1" \
  "https://go.dev/dl/"

check_tool "docker" "" \
  "docker --version | grep -oP '[0-9]+\.[0-9]+\.[0-9]+' | head -1" \
  "https://docs.docker.com/get-docker/"

check_tool "kubectl" "" \
  "kubectl version --client -o json 2>/dev/null | grep -oP '\"gitVersion\":\\s*\"v\K[0-9]+\.[0-9]+\.[0-9]+' | head -1" \
  "https://kubernetes.io/docs/tasks/tools/"

check_tool "k3d" "" \
  "k3d version | head -1 | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+'" \
  "https://k3d.io/ or: curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash"

check_tool "cue" "" \
  "cue version | head -1 | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+'" \
  "go install cuelang.org/go/cmd/cue@latest"

echo -e "\n${BOLD}Node.js / Frontend${NC}"
check_tool "node" "22.0.0" \
  "node --version | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+'" \
  "https://nodejs.org/ or use nvm: nvm install 22"

check_tool "yarn" "" \
  "yarn --version 2>/dev/null | head -1" \
  "corepack enable && corepack prepare yarn@4 --activate"

echo -e "\n${BOLD}Runtime Checks${NC}"

if docker info &>/dev/null; then
  pass "Docker daemon is running"
else
  fail "Docker daemon is not running. Start Docker first."
fi

if [[ -f "$HOME/.kube/config" ]] || [[ -n "${KUBECONFIG:-}" ]]; then
  pass "Kubeconfig found"
else
  warn "No kubeconfig found (~/.kube/config). One will be created when you run 'task setup:cluster'."
fi

# ---------------------------------------------------------------------------
# Optional: .env validation (--env flag)
# ---------------------------------------------------------------------------
CHECK_ENV=false
for arg in "$@"; do
  [[ "$arg" == "--env" ]] && CHECK_ENV=true
done

if $CHECK_ENV; then
  echo -e "\n${BOLD}Environment Variables (.env)${NC}"

  REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
  ENV_FILE="${REPO_ROOT}/.env"

  if [[ ! -f "$ENV_FILE" ]]; then
    fail ".env file not found at repo root. Run: cp .env.example .env"
  else
    pass ".env file exists"

    REQUIRED_VARS=(GITHUB_TOKEN GITHUB_USER AUTH_GITHUB_CLIENT_ID AUTH_GITHUB_CLIENT_SECRET)
    set -a; source "$ENV_FILE"; set +a

    for var in "${REQUIRED_VARS[@]}"; do
      val="${!var:-}"
      if [[ -z "$val" || "$val" == ghp_xxxx* || "$val" == "your-"* ]]; then
        fail "$var is not set (or still has placeholder value) in .env"
      else
        pass "$var is configured"
      fi
    done
  fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "----------------------------------------------"
if [[ $ERRORS -gt 0 ]]; then
  echo -e "${RED}${BOLD}$ERRORS error(s)${NC} and ${YELLOW}$WARNINGS warning(s)${NC}. Fix the errors above before proceeding."
  exit 1
elif [[ $WARNINGS -gt 0 ]]; then
  echo -e "${GREEN}${BOLD}All required tools found.${NC} ${YELLOW}$WARNINGS warning(s)${NC} to review."
  exit 0
else
  echo -e "${GREEN}${BOLD}All checks passed!${NC}"
  exit 0
fi
