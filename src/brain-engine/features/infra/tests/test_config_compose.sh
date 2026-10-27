#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fixture.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fake-docker.sh"

VALIDATOR="${ROOT_DIR}/scripts/validate-config.sh"
COMPOSE_HELPER="${ROOT_DIR}/scripts/lib/compose.sh"
COMPOSE_FILE="${ROOT_DIR}/compose.yaml"
INSPECT_SCRIPT="${ROOT_DIR}/scripts/inspect.sh"
VECTOR_MIGRATION="${ROOT_DIR}/postgres/migrations/0001_infra_init.sql"
TEST_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-config.XXXXXX")"
trap 'rm -rf "$TEST_TEMP"' EXIT

run_validator() {
  local env_file="$1"
  local output status
  set +e
  output="$(bash "$VALIDATOR" "$env_file" 2>&1)"
  status=$?
  set -e
  printf '%s\n' "$status" > "${TEST_TEMP}/last-status"
  printf '%s' "$output" > "${TEST_TEMP}/last-output"
}

assert_rejected_without_secret() {
  local env_file="$1"
  local label="$2"
  run_validator "$env_file"
  local status output
  status="$(cat "${TEST_TEMP}/last-status")"
  output="$(cat "${TEST_TEMP}/last-output")"
  assert_failure "$status" "$label is rejected"
  assert_not_contains "$output" "synthetic-cluster-password-123456" "$label does not leak cluster password"
  assert_not_contains "$output" "synthetic-develop-password-123456" "$label does not leak develop password"
  assert_not_contains "$output" "synthetic-main-password-123456" "$label does not leak main password"
  assert_not_contains "$output" "synthetic-develop-jwt-secret-with-32-bytes" "$label does not leak develop JWT secret"
  assert_not_contains "$output" "synthetic-main-jwt-secret-with-32-bytes" "$label does not leak main JWT secret"
}

test_valid_configuration_passes() {
  local env_file="${TEST_TEMP}/valid.env"
  make_valid_env "$env_file"
  run_validator "$env_file"
  assert_success "$(cat "${TEST_TEMP}/last-status")" "valid configuration passes before Docker"
}

