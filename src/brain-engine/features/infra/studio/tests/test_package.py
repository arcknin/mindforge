from __future__ import annotations

import json
import re
import shutil
import subprocess
import tempfile
import unittest
from html.parser import HTMLParser
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = ROOT / "scripts"
TEST_VALUES = {
    "STUDIO_MAIN_DB_NAME": "mindforge",
    "STUDIO_MAIN_DB_USER": "mf_main_fixture",
    "STUDIO_MAIN_DB_PASSWORD": "MainDbFixture_0123456789",
    "STUDIO_MAIN_META_CRYPTO_KEY": "MainCryptoFixture_012345678901234",
    "STUDIO_MAIN_AUTH_JWT_SECRET": "MainJwtFixture_012345678901234567890",
    "STUDIO_DEVELOP_DB_NAME": "mindforge_develop",
    "STUDIO_DEVELOP_DB_USER": "mf_develop_fixture",
    "STUDIO_DEVELOP_DB_PASSWORD": "DevelopDbFixture_0123456789",
    "STUDIO_DEVELOP_META_CRYPTO_KEY": "DevelopCryptoFixture_0123456789012",
    "STUDIO_DEVELOP_AUTH_JWT_SECRET": "DevelopJwtFixture_01234567890123456789",
}
EXPECTED_SERVICES = {"studio-main", "meta-main", "studio-develop", "meta-develop"}
SENSITIVE_KEYS = {
    "STUDIO_MAIN_DB_PASSWORD",
    "STUDIO_MAIN_META_CRYPTO_KEY",
    "STUDIO_MAIN_AUTH_JWT_SECRET",
    "STUDIO_DEVELOP_DB_PASSWORD",
    "STUDIO_DEVELOP_META_CRYPTO_KEY",
    "STUDIO_DEVELOP_AUTH_JWT_SECRET",
}
PORTAL_LINKS = {
    "http://nsr.db.home.arpa",
    "http://mindforge.db.home.arpa",
    "http://develop.db.home.arpa",
}


def run_validator(values: dict[str, str]) -> subprocess.CompletedProcess[str]:
    with tempfile.TemporaryDirectory(prefix="studio-env-") as temp:
        env_file = Path(temp) / "studio.env"
        env_file.write_text(
            "".join(f"{key}={value}\n" for key, value in values.items()),
            encoding="utf-8",
        )
        return subprocess.run(
            ["bash", str(SCRIPTS / "validate-config.sh"), str(env_file)],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )


class PortalParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__()
        self.lang = None
        self.links: list[tuple[str, str]] = []
        self._href = None
        self._text: list[str] = []
        self.scripts = 0
        self.h1_count = 0

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == "html":
            self.lang = attrs.get("lang")
        if tag == "a":
            self._href = attrs.get("href")
            self._text = []
        if tag == "script":
            self.scripts += 1
        if tag == "h1":
            self.h1_count += 1

    def handle_data(self, data):
        if self._href is not None:
            self._text.append(data)

    def handle_endtag(self, tag):
        if tag == "a" and self._href is not None:
            self.links.append((self._href, " ".join("".join(self._text).split())))
            self._href = None
            self._text = []


