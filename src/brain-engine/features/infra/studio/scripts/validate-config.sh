#!/usr/bin/env bash
set -euo pipefail

fail_config() {
  printf 'Invalid Studio configuration: %s\n' "$1" >&2
  exit 1
}

is_allowed_key() {
  case "$1" in
    STUDIO_MAIN_DB_NAME|STUDIO_MAIN_DB_USER|STUDIO_MAIN_DB_PASSWORD|STUDIO_MAIN_META_CRYPTO_KEY|STUDIO_MAIN_AUTH_JWT_SECRET|STUDIO_DEVELOP_DB_NAME|STUDIO_DEVELOP_DB_USER|STUDIO_DEVELOP_DB_PASSWORD|STUDIO_DEVELOP_META_CRYPTO_KEY|STUDIO_DEVELOP_AUTH_JWT_SECRET)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

clear_known_values() {
  unset STUDIO_MAIN_DB_NAME STUDIO_MAIN_DB_USER STUDIO_MAIN_DB_PASSWORD
  unset STUDIO_MAIN_META_CRYPTO_KEY STUDIO_MAIN_AUTH_JWT_SECRET
  unset STUDIO_DEVELOP_DB_NAME STUDIO_DEVELOP_DB_USER STUDIO_DEVELOP_DB_PASSWORD
  unset STUDIO_DEVELOP_META_CRYPTO_KEY STUDIO_DEVELOP_AUTH_JWT_SECRET
}

require_value() {
  local name="$1"
  local value="$2"
  [[ -n "$value" ]] || fail_config "$name is required"
  case "$value" in
    replace-with-*|*SAMPLE_KEY*|*sample-key*)
      fail_config "$name still uses a placeholder or known sample"
      ;;
  esac
}

require_identifier() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  [[ "$value" =~ ^[a-z_][a-z0-9_]*$ ]] || fail_config "$name must be a lowercase SQL role identifier"
}

require_database_password() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  [[ "$value" =~ ^[A-Za-z0-9._~-]+$ ]] || fail_config "$name must use URI-safe characters"
}

require_crypto_key() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  [[ "$value" =~ ^[A-Za-z0-9._~+/=-]+$ ]] || fail_config "$name contains unsupported characters"
  (( ${#value} >= 32 )) || fail_config "$name must contain at least 32 characters"
}

require_jwt_secret() {
  local name="$1"
  local value="$2"
  require_value "$name" "$value"
  [[ "$value" =~ ^[A-Za-z0-9._~-]+$ ]] || fail_config "$name must use URI-safe characters"
  (( ${#value} >= 32 )) || fail_config "$name must contain at least 32 characters"
}

require_distinct() {
  local left_name="$1"
  local left_value="$2"
  local right_name="$3"
  local right_value="$4"
  [[ "$left_value" != "$right_value" ]] || fail_config "$left_name and $right_name must be distinct"
}

load_private_config() {
  local env_file="$1"
  local line key value line_number=0 seen_keys='|'

  [[ -f "$env_file" && -r "$env_file" ]] || fail_config 'private environment file is required and must be readable'
  clear_known_values

  while IFS= read -r line || [[ -n "$line" ]]; do
    line_number=$((line_number + 1))
    line="${line%$'\r'}"
    case "$line" in
      ''|'#'*) continue ;;
    esac

    case "$line" in
      *=*) ;;
      *) fail_config "line $line_number must use KEY=VALUE" ;;
    esac
    key="${line%%=*}"
    value="${line#*=}"
    [[ "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || fail_config "line $line_number has an invalid key"
    is_allowed_key "$key" || fail_config "key $key is not allowed"
    case "$seen_keys" in
      *"|$key|"*) fail_config "key $key is duplicated" ;;
    esac
    seen_keys="${seen_keys}${key}|"
    printf -v "$key" '%s' "$value"
  done < "$env_file"
}

validate_config() {
  require_value STUDIO_MAIN_DB_NAME "${STUDIO_MAIN_DB_NAME:-}"
  require_value STUDIO_DEVELOP_DB_NAME "${STUDIO_DEVELOP_DB_NAME:-}"
  [[ "$STUDIO_MAIN_DB_NAME" == "mindforge" ]] || fail_config 'main database target must be mindforge'
  [[ "$STUDIO_DEVELOP_DB_NAME" == "mindforge_develop" ]] || fail_config 'develop database target must be mindforge_develop'

  require_identifier STUDIO_MAIN_DB_USER "${STUDIO_MAIN_DB_USER:-}"
  require_identifier STUDIO_DEVELOP_DB_USER "${STUDIO_DEVELOP_DB_USER:-}"
  require_distinct STUDIO_MAIN_DB_USER "$STUDIO_MAIN_DB_USER" STUDIO_DEVELOP_DB_USER "$STUDIO_DEVELOP_DB_USER"

  require_database_password STUDIO_MAIN_DB_PASSWORD "${STUDIO_MAIN_DB_PASSWORD:-}"
  require_database_password STUDIO_DEVELOP_DB_PASSWORD "${STUDIO_DEVELOP_DB_PASSWORD:-}"
  require_crypto_key STUDIO_MAIN_META_CRYPTO_KEY "${STUDIO_MAIN_META_CRYPTO_KEY:-}"
  require_crypto_key STUDIO_DEVELOP_META_CRYPTO_KEY "${STUDIO_DEVELOP_META_CRYPTO_KEY:-}"
  require_jwt_secret STUDIO_MAIN_AUTH_JWT_SECRET "${STUDIO_MAIN_AUTH_JWT_SECRET:-}"
  require_jwt_secret STUDIO_DEVELOP_AUTH_JWT_SECRET "${STUDIO_DEVELOP_AUTH_JWT_SECRET:-}"

  local -a sensitive_values
  sensitive_values=(
    "$STUDIO_MAIN_DB_PASSWORD"
    "$STUDIO_MAIN_META_CRYPTO_KEY"
    "$STUDIO_MAIN_AUTH_JWT_SECRET"
    "$STUDIO_DEVELOP_DB_PASSWORD"
    "$STUDIO_DEVELOP_META_CRYPTO_KEY"
    "$STUDIO_DEVELOP_AUTH_JWT_SECRET"
  )
  local i j
  for ((i = 0; i < ${#sensitive_values[@]}; i++)); do
    for ((j = i + 1; j < ${#sensitive_values[@]}; j++)); do
      [[ "${sensitive_values[$i]}" != "${sensitive_values[$j]}" ]] || fail_config 'database passwords and environment keys must be distinct'
    done
  done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [[ "$#" -eq 1 ]] || fail_config 'usage: validate-config.sh PRIVATE_ENV_FILE'
  load_private_config "$1"
  validate_config
  printf 'Studio configuration shape is valid; database credentials and runtime were not checked.\n'
fi
