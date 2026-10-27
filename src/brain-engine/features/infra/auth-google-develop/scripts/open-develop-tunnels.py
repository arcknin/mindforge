#!/usr/bin/env python3
from __future__ import annotations

import argparse
import shutil
import subprocess
import sys

from private_config import ConfigError, validate_ssh


def build_ssh_args(values: dict[str, str]) -> list[str]:
    return [
        "ssh",
        "-F",
        "/dev/null",
        "-N",
        "-T",
        "-o",
        "StrictHostKeyChecking=yes",
        "-o",
        "ExitOnForwardFailure=yes",
        "-p",
        values["VM_SSH_PORT"],
        "-L",
        "127.0.0.1:11001:127.0.0.1:18087",
        f"{values['VM_SSH_USER']}@{values['VM_SSH_HOST']}",
    ]


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Open the approved loopback-only Brain Engine API SSH forward."
    )
    parser.add_argument("--file", required=True, help="path to a mode-0600 private SSH endpoint file")
    args = parser.parse_args(argv)
    try:
        values = validate_ssh(args.file)
    except ConfigError as error:
        print(f"SSH tunnel configuration rejected: {error}", file=sys.stderr)
        return 1

    if shutil.which("ssh") is None:
        print("OpenSSH client is unavailable", file=sys.stderr)
        return 1
    try:
        result = subprocess.run(build_ssh_args(values), check=False)
    except OSError:
        print("SSH tunnel could not start", file=sys.stderr)
        return 1
    if result.returncode != 0:
        print("SSH tunnel ended with an error; no existing local listener was stopped", file=sys.stderr)
        return result.returncode if 1 <= result.returncode <= 255 else 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
