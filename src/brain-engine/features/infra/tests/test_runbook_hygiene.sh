#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

find_workspace_root() {
  local candidate="${ROOT_DIR}"
  while :; do
    if [[ -f "${candidate}/contracts/openapi/registry.yaml" ]]; then
      printf '%s\n' "$candidate"
      return 0
    fi
    [[ "$candidate" == '/' ]] && break
    candidate="$(cd "${candidate}/.." && pwd -P)" || return 1
  done
  return 1
}

WORKSPACE_ROOT="$(find_workspace_root)" || {
  printf 'Unable to locate workspace marker contracts/openapi/registry.yaml\n' >&2
  exit 1
}
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"

RUNBOOK="${ROOT_DIR}/README.md"
ENV_EXAMPLE="${ROOT_DIR}/.env.example"
IGNORE_FILE="${ROOT_DIR}/.gitignore"
SURFACE_MANIFEST="${ROOT_DIR}/tests/versioned-surface.txt"

test_runbook_matches_the_vm_only_script_contract() {
  local content
  content="$(cat "$RUNBOOK")"
  assert_contains "$content" './scripts/validate-config.sh .env' "runbook documents fail-closed configuration validation"
  assert_contains "$content" './scripts/up.sh .env infra operator-id authorization-ref' "runbook documents authorized ordered startup"
  assert_contains "$content" './scripts/inspect.sh .env infra operator-id authorization-ref' "runbook documents independent target inspection"
  assert_contains "$content" 'mindforge_develop' "runbook documents the develop database"
  assert_contains "$content" 'mindforge' "runbook documents the main database"
  assert_contains "$content" 'GOTRUE_API_EXTERNAL_URL' "runbook documents the Auth external URL boundary"
  assert_contains "$content" 'curl' "runbook lists curl for the documented Auth HTTP check"
  assert_contains "$content" 'host port' "runbook distinguishes host ports from internal ports"
  assert_contains "$content" 'external backup directory' "runbook requires backup output outside the checkout"
  assert_contains "$content" 'GOOS' "runbook documents an explicit binary target platform"
  assert_contains "$content" 'GOARCH' "runbook documents an explicit binary target architecture"
  assert_contains "$content" 'pg_restore' "runbook documents the external PostgreSQL client prerequisite"
  assert_contains "$content" 'restore proof' "runbook preserves the restore proof boundary"
  assert_contains "$content" 'VM-only' "runbook states the runtime boundary"
  assert_not_contains "$content" 'pasta_infra' "runbook no longer describes restore with a folder argument"
  assert_not_contains "$content" 'docker-entrypoint-initdb.d' "runbook no longer claims initdb is the migration runner"
  assert_not_contains "$content" 'docker volume rm' "runbook does not offer direct volume removal as recovery"
  assert_not_contains "$content" 'name: infra_mindforge-postgres-data' "runbook does not hardcode the protected Compose-prefixed volume name"
  assert_not_contains "$content" 'acceptance-disposable' "runbook has no disposable acceptance runner"
  assert_not_contains "$content" 'integration_disposable' "runbook has no disposable integration runner"
  assert_not_contains "$content" 'mf-accept-' "runbook has no disposable project topology"
  assert_not_contains "$content" 'POSTGRES_DB=' "runbook has no legacy single-database configuration"
  assert_not_contains "$content" 'GOTRUE_DB_DATABASE_URL' "runbook has no legacy shared Auth configuration"
  assert_not_contains "$content" 'auth-migrate' "runbook has no stale shared Auth migration service"
  assert_contains "$content" 'The cluster inventory may include unrelated databases such as `nsrdb`' "runbook treats unrelated project databases as passive shared-cluster context"
  assert_contains "$content" 'Mindforge operations accept only `mindforge_develop` or `mindforge`' "runbook restricts operations to the two explicit Mindforge targets"
  assert_contains "$content" 'does not inspect the contents of unrelated databases' "runbook limits unrelated inventory to minimal read-only facts"
  assert_contains "$content" 'unrelated databases are never operation targets' "runbook prevents unrelated inventory names from becoming targets"
}

