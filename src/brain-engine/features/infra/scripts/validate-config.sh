#!/usr/bin/env bash
set -euo pipefail

fail_config() {
  printf 'Invalid infrastructure configuration: %s\n' "$1" >&2
  exit 1
}

is_allowed_key() {
  case "$1" in
    POSTGRES_DB|POSTGRES_USER|POSTGRES_PASSWORD|POSTGRES_HOST|POSTGRES_PORT|DB_BIND_ADDR|AUTH_PORT|AUTH_BIND_ADDR|ALLOW_LAN_BINDING|GOTRUE_DB_DATABASE_URL|GOTRUE_SITE_URL|GOTRUE_ADDITIONAL_REDIRECT_URLS|GOTRUE_URI_ALLOW_LIST|GOTRUE_API_EXTERNAL_URL|GOTRUE_JWT_SECRET|GOTRUE_OPERATOR_TOKEN|GOTRUE_DB_MAX_POOL_SIZE|GOTRUE_MAILER_AUTOCONFIRM|TZ|MF_ENV_CLASS|MF_DISPOSABLE_TARGET_ID|MF_DISPOSABLE_CONFIRMATION|MF_EXPECTED_VOLUME)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

clear_known_values() {
  unset POSTGRES_DB POSTGRES_USER POSTGRES_PASSWORD POSTGRES_HOST POSTGRES_PORT
  unset DB_BIND_ADDR AUTH_PORT AUTH_BIND_ADDR ALLOW_LAN_BINDING
  unset GOTRUE_DB_DATABASE_URL GOTRUE_SITE_URL GOTRUE_ADDITIONAL_REDIRECT_URLS
  unset GOTRUE_URI_ALLOW_LIST GOTRUE_API_EXTERNAL_URL GOTRUE_JWT_SECRET
  unset GOTRUE_OPERATOR_TOKEN GOTRUE_DB_MAX_POOL_SIZE GOTRUE_MAILER_AUTOCONFIRM TZ
  unset MF_ENV_CLASS MF_DISPOSABLE_TARGET_ID MF_DISPOSABLE_CONFIRMATION MF_EXPECTED_VOLUME
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
  [[ -n "$host" ]] || fail_config "$name must include a host"
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

validate_config() {
  load_dotenv_restricted

  require_value POSTGRES_DB "${POSTGRES_DB:-}"
  require_value POSTGRES_USER "${POSTGRES_USER:-}"
  require_value POSTGRES_PASSWORD "${POSTGRES_PASSWORD:-}"
  require_value POSTGRES_HOST "${POSTGRES_HOST:-}"
  require_value GOTRUE_JWT_SECRET "${GOTRUE_JWT_SECRET:-}"
  require_value GOTRUE_OPERATOR_TOKEN "${GOTRUE_OPERATOR_TOKEN:-}"
  require_value ALLOW_LAN_BINDING "${ALLOW_LAN_BINDING:-}"
  require_value GOTRUE_DB_DATABASE_URL "${GOTRUE_DB_DATABASE_URL:-}"

  [[ "${POSTGRES_HOST:-}" == 'db' ]] || fail_config 'POSTGRES_HOST must be the internal db service'
  [[ "${ALLOW_LAN_BINDING:-}" == 'yes' || "${ALLOW_LAN_BINDING:-}" == 'no' ]] || fail_config 'ALLOW_LAN_BINDING must be yes or no'
  (( ${#POSTGRES_PASSWORD} >= 16 )) || fail_config 'POSTGRES_PASSWORD must contain at least 16 characters'
  (( ${#GOTRUE_JWT_SECRET} >= 32 )) || fail_config 'GOTRUE_JWT_SECRET must contain at least 32 characters'
  (( ${#GOTRUE_OPERATOR_TOKEN} >= 32 )) || fail_config 'GOTRUE_OPERATOR_TOKEN must contain at least 32 characters'

  require_port POSTGRES_PORT "${POSTGRES_PORT:-}"
  require_port AUTH_PORT "${AUTH_PORT:-}"
  require_bind_address DB_BIND_ADDR "${DB_BIND_ADDR:-}"
  require_bind_address AUTH_BIND_ADDR "${AUTH_BIND_ADDR:-}"
  require_url GOTRUE_SITE_URL "${GOTRUE_SITE_URL:-}"
  require_redirect_list GOTRUE_ADDITIONAL_REDIRECT_URLS "${GOTRUE_ADDITIONAL_REDIRECT_URLS:-}"
  require_redirect_list GOTRUE_URI_ALLOW_LIST "${GOTRUE_URI_ALLOW_LIST:-}"
  require_url GOTRUE_API_EXTERNAL_URL "${GOTRUE_API_EXTERNAL_URL:-}"
  if [[ -n "${GOTRUE_MAILER_AUTOCONFIRM:-}" ]]; then
    [[ "$GOTRUE_MAILER_AUTOCONFIRM" == 'true' || "$GOTRUE_MAILER_AUTOCONFIRM" == 'false' ]] || fail_config 'GOTRUE_MAILER_AUTOCONFIRM must be true or false'
  fi

  local expected_database_url
  expected_database_url="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@db:5432/${POSTGRES_DB}?sslmode=disable&search_path=auth"
  [[ "$GOTRUE_DB_DATABASE_URL" == "$expected_database_url" ]] || fail_config 'GOTRUE_DB_DATABASE_URL must match the PostgreSQL settings, internal db:5432 endpoint, and auth search path'
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  ENV_FILE="${1:-}"
  validate_config
fi
