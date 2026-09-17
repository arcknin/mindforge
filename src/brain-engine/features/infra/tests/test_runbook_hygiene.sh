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

test_runbook_matches_the_safe_script_contract() {
  local content
  content="$(cat "$RUNBOOK")"
  assert_contains "$content" './scripts/validate-config.sh .env' "runbook documents fail-closed configuration validation"
  assert_contains "$content" './scripts/up.sh .env' "runbook documents ordered startup"
  assert_contains "$content" './scripts/inspect.sh .env' "runbook documents independent inspection"
  assert_contains "$content" 'GOTRUE_API_EXTERNAL_URL' "runbook documents the Auth external URL boundary"
  assert_contains "$content" 'curl' "runbook lists curl for the documented Auth HTTP check"
  assert_contains "$content" 'host port' "runbook distinguishes host ports from internal ports"
  assert_contains "$content" 'external backup directory' "runbook requires backup output outside the checkout"
  assert_contains "$content" 'mf-accept-' "runbook describes exclusive disposable restore projects"
  assert_contains "$content" 'MF_VM_SSH_IDENTITY_FILE' "runbook documents the explicit owner SSH identity for VM validation"
  assert_not_contains "$content" 'pasta_infra' "runbook no longer describes restore with a folder argument"
  assert_not_contains "$content" 'docker-entrypoint-initdb.d' "runbook no longer claims initdb is the migration runner"
  assert_not_contains "$content" 'docker volume rm' "runbook does not offer direct volume removal as recovery"
  assert_not_contains "$content" 'name: infra_mindforge-postgres-data' "runbook does not hardcode the protected Compose-prefixed volume name"
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
  fi

  if [[ -f "$SURFACE_MANIFEST" ]]; then
    while IFS= read -r path || [[ -n "$path" ]]; do
      [[ -n "$path" ]] || continue
      [[ "$path" > "$previous" ]] || fail "versioned-surface manifest is strictly sorted"
      previous="$path"
      [[ -f "${ROOT_DIR}/${path}" ]] || fail "manifest path exists: ${path}"
      case "$path" in
        .env|.env/*|*.dump|*.backup|backups/*|acceptance-artifacts/*|*.secret|*.token)
          fail "manifest excludes secret and backup paths"
          ;;
      esac
    done < "$SURFACE_MANIFEST"
  fi
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
  assert_contains "$registry" 'endpoints: []' "OpenAPI registry remains empty of Mindforge endpoints"
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
    test_runbook_matches_the_safe_script_contract
    test_ignore_and_manifest_cover_the_local_surface
    test_surface_has_no_domain_or_extra_service
    test_sanitized_hash_evidence_is_allowlisted
    ;;
  *)
    printf 'unknown runbook hygiene test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
