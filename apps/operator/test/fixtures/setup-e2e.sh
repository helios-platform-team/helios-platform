#!/bin/bash

set -e

echo "===================================="
echo "Helios Operator E2E Test Setup"
echo "===================================="

# Get the namespace from argument or use default
NAMESPACE=${1:-e2e-test}

echo "📦 Creating test namespace: $NAMESPACE"
kubectl create namespace $NAMESPACE || echo "Namespace already exists"

echo "🔑 Creating ServiceAccount for Tekton"
kubectl create serviceaccount tekton-bot-sa -n $NAMESPACE || echo "ServiceAccount already exists"

echo "⚙️  Installing mock manifest-generation-pipeline"
kubectl apply -f $(dirname "$0")/mock-manifest-pipeline.yaml -n $NAMESPACE

echo "✅ E2E test environment setup complete!"
echo ""
echo "Test namespace: $NAMESPACE"
echo "ServiceAccount: tekton-bot-sa"
echo ""
echo "Run tests with: make test-e2e"
