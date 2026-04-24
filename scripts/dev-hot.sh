#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy"
ENV_FILE="${DEPLOY_DIR}/.env"
ENV_EXAMPLE="${DEPLOY_DIR}/.env.example"
COMPOSE_FILE="${DEPLOY_DIR}/docker-compose.dev.yml"
TMP_OVERRIDE_FILE=""
TMP_AIR_TOML=""

usage() {
  cat <<'EOF'
用法:
  ./scripts/dev-hot.sh [all|infra|backend|frontend|down|logs]

说明:
  - 启动本地 PostgreSQL/Redis 容器，但后端/前端直接在本机运行
  - 前端默认 Vite 热更新
  - 后端如果检测到 air，会自动启用热重载；否则使用普通 go run
  - 默认 all：先启动 infra，再启动 backend + frontend

可用环境变量:
  HOT_SERVER_PORT=8080      后端端口
  HOT_FRONTEND_PORT=3000    前端端口
  HOT_DATABASE_PORT=5432    本机映射 PostgreSQL 端口
  HOT_REDIS_PORT=6379       本机映射 Redis 端口
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
  fi
}

ensure_frontend_deps() {
  if [[ ! -d "${REPO_ROOT}/frontend/node_modules" || ! -x "${REPO_ROOT}/frontend/node_modules/.bin/vite" ]]; then
    echo "[dev-hot] frontend: 检测到依赖未安装，正在执行 pnpm install..."
    (
      cd "${REPO_ROOT}/frontend"
      pnpm install
    )
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
      if (!updated) print key "=" value
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
    chmod 600 "${ENV_FILE}" || true
  fi
}

cleanup() {
  if [[ -n "${TMP_OVERRIDE_FILE}" && -f "${TMP_OVERRIDE_FILE}" ]]; then
    rm -f "${TMP_OVERRIDE_FILE}"
  fi
  if [[ -n "${TMP_AIR_TOML}" && -f "${TMP_AIR_TOML}" ]]; then
    rm -f "${TMP_AIR_TOML}"
  fi
}

build_compose_args() {
  local -n out_ref=$1
  local db_port="${HOT_DATABASE_PORT:-5432}"
  local redis_port="${HOT_REDIS_PORT:-6379}"

  TMP_OVERRIDE_FILE="$(mktemp)"
  cat > "${TMP_OVERRIDE_FILE}" <<EOF
services:
  postgres:
    ports:
      - "127.0.0.1:${db_port}:5432"
    volumes:
      - sub2api_hot_postgres_data:/var/lib/postgresql/data
  redis:
    ports:
      - "127.0.0.1:${redis_port}:6379"
    volumes:
      - sub2api_hot_redis_data:/data

volumes:
  sub2api_hot_postgres_data:
  sub2api_hot_redis_data:
EOF

  out_ref=(--env-file "${ENV_FILE}" -f "${COMPOSE_FILE}" -f "${TMP_OVERRIDE_FILE}")
}

start_infra() {
  local compose_args=()
  build_compose_args compose_args
  docker compose "${compose_args[@]}" up -d postgres redis
}

stop_infra() {
  local compose_args=()
  build_compose_args compose_args
  docker compose "${compose_args[@]}" down
}

logs_infra() {
  local compose_args=()
  build_compose_args compose_args
  docker compose "${compose_args[@]}" logs -f postgres redis
}

load_env_file() {
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
}

run_backend() {
  load_env_file

  local server_port="${HOT_SERVER_PORT:-${SERVER_PORT:-8080}}"
  local db_port="${HOT_DATABASE_PORT:-5432}"
  local redis_port="${HOT_REDIS_PORT:-6379}"
  local data_dir="${REPO_ROOT}/.dev-data"

  mkdir -p "${data_dir}"

  export AUTO_SETUP=true
  export DATA_DIR="${data_dir}"
  export SERVER_HOST=0.0.0.0
  export SERVER_PORT="${server_port}"
  export SERVER_MODE=debug
  export RUN_MODE="${RUN_MODE:-standard}"
  export DATABASE_HOST=127.0.0.1
  export DATABASE_PORT="${db_port}"
  export DATABASE_USER="${POSTGRES_USER:-sub2api}"
  export DATABASE_PASSWORD="${POSTGRES_PASSWORD}"
  export DATABASE_DBNAME="${POSTGRES_DB:-sub2api}"
  export DATABASE_SSLMODE="${DATABASE_SSLMODE:-disable}"
  export REDIS_HOST=127.0.0.1
  export REDIS_PORT="${redis_port}"
  export REDIS_PASSWORD="${REDIS_PASSWORD:-}"
  export REDIS_DB="${REDIS_DB:-0}"
  export ADMIN_EMAIL="${ADMIN_EMAIL:-admin@sub2api.local}"
  export ADMIN_PASSWORD="${ADMIN_PASSWORD:-}"
  export JWT_SECRET="${JWT_SECRET}"
  export TOTP_ENCRYPTION_KEY="${TOTP_ENCRYPTION_KEY}"
  export TZ="${TZ:-Asia/Shanghai}"

  cd "${REPO_ROOT}/backend"

  if command -v air >/dev/null 2>&1; then
    TMP_AIR_TOML="$(mktemp)"
    cat > "${TMP_AIR_TOML}" <<EOF
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/sub2api-dev ./cmd/server"
bin = "./tmp/sub2api-dev"
include_ext = ["go", "yaml", "yml", "json"]
exclude_dir = ["tmp", "vendor", "frontend"]
delay = 300
stop_on_error = true
EOF
    echo "[dev-hot] backend: air 热重载已启用"
    exec air -c "${TMP_AIR_TOML}"
  fi

  echo "[dev-hot] backend: 未检测到 air，使用普通 go run（修改 Go 代码后需手动重启本脚本）"
  exec go run ./cmd/server
}

run_frontend() {
  local server_port="${HOT_SERVER_PORT:-8080}"
  local frontend_port="${HOT_FRONTEND_PORT:-3000}"

  ensure_frontend_deps

  cd "${REPO_ROOT}"
  export VITE_DEV_PROXY_TARGET="http://127.0.0.1:${server_port}"
  export VITE_DEV_PORT="${frontend_port}"

  echo "[dev-hot] frontend: Vite 热更新已启用 -> http://127.0.0.1:${frontend_port}"
  exec pnpm --dir frontend run dev
}

run_all() {
  local backend_pid=""
  trap '[[ -n "${backend_pid}" ]] && kill "${backend_pid}" 2>/dev/null || true; cleanup' EXIT INT TERM

  start_infra
  run_backend &
  backend_pid=$!

  run_frontend
}

main() {
  local action="${1:-all}"

  case "${action}" in
    all|infra|backend|frontend|down|logs) ;;
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

  case "${action}" in
    all|frontend)
      require_cmd pnpm
      ;;
  esac

  case "${action}" in
    all|backend)
      require_cmd go
      ;;
  esac

  trap cleanup EXIT
  ensure_env

  case "${action}" in
    all) run_all ;;
    infra) start_infra ;;
    backend) run_backend ;;
    frontend) run_frontend ;;
    down) stop_infra ;;
    logs) logs_infra ;;
  esac
}

main "$@"
