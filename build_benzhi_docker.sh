#!/bin/bash
# 确保脚本在任何命令失败时立即退出。
set -e

# 第一个参数作为镜像名，第二个参数作为目标平台。
IMAGE_NAME=${1:-my-project}
DOCKER_PLATFORM=${2:-linux/amd64}

docker build --platform "$DOCKER_PLATFORM" -f benzhi.Dockerfile -t "$IMAGE_NAME" .

echo ""
echo "✅ Docker image '$IMAGE_NAME' built successfully!"
echo ""
echo "📋 Next steps (for testing):"
echo "  • Interactive shell: docker run -it $IMAGE_NAME:latest"
