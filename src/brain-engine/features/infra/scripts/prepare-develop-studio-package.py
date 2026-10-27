#!/usr/bin/env python3
"""Prepare immutable local source/build artifacts. Never contacts the VM or DB."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile

SOURCE_EXTENSIONS = {".go", ".sql", ".json", ".yaml", ".yml", ".sh", ".md",
                     ".html", ".css", ".conf", ".template", ".py", ".txt"}
PRIVATE_PARTS = {"secrets", "keys", "tokens", "certs", "logs", "tmp", "backups",
                 "__pycache__", ".git"}


def source_files(root):
    root = Path(root).resolve()
    selected = []
    tree = root / "src/brain-engine"
    for path in tree.rglob("*"):
        relative = path.relative_to(root)
        if any(part in PRIVATE_PARTS for part in relative.parts):
            continue
        if any(part.startswith(".") and part != ".env.example" for part in relative.parts):
            continue
        allowed = (path.name in {"go.mod", "go.sum", ".env.example"}
                   or path.suffix in SOURCE_EXTENSIONS)
        if not allowed or path.name.endswith((".manifest.json", ".evidence.json")):
            continue
        if path.is_symlink():
            raise ValueError("symlink sources cannot enter the deployment package")
        if path.is_file():
            selected.append(path)
    return sorted(selected)


def validate_output(root, output):
    root, output = Path(root).resolve(), Path(output).resolve()
    if output == root or root in output.parents:
        raise ValueError("output must be outside the checkout")
    return output


def source_manifest(root, files):
    root = Path(root).resolve()
    files = [Path(path).resolve() for path in files]
    return "".join(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  "
                   f"{path.relative_to(root).as_posix()}\n" for path in sorted(files))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-root", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--goarch", choices=("amd64", "arm64"), required=True)
    args = parser.parse_args()
    root = args.source_root.resolve()
    try:
        output = validate_output(root, args.output_dir)
        if output.exists() and any(output.iterdir()):
            raise ValueError("output must be a new or empty directory")
        files = source_files(root)
        required = ["go.mod", "go.sum", "features/infra/studio/compose.yaml",
                    "features/brain/migrations/0001_brain_core.sql",
                    "features/brain/migrations/0001_brain_core.preflight.sql",
                    "features/brain/migrations/0001_brain_core.verify.sql",
                    "features/infra/postgres/feature-migrations/catalog.json",
                    "features/infra/tools/cmd/verify-brain-core/main.go"]
        if any(not (root / "src/brain-engine" / name).is_file() for name in required):
            raise ValueError("required source or runtime verifier is incomplete")
        output.mkdir(mode=0o700, parents=True, exist_ok=True)
        bin_dir = output / "bin"
        bin_dir.mkdir(mode=0o700)
        env = dict(os.environ, GOOS="linux", GOARCH=args.goarch, CGO_ENABLED="0",
                   GOTOOLCHAIN="local", GOPROXY="off")
        env.setdefault("GOCACHE", "/private/tmp/mindforge-develop-studio-gocache")
        for command in ("mindforge-infra", "verify-brain-core"):
            result = subprocess.run(
                ["go", "build", "-trimpath", "-o", str(bin_dir / command),
                 f"./features/infra/tools/cmd/{command}"],
                cwd=root / "src/brain-engine", env=env, capture_output=True)
            if result.returncode:
                raise ValueError(f"local Linux build failed for {command}; no deployment occurred")
        manifest = source_manifest(root, files)
        (output / "source-sha256.txt").write_text(manifest)
        archive = output / "mindforge-develop-studio.tar.gz"
        with tarfile.open(archive, "w:gz") as tar:
            for path in files:
                tar.add(path, arcname=path.relative_to(root).as_posix(), recursive=False)
            for path in sorted(bin_dir.iterdir()):
                tar.add(path, arcname=f"bin/{path.name}", recursive=False)
            tar.add(output / "source-sha256.txt", arcname="source-sha256.txt")
        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        (output / "package-sha256.txt").write_text(f"{digest}  {archive.name}\n")
        (output / "package.json").write_text(json.dumps({
            "schema_version": 1, "platform": f"linux/{args.goarch}",
            "source_files": len(files), "archive_sha256": digest,
            "authority": "local_preparation_only", "contains_private_configuration": False,
        }, indent=2) + "\n")
        print(f"Local package prepared: {output}")
        print(f"Source files: {len(files)}; platform: linux/{args.goarch}; SHA-256: {digest}")
        print("No VM, Docker daemon, PostgreSQL or external operation was executed.")
        return 0
    except (ValueError, OSError) as exc:
        print(f"Package preparation refused: {exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
