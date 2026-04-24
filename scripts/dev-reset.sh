#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.dev.yml"
ENV_FILE="${DEPLOY_DIR}/.env"
TMP_OVERRIDE_FILE=""
KEEP_ENV=0

usage() {
  cat <<'USAGE'
用法:
  ./scripts/dev-reset.sh [--keep-env]

说明:
  - 停掉本地 dev-up / dev-hot 相关容器
  - 删除本地开发 PostgreSQL/Redis 持久化数据
  - 删除本地开发 data 目录与 .dev-data
  - 默认保留 deploy/.env（这样你改过的 ADMIN_EMAIL / ADMIN_PASSWORD 会在下一次首次初始化时生效）

常见场景:
  - 改了 ADMIN_PASSWORD 但登录还是旧密码
  - 想完全清空本地开发库，重新按当前 .env 初始化
USAGE
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
  fi
}

cleanup() {
  if [[ -n "${TMP_OVERRIDE_FILE}" && -f "${TMP_OVERRIDE_FILE}" ]]; then
    rm -f "${TMP_OVERRIDE_FILE}"
  fi
}

build_compose_args() {
  local -n out_ref=$1
  out_ref=()

  if [[ -f "${ENV_FILE}" ]]; then
    out_ref+=(--env-file "${ENV_FILE}")
  fi

  out_ref+=(-f "${COMPOSE_FILE}")

  if [[ "${FORCE_DOCKER_NAMED_VOLUMES:-}" == "1" || "${REPO_ROOT}" == /mnt/* ]]; then
    TMP_OVERRIDE_FILE="$(mktemp)"
    cat > "${TMP_OVERRIDE_FILE}" <<'OVERRIDE'
services:
  postgres:
    volumes:
      - sub2api_dev_postgres_data:/var/lib/postgresql/data
  redis:
    volumes:
      - sub2api_dev_redis_data:/data

volumes:
  sub2api_dev_postgres_data:
  sub2api_dev_redis_data:
OVERRIDE
    out_ref+=(-f "${TMP_OVERRIDE_FILE}")
  fi
}

remove_volume_if_exists() {
  local volume="$1"
  if docker volume inspect "${volume}" >/dev/null 2>&1; then
    docker volume rm -f "${volume}" >/dev/null
    echo "已删除 volume: ${volume}"
  fi
}

remove_dir_contents() {
  local dir="$1"
  if [[ -d "${dir}" ]]; then
    rm -rf "${dir}"
    echo "已删除目录: ${dir}"
  fi
}

main() {
  local compose_args=()

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --keep-env)
        KEEP_ENV=1
        shift
        ;;
      -h|--help|help)
        usage
        exit 0
        ;;
      *)
        echo "不支持的参数: $1" >&2
        usage
        exit 1
        ;;
    esac
  done

  require_cmd docker
  trap cleanup EXIT

  build_compose_args compose_args

  cd "${DEPLOY_DIR}"

  docker compose "${compose_args[@]}" down >/dev/null 2>&1 || true

  remove_volume_if_exists "deploy_sub2api_dev_postgres_data"
  remove_volume_if_exists "deploy_sub2api_dev_redis_data"
  remove_volume_if_exists "deploy_sub2api_hot_postgres_data"
  remove_volume_if_exists "deploy_sub2api_hot_redis_data"

  remove_dir_contents "${DEPLOY_DIR}/postgres_data"
  remove_dir_contents "${DEPLOY_DIR}/redis_data"
  remove_dir_contents "${DEPLOY_DIR}/data"
  remove_dir_contents "${REPO_ROOT}/.dev-data"

  mkdir -p "${DEPLOY_DIR}/postgres_data" "${DEPLOY_DIR}/redis_data" "${DEPLOY_DIR}/data"

  if [[ ${KEEP_ENV} -eq 0 ]]; then
    echo "保留 ${ENV_FILE} 不变（默认行为）"
  fi

  echo
  echo "本地开发数据已重置。"
  echo "下一步可执行："
  echo "  ./scripts/dev-up.sh up"
  echo "或"
  echo "  ./scripts/dev-hot.sh all"
  echo
  echo "注意：当前 .env 中的 ADMIN_EMAIL / ADMIN_PASSWORD 将在下一次首次初始化时生效。"
}

main "$@"
