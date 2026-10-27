#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPOSITORY_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"
ENV_FILE="${1:-}"
TARGET="${2:-}"
OPERATOR_ID="${3:-}"
AUTHORIZATION_REF="${4:-}"
SCOPE="${5:-infrastructure}"
VERIFICATION_REF="${6:-migrate/${TARGET}/${SCOPE}}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"

migration_fail() {
  printf 'Migration error: %s\n' "$1" >&2
  exit 1
}

[[ -n "$ENV_FILE" && -n "$TARGET" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" ]] || {
  printf 'usage: %s <env-file> <target> <operator-id> <authorization-ref> [scope] [verification-ref]\n' "$0" >&2
  exit 2
}
[[ "$SCOPE" == 'infrastructure' || "$SCOPE" == 'feature' ]] || migration_fail 'scope must be infrastructure or feature'

ENV_FILE="$ENV_FILE" validate_config
tool="${MINDFORGE_INFRA_BIN:-${ROOT_DIR}/tools/mindforge-infra}"
[[ -x "$tool" ]] || migration_fail 'the built mindforge-infra binary is required'
preflight_args=(
  preflight
  --config "$ENV_FILE"
  --source-root "$REPOSITORY_ROOT"
  --target "$TARGET"
  --operator-id "$OPERATOR_ID"
  --authorization-ref "$AUTHORIZATION_REF"
  --scope "$SCOPE"
)
if [[ "$SCOPE" == 'infrastructure' ]]; then
  preflight_args+=(--allow-infrastructure-bootstrap)
fi
preflight_output="$("$tool" "${preflight_args[@]}")" || migration_fail 'read-only preflight failed'
preflight_ref="$(printf '%s\n' "$preflight_output" | sed -n 's/^preflight-reference=//p')"
[[ -n "$preflight_ref" ]] || migration_fail 'preflight did not return a reference'

"$tool" apply \
  --config "$ENV_FILE" \
  --source-root "$REPOSITORY_ROOT" \
  --target "$TARGET" \
  --operator-id "$OPERATOR_ID" \
  --authorization-ref "$AUTHORIZATION_REF" \
  --scope "$SCOPE" \
  --preflight-ref "$preflight_ref" \
  --verification-ref "$VERIFICATION_REF" || migration_fail 'migration application failed'
