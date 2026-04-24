#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
DEPLOY_DIR="${REPO_ROOT}/deploy"
ENV_EXAMPLE="${DEPLOY_DIR}/.env.example"
COMPOSE_SOURCE="${DEPLOY_DIR}/docker-compose.local.yml"

usage() {
  cat <<'EOF'
用法:
  ./scripts/deploy-remote.sh --host root@1.2.3.4 --node 1 [可选参数]

环境变量:
  SSH_PASSWORD           可选；设置后脚本会用密码方式自动执行 ssh/文件上传

必填:
  --host HOST             远程 SSH 地址，例如 root@1.2.3.4
  --node NODE             1 / 2 / node-1 / node-2

可选:
  --image IMAGE:TAG       要部署的本地镜像，默认 sub2api:latest
  --env-file FILE         本地环境变量文件，默认 deploy/.env
  --remote-base DIR       远程基目录，默认 /root/sub2api
  --port PORT             远程宿主机端口；node-1 默认 8081，node-2 默认 8082
  --external-services     使用外部 PostgreSQL + Redis，不启动 compose 内置 postgres/redis
  --external-db           仅使用外部 PostgreSQL
  --external-redis        仅使用外部 Redis
  --external-network NET  让 sub2api 额外加入指定外部 Docker 网络（可重复传入）
  --build                 如果本地镜像不存在，则先自动构建
  --sync-env              强制用本地 env 覆盖远程 .env
  --pull-never            启动时添加 --pull never，避免 compose 误拉远程镜像

示例:
  ./scripts/deploy-remote.sh --host root@1.2.3.4 --node 1 --build
  ./scripts/deploy-remote.sh --host root@1.2.3.4 --node node-2 --image sub2api:20260424-abcd --port 8092
  ./scripts/deploy-remote.sh --host root@1.2.3.4 --node 1 --external-services --sync-env
EOF
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令: $1" >&2
    exit 1
  fi
}

SSH_PASSWORD="${SSH_PASSWORD:-}"
ASKPASS_FILE=""
SSH_RETRY_COUNT="${SSH_RETRY_COUNT:-3}"
SSH_CONNECT_TIMEOUT="${SSH_CONNECT_TIMEOUT:-15}"

setup_ssh_auth() {
  if [[ -z "${SSH_PASSWORD}" ]]; then
    return 0
  fi

  ASKPASS_FILE="$(mktemp)"
  cat > "${ASKPASS_FILE}" <<EOF
#!/usr/bin/env bash
echo '${SSH_PASSWORD}'
EOF
  chmod 700 "${ASKPASS_FILE}"
}

cleanup_ssh_auth() {
  if [[ -n "${ASKPASS_FILE}" ]]; then
    rm -f "${ASKPASS_FILE}"
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

normalize_node() {
  case "$1" in
    1|node-1) echo "node-1" ;;
    2|node-2) echo "node-2" ;;
    *)
      echo "非法节点: $1（仅支持 1 / 2 / node-1 / node-2）" >&2
      exit 1
      ;;
  esac
}

default_port_for_node() {
  case "$1" in
    node-1) echo "8081" ;;
    node-2) echo "8082" ;;
  esac
}

ensure_local_image() {
  local image_ref="$1"
  local should_build="$2"

  if docker image inspect "${image_ref}" >/dev/null 2>&1; then
    return 0
  fi

  if [[ "${should_build}" != "true" ]]; then
    echo "本地镜像不存在: ${image_ref}" >&2
    echo "请先执行 ./scripts/build-image.sh 或加上 --build" >&2
    exit 1
  fi

  local image_name="${image_ref%:*}"
  local image_tag="${image_ref##*:}"
  "${REPO_ROOT}/scripts/build-image.sh" --image "${image_name}" --tag "${image_tag}"
}

