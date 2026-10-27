#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sys

from private_config import ConfigError, validate


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Validate one private Mindforge manual-online environment file without displaying its values."
    )
    parser.add_argument("kind", choices=("google", "runtime", "ssh"))
    parser.add_argument("--file", required=True, help="path to a mode-0600 private environment file")
    args = parser.parse_args(argv)
    try:
        validate(args.kind, args.file)
    except ConfigError as error:
        print(f"configuration rejected: {error}", file=sys.stderr)
        return 1
    print(f"{args.kind} configuration is valid; sensitive values omitted")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