test_invalid_configurations_fail_closed() {
  local env_file

  env_file="${TEST_TEMP}/missing-client-host.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" POSTGRES_CLIENT_HOST
  assert_rejected_without_secret "$env_file" "missing host-side PostgreSQL client address"

  env_file="${TEST_TEMP}/unsafe-client-host.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_CLIENT_HOST 'db;select'
  assert_rejected_without_secret "$env_file" "unsafe host-side PostgreSQL client address"

  env_file="${TEST_TEMP}/malformed-client-host.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_CLIENT_HOST 'db..home.arpa'
  assert_rejected_without_secret "$env_file" "malformed host-side PostgreSQL client hostname"

  env_file="${TEST_TEMP}/missing-password.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" POSTGRES_ADMIN_PASSWORD
  assert_rejected_without_secret "$env_file" "missing required value"

  env_file="${TEST_TEMP}/empty-password.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_ADMIN_PASSWORD ""
  assert_rejected_without_secret "$env_file" "empty required value"

  env_file="${TEST_TEMP}/placeholder.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_ADMIN_PASSWORD "replace-with-strong-password"
  assert_rejected_without_secret "$env_file" "placeholder value"

  env_file="${TEST_TEMP}/short-jwt.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_JWT_SECRET "too-short"
  assert_rejected_without_secret "$env_file" "short JWT"

  env_file="${TEST_TEMP}/missing-external-url.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" DEVELOP_AUTH_API_EXTERNAL_URL
  assert_rejected_without_secret "$env_file" "missing Auth external URL"

  env_file="${TEST_TEMP}/invalid-external-url.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_API_EXTERNAL_URL "not-a-url"
  assert_rejected_without_secret "$env_file" "invalid Auth external URL"

  env_file="${TEST_TEMP}/wildcard-redirect.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_URI_ALLOW_LIST "https://*.example.invalid"
  assert_rejected_without_secret "$env_file" "wildcard redirect"

  env_file="${TEST_TEMP}/lan-without-opt-in.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_BIND_ADDR "0.0.0.0"
  assert_rejected_without_secret "$env_file" "LAN bind without opt-in"

  env_file="${TEST_TEMP}/mismatched-database-url.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_DB_URL "postgres://other:wrong@db:5432/other?sslmode=disable"
  assert_rejected_without_secret "$env_file" "mismatched GoTrue database URL"

  env_file="${TEST_TEMP}/missing-auth-search-path.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_DB_URL "postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop?sslmode=disable"
  assert_rejected_without_secret "$env_file" "GoTrue database URL without the Auth search path"

  env_file="${TEST_TEMP}/superseded-main-target.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_DB_NAME mindforge_main
  assert_rejected_without_secret "$env_file" "superseded physical main database name"

  env_file="${TEST_TEMP}/missing-main-target.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" MAIN_AUTH_DB_URL
  assert_rejected_without_secret "$env_file" "missing explicit main target"

  env_file="${TEST_TEMP}/swapped-auth-target.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" DEVELOP_AUTH_DB_URL "postgres://mf_main:synthetic-main-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth"
  assert_rejected_without_secret "$env_file" "swapped Auth database target"

  env_file="${TEST_TEMP}/shared-auth-secret.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_AUTH_JWT_SECRET "synthetic-develop-jwt-secret-with-32-bytes"
  assert_rejected_without_secret "$env_file" "shared Auth secret"

  env_file="${TEST_TEMP}/shared-database-role.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_DB_USER mf_develop
  replace_env_value "$env_file" MAIN_AUTH_DB_URL "postgres://mf_develop:synthetic-main-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth"
  assert_rejected_without_secret "$env_file" "shared application database role"

  env_file="${TEST_TEMP}/shared-database-password.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_DB_PASSWORD synthetic-develop-password-123456
  replace_env_value "$env_file" MAIN_AUTH_DB_URL "postgres://mf_main:synthetic-develop-password-123456@db:5432/mindforge?sslmode=disable&search_path=auth"
  assert_rejected_without_secret "$env_file" "shared application database password"

  env_file="${TEST_TEMP}/admin-develop-role.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_ADMIN_USER mf_develop
  replace_env_value "$env_file" POSTGRES_ADMIN_PASSWORD synthetic-develop-password-123456
  assert_rejected_without_secret "$env_file" "cluster admin sharing the develop role"

  env_file="${TEST_TEMP}/duplicate-auth-url.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_AUTH_API_EXTERNAL_URL "http://127.0.0.1:19999"
  assert_rejected_without_secret "$env_file" "duplicate Auth external URL"

  env_file="${TEST_TEMP}/duplicate-auth-port.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" MAIN_AUTH_PORT 19999
  assert_rejected_without_secret "$env_file" "duplicate Auth host port"

  env_file="${TEST_TEMP}/legacy-single-target.env"
  make_valid_env "$env_file"
  printf 'POSTGRES_DB=legacy_single_target\n' >> "$env_file"
  assert_rejected_without_secret "$env_file" "legacy single-target configuration"

  env_file="${TEST_TEMP}/third-target.env"
  make_valid_env "$env_file"
  printf 'EXTRA_DB_NAME=mindforge_extra\n' >> "$env_file"
  assert_rejected_without_secret "$env_file" "third application database configuration"
}

test_client_postgres_address_is_independent_from_auth_endpoint() {
  local env_file
  env_file="${TEST_TEMP}/client-host.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_CLIENT_HOST 127.0.0.1
  run_validator "$env_file"
  assert_success "$(cat "${TEST_TEMP}/last-status")" "explicit host-side client address passes validation"
  assert_contains "$(cat "$env_file")" 'POSTGRES_HOST=db' "Auth retains the internal Compose database alias"
  assert_contains "$(cat "$env_file")" 'DEVELOP_AUTH_DB_URL=postgres://mf_develop:synthetic-develop-password-123456@db:5432/mindforge_develop' "develop Auth URL keeps its internal endpoint"
  assert_contains "$(cat "$env_file")" 'MAIN_AUTH_DB_URL=postgres://mf_main:synthetic-main-password-123456@db:5432/mindforge' "main Auth URL keeps its internal endpoint"
}

