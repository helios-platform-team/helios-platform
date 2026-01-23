#!/bin/bash
# Script to clean up and rebuild devcontainer

set -e

echo "🧹 Cleaning up DevContainer resources..."
echo ""

# Stop and remove containers
echo "📦 Stopping and removing containers..."
CONTAINERS=$(docker ps -a --filter "label=devcontainer.local_folder" --format "{{.ID}} {{.Names}}" 2>/dev/null || true)
if [ -n "$CONTAINERS" ]; then
    echo "$CONTAINERS" | while read -r id name; do
        echo "  Removing container: $name ($id)"
        docker rm -f "$id" 2>/dev/null || true
    done
    echo "✅ Containers removed"
else
    echo "  No containers found"
fi
echo ""

# Clean up dangling images
echo "🖼️  Cleaning up dangling images..."
DANGLING=$(docker images -f "dangling=true" -q 2>/dev/null | wc -l)
if [ "$DANGLING" -gt 0 ]; then
    docker image prune -f
    echo "✅ Removed $DANGLING dangling images"
else
    echo "  No dangling images found"
fi
echo ""

# Optional: Clean build cache (commented out by default)
# Uncomment if you want to clean build cache
# echo "🗑️  Cleaning build cache..."
# docker builder prune -f
# echo "✅ Build cache cleaned"
# echo ""

# Optional: Remove volumes (commented out to preserve cache)
# Uncomment if you want to start completely fresh
# echo "💾 Removing volumes..."
# docker volume rm helios-operator-go-mod-cache 2>/dev/null || true
# docker volume rm helios-operator-go-build-cache 2>/dev/null || true
# docker volume rm dind-var-lib-docker 2>/dev/null || true
# echo "✅ Volumes removed"
# echo ""

echo "✅ Cleanup complete!"
echo ""
echo "🧭 Next Steps:"
echo "  1. Close this VS Code window"
echo "  2. Reopen in container: Press F1 → 'Dev Containers: Reopen in Container'"
echo ""
echo "💡 For a complete rebuild:"
echo "  Press F1 → 'Dev Containers: Rebuild Container'"
echo ""
