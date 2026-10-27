#!/usr/bin/env bash
# VM-only tool container: its entrypoint is the Go operator, never PostgreSQL.
# This script creates no database server. It must not be run on a local test DB.
set -euo pipefail

refuse() { printf 'Develop operation refused: %s\n' "$1" >&2; exit 2; }
[[ "$#" == 6 ]] || refuse 'usage: run-vm-develop-operator.sh RELEASE_ROOT PRIVATE_ENV OPERATOR AUTHORIZATION provision|extensions|infra|feature|verify RUN_ID'
release="$1"; private_env="$2"; operator="$3"; authorization="$4"; operation="$5"; run_id="$6"
[[ -n "$operator" && -n "$authorization" ]] || refuse 'explicit operator and owner authorization are required'
[[ "$run_id" =~ ^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$ ]] || refuse 'a safe run ID is required'
for path in "$release" "$private_env"; do
  [[ "$path" =~ ^/[A-Za-z0-9._/-]+$ && "$path" != / && "$path" != *'/../'* ]] || refuse 'safe absolute private/release paths are required'
done
[[ -d "$release" && -f "$private_env" ]] || refuse 'release directory and private configuration must exist'
[[ -f "$release/bin/mindforge-infra" && -f "$release/bin/verify-brain-core" ]] || refuse 'packaged Linux binaries are required'
case "$operation" in provision|extensions|infra|feature|verify) ;; *) refuse 'operation is outside the develop-only boundary' ;; esac

image='pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b'
invoke() {
  local binary="$1"; shift
  docker run --rm --platform linux/amd64 --network infra_default \
    --read-only --tmpfs /tmp:rw,nosuid,nodev,size=32m \
    --entrypoint "/workspace/bin/$binary" \
    --mount "type=bind,source=$release,target=/workspace,readonly" \
    --mount "type=bind,source=$private_env,target=/run/mindforge.env,readonly" \
    "$image" "$@"
}
base=(--config /run/mindforge.env --target mindforge_develop
      --operator-id "$operator" --authorization-ref "$authorization")
case "$operation" in
  provision)
    invoke mindforge-infra provision-develop "${base[@]}"
    ;;
  extensions)
    invoke mindforge-infra bootstrap-develop-extensions "${base[@]}" --source-root /workspace
    ;;
  infra|feature)
    scope='infrastructure'
    preflight_extra=(--allow-infrastructure-bootstrap)
    if [[ "$operation" == feature ]]; then scope='feature'; preflight_extra=(); fi
    result="$(invoke mindforge-infra preflight "${base[@]}" --source-root /workspace --scope "$scope" "${preflight_extra[@]}")" || refuse 'read-only preflight failed'
    reference="$(printf '%s\n' "$result" | sed -n 's/^preflight-reference=//p')"
    [[ "$reference" == preflight/mindforge_develop/* && "$reference" != *$'\n'* ]] || refuse 'preflight must identify exactly the develop target'
    invoke mindforge-infra apply "${base[@]}" --source-root /workspace --scope "$scope" \
      --preflight-ref "$reference" --verification-ref "verification/brain-core/$run_id"
    ;;
  verify)
    invoke verify-brain-core --config /run/mindforge.env --operator-id "$operator" \
      --authorization-ref "$authorization" --run-id "$run_id" --allow-synthetic-writes
    ;;
esac
