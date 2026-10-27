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
MIGRATE_SCRIPT="${ROOT_DIR}/scripts/migrate.sh"
UP_SCRIPT="${ROOT_DIR}/scripts/up.sh"
TEST_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-operations.XXXXXX")"
trap 'rm -rf "$TEST_TEMP"' EXIT

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

make_fake_infra_tool() {
  local path="$1"
  cat > "$path" <<'EOF'
#!/usr/bin/env bash
set -eu
if [[ -n "${FAKE_INFRA_LOG:-}" ]]; then
  printf '%s\n' "$*" >> "$FAKE_INFRA_LOG"
fi
if [[ "${1:-}" == 'preflight' ]]; then
  target=''
  while (($#)); do
    if [[ "$1" == '--target' && $# -ge 2 ]]; then
      target="$2"
      shift 2
    else
      shift
    fi
  done
  if [[ "$target" == 'mindforge_develop' && "${FAKE_INFRA_DEVELOP_PREFLIGHT_FAIL:-}" == '1' ]]; then
    exit 3
  fi
  if [[ "$target" == 'mindforge' && "${FAKE_INFRA_MAIN_PREFLIGHT_FAIL:-}" == '1' ]]; then
    exit 3
  fi
  printf 'preflight-reference=preflight/%s/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n' "$target"
  exit 0
fi
if [[ "${1:-}" == 'apply' || "${1:-}" == 'backup' || "${1:-}" == 'restore' || "${1:-}" == 'promote' || "${1:-}" == 'provision-develop' ]]; then
  exit 0
fi
exit 2
EOF
  chmod +x "$path"
}

test_b03_scripts_are_explicit_binary_adapters() {
  local script source
  for script in "$ROOT_DIR/scripts/up.sh" "$ROOT_DIR/scripts/migrate.sh" "$ROOT_DIR/scripts/inspect.sh"; do
    source="$(cat "$script")"
    if [[ "$source" == *'infra_tool'* || "$source" == *'MINDFORGE_INFRA_BIN'* ]]; then
      pass "$(basename "$script") resolves the distributable binary"
    else
      fail "$(basename "$script") resolves the distributable binary"
    fi
    assert_contains "$source" 'preflight' "$(basename "$script") performs an explicit preflight"
    assert_not_contains "$source" 'CREATE TABLE' "$(basename "$script") does not own database DDL"
  done
  assert_contains "$(cat "$ROOT_DIR/scripts/up.sh")" 'auth-develop-migrate' 'startup runs the develop Auth migrator explicitly'
  assert_contains "$(cat "$ROOT_DIR/scripts/up.sh")" 'auth-migrate' 'startup runs the retained main Auth migrator explicitly'
}

test_b03_main_target_and_provision_adapter_surface() {
  local up_source inspect_source provision_source
  up_source="$(cat "$UP_SCRIPT")"
  inspect_source="$(cat "$INSPECT_SCRIPT")"
  provision_source="$(cat "$ROOT_DIR/scripts/provision-develop.sh" 2>/dev/null || true)"
  assert_contains "$up_source" 'run_target_infrastructure mindforge' 'startup selects the physical main database name'
  assert_contains "$up_source" 'auth-migrate' 'startup uses the retained main Auth migrator service'
  assert_contains "$up_source" 'up -d --no-deps auth' 'startup uses the retained main Auth service'
  assert_not_contains "$up_source" 'mindforge_main' 'startup has no superseded physical main target'
  assert_contains "$inspect_source" 'check_target mindforge auth' 'inspection selects physical main and retained Auth service'
  assert_not_contains "$inspect_source" 'mindforge_main' 'inspection has no superseded physical main target'
  assert_contains "$provision_source" 'provision-develop' 'explicit develop provisioning adapter exists'
  assert_contains "$provision_source" '--target mindforge_develop' 'provision adapter fixes its operation target'
  assert_contains "$provision_source" '--operator-id' 'provision adapter requires operator authority'
  assert_contains "$provision_source" '--authorization-ref' 'provision adapter requires authorization reference'
  assert_not_contains "$provision_source" 'compose_cmd' 'provision adapter does not invoke Compose'
  assert_not_contains "$provision_source" 'psql' 'provision adapter does not invoke PostgreSQL directly'
  assert_contains "$(cat "$ROOT_DIR/tests/versioned-surface.txt")" 'scripts/provision-develop.sh' 'versioned surface registers develop provisioning adapter'
}

test_b03_startup_orders_each_target_and_auth_pair() {
  local env_file fake_bin tool log status
  env_file="${TEST_TEMP}/b03-startup.env"
  fake_bin="${TEST_TEMP}/b03-startup-bin"
  tool="${TEST_TEMP}/mindforge-infra"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  : > "${TEST_TEMP}/b03-startup-infra.log"
  : > "${TEST_TEMP}/b03-startup-docker.log"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_DOCKER_VOLUME_EXISTS=1 FAKE_DOCKER_LOG="${TEST_TEMP}/b03-startup-docker.log" FAKE_INFRA_LOG="${TEST_TEMP}/b03-startup-infra.log" bash "$UP_SCRIPT" "$env_file" mf-startup-test operator-1 owner-2026-09-19 > "${TEST_TEMP}/b03-startup.output" 2>&1
  status=$?
  set -e
  assert_success "$status" 'B03 startup accepts an already complete initialized cluster'
  log="$(cat "${TEST_TEMP}/b03-startup-docker.log")"
  assert_not_contains "$log" 'up -d db' 'startup does not restart PostgreSQL on an already initialized cluster'
  assert_contains "$log" 'run --rm auth-develop-migrate' 'startup migrates develop Auth before serving it'
  assert_contains "$log" 'up -d --no-deps auth-develop' 'startup serves develop Auth after its migration'
  assert_contains "$log" 'run --rm auth-migrate' 'startup migrates main Auth before serving it'
  assert_contains "$log" 'up -d --no-deps auth' 'startup serves main Auth after its migration'
  assert_contains "$(cat "${TEST_TEMP}/b03-startup-infra.log")" '--target mindforge' 'startup preflights the physical main database'
  assert_not_contains "$(cat "${TEST_TEMP}/b03-startup-infra.log")" 'provision-develop' 'startup never invokes additive provisioning'
}

test_b03_missing_develop_keeps_existing_main_and_postgres_read_only() {
  local env_file fake_bin tool log output status
  env_file="${TEST_TEMP}/existing-missing-develop.env"
  fake_bin="${TEST_TEMP}/existing-missing-develop-bin"
  tool="${TEST_TEMP}/existing-missing-develop-tool"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_DOCKER_VOLUME_EXISTS=1 FAKE_DOCKER_LOG="${TEST_TEMP}/existing-missing-develop-docker.log" FAKE_INFRA_LOG="${TEST_TEMP}/existing-missing-develop-infra.log" FAKE_INFRA_DEVELOP_PREFLIGHT_FAIL=1 bash "$UP_SCRIPT" "$env_file" infra operator-1 owner-2026-09-23 > "${TEST_TEMP}/existing-missing-develop.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'startup reports an incomplete existing develop target'
  log="$(cat "${TEST_TEMP}/existing-missing-develop-docker.log")"
  output="$(cat "${TEST_TEMP}/existing-missing-develop.output")"
  assert_contains "$output" 'develop' 'startup reports the develop condition'
  assert_contains "$log" 'ps -q auth' 'startup checks existing main Auth presence read-only'
  assert_contains "$log" 'exec -T auth wget' 'startup checks existing main Auth health read-only'
  assert_not_contains "$log" 'up -d db' 'incomplete develop does not start or restart PostgreSQL'
  assert_not_contains "$log" 'run --rm auth-migrate' 'incomplete develop does not run main Auth migration'
  assert_not_contains "$log" 'up -d --no-deps auth' 'incomplete develop does not mutate main Auth lifecycle'
  assert_not_contains "$(cat "${TEST_TEMP}/existing-missing-develop-infra.log")" '--target mindforge ' 'incomplete develop does not preflight or mutate main'
  assert_not_contains "$(cat "${TEST_TEMP}/existing-missing-develop-infra.log")" 'provision-develop' 'startup never provisions develop implicitly'
}

test_b03_unknown_existing_topology_leaves_stack_untouched() {
  local env_file fake_bin tool log output status
  env_file="${TEST_TEMP}/existing-unknown.env"
  fake_bin="${TEST_TEMP}/existing-unknown-bin"
  tool="${TEST_TEMP}/existing-unknown-tool"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_DOCKER_VOLUME_EXISTS=1 FAKE_DOCKER_EMPTY_PS=1 FAKE_DOCKER_LOG="${TEST_TEMP}/existing-unknown-docker.log" bash "$UP_SCRIPT" "$env_file" infra operator-1 owner-2026-09-23 > "${TEST_TEMP}/existing-unknown.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'startup reports unknown topology for a stopped existing cluster'
  log="$(cat "${TEST_TEMP}/existing-unknown-docker.log")"
  output="$(cat "${TEST_TEMP}/existing-unknown.output")"
  assert_contains "$output" 'unknown' 'startup reports topology as unknown'
  assert_contains "$log" 'ps -q auth' 'unknown topology still checks main Auth presence'
  assert_not_contains "$log" 'up -d db' 'unknown topology does not start PostgreSQL'
  assert_not_contains "$log" 'run --rm auth-migrate' 'unknown topology does not migrate main Auth'
  assert_not_contains "$log" 'up -d --no-deps auth' 'unknown topology does not mutate main Auth lifecycle'
}

test_b03_fresh_cluster_retains_separate_bootstrap_path() {
  local env_file fake_bin tool log status
  env_file="${TEST_TEMP}/fresh-cluster.env"
  fake_bin="${TEST_TEMP}/fresh-cluster-bin"
  tool="${TEST_TEMP}/fresh-cluster-tool"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_DOCKER_EMPTY_PS=1 FAKE_DOCKER_LOG="${TEST_TEMP}/fresh-cluster-docker.log" FAKE_INFRA_LOG="${TEST_TEMP}/fresh-cluster-infra.log" bash "$UP_SCRIPT" "$env_file" infra operator-1 owner-2026-09-23 > "${TEST_TEMP}/fresh-cluster.output" 2>&1
  status=$?
  set -e
  assert_success "$status" 'fresh cluster still uses the explicit initdb bootstrap path'
  log="$(cat "${TEST_TEMP}/fresh-cluster-docker.log")"
  assert_contains "$log" 'up -d db' 'fresh cluster starts PostgreSQL for initdb bootstrap'
  assert_not_contains "$(cat "${TEST_TEMP}/fresh-cluster-infra.log")" 'provision-develop' 'fresh startup uses bootstrap rather than additive provisioning'
}

test_b03_migrate_and_inspect_require_explicit_targets() {
  local env_file fake_bin tool status
  env_file="${TEST_TEMP}/b03-explicit.env"
  fake_bin="${TEST_TEMP}/b03-explicit-bin"
  tool="${TEST_TEMP}/mindforge-infra"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"

  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" bash "$ROOT_DIR/scripts/inspect.sh" "$env_file" mf-inspect-test > "${TEST_TEMP}/b03-inspect-missing.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'inspection rejects omitted operator and authorization references'

  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/b03-migrate.log" bash "$MIGRATE_SCRIPT" "$env_file" mindforge operator-1 owner-2026-09-19 feature verification/feature-1 > "${TEST_TEMP}/b03-migrate.output" 2>&1
  status=$?
  set -e
  assert_success "$status" 'migration adapter forwards an explicit target and scope'
  assert_contains "$(cat "${TEST_TEMP}/b03-migrate.log")" '--target mindforge' 'migration adapter forwards the physical main target'
}

test_b03_provision_adapter_requires_explicit_authority_and_calls_only_binary() {
  local env_file fake_bin tool status log
  env_file="${TEST_TEMP}/provision-adapter.env"
  fake_bin="${TEST_TEMP}/provision-adapter-bin"
  tool="${TEST_TEMP}/provision-adapter-tool"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  : > "${TEST_TEMP}/provision-adapter-docker.log"
  : > "${TEST_TEMP}/provision-adapter.log"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/provision-adapter.log" FAKE_DOCKER_LOG="${TEST_TEMP}/provision-adapter-docker.log" bash "$ROOT_DIR/scripts/provision-develop.sh" "$env_file" mindforge_develop operator-1 owner-provision-2026-09-23 > "${TEST_TEMP}/provision-adapter.output" 2>&1
  status=$?
  set -e
  assert_success "$status" 'explicit provision adapter delegates with complete authority'
  log="$(cat "${TEST_TEMP}/provision-adapter.log")"
  assert_contains "$log" 'provision-develop' 'provision adapter calls only the explicit binary operation'
  assert_contains "$log" '--target mindforge_develop' 'provision adapter forwards fixed develop target'
  assert_contains "$log" '--operator-id operator-1' 'provision adapter forwards operator identity'
  assert_contains "$log" '--authorization-ref owner-provision-2026-09-23' 'provision adapter forwards authorization reference'
  assert_not_contains "$(cat "${TEST_TEMP}/provision-adapter-docker.log")" 'compose' 'provision adapter never invokes Compose'

  : > "${TEST_TEMP}/provision-adapter.log"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/provision-adapter.log" bash "$ROOT_DIR/scripts/provision-develop.sh" "$env_file" mindforge operator-1 owner-provision-2026-09-23 > "${TEST_TEMP}/provision-adapter-wrong-target.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'provision adapter rejects main target'
  [[ ! -s "${TEST_TEMP}/provision-adapter.log" ]]
  assert_success "$?" 'invalid provisioning target reaches no binary operation'
}

test_b04_backup_restore_are_binary_adapters_without_compose() {
  local script source
  for script in "$ROOT_DIR/scripts/migrate.sh" "$ROOT_DIR/scripts/backup.sh" "$ROOT_DIR/scripts/restore.sh"; do
    source="$(cat "$script")"
    assert_contains "$source" 'MINDFORGE_INFRA_BIN' "$(basename "$script") resolves the distributable binary"
    assert_not_contains "$source" 'compose_cmd' "$(basename "$script") does not invoke Compose"
    assert_not_contains "$source" 'compose_init' "$(basename "$script") does not initialize Compose"
    assert_not_contains "$source" 'pg_dump' "$(basename "$script") does not invoke pg_dump directly"
    assert_not_contains "$source" 'pg_restore' "$(basename "$script") does not invoke pg_restore directly"
    assert_not_contains "$source" 'psql' "$(basename "$script") does not invoke SQL directly"
  done
}

test_binary_adapters_fail_before_the_tool_when_inputs_are_missing() {
  local env_file tool status
  env_file="${TEST_TEMP}/missing-inputs.env"
  tool="${TEST_TEMP}/missing-inputs-tool"
  make_valid_env "$env_file"
  make_fake_infra_tool "$tool"
  : > "${TEST_TEMP}/missing-inputs.log"

  set +e
  MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/missing-inputs.log" bash "$MIGRATE_SCRIPT" "$env_file" > "${TEST_TEMP}/missing-migrate.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'migration adapter rejects omitted target and authority'

  set +e
  MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/missing-inputs.log" bash "$BACKUP_SCRIPT" "$env_file" > "${TEST_TEMP}/missing-backup.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'backup adapter rejects omitted target and output'

  set +e
  MINDFORGE_INFRA_BIN="$tool" FAKE_INFRA_LOG="${TEST_TEMP}/missing-inputs.log" bash "$RESTORE_SCRIPT" > "${TEST_TEMP}/missing-restore.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" 'restore adapter rejects omitted target and authority'
  [[ ! -s "${TEST_TEMP}/missing-inputs.log" ]]
  assert_success "$?" 'missing adapter inputs reach no binary operation'
}

test_b03_inspect_accumulates_reds() {
  local env_file fake_bin tool status
  env_file="${TEST_TEMP}/inspect.env"
  fake_bin="${TEST_TEMP}/inspect-bin"
  tool="${TEST_TEMP}/inspect-tool"
  make_valid_env "$env_file"
  make_fake_docker "$fake_bin"
  make_fake_infra_tool "$tool"
  set +e
  PATH="${fake_bin}:$PATH" MINDFORGE_INFRA_BIN="$tool" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" FAKE_DOCKER_FAIL_MATCH='pg_isready' bash "$INSPECT_SCRIPT" "$env_file" mf-inspect-test operator-1 owner-inspect-2026-09-19 > "${TEST_TEMP}/inspect.output" 2>&1
  status=$?
  set -e
  assert_failure "$status" "inspect fails closed when PostgreSQL readiness is red"
}

test_b03_inspect_requires_a_usable_gotrue_catalog() {
  local operations_source
  operations_source="$(cat "$ROOT_DIR/tools/internal/operations/operations.go")"
  assert_contains "$operations_source" "to_regclass('auth.identities')" "preflight requires the GoTrue identities catalog relation"
  assert_contains "$operations_source" "count(*) AS applied FROM auth.schema_migrations" "preflight reads an applied GoTrue migration marker without binding an absent relation"
  assert_contains "$operations_source" 'schemaCount < 1' "preflight rejects an empty GoTrue migration history"
}

test_b03_startup_uses_pinned_vector_compatibility_range() {
  local inspect_source
  inspect_source="$(cat "$INSPECT_SCRIPT")"
  assert_contains "$inspect_source" 'minor >= 8' 'inspection accepts the minimum supported vector minor'
  assert_contains "$inspect_source" 'major == 0' 'inspection rejects vector major changes'
}

case "${1:-all}" in
  operations|all)
    test_b03_scripts_are_explicit_binary_adapters
    test_b03_main_target_and_provision_adapter_surface
    test_b03_startup_orders_each_target_and_auth_pair
    test_b03_missing_develop_keeps_existing_main_and_postgres_read_only
    test_b03_unknown_existing_topology_leaves_stack_untouched
    test_b03_fresh_cluster_retains_separate_bootstrap_path
    test_b03_migrate_and_inspect_require_explicit_targets
    test_b03_provision_adapter_requires_explicit_authority_and_calls_only_binary
    test_b04_backup_restore_are_binary_adapters_without_compose
    test_binary_adapters_fail_before_the_tool_when_inputs_are_missing
    test_b03_inspect_accumulates_reds
    test_b03_inspect_requires_a_usable_gotrue_catalog
    test_b03_startup_uses_pinned_vector_compatibility_range
    ;;
  *)
    printf 'unknown operations test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
