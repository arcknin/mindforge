#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODE="${1:-}"
WORK_DIR=''
RUN_ID=''
SRC_ENV=''
RESTORE_ENV=''
REPLAY_ENV=''
SRC_PROJECT=''
RESTORE_PROJECT=''
REPLAY_PROJECT=''

integration_fail() {
  printf 'Disposable integration error: %s\n' "$1" >&2
  exit 1
}

compose_for() {
  local env_file="$1"
  local project_name="$2"
  shift 2
  bash "${ROOT_DIR}/scripts/lib/compose.sh" "$env_file" "$project_name" "$@"
}

random_hex() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 24
  elif command -v uuidgen >/dev/null 2>&1; then
    uuidgen | tr -d '-'
  else
    integration_fail 'openssl or uuidgen is required to create an unpredictable disposable run ID'
  fi
}

pick_loopback_port() {
  command -v python3 >/dev/null 2>&1 || integration_fail 'python3 is required to select a dynamic loopback port'
  python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}

write_disposable_env() {
  local path="$1"
  local role="$2"
  local project_name="$3"
  local database_name="$4"
  local db_port="$5"
  local auth_port="$6"
  local password jwt_secret operator_token
  password="mf$(random_hex)"
  jwt_secret="$(random_hex)$(random_hex)"
  operator_token="$(random_hex)$(random_hex)"
  umask 077
  cat > "$path" <<EOF
POSTGRES_DB=${database_name}
POSTGRES_USER=mf_accept
POSTGRES_PASSWORD=${password}
POSTGRES_HOST=db
POSTGRES_PORT=${db_port}
DB_BIND_ADDR=127.0.0.1
AUTH_PORT=${auth_port}
AUTH_BIND_ADDR=127.0.0.1
ALLOW_LAN_BINDING=no
GOTRUE_DB_DATABASE_URL=postgres://mf_accept:${password}@db:5432/${database_name}?sslmode=disable&search_path=auth
GOTRUE_SITE_URL=http://127.0.0.1:3000
GOTRUE_ADDITIONAL_REDIRECT_URLS=http://127.0.0.1:3000
GOTRUE_URI_ALLOW_LIST=http://127.0.0.1:3000
GOTRUE_API_EXTERNAL_URL=http://127.0.0.1:${auth_port}
GOTRUE_JWT_SECRET=${jwt_secret}
GOTRUE_OPERATOR_TOKEN=${operator_token}
GOTRUE_DB_MAX_POOL_SIZE=10
GOTRUE_MAILER_AUTOCONFIRM=true
TZ=UTC
MF_ENV_CLASS=disposable
MF_DISPOSABLE_TARGET_ID=${RUN_ID}
MF_DISPOSABLE_CONFIRMATION=RESTORE:${RUN_ID}
MF_EXPECTED_VOLUME=${project_name}_mindforge-postgres-data
EOF
  chmod 600 "$path"
}

safe_auth_error_hint() {
  local response_file="$1"
  local hint
  hint='unclassified_error'
  if command -v jq >/dev/null 2>&1; then
    hint="$(jq -r 'if type == "object" then (.msg // .message // .error_description // .error // "unclassified_error") else "unclassified_error" end' "$response_file" 2>/dev/null || printf 'unclassified_error')"
  fi
  hint="$(printf '%s' "$hint" | tr '\r\n' ' ' | sed -E -e 's/[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+/[synthetic-email]/g' -e 's/([Pp]assword|[Ss]ecret|[Tt]oken)[^[:space:],;]*/\1=[REDACTED]/g' -e 's/[A-Za-z0-9_-]{24,}/[REDACTED]/g')"
  [[ -n "$hint" ]] || hint='unclassified_error'
  printf '%.120s' "$hint"
}