prepare_temp_env() {
  local source_file="$1"
  local port="$2"
  local target_file="$3"

  cp "${source_file}" "${target_file}"

  if ! grep -q '^POSTGRES_PASSWORD=' "${target_file}" || [[ -z "$(awk -F= '/^POSTGRES_PASSWORD=/{print $2}' "${target_file}" | tail -n1)" ]]; then
    set_env_value "${target_file}" "POSTGRES_PASSWORD" "$(generate_secret)"
  fi

  if ! grep -q '^JWT_SECRET=' "${target_file}" || [[ -z "$(awk -F= '/^JWT_SECRET=/{print $2}' "${target_file}" | tail -n1)" ]]; then
    set_env_value "${target_file}" "JWT_SECRET" "$(generate_secret)"
  fi

  if ! grep -q '^TOTP_ENCRYPTION_KEY=' "${target_file}" || [[ -z "$(awk -F= '/^TOTP_ENCRYPTION_KEY=/{print $2}' "${target_file}" | tail -n1)" ]]; then
    set_env_value "${target_file}" "TOTP_ENCRYPTION_KEY" "$(generate_secret)"
  fi

  set_env_value "${target_file}" "SERVER_PORT" "${port}"
}

get_env_value() {
  local file="$1"
  local key="$2"
  awk -F= -v key="${key}" '$1 == key {print substr($0, index($0, "=") + 1)}' "${file}" | tail -n1
}

require_env_keys() {
  local file="$1"
  shift
  local missing=()
  local key value
  for key in "$@"; do
    value="$(get_env_value "${file}" "${key}")"
    if [[ -z "${value}" ]]; then
      missing+=("${key}")
    fi
  done

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "以下配置在 ${file} 中缺失，无法使用外部服务模式:" >&2
    printf '  - %s\n' "${missing[@]}" >&2
    exit 1
  fi
}

ssh_base() {
  if [[ -n "${ASKPASS_FILE}" ]]; then
    DISPLAY=:999 SSH_ASKPASS="${ASKPASS_FILE}" SSH_ASKPASS_REQUIRE=force \
      setsid ssh -o ConnectTimeout="${SSH_CONNECT_TIMEOUT}" -o NumberOfPasswordPrompts=1 "$@"
  else
    ssh -o ConnectTimeout="${SSH_CONNECT_TIMEOUT}" "$@"
  fi
}

run_remote() {
  local cmd="$1"
  local i
  for i in $(seq 1 "${SSH_RETRY_COUNT}"); do
    if ssh_base "${REMOTE_HOST}" "${cmd}" < /dev/null; then
      return 0
    fi
    sleep 2
  done
  return 1
}

upload_remote_file() {
  local src="$1"
  local dst="$2"
  local i
  for i in $(seq 1 "${SSH_RETRY_COUNT}"); do
    if cat "${src}" | ssh_base "${REMOTE_HOST}" "cat > '${dst}'"; then
      return 0
    fi
    sleep 2
  done
  return 1
}

REMOTE_HOST=""
NODE_RAW=""
IMAGE_REF="sub2api:latest"
LOCAL_ENV_FILE="${DEPLOY_DIR}/.env"
REMOTE_BASE="/root/sub2api"
REMOTE_PORT=""
AUTO_BUILD="false"
SYNC_ENV="false"
PULL_NEVER="false"
USE_EXTERNAL_DB="false"
USE_EXTERNAL_REDIS="false"
EXTERNAL_NETWORKS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --host)
      REMOTE_HOST="$2"
      shift 2
      ;;
    --node)
      NODE_RAW="$2"
      shift 2
      ;;
    --image)
      IMAGE_REF="$2"
      shift 2
      ;;
    --env-file)
      LOCAL_ENV_FILE="$2"
      shift 2
      ;;
    --remote-base)
      REMOTE_BASE="$2"
      shift 2
      ;;
    --port)
      REMOTE_PORT="$2"
      shift 2
      ;;
    --external-services)
      USE_EXTERNAL_DB="true"
      USE_EXTERNAL_REDIS="true"
      shift
      ;;
    --external-db)
      USE_EXTERNAL_DB="true"
      shift
      ;;
    --external-redis)
      USE_EXTERNAL_REDIS="true"
      shift
      ;;
    --external-network)
      EXTERNAL_NETWORKS+=("$2")
      shift 2
      ;;
    --build)
      AUTO_BUILD="true"
      shift
      ;;
    --sync-env)
      SYNC_ENV="true"
      shift
      ;;
    --pull-never)
      PULL_NEVER="true"
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

