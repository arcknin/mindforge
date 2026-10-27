#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${ROOT_DIR}/../../../.." && pwd)"
ENV_FILE="${1:-}"
TARGET="${2:-}"
OPERATOR_ID="${3:-}"
AUTHORIZATION_REF="${4:-}"
OUTPUT_DIR="${5:-}"

# shellcheck disable=SC1091
source "${ROOT_DIR}/scripts/validate-config.sh"

backup_fail() {
  printf 'Backup error: %s\n' "$1" >&2
  exit 1
}

[[ -n "$ENV_FILE" && -n "$TARGET" && -n "$OPERATOR_ID" && -n "$AUTHORIZATION_REF" && -n "$OUTPUT_DIR" ]] || {
  printf 'usage: %s <env-file> <target> <operator-id> <authorization-ref> <external-backup-dir>\n' "$0" >&2
  exit 2
}

ENV_FILE="$ENV_FILE" validate_config || backup_fail 'configuration validation failed'
tool="${MINDFORGE_INFRA_BIN:-${ROOT_DIR}/tools/mindforge-infra}"
[[ -x "$tool" ]] || backup_fail 'the built mindforge-infra binary is required'
"$tool" backup \
  --config "$ENV_FILE" \
  --source-root "$REPO_ROOT" \
  --target "$TARGET" \
  --operator-id "$OPERATOR_ID" \
  --authorization-ref "$AUTHORIZATION_REF" \
  --output-dir "$OUTPUT_DIR" || backup_fail 'backup operation failed'