test_revision6_runbook_shared_cluster_and_feature_migration_boundary() {
  local content
  content="$(cat "$RUNBOOK")"
  assert_contains "$content" 'approved infra contract revision 6' "runbook identifies the current infra contract revision"
  assert_contains "$content" 'production/main database is `mindforge`' "runbook identifies the physical production/main database"
  assert_contains "$content" '`mindforge_develop` is a distinct database' "runbook keeps develop separate from main"
  assert_contains "$content" 'The Compose project name `infra` preserves the existing' "runbook preserves the designated Compose project identity"
  assert_contains "$content" '`infra_mindforge-postgres-data` volume' "runbook preserves the designated cluster volume identity"
  assert_contains "$content" 'Adding develop to the existing volume is an explicit, separately authorized' "runbook requires separate authority for develop provisioning"
  assert_contains "$content" 'provisioning action' "runbook describes the explicit develop addition"
  assert_contains "$content" './scripts/provision-develop.sh .env mindforge_develop operator-id authorization-ref' "runbook documents the explicit fixed-target provisioning invocation"
  assert_contains "$content" 'startup and restart never create, replace, rename, or copy either database' "runbook excludes implicit database provisioning"
  assert_contains "$content" '`auth` service remains bound to `mindforge`' "runbook preserves the existing main Auth binding"
  assert_contains "$content" 'SQL execution and real backup handling remain VM-only' "runbook keeps SQL and backup VM-only"
  assert_contains "$content" 'require separate' "runbook keeps SQL and backup authorization separate"
  assert_contains "$content" 'owner authorization' "runbook identifies who authorizes SQL and backup work"
  assert_contains "$content" 'Backup validation does not prove restoration' "runbook distinguishes backup validation from restore proof"
  assert_contains "$content" 'Practical restoration of a fresh dump remains pending' "runbook records the pending restore proof"
  assert_contains "$content" 'global infra-catalog ID `0006`' "runbook identifies Brain Core's global migration identity"
  assert_contains "$content" 'src/brain-engine/features/brain/migrations/0001_brain_core.sql' "runbook names the feature-owned canonical migration source"
  assert_contains "$content" '0001_brain_core.preflight.sql' "runbook names the feature-owned read-only preflight check"
  assert_contains "$content" '0001_brain_core.verify.sql' "runbook names the feature-owned schema verification check"
  assert_contains "$content" '`infra.migration_ledger` is the only applied-migration ledger' "runbook names the sole authoritative migration ledger"
  assert_contains "$content" 'same transaction' "runbook documents atomic payload, verification, and ledger application"
  assert_contains "$content" 'does not prove VM state, SQL application, backup or restore behavior, or runtime behavior' "runbook limits local evidence claims"
  assert_not_contains "$content" 'feature migration catalog starts empty' "runbook no longer describes the approved catalog as empty"
  assert_not_contains "$content" 'migration-policy mismatch remains a separate unresolved gate' "runbook removes the resolved Brain Core policy blocker"
  assert_not_contains "$content" 'standalone Brain Core migration path' "runbook does not preserve a parallel Brain Core migration route"
  assert_not_contains "$content" 'brain_core.migration_ledger' "runbook does not describe a second Brain Core migration ledger"
  assert_not_contains "$content" 'mindforge_main' "runbook has no active instruction naming the superseded physical main target"
}

