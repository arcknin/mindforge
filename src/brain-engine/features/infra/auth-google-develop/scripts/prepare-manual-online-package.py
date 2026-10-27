#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import stat
import subprocess
import sys
import tempfile
from pathlib import Path


FEATURE_ROOT = Path(__file__).resolve().parents[1]
REPO_ROOT = next((path for path in FEATURE_ROOT.parents if (path / ".git").exists()), None)
SOURCE_ROOTS = ("cmd/brain-engine", "api", "auth", "features/brain", "features/documents")
PACKAGE_FILES = (
    (FEATURE_ROOT / ".dockerignore", ".dockerignore"),
    (FEATURE_ROOT / "Dockerfile", "Dockerfile"),
    (FEATURE_ROOT / "compose.brain-engine-develop.yaml", "compose.yaml"),
    (FEATURE_ROOT / "PACKAGE-RUNBOOK.md", "README.md"),
    (FEATURE_ROOT / "brain-engine-runtime.env.example", ".env.runtime.example"),
    (FEATURE_ROOT / "scripts" / "validate-config.py", "scripts/validate-config.py"),
    (FEATURE_ROOT / "scripts" / "private_config.py", "scripts/private_config.py"),
)
FORBIDDEN_SUFFIXES = {
    ".pem",
    ".key",
    ".p12",
    ".pfx",
    ".tmp",
    ".temp",
    ".swp",
    ".bak",
    ".orig",
}


class PackageError(ValueError):
    pass


def _digest(path: Path) -> str:
    hasher = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            hasher.update(block)
    return hasher.hexdigest()


def _is_within(path: Path, parent: Path) -> bool:
    try:
        path.relative_to(parent)
        return True
    except ValueError:
        return False


def _reject_unsafe_inputs(engine_root: Path) -> list[Path]:
    main_file = engine_root / "cmd" / "brain-engine" / "main.go"
    if not main_file.is_file() or main_file.is_symlink():
        raise PackageError("Brain Engine command entrypoint is unavailable")
    source_files: list[Path] = []
    for relative in SOURCE_ROOTS:
        root = engine_root / relative
        if not root.exists() or not root.is_dir() or root.is_symlink():
            raise PackageError("required Brain Engine source directory is unavailable")
        go_files = 0
        for current, directories, files in os.walk(root, followlinks=False):
            current_path = Path(current)
            for name in list(directories):
                child = current_path / name
                if child.is_symlink():
                    raise PackageError("source tree contains a symlink")
                if _unsafe_name(name):
                    raise PackageError("source tree contains a private, secret, or temporary path")
            for name in files:
                child = current_path / name
                if child.is_symlink():
                    raise PackageError("source tree contains a symlink")
                if _unsafe_name(name):
                    raise PackageError("source tree contains a private, secret, or temporary path")
                try:
                    metadata = child.lstat()
                except OSError:
                    raise PackageError("source tree changed while it was being inspected") from None
                if not stat.S_ISREG(metadata.st_mode):
                    raise PackageError("source tree contains a non-regular file")
                if child.suffix == ".go":
                    source_files.append(child)
                    go_files += 1
        if go_files == 0:
            raise PackageError("required Brain Engine source files are unavailable")

    for filename in ("go.mod", "go.sum"):
        path = engine_root / filename
        if not path.is_file() or path.is_symlink():
            raise PackageError("Go module metadata is unavailable")
        source_files.append(path)
    source_files.sort(key=lambda item: item.relative_to(engine_root).as_posix())
    return source_files


def _unsafe_name(name: str) -> bool:
    lower = name.lower()
    if (
        lower == ".env"
        or lower.startswith(".env.")
        or lower.endswith(".env")
        or lower.endswith(".private")
        or lower.endswith("~")
        or lower in {".ds_store", "__pycache__"}
        or any(token in lower for token in ("secret", "credential"))
        or Path(lower).suffix in FORBIDDEN_SUFFIXES
    ):
        return True
    return False


def _copy_package_file(source: Path, destination: Path) -> None:
    if not source.is_file() or source.is_symlink():
        raise PackageError("a required package file is unavailable")
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, destination)


