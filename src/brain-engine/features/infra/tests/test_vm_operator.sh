#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
mkdir -p "$work/release/bin" "$work/bin"
touch "$work/release/bin/mindforge-infra" "$work/release/bin/verify-brain-core" "$work/private.env"
cat > "$work/bin/docker" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$@" >> "$FAKE_LOG"
if [[ "$*" == *" provision-develop "* && "$*" == *" --source-root "* ]]; then
  printf 'unsupported provision-develop flag\n' >&2; exit 2
fi
if [[ "$*" == *" preflight "* ]]; then
  printf 'preflight-reference=preflight/mindforge_develop/synthetic\n'
fi
EOF
chmod +x "$work/bin/docker"
export PATH="$work/bin:$PATH" FAKE_LOG="$work/calls"
wrapper="$ROOT/scripts/run-vm-develop-operator.sh"
bash "$wrapper" "$work/release" "$work/private.env" operator owner-approved infra proof-run
[[ "$(rg -c '^--rm$' "$FAKE_LOG")" == 2 ]]
rg -q '^--target$' "$FAKE_LOG"
rg -q '^mindforge_develop$' "$FAKE_LOG"
rg -q '^--entrypoint$' "$FAKE_LOG"
rg -q '^/workspace/bin/mindforge-infra$' "$FAKE_LOG"
! rg -q '^postgres$|^--privileged$|PGPASSWORD=|^mindforge$' "$FAKE_LOG"
bash "$wrapper" "$work/release" "$work/private.env" operator owner-approved provision proof-run
before="$(wc -l < "$FAKE_LOG")"
if bash "$wrapper" "$work/release" "$work/private.env" operator '' infra proof-run >/dev/null 2>&1; then
  printf 'FAIL: missing authorization accepted\n' >&2; exit 1
fi
if bash "$wrapper" "$work/release" "$work/private.env" operator owner-approved promote proof-run >/dev/null 2>&1; then
  printf 'FAIL: main promotion accepted\n' >&2; exit 1
fi
[[ "$(wc -l < "$FAKE_LOG")" == "$before" ]]
printf 'PASS: VM operator target, authority, entrypoint and private-file boundaries (fake Docker only)\n'
