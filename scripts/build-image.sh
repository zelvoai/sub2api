#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
  cat <<'EOF'
用法:
  ./scripts/build-image.sh [--image NAME] [--tag TAG] [--latest]

示例:
  ./scripts/build-image.sh
  ./scripts/build-image.sh --tag 2026-04-24-node
  ./scripts/build-image.sh --image registry.example.com/sub2api --tag v1.0.0 --latest
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
  fi
}

IMAGE_NAME="sub2api"
IMAGE_TAG=""
TAG_LATEST="false"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --image)
      IMAGE_NAME="$2"
      shift 2
      ;;
    --tag)
      IMAGE_TAG="$2"
      shift 2
      ;;
    --latest)
      TAG_LATEST="true"
      shift
      ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "未知参数: $1" >&2
      usage
      exit 1
      ;;
  esac
done

require_cmd docker
require_cmd git

if [[ -z "${IMAGE_TAG}" ]]; then
  GIT_SHA="$(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
  BUILD_TIME="$(date +%Y%m%d-%H%M%S)"
  IMAGE_TAG="${BUILD_TIME}-${GIT_SHA}"
fi

PRIMARY_REF="${IMAGE_NAME}:${IMAGE_TAG}"

echo "构建镜像: ${PRIMARY_REF}"

BUILD_ARGS=(
  --build-arg "GOPROXY=${GOPROXY:-https://goproxy.cn,direct}"
  --build-arg "GOSUMDB=${GOSUMDB:-sum.golang.google.cn}"
)

docker build \
  -t "${PRIMARY_REF}" \
  "${BUILD_ARGS[@]}" \
  -f "${REPO_ROOT}/Dockerfile" \
  "${REPO_ROOT}"

if [[ "${TAG_LATEST}" == "true" ]]; then
  docker tag "${PRIMARY_REF}" "${IMAGE_NAME}:latest"
  echo "已额外打 tag: ${IMAGE_NAME}:latest"
fi

echo "完成: ${PRIMARY_REF}"
