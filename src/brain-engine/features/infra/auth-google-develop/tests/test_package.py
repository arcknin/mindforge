from __future__ import annotations

import hashlib
import importlib.util
import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BUILDER = ROOT / "scripts" / "prepare-manual-online-package.py"


def load_builder():
    spec = importlib.util.spec_from_file_location("manual_online_builder", BUILDER)
    if spec is None or spec.loader is None:
        raise AssertionError("package builder module could not be loaded")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def make_fake_repo(parent: Path) -> Path:
    parent.mkdir(parents=True, exist_ok=True)
    repo = parent / "repo"
    (repo / ".git").mkdir(parents=True)
    (repo / "src" / "brain-engine" / "cmd" / "brain-engine").mkdir(parents=True)
    for directory in ("api", "auth", "features/brain", "features/documents"):
        (repo / "src" / "brain-engine" / directory).mkdir(parents=True)
    (repo / "src" / "brain-engine" / "go.mod").write_text("module example.test/brain\n\ngo 1.23\n", encoding="utf-8")
    (repo / "src" / "brain-engine" / "go.sum").write_text("example.test/dependency v1.0.0 h1:fixture\n", encoding="utf-8")
    for path, contents in {
        "cmd/brain-engine/main.go": "package main\nfunc main() {}\n",
        "api/server.go": "package api\n",
        "auth/verify.go": "package auth\n",
        "features/brain/engine.go": "package brain\n",
        "features/documents/engine.go": "package documents\n",
    }.items():
        target = repo / "src" / "brain-engine" / path
        target.write_text(contents, encoding="utf-8")
    return repo


def install_fake_go(parent: Path) -> Path:
    directory = parent / "bin"
    directory.mkdir()
    fake_go = directory / "go"
    fake_go.write_text(
        "#!/usr/bin/env python3\n"
        "import json, os, pathlib, sys\n"
        "args=sys.argv[1:]\n"
        "out=pathlib.Path(args[args.index('-o')+1])\n"
        "out.write_bytes(b'fixture static linux binary')\n"
        "record={k: os.environ.get(k) for k in ('GOOS','GOARCH','CGO_ENABLED','GOPROXY','GOSUMDB','GOTOOLCHAIN','GOCACHE','GOMODCACHE')}\n"
        "record['sensitive_env']={k: os.environ.get(k) for k in ('MINDFORGE_DATABASE_URL','GOOGLE_AUTH_GOOGLE_SECRET')}\n"
        "record['args']=args\n"
        "(pathlib.Path.home()/'fake-go.json').write_text(json.dumps(record), encoding='utf-8')\n",
        encoding="utf-8",
    )
    fake_go.chmod(0o755)
    return directory


