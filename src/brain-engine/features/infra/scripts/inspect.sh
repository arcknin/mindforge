#!/usr/bin/env bash
set -u -o pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/lib/compose.sh"

CHECK_FAILURES=0

vector_version_is_compatible() {
  local version="${1:-}"
  local major minor
  [[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || return 1
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  (( major == 0 && minor >= 8 ))
}

check() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf '%s: OK\n' "$label"
  else
    printf '%s: FAIL\n' "$label" >&2
    CHECK_FAILURES=$((CHECK_FAILURES + 1))
  fi
}

check_target() {
  local target="$1"
  local auth_service="$2"
  local tool="$3"
  local output
  check "${target} Auth container" compose_cmd ps -q "$auth_service"
  check "${target} Auth health" compose_cmd exec -T "$auth_service" wget -q --spider http://127.0.0.1:9999/health
  if output="$("$tool" preflight \
    --config "$ENV_FILE" \
    --source-root "${REPOSITORY_ROOT}" \
    --target "$target" \
    --operator-id "$OPERATOR_ID" \
    --authorization-ref "$AUTHORIZATION_REF" \
    --scope infrastructure 2>/dev/null)" && [[ "$output" == preflight-reference=* ]]; then
    printf '%s preflight: OK\n' "$target"
  else
    printf '%s preflight: FAIL\n' "$target" >&2
    CHECK_FAILURES=$((CHECK_FAILURES + 1))
  fi
}

main() {
  ENV_FILE="${1:-}"
  PROJECT_NAME="${2:-}"
  OPERATOR_ID="${3:-}"
  AUTHORIZATION_REF="${4:-}"
  local tool

  [[ -n "$ENV_FILE" && -n "$PROJECT_NAME" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" ]] || {
    printf 'usage: %s <env-file> <project-name> <operator-id> <authorization-ref>\n' "$0" >&2
    return 2
  }
  ENV_FILE="$ENV_FILE" validate_config || return 1
  compose_init "$ENV_FILE" "$PROJECT_NAME" || return 1
  tool="$(infra_tool)" || return 1
  check 'Database container' compose_cmd ps -q db
  check 'PostgreSQL readiness' compose_cmd exec -T db pg_isready -U "$POSTGRES_ADMIN_USER" -d postgres
  check_target mindforge_develop auth-develop "$tool"
  check_target mindforge auth "$tool"

  if [[ "$CHECK_FAILURES" -ne 0 ]]; then
    printf 'Inspection failed with %s red check(s)\n' "$CHECK_FAILURES" >&2
    return 1
  fi
  printf 'Inspection completed with independent database/Auth checks green\n'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
