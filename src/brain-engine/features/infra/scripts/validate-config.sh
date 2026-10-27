#!/usr/bin/env bash
set -euo pipefail

fail_config() {
  printf 'Invalid infrastructure configuration: %s\n' "$1" >&2
  exit 1
}

is_allowed_key() {
  case "$1" in
    POSTGRES_ADMIN_USER|POSTGRES_ADMIN_PASSWORD|POSTGRES_HOST|POSTGRES_CLIENT_HOST|POSTGRES_PORT|DB_BIND_ADDR|DEVELOP_DB_NAME|DEVELOP_DB_USER|DEVELOP_DB_PASSWORD|MAIN_DB_NAME|MAIN_DB_USER|MAIN_DB_PASSWORD|DEVELOP_AUTH_DB_URL|MAIN_AUTH_DB_URL|DEVELOP_AUTH_PORT|MAIN_AUTH_PORT|DEVELOP_AUTH_BIND_ADDR|MAIN_AUTH_BIND_ADDR|ALLOW_LAN_BINDING|DEVELOP_AUTH_SITE_URL|MAIN_AUTH_SITE_URL|DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS|MAIN_AUTH_ADDITIONAL_REDIRECT_URLS|DEVELOP_AUTH_URI_ALLOW_LIST|MAIN_AUTH_URI_ALLOW_LIST|DEVELOP_AUTH_API_EXTERNAL_URL|MAIN_AUTH_API_EXTERNAL_URL|DEVELOP_AUTH_JWT_SECRET|MAIN_AUTH_JWT_SECRET|DEVELOP_AUTH_OPERATOR_TOKEN|MAIN_AUTH_OPERATOR_TOKEN|GOTRUE_DB_MAX_POOL_SIZE|GOTRUE_MAILER_AUTOCONFIRM|TZ)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

clear_known_values() {
  unset POSTGRES_ADMIN_USER POSTGRES_ADMIN_PASSWORD POSTGRES_HOST POSTGRES_CLIENT_HOST POSTGRES_PORT DB_BIND_ADDR
  unset DEVELOP_DB_NAME DEVELOP_DB_USER DEVELOP_DB_PASSWORD MAIN_DB_NAME MAIN_DB_USER MAIN_DB_PASSWORD
  unset DEVELOP_AUTH_DB_URL MAIN_AUTH_DB_URL DEVELOP_AUTH_PORT MAIN_AUTH_PORT
  unset DEVELOP_AUTH_BIND_ADDR MAIN_AUTH_BIND_ADDR ALLOW_LAN_BINDING
  unset DEVELOP_AUTH_SITE_URL MAIN_AUTH_SITE_URL
  unset DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS MAIN_AUTH_ADDITIONAL_REDIRECT_URLS
  unset DEVELOP_AUTH_URI_ALLOW_LIST MAIN_AUTH_URI_ALLOW_LIST
  unset DEVELOP_AUTH_API_EXTERNAL_URL MAIN_AUTH_API_EXTERNAL_URL
  unset DEVELOP_AUTH_JWT_SECRET MAIN_AUTH_JWT_SECRET
  unset DEVELOP_AUTH_OPERATOR_TOKEN MAIN_AUTH_OPERATOR_TOKEN
  unset GOTRUE_DB_MAX_POOL_SIZE GOTRUE_MAILER_AUTOCONFIRM TZ
}

load_dotenv_restricted() {
  local line key value seen_keys='|' line_number=0

  [[ -n "$ENV_FILE" && -f "$ENV_FILE" ]] || fail_config 'environment file is required'
  clear_known_values

  while IFS= read -r line || [[ -n "$line" ]]; do
    line_number=$((line_number + 1))
    line="${line%$'\r'}"
    case "$line" in
      ''|'#'*)
        continue
        ;;
    esac

    case "$line" in
      *=*)
        key="${line%%=*}"
        value="${line#*=}"
        ;;
      *)
        fail_config "environment line ${line_number} must be KEY=VALUE"
        ;;
    esac

    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || fail_config 'environment key is invalid'
    is_allowed_key "$key" || fail_config "environment key $key is not allowed"
    case "$seen_keys" in
      *"|${key}|"*)
        fail_config "environment key $key is duplicated"
        ;;
    esac
    seen_keys="${seen_keys}${key}|"
    export "$key=$value"
  done < "$ENV_FILE"
}