test_cluster_admin_can_explicitly_reuse_main_superuser() {
  local env_file
  env_file="${TEST_TEMP}/main-admin-alias.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_ADMIN_USER mf_main
  replace_env_value "$env_file" POSTGRES_ADMIN_PASSWORD synthetic-main-password-123456
  run_validator "$env_file"
  assert_success "$(cat "${TEST_TEMP}/last-status")" "cluster admin may reuse the explicitly configured main superuser"

  replace_env_value "$env_file" POSTGRES_ADMIN_PASSWORD different-admin-password-123456
  assert_rejected_without_secret "$env_file" "aliased main admin with mismatched password"
}

test_compose_topology_isolated_and_pinned() {
  local compose main_auth main_migration develop_auth develop_migration
  compose="$(cat "$COMPOSE_FILE")"
  assert_not_contains "$compose" "container_name:" "Compose has no fixed container names"
  assert_contains "$compose" "image: pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b" "PostgreSQL image uses the approved exact digest"
  assert_contains "$compose" "platform: linux/amd64" "PostgreSQL image declares the approved architecture"
  assert_not_contains "$compose" "image: pgvector/pgvector:0.8.0-pg16" "PostgreSQL image no longer uses the incompatible version tag"
  assert_not_contains "$compose" "supabase/gotrue:latest" "GoTrue image is not mutable"
  assert_contains "$compose" "image: supabase/gotrue:v2.196.0" "GoTrue image uses the approved pinned version"
  assert_contains "$compose" 'auth-develop-migrate:' "Compose has a develop Auth migration service"
  assert_contains "$compose" 'auth-develop:' "Compose has a develop Auth service"
  assert_contains "$compose" $'  auth-migrate:' "Compose preserves the existing main Auth migration service identity"
  assert_contains "$compose" $'  auth:' "Compose preserves the existing main Auth service identity"
  assert_not_contains "$compose" 'auth-main-migrate:' "Compose does not rename the existing main Auth migration service"
  assert_not_contains "$compose" $'  auth-main:' "Compose does not rename the existing main Auth service"
  main_migration="$(compose_service auth-migrate)"
  main_auth="$(compose_service auth)"
  develop_migration="$(compose_service auth-develop-migrate)"
  develop_auth="$(compose_service auth-develop)"
  assert_contains "$main_migration" 'GOTRUE_DB_DATABASE_URL: ${MAIN_AUTH_DB_URL}' "main Auth migration is bound to the main database URL"
  assert_contains "$main_auth" 'GOTRUE_DB_DATABASE_URL: ${MAIN_AUTH_DB_URL}' "existing main Auth service is bound to the main database URL"
  assert_not_contains "$main_auth" '${DEVELOP_AUTH_DB_URL}' "main Auth does not use develop credentials"
  assert_contains "$develop_migration" 'GOTRUE_DB_DATABASE_URL: ${DEVELOP_AUTH_DB_URL}' "develop Auth migration is bound to the develop database URL"
  assert_contains "$develop_auth" 'GOTRUE_DB_DATABASE_URL: ${DEVELOP_AUTH_DB_URL}' "develop Auth is bound to the develop database URL"
  assert_not_contains "$develop_auth" '${MAIN_AUTH_DB_URL}' "develop Auth does not use main credentials"
  assert_contains "$compose" 'MIND_FORGE_DEVELOP_DB: ${DEVELOP_DB_NAME}' "Bootstrap receives the develop database name"
  assert_contains "$compose" 'MIND_FORGE_MAIN_DB: ${MAIN_DB_NAME}' "Bootstrap receives the main database name"
  assert_contains "$compose" 'GOTRUE_API_PORT: "9999"' "Auth internal API port is fixed"
  assert_contains "$compose" 'http://127.0.0.1:9999/health' "Auth healthcheck uses the fixed internal port"
  assert_contains "$compose" ':9999"' "Auth publishes configurable host port to fixed internal port"
  assert_contains "$compose" 'DB_NAMESPACE: "auth"' "GoTrue uses the explicit Auth database namespace"
}

test_mutable_compose_image_fixture_is_rejected() {
  local env_file="${TEST_TEMP}/mutable-image.env"
  local mutable_compose output status
  make_valid_env "$env_file"
  mutable_compose="${TEST_TEMP}/mutable-compose.yaml"
  sed 's/supabase\/gotrue:v2.196.0/supabase\/gotrue:latest/g' "$COMPOSE_FILE" > "$mutable_compose"

  set +e
  output="$(bash -c 'source "$1"; compose_validate_images "$2"' _ "$COMPOSE_HELPER" "$COMPOSE_FILE" 2>&1)"
  status=$?
  set -e
  assert_success "$status" "pinned Compose definition passes image validation"

  set +e
  output="$(bash -c 'source "$1"; compose_validate_images "$2"' _ "$COMPOSE_HELPER" "$mutable_compose" 2>&1)"
  status=$?
  set -e
  assert_failure "$status" "mutable GoTrue image fixture is rejected"
  assert_contains "$output" "unsupported or mutable image" "mutable image failure is explicit"
}