if [[ -z "${REMOTE_HOST}" || -z "${NODE_RAW}" ]]; then
  usage
  exit 1
fi

require_cmd docker
require_cmd ssh
require_cmd openssl

setup_ssh_auth
trap cleanup_ssh_auth EXIT

NODE_NAME="$(normalize_node "${NODE_RAW}")"
REMOTE_DIR="${REMOTE_BASE}/${NODE_NAME}"
REMOTE_PROJECT="sub2api-${NODE_NAME}"

if [[ -z "${REMOTE_PORT}" ]]; then
  REMOTE_PORT="$(default_port_for_node "${NODE_NAME}")"
fi

ensure_local_image "${IMAGE_REF}" "${AUTO_BUILD}"

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT

TEMP_ENV_FILE="${TEMP_DIR}/.env"
TEMP_OVERRIDE_FILE="${TEMP_DIR}/docker-compose.override.yml"

if [[ -f "${LOCAL_ENV_FILE}" ]]; then
  prepare_temp_env "${LOCAL_ENV_FILE}" "${REMOTE_PORT}" "${TEMP_ENV_FILE}"
else
  prepare_temp_env "${ENV_EXAMPLE}" "${REMOTE_PORT}" "${TEMP_ENV_FILE}"
fi

if [[ "${USE_EXTERNAL_DB}" == "true" ]]; then
  require_env_keys "${TEMP_ENV_FILE}" \
    DATABASE_HOST DATABASE_PORT DATABASE_USER DATABASE_PASSWORD DATABASE_DBNAME
fi

if [[ "${USE_EXTERNAL_REDIS}" == "true" ]]; then
  require_env_keys "${TEMP_ENV_FILE}" \
    REDIS_HOST REDIS_PORT
fi

SUB2API_ENV_LINES=""

if [[ "${USE_EXTERNAL_DB}" == "true" ]]; then
  SUB2API_ENV_LINES="${SUB2API_ENV_LINES}
      DATABASE_HOST: \${DATABASE_HOST}
      DATABASE_PORT: \${DATABASE_PORT}
      DATABASE_USER: \${DATABASE_USER}
      DATABASE_PASSWORD: \${DATABASE_PASSWORD}
      DATABASE_DBNAME: \${DATABASE_DBNAME}
      DATABASE_SSLMODE: \${DATABASE_SSLMODE:-disable}
      DATABASE_MAX_OPEN_CONNS: \${DATABASE_MAX_OPEN_CONNS:-50}
      DATABASE_MAX_IDLE_CONNS: \${DATABASE_MAX_IDLE_CONNS:-10}
      DATABASE_CONN_MAX_LIFETIME_MINUTES: \${DATABASE_CONN_MAX_LIFETIME_MINUTES:-30}
      DATABASE_CONN_MAX_IDLE_TIME_MINUTES: \${DATABASE_CONN_MAX_IDLE_TIME_MINUTES:-5}"
fi

if [[ "${USE_EXTERNAL_REDIS}" == "true" ]]; then
  SUB2API_ENV_LINES="${SUB2API_ENV_LINES}
      REDIS_HOST: \${REDIS_HOST}
      REDIS_PORT: \${REDIS_PORT}
      REDIS_PASSWORD: \${REDIS_PASSWORD:-}
      REDIS_DB: \${REDIS_DB:-0}
      REDIS_POOL_SIZE: \${REDIS_POOL_SIZE:-1024}
      REDIS_MIN_IDLE_CONNS: \${REDIS_MIN_IDLE_CONNS:-10}
      REDIS_ENABLE_TLS: \${REDIS_ENABLE_TLS:-false}"
fi