require_value() {
  local name="$1"
  local value="$2"
  [[ -n "$value" ]] || fail_config "$name is required"
  case "$value" in
    replace-with-*)
      fail_config "$name still uses a placeholder"
      ;;
  esac
}

require_identifier() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  [[ "$value" =~ ^[a-z_][a-z0-9_]*$ ]] || fail_config "$name must be a lowercase SQL identifier"
}

require_hostname() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  (( ${#value} <= 253 )) || fail_config "$name must be a safe hostname"
  [[ "$value" =~ ^([A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?\.)*[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?$ ]] || fail_config "$name must be a safe hostname"
}

require_uri_safe_secret() {
  local name="$1"
  local value="$2"
  local minimum_length="$3"
  require_value "$name" "$value"
  [[ "$value" =~ ^[A-Za-z0-9._~-]+$ ]] || fail_config "$name must contain only URI-safe characters"
  (( ${#value} >= minimum_length )) || fail_config "$name must contain at least ${minimum_length} characters"
}

require_port() {
  local name="$1"
  local port="$2"
  [[ "$port" =~ ^[1-9][0-9]{0,4}$ ]] || fail_config "$name must be a TCP port"
  (( port <= 65535 )) || fail_config "$name must be a TCP port"
}

require_url() {
  local name="$1"
  local value="$2"
  local rest host
  require_value "$name" "$value"
  case "$value" in
    http://*|https://*)
      ;;
    *)
      fail_config "$name must use http or https"
      ;;
  esac
  [[ "$value" != *'*'* && "$value" != *' '* ]] || fail_config "$name cannot contain wildcards or spaces"
  rest="${value#*://}"
  host="${rest%%/*}"
  [[ -n "$host" && "$host" != *'?'* && "$host" != *'#'* ]] || fail_config "$name must include a valid host"
}

require_redirect_list() {
  local name="$1"
  local value="$2"
  local item
  local -a items
  require_value "$name" "$value"
  [[ "$value" != *'*'* && "$value" != *' '* ]] || fail_config "$name cannot contain wildcards or spaces"
  IFS=',' read -r -a items <<< "$value"
  for item in "${items[@]}"; do
    require_url "$name" "$item"
  done
}

require_bind_address() {
  local name="$1"
  local value="$2"
  case "$value" in
    127.0.0.1)
      ;;
    0.0.0.0)
      [[ "${ALLOW_LAN_BINDING:-}" == 'yes' ]] || fail_config "$name requires ALLOW_LAN_BINDING=yes"
      ;;
    *)
      fail_config "$name must be 127.0.0.1 or explicit 0.0.0.0 opt-in"
      ;;
  esac
}

require_distinct() {
  local left_name="$1"
  local left_value="$2"
  local right_name="$3"
  local right_value="$4"
  [[ "$left_value" != "$right_value" ]] || fail_config "$left_name and $right_name must be distinct"
}

validate_config() {
  load_dotenv_restricted

  require_identifier POSTGRES_ADMIN_USER "${POSTGRES_ADMIN_USER:-}"
  require_uri_safe_secret POSTGRES_ADMIN_PASSWORD "${POSTGRES_ADMIN_PASSWORD:-}" 16
  require_value POSTGRES_HOST "${POSTGRES_HOST:-}"
  require_hostname POSTGRES_CLIENT_HOST "${POSTGRES_CLIENT_HOST:-}"
  require_value ALLOW_LAN_BINDING "${ALLOW_LAN_BINDING:-}"
  [[ "$POSTGRES_HOST" == 'db' ]] || fail_config 'POSTGRES_HOST must be the internal db service'
  [[ "$ALLOW_LAN_BINDING" == 'yes' || "$ALLOW_LAN_BINDING" == 'no' ]] || fail_config 'ALLOW_LAN_BINDING must be yes or no'

  require_port POSTGRES_PORT "${POSTGRES_PORT:-}"
  require_bind_address DB_BIND_ADDR "${DB_BIND_ADDR:-}"

  require_value DEVELOP_DB_NAME "${DEVELOP_DB_NAME:-}"
  require_value MAIN_DB_NAME "${MAIN_DB_NAME:-}"
  [[ "$DEVELOP_DB_NAME" == 'mindforge_develop' ]] || fail_config 'DEVELOP_DB_NAME must be mindforge_develop'
  [[ "$MAIN_DB_NAME" == 'mindforge' ]] || fail_config 'MAIN_DB_NAME must be mindforge'
  require_identifier DEVELOP_DB_USER "${DEVELOP_DB_USER:-}"
  require_identifier MAIN_DB_USER "${MAIN_DB_USER:-}"
  require_distinct DEVELOP_DB_USER "$DEVELOP_DB_USER" MAIN_DB_USER "$MAIN_DB_USER"
  require_distinct POSTGRES_ADMIN_USER "$POSTGRES_ADMIN_USER" DEVELOP_DB_USER "$DEVELOP_DB_USER"
  require_uri_safe_secret DEVELOP_DB_PASSWORD "${DEVELOP_DB_PASSWORD:-}" 16
  require_uri_safe_secret MAIN_DB_PASSWORD "${MAIN_DB_PASSWORD:-}" 16
  require_distinct DEVELOP_DB_PASSWORD "$DEVELOP_DB_PASSWORD" MAIN_DB_PASSWORD "$MAIN_DB_PASSWORD"
  require_distinct POSTGRES_ADMIN_PASSWORD "$POSTGRES_ADMIN_PASSWORD" DEVELOP_DB_PASSWORD "$DEVELOP_DB_PASSWORD"
  if [[ "$POSTGRES_ADMIN_USER" == "$MAIN_DB_USER" ]]; then
    [[ "$POSTGRES_ADMIN_PASSWORD" == "$MAIN_DB_PASSWORD" ]] || fail_config 'POSTGRES_ADMIN_PASSWORD must match MAIN_DB_PASSWORD when their users are the same'
  else
    require_distinct POSTGRES_ADMIN_PASSWORD "$POSTGRES_ADMIN_PASSWORD" MAIN_DB_PASSWORD "$MAIN_DB_PASSWORD"
  fi

  require_value DEVELOP_AUTH_DB_URL "${DEVELOP_AUTH_DB_URL:-}"
  require_value MAIN_AUTH_DB_URL "${MAIN_AUTH_DB_URL:-}"
  local expected_develop_database_url expected_main_database_url
  expected_develop_database_url="postgres://${DEVELOP_DB_USER}:${DEVELOP_DB_PASSWORD}@${POSTGRES_HOST}:5432/${DEVELOP_DB_NAME}?sslmode=disable&search_path=auth"
  expected_main_database_url="postgres://${MAIN_DB_USER}:${MAIN_DB_PASSWORD}@${POSTGRES_HOST}:5432/${MAIN_DB_NAME}?sslmode=disable&search_path=auth"
  [[ "$DEVELOP_AUTH_DB_URL" == "$expected_develop_database_url" ]] || fail_config 'DEVELOP_AUTH_DB_URL must match the develop role, database, internal db:5432 endpoint, and auth search path'
  [[ "$MAIN_AUTH_DB_URL" == "$expected_main_database_url" ]] || fail_config 'MAIN_AUTH_DB_URL must match the main role, database, internal db:5432 endpoint, and auth search path'
  require_distinct DEVELOP_AUTH_DB_URL "$DEVELOP_AUTH_DB_URL" MAIN_AUTH_DB_URL "$MAIN_AUTH_DB_URL"

  require_port DEVELOP_AUTH_PORT "${DEVELOP_AUTH_PORT:-}"
  require_port MAIN_AUTH_PORT "${MAIN_AUTH_PORT:-}"
  require_distinct DEVELOP_AUTH_PORT "$DEVELOP_AUTH_PORT" MAIN_AUTH_PORT "$MAIN_AUTH_PORT"
  require_distinct POSTGRES_PORT "$POSTGRES_PORT" DEVELOP_AUTH_PORT "$DEVELOP_AUTH_PORT"
  require_distinct POSTGRES_PORT "$POSTGRES_PORT" MAIN_AUTH_PORT "$MAIN_AUTH_PORT"
  require_bind_address DEVELOP_AUTH_BIND_ADDR "${DEVELOP_AUTH_BIND_ADDR:-}"
  require_bind_address MAIN_AUTH_BIND_ADDR "${MAIN_AUTH_BIND_ADDR:-}"

  require_url DEVELOP_AUTH_SITE_URL "${DEVELOP_AUTH_SITE_URL:-}"
  require_url MAIN_AUTH_SITE_URL "${MAIN_AUTH_SITE_URL:-}"
  require_redirect_list DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS "${DEVELOP_AUTH_ADDITIONAL_REDIRECT_URLS:-}"
  require_redirect_list MAIN_AUTH_ADDITIONAL_REDIRECT_URLS "${MAIN_AUTH_ADDITIONAL_REDIRECT_URLS:-}"
  require_redirect_list DEVELOP_AUTH_URI_ALLOW_LIST "${DEVELOP_AUTH_URI_ALLOW_LIST:-}"
  require_redirect_list MAIN_AUTH_URI_ALLOW_LIST "${MAIN_AUTH_URI_ALLOW_LIST:-}"
  require_url DEVELOP_AUTH_API_EXTERNAL_URL "${DEVELOP_AUTH_API_EXTERNAL_URL:-}"
  require_url MAIN_AUTH_API_EXTERNAL_URL "${MAIN_AUTH_API_EXTERNAL_URL:-}"
  require_distinct DEVELOP_AUTH_API_EXTERNAL_URL "$DEVELOP_AUTH_API_EXTERNAL_URL" MAIN_AUTH_API_EXTERNAL_URL "$MAIN_AUTH_API_EXTERNAL_URL"

  require_uri_safe_secret DEVELOP_AUTH_JWT_SECRET "${DEVELOP_AUTH_JWT_SECRET:-}" 32
  require_uri_safe_secret MAIN_AUTH_JWT_SECRET "${MAIN_AUTH_JWT_SECRET:-}" 32
  require_uri_safe_secret DEVELOP_AUTH_OPERATOR_TOKEN "${DEVELOP_AUTH_OPERATOR_TOKEN:-}" 32
  require_uri_safe_secret MAIN_AUTH_OPERATOR_TOKEN "${MAIN_AUTH_OPERATOR_TOKEN:-}" 32
  require_distinct DEVELOP_AUTH_JWT_SECRET "$DEVELOP_AUTH_JWT_SECRET" MAIN_AUTH_JWT_SECRET "$MAIN_AUTH_JWT_SECRET"
  require_distinct DEVELOP_AUTH_OPERATOR_TOKEN "$DEVELOP_AUTH_OPERATOR_TOKEN" MAIN_AUTH_OPERATOR_TOKEN "$MAIN_AUTH_OPERATOR_TOKEN"
  require_distinct DEVELOP_AUTH_JWT_SECRET "$DEVELOP_AUTH_JWT_SECRET" DEVELOP_AUTH_OPERATOR_TOKEN "$DEVELOP_AUTH_OPERATOR_TOKEN"
  require_distinct MAIN_AUTH_JWT_SECRET "$MAIN_AUTH_JWT_SECRET" MAIN_AUTH_OPERATOR_TOKEN "$MAIN_AUTH_OPERATOR_TOKEN"
  require_distinct DEVELOP_AUTH_JWT_SECRET "$DEVELOP_AUTH_JWT_SECRET" MAIN_AUTH_OPERATOR_TOKEN "$MAIN_AUTH_OPERATOR_TOKEN"
  require_distinct MAIN_AUTH_JWT_SECRET "$MAIN_AUTH_JWT_SECRET" DEVELOP_AUTH_OPERATOR_TOKEN "$DEVELOP_AUTH_OPERATOR_TOKEN"

  require_value GOTRUE_DB_MAX_POOL_SIZE "${GOTRUE_DB_MAX_POOL_SIZE:-}"
  [[ "$GOTRUE_DB_MAX_POOL_SIZE" =~ ^[1-9][0-9]*$ ]] || fail_config 'GOTRUE_DB_MAX_POOL_SIZE must be a positive integer'
  if [[ -n "${GOTRUE_MAILER_AUTOCONFIRM:-}" ]]; then
    [[ "$GOTRUE_MAILER_AUTOCONFIRM" == 'true' || "$GOTRUE_MAILER_AUTOCONFIRM" == 'false' ]] || fail_config 'GOTRUE_MAILER_AUTOCONFIRM must be true or false'
  fi
  require_value TZ "${TZ:-}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  ENV_FILE="${1:-}"
  validate_config
fi