test_ignore_and_manifest_cover_the_local_surface() {
  local previous=''
  [[ -f "$IGNORE_FILE" ]]
  assert_success "$?" "local ignore file exists"
  [[ -f "$SURFACE_MANIFEST" ]]
  assert_success "$?" "versionable-surface manifest exists"

  if [[ -f "$IGNORE_FILE" ]]; then
    local ignore
    ignore="$(cat "$IGNORE_FILE")"
    assert_contains "$ignore" '.env' "ignore covers real environment files"
    assert_contains "$ignore" '*.dump' "ignore covers database dumps"
    assert_contains "$ignore" 'backups/' "ignore covers backup directories"
    assert_contains "$ignore" '*.manifest.json' "ignore covers backup manifests"
    assert_contains "$ignore" 'mindforge-infra' "ignore covers generated infrastructure binaries"
    assert_contains "$ignore" 'promotion-evidence/' "ignore covers sanitized promotion evidence"
  fi

  if [[ -f "$ENV_EXAMPLE" ]]; then
    local env_example
    env_example="$(cat "$ENV_EXAMPLE")"
    assert_contains "$env_example" 'POSTGRES_CLIENT_HOST=127.0.0.1' "example separates host-side client address"
    assert_contains "$env_example" 'POSTGRES_HOST=db' "example keeps Auth on the internal Compose endpoint"
  fi

  if [[ -f "$SURFACE_MANIFEST" ]]; then
    while IFS= read -r path || [[ -n "$path" ]]; do
      [[ -n "$path" ]] || continue
      [[ "$path" > "$previous" ]] || fail "versioned-surface manifest is strictly sorted"
      previous="$path"
      [[ -f "${ROOT_DIR}/${path}" ]] || fail "manifest path exists: ${path}"
      case "$path" in
        .env|.env/*|*.dump|*.backup|*.partial|backups/*|*.manifest.json|*.evidence.json|promotion-evidence/*|restore-evidence/*|mindforge-infra|*.secret|*.token)
          fail "manifest excludes secret and backup paths"
          ;;
      esac
    done < "$SURFACE_MANIFEST"
  fi
}

test_execution_surface_has_no_disposable_or_inline_database_runner() {
  local script source
  [[ ! -e "${ROOT_DIR}/scripts/acceptance-disposable.sh" ]]
  assert_success "$?" "disposable acceptance script is removed"
  [[ ! -e "${ROOT_DIR}/tests/integration_disposable.sh" ]]
  assert_success "$?" "disposable integration runner is removed"
  [[ ! -e "${ROOT_DIR}/tests/test_acceptance_runner.sh" ]]
  assert_success "$?" "disposable acceptance tests are removed"

  source="$(cat "${ROOT_DIR}/tests/run.sh")"
  assert_not_contains "$source" 'integration' "local test runner has no integration mode"
  assert_not_contains "$source" 'acceptance-runner' "local test runner has no disposable acceptance selection"

  for script in "${ROOT_DIR}/scripts/migrate.sh" "${ROOT_DIR}/scripts/backup.sh" "${ROOT_DIR}/scripts/restore.sh"; do
    source="$(cat "$script")"
    assert_not_contains "$source" 'compose_cmd' "$(basename "$script") does not invoke Compose"
    assert_not_contains "$source" 'docker' "$(basename "$script") does not invoke Docker"
    assert_not_contains "$source" 'pg_dump' "$(basename "$script") does not run pg_dump directly"
    assert_not_contains "$source" 'pg_restore' "$(basename "$script") does not run pg_restore directly"
    assert_not_contains "$source" 'psql' "$(basename "$script") does not run SQL directly"
    assert_contains "$source" 'MINDFORGE_INFRA_BIN' "$(basename "$script") resolves an explicit distributable binary"
  done

  for script in "${ROOT_DIR}/scripts/up.sh" "${ROOT_DIR}/scripts/inspect.sh" "${ROOT_DIR}/scripts/lib/compose.sh"; do
    source="$(cat "$script")"
    assert_not_contains "$source" 'acceptance-disposable' "$(basename "$script") has no disposable acceptance route"
    assert_not_contains "$source" 'integration_disposable' "$(basename "$script") has no disposable integration route"
  done
}

test_versioned_surface_matches_the_authorized_package() {
  local path
  for path in \
    'postgres/migrations/0005_infra_migration_evidence.sql' \
    'postgres/feature-migrations/README.md' \
    'postgres/feature-migrations/catalog.json' \
    'tools/cmd/mindforge-infra/main.go' \
    'tools/internal/evidence/evidence.go' \
    'tools/internal/migrations/catalog.go' \
    'tools/internal/operations/operations.go'; do
    grep -Fxq "$path" "$SURFACE_MANIFEST"
    assert_success "$?" "versioned-surface includes ${path}"
  done
  for path in \
    'scripts/acceptance-disposable.sh' \
    'tests/integration_disposable.sh' \
    'tests/test_acceptance_runner.sh'; do
    grep -Fxq "$path" "$SURFACE_MANIFEST"
    assert_failure "$?" "versioned-surface excludes removed ${path}"
  done
}

test_surface_has_no_domain_or_extra_service() {
  local compose migration_sql registry services
  compose="$(cat "${ROOT_DIR}/compose.yaml")"
  migration_sql="$(cat "${ROOT_DIR}/postgres/migrations/"*.sql)"
  registry="$(cat "${WORKSPACE_ROOT}/contracts/openapi/registry.yaml")"
  services="$(awk '/^  [a-z][a-z-]*:$/ { sub(/:.*/, "", $1); print $1 }' "${ROOT_DIR}/compose.yaml")"

  assert_not_contains "$migration_sql" 'CREATE TABLE users' "migrations do not create a Mindforge users table"
  assert_not_contains "$migration_sql" 'CREATE TABLE brains' "migrations do not create a Mindforge brains table"
  assert_not_contains "$migration_sql" 'CREATE TABLE neural_layers' "migrations do not create a Mindforge neural_layers table"
  assert_not_contains "$migration_sql" 'CREATE TABLE documents' "migrations do not create a Mindforge documents table"
  local infra_endpoint_owners
  infra_endpoint_owners="$(printf '%s\n' "$registry" | awk '/^[[:space:]]*owner:[[:space:]]*infra([[:space:]]|$)/ { print }')"
  assert_not_contains "$infra_endpoint_owners" 'owner:' "infrastructure does not own product HTTP endpoints; independent Brain contracts are allowed"
  assert_not_contains "$services" $'postgrest\n' "Compose includes no PostgREST service"
  assert_not_contains "$services" $'studio\n' "Compose includes no Studio service"
  assert_not_contains "$services" $'storage\n' "Compose includes no Storage service"
}

