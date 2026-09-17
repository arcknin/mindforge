#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fixture.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fake-docker.sh"

BACKUP_SCRIPT="${ROOT_DIR}/scripts/backup.sh"
RESTORE_SCRIPT="${ROOT_DIR}/scripts/restore.sh"
INSPECT_SCRIPT="${ROOT_DIR}/scripts/inspect.sh"
TEST_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-operations.XXXXXX")"
trap 'rm -rf "$TEST_TEMP"' EXIT

make_disposable_env() {
  local path="$1"
  make_valid_env "$path"
  cat >> "$path" <<'EOF'
MF_DISPOSABLE_TARGET_ID=opstest1234
MF_DISPOSABLE_CONFIRMATION=RESTORE:opstest1234
MF_EXPECTED_VOLUME=mf-accept-opstest1234-restore_mindforge-postgres-data
EOF
  replace_env_value "$path" POSTGRES_DB "mf_restore_opstest1234"
  replace_env_value "$path" GOTRUE_DB_DATABASE_URL "postgres://mf_user:synthetic-postgres-password-123456@db:5432/mf_restore_opstest1234?sslmode=disable&search_path=auth"
}

run_with_fake_docker() {
  local output_file="$1"
  shift
  local fake_bin="${TEST_TEMP}/bin"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/docker.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" "$@" > "$output_file" 2>&1
  RUN_STATUS=$?
  set -e
}

file_mode() {
  if stat -f '%Lp' "$1" >/dev/null 2>&1; then
    stat -f '%Lp' "$1"
  else
    stat -c '%a' "$1"
  fi
}

test_backup_is_external_private_and_atomic() {
  local env_file output_dir output
  env_file="${TEST_TEMP}/backup.env"
  output_dir="${TEST_TEMP}/external-backups"
  mkdir -p "$output_dir"
  make_valid_env "$env_file"

  run_with_fake_docker "${TEST_TEMP}/backup.output" env FAKE_DOCKER_DUMP_CONTENT='synthetic-dump' "$BACKUP_SCRIPT" "$env_file" mf-backup-test "$output_dir"
  assert_success "$RUN_STATUS" "backup accepts an explicit external destination with fake Docker"
  output="$(find "$output_dir" -maxdepth 1 -type f -name '*.dump' -print)"
  [[ -n "$output" ]]
  assert_success "$?" "backup promotes a final dump only after producer success"
  if [[ -n "$output" ]]; then
    [[ "$(file_mode "$output")" == '600' ]]
    assert_success "$?" "backup dump is private to its owner"
  fi

  run_with_fake_docker "${TEST_TEMP}/backup-inside.output" "$BACKUP_SCRIPT" "$env_file" mf-backup-test "$ROOT_DIR"
  assert_failure "$RUN_STATUS" "backup rejects a destination inside the checkout"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.log")" "exec" "rejected backup reaches no Docker mutation"

  rm -f "$output"
  run_with_fake_docker "${TEST_TEMP}/backup-fail.output" env FAKE_DOCKER_FAIL_MATCH='pg_dump' "$BACKUP_SCRIPT" "$env_file" mf-backup-test "$output_dir"
  assert_failure "$RUN_STATUS" "failed backup producer returns non-zero"
  [[ -z "$(find "$output_dir" -maxdepth 1 -type f -print)" ]]
  assert_success "$?" "failed backup leaves no final or partial artifact"
}

test_backup_rejects_nonempty_invalid_dump_before_promotion() {
  local env_file output_dir restore_log restore_log_contents
  env_file="${TEST_TEMP}/backup-invalid.env"
  output_dir="${TEST_TEMP}/external-invalid-backups"
  restore_log="${TEST_TEMP}/pg-restore.log"
  mkdir -p "$output_dir"
  make_valid_env "$env_file"

  run_with_fake_docker "${TEST_TEMP}/backup-invalid.output" env \
    FAKE_DOCKER_DUMP_CONTENT='not-a-postgresql-archive' \
    FAKE_PG_RESTORE_FAIL=1 \
    FAKE_PG_RESTORE_LOG="$restore_log" \
    "$BACKUP_SCRIPT" "$env_file" mf-backup-invalid-test "$output_dir"
  assert_failure "$RUN_STATUS" "backup rejects a non-empty dump whose pg_restore preflight fails"
  restore_log_contents=''
  [[ -f "$restore_log" ]] && restore_log_contents="$(cat "$restore_log")"
  assert_contains "$restore_log_contents" "--list" "backup validates the temporary dump with pg_restore --list"
  [[ -z "$(find "$output_dir" -maxdepth 1 -type f -print)" ]]
  assert_success "$?" "invalid dump leaves no promoted or partial artifact"
}

