#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"
ENV_FILE="${1:-}"
PROJECT_NAME="${2:-}"
OPERATOR_ID="${3:-}"
AUTHORIZATION_REF="${4:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

startup_fail() {
  printf 'Startup error: %s\n' "$1" >&2
  exit 1
}

report_main_auth_health() {
  local container_id
  if ! container_id="$(compose_cmd ps -q auth 2>/dev/null)"; then
    printf 'Main Auth health: unknown (presence check failed)\n'
  elif [[ -z "$container_id" ]]; then
    printf 'Main Auth health: not running\n'
  elif compose_cmd exec -T auth wget -q --spider http://127.0.0.1:9999/health >/dev/null 2>&1; then
    printf 'Main Auth health: healthy\n'
  else
    printf 'Main Auth health: unhealthy\n'
  fi
}

startup_incomplete() {
  printf 'Startup incomplete: %s\n' "$1" >&2
  report_main_auth_health
  exit 1
}

read_only_preflight() {
  local target="$1"
  local allow_bootstrap="${2:-no}"
  local tool output reference
  tool="$(infra_tool)" || return 1
  local -a arguments=(
    preflight
    --config "$ENV_FILE"
    --source-root "$REPOSITORY_ROOT"
    --target "$target"
    --operator-id "$OPERATOR_ID"
    --authorization-ref "$AUTHORIZATION_REF"
    --scope infrastructure
  )
  if [[ "$allow_bootstrap" == 'yes' ]]; then
    arguments+=(--allow-infrastructure-bootstrap)
  fi
  output="$("$tool" "${arguments[@]}")" || return 1
  reference="$(printf '%s\n' "$output" | sed -n 's/^preflight-reference=//p')"
  [[ -n "$reference" ]]
}

wait_for_db() {
  local attempt
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if compose_cmd exec -T db pg_isready -U "$POSTGRES_ADMIN_USER" -d postgres >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  startup_fail 'PostgreSQL did not become ready'
}

wait_for_auth() {
  local service="$1"
  local attempt
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if compose_cmd exec -T "$service" wget -q --spider http://127.0.0.1:9999/health >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  startup_fail "${service} did not become ready after its migration"
}

run_target_infrastructure() {
  local target="$1"
  local tool preflight_output preflight_ref verification_ref
  tool="$(infra_tool)"
  verification_ref="startup/${PROJECT_NAME}/${target}"
  preflight_output="$({
    "$tool" preflight \
      --config "$ENV_FILE" \
      --source-root "$REPOSITORY_ROOT" \
      --target "$target" \
      --operator-id "$OPERATOR_ID" \
      --authorization-ref "$AUTHORIZATION_REF" \
      --scope infrastructure \
      --allow-infrastructure-bootstrap
  })" || startup_fail "${target} read-only preflight failed"
  preflight_ref="$(printf '%s\n' "$preflight_output" | sed -n 's/^preflight-reference=//p')"
  [[ -n "$preflight_ref" ]] || startup_fail "${target} preflight did not return a reference"
  "$tool" apply \
    --config "$ENV_FILE" \
    --source-root "$REPOSITORY_ROOT" \
    --target "$target" \
    --operator-id "$OPERATOR_ID" \
    --authorization-ref "$AUTHORIZATION_REF" \
    --scope infrastructure \
    --preflight-ref "$preflight_ref" \
    --verification-ref "$verification_ref" || startup_fail "${target} infrastructure application failed"
}

[[ -n "$ENV_FILE" && -n "$PROJECT_NAME" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" ]] || {
  printf 'usage: %s <env-file> <project-name> <operator-id> <authorization-ref>\n' "$0" >&2
  exit 2
}

ENV_FILE="$ENV_FILE" validate_config
compose_init "$ENV_FILE" "$PROJECT_NAME"

# Establish whether this is an existing initialized stack before any Compose
# lifecycle mutation. A stopped or inconsistent existing cluster stays put.
db_containers="$(compose_cmd ps -aq db)" || startup_incomplete 'cluster topology is unknown; no lifecycle changes were made'
db_running="$(compose_cmd ps -q db)" || startup_incomplete 'cluster topology is unknown; no lifecycle changes were made'
main_auth_running="$(compose_cmd ps -q auth)" || startup_incomplete 'cluster topology is unknown; no lifecycle changes were made'
volume_name="${PROJECT_NAME}_mindforge-postgres-data"
if docker volume inspect "$volume_name" >/dev/null 2>&1; then
  volume_exists='yes'
else
  volume_exists='no'
fi

if [[ "$volume_exists" == 'yes' && -n "$db_containers" && -n "$db_running" ]]; then
  # Existing volume and running PostgreSQL: topology must already be complete
  # for both targets before any migration or Auth lifecycle can run.
  read_only_preflight mindforge_develop || startup_incomplete 'develop target is absent or incomplete'
  read_only_preflight mindforge || startup_incomplete 'main target topology could not be verified'
elif [[ "$volume_exists" == 'no' && -z "$db_containers" && -z "$db_running" && -z "$main_auth_running" ]]; then
  # Fresh cluster only: initdb creates both contracted databases on the new
  # project volume. This branch never provisions an initialized volume.
  compose_cmd up -d db || startup_fail 'fresh-cluster database Compose start failed'
  wait_for_db
  read_only_preflight mindforge_develop yes || startup_fail 'fresh develop bootstrap preflight failed'
  read_only_preflight mindforge yes || startup_fail 'fresh main bootstrap preflight failed'
else
  startup_incomplete 'existing cluster topology is unknown or inconsistent; no lifecycle changes were made'
fi

if [[ "$volume_exists" == 'yes' ]]; then
  wait_for_db
fi

run_target_infrastructure mindforge_develop
compose_cmd run --rm auth-develop-migrate || startup_fail 'develop Auth migration failed'
compose_cmd up -d --no-deps auth-develop || startup_fail 'develop Auth startup failed'
wait_for_auth auth-develop

run_target_infrastructure mindforge
compose_cmd run --rm auth-migrate || startup_fail 'main Auth migration failed'
compose_cmd up -d --no-deps auth || startup_fail 'main Auth startup failed'
wait_for_auth auth

printf 'PostgreSQL, independent infrastructure migrations, GoTrue migrations and Auth pairs are ready\n'
