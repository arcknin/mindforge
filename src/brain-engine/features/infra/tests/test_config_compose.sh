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
  assert_not_contains "$output" "synthetic-postgres-password-123456" "$label does not leak database password"
  assert_not_contains "$output" "synthetic-jwt-secret-with-at-least-32-bytes" "$label does not leak JWT secret"
}

test_valid_configuration_passes() {
  local env_file="${TEST_TEMP}/valid.env"
  make_valid_env "$env_file"
  run_validator "$env_file"
  assert_success "$(cat "${TEST_TEMP}/last-status")" "valid configuration passes before Docker"
}

test_invalid_configurations_fail_closed() {
  local env_file

  env_file="${TEST_TEMP}/missing-password.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" POSTGRES_PASSWORD
  assert_rejected_without_secret "$env_file" "missing required value"

  env_file="${TEST_TEMP}/empty-password.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_PASSWORD ""
  assert_rejected_without_secret "$env_file" "empty required value"

  env_file="${TEST_TEMP}/placeholder.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" POSTGRES_PASSWORD "replace-with-strong-password"
  assert_rejected_without_secret "$env_file" "placeholder value"

  env_file="${TEST_TEMP}/short-jwt.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" GOTRUE_JWT_SECRET "too-short"
  assert_rejected_without_secret "$env_file" "short JWT"

  env_file="${TEST_TEMP}/missing-external-url.env"
  make_valid_env "$env_file"
  remove_env_key "$env_file" GOTRUE_API_EXTERNAL_URL
  assert_rejected_without_secret "$env_file" "missing Auth external URL"

  env_file="${TEST_TEMP}/invalid-external-url.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" GOTRUE_API_EXTERNAL_URL "not-a-url"
  assert_rejected_without_secret "$env_file" "invalid Auth external URL"

  env_file="${TEST_TEMP}/wildcard-redirect.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" GOTRUE_URI_ALLOW_LIST "https://*.example.invalid"
  assert_rejected_without_secret "$env_file" "wildcard redirect"

  env_file="${TEST_TEMP}/lan-without-opt-in.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" AUTH_BIND_ADDR "0.0.0.0"
  assert_rejected_without_secret "$env_file" "LAN bind without opt-in"

  env_file="${TEST_TEMP}/mismatched-database-url.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" GOTRUE_DB_DATABASE_URL "postgres://other:wrong@db:5432/other?sslmode=disable"
  assert_rejected_without_secret "$env_file" "mismatched GoTrue database URL"

  env_file="${TEST_TEMP}/missing-auth-search-path.env"
  make_valid_env "$env_file"
  replace_env_value "$env_file" GOTRUE_DB_DATABASE_URL "postgres://mf_user:synthetic-postgres-password-123456@db:5432/mf_test?sslmode=disable"
  assert_rejected_without_secret "$env_file" "GoTrue database URL without the Auth search path"
}

test_compose_topology_isolated_and_pinned() {
  local compose
  compose="$(cat "$COMPOSE_FILE")"
  assert_not_contains "$compose" "container_name:" "Compose has no fixed container names"
  assert_contains "$compose" "image: pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b" "PostgreSQL image uses the approved exact digest"
  assert_contains "$compose" "platform: linux/amd64" "PostgreSQL image declares the approved architecture"
  assert_not_contains "$compose" "image: pgvector/pgvector:0.8.0-pg16" "PostgreSQL image no longer uses the incompatible version tag"
  assert_contains "$compose" 'GOTRUE_API_PORT: "9999"' "Auth internal API port is fixed"
  assert_contains "$compose" 'http://127.0.0.1:9999/health' "Auth healthcheck uses the fixed internal port"
  assert_contains "$compose" ':9999"' "Auth publishes configurable host port to fixed internal port"
  assert_contains "$compose" 'DB_NAMESPACE: "auth"' "GoTrue uses the explicit Auth database namespace"
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

test_compose_volume_identity_is_historical_and_project_scoped() {
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
  render="$(render_compose_config "$env_file" mf-accept-volume-test 2>&1)"
  status=$?
  set -e
  assert_success "$status" "Compose renders the disposable project for volume identity validation"
  assert_contains "$render" "name: mf-accept-volume-test_mindforge-postgres-data" "disposable volume remains project-prefixed and isolated"
  assert_not_contains "$render" "name: infra_mindforge-postgres-data" "disposable project does not attach the protected infra volume"
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
    ;;
  compose)
    test_compose_topology_isolated_and_pinned
    test_compose_volume_identity_is_historical_and_project_scoped
    test_vector_version_compatibility_range
    test_vector_migration_remains_create_only
    test_compose_helper_requires_explicit_project
    ;;
  all)
    test_valid_configuration_passes
    test_invalid_configurations_fail_closed
    test_compose_topology_isolated_and_pinned
    test_compose_volume_identity_is_historical_and_project_scoped
    test_vector_version_compatibility_range
    test_vector_migration_remains_create_only
    test_compose_helper_requires_explicit_project
    ;;
  *)
    printf 'unknown config/compose test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
