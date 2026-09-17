#!/usr/bin/env bash
set -u -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

vector_version_is_compatible() {
  local version="${1:-}"
  local major minor

  [[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || return 1
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"

  (( major == 0 && minor >= 8 ))
}

check_command() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf '%s: OK\n' "$label"
  else
    printf '%s: FAIL\n' "$label" >&2
    CHECK_FAILURES=$((CHECK_FAILURES + 1))
  fi
}

check_nonempty_command() {
  local label="$1"
  local output
  shift
  if output="$("$@" 2>/dev/null)" && [[ -n "$output" ]]; then
    printf '%s: OK\n' "$label"
  else
    printf '%s: FAIL\n' "$label" >&2
    CHECK_FAILURES=$((CHECK_FAILURES + 1))
  fi
}

check_vector_version() {
  local version
  if version="$(compose_cmd exec -T db psql -v ON_ERROR_STOP=1 -At -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT extversion FROM pg_extension WHERE extname = 'vector';")" && vector_version_is_compatible "$version"; then
    printf 'Vector extension version: OK\n'
  else
    printf 'Vector extension version: FAIL\n' >&2
    CHECK_FAILURES=$((CHECK_FAILURES + 1))
  fi
}

main() {
  local ENV_FILE="${1:-}"
  local PROJECT_NAME="${2:-}"
  CHECK_FAILURES=0

  [[ -n "$ENV_FILE" && -n "$PROJECT_NAME" ]] || {
    printf 'usage: %s <env-file> <project-name>\n' "$0" >&2
    return 2
  }
  ENV_FILE="$ENV_FILE" validate_config
  compose_init "$ENV_FILE" "$PROJECT_NAME"

  check_nonempty_command 'Database container' compose_cmd ps -q db
  check_nonempty_command 'Auth container' compose_cmd ps -q auth
  check_command 'PostgreSQL readiness' compose_cmd exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null'
  check_command 'Auth health' compose_cmd exec -T auth wget -q --spider http://127.0.0.1:9999/health
  check_command 'Infrastructure migration ledger' compose_cmd exec -T db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "DO \$\$ BEGIN IF to_regclass('infra.migration_ledger') IS NULL OR (SELECT count(*) FROM infra.migration_ledger) < 2 THEN RAISE EXCEPTION 'infra ledger incomplete'; END IF; END \$\$;"
  check_command 'GoTrue schema and migration marker' compose_cmd exec -T db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "DO \$\$ BEGIN IF to_regnamespace('auth') IS NULL OR to_regclass('auth.users') IS NULL OR to_regclass('auth.identities') IS NULL OR to_regclass('auth.schema_migrations') IS NULL OR (SELECT count(*) FROM auth.schema_migrations) = 0 THEN RAISE EXCEPTION 'GoTrue catalog incomplete'; END IF; END \$\$;"
  check_command 'Approved extensions' compose_cmd exec -T db psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'uuid-ossp') OR NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'citext') THEN RAISE EXCEPTION 'required extension is missing'; END IF; END \$\$;"
  check_vector_version

  if [[ "$CHECK_FAILURES" -ne 0 ]]; then
    printf 'Inspection failed with %s red check(s)\n' "$CHECK_FAILURES" >&2
    return 1
  fi
  printf 'Inspection completed with all independent checks green\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
