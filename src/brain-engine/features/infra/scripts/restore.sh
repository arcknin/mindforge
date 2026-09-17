#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_FILE="${1:-}"
ENV_FILE="${2:-}"
PROJECT_NAME="${3:-}"
TARGET_ID="${4:-}"
CONFIRMATION="${5:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

restore_fail() {
  printf 'Restore error: %s\n' "$1" >&2
  exit 1
}

wait_for_db() {
  local attempt
  for ((attempt = 1; attempt <= 30; attempt++)); do
    if compose_cmd exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null'; then
      return 0
    fi
    sleep 1
  done
  restore_fail 'PostgreSQL did not become ready in the disposable target'
}

require_disposable_target() {
  local expected_project expected_volume expected_database
  [[ "$TARGET_ID" =~ ^[a-z0-9][a-z0-9-]{7,63}$ ]] || restore_fail 'target ID is invalid'
  expected_project="mf-accept-${TARGET_ID}-restore"
  expected_volume="${expected_project}_mindforge-postgres-data"
  expected_database="mf_restore_${TARGET_ID//-/_}"
  [[ "$PROJECT_NAME" == "$expected_project" ]] || restore_fail 'project name is not the exclusive restore project for this target'
  [[ "${MF_DISPOSABLE_TARGET_ID:-}" == "$TARGET_ID" ]] || restore_fail 'target ID does not match the disposable environment'
  [[ "${MF_DISPOSABLE_CONFIRMATION:-}" == "RESTORE:${TARGET_ID}" ]] || restore_fail 'environment confirmation is not target-bound'
  [[ "$CONFIRMATION" == "$MF_DISPOSABLE_CONFIRMATION" ]] || restore_fail 'confirmation must exactly match the disposable environment'
  [[ "${MF_EXPECTED_VOLUME:-}" == "$expected_volume" ]] || restore_fail 'environment volume is not the exclusive restore volume'
  [[ "${POSTGRES_DB:-}" == "$expected_database" ]] || restore_fail 'restore database is not the exclusive target database'
}

[[ -n "$BACKUP_FILE" && -n "$ENV_FILE" && -n "$PROJECT_NAME" && -n "$TARGET_ID" && -n "$CONFIRMATION" ]] || {
  printf 'usage: %s <backup-file> <env-file> <project-name> <target-id> <confirmation>\n' "$0" >&2
  exit 2
}
[[ -f "$BACKUP_FILE" ]] || restore_fail 'backup file is required'
[[ -f "$ENV_FILE" ]] || restore_fail 'environment file is required'
ENV_FILE="$ENV_FILE" validate_config
require_disposable_target
command -v pg_restore >/dev/null 2>&1 || restore_fail 'pg_restore is required for a read-only dump preflight'
pg_restore --list "$BACKUP_FILE" >/dev/null 2>&1 || restore_fail 'backup preflight failed before Docker mutation'

compose_init "$ENV_FILE" "$PROJECT_NAME"
if docker volume inspect "$MF_EXPECTED_VOLUME" >/dev/null 2>&1; then
  restore_fail 'exclusive restore volume already exists'
fi
if compose_cmd ps -aq | grep -q .; then
  restore_fail 'exclusive restore project already has containers'
fi
if compose_cmd ps -q auth | grep -q .; then
  restore_fail 'Auth must remain stopped during restore'
fi

compose_cmd up -d db
wait_for_db
if ! compose_cmd exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" exec pg_restore --single-transaction --exit-on-error --no-owner --no-acl -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < "$BACKUP_FILE"; then
  restore_fail 'pg_restore failed; the target transaction was not promoted'
fi
printf 'Restore completed only in exclusive disposable target %s\n' "$TARGET_ID"
