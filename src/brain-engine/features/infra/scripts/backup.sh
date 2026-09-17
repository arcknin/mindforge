#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-}"
PROJECT_NAME="${2:-}"
BACKUP_DIR="${3:-}"
TEMP_FILE=''

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

backup_fail() {
  printf 'Backup error: %s\n' "$1" >&2
  exit 1
}

cleanup_backup_temp() {
  if [[ -n "$TEMP_FILE" && -e "$TEMP_FILE" ]]; then
    rm -f "$TEMP_FILE"
  fi
}

require_external_backup_directory() {
  local canonical_dir
  [[ -n "$BACKUP_DIR" && -d "$BACKUP_DIR" ]] || backup_fail 'an existing explicit backup directory is required'
  canonical_dir="$(cd "$BACKUP_DIR" && pwd -P)"
  case "$canonical_dir" in
    "$ROOT_DIR"|"$ROOT_DIR"/*)
      backup_fail 'backup directory must be outside the checkout'
      ;;
  esac
  BACKUP_DIR="$canonical_dir"
}

require_pg_restore() {
  command -v pg_restore >/dev/null 2>&1 || backup_fail 'pg_restore is required to validate backup archives'
}

wait_for_db() {
  compose_cmd ps -q db | grep -q . || backup_fail 'db is not running'
  compose_cmd exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null' || backup_fail 'PostgreSQL readiness failed'
}

[[ -n "$ENV_FILE" && -n "$PROJECT_NAME" && -n "$BACKUP_DIR" ]] || {
  printf 'usage: %s <env-file> <project-name> <external-backup-dir>\n' "$0" >&2
  exit 2
}

umask 077
trap cleanup_backup_temp EXIT HUP INT TERM
require_external_backup_directory
require_pg_restore
ENV_FILE="$ENV_FILE" validate_config
compose_init "$ENV_FILE" "$PROJECT_NAME"
wait_for_db

TEMP_FILE="$(mktemp "${BACKUP_DIR}/.mindforge-infra.XXXXXX")"
if ! compose_cmd exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" exec pg_dump --format=custom --no-owner --no-acl -U "$POSTGRES_USER" -d "$POSTGRES_DB"' > "$TEMP_FILE"; then
  backup_fail 'pg_dump failed; no backup artifact was promoted'
fi
[[ -s "$TEMP_FILE" ]] || backup_fail 'pg_dump produced an empty artifact'
if ! pg_restore --list "$TEMP_FILE" >/dev/null 2>&1; then
  backup_fail 'backup archive preflight failed; no backup artifact was promoted'
fi

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
suffix="${TEMP_FILE##*.}"
FINAL_FILE="${BACKUP_DIR}/mindforge-infra-${timestamp}-${suffix}.dump"
[[ ! -e "$FINAL_FILE" ]] || backup_fail 'refusing to overwrite an existing backup'
mv "$TEMP_FILE" "$FINAL_FILE"
TEMP_FILE=''
chmod 600 "$FINAL_FILE"
printf 'Backup completed: %s\n' "$FINAL_FILE"
