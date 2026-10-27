#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"
BACKUP_FILE="${1:-}"
MANIFEST_FILE="${2:-}"
ENV_FILE="${3:-}"
TARGET="${4:-}"
OPERATOR_ID="${5:-}"
AUTHORIZATION_REF="${6:-}"
REPLACEMENT_CONFIRMATION="${7:-}"
EVIDENCE_DIR="${8:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"

restore_fail() {
  printf 'Restore error: %s\n' "$1" >&2
  exit 1
}

[[ -n "$BACKUP_FILE" && -n "$MANIFEST_FILE" && -n "$ENV_FILE" && -n "$TARGET" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" && -n "$REPLACEMENT_CONFIRMATION" ]] || {
  printf 'usage: %s <backup-file> <manifest-file> <env-file> <target> <operator-id> <authorization-ref> <replacement-confirmation> [evidence-dir]\n' "$0" >&2
  exit 2
}

ENV_FILE="$ENV_FILE" validate_config || restore_fail 'configuration validation failed'
tool="${MINDFORGE_INFRA_BIN:-${ROOT_DIR}/tools/mindforge-infra}"
[[ -x "$tool" ]] || restore_fail 'the built mindforge-infra binary is required'
restore_args=(
  restore
  --config "$ENV_FILE"
  --source-root "$REPO_ROOT"
  --target "$TARGET"
  --operator-id "$OPERATOR_ID"
  --authorization-ref "$AUTHORIZATION_REF"
  --backup-file "$BACKUP_FILE"
  --manifest-file "$MANIFEST_FILE"
  --replacement-confirmation "$REPLACEMENT_CONFIRMATION"
)
if [[ -n "$EVIDENCE_DIR" ]]; then
  restore_args+=(--evidence-dir "$EVIDENCE_DIR")
fi
"$tool" "${restore_args[@]}" || restore_fail 'restore operation failed'