def prepare_package(repo_root: str | Path, output_path: str | Path, goarch: str) -> Path:
    if goarch not in {"amd64", "arm64"}:
        raise PackageError("GOARCH must be amd64 or arm64")
    root = Path(repo_root).resolve()
    if not (root / ".git").exists():
        raise PackageError("repository root could not be confirmed")
    engine_root = root / "src" / "brain-engine"
    source_files = _reject_unsafe_inputs(engine_root)

    requested_output = Path(output_path).expanduser()
    if requested_output.is_symlink():
        raise PackageError("package destination must not be a symlink")
    output = requested_output.resolve()
    if _is_within(output, root):
        raise PackageError("package destination must be outside the repository")
    if output.exists() and (not output.is_dir() or any(output.iterdir())):
        raise PackageError("package destination must be a missing path or an empty directory")
    requested_gocache = os.environ.get("GOCACHE")
    if requested_gocache == "":
        raise PackageError("GOCACHE must not be empty")
    gocache_value = requested_gocache or "/tmp/mindforge-manual-online-gocache"
    gomodcache_value = os.environ.get("GOMODCACHE")

    try:
        output.parent.mkdir(parents=True, exist_ok=True)
        stage_context = tempfile.TemporaryDirectory(prefix="mindforge-manual-online-", dir=output.parent)
    except OSError:
        raise PackageError("package destination parent is not writable") from None

    stage = Path(stage_context.name)
    try:
        binary = stage / "brain-engine"
        build_env = {
            "PATH": os.environ.get("PATH", "/usr/bin:/bin"),
            "HOME": str(Path.home()),
            "TMPDIR": os.environ.get("TMPDIR", "/tmp"),
            "GOOS": "linux",
            "GOARCH": goarch,
            "CGO_ENABLED": "0",
            "GOPROXY": "off",
            "GOSUMDB": "off",
            "GOTOOLCHAIN": "local",
            "GOENV": "off",
            "GOVCS": "*:off",
            "GOFLAGS": "",
            "GOCACHE": gocache_value,
        }
        if gomodcache_value is not None:
            build_env["GOMODCACHE"] = gomodcache_value
        cache_paths = [("GOCACHE", gocache_value)]
        if gomodcache_value is not None:
            cache_paths.append(("GOMODCACHE", gomodcache_value))
        for variable, value in cache_paths:
            cache_path = Path(value)
            if not cache_path.is_absolute():
                cache_path = engine_root / cache_path
            cache_path = cache_path.resolve()
            if _is_within(cache_path, output) or _is_within(cache_path, stage):
                raise PackageError(f"{variable} must be outside the package and build staging directories")
        command = [
            "go",
            "build",
            "-trimpath",
            "-buildvcs=false",
            "-mod=readonly",
            "-ldflags=-s -w",
            "-o",
            str(binary),
            "./cmd/brain-engine",
        ]
        try:
            result = subprocess.run(
                command,
                cwd=engine_root,
                env=build_env,
                stdin=subprocess.DEVNULL,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
                timeout=600,
            )
        except (OSError, subprocess.TimeoutExpired):
            raise PackageError("offline Linux Go build could not be completed; details omitted") from None
        if result.returncode != 0:
            raise PackageError("offline Linux Go build failed; details omitted")
        if binary.is_symlink() or not binary.is_file() or binary.stat().st_size == 0:
            raise PackageError("Go build did not produce a regular executable")
        binary.chmod(0o755)

        for source, package_name in PACKAGE_FILES:
            _copy_package_file(source, stage / package_name)
        package_hashes = []
        for relative in sorted(
            path
            for path in stage.rglob("*")
            if path.is_file() and path.name not in {"source-manifest.json", "brain-engine"}
        ):
            package_hashes.append(
                {
                    "path": relative.relative_to(stage).as_posix(),
                    "sha256": _digest(relative),
                }
            )
        manifest = {
            "schema_version": 1,
            "target": {"goos": "linux", "goarch": goarch, "cgo_enabled": False, "network_access": "disabled"},
            "binary": {
                "path": "brain-engine",
                "sha256": _digest(binary),
                "size_bytes": binary.stat().st_size,
            },
            "sources": [
                {
                    "path": path.relative_to(engine_root).as_posix(),
                    "sha256": _digest(path),
                }
                for path in source_files
            ],
            "package_files": package_hashes,
        }
        (stage / "source-manifest.json").write_text(
            json.dumps(manifest, indent=2, sort_keys=True) + "\n",
            encoding="utf-8",
        )

        if output.exists():
            try:
                output.rmdir()
            except OSError:
                raise PackageError("package destination is no longer empty") from None
        os.replace(stage, output)
    except PackageError:
        raise
    except OSError:
        raise PackageError("package output could not be completed; partial files were removed") from None
    finally:
        stage_context.cleanup()
    return output


def _default_repo_root() -> Path:
    if REPO_ROOT is None:
        raise PackageError("repository root could not be confirmed")
    return REPO_ROOT


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Build a credential-free, offline Linux Brain Engine package outside the Git repository."
    )
    parser.add_argument("--output", required=True, help="new or empty directory outside the repository")
    parser.add_argument("--goarch", choices=("amd64", "arm64"), default="amd64")
    args = parser.parse_args(argv)
    try:
        output = prepare_package(_default_repo_root(), args.output, args.goarch)
    except PackageError as error:
        print(f"package preparation failed: {error}", file=sys.stderr)
        return 1
    print(f"credential-free Linux/{args.goarch} package prepared; manifest: {output / 'source-manifest.json'}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
