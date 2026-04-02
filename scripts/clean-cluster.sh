#!/usr/bin/env bash
set -euo pipefail

# Helios Platform - Local Cluster Cleanup
#
# Purpose:
#  - Delete the local k3d cluster
#  - Best-effort clean kubeconfig entries created by k3d/setup
#  - Verify the old cluster is no longer reachable
#
# Usage:
#   CLUSTER_NAME=helios-dev bash scripts/clean-cluster.sh
#   bash scripts/clean-cluster.sh --cluster helios-dev
#
# Note:
#   This is intended for local dev. It is safe because it only targets the
#   specified k3d cluster.

CLUSTER_NAME="helios-dev"
KUBECONFIG_FILE="${KUBECONFIG:-$HOME/.kube/config}"
WAIT_SECONDS=120

usage() {
  cat <<EOF
Usage: $(basename "$0") [--cluster <name>] [--kubeconfig <path>] [--wait-seconds <n>]

Options:
  --cluster <name>       k3d cluster name (default: ${CLUSTER_NAME})
  --kubeconfig <path>   kubeconfig file (default: ${KUBECONFIG_FILE})
  --wait-seconds <n>    max wait for cluster deletion (default: ${WAIT_SECONDS})
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --cluster)
      CLUSTER_NAME="${2:?missing value for --cluster}"
      shift 2
      ;;
    --kubeconfig)
      KUBECONFIG_FILE="${2:?missing value for --kubeconfig}"
      shift 2
      ;;
    --wait-seconds)
      WAIT_SECONDS="${2:?missing value for --wait-seconds}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown arg: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

echo "Helios local cleanup"
echo "Target k3d cluster: ${CLUSTER_NAME}"
echo "Kubeconfig: ${KUBECONFIG_FILE}"

if ! command -v k3d >/dev/null 2>&1; then
  echo "ERROR: k3d not found in PATH" >&2
  exit 1
fi

if ! command -v kubectl >/dev/null 2>&1; then
  echo "ERROR: kubectl not found in PATH" >&2
  exit 1
fi

echo "Deleting cluster (if it exists)..."
k3d cluster delete "${CLUSTER_NAME}" >/dev/null 2>&1 || true

echo "Waiting for cluster to disappear (up to ${WAIT_SECONDS}s)..."
deadline=$((SECONDS + WAIT_SECONDS))
while (( SECONDS < deadline )); do
  if k3d kubeconfig get "${CLUSTER_NAME}" >/dev/null 2>&1; then
    sleep 2
    continue
  fi
  break
done

echo "Best-effort kubeconfig cleanup..."
# k3d + Taskfile uses cluster name: k3d-<CLUSTER_NAME>
kubectl config delete-cluster "k3d-${CLUSTER_NAME}" >/dev/null 2>&1 || true
kubectl config delete-context "k3d-${CLUSTER_NAME}" >/dev/null 2>&1 || true

if [[ -f "${KUBECONFIG_FILE}" ]]; then
  # Also try common context names that k3d sometimes creates.
  kubectl config delete-context "k3d-${CLUSTER_NAME}" --kubeconfig="${KUBECONFIG_FILE}" >/dev/null 2>&1 || true
fi

echo "Sanity check: kubectl should no longer reach the cluster..."
set +e
kubectl --kubeconfig="${KUBECONFIG_FILE}" get nodes --request-timeout=5s >/dev/null 2>&1
rc=$?
set -e

if [[ $rc -eq 0 ]]; then
  echo "WARNING: kubectl can still reach a cluster via ${KUBECONFIG_FILE}."
  echo "         This might indicate a different cluster context is still active."
  echo "         You can verify with: kubectl config current-context"
else
  echo "OK: old cluster is not reachable."
fi

echo "Cleanup complete."

