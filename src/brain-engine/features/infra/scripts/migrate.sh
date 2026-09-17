#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-}"
PROJECT_NAME="${2:-}"
MIGRATIONS_DIR="${INFRA_MIGRATIONS_DIR:-${ROOT_DIR}/postgres/migrations}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

migration_fail() {
  printf 'Migration error: %s\n' "$1" >&2
  exit 1
}

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    migration_fail 'sha256 utility is required'
  fi
}

load_migrations() {
  local file base id seen_ids='|'
  MIGRATION_FILES=()
  MIGRATION_IDS=()
  MIGRATION_HASHES=()

  [[ -d "$MIGRATIONS_DIR" ]] || migration_fail 'migration directory is required'
  while IFS= read -r file; do
    base="$(basename "$file")"
    [[ "$base" =~ ^[0-9]{4}_[a-z0-9][a-z0-9_-]*\.sql$ ]] || migration_fail "non-canonical migration filename: $base"
    id="${base%%_*}"
    case "$seen_ids" in
      *"|${id}|"*)
        migration_fail "duplicate migration id: $id"
        ;;
    esac
    seen_ids="${seen_ids}${id}|"
    MIGRATION_FILES+=("$file")
    MIGRATION_IDS+=("$id")
    MIGRATION_HASHES+=("$(sha256_file "$file")")
  done < <(find "$MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.sql' -print | LC_ALL=C sort)

  [[ "${#MIGRATION_FILES[@]}" -gt 0 ]] || migration_fail 'no infrastructure migrations found'
}

db_psql() {
  compose_cmd exec -T db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
}

bootstrap_ledger() {
  db_psql <<'SQL'
CREATE SCHEMA IF NOT EXISTS infra;
CREATE TABLE IF NOT EXISTS infra.migration_ledger (
  id text PRIMARY KEY,
  sha256 text NOT NULL,
  applied_at timestamptz NOT NULL DEFAULT now(),
  CHECK (id ~ '^[0-9]{4}$'),
  CHECK (sha256 ~ '^[0-9a-f]{64}$')
);
SQL
}

read_ledger() {
  LEDGER_ROWS="$(db_psql -A -t -F '|' -c 'SELECT id, sha256 FROM infra.migration_ledger ORDER BY id;')"
}

read_existing_ledger() {
  local relation_state
  if LEDGER_ROWS="$(db_psql -A -t -F '|' -c 'SELECT id, sha256 FROM infra.migration_ledger ORDER BY id;' 2>/dev/null)"; then
    LEDGER_TABLE_EXISTS=1
    return 0
  fi

  if ! relation_state="$(db_psql -A -t -c "SELECT COALESCE(to_regnamespace('infra')::text, '') || '|' || COALESCE(to_regclass('infra.migration_ledger')::text, '');" 2>/dev/null)"; then
    migration_fail 'unable to inspect migration ledger'
  fi
  [[ "$relation_state" == '|' ]] || migration_fail 'migration ledger exists but could not be read'
  LEDGER_ROWS=''
  LEDGER_TABLE_EXISTS=0
}

ledger_hash_for() {
  local requested_id="$1"
  local row row_id row_hash
  LEDGER_HASH=''
  while IFS= read -r row; do
    [[ -n "$row" ]] || continue
    row_id="${row%%|*}"
    row_hash="${row#*|}"
    [[ "$row_id" =~ ^[0-9]{4}$ && "$row_hash" =~ ^[0-9a-f]{64}$ ]] || migration_fail 'migration ledger contains an invalid record'
    if [[ "$row_id" == "$requested_id" ]]; then
      [[ -z "$LEDGER_HASH" ]] || migration_fail "migration ledger has duplicate id: $requested_id"
      LEDGER_HASH="$row_hash"
    fi
  done <<< "$LEDGER_ROWS"
}

highest_ledger_id() {
  local row row_id row_hash
  HIGHEST_LEDGER_ID=''
  while IFS= read -r row; do
    [[ -n "$row" ]] || continue
    row_id="${row%%|*}"
    row_hash="${row#*|}"
    [[ "$row_id" =~ ^[0-9]{4}$ && "$row_hash" =~ ^[0-9a-f]{64}$ ]] || migration_fail 'migration ledger contains an invalid record'
    if [[ -z "$HIGHEST_LEDGER_ID" || "$row_id" > "$HIGHEST_LEDGER_ID" ]]; then
      HIGHEST_LEDGER_ID="$row_id"
    fi
  done <<< "$LEDGER_ROWS"
}

validate_ledger_before_ddl() {
  local row row_id row_hash index id hash found
  highest_ledger_id

  while IFS= read -r row; do
    [[ -n "$row" ]] || continue
    row_id="${row%%|*}"
    row_hash="${row#*|}"
    [[ "$row_id" =~ ^[0-9]{4}$ && "$row_hash" =~ ^[0-9a-f]{64}$ ]] || migration_fail 'migration ledger contains an invalid record'
    found=0
    for ((index = 0; index < ${#MIGRATION_FILES[@]}; index++)); do
      if [[ "${MIGRATION_IDS[$index]}" == "$row_id" ]]; then
        found=1
        [[ "$row_hash" == "${MIGRATION_HASHES[$index]}" ]] || migration_fail "applied migration hash drift: $row_id"
        break
      fi
    done
    if (( found == 0 )); then
      migration_fail "applied migration has no canonical file: $row_id"
    fi
  done <<< "$LEDGER_ROWS"

  for ((index = 0; index < ${#MIGRATION_FILES[@]}; index++)); do
    id="${MIGRATION_IDS[$index]}"
    hash="${MIGRATION_HASHES[$index]}"
    ledger_hash_for "$id"
    if [[ -n "$LEDGER_HASH" && "$LEDGER_HASH" != "$hash" ]]; then
      migration_fail "applied migration hash drift: $id"
    fi
    if [[ -z "$LEDGER_HASH" && -n "$HIGHEST_LEDGER_ID" && "$id" < "$HIGHEST_LEDGER_ID" ]]; then
      migration_fail "pending migration precedes the highest applied id: $id"
    fi
  done
}

apply_migration() {
  local file="$1"
  local id="$2"
  local hash="$3"
  {
    printf 'BEGIN;\n'
    cat "$file"
    printf "\nINSERT INTO infra.migration_ledger (id, sha256) VALUES ('%s', '%s');\n" "$id" "$hash"
    printf 'COMMIT;\n'
  } | db_psql
}

run_migrations() {
  local index id hash
  load_migrations
  ENV_FILE="$ENV_FILE" validate_config
  compose_init "$ENV_FILE" "$PROJECT_NAME"
  read_existing_ledger
  if (( LEDGER_TABLE_EXISTS == 0 )); then
    bootstrap_ledger
    read_ledger
  fi
  validate_ledger_before_ddl

  for ((index = 0; index < ${#MIGRATION_FILES[@]}; index++)); do
    id="${MIGRATION_IDS[$index]}"
    hash="${MIGRATION_HASHES[$index]}"
    ledger_hash_for "$id"
    if [[ -n "$LEDGER_HASH" ]]; then
      printf 'Migration %s already applied with the same hash\n' "$id"
    else
      apply_migration "${MIGRATION_FILES[$index]}" "$id" "$hash"
      printf 'Applied infrastructure migration %s\n' "$id"
    fi
  done
}

[[ -n "$ENV_FILE" && -n "$PROJECT_NAME" ]] || {
  printf 'usage: %s <env-file> <project-name>\n' "$0" >&2
  exit 2
}
run_migrations
