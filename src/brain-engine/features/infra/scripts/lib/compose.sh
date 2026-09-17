#!/usr/bin/env bash
set -euo pipefail

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INFRA_ROOT="$(cd "${LIB_DIR}/../.." && pwd)"
VALIDATE_CONFIG="${INFRA_ROOT}/scripts/validate-config.sh"
INFRA_COMPOSE_ENV_FILE=''
INFRA_COMPOSE_PROJECT=''

compose_fail() {
  printf 'Compose setup error: %s\n' "$1" >&2
  return 1
}

compose_init() {
  INFRA_COMPOSE_ENV_FILE="${1:-}"
  INFRA_COMPOSE_PROJECT="${2:-}"
  [[ -n "$INFRA_COMPOSE_ENV_FILE" && -f "$INFRA_COMPOSE_ENV_FILE" ]] || compose_fail 'an explicit environment file is required'
  [[ "$INFRA_COMPOSE_PROJECT" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || compose_fail 'an explicit lowercase project name is required'
  bash "$VALIDATE_CONFIG" "$INFRA_COMPOSE_ENV_FILE"
}

compose_cmd() {
  if docker compose version >/dev/null 2>&1; then
    docker compose --project-name "$INFRA_COMPOSE_PROJECT" --env-file "$INFRA_COMPOSE_ENV_FILE" -f "${INFRA_ROOT}/compose.yaml" "$@"
  elif command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1; then
    docker-compose --project-name "$INFRA_COMPOSE_PROJECT" --env-file "$INFRA_COMPOSE_ENV_FILE" -f "${INFRA_ROOT}/compose.yaml" "$@"
  else
    compose_fail 'Docker Compose v2 is required'
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [[ "$#" -ge 3 ]] || {
    printf 'usage: %s <env-file> <project-name> <docker-compose arguments...>\n' "$0" >&2
    exit 2
  }
  env_file="$1"
  project_name="$2"
  shift 2
  compose_init "$env_file" "$project_name"
  compose_cmd "$@"
fi