test_bootstrap_topology_has_exact_databases_and_no_product_objects() {
  local bootstrap database_count
  bootstrap="$(cat "${ROOT_DIR}/postgres/bootstrap/001-create-databases.sh")"
  assert_contains "$bootstrap" 'mindforge_develop' "Bootstrap creates the fixed develop database"
  assert_contains "$bootstrap" "MIND_FORGE_MAIN_DB\" == 'mindforge'" "Bootstrap creates the existing physical main database name on a fresh cluster"
  assert_not_contains "$bootstrap" 'mindforge_main' "Bootstrap rejects the superseded physical main database name"
  assert_contains "$bootstrap" 'CREATE DATABASE' "Bootstrap creates databases only during fresh cluster initialization"
  database_count="$(grep -c 'CREATE DATABASE' "${ROOT_DIR}/postgres/bootstrap/001-create-databases.sh")"
  if [[ "$database_count" == '2' ]]; then
    pass "fresh-cluster bootstrap creates exactly two application databases"
  else
    fail "fresh-cluster bootstrap must create exactly two application databases (found $database_count)"
  fi
  assert_not_contains "$bootstrap" 'mindforge_extra' "Bootstrap has no third Mindforge database"
  assert_not_contains "$bootstrap" 'CREATE TABLE' "Bootstrap does not create product tables"
  assert_not_contains "$bootstrap" 'CREATE SCHEMA public' "Bootstrap does not recreate the product schema"
}

compose_service() {
  local service="$1"
  awk -v service="$service" '
    $0 == "  " service ":" { found = 1; next }
    found && /^  [a-z0-9-]+:/ { exit }
    found { print }
  ' "$COMPOSE_FILE"
}

render_compose_config() {
  local env_file="$1"
  local project_name="$2"
  if docker compose version >/dev/null 2>&1; then
    docker compose --project-name "$project_name" --env-file "$env_file" -f "$COMPOSE_FILE" config
  elif command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1; then
    docker-compose --project-name "$project_name" --env-file "$env_file" -f "$COMPOSE_FILE" config
  else
    return 1
  fi
}

test_compose_volume_identity_is_project_scoped() {
  local env_file render status
  env_file="${TEST_TEMP}/volume-identity.env"
  make_valid_env "$env_file"

  set +e
  render="$(render_compose_config "$env_file" infra 2>&1)"
  status=$?
  set -e
  assert_success "$status" "Compose renders the infra project for volume identity validation"
  assert_contains "$render" "name: infra_mindforge-postgres-data" "infra renders the protected historical volume name"
  assert_not_contains "$render" "name: infra_postgres-data" "infra does not render the regressed volume name"

  set +e
  render="$(render_compose_config "$env_file" owner-stack-check 2>&1)"
  status=$?
  set -e
  assert_success "$status" "Compose renders another explicitly named VM project"
  assert_contains "$render" "name: owner-stack-check_mindforge-postgres-data" "named projects retain project-scoped storage"
  assert_not_contains "$render" "name: infra_mindforge-postgres-data" "other named project does not attach the protected infra volume"
}

