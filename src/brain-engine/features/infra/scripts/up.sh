#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-}"
PROJECT_NAME="${2:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

startup_fail() {
  printf 'Startup error: %s\n' "$1" >&2
  exit 1
}

wait_for_db() {
  local attempt
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if compose_cmd exec -T db pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  startup_fail 'PostgreSQL did not become ready'
}

wait_for_auth() {
  local attempt
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if compose_cmd exec -T auth wget -q --spider http://127.0.0.1:9999/health >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  startup_fail 'GoTrue did not become ready after its migration'
}

[[ -n "$ENV_FILE" && -n "$PROJECT_NAME" ]] || {
  printf 'usage: %s <env-file> <project-name>\n' "$0" >&2
  exit 2
}

ENV_FILE="$ENV_FILE" validate_config
compose_init "$ENV_FILE" "$PROJECT_NAME"
compose_cmd up -d db || startup_fail 'database Compose start failed'
wait_for_db
"${ROOT_DIR}/scripts/migrate.sh" "$ENV_FILE" "$PROJECT_NAME"
compose_cmd run --rm auth-migrate
compose_cmd up -d --no-deps auth
wait_for_auth
printf 'PostgreSQL, infrastructure migrations, GoTrue migrations and Auth are ready\n'
