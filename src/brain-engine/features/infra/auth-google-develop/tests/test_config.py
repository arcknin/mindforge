from __future__ import annotations

import json
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "validate-config.py"
INFRA = ROOT.parent
AUTH_KEYS = {
    "GOOGLE_AUTH_API_EXTERNAL_URL": "http://127.0.0.1:11001/v1/auth",
    "GOOGLE_AUTH_SITE_URL": "http://web-mind.home.arpa",
    "GOOGLE_AUTH_ADDITIONAL_REDIRECT_URLS": "mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback",
    "GOOGLE_AUTH_URI_ALLOW_LIST": "mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback",
    "GOOGLE_AUTH_EXTERNAL_EMAIL_ENABLED": "false",
    "GOOGLE_AUTH_EXTERNAL_PHONE_ENABLED": "false",
    "GOOGLE_AUTH_EXTERNAL_ANONYMOUS_USERS_ENABLED": "false",
    "GOOGLE_AUTH_EXTERNAL_GOOGLE_ENABLED": "true",
    "GOOGLE_AUTH_GOOGLE_CLIENT_ID": "123456789.apps.googleusercontent.com",
    "GOOGLE_AUTH_GOOGLE_SECRET": "fixtureOAuthSecret_0123456789",
    "GOOGLE_AUTH_GOOGLE_REDIRECT_URI": "http://127.0.0.1:11001/v1/auth/callback",
}
FINAL_CALLBACKS = (
    "mindforge-dev://auth/callback",
    "http://web-mind.home.arpa/login/callback",
)
RUNTIME_KEYS = {
    "MINDFORGE_ENVIRONMENT": "develop",
    "MINDFORGE_DATABASE_URL": (
        "postgres://mindforge_develop:DevelopDbFixture_0123456789@db:5432/"
        "mindforge_develop?sslmode=disable"
    ),
    "MINDFORGE_AUTH_USER_URL": "http://auth-develop:9999/user",
    "MINDFORGE_AUTH_CALLBACK_URL": "http://127.0.0.1:11001/v1/auth/callback",
    "MINDFORGE_AUTH_REDIRECT_URLS": "mindforge-dev://auth/callback,http://web-mind.home.arpa/login/callback",
    "MINDFORGE_LISTEN_ADDR": ":8080",
}
SSH_KEYS = {
    "VM_SSH_HOST": "vm.mindforge.test",
    "VM_SSH_PORT": "2222",
    "VM_SSH_USER": "mindforge-dev",
}
OVERLAY_ENVIRONMENT_KEYS = {
    "GOTRUE_JWT_AUD",
    "GOTRUE_JWT_DEFAULT_GROUP_NAME",
    "GOTRUE_SITE_URL",
    "GOTRUE_ADDITIONAL_REDIRECT_URLS",
    "GOTRUE_URI_ALLOW_LIST",
    "API_EXTERNAL_URL",
    "GOTRUE_EXTERNAL_EMAIL_ENABLED",
    "GOTRUE_EXTERNAL_PHONE_ENABLED",
    "GOTRUE_EXTERNAL_ANONYMOUS_USERS_ENABLED",
    "GOTRUE_EXTERNAL_GOOGLE_ENABLED",
    "GOTRUE_EXTERNAL_GOOGLE_CLIENT_ID",
    "GOTRUE_EXTERNAL_GOOGLE_SECRET",
    "GOTRUE_EXTERNAL_GOOGLE_REDIRECT_URI",
}


def write_private_file(path: Path, values: dict[str, str]) -> Path:
    path.write_text("".join(f"{key}={value}\n" for key, value in values.items()), encoding="utf-8")
    path.chmod(0o600)
    return path


def validate(kind: str, values: dict[str, str] | None = None, path: Path | None = None):
    temp = None
    if path is None:
        temp = tempfile.TemporaryDirectory(prefix="mindforge-private-")
        path = Path(temp.name) / f"{kind}.env.private"
        write_private_file(path, values or {})
    try:
        return subprocess.run(
            ["python3", str(SCRIPT), kind, "--file", str(path)],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=10,
        )
    finally:
        if temp is not None:
            temp.cleanup()


def yaml_to_json(path: Path) -> dict:
    if shutil.which("ruby") is None:
        raise unittest.SkipTest("Ruby standard YAML parser is unavailable")
    parsed = subprocess.run(
        [
            "ruby",
            "-e",
            'require "yaml"; require "json"; print JSON.generate(YAML.load_file(ARGV.fetch(0)))',
            str(path),
        ],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
        timeout=10,
    )
    if parsed.returncode != 0:
        raise AssertionError("YAML parse failed")
    return json.loads(parsed.stdout)


