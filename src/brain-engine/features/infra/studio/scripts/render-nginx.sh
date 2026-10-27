#!/usr/bin/env bash
set -euo pipefail

fail_render() {
  printf 'Cannot render Studio Nginx template: %s\n' "$1" >&2
  exit 1
}

usage() {
  printf '%s\n' 'usage: render-nginx.sh --portal-root ABSOLUTE_PATH --auth-file ABSOLUTE_PATH --output NEW_ABSOLUTE_PATH'
}

require_safe_path() {
  local label="$1"
  local value="$2"
  [[ "$value" == /* && "$value" != "/" && "$value" != */ ]] || fail_render "$label must be an absolute non-root path"
  [[ "$value" =~ ^/[A-Za-z0-9._/-]+$ ]] || fail_render "$label contains unsupported characters"
  case "/$value/" in
    */../*) fail_render "$label cannot contain parent-directory segments" ;;
  esac
}

portal_root=''
auth_file=''
output_file=''
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --portal-root)
      [[ "$#" -ge 2 ]] || fail_render 'missing --portal-root value'
      portal_root="$2"
      shift 2
      ;;
    --auth-file)
      [[ "$#" -ge 2 ]] || fail_render 'missing --auth-file value'
      auth_file="$2"
      shift 2
      ;;
    --output)
      [[ "$#" -ge 2 ]] || fail_render 'missing --output value'
      output_file="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail_render 'unknown argument'
      ;;
  esac
done

[[ -n "$portal_root" && -n "$auth_file" && -n "$output_file" ]] || {
  usage >&2
  fail_render 'all three explicit paths are required'
}
require_safe_path 'portal root' "$portal_root"
require_safe_path 'auth file path' "$auth_file"
require_safe_path 'output path' "$output_file"

output_dir="$(dirname "$output_file")"
[[ -d "$output_dir" && -w "$output_dir" ]] || fail_render 'output directory must already exist and be writable'
[[ ! -e "$output_file" && ! -L "$output_file" ]] || fail_render 'output path already exists'

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
template="$script_dir/../nginx/hosts.conf.template"
[[ -r "$template" ]] || fail_render 'host template is missing'

umask 077
temporary_file="$(mktemp "${output_file}.tmp.XXXXXX")" || fail_render 'could not create a temporary output'
trap 'rm -f "$temporary_file"' EXIT
sed \
  -e "s|@PORTAL_ROOT@|$portal_root|g" \
  -e "s|@AUTH_FILE@|$auth_file|g" \
  "$template" > "$temporary_file"
if grep -Fq '@PORTAL_ROOT@' "$temporary_file" || grep -Fq '@AUTH_FILE@' "$temporary_file"; then
  fail_render 'template contains an unresolved required path'
fi
chmod 600 "$temporary_file"
mv "$temporary_file" "$output_file"
trap - EXIT
printf 'Nginx host template rendered; no route was installed or reloaded.\n'
