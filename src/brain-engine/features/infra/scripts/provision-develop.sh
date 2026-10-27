#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${1:-}"
TARGET="${2:-}"
OPERATOR_ID="${3:-}"
AUTHORIZATION_REF="${4:-}"

provision_fail() {
  printf 'Develop provisioning error: %s\n' "$1" >&2
  exit 1
}

[[ -n "$ENV_FILE" && -n "$TARGET" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" ]] || {
  printf 'usage: %s <env-file> mindforge_develop <operator-id> <authorization-ref>\n' "$0" >&2
  exit 2
}
[[ "$TARGET" == 'mindforge_develop' ]] || provision_fail 'target must be mindforge_develop'

bash "${ROOT_DIR}/scripts/validate-config.sh" "$ENV_FILE" || provision_fail 'configuration validation failed'
tool="${MINDFORGE_INFRA_BIN:-${ROOT_DIR}/tools/mindforge-infra}"
[[ -x "$tool" ]] || provision_fail 'the built mindforge-infra binary is required'

"$tool" provision-develop \
  --config "$ENV_FILE" \
  --target mindforge_develop \
  --operator-id "$OPERATOR_ID" \
  --authorization-ref "$AUTHORIZATION_REF" || provision_fail 'additive develop provisioning failed'
