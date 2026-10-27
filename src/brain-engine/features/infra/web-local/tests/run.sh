#!/usr/bin/env bash
set -euo pipefail

web_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
python3 -m unittest discover -s "$web_root/tests" -p 'test_*.py'
bash -n "$web_root/tests/run.sh"
