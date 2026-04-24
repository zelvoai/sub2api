#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
ENV_EXAMPLE="${DEPLOY_DIR}/.env.example"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.dev.yml"

usage() {
  cat <<'EOF'
用法:
  ./scripts/dev-up.sh [up|down|restart|logs|ps]

说明:
  - 自动初始化 deploy/.env（如果不存在）
  - 自动创建 deploy/data、deploy/postgres_data、deploy/redis_data
  - 使用 deploy/docker-compose.dev.yml 启动本地开发环境
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
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

  case "${action}" in
    up|down|restart|logs|ps) ;;
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

  ensure_env
  ensure_dirs

  cd "${DEPLOY_DIR}"

  case "${action}" in
    up)
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up --build -d
      echo
      echo "本地开发环境已启动:"
      echo "  Web: http://localhost:$(awk -F= '/^SERVER_PORT=/{print $2}' "${ENV_FILE}" | tail -n1)"
      echo "  日志: ./scripts/dev-up.sh logs"
      ;;
    down)
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" down
      ;;
    restart)
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" down
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" up --build -d
      ;;
    logs)
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" logs -f sub2api
      ;;
    ps)
      docker compose --env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" ps
      ;;
  esac
}

main "$@"