test_sanitized_hash_evidence_is_allowlisted() {
  local temp_manifest path
  temp_manifest="$(mktemp "${TMPDIR:-/tmp}/mindforge-infra-surface.XXXXXX")"
  trap 'rm -f "$temp_manifest"' RETURN
  while IFS= read -r path || [[ -n "$path" ]]; do
    [[ -n "$path" ]] || continue
    if command -v shasum >/dev/null 2>&1; then
      shasum -a 256 "${ROOT_DIR}/${path}" | awk -v path="$path" '{print $1 " " path}' >> "$temp_manifest"
    else
      sha256sum "${ROOT_DIR}/${path}" | awk -v path="$path" '{print $1 " " path}' >> "$temp_manifest"
    fi
  done < "$SURFACE_MANIFEST"
  [[ -s "$temp_manifest" ]]
  assert_success "$?" "sanitized SHA-256 evidence is generated only from the manifest"
  assert_not_contains "$(cat "$temp_manifest")" "$ROOT_DIR/.env" "hash evidence omits real environment files"
}

case "${1:-all}" in
  runbook-hygiene|all)
    test_runbook_matches_the_vm_only_script_contract
    test_revision6_runbook_shared_cluster_and_feature_migration_boundary
    test_ignore_and_manifest_cover_the_local_surface
    test_execution_surface_has_no_disposable_or_inline_database_runner
    test_versioned_surface_matches_the_authorized_package
    test_surface_has_no_domain_or_extra_service
    test_sanitized_hash_evidence_is_allowlisted
    ;;
  *)
    printf 'unknown runbook hygiene test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
