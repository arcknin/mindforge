#!/usr/bin/env bash
# Compose lifecycle is reserved for the owner-designated VM stack. Migration,
# backup and restore adapters call the distributable binary directly.
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

compose_validate_images() {
  local compose_file="${1:-${INFRA_ROOT}/compose.yaml}"
  local image
  local image_count=0

  [[ -f "$compose_file" ]] || compose_fail 'Compose definition is missing'
  while IFS= read -r image; do
    [[ -n "$image" ]] || continue
    image_count=$((image_count + 1))
    case "$image" in
      pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b|supabase/gotrue:v2.196.0)
        ;;
      *)
        compose_fail "unsupported or mutable image: $image"
        ;;
    esac
  done < <(awk '/^[[:space:]]+image:[[:space:]]+/ { print $2 }' "$compose_file")

  (( image_count > 0 )) || compose_fail 'Compose definition has no images'
}

compose_init() {
  INFRA_COMPOSE_ENV_FILE="${1:-}"
  INFRA_COMPOSE_PROJECT="${2:-}"
  [[ -n "$INFRA_COMPOSE_ENV_FILE" && -f "$INFRA_COMPOSE_ENV_FILE" ]] || compose_fail 'an explicit environment file is required'
  [[ "$INFRA_COMPOSE_PROJECT" =~ ^[a-z0-9][a-z0-9_-]*$ ]] || compose_fail 'an explicit lowercase project name is required'
  bash "$VALIDATE_CONFIG" "$INFRA_COMPOSE_ENV_FILE"
  compose_validate_images
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

infra_tool() {
  local tool="${MINDFORGE_INFRA_BIN:-${INFRA_ROOT}/tools/mindforge-infra}"
  [[ -x "$tool" ]] || compose_fail 'the built mindforge-infra binary is required'
  printf '%s\n' "$tool"
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
