#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"

COMPOSE_FILE="${ROOT_DIR}/compose.yaml"
MIGRATE_SCRIPT="${ROOT_DIR}/scripts/migrate.sh"
UP_SCRIPT="${ROOT_DIR}/scripts/up.sh"
MIGRATIONS_DIR="${ROOT_DIR}/postgres/migrations"
FEATURE_MIGRATIONS_DIR="${ROOT_DIR}/postgres/feature-migrations"

test_migration_surface_is_canonical() {
  local compose evidence_migration feature_sql feature_catalog feature_readme canonical_hash
  compose="$(cat "$COMPOSE_FILE")"
  [[ -f "${MIGRATIONS_DIR}/0001_infra_init.sql" ]]
  assert_success "$?" "initial infrastructure migration has a zero-padded numeric ID"
  [[ -f "${MIGRATIONS_DIR}/0005_infra_migration_evidence.sql" ]]
  assert_success "$?" "infrastructure target evidence migration has the next canonical ID"
  [[ ! -f "${MIGRATIONS_DIR}/2026xx01_infra_init.sql" ]]
  assert_success "$?" "non-canonical placeholder migration is absent"
  assert_contains "$compose" './postgres/bootstrap:/docker-entrypoint-initdb.d:ro' "Compose uses initdb only for fresh-cluster bootstrap"
  evidence_migration="$(cat "${MIGRATIONS_DIR}/0005_infra_migration_evidence.sql")"
  assert_contains "$evidence_migration" 'CREATE TABLE IF NOT EXISTS infra.migration_target_evidence' "infrastructure migration defines target evidence records"
  assert_contains "$evidence_migration" 'CREATE TABLE IF NOT EXISTS infra.migration_attempts' "infrastructure migration defines incomplete and failed attempt records"
  assert_contains "$evidence_migration" 'migration_sha256' "target evidence preserves immutable migration hashes"
  assert_contains "$evidence_migration" 'authorization_reference' "target evidence preserves owner authority references"
  assert_contains "$evidence_migration" 'sanitized_outcome' "target evidence records only sanitized outcomes"
  assert_contains "$evidence_migration" "target_name text NOT NULL CHECK (target_name IN ('mindforge_develop', 'mindforge'))" "migration evidence accepts only the physical develop and main database names"
  assert_not_contains "$evidence_migration" 'mindforge_main' "migration evidence has no superseded physical main alias"
  [[ -f "${FEATURE_MIGRATIONS_DIR}/.gitkeep" && -f "${FEATURE_MIGRATIONS_DIR}/README.md" && -f "${FEATURE_MIGRATIONS_DIR}/catalog.json" ]]
  assert_success "$?" "global feature migration catalog descriptor is versioned"
  feature_catalog="$(cat "${FEATURE_MIGRATIONS_DIR}/catalog.json")"
  assert_contains "$feature_catalog" '"id": "0006"' "Brain Core follows infra 0001 through 0005 under one global ID"
  assert_contains "$feature_catalog" 'src/brain-engine/features/brain/migrations/0001_brain_core.sql' "catalog resolves the feature-owned canonical source"
  assert_contains "$feature_catalog" '0001_brain_core.preflight.sql' "catalog registers the feature-owned read-only preflight"
  assert_contains "$feature_catalog" '0001_brain_core.verify.sql' "catalog registers the feature-owned schema verification"
  canonical_hash="$(shasum -a 256 "${REPOSITORY_ROOT}/src/brain-engine/features/brain/migrations/0001_brain_core.sql" | awk '{print $1}')"
  assert_contains "$feature_catalog" "\"sha256\": \"${canonical_hash}\"" "catalog pins the exact canonical source bytes"
  feature_readme="$(cat "${FEATURE_MIGRATIONS_DIR}/README.md")"
  assert_contains "$feature_readme" 'Entry `0006` references Brain Core' "catalog documentation names the approved global Brain Core entry"
  assert_contains "$feature_readme" 'canonical feature-owned' "catalog documentation keeps the payload feature-owned"
  assert_contains "$feature_readme" 'exact SHA-256 of all three files' "catalog documentation describes payload and check hash pins"
  assert_contains "$feature_readme" 'one `infra.migration_ledger` insert in a single transaction' "catalog documentation describes the sole atomic ledger route"
  assert_contains "$feature_readme" 'A matching global ID and hash is a no-op' "catalog documentation describes same-hash idempotency"
  assert_contains "$feature_readme" 'does not prove VM state, SQL application, backup or restore behavior, or runtime behavior' "catalog documentation limits local evidence claims"
  assert_not_contains "$feature_readme" 'brain_core.migration_ledger' "catalog documentation does not introduce a second feature ledger"
  assert_contains "$(cat "${ROOT_DIR}/tests/versioned-surface.txt")" 'postgres/feature-migrations/catalog.json' "versioned surface registers the global catalog descriptor"
  feature_sql="$(find "$FEATURE_MIGRATIONS_DIR" -maxdepth 1 -type f -name '*.sql' -print)"
  [[ -z "$feature_sql" ]]
  assert_success "$?" "feature migration payload is not copied into infra"
}