class StudioPackageTests(unittest.TestCase):
    def test_package_has_only_the_scoped_deliverables(self):
        required = [
            ROOT / "compose.yaml",
            ROOT / ".env.example",
            ROOT / ".gitignore",
            ROOT / "README.md",
            ROOT / "portal" / "index.html",
            ROOT / "nginx" / "hosts.conf.template",
            SCRIPTS / "validate-config.sh",
            SCRIPTS / "render-nginx.sh",
            ROOT / "tests" / "run.sh",
        ]
        missing = [str(path.relative_to(ROOT)) for path in required if not path.is_file()]
        self.assertEqual(missing, [], f"missing package files: {missing}")

    def test_private_config_validates_synthetic_target_credentials(self):
        example_keys = {
            line.split("=", 1)[0]
            for line in (ROOT / ".env.example").read_text(encoding="utf-8").splitlines()
            if line and not line.startswith("#") and "=" in line
        }
        self.assertEqual(example_keys, set(TEST_VALUES))
        result = run_validator(TEST_VALUES)
        self.assertEqual(result.returncode, 0, "valid synthetic config was rejected")
        self.assertNotIn(TEST_VALUES["STUDIO_MAIN_DB_PASSWORD"], result.stdout + result.stderr)
        self.assertNotIn(TEST_VALUES["STUDIO_DEVELOP_DB_PASSWORD"], result.stdout + result.stderr)

    def test_existing_short_uri_safe_main_password_is_accepted_without_rotation(self):
        values = dict(TEST_VALUES)
        values["STUDIO_MAIN_DB_PASSWORD"] = "m1n-main-7"
        self.assertLess(len(values["STUDIO_MAIN_DB_PASSWORD"]), 16)
        result = run_validator(values)
        self.assertEqual(result.returncode, 0, "existing short main password was rejected")

    def test_private_config_rejects_target_swap(self):
        values = dict(TEST_VALUES)
        values["STUDIO_MAIN_DB_NAME"] = "mindforge_develop"
        result = run_validator(values)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn(TEST_VALUES["STUDIO_MAIN_DB_PASSWORD"], result.stdout + result.stderr)

    def test_private_config_rejects_shared_users_passwords_and_crypto(self):
        mutations = [
            ("STUDIO_DEVELOP_DB_USER", TEST_VALUES["STUDIO_MAIN_DB_USER"]),
            ("STUDIO_DEVELOP_DB_PASSWORD", TEST_VALUES["STUDIO_MAIN_DB_PASSWORD"]),
            ("STUDIO_DEVELOP_META_CRYPTO_KEY", TEST_VALUES["STUDIO_MAIN_META_CRYPTO_KEY"]),
            ("STUDIO_DEVELOP_AUTH_JWT_SECRET", TEST_VALUES["STUDIO_MAIN_AUTH_JWT_SECRET"]),
        ]
        for key, value in mutations:
            with self.subTest(key=key):
                values = dict(TEST_VALUES)
                values[key] = value
                result = run_validator(values)
                self.assertNotEqual(result.returncode, 0)
                combined = result.stdout + result.stderr
                for key in SENSITIVE_KEYS:
                    secret = TEST_VALUES[key]
                    self.assertNotIn(secret, combined)

    def test_private_config_rejects_defaults_unknown_keys_and_missing_values(self):
        invalid_values = [
            {**TEST_VALUES, "STUDIO_MAIN_META_CRYPTO_KEY": "SAMPLE_KEY"},
            {**TEST_VALUES, "STUDIO_MAIN_DB_PASSWORD": ""},
            {**TEST_VALUES, "UNRELATED_SECRET": "not-allowed"},
        ]
        for values in invalid_values:
            with self.subTest(values=set(values)):
                result = run_validator(values)
                self.assertNotEqual(result.returncode, 0)

    def test_private_config_is_not_executed_as_shell_and_rejects_placeholders(self):
        validator = SCRIPTS / "validate-config.sh"
        source = validator.read_text(encoding="utf-8")
        self.assertNotRegex(source, r"(?m)^\s*(source|\.)\s")
        self.assertNotRegex(source, r"\beval\b")

        with tempfile.TemporaryDirectory(prefix="studio-noeval-") as temp:
            marker = Path(temp) / "unexpected"
            values = dict(TEST_VALUES)
            values["STUDIO_MAIN_DB_PASSWORD"] = f"ValidLookingPass123$(touch {marker})"
            result = run_validator(values)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(marker.exists())

        example = subprocess.run(
            ["bash", str(validator), str(ROOT / ".env.example")],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
        )
        self.assertNotEqual(example.returncode, 0)
        for key in SENSITIVE_KEYS:
            self.assertNotIn(TEST_VALUES[key], example.stdout + example.stderr)

    def test_compose_maps_each_pair_to_its_exact_database_and_secret_set(self):
        ruby_parser = (
            'require "yaml"; require "json"; '
            'print JSON.generate(YAML.load_file(ARGV.fetch(0)))'
        )
        parsed = subprocess.run(
            ["ruby", "-e", ruby_parser, str(ROOT / "compose.yaml")],
            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
        )
        self.assertEqual(parsed.returncode, 0, "Compose YAML parse failed")
        config = json.loads(parsed.stdout)
        self.assertEqual(config["name"], "mindforge-studio")
        services = config["services"]
        self.assertEqual(set(services), EXPECTED_SERVICES)

        for target in ("main", "develop"):
            studio = services[f"studio-{target}"]
            meta = services[f"meta-{target}"]
            prefix = f"STUDIO_{target.upper()}_"
            studio_env = studio["environment"]
            meta_env = meta["environment"]
            self.assertEqual(studio["image"], "supabase/studio:2026.09.07-sha-7996410")
            self.assertEqual(meta["image"], "supabase/postgres-meta:v0.99.0")
            self.assertEqual(studio_env["POSTGRES_HOST"], "db")
            self.assertEqual(studio_env["POSTGRES_PORT"], "5432")
            self.assertEqual(studio_env["POSTGRES_DB"], f"${{{prefix}DB_NAME:?}}")
            self.assertEqual(studio_env["POSTGRES_USER_READ_ONLY"], f"${{{prefix}DB_USER:?}}")
            self.assertEqual(studio_env["POSTGRES_USER_READ_WRITE"], f"${{{prefix}DB_USER:?}}")
            self.assertEqual(studio_env["POSTGRES_PASSWORD"], f"${{{prefix}DB_PASSWORD:?}}")
            self.assertEqual(studio_env["STUDIO_PG_META_URL"], f"http://meta-{target}:8080")
            self.assertEqual(studio_env["PG_META_CRYPTO_KEY"], f"${{{prefix}META_CRYPTO_KEY:?}}")
            self.assertEqual(studio_env["AUTH_JWT_SECRET"], f"${{{prefix}AUTH_JWT_SECRET:?}}")
            self.assertEqual(meta_env["PG_META_DB_HOST"], "db")
            self.assertEqual(meta_env["PG_META_DB_PORT"], "5432")
            self.assertEqual(meta_env["PG_META_DB_NAME"], f"${{{prefix}DB_NAME:?}}")
            self.assertEqual(meta_env["PG_META_DB_USER"], f"${{{prefix}DB_USER:?}}")
            self.assertEqual(meta_env["PG_META_DB_PASSWORD"], f"${{{prefix}DB_PASSWORD:?}}")
            self.assertEqual(meta_env["CRYPTO_KEY"], f"${{{prefix}META_CRYPTO_KEY:?}}")
            self.assertNotIn("ports", meta)

        self.assertEqual(services["studio-main"]["ports"], ["127.0.0.1:18085:3000"])
        self.assertEqual(services["studio-develop"]["ports"], ["127.0.0.1:18086:3000"])
        self.assertEqual(services["studio-main"]["networks"], ["studio-main-admin", "infra"])
        self.assertEqual(services["studio-develop"]["networks"], ["studio-develop-admin", "infra"])
        self.assertTrue(config["networks"]["infra"]["external"])
        self.assertEqual(config["networks"]["infra"]["name"], "infra_default")

    def test_compose_config_render_uses_only_synthetic_values_and_never_starts_services(self):
        compose = None
        docker = shutil.which("docker")
        if docker is not None:
            plugin = subprocess.run(
                [docker, "compose", "version"],
                text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
            )
            if plugin.returncode == 0:
                compose = [docker, "compose"]
        if compose is None:
            standalone = shutil.which("docker-compose")
            if standalone is not None:
                version = subprocess.run(
                    [standalone, "version"],
                    text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=False,
                )
                if version.returncode == 0:
                    compose = [standalone]
        if compose is None:
            self.skipTest("Docker Compose plugin and standalone binary unavailable")
        with tempfile.TemporaryDirectory(prefix="studio-compose-") as temp:
            env_file = Path(temp) / "fixture.env"
            env_file.write_text(
                "".join(f"{key}={value}\n" for key, value in TEST_VALUES.items()),
                encoding="utf-8",
            )
            rendered = subprocess.run(
                [
                    *compose, "--project-directory", str(ROOT),
                    "--env-file", str(env_file), "-f", str(ROOT / "compose.yaml"),
                    "config", "--format", "json",
                ],
                text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                check=False, timeout=30,
            )
            self.assertEqual(rendered.returncode, 0, "Compose config render failed")
            config = json.loads(rendered.stdout)
            self.assertEqual(set(config["services"]), EXPECTED_SERVICES)

    def test_nginx_template_authenticates_each_host_and_maps_only_approved_upstreams(self):
        template_path = ROOT / "nginx" / "hosts.conf.template"
        self.assertTrue(template_path.is_file(), "missing Nginx host template")
        template = template_path.read_text(encoding="utf-8")
        server_blocks = re.split(r"(?=server\s*\{)", template)
        expected_hosts = {
            "db.home.arpa": None,
            "nsr.db.home.arpa": 18084,
            "mindforge.db.home.arpa": 18085,
            "develop.db.home.arpa": 18086,
        }
        for host, port in expected_hosts.items():
            with self.subTest(host=host):
                block = next(
                    (part for part in server_blocks if f"server_name {host};" in part),
                    None,
                )
                self.assertIsNotNone(block, f"missing server block for {host}")
                self.assertIn("auth_basic", block)
                self.assertIn("@AUTH_FILE@", block)
                if port is None:
                    self.assertIn("root @PORTAL_ROOT@;", block)
                    self.assertNotIn("proxy_pass", block)
                else:
                    self.assertEqual(
                        block.count("proxy_pass"),
                        1,
                        f"{host} must have one upstream",
                    )
                    self.assertIn(f"proxy_pass http://127.0.0.1:{port};", block)
                    if host == "nsr.db.home.arpa":
                        self.assertIn("proxy_set_header Host db.home.arpa;", block)
                        self.assertIn("proxy_set_header X-Forwarded-Host $host;", block)
                        self.assertIn("proxy_set_header Authorization $http_authorization;", block)
                    else:
                        self.assertIn("proxy_set_header Host $host;", block)
                        self.assertIn('proxy_set_header Authorization "";', block)
        self.assertIn("root @PORTAL_ROOT@;", template)
        self.assertIn("proxy_pass http://127.0.0.1:18084;", template)
        self.assertIn("proxy_pass http://127.0.0.1:18085;", template)
        self.assertIn("proxy_pass http://127.0.0.1:18086;", template)
        self.assertNotRegex(template, r"proxy_pass\s+http://meta")
        self.assertIn("proxy_set_header Host $host;", template)

    def test_nginx_renderer_requires_explicit_safe_paths_and_refuses_overwrite(self):
        renderer = SCRIPTS / "render-nginx.sh"
        self.assertTrue(renderer.is_file(), "missing Nginx renderer")
        no_args = subprocess.run(["bash", str(renderer)], capture_output=True, text=True)
        self.assertNotEqual(no_args.returncode, 0)
        with tempfile.TemporaryDirectory(prefix="studio-render-") as temp:
            temp_path = Path(temp)
            output = temp_path / "hosts.conf"
            portal = temp_path / "portal"
            auth_file = temp_path / "private" / "studio.htpasswd"
            result = subprocess.run(
                [
                    "bash", str(renderer),
                    "--portal-root", str(portal),
                    "--auth-file", str(auth_file),
                    "--output", str(output),
                ],
                capture_output=True,
                text=True,
            )
            self.assertEqual(result.returncode, 0, "valid paths did not render")
            rendered = output.read_text(encoding="utf-8")
            self.assertIn(f"root {portal};", rendered)
            self.assertIn(f"auth_basic_user_file {auth_file};", rendered)
            self.assertNotIn("@PORTAL_ROOT@", rendered)
            self.assertNotIn("@AUTH_FILE@", rendered)
            overwrite = subprocess.run(
                [
                    "bash", str(renderer),
                    "--portal-root", str(portal),
                    "--auth-file", str(auth_file),
                    "--output", str(output),
                ],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(overwrite.returncode, 0)

            unsafe = temp_path / "unsafe.conf"
            marker = temp_path / "injected"
            injection = subprocess.run(
                [
                    "bash", str(renderer),
                    "--portal-root", f"{temp_path}/portal; touch {marker}",
                    "--auth-file", str(auth_file),
                    "--output", str(unsafe),
                ],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(injection.returncode, 0)
            self.assertFalse(marker.exists())

    def test_portal_has_three_semantic_keyboard_native_links(self):
        portal_path = ROOT / "portal" / "index.html"
        self.assertTrue(portal_path.is_file(), "missing static portal")
        source = portal_path.read_text(encoding="utf-8")
        parser = PortalParser()
        parser.feed(source)
        self.assertEqual(parser.lang, "pt-BR")
        self.assertEqual(parser.h1_count, 1)
        self.assertEqual(parser.scripts, 0)
        self.assertEqual({href for href, _ in parser.links}, PORTAL_LINKS)
        labels = {href: label for href, label in parser.links}
        self.assertIn("NSR", labels["http://nsr.db.home.arpa"])
        self.assertRegex(labels["http://mindforge.db.home.arpa"], r"(?i)produção")
        self.assertRegex(labels["http://mindforge.db.home.arpa"], r"(?i)admin")
        self.assertRegex(labels["http://develop.db.home.arpa"], r"(?i)develop")
        self.assertIn(":focus-visible", source)
        self.assertRegex(source, r"(?is):focus-visible\s*\{[^}]*outline")
        self.assertNotRegex(source, r"(?i)target=[\"']_blank")


if __name__ == "__main__":
    unittest.main()
