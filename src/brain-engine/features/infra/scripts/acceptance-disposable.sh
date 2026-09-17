#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-}"
RUN_ID="${2:-}"
ROLE="${3:-}"
ENV_FILE="${4:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

acceptance_fail() {
  printf 'Disposable acceptance preflight error: %s\n' "$1" >&2
  exit 1
}

expected_database_for_role() {
  case "$ROLE" in
    src)
      printf 'mf_src_%s' "$RUN_ID"
      ;;
    restore)
      printf 'mf_restore_%s' "$RUN_ID"
      ;;
    replay)
      printf 'mf_replay_%s' "$RUN_ID"
      ;;
    *)
      acceptance_fail 'role must be src, restore, or replay'
      ;;
  esac
}

[[ "$MODE" == '--local-disposable' || "$MODE" == '--vm-disposable' ]] || {
  printf 'usage: %s <--local-disposable|--vm-disposable> <run-id> <src|restore|replay> <env-file>\n' "$0" >&2
  exit 2
}
[[ "$RUN_ID" =~ ^[a-z0-9][a-z0-9-]{7,63}$ ]] || acceptance_fail 'run ID is invalid'
[[ -f "$ENV_FILE" ]] || acceptance_fail 'environment file is required'
[[ "$ENV_FILE" != "${ROOT_DIR}/.env" ]] || acceptance_fail 'the primary .env is never an acceptance environment'

if [[ "$MODE" == '--vm-disposable' && "${MF_LOCAL_ACCEPTANCE_GREEN:-}" != '1' ]]; then
  acceptance_fail 'VM mode is blocked until the local disposable matrix is green'
fi

ENV_FILE="$ENV_FILE" validate_config
PROJECT_NAME="mf-accept-${RUN_ID}-${ROLE}"
EXPECTED_DATABASE="$(expected_database_for_role)"
EXPECTED_VOLUME="${PROJECT_NAME}_mindforge-postgres-data"

[[ "${MF_ENV_CLASS:-}" == 'disposable' ]] || acceptance_fail 'environment must declare MF_ENV_CLASS=disposable'
[[ "${DB_BIND_ADDR:-}" == '127.0.0.1' && "${AUTH_BIND_ADDR:-}" == '127.0.0.1' ]] || acceptance_fail 'acceptance binds must remain loopback-only'
[[ "${POSTGRES_PORT:-}" != '5432' && "${AUTH_PORT:-}" != '9999' ]] || acceptance_fail 'acceptance must use dynamic non-primary host ports'
[[ "${MF_DISPOSABLE_TARGET_ID:-}" == "$RUN_ID" ]] || acceptance_fail 'disposable target ID diverges from run ID'
[[ "${MF_DISPOSABLE_CONFIRMATION:-}" == "RESTORE:${RUN_ID}" ]] || acceptance_fail 'disposable confirmation is not target-bound'
[[ "${MF_EXPECTED_VOLUME:-}" == "$EXPECTED_VOLUME" ]] || acceptance_fail 'disposable volume name is not project-exclusive'
[[ "${POSTGRES_DB:-}" == "$EXPECTED_DATABASE" ]] || acceptance_fail 'database name is not role-exclusive'
[[ "${GOTRUE_MAILER_AUTOCONFIRM:-}" == 'true' ]] || acceptance_fail 'disposable Auth flow requires auto-confirmed synthetic identities'

command -v docker >/dev/null 2>&1 || acceptance_fail 'docker is required after disposable preflight validation'
compose_init "$ENV_FILE" "$PROJECT_NAME"
if docker volume inspect "$EXPECTED_VOLUME" >/dev/null 2>&1; then
  acceptance_fail 'exclusive acceptance volume already exists'
fi
if compose_cmd ps -aq | grep -q .; then
  acceptance_fail 'exclusive acceptance project already has containers'
fi
if compose_cmd ps -q auth | grep -q .; then
  acceptance_fail 'Auth must not already be running in an acceptance project'
fi

printf 'Disposable preflight passed for project %s and volume %s\n' "$PROJECT_NAME" "$EXPECTED_VOLUME"