auth_runtime_error_hint() {
  local env_file="$1"
  local project_name="$2"
  local missing_relation migration_count
  if compose_for "$env_file" "$project_name" logs auth 2>/dev/null | grep -qi 'permission denied'; then
    printf 'auth_db_permission'
  elif compose_for "$env_file" "$project_name" logs auth 2>/dev/null | grep -qiE 'relation .* does not exist'; then
    missing_relation="$(compose_for "$env_file" "$project_name" logs auth 2>/dev/null | tr -d '\\' | sed -nE 's/.*relation "?([A-Za-z_][A-Za-z0-9_.]*)"? does not exist.*/\1/p' | head -n 1)"
    migration_count="$(query_scalar "$env_file" "$project_name" 'SELECT count(*) FROM auth.schema_migrations;' 2>/dev/null || true)"
    [[ "$migration_count" =~ ^[0-9]+$ ]] || migration_count='unknown'
    if [[ "$missing_relation" =~ ^[A-Za-z_][A-Za-z0-9_.]*$ ]]; then
      printf 'auth_catalog_relation_missing_%s_migrations_%s' "$missing_relation" "$migration_count"
    else
      printf 'auth_catalog_relation_missing'
    fi
  elif compose_for "$env_file" "$project_name" logs auth 2>/dev/null | grep -qiE 'column .* does not exist'; then
    printf 'auth_catalog_column_missing'
  elif compose_for "$env_file" "$project_name" logs auth 2>/dev/null | grep -qiE 'role .* does not exist'; then
    printf 'auth_role_missing'
  elif compose_for "$env_file" "$project_name" logs auth 2>/dev/null | grep -qiE 'connection|connect'; then
    printf 'auth_db_connection'
  else
    printf 'auth_runtime_unclassified'
  fi
}

verify_auth_flow() {
  local env_file="$1"
  local project_name="$2"
  local auth_port="$3"
  local email password signup_code token_code signup_response signup_error signup_hint runtime_hint
  email="accept-${RUN_ID}@example.invalid"
  password="$(random_hex)"
  signup_response="$(mktemp "${WORK_DIR}/signup-response.XXXXXX")"
  chmod 600 "$signup_response"
  signup_code="$(curl -sS -o "$signup_response" -w '%{http_code}' -H 'Content-Type: application/json' --data "{\"email\":\"${email}\",\"password\":\"${password}\"}" "http://127.0.0.1:${auth_port}/signup")"
  if [[ "$signup_code" != '200' && "$signup_code" != '201' ]]; then
    signup_error='unclassified_error'
    if grep -qiE 'signup_disabled|signups? (are )?(disabled|not allowed)' "$signup_response"; then
      signup_error='signup_disabled'
    elif grep -qiE 'email_address_not_authorized|email address .* is invalid' "$signup_response"; then
      signup_error='email_validation'
    elif grep -qiE 'password.*(short|weak|invalid|required)' "$signup_response"; then
      signup_error='password_validation'
    elif grep -qiE 'database error|error creating user|creating confirmation token' "$signup_response"; then
      signup_error='database_error'
    elif grep -qiE 'sending.*mail|mailer|email.*send' "$signup_response"; then
      signup_error='mailer_error'
    elif grep -qiE 'invalid|malformed|required|bad request' "$signup_response"; then
      signup_error='request_validation'
    fi
    signup_hint="$(safe_auth_error_hint "$signup_response")"
    runtime_hint="$(auth_runtime_error_hint "$env_file" "$project_name")"
    rm -f "$signup_response"
    integration_fail "synthetic Auth sign-up returned HTTP ${signup_code} (${signup_error}; ${signup_hint}; ${runtime_hint})"
  fi
  rm -f "$signup_response"
  token_code="$(curl -sS -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' --data "{\"email\":\"${email}\",\"password\":\"${password}\"}" "http://127.0.0.1:${auth_port}/token?grant_type=password")"
  [[ "$token_code" == '200' ]] || integration_fail "synthetic Auth password grant returned HTTP ${token_code}"
}

run_psql() {
  local env_file="$1"
  local project_name="$2"
  local sql="$3"
  compose_for "$env_file" "$project_name" exec -T db sh -ec 'PGPASSWORD="$POSTGRES_PASSWORD" exec psql -At -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"' -- -c "$sql"
}

query_scalar() {
  local env_file="$1"
  local project_name="$2"
  local sql="$3"
  local output
  output="$(run_psql "$env_file" "$project_name" "$sql")" || return 1
  printf '%s\n' "$output" | tr -d '\r' | sed -nE '/^[0-9]+(\|[01])?$/p' | tail -n 1
}

