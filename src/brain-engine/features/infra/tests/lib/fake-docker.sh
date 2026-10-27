#!/usr/bin/env bash

make_fake_docker() {
  local directory="$1"
  mkdir -p "$directory"
  cat > "$directory/docker" <<'EOF'
#!/usr/bin/env bash
set -eu
if [[ -n "${FAKE_DOCKER_LOG:-}" ]]; then
  printf '%s\n' "$*" >> "$FAKE_DOCKER_LOG"
fi
if [[ -n "${FAKE_DOCKER_STDIN_LOG:-}" ]]; then
  cat >> "$FAKE_DOCKER_STDIN_LOG"
fi
if [[ -n "${FAKE_DOCKER_FAIL_MATCH:-}" && " $* " == *" ${FAKE_DOCKER_FAIL_MATCH} "* ]]; then
  exit 91
fi
if [[ "${FAKE_DOCKER_FAIL:-}" == "1" ]]; then
  exit 91
fi
if [[ "${1:-}" == 'volume' && "${2:-}" == 'inspect' ]]; then
  if [[ "${FAKE_DOCKER_VOLUME_EXISTS:-}" == '1' ]]; then
    exit 0
  fi
  exit 1
fi
if [[ " $* " == *' ps -q db '* || " $* " == *' ps -q auth '* || " $* " == *' ps -q auth-develop '* || " $* " == *' ps -q auth-main '* || " $* " == *' ps -aq '* ]]; then
  if [[ "${FAKE_DOCKER_EMPTY_PS:-}" != '1' ]]; then
    printf 'fake-container-id\n'
  fi
  exit 0
fi
if [[ " $* " == *' pg_dump '* && -n "${FAKE_DOCKER_DUMP_CONTENT:-}" ]]; then
  printf '%s' "$FAKE_DOCKER_DUMP_CONTENT"
  exit 0
fi
if [[ " $* " == *' SELECT id, sha256 FROM infra.migration_ledger '* ]]; then
  if [[ -n "${FAKE_DOCKER_LEDGER:-}" && -f "${FAKE_DOCKER_LEDGER}" ]]; then
    cat "$FAKE_DOCKER_LEDGER"
  fi
  exit 0
fi
if [[ " $* " == *' SELECT extversion FROM pg_extension '* ]]; then
  printf '%s\n' "${FAKE_DOCKER_VECTOR_VERSION:-0.8.6}"
  exit 0
fi
case "${1:-}" in
  compose)
    if [[ " $* " == *" config "* ]]; then
      exit 0
    fi
    ;;
esac
exit 0
EOF
  chmod +x "$directory/docker"
  cat > "$directory/pg_restore" <<'EOF'
#!/usr/bin/env bash
set -eu
if [[ -n "${FAKE_PG_RESTORE_LOG:-}" ]]; then
  printf '%s\n' "$*" >> "$FAKE_PG_RESTORE_LOG"
fi
if [[ "${FAKE_PG_RESTORE_FAIL:-}" == "1" ]]; then
  exit 92
fi
exit 0
EOF
  chmod +x "$directory/pg_restore"
}