SUB2API_ENV_BLOCK=""
if [[ -n "${SUB2API_ENV_LINES}" ]]; then
  SUB2API_ENV_BLOCK="
    environment:${SUB2API_ENV_LINES}"
fi

SUB2API_NETWORK_BLOCK=""
EXTERNAL_NETWORK_DECLS=""
NETWORKS_BLOCK=""
if [[ ${#EXTERNAL_NETWORKS[@]} -gt 0 ]]; then
  SUB2API_NETWORK_BLOCK="
    networks:
      - default"
  for net in "${EXTERNAL_NETWORKS[@]}"; do
    SUB2API_NETWORK_BLOCK="${SUB2API_NETWORK_BLOCK}
      - ${net}"
    EXTERNAL_NETWORK_DECLS="${EXTERNAL_NETWORK_DECLS}
  ${net}:
    external: true"
  done
  NETWORKS_BLOCK="
networks:${EXTERNAL_NETWORK_DECLS}"
fi

cat > "${TEMP_OVERRIDE_FILE}" <<EOF
services:
  sub2api:
    image: ${IMAGE_REF}
    container_name: ${NODE_NAME}-sub2api${SUB2API_ENV_BLOCK}${SUB2API_NETWORK_BLOCK}
${NETWORKS_BLOCK}
EOF

echo "==> 准备远程目录: ${REMOTE_HOST}:${REMOTE_DIR}"
run_remote "mkdir -p '${REMOTE_DIR}'"

echo "==> 上传 compose 文件"
upload_remote_file "${COMPOSE_SOURCE}" "${REMOTE_DIR}/docker-compose.local.yml"
upload_remote_file "${TEMP_OVERRIDE_FILE}" "${REMOTE_DIR}/docker-compose.override.yml"

if [[ "${SYNC_ENV}" == "true" ]]; then
  echo "==> 强制同步远程 .env"
  upload_remote_file "${TEMP_ENV_FILE}" "${REMOTE_DIR}/.env"
else
  echo "==> 检查远程 .env"
  if run_remote "[ -f '${REMOTE_DIR}/.env' ]"; then
    echo "远程 .env 已存在，保留原配置"
  else
    echo "远程 .env 不存在，上传初始化版本"
    upload_remote_file "${TEMP_ENV_FILE}" "${REMOTE_DIR}/.env"
  fi
fi

echo "==> 传输镜像: ${IMAGE_REF}"
docker save "${IMAGE_REF}" | gzip | ssh_base "${REMOTE_HOST}" "gunzip | docker load"

PULL_ARG=""
if [[ "${PULL_NEVER}" == "true" ]]; then
  PULL_ARG="--pull never"
fi

echo "==> 启动 ${NODE_NAME}"
COMPOSE_UP_ARGS=(up -d)
if [[ "${PULL_NEVER}" == "true" ]]; then
  COMPOSE_UP_ARGS+=(--pull never)
fi
if [[ "${USE_EXTERNAL_DB}" == "true" || "${USE_EXTERNAL_REDIS}" == "true" ]]; then
  COMPOSE_UP_ARGS+=(--no-deps sub2api)
fi

run_remote "cd '${REMOTE_DIR}' && mkdir -p data postgres_data redis_data && docker compose --project-name '${REMOTE_PROJECT}' -f docker-compose.local.yml -f docker-compose.override.yml ${COMPOSE_UP_ARGS[*]} && docker compose --project-name '${REMOTE_PROJECT}' -f docker-compose.local.yml -f docker-compose.override.yml ps"

echo
echo "部署完成"
echo "  节点目录: ${REMOTE_DIR}"
echo "  节点名称: ${NODE_NAME}"
echo "  访问地址: http://${REMOTE_HOST#*@}:${REMOTE_PORT}"
echo "  查看日志:"
echo "    ssh ${REMOTE_HOST} \"cd ${REMOTE_DIR} && docker compose --project-name ${REMOTE_PROJECT} -f docker-compose.local.yml -f docker-compose.override.yml logs -f sub2api\""