class ManualOnlinePackageTests(unittest.TestCase):
    def test_builder_uses_static_linux_offline_build_and_only_allowlisted_package_files(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-") as temp:
            parent = Path(temp)
            repo = make_fake_repo(parent)
            record = parent / "fake-go.json"
            fake_bin = install_fake_go(parent)
            previous_path = os.environ.get("PATH", "")
            previous_home = os.environ.get("HOME")
            previous_db_url = os.environ.get("MINDFORGE_DATABASE_URL")
            previous_google_secret = os.environ.get("GOOGLE_AUTH_GOOGLE_SECRET")
            previous_gocache = os.environ.get("GOCACHE")
            previous_gomodcache = os.environ.get("GOMODCACHE")
            os.environ["PATH"] = f"{fake_bin}:{previous_path}"
            os.environ["HOME"] = str(parent)
            os.environ["MINDFORGE_DATABASE_URL"] = "postgres://sensitive-fixture"
            os.environ["GOOGLE_AUTH_GOOGLE_SECRET"] = "sensitive-google-fixture"
            os.environ["GOCACHE"] = str(parent / "go-cache")
            os.environ["GOMODCACHE"] = str(parent / "go-module-cache")
            try:
                output = parent / "release" / "manual-online"
                builder.prepare_package(repo_root=repo, output_path=output, goarch="arm64")
            finally:
                os.environ["PATH"] = previous_path
                if previous_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = previous_home
                if previous_db_url is None:
                    os.environ.pop("MINDFORGE_DATABASE_URL", None)
                else:
                    os.environ["MINDFORGE_DATABASE_URL"] = previous_db_url
                if previous_google_secret is None:
                    os.environ.pop("GOOGLE_AUTH_GOOGLE_SECRET", None)
                else:
                    os.environ["GOOGLE_AUTH_GOOGLE_SECRET"] = previous_google_secret
                if previous_gocache is None:
                    os.environ.pop("GOCACHE", None)
                else:
                    os.environ["GOCACHE"] = previous_gocache
                if previous_gomodcache is None:
                    os.environ.pop("GOMODCACHE", None)
                else:
                    os.environ["GOMODCACHE"] = previous_gomodcache

            build = json.loads(record.read_text(encoding="utf-8"))
            self.assertEqual(
                {key: build[key] for key in ("GOOS", "GOARCH", "CGO_ENABLED", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN")},
                {
                    "GOOS": "linux",
                    "GOARCH": "arm64",
                    "CGO_ENABLED": "0",
                    "GOPROXY": "off",
                    "GOSUMDB": "off",
                    "GOTOOLCHAIN": "local",
                },
            )
            self.assertEqual(build["GOCACHE"], str(parent / "go-cache"))
            self.assertEqual(build["GOMODCACHE"], str(parent / "go-module-cache"))
            self.assertEqual(build["args"][0], "build")
            self.assertIn("./cmd/brain-engine", build["args"])
            self.assertIn("-trimpath", build["args"])
            self.assertIn("-buildvcs=false", build["args"])
            self.assertIn("-mod=readonly", build["args"])
            self.assertEqual(build["sensitive_env"], {"MINDFORGE_DATABASE_URL": None, "GOOGLE_AUTH_GOOGLE_SECRET": None})
            self.assertEqual(
                {path.relative_to(output).as_posix() for path in output.rglob("*") if path.is_file()},
                {
                    ".dockerignore",
                    "brain-engine",
                    "Dockerfile",
                    "compose.yaml",
                    "README.md",
                    ".env.runtime.example",
                    "scripts/validate-config.py",
                    "scripts/private_config.py",
                    "source-manifest.json",
                },
            )
            self.assertEqual((output / "brain-engine").read_bytes(), b"fixture static linux binary")
            runtime_example = (output / ".env.runtime.example").read_text(encoding="utf-8")
            self.assertIn(
                "MINDFORGE_AUTH_CALLBACK_URL=http://127.0.0.1:11001/v1/auth/callback",
                runtime_example,
            )
            self.assertIn(
                "MINDFORGE_AUTH_REDIRECT_URLS=mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback",
                runtime_example,
            )
            manifest = json.loads((output / "source-manifest.json").read_text(encoding="utf-8"))
            expected_hash = hashlib.sha256(b"package api\n").hexdigest()
            entry = next(item for item in manifest["sources"] if item["path"] == "api/server.go")
            self.assertEqual(entry["sha256"], expected_hash)
            self.assertIn(
                "features/documents/engine.go",
                {item["path"] for item in manifest["sources"]},
            )
            self.assertEqual(manifest["target"]["goarch"], "arm64")

    def test_builder_defaults_go_cache_outside_repository_and_keeps_it_out_of_package(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-default-cache-") as temp:
            parent = Path(temp)
            repo = make_fake_repo(parent)
            record = parent / "fake-go.json"
            fake_bin = install_fake_go(parent)
            previous_path = os.environ.get("PATH", "")
            previous_home = os.environ.get("HOME")
            previous_gocache = os.environ.pop("GOCACHE", None)
            previous_gomodcache = os.environ.pop("GOMODCACHE", None)
            os.environ["PATH"] = f"{fake_bin}:{previous_path}"
            os.environ["HOME"] = str(parent)
            try:
                output = parent / "release" / "manual-online"
                builder.prepare_package(repo_root=repo, output_path=output, goarch="amd64")
            finally:
                os.environ["PATH"] = previous_path
                if previous_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = previous_home
                if previous_gocache is not None:
                    os.environ["GOCACHE"] = previous_gocache
                if previous_gomodcache is not None:
                    os.environ["GOMODCACHE"] = previous_gomodcache

            build = json.loads(record.read_text(encoding="utf-8"))
            self.assertEqual(build["GOCACHE"], "/tmp/mindforge-manual-online-gocache")
            self.assertIsNone(build["GOMODCACHE"])
            cache_path = Path(build["GOCACHE"]).resolve()
            self.assertFalse(cache_path.is_relative_to(repo.resolve()))
            self.assertFalse(cache_path.is_relative_to(output.resolve()))
            self.assertNotIn("go-cache", {path.name for path in output.rglob("*")})

    def test_builder_refuses_output_inside_repo_or_nonempty_destination_without_overwriting(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-guard-") as temp:
            parent = Path(temp)
            repo = make_fake_repo(parent)
            with self.assertRaisesRegex(ValueError, "outside the repository"):
                builder.prepare_package(repo_root=repo, output_path=repo / "out", goarch="amd64")

            output = parent / "existing"
            output.mkdir()
            protected = output / "owner-file.txt"
            protected.write_text("preserve", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "empty directory"):
                builder.prepare_package(repo_root=repo, output_path=output, goarch="amd64")
            self.assertEqual(protected.read_text(encoding="utf-8"), "preserve")

    def test_builder_rejects_symlinks_secret_env_and_temporary_sources_before_build(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-input-guard-") as temp:
            parent = Path(temp)
            fake_bin = install_fake_go(parent)
            previous_path = os.environ.get("PATH", "")
            previous_home = os.environ.get("HOME")
            os.environ["PATH"] = f"{fake_bin}:{previous_path}"
            os.environ["HOME"] = str(parent)
            try:
                for name, prepare in (
                    ("env", lambda repo: (repo / "src/brain-engine/api/.env.private").write_text("SECRET=fixture", encoding="utf-8")),
                    ("secret", lambda repo: (repo / "src/brain-engine/api/oauth-secret.json").write_text("fixture", encoding="utf-8")),
                    ("temporary", lambda repo: (repo / "src/brain-engine/api/handler.go.tmp").write_text("fixture", encoding="utf-8")),
                    ("symlink", lambda repo: (repo / "src/brain-engine/api/outside.go").symlink_to(repo / "src/brain-engine/api/server.go")),
                ):
                    with self.subTest(name=name):
                        repo = make_fake_repo(parent / name)
                        prepare(repo)
                        with self.assertRaises(ValueError):
                            builder.prepare_package(
                                repo_root=repo,
                                output_path=parent / f"out-{name}",
                                goarch="amd64",
                            )
                        self.assertFalse((parent / "fake-go.json").exists())
            finally:
                os.environ["PATH"] = previous_path
                if previous_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = previous_home

    def test_builder_rejects_symlink_or_nonempty_destination(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-output-") as temp:
            parent = Path(temp)
            repo = make_fake_repo(parent)
            target = parent / "target"
            target.mkdir()
            linked = parent / "linked"
            linked.symlink_to(target)
            with self.assertRaises(ValueError):
                builder.prepare_package(repo_root=repo, output_path=linked, goarch="amd64")

    def test_builder_refuses_go_cache_inside_package_destination_before_build(self):
        builder = load_builder()
        with tempfile.TemporaryDirectory(prefix="mindforge-package-cache-guard-") as temp:
            parent = Path(temp)
            repo = make_fake_repo(parent)
            fake_bin = install_fake_go(parent)
            previous_path = os.environ.get("PATH", "")
            previous_home = os.environ.get("HOME")
            previous_gocache = os.environ.get("GOCACHE")
            os.environ["PATH"] = f"{fake_bin}:{previous_path}"
            os.environ["HOME"] = str(parent)
            output = parent / "release" / "manual-online"
            os.environ["GOCACHE"] = str(output / "go-cache")
            try:
                with self.assertRaisesRegex(ValueError, "outside the package"):
                    builder.prepare_package(repo_root=repo, output_path=output, goarch="amd64")
                self.assertFalse((parent / "fake-go.json").exists())
            finally:
                os.environ["PATH"] = previous_path
                if previous_home is None:
                    os.environ.pop("HOME", None)
                else:
                    os.environ["HOME"] = previous_home
                if previous_gocache is None:
                    os.environ.pop("GOCACHE", None)
                else:
                    os.environ["GOCACHE"] = previous_gocache


if __name__ == "__main__":
    unittest.main()