test_auth_bootstrap_precedes_auth_service() {
  local compose
  compose="$(cat "$COMPOSE_FILE")"
  [[ "$compose" == *$'  auth-develop-migrate:'* && "$compose" == *$'  auth-migrate:'* && "$compose" == *'command: gotrue migrate'* ]]
  assert_success "$?" "two pinned target-specific GoTrue migration services are declared"
  [[ "$compose" == *'condition: service_completed_successfully'* ]]
  assert_success "$?" "each Auth service waits for its successful GoTrue migration"
  [[ "$compose" == *$'  auth-develop:'* && "$compose" == *$'  auth:'* && "$compose" == *'command: gotrue serve'* ]]
  assert_success "$?" "two Auth serve commands remain explicit after target-specific bootstrap"
  assert_not_contains "$compose" $'  auth-main-migrate:' "Compose has no superseded main Auth migration service identity"
  assert_not_contains "$compose" $'  auth-main:' "Compose has no superseded main Auth service identity"
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
  assert_success "$?" "infrastructure provides the PostgreSQL role required by the Auth catalog"
  if [[ -f "$auth_precondition" ]]; then
    assert_contains "$(cat "$auth_precondition")" 'CREATE ROLE postgres NOLOGIN;' "Auth PostgreSQL role precondition is least-privilege and idempotent"
  fi
}

test_migration_and_start_scripts_exist() {
  [[ -x "$MIGRATE_SCRIPT" ]]
  assert_success "$?" "deterministic infrastructure migration adapter exists"
  [[ -x "$UP_SCRIPT" ]]
  assert_success "$?" "VM startup sequence runner exists"
}

test_migration_adapter_is_binary_only_and_explicit() {
  local source
  source="$(cat "$MIGRATE_SCRIPT")"
  assert_contains "$source" 'MINDFORGE_INFRA_BIN' "migration adapter resolves the distributable binary"
  assert_contains "$source" 'TARGET' "migration adapter requires an explicit target"
  assert_contains "$source" 'OPERATOR_ID' "migration adapter requires an explicit operator"
  assert_contains "$source" 'AUTHORIZATION_REF' "migration adapter requires an explicit authority reference"
  assert_not_contains "$source" 'compose_init' "migration adapter does not initialize Compose"
  assert_not_contains "$source" 'compose_cmd' "migration adapter does not invoke Compose"
  assert_not_contains "$source" 'pg_dump' "migration adapter does not run backup clients"
  assert_not_contains "$source" 'pg_restore' "migration adapter does not run restore clients"
  assert_not_contains "$source" 'psql' "migration adapter does not execute SQL directly"
  assert_contains "$source" 'REPOSITORY_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"' "migration adapter resolves the repository root for the global catalog"
  assert_contains "$source" '--source-root "$REPOSITORY_ROOT"' "migration adapter passes the explicit repository root"
  assert_contains "$(cat "$UP_SCRIPT")" '--source-root "$REPOSITORY_ROOT"' "startup passes the repository root to the operator"
  assert_contains "$(cat "${ROOT_DIR}/scripts/inspect.sh")" '--source-root "${REPOSITORY_ROOT}"' "inspect passes the repository root to the operator"
}

test_startup_is_the_only_compose_mutation_adapter() {
  local migrate_source backup_source restore_source up_source
  migrate_source="$(cat "$MIGRATE_SCRIPT")"
  backup_source="$(cat "${ROOT_DIR}/scripts/backup.sh")"
  restore_source="$(cat "${ROOT_DIR}/scripts/restore.sh")"
  up_source="$(cat "$UP_SCRIPT")"
  assert_contains "$up_source" 'compose_cmd up -d db' "VM startup owns the database Compose lifecycle"
  assert_contains "$up_source" 'compose_cmd run --rm auth-develop-migrate' "VM startup owns the develop Auth migration lifecycle"
  assert_contains "$up_source" 'compose_cmd run --rm auth-migrate' "VM startup owns the main Auth migration lifecycle"
  assert_not_contains "$migrate_source" 'docker' "migration adapter has no Docker runtime path"
  assert_not_contains "$backup_source" 'docker' "backup adapter has no Docker runtime path"
  assert_not_contains "$restore_source" 'docker' "restore adapter has no Docker runtime path"
}

test_no_disposable_runtime_surface_remains() {
  [[ ! -e "${ROOT_DIR}/scripts/acceptance-disposable.sh" ]]
  assert_success "$?" "disposable acceptance script is absent"
  [[ ! -e "${ROOT_DIR}/tests/integration_disposable.sh" ]]
  assert_success "$?" "disposable integration runner is absent"
  [[ ! -e "${ROOT_DIR}/tests/test_acceptance_runner.sh" ]]
  assert_success "$?" "disposable acceptance tests are absent"
}

case "${1:-all}" in
  migrations|all)
    test_migration_surface_is_canonical
    test_auth_bootstrap_precedes_auth_service
    test_auth_schema_precondition_exists_before_gotrue_migrate
    test_auth_postgres_role_precondition_exists_before_gotrue_migrate
    test_migration_and_start_scripts_exist
    test_migration_adapter_is_binary_only_and_explicit
    test_startup_is_the_only_compose_mutation_adapter
    test_no_disposable_runtime_surface_remains
    ;;
  *)
    printf 'unknown migration test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
