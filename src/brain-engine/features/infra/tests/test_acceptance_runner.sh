#!/usr/bin/env bash
set -u

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/assert.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fixture.sh"
# shellcheck disable=SC1091
source "${ROOT_DIR}/tests/lib/fake-docker.sh"

RUNNER="${ROOT_DIR}/scripts/acceptance-disposable.sh"
INTEGRATION_RUNNER="${ROOT_DIR}/tests/integration_disposable.sh"
TEST_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-acceptance-runner.XXXXXX")"
trap 'rm -rf "$TEST_TEMP"' EXIT
RUN_ID='accepttest1234'

make_acceptance_env() {
  local path="$1"
  local role="$2"
  local database
  case "$role" in
    src) database="mf_src_${RUN_ID}" ;;
    restore) database="mf_restore_${RUN_ID}" ;;
    replay) database="mf_replay_${RUN_ID}" ;;
    *) return 2 ;;
  esac
  make_valid_env "$path"
  replace_env_value "$path" POSTGRES_DB "$database"
  replace_env_value "$path" POSTGRES_PORT 25432
  replace_env_value "$path" AUTH_PORT 29999
  replace_env_value "$path" GOTRUE_DB_DATABASE_URL "postgres://mf_user:synthetic-postgres-password-123456@db:5432/${database}?sslmode=disable&search_path=auth"
  replace_env_value "$path" GOTRUE_API_EXTERNAL_URL 'http://127.0.0.1:29999'
  cat >> "$path" <<EOF
MF_ENV_CLASS=disposable
MF_DISPOSABLE_TARGET_ID=${RUN_ID}
MF_DISPOSABLE_CONFIRMATION=RESTORE:${RUN_ID}
MF_EXPECTED_VOLUME=mf-accept-${RUN_ID}-${role}_mindforge-postgres-data
GOTRUE_MAILER_AUTOCONFIRM=true
EOF
}

run_preflight() {
  local mode="$1"
  local role="$2"
  local env_file="$3"
  local extra_env="${4:-}"
  local fake_bin="${TEST_TEMP}/bin"
  make_fake_docker "$fake_bin"
  : > "${TEST_TEMP}/docker.log"
  set +e
  PATH="${fake_bin}:$PATH" FAKE_DOCKER_LOG="${TEST_TEMP}/docker.log" FAKE_DOCKER_EMPTY_PS=1 $extra_env bash "$RUNNER" "$mode" "$RUN_ID" "$role" "$env_file" > "${TEST_TEMP}/runner.output" 2>&1
  RUN_STATUS=$?
  set -e
}

test_runner_exists_and_accepts_fresh_loopback_environment() {
  local env_file="${TEST_TEMP}/src.env"
  make_acceptance_env "$env_file" src
  [[ -x "$RUNNER" ]]
  assert_success "$?" "disposable acceptance runner exists"
  [[ -x "$INTEGRATION_RUNNER" ]]
  assert_success "$?" "disposable integration runner exists"
  run_preflight --local-disposable src "$env_file"
  assert_success "$RUN_STATUS" "runner accepts a fresh unique loopback source environment"
}

test_runner_rejects_unsafe_barriers() {
  local env_file

  env_file="${TEST_TEMP}/primary.env"
  make_acceptance_env "$env_file" src
  replace_env_value "$env_file" MF_ENV_CLASS primary
  run_preflight --local-disposable src "$env_file"
  assert_failure "$RUN_STATUS" "runner rejects a primary environment"

  env_file="${TEST_TEMP}/lan.env"
  make_acceptance_env "$env_file" src
  replace_env_value "$env_file" AUTH_BIND_ADDR 0.0.0.0
  replace_env_value "$env_file" ALLOW_LAN_BINDING yes
  run_preflight --local-disposable src "$env_file"
  assert_failure "$RUN_STATUS" "runner rejects a non-loopback bind even with config opt-in"

  env_file="${TEST_TEMP}/existing-volume.env"
  make_acceptance_env "$env_file" src
  run_preflight --local-disposable src "$env_file" 'FAKE_DOCKER_VOLUME_EXISTS=1'
  assert_failure "$RUN_STATUS" "runner rejects a reused disposable volume"

  env_file="${TEST_TEMP}/wrong-target.env"
  make_acceptance_env "$env_file" restore
  replace_env_value "$env_file" MF_DISPOSABLE_TARGET_ID another-target
  run_preflight --local-disposable restore "$env_file"
  assert_failure "$RUN_STATUS" "runner rejects a target ID divergent from the run ID"

  env_file="${TEST_TEMP}/vm-before-local.env"
  make_acceptance_env "$env_file" src
  run_preflight --vm-disposable src "$env_file"
  assert_failure "$RUN_STATUS" "runner rejects VM mode before local greens"
}

test_vm_runner_requires_an_explicit_owner_identity() {
  local integration_source
  integration_source="$(cat "$INTEGRATION_RUNNER")"
  assert_contains "$integration_source" 'MF_VM_SSH_HOST' "VM runner requires an explicit SSH host"
  assert_contains "$integration_source" 'MF_VM_SSH_USER' "VM runner requires an explicit SSH user"
  assert_contains "$integration_source" 'MF_VM_SSH_PORT' "VM runner accepts an optional SSH port"
  assert_contains "$integration_source" "vm_ssh_port='22'" "VM runner defaults the SSH port to 22 only when unset"
  assert_contains "$integration_source" 'MF_VM_SSH_IDENTITY_FILE' "VM runner requires an explicit owner SSH identity path"
  assert_contains "$integration_source" 'IdentitiesOnly=yes' "VM runner prevents fallback to unintended SSH identities"
  assert_contains "$integration_source" 'BatchMode=yes' "VM runner prevents interactive SSH fallback"
  assert_contains "$integration_source" 'MF_LOCAL_ACCEPTANCE_GREEN' "VM runner preserves the local-greens gate"
  assert_contains "$integration_source" 'numeric TCP port from 1 to 65535' "VM runner rejects unsafe SSH ports"
  assert_contains "$integration_source" '--no-xattrs' "VM runner excludes host extended attributes from the portable package"
  assert_contains "$integration_source" '--no-mac-metadata' "VM runner excludes macOS metadata sidecars from the portable package"
}

test_vm_runner_has_no_fixed_target_or_shell_loading() {
  local integration_source
  integration_source="$(cat "$INTEGRATION_RUNNER")"
  assert_not_contains "$integration_source" 'fixed-vm.example.invalid' "VM runner has no fixed SSH host"
  assert_not_contains "$integration_source" 'fixed-user@' "VM runner has no fixed SSH user"
  assert_not_contains "$integration_source" '-p 65530' "VM runner has no fixed SSH port"
  assert_not_contains "$integration_source" 'eval ' "VM runner does not evaluate shell input"
  assert_not_contains "$integration_source" 'source ' "VM runner does not load shell input"
}

case "${1:-all}" in
  acceptance-runner|all)
    test_runner_exists_and_accepts_fresh_loopback_environment
    test_runner_rejects_unsafe_barriers
    test_vm_runner_requires_an_explicit_owner_identity
    test_vm_runner_has_no_fixed_target_or_shell_loading
    ;;
  *)
    printf 'unknown acceptance runner test selection: %s\n' "$1" >&2
    exit 2
    ;;
esac

finish_tests
