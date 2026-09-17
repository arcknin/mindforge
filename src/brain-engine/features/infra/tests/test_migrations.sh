#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fixture.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fake-docker.sh"

COMPOSE_FILE="${ROOT_DIR}/compose.yaml"
MIGRATE_SCRIPT="${ROOT_DIR}/scripts/migrate.sh"
UP_SCRIPT="${ROOT_DIR}/scripts/up.sh"
MIGRATIONS_DIR="${ROOT_DIR}/postgres/migrations"
TEST_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-migrations.XXXXXX")"
trap 'rm -rf "$TEST_TEMP"' EXIT

sha256_file() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
  else
    sha256sum "$1" | awk '{print $1}'
  fi
}

make_migrations() {
  local directory="$1"
  mkdir -p "$directory"
  cat > "${directory}/0001_infra_init.sql" <<'EOF'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
EOF
  cat > "${directory}/0002_auth_bootstrap.sql" <<'EOF'
DO $$ BEGIN RAISE NOTICE 'auth bootstrap is owned by gotrue migrate'; END $$;
EOF
}

make_orphan_ledger_fixture() {
  local directory="$1"
  local ledger_file="$2"
  local id file hash
  mkdir -p "$directory"
  : > "$ledger_file"
  for id in 0001 0002 0003 0004; do
    file="${directory}/${id}_migration_${id}.sql"
    printf 'CREATE TABLE IF NOT EXISTS infra.test_migration_%s (id integer);\n' "$id" > "$file"
    hash="$(sha256_file "$file")"
    printf '%s|%s\n' "$id" "$hash" >> "$ledger_file"
  done
  printf '0005|0000000000000000000000000000000000000000000000000000000000000000\n' >> "$ledger_file"
}

run_migrate() {
  local migrations_dir="$1"
  local ledger_file="$2"
  local fail_match="${3:-}"
  local env_file="${TEST_TEMP}/runner.env"
  local fake_bin="${TEST_TEMP}/bin"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/docker.log"
  : > "${TEST_TEMP}/docker.stdin"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" FAKE_DOCKER_STDIN_LOG="${TEST_TEMP}/docker.stdin" FAKE_DOCKER_LEDGER="$ledger_file" FAKE_DOCKER_FAIL_MATCH="$fail_match" INFRA_MIGRATIONS_DIR="$migrations_dir" bash "$MIGRATE_SCRIPT" "$env_file" mf-migration-test > "${TEST_TEMP}/runner.output" 2>&1
  MIGRATE_STATUS=$?
  set -e
}

test_migration_surface_is_canonical() {
  local compose
  compose="$(cat "$COMPOSE_FILE")"
  [[ -f "${MIGRATIONS_DIR}/0001_infra_init.sql" ]]
  assert_success "$?" "initial infrastructure migration has a zero-padded numeric ID"
  [[ ! -f "${MIGRATIONS_DIR}/2026xx01_infra_init.sql" ]]
  assert_success "$?" "non-canonical placeholder migration is absent"
  [[ "$compose" != *'/docker-entrypoint-initdb.d'* ]]
  assert_success "$?" "Compose does not use initdb as the future migration executor"
}

test_auth_bootstrap_precedes_auth_service() {
  local compose
  compose="$(cat "$COMPOSE_FILE")"
  [[ "$compose" == *$'  auth-migrate:'* && "$compose" == *'command: gotrue migrate'* ]]
  assert_success "$?" "pinned GoTrue migration service is declared"
  [[ "$compose" == *'condition: service_completed_successfully'* ]]
  assert_success "$?" "Auth waits for successful GoTrue migrations"
  [[ "$compose" == *'command: gotrue serve'* ]]
  assert_success "$?" "Auth serve command remains explicit after bootstrap"
}

