#!/usr/bin/env bash
set -euo pipefail

studio_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 -m unittest discover -s "$studio_root/tests" -p 'test_*.py'
bash -n "$studio_root/scripts/validate-config.sh"
bash -n "$studio_root/scripts/render-nginx.sh"
bash -n "$studio_root/tests/run.sh"