test_no_legacy_single_database_or_disposable_config_surface() {
  local env_example
  env_example="$(cat "${ROOT_DIR}/.env.example")"
  assert_contains "$env_example" 'DEVELOP_DB_NAME=mindforge_develop' "configuration example names the separate develop database"
  assert_contains "$env_example" 'MAIN_DB_NAME=mindforge' "configuration example preserves the physical main database name"
  assert_contains "$env_example" 'POSTGRES_CLIENT_HOST=127.0.0.1' "configuration example separates host-side PostgreSQL clients"
  assert_contains "$env_example" 'POSTGRES_HOST=db' "configuration example retains the internal Auth endpoint"
  assert_contains "$env_example" '/mindforge?sslmode=disable&search_path=auth' "main Auth URL targets the physical main database"
  assert_contains "$env_example" 'MAIN_AUTH_PORT=9999' "configuration example preserves the main Auth host port"
  assert_contains "$env_example" 'DEVELOP_AUTH_PORT=10000' "configuration example assigns a distinct develop Auth host port"
  assert_not_contains "$env_example" 'mindforge_main' "configuration example has no superseded main database name"
  assert_not_contains "$env_example" 'POSTGRES_DB=' "configuration example has no legacy single-database variable"
  assert_not_contains "$env_example" 'GOTRUE_DB_DATABASE_URL' "configuration example has no legacy shared Auth URL"
  assert_not_contains "$env_example" 'MF_ENV_CLASS=' "configuration example has no disposable environment class"
  assert_not_contains "$env_example" 'MF_DISPOSABLE_' "configuration example has no disposable target controls"
}

run_vector_version_check() {
  local version="$1"
  set +e
  bash -c 'inspect_script="$1"; version="$2"; set --; source "$inspect_script"; vector_version_is_compatible "$version"' _ "$INSPECT_SCRIPT" "$version"
  local status=$?
  return "$status"
}

test_vector_version_compatibility_range() {
  local status

  set +e
  run_vector_version_check '0.8.0'
  status=$?
  assert_success "$status" "vector 0.8.0 is accepted"

  set +e
  run_vector_version_check '0.8.6'
  status=$?
  assert_success "$status" "vector 0.8.6 is accepted"

  set +e
  run_vector_version_check '0.7.99'
  status=$?
  assert_failure "$status" "vector below 0.8.0 is rejected"

  set +e
  run_vector_version_check '1.0.0'
  status=$?
  assert_failure "$status" "vector 1.0.0 and newer are rejected"

  set +e
  run_vector_version_check '0.8'
  status=$?
  assert_failure "$status" "malformed vector version is rejected"
}

test_vector_migration_remains_create_only() {
  local migration
  migration="$(cat "$VECTOR_MIGRATION")"
  assert_contains "$migration" 'CREATE EXTENSION IF NOT EXISTS "vector";' "vector migration remains create-if-absent"
  assert_not_contains "$migration" 'ALTER EXTENSION' "vector migration does not alter or downgrade an installed extension"
}

test_compose_helper_requires_explicit_project() {
  local env_file="${TEST_TEMP}/valid.env"
  local fake_bin="${TEST_TEMP}/bin"
  local output status
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"

  set +e
  output="$(PATH="${fake_bin}:$PATH" bash "$COMPOSE_HELPER" "$env_file" '' config --quiet 2>&1)"
  status=$?
  set -e
  assert_failure "$status" "Compose helper rejects absent project name"

  : > "${TEST_TEMP}/docker.log"
  set +e
  output="$(PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" bash "$COMPOSE_HELPER" "$env_file" mf-config-test config --quiet 2>&1)"
  status=$?
  set -e
  assert_success "$status" "Compose helper renders synthetic configuration with explicit project"
  assert_contains "$(cat "${TEST_TEMP}/docker.log")" "--project-name mf-config-test" "Compose helper forwards its explicit project name"
}

case "${1:-all}" in
  config)
    test_valid_configuration_passes
    test_invalid_configurations_fail_closed
    test_client_postgres_address_is_independent_from_auth_endpoint
    test_cluster_admin_can_explicitly_reuse_main_superuser
    ;;
  compose)
    test_compose_topology_isolated_and_pinned
    test_mutable_compose_image_fixture_is_rejected
    test_bootstrap_topology_has_exact_databases_and_no_product_objects
    test_compose_volume_identity_is_project_scoped
    test_vector_version_compatibility_range
    test_vector_migration_remains_create_only
    test_compose_helper_requires_explicit_project
    test_no_legacy_single_database_or_disposable_config_surface
    ;;
  all)
    test_valid_configuration_passes
    test_invalid_configurations_fail_closed
    test_compose_topology_isolated_and_pinned
    test_bootstrap_topology_has_exact_databases_and_no_product_objects
    test_compose_volume_identity_is_project_scoped
    test_vector_version_compatibility_range
    test_vector_migration_remains_create_only
    test_compose_helper_requires_explicit_project
    test_no_legacy_single_database_or_disposable_config_surface
    ;;
  *)
    printf 'unknown config/compose test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
