#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
ENV_EXAMPLE="${DEPLOY_DIR}/.env.example"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.dev.yml"
TMP_OVERRIDE_FILE=""

usage() {
  cat <<'EOF'
用法:
  ./scripts/dev-up.sh [up|build|rebuild|down|restart|logs|ps]

说明:
  - 自动初始化 deploy/.env（如果不存在）
  - 自动创建 deploy/data、deploy/postgres_data、deploy/redis_data
  - 使用 deploy/docker-compose.dev.yml 启动本地开发环境
  - up: 默认不强制重建镜像，适合日常反复启动
  - build: 强制重新构建并启动
  - rebuild: 等同于 down 后再 build
EOF
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
  out_ref=(--env-file "${ENV_FILE}" -f "${COMPOSE_FILE}")

  if [[ "${FORCE_DOCKER_NAMED_VOLUMES:-}" == "1" || "${REPO_ROOT}" == /mnt/* ]]; then
    TMP_OVERRIDE_FILE="$(mktemp)"
    cat > "${TMP_OVERRIDE_FILE}" <<'EOF'
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
EOF
    out_ref+=(-f "${TMP_OVERRIDE_FILE}")
  fi
}

set_env_value() {
  local file="$1"
  local key="$2"
  local value="$3"
  local tmp
  tmp="$(mktemp)"
  awk -v key="${key}" -v value="${value}" '
    BEGIN { updated = 0 }
    index($0, key "=") == 1 {
      print key "=" value
      updated = 1
      next
    }
    { print }
    END {
      if (!updated) {
        print key "=" value
      }
    }
  ' "${file}" > "${tmp}"
  mv "${tmp}" "${file}"
}

generate_secret() {
  openssl rand -hex 32
}

ensure_env() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    cp "${ENV_EXAMPLE}" "${ENV_FILE}"
    set_env_value "${ENV_FILE}" "POSTGRES_PASSWORD" "$(generate_secret)"
    set_env_value "${ENV_FILE}" "JWT_SECRET" "$(generate_secret)"
    set_env_value "${ENV_FILE}" "TOTP_ENCRYPTION_KEY" "$(generate_secret)"
    set_env_value "${ENV_FILE}" "SERVER_PORT" "8080"
    chmod 600 "${ENV_FILE}" || true
    echo "已初始化 ${ENV_FILE}"
  fi
}

ensure_dirs() {
  mkdir -p \
    "${DEPLOY_DIR}/data" \
    "${DEPLOY_DIR}/postgres_data" \
    "${DEPLOY_DIR}/redis_data"
}

main() {
  local action="${1:-up}"
  local compose_args=()

  case "${action}" in
    up|build|rebuild|down|restart|logs|ps) ;;
    -h|--help|help)
      usage
      exit 0
      ;;
    *)
      echo "不支持的动作: ${action}" >&2
      usage
      exit 1
      ;;
  esac

  require_cmd docker
  require_cmd openssl
  trap cleanup EXIT

  ensure_env
  ensure_dirs
  build_compose_args compose_args

  cd "${DEPLOY_DIR}"

  case "${action}" in
    up)
      docker compose "${compose_args[@]}" up -d
      echo
      echo "本地开发环境已启动:"
      echo "  Web: http://localhost:$(awk -F= '/^SERVER_PORT=/{print $2}' "${ENV_FILE}" | tail -n1)"
      echo "  日志: ./scripts/dev-up.sh logs"
      if [[ -n "${TMP_OVERRIDE_FILE}" ]]; then
        echo "  提示: 当前仓库位于 /mnt/*，PostgreSQL/Redis 已自动改用 Docker named volumes，避免 Windows/NTFS 权限问题"
      fi
      ;;
    build)
      docker compose "${compose_args[@]}" up --build -d
      ;;
    rebuild)
      docker compose "${compose_args[@]}" down
      docker compose "${compose_args[@]}" up --build -d
      ;;
    down)
      docker compose "${compose_args[@]}" down
      ;;
    restart)
      docker compose "${compose_args[@]}" down
      docker compose "${compose_args[@]}" up -d
      ;;
    logs)
      docker compose "${compose_args[@]}" logs -f sub2api
      ;;
    ps)
      docker compose "${compose_args[@]}" ps
      ;;
  esac
}

main "$@"