test_auth_schema_precondition_exists_before_gotrue_migrate() {
  local auth_precondition
  auth_precondition="${MIGRATIONS_DIR}/0003_auth_schema.sql"
  [[ -f "$auth_precondition" ]]
  assert_success "$?" "infrastructure creates the Auth schema before GoTrue migrates"
  if [[ -f "$auth_precondition" ]]; then
    assert_contains "$(cat "$auth_precondition")" 'CREATE SCHEMA IF NOT EXISTS auth;' "Auth schema precondition is explicit and idempotent"
  fi
}

test_auth_postgres_role_precondition_exists_before_gotrue_migrate() {
  local auth_precondition
  auth_precondition="${MIGRATIONS_DIR}/0004_auth_postgres_role.sql"
  [[ -f "$auth_precondition" ]]
  assert_success "$?" "infrastructure provides the PostgreSQL role required by the GoTrue catalog"
  if [[ -f "$auth_precondition" ]]; then
    assert_contains "$(cat "$auth_precondition")" 'CREATE ROLE postgres NOLOGIN;' "Auth PostgreSQL role precondition is least-privilege and idempotent"
  fi
}

test_migration_and_start_scripts_exist() {
  [[ -x "$MIGRATE_SCRIPT" ]]
  assert_success "$?" "deterministic infrastructure migration runner exists"
  [[ -x "$UP_SCRIPT" ]]
  assert_success "$?" "startup sequence runner exists"
}

test_migration_runner_replays_and_rejects_unsafe_inputs() {
  local migrations_dir ledger_file hash
  migrations_dir="${TEST_TEMP}/valid-migrations"
  ledger_file="${TEST_TEMP}/empty-ledger"
  : > "$ledger_file"
  make_migrations "$migrations_dir"

  run_migrate "$migrations_dir" "$ledger_file"
  assert_success "$MIGRATE_STATUS" "migration runner applies a canonical empty ledger in fake Docker"
  assert_contains "$(cat "${TEST_TEMP}/docker.stdin")" "BEGIN;" "migration runner sends an explicit transaction to PostgreSQL"

  migrations_dir="${TEST_TEMP}/bad-name"
  ledger_file="${TEST_TEMP}/empty-ledger"
  make_migrations "$migrations_dir"
  mv "${migrations_dir}/0002_auth_bootstrap.sql" "${migrations_dir}/2026xx01_auth_bootstrap.sql"
  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "migration runner rejects a non-canonical name"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.stdin")" "BEGIN;" "non-canonical migration reaches no migration DDL"

  migrations_dir="${TEST_TEMP}/duplicate-id"
  ledger_file="${TEST_TEMP}/empty-ledger"
  make_migrations "$migrations_dir"
  cp "${migrations_dir}/0001_infra_init.sql" "${migrations_dir}/0001_duplicate.sql"
  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "migration runner rejects duplicate IDs"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.stdin")" "BEGIN;" "duplicate IDs reach no migration DDL"

  migrations_dir="${TEST_TEMP}/drift"
  ledger_file="${TEST_TEMP}/drift-ledger"
  make_migrations "$migrations_dir"
  printf '0001|aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n' > "$ledger_file"
  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "migration runner rejects an altered applied migration"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.stdin")" "BEGIN;" "migration drift reaches no new migration DDL"

  migrations_dir="${TEST_TEMP}/out-of-order"
  ledger_file="${TEST_TEMP}/out-of-order-ledger"
  make_migrations "$migrations_dir"
  hash="$(sha256_file "${migrations_dir}/0002_auth_bootstrap.sql")"
  printf '0002|%s\n' "$hash" > "$ledger_file"
  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "migration runner rejects a pending migration before the highest applied ID"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.stdin")" "BEGIN;" "out-of-order ledger reaches no new migration DDL"
}