class PrivateConfigTests(unittest.TestCase):
    def test_google_private_config_accepts_synthetic_values_without_printing_them(self):
        result = validate("google", AUTH_KEYS)
        self.assertEqual(result.returncode, 0, result.stderr)
        combined = result.stdout + result.stderr
        self.assertIn("sensitive values omitted", combined)
        self.assertNotIn(AUTH_KEYS["GOOGLE_AUTH_GOOGLE_SECRET"], combined)
        self.assertNotIn(AUTH_KEYS["GOOGLE_AUTH_GOOGLE_CLIENT_ID"], combined)

    def test_google_example_is_complete_but_rejected_until_credentials_exist(self):
        example = ROOT / "google-auth.env.example"
        self.assertTrue(example.is_file())
        keys = {
            line.split("=", 1)[0]
            for line in example.read_text(encoding="utf-8").splitlines()
            if line and not line.startswith("#") and "=" in line
        }
        self.assertEqual(keys, set(AUTH_KEYS))
        with tempfile.TemporaryDirectory(prefix="mindforge-google-example-") as temp:
            private_copy = Path(temp) / "google.env.private"
            private_copy.write_text(example.read_text(encoding="utf-8"), encoding="utf-8")
            private_copy.chmod(0o600)
            result = validate("google", path=private_copy)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("replace-with", result.stdout + result.stderr)

    def test_google_config_rejects_unknown_missing_duplicate_and_unsafe_values(self):
        cases = []
        cases.append({**AUTH_KEYS, "UNRELATED_SECRET": "fixture-value"})
        cases.append({key: value for key, value in AUTH_KEYS.items() if key != "GOOGLE_AUTH_GOOGLE_SECRET"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_SITE_URL": "mindforge-dev://other/callback"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_ADDITIONAL_REDIRECT_URLS": FINAL_CALLBACKS[0]})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_URI_ALLOW_LIST": FINAL_CALLBACKS[1]})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_GOOGLE_REDIRECT_URI": "http://127.0.0.1:11000/callback"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_EXTERNAL_EMAIL_ENABLED": "true"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_EXTERNAL_PHONE_ENABLED": "true"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_EXTERNAL_ANONYMOUS_USERS_ENABLED": "true"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_EXTERNAL_GOOGLE_ENABLED": "false"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_GOOGLE_CLIENT_ID": "not-a-google-client"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_GOOGLE_SECRET": "replace-with-real-secret"})
        cases.append({**AUTH_KEYS, "GOOGLE_AUTH_GOOGLE_SECRET": "$(touch /tmp/mindforge-config-injected)"})
        for values in cases:
            with self.subTest(keys=set(values)):
                result = validate("google", values)
                self.assertNotEqual(result.returncode, 0)
                combined = result.stdout + result.stderr
                self.assertNotIn(AUTH_KEYS["GOOGLE_AUTH_GOOGLE_SECRET"], combined)
                self.assertNotIn("replace-with-real-secret", combined)
        self.assertFalse(Path("/tmp/mindforge-config-injected").exists())

    def test_google_config_uses_api_facade_callback_and_exact_final_callback_pair(self):
        result = validate("google", AUTH_KEYS)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(
            AUTH_KEYS["GOOGLE_AUTH_ADDITIONAL_REDIRECT_URLS"].split(","),
            list(FINAL_CALLBACKS),
        )
        self.assertEqual(
            AUTH_KEYS["GOOGLE_AUTH_URI_ALLOW_LIST"].split(","),
            list(FINAL_CALLBACKS),
        )

    def test_google_parser_rejects_duplicate_keys_without_shell_execution(self):
        with tempfile.TemporaryDirectory(prefix="mindforge-noeval-") as temp:
            base = Path(temp) / "google.env.private"
            lines = [f"{key}={value}" for key, value in AUTH_KEYS.items()]
            lines.append("GOOGLE_AUTH_EXTERNAL_GOOGLE_ENABLED=true")
            base.write_text("\n".join(lines) + "\n", encoding="utf-8")
            base.chmod(0o600)
            result = validate("google", path=base)
            self.assertNotEqual(result.returncode, 0)

            marker = Path(temp) / "executed"
            lines[0] = f"GOOGLE_AUTH_API_EXTERNAL_URL=$(touch {marker})"
            base.write_text("\n".join(lines) + "\n", encoding="utf-8")
            base.chmod(0o600)
            result = validate("google", path=base)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(marker.exists())

    def test_private_config_rejects_symlink_and_insecure_permissions(self):
        with tempfile.TemporaryDirectory(prefix="mindforge-file-check-") as temp:
            directory = Path(temp)
            target = write_private_file(directory / "real.env", AUTH_KEYS)
            symlink = directory / "linked.env"
            symlink.symlink_to(target)
            linked = validate("google", path=symlink)
            self.assertNotEqual(linked.returncode, 0)

            target.chmod(0o644)
            insecure = validate("google", path=target)
            self.assertNotEqual(insecure.returncode, 0)

    def test_runtime_config_accepts_develop_only_connection_without_printing_url(self):
        result = validate("runtime", RUNTIME_KEYS)
        self.assertEqual(result.returncode, 0, result.stderr)
        combined = result.stdout + result.stderr
        self.assertIn("sensitive values omitted", combined)
        self.assertNotIn(RUNTIME_KEYS["MINDFORGE_DATABASE_URL"], combined)
        self.assertNotIn("DevelopDbFixture_0123456789", combined)

    def test_runtime_config_rejects_wrong_database_environment_or_service_urls(self):
        cases = [
            {**RUNTIME_KEYS, "MINDFORGE_ENVIRONMENT": "main"},
            {
                **RUNTIME_KEYS,
                "MINDFORGE_DATABASE_URL": RUNTIME_KEYS["MINDFORGE_DATABASE_URL"].replace(
                    "mindforge_develop", "mindforge", 1
                ),
            },
            {
                **RUNTIME_KEYS,
                "MINDFORGE_DATABASE_URL": RUNTIME_KEYS["MINDFORGE_DATABASE_URL"].replace(
                    "@db:5432", "@localhost:5432"
                ),
            },
            {
                **RUNTIME_KEYS,
                "MINDFORGE_DATABASE_URL": RUNTIME_KEYS["MINDFORGE_DATABASE_URL"].replace(
                    "mindforge_develop:", "mindforge:"
                ),
            },
            {**RUNTIME_KEYS, "MINDFORGE_AUTH_USER_URL": "http://auth:9999/user"},
            {**RUNTIME_KEYS, "MINDFORGE_AUTH_CALLBACK_URL": "http://127.0.0.1:11000/callback"},
            {**RUNTIME_KEYS, "MINDFORGE_AUTH_REDIRECT_URLS": "mindforge-dev://auth/callback"},
            {**RUNTIME_KEYS, "MINDFORGE_LISTEN_ADDR": "0.0.0.0:8080"},
            {
                **RUNTIME_KEYS,
                "MINDFORGE_DATABASE_URL": RUNTIME_KEYS["MINDFORGE_DATABASE_URL"].replace(
                    "DevelopDbFixture_0123456789", "replace-with-password"
                ),
            },
            {**RUNTIME_KEYS, "UNRELATED_SECRET": "fixture-value"},
        ]
        for values in cases:
            with self.subTest(values={key: value for key, value in values.items() if "URL" not in key}):
                result = validate("runtime", values)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn("DevelopDbFixture_0123456789", result.stdout + result.stderr)

    def test_runtime_example_is_rejected_until_develop_credentials_are_supplied(self):
        example = ROOT / "brain-engine-runtime.env.example"
        keys = {
            line.split("=", 1)[0]
            for line in example.read_text(encoding="utf-8").splitlines()
            if line and not line.startswith("#") and "=" in line
        }
        self.assertEqual(keys, set(RUNTIME_KEYS))
        with tempfile.TemporaryDirectory(prefix="mindforge-runtime-example-") as temp:
            private_copy = Path(temp) / "runtime.env.private"
            private_copy.write_text(example.read_text(encoding="utf-8"), encoding="utf-8")
            private_copy.chmod(0o600)
            result = validate("runtime", path=private_copy)
        self.assertNotEqual(result.returncode, 0)
        self.assertNotIn("replace-with", result.stdout + result.stderr)

    def test_ssh_config_requires_owner_values_and_never_accepts_command_shaped_host(self):
        good = validate("ssh", SSH_KEYS)
        self.assertEqual(good.returncode, 0, good.stderr)
        bad_values = [
            {**SSH_KEYS, "VM_SSH_HOST": "-oProxyCommand=sh"},
            {**SSH_KEYS, "VM_SSH_PORT": "0"},
            {**SSH_KEYS, "VM_SSH_PORT": "2222;touch/tmp/x"},
            {**SSH_KEYS, "VM_SSH_USER": "replace-with-user"},
            {**SSH_KEYS, "UNRELATED_SECRET": "fixture-value"},
        ]
        for values in bad_values:
            with self.subTest(keys=set(values)):
                result = validate("ssh", values)
                self.assertNotEqual(result.returncode, 0)

    def test_google_overlay_is_one_service_and_only_overrides_develop_auth_configuration(self):
        overlay = yaml_to_json(ROOT / "compose.auth-google-develop.yaml")
        self.assertEqual(set(overlay["services"]), {"auth-develop"})
        self.assertEqual(set(overlay["services"]["auth-develop"]["environment"]), OVERLAY_ENVIRONMENT_KEYS)
        self.assertNotIn("ports", overlay["services"]["auth-develop"])
        self.assertNotIn("volumes", overlay["services"]["auth-develop"])
        self.assertNotIn("depends_on", overlay["services"]["auth-develop"])
        self.assertNotIn("networks", overlay)
        self.assertNotIn("volumes", overlay)

    def test_brain_engine_compose_is_isolated_api_only_container(self):
        compose = yaml_to_json(ROOT / "compose.brain-engine-develop.yaml")
        self.assertEqual(set(compose["services"]), {"brain-engine-develop"})
        service = compose["services"]["brain-engine-develop"]
        self.assertEqual(service["ports"], ["127.0.0.1:18087:8080"])
        self.assertEqual(service["networks"], ["infra"])
        self.assertEqual(
            set(service["environment"]),
            {
                "MINDFORGE_ENVIRONMENT",
                "MINDFORGE_DATABASE_URL",
                "MINDFORGE_AUTH_USER_URL",
                "MINDFORGE_AUTH_CALLBACK_URL",
                "MINDFORGE_AUTH_REDIRECT_URLS",
                "MINDFORGE_LISTEN_ADDR",
            },
        )
        self.assertNotIn("volumes", service)
        self.assertNotIn("depends_on", service)
        self.assertNotIn("healthcheck", service)
        self.assertEqual(compose["networks"]["infra"], {"external": True, "name": "infra_default"})
        dockerfile = (ROOT / "Dockerfile").read_text(encoding="utf-8")
        self.assertIn("FROM scratch", dockerfile)
        self.assertIn("USER 65532:65532", dockerfile)
        self.assertIn('ENTRYPOINT ["/brain-engine"]', dockerfile)
        self.assertNotIn("COPY .", dockerfile)
        dockerignore = (ROOT / ".dockerignore").read_text(encoding="utf-8")
        self.assertIn(".env.*", dockerignore)
        self.assertIn("*.private", dockerignore)
        self.assertIn("scripts/", dockerignore)

    def test_brain_engine_compose_renders_with_synthetic_values_without_starting_services(self):
        compose_cli = None
        docker = shutil.which("docker")
        if docker:
            version = subprocess.run([docker, "compose", "version"], capture_output=True, text=True)
            if version.returncode == 0:
                compose_cli = [docker, "compose"]
        if compose_cli is None:
            standalone = shutil.which("docker-compose")
            if standalone:
                version = subprocess.run([standalone, "version"], capture_output=True, text=True)
                if version.returncode == 0:
                    compose_cli = [standalone]
        if compose_cli is None:
            self.skipTest("Docker Compose unavailable; no daemon is required for this render")

        with tempfile.TemporaryDirectory(prefix="mindforge-api-compose-") as temp:
            env_file = write_private_file(Path(temp) / "runtime.env.private", RUNTIME_KEYS)
            rendered = subprocess.run(
                [
                    *compose_cli,
                    "--project-directory",
                    str(ROOT),
                    "--env-file",
                    str(env_file),
                    "-f",
                    str(ROOT / "compose.brain-engine-develop.yaml"),
                    "config",
                    "--format",
                    "json",
                ],
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
                timeout=30,
            )
            self.assertEqual(rendered.returncode, 0, "Compose config render failed")
            config = json.loads(rendered.stdout)
            self.assertEqual(set(config["services"]), {"brain-engine-develop"})
            service = config["services"]["brain-engine-develop"]
            self.assertEqual(service["ports"], [{"mode": "ingress", "target": 8080, "published": "18087", "protocol": "tcp", "host_ip": "127.0.0.1"}])
            self.assertEqual(service["environment"]["MINDFORGE_ENVIRONMENT"], "develop")
            self.assertEqual(service["environment"]["MINDFORGE_AUTH_USER_URL"], "http://auth-develop:9999/user")
            self.assertEqual(service["environment"]["MINDFORGE_AUTH_CALLBACK_URL"], RUNTIME_KEYS["MINDFORGE_AUTH_CALLBACK_URL"])
            self.assertEqual(service["environment"]["MINDFORGE_AUTH_REDIRECT_URLS"], RUNTIME_KEYS["MINDFORGE_AUTH_REDIRECT_URLS"])
            self.assertEqual(config["networks"]["infra"]["name"], "infra_default")
            self.assertNotIn("DevelopDbFixture_0123456789", rendered.stderr)

    def test_auth_overlay_render_preserves_inherited_db_jwt_and_main_auth(self):
        compose_cli = None
        docker = shutil.which("docker")
        if docker:
            version = subprocess.run([docker, "compose", "version"], capture_output=True, text=True)
            if version.returncode == 0:
                compose_cli = [docker, "compose"]
        if compose_cli is None:
            standalone = shutil.which("docker-compose")
            if standalone:
                version = subprocess.run([standalone, "version"], capture_output=True, text=True)
                if version.returncode == 0:
                    compose_cli = [standalone]
        if compose_cli is None:
            self.skipTest("Docker Compose unavailable; no daemon is required for this render")

        with tempfile.TemporaryDirectory(prefix="mindforge-compose-") as temp:
            base_env = Path(temp) / "infra.env"
            base_env.write_text((INFRA / ".env.example").read_text(encoding="utf-8"), encoding="utf-8")
            auth_env = write_private_file(Path(temp) / "google.env.private", AUTH_KEYS)
            rendered = subprocess.run(
                [
                    *compose_cli,
                    "--project-directory",
                    str(INFRA),
                    "--env-file",
                    str(base_env),
                    "--env-file",
                    str(auth_env),
                    "-f",
                    str(INFRA / "compose.yaml"),
                    "-f",
                    str(ROOT / "compose.auth-google-develop.yaml"),
                    "config",
                    "--format",
                    "json",
                ],
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                check=False,
                timeout=30,
            )
            self.assertEqual(rendered.returncode, 0, "Compose config render failed")
            config = json.loads(rendered.stdout)
            services = config["services"]
            self.assertEqual(set(services), {"db", "auth-develop-migrate", "auth-develop", "auth-migrate", "auth"})
            develop = services["auth-develop"]["environment"]
            main = services["auth"]["environment"]
            base_values = {
                line.split("=", 1)[0]: line.split("=", 1)[1]
                for line in base_env.read_text(encoding="utf-8").splitlines()
                if line and not line.startswith("#") and "=" in line
            }
            self.assertEqual(develop["GOTRUE_DB_DATABASE_URL"], base_values["DEVELOP_AUTH_DB_URL"])
            self.assertEqual(develop["GOTRUE_JWT_SECRET"], base_values["DEVELOP_AUTH_JWT_SECRET"])
            self.assertEqual(develop["GOTRUE_JWT_AUD"], "authenticated")
            self.assertEqual(develop["GOTRUE_JWT_DEFAULT_GROUP_NAME"], "authenticated")
            overlay = yaml_to_json(ROOT / "compose.auth-google-develop.yaml")
            self.assertNotIn("GOTRUE_JWT_ISSUER", overlay["services"]["auth-develop"]["environment"])
            self.assertEqual(develop["API_EXTERNAL_URL"], AUTH_KEYS["GOOGLE_AUTH_API_EXTERNAL_URL"])
            self.assertEqual(develop["GOTRUE_EXTERNAL_GOOGLE_REDIRECT_URI"], AUTH_KEYS["GOOGLE_AUTH_GOOGLE_REDIRECT_URI"])
            self.assertEqual(develop["GOTRUE_EXTERNAL_GOOGLE_ENABLED"], "true")
            self.assertEqual(develop["GOTRUE_EXTERNAL_EMAIL_ENABLED"], "false")
            self.assertEqual(develop["GOTRUE_EXTERNAL_PHONE_ENABLED"], "false")
            self.assertEqual(develop["GOTRUE_EXTERNAL_ANONYMOUS_USERS_ENABLED"], "false")
            self.assertEqual(main["GOTRUE_DB_DATABASE_URL"], base_values["MAIN_AUTH_DB_URL"])
            self.assertNotIn("GOTRUE_EXTERNAL_GOOGLE_CLIENT_ID", main)
            self.assertNotIn(AUTH_KEYS["GOOGLE_AUTH_GOOGLE_SECRET"], rendered.stderr)


if __name__ == "__main__":
    unittest.main()