test_restore_refuses_unsafe_targets_before_mutation() {
  local env_file dump_file
  env_file="${TEST_TEMP}/restore.env"
  dump_file="${TEST_TEMP}/synthetic.dump"
  make_disposable_env "$env_file"
  printf 'synthetic dump' > "$dump_file"

  run_with_fake_docker "${TEST_TEMP}/restore-wrong-target.output" "$RESTORE_SCRIPT" "$dump_file" "$env_file" mf-accept-opstest1234-restore wrong-target 'RESTORE:wrong-target'
  assert_failure "$RUN_STATUS" "restore rejects a target ID that diverges from its disposable environment"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.log")" " up " "rejected restore does not start Docker services"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.log")" " exec " "rejected restore does not mutate Docker services"

  run_with_fake_docker "${TEST_TEMP}/restore-confirmation.output" "$RESTORE_SCRIPT" "$dump_file" "$env_file" mf-accept-opstest1234-restore opstest1234 'wrong-confirmation'
  assert_failure "$RUN_STATUS" "restore requires exact target-bound confirmation"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.log")" " up " "wrong confirmation reaches no Docker startup"

  run_with_fake_docker "${TEST_TEMP}/restore-bad-dump.output" env FAKE_PG_RESTORE_FAIL=1 "$RESTORE_SCRIPT" "$dump_file" "$env_file" mf-accept-opstest1234-restore opstest1234 'RESTORE:opstest1234'
  assert_failure "$RUN_STATUS" "restore rejects a dump that fails host-side preflight"
  assert_not_contains "$(cat "${TEST_TEMP}/docker.log")" " up " "invalid dump reaches no Docker startup"

  local fake_bin status log
  fake_bin="${TEST_TEMP}/restore-bin"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/docker.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" FAKE_DOCKER_EMPTY_PS=1 bash "$RESTORE_SCRIPT" "$dump_file" "$env_file" mf-accept-opstest1234-restore opstest1234 'RESTORE:opstest1234' > "${TEST_TEMP}/restore-good.output" 2>&1
  status=$?
  set -e
  assert_success "$status" "restore reaches only its fresh exclusive target with fake Docker"
  log="$(cat "${TEST_TEMP}/docker.log")"
  assert_contains "$log" "up -d db" "valid restore starts only the disposable database"
  assert_contains "$log" "--single-transaction" "restore uses a single PostgreSQL transaction"
  assert_not_contains "$log" "synthetic-postgres-password-123456" "restore does not put database credentials in Docker arguments"
}

test_inspect_accumulates_reds() {
  local env_file fake_bin
  env_file="${TEST_TEMP}/inspect.env"
  fake_bin="${TEST_TEMP}/inspect-bin"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/docker.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" FAKE_DOCKER_FAIL_MATCH='pg_isready' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-test > "${TEST_TEMP}/inspect.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_failure "$RUN_STATUS" "inspect fails closed when PostgreSQL readiness is red"
}

test_inspect_requires_a_usable_gotrue_catalog() {
  local inspect_source
  inspect_source="$(cat "$INSPECT_SCRIPT")"
  assert_contains "$inspect_source" "to_regclass('auth.identities')" "inspect requires the GoTrue identities catalog relation"
  assert_contains "$inspect_source" "count(*) FROM auth.schema_migrations" "inspect requires an applied GoTrue migration marker"
}

test_inspect_checks_vector_compatibility_range() {
  local env_file fake_bin
  env_file="${TEST_TEMP}/inspect-vector.env"
  fake_bin="${TEST_TEMP}/inspect-vector-bin"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"

  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_VECTOR_VERSION='0.8.0' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-vector-test > "${TEST_TEMP}/inspect-vector-080.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_success "$RUN_STATUS" "inspect accepts vector 0.8.0"

  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_VECTOR_VERSION='0.8.6' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-vector-test > "${TEST_TEMP}/inspect-vector-086.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_success "$RUN_STATUS" "inspect accepts vector 0.8.6"

  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_VECTOR_VERSION='0.7.99' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-vector-test > "${TEST_TEMP}/inspect-vector-0799.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_failure "$RUN_STATUS" "inspect rejects vector below 0.8.0"

  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_VECTOR_VERSION='1.0.0' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-vector-test > "${TEST_TEMP}/inspect-vector-100.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_failure "$RUN_STATUS" "inspect rejects vector 1.0.0"

  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_VECTOR_VERSION='malformed' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-vector-test > "${TEST_TEMP}/inspect-vector-malformed.output" 2>&1
  RUN_STATUS=$?
  set -e
  assert_failure "$RUN_STATUS" "inspect rejects a malformed vector version"
}

test_malicious_dotenv_is_never_executed() {
  local env_file marker script output
  marker="${TEST_TEMP}/dotenv-executed"
  env_file="${TEST_TEMP}/malicious.env"
  make_disposable_env "$env_file"
  printf 'MALICIOUS=$(touch %s)\n' "$marker" >> "$env_file"

  for script in "$BACKUP_SCRIPT" "$RESTORE_SCRIPT" "$INSPECT_SCRIPT"; do
    rm -f "$marker"
    case "$(basename "$script")" in
      restore.sh)
        run_with_fake_docker "${TEST_TEMP}/malicious.output" bash "$script" "${TEST_TEMP}/synthetic.dump" "$env_file" mf-accept-opstest1234-restore opstest1234 'RESTORE:opstest1234'
        ;;
      *)
        run_with_fake_docker "${TEST_TEMP}/malicious.output" bash "$script" "$env_file" mf-accept-opstest1234-restore "${TEST_TEMP}/synthetic.dump" opstest1234 'RESTORE:opstest1234'
        ;;
    esac
    [[ ! -e "$marker" ]]
    assert_success "$?" "$(basename "$script") does not execute dotenv content"
  done
}

case "${1:-all}" in
  operations|all)
    test_backup_is_external_private_and_atomic
    test_backup_rejects_nonempty_invalid_dump_before_promotion
    test_restore_refuses_unsafe_targets_before_mutation
    test_inspect_accumulates_reds
    test_inspect_requires_a_usable_gotrue_catalog
    test_inspect_checks_vector_compatibility_range
    test_malicious_dotenv_is_never_executed
    ;;
  *)
    printf 'unknown operations test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
