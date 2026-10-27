"""Database-free TDD for the deploy package's source and secret boundary."""
import importlib.util
from pathlib import Path
import tempfile
import unittest

SCRIPT = Path(__file__).resolve().parents[1] / "scripts/prepare-develop-studio-package.py"
spec = importlib.util.spec_from_file_location("package_builder", SCRIPT)
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class PackageBoundaryTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = (Path(self.tmp.name) / "repo").resolve()
        self.root.mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def put(self, relative, content="synthetic"):
        path = self.root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content)
        return path

    def test_packages_untracked_canonical_sources_but_never_private_files(self):
        wanted = [
            "src/brain-engine/go.mod",
            "src/brain-engine/features/brain/migrations/0001_brain_core.sql",
            "src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql",
            "src/brain-engine/features/infra/postgres/feature-migrations/catalog.json",
            "src/brain-engine/features/infra/.env.example",
            "src/brain-engine/features/infra/studio/.env.example",
            "src/brain-engine/features/infra/studio/portal/index.html",
            "src/brain-engine/features/infra/studio/nginx.conf.template",
            "src/brain-engine/features/infra/tools/internal/config/config.go",
        ]
        forbidden = [
            "src/brain-engine/features/infra/.env",
            "src/brain-engine/features/infra/.env.vm.local",
            "src/brain-engine/features/infra/studio/.env",
            "src/brain-engine/features/infra/studio/secrets/private.json",
            "src/brain-engine/features/infra/studio/keys/private.key",
            "src/brain-engine/features/infra/studio/passwords.htpasswd",
            "src/brain-engine/features/infra/studio/data.dump",
            "src/brain-engine/features/infra/studio/result.evidence.json",
        ]
        for name in wanted + forbidden:
            self.put(name)
        selected = {p.relative_to(self.root).as_posix() for p in builder.source_files(self.root)}
        self.assertEqual(selected, set(wanted))

    def test_rejects_symlink_source_instead_of_exporting_its_target(self):
        outside = Path(self.tmp.name) / "owner-private.sql"
        outside.write_text("must never be packaged")
        path = self.root / "src/brain-engine/features/brain/migrations/leak.sql"
        path.parent.mkdir(parents=True)
        path.symlink_to(outside)
        with self.assertRaises(ValueError):
            builder.source_files(self.root)

    def test_output_must_be_outside_checkout(self):
        with self.assertRaises(ValueError):
            builder.validate_output(self.root, self.root / "deploy")
        self.assertEqual(builder.validate_output(self.root, Path(self.tmp.name) / "out"),
                         (Path(self.tmp.name) / "out").resolve())

    def test_manifest_detects_actual_bytes_and_is_sorted(self):
        a = self.put("src/brain-engine/a.go", "first")
        b = self.put("src/brain-engine/b.go", "second")
        first = builder.source_manifest(self.root, [b, a])
        self.assertEqual([line.split("  ", 1)[1] for line in first.splitlines()],
                         ["src/brain-engine/a.go", "src/brain-engine/b.go"])
        a.write_text("changed")
        self.assertNotEqual(first, builder.source_manifest(self.root, [b, a]))


if __name__ == "__main__":
    unittest.main()
