#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-}"
SELECTION="${2:-all}"

case "$MODE" in
  static)
    case "$SELECTION" in
      config|compose|all)
        if [[ "$SELECTION" == 'all' ]]; then
          bash "${ROOT_DIR}/tests/test_config_compose.sh" all
          bash "${ROOT_DIR}/tests/test_runbook_hygiene.sh" runbook-hygiene
        else
          bash "${ROOT_DIR}/tests/test_config_compose.sh" "$SELECTION"
        fi
        ;;
      runbook-hygiene)
        bash "${ROOT_DIR}/tests/test_runbook_hygiene.sh" runbook-hygiene
        ;;
      *)
        printf 'unknown static selection: %s\n' "$SELECTION" >&2
        exit 2
        ;;
    esac
    ;;
  unit)
    case "$SELECTION" in
      migrations)
        bash "${ROOT_DIR}/tests/test_migrations.sh" migrations
        ;;
      operations)
        bash "${ROOT_DIR}/tests/test_operations.sh" operations
        ;;
      acceptance-runner)
        bash "${ROOT_DIR}/tests/test_acceptance_runner.sh" acceptance-runner
        ;;
      all)
        bash "${ROOT_DIR}/tests/test_migrations.sh" migrations
        bash "${ROOT_DIR}/tests/test_operations.sh" operations
        bash "${ROOT_DIR}/tests/test_acceptance_runner.sh" acceptance-runner
        ;;
      *)
        printf 'unknown unit selection: %s\n' "$SELECTION" >&2
        exit 2
        ;;
    esac
    ;;
  integration)
    case "$SELECTION" in
      --local-disposable|--vm-disposable)
        bash "${ROOT_DIR}/tests/integration_disposable.sh" "$SELECTION"
        ;;
      *)
        printf 'unknown integration selection: %s\n' "$SELECTION" >&2
        exit 2
        ;;
    esac
    ;;
  *)
    printf 'usage: %s {static|unit} selection\n' "$0" >&2
    exit 2
    ;;
esac