gotrue_catalog_summary() {
  local env_file="$1"
  local project_name="$2"
  local catalog_state
  catalog_state="$(query_scalar "$env_file" "$project_name" "SELECT count(*)::text || '|' || (to_regclass('auth.identities') IS NOT NULL)::int::text FROM auth.schema_migrations;" 2>/dev/null)"
  [[ "$catalog_state" =~ ^[0-9]+\|[01]$ ]] || integration_fail 'GoTrue catalog summary could not be read'
  printf 'GoTrue catalog summary: migration_rows=%s identities_present=%s\n' "${catalog_state%%|*}" "${catalog_state#*|}"
}

assert_marker() {
  local env_file="$1"
  local project_name="$2"
  local result
  result="$(query_scalar "$env_file" "$project_name" "SELECT 1 FROM infra.disposable_validation_marker WHERE run_id = '${RUN_ID}';")"
  [[ "$result" == '1' ]] || integration_fail 'synthetic persistence marker is absent'
}

stop_project_preserving_volume() {
  local env_file="$1"
  local project_name="$2"
  [[ -n "$env_file" && -f "$env_file" ]] || return 0
  compose_for "$env_file" "$project_name" down >/dev/null 2>&1 || true
}

cleanup_local_runtime() {
  stop_project_preserving_volume "$SRC_ENV" "$SRC_PROJECT"
  stop_project_preserving_volume "$RESTORE_ENV" "$RESTORE_PROJECT"
  stop_project_preserving_volume "$REPLAY_ENV" "$REPLAY_PROJECT"
  if [[ -n "$WORK_DIR" && -d "$WORK_DIR" ]]; then
    rm -f "$WORK_DIR"/*.env "$WORK_DIR"/*.dump "$WORK_DIR"/signup-response.* "$WORK_DIR"/backups/*.dump 2>/dev/null || true
    rmdir "$WORK_DIR"/backups 2>/dev/null || true
    rmdir "$WORK_DIR" 2>/dev/null || true
  fi
}

run_local_matrix() {
  command -v docker >/dev/null 2>&1 || integration_fail 'Docker CLI is unavailable; local disposable acceptance cannot start'
  if ! docker compose version >/dev/null 2>&1 && ! (command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1); then
    integration_fail 'Docker Compose is unavailable; do not substitute the VM for local acceptance'
  fi
  docker info >/dev/null 2>&1 || integration_fail 'Docker daemon is unavailable; do not substitute the VM for local acceptance'
  command -v curl >/dev/null 2>&1 || integration_fail 'curl is required for the Auth flow check'
  command -v pg_restore >/dev/null 2>&1 || integration_fail 'pg_restore is required for isolated restore preflight'

  RUN_ID="$(random_hex | cut -c1-16)"
  WORK_DIR="$(mktemp -d "${TMPDIR:-/tmp}/mindforge-infra-acceptance.XXXXXX")"
  mkdir -p "${WORK_DIR}/backups"
  chmod 700 "$WORK_DIR" "${WORK_DIR}/backups"
  trap cleanup_local_runtime EXIT HUP INT TERM

  SRC_PROJECT="mf-accept-${RUN_ID}-src"
  RESTORE_PROJECT="mf-accept-${RUN_ID}-restore"
  REPLAY_PROJECT="mf-accept-${RUN_ID}-replay"
  SRC_ENV="${WORK_DIR}/src.env"
  RESTORE_ENV="${WORK_DIR}/restore.env"
  REPLAY_ENV="${WORK_DIR}/replay.env"
  write_disposable_env "$SRC_ENV" src "$SRC_PROJECT" "mf_src_${RUN_ID}" "$(pick_loopback_port)" "$(pick_loopback_port)"
  write_disposable_env "$RESTORE_ENV" restore "$RESTORE_PROJECT" "mf_restore_${RUN_ID}" "$(pick_loopback_port)" "$(pick_loopback_port)"
  write_disposable_env "$REPLAY_ENV" replay "$REPLAY_PROJECT" "mf_replay_${RUN_ID}" "$(pick_loopback_port)" "$(pick_loopback_port)"

  bash "${ROOT_DIR}/scripts/acceptance-disposable.sh" --local-disposable "$RUN_ID" src "$SRC_ENV"
  bash "${ROOT_DIR}/scripts/up.sh" "$SRC_ENV" "$SRC_PROJECT"
  gotrue_catalog_summary "$SRC_ENV" "$SRC_PROJECT"
  bash "${ROOT_DIR}/scripts/inspect.sh" "$SRC_ENV" "$SRC_PROJECT"
  source_auth_port="$(awk -F= '$1 == "AUTH_PORT" { print $2 }' "$SRC_ENV")"
  verify_auth_flow "$SRC_ENV" "$SRC_PROJECT" "$source_auth_port"
  run_psql "$SRC_ENV" "$SRC_PROJECT" "BEGIN; SELECT '[1,2,3]'::vector <-> '[1,2,3]'::vector; ROLLBACK;" >/dev/null
  run_psql "$SRC_ENV" "$SRC_PROJECT" "CREATE TABLE IF NOT EXISTS infra.disposable_validation_marker (run_id text PRIMARY KEY, created_at timestamptz NOT NULL DEFAULT now()); INSERT INTO infra.disposable_validation_marker (run_id) VALUES ('${RUN_ID}') ON CONFLICT (run_id) DO NOTHING;" >/dev/null

  compose_for "$SRC_ENV" "$SRC_PROJECT" down
  bash "${ROOT_DIR}/scripts/up.sh" "$SRC_ENV" "$SRC_PROJECT"
  assert_marker "$SRC_ENV" "$SRC_PROJECT"

  bash "${ROOT_DIR}/scripts/backup.sh" "$SRC_ENV" "$SRC_PROJECT" "${WORK_DIR}/backups"
  BACKUP_FILE="$(find "${WORK_DIR}/backups" -maxdepth 1 -type f -name '*.dump' -print)"
  [[ -n "$BACKUP_FILE" && -f "$BACKUP_FILE" ]] || integration_fail 'backup artifact was not created'

  bash "${ROOT_DIR}/scripts/acceptance-disposable.sh" --local-disposable "$RUN_ID" restore "$RESTORE_ENV"
  bash "${ROOT_DIR}/scripts/restore.sh" "$BACKUP_FILE" "$RESTORE_ENV" "$RESTORE_PROJECT" "$RUN_ID" "RESTORE:${RUN_ID}"
  bash "${ROOT_DIR}/scripts/up.sh" "$RESTORE_ENV" "$RESTORE_PROJECT"
  bash "${ROOT_DIR}/scripts/inspect.sh" "$RESTORE_ENV" "$RESTORE_PROJECT"
  assert_marker "$RESTORE_ENV" "$RESTORE_PROJECT"

  bash "${ROOT_DIR}/scripts/acceptance-disposable.sh" --local-disposable "$RUN_ID" replay "$REPLAY_ENV"
  bash "${ROOT_DIR}/scripts/up.sh" "$REPLAY_ENV" "$REPLAY_PROJECT"
  bash "${ROOT_DIR}/scripts/migrate.sh" "$REPLAY_ENV" "$REPLAY_PROJECT"
  bash "${ROOT_DIR}/scripts/migrate.sh" "$REPLAY_ENV" "$REPLAY_PROJECT"
  bash "${ROOT_DIR}/scripts/inspect.sh" "$REPLAY_ENV" "$REPLAY_PROJECT"
  run_psql "$REPLAY_ENV" "$REPLAY_PROJECT" "DO \$\$ BEGIN IF to_regclass('public.users') IS NOT NULL OR to_regclass('public.brains') IS NOT NULL OR to_regclass('public.neural_layers') IS NOT NULL OR to_regclass('public.documents') IS NOT NULL THEN RAISE EXCEPTION 'domain table detected'; END IF; END \$\$;" >/dev/null

  local evidence_file
  evidence_file="${TMPDIR:-/tmp}/mindforge-infra-acceptance-${RUN_ID}.volumes"
  umask 077
  printf '%s\n' "${SRC_PROJECT}_mindforge-postgres-data" "${RESTORE_PROJECT}_mindforge-postgres-data" "${REPLAY_PROJECT}_mindforge-postgres-data" > "$evidence_file"
  stop_project_preserving_volume "$SRC_ENV" "$SRC_PROJECT"
  stop_project_preserving_volume "$RESTORE_ENV" "$RESTORE_PROJECT"
  stop_project_preserving_volume "$REPLAY_ENV" "$REPLAY_PROJECT"
  printf 'Local disposable acceptance passed. Retained exclusive volumes are recorded in %s\n' "$evidence_file"
}

run_vm_matrix() {
  local remote_dir vm_ssh_host vm_ssh_user vm_ssh_port vm_ssh_port_number
  local -a ssh_command archive_metadata_args
  [[ "${MF_LOCAL_ACCEPTANCE_GREEN:-}" == '1' ]] || integration_fail 'VM matrix requires local disposable greens in this same invocation'
  command -v ssh >/dev/null 2>&1 || integration_fail 'ssh is required for authorized VM validation'
  command -v tar >/dev/null 2>&1 || integration_fail 'tar is required to copy only the corrected package'
  vm_ssh_host="${MF_VM_SSH_HOST:-}"
  vm_ssh_user="${MF_VM_SSH_USER:-}"
  if [[ "${MF_VM_SSH_PORT+x}" == 'x' ]]; then
    vm_ssh_port="$MF_VM_SSH_PORT"
  else
    vm_ssh_port='22'
  fi
  [[ -n "$vm_ssh_host" ]] || integration_fail 'MF_VM_SSH_HOST is required for VM validation'
  [[ -n "$vm_ssh_user" ]] || integration_fail 'MF_VM_SSH_USER is required for VM validation'
  [[ "$vm_ssh_port" =~ ^[0-9]{1,5}$ ]] || integration_fail 'MF_VM_SSH_PORT must be a numeric TCP port from 1 to 65535'
  vm_ssh_port_number=$((10#$vm_ssh_port))
  if (( vm_ssh_port_number < 1 || vm_ssh_port_number > 65535 )); then
    integration_fail 'MF_VM_SSH_PORT must be a numeric TCP port from 1 to 65535'
  fi
  [[ -n "${MF_VM_SSH_IDENTITY_FILE:-}" && -f "$MF_VM_SSH_IDENTITY_FILE" && -r "$MF_VM_SSH_IDENTITY_FILE" ]] || integration_fail 'an explicit readable owner SSH identity is required for VM validation'
  ssh_command=(ssh -i "$MF_VM_SSH_IDENTITY_FILE" -o IdentitiesOnly=yes -o BatchMode=yes -p "$vm_ssh_port" "${vm_ssh_user}@${vm_ssh_host}")
  remote_dir="/tmp/mindforge-infra-acceptance-$(random_hex | cut -c1-16)"
  "${ssh_command[@]}" "umask 077 && mkdir -p '${remote_dir}'"
  archive_metadata_args=(--no-xattrs)
  if tar --no-mac-metadata -cf /dev/null -T /dev/null >/dev/null 2>&1; then
    archive_metadata_args+=(--no-mac-metadata)
  fi
  tar -C "$ROOT_DIR" "${archive_metadata_args[@]}" --exclude=.env --exclude=backups --exclude='*.dump' --exclude=acceptance-artifacts -cf - . | "${ssh_command[@]}" "tar -C '${remote_dir}' -xf -"
  "${ssh_command[@]}" "cd '${remote_dir}' && bash tests/integration_disposable.sh --local-disposable"
  printf 'VM disposable acceptance passed in retained remote package directory %s\n' "$remote_dir"
}

case "$MODE" in
  --local-disposable)
    run_local_matrix
    ;;
  --vm-disposable)
    run_local_matrix
    MF_LOCAL_ACCEPTANCE_GREEN=1
    run_vm_matrix
    ;;
  *)
    printf 'usage: %s <--local-disposable|--vm-disposable>\n' "$0" >&2
    exit 2
    ;;
esac
