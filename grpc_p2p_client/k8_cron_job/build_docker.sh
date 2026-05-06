#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Navigate to the parent directory (grpc_p2p_client) where the source files are located
cd "$(dirname "$0")/.."

echo "========================================================="
echo "Building P2P Load Test Docker Image for Kubernetes"
echo "========================================================="

echo ">>> Building Docker image (mump2p-load-test:latest) using Multi-Stage Build..."

echo ">>> Building Docker image (mump2p-load-test:latest)..."
# Build the docker image using the parent directory as the build context
docker build -f k8_cron_job/Dockerfile -t mump2p-load-test:latest .

echo ">>> Build completed successfully!"
echo ">>> You can now tag and push the image to your container registry."
echo "Example:"
echo "  docker tag mump2p-load-test:latest your-registry.com/mump2p-load-test:latest"
echo "  docker push your-registry.com/mump2p-load-test:latest"
echo "========================================================="