test_migration_runner_rejects_orphan_ledger_id_before_migration_ddl() {
  local migrations_dir ledger_file stdin_log
  migrations_dir="${TEST_TEMP}/orphan-ledger-migrations"
  ledger_file="${TEST_TEMP}/orphan-ledger"
  make_orphan_ledger_fixture "$migrations_dir" "$ledger_file"

  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "migration runner rejects an applied ledger ID whose canonical file is absent"
  stdin_log="$(cat "${TEST_TEMP}/docker.stdin")"
  assert_not_contains "$stdin_log" "test_migration_0001" "orphan ledger fails before any migration DDL is sent"
  assert_not_contains "$stdin_log" "BEGIN;" "orphan ledger fails before any migration transaction is sent"
}

test_existing_orphan_ledger_fails_before_any_bootstrap_ddl() {
  local migrations_dir ledger_file fake_log
  migrations_dir="${TEST_TEMP}/existing-orphan-ledger-migrations"
  ledger_file="${TEST_TEMP}/existing-orphan-ledger"
  make_orphan_ledger_fixture "$migrations_dir" "$ledger_file"

  run_migrate "$migrations_dir" "$ledger_file"
  assert_failure "$MIGRATE_STATUS" "existing orphan ledger fails closed"
  fake_log="$(cat "${TEST_TEMP}/docker.log" "${TEST_TEMP}/docker.stdin")"
  assert_not_contains "$fake_log" "CREATE SCHEMA" "existing orphan ledger sends no schema bootstrap DDL"
  assert_not_contains "$fake_log" "CREATE TABLE" "existing orphan ledger sends no table bootstrap DDL"
  assert_not_contains "$fake_log" "BEGIN;" "existing orphan ledger sends no migration transaction"
}

test_startup_sequences_auth_migration_before_serve() {
  local env_file fake_bin output status log
  env_file="${TEST_TEMP}/startup.env"
  fake_bin="${TEST_TEMP}/startup-bin"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/startup.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/startup.log" bash "$UP_SCRIPT" "$env_file" mf-startup-test > "${TEST_TEMP}/startup.output" 2>&1
  status=$?
  set -e
  assert_success "$status" "startup runs db, infrastructure migration, GoTrue migration and Auth with fake Docker"
  log="$(cat "${TEST_TEMP}/startup.log")"
  assert_contains "$log" "run --rm auth-migrate" "startup invokes the exact GoTrue migration command"
  assert_contains "$log" "up -d --no-deps auth" "startup only serves Auth after the migration step"

  : > "${TEST_TEMP}/startup.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/startup.log" FAKE_DOCKER_FAIL_MATCH='run --rm auth-migrate' bash "$UP_SCRIPT" "$env_file" mf-startup-test > "${TEST_TEMP}/startup-fail.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" "failed GoTrue migration fails startup"
  assert_not_contains "$(cat "${TEST_TEMP}/startup.log")" "up -d --no-deps auth" "failed GoTrue migration never starts Auth serve"

  : > "${TEST_TEMP}/startup.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/startup.log" FAKE_DOCKER_FAIL_MATCH='up -d db' bash "$UP_SCRIPT" "$env_file" mf-startup-test > "${TEST_TEMP}/startup-db-fail.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" "failed database compose startup fails startup"
  assert_contains "$(cat "${TEST_TEMP}/startup-db-fail.output")" 'Startup error: database Compose start failed' "database compose failure is reported without runtime output"
}

case "${1:-all}" in
  migrations|all)
    test_migration_surface_is_canonical
    test_auth_bootstrap_precedes_auth_service
    test_auth_schema_precondition_exists_before_gotrue_migrate
    test_auth_postgres_role_precondition_exists_before_gotrue_migrate
    test_migration_and_start_scripts_exist
    test_migration_runner_replays_and_rejects_unsafe_inputs
    test_migration_runner_rejects_orphan_ledger_id_before_migration_ddl
    test_existing_orphan_ledger_fails_before_any_bootstrap_ddl
    test_startup_sequences_auth_migration_before_serve
    ;;
  *)
    printf 'unknown migration test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
