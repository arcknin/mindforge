from __future__ import annotations

import json
import re
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
COMPOSE = ROOT / "compose.yaml"
DOCKERFILE = ROOT / "Dockerfile"
DOCKERIGNORE = ROOT / ".dockerignore"
CONTAINER_NGINX = ROOT / "nginx" / "default.conf"
HOST_NGINX = ROOT / "nginx" / "web-mind.home.arpa.conf"
EXPECTED_API_HOST = "mindforge-brain-engine-develop-brain-engine-develop-1"
EXPECTED_API = f"{EXPECTED_API_HOST}:8080"


def parse_yaml(path: Path) -> dict:
    ruby = shutil.which("ruby")
    if ruby is None:
        raise unittest.SkipTest("Ruby YAML parser is unavailable")
    result = subprocess.run(
        [
            ruby,
            "-e",
            'require "yaml"; require "json"; print JSON.generate(YAML.load_file(ARGV.fetch(0)))',
            str(path),
        ],
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=False,
    )
    if result.returncode != 0:
        raise AssertionError(f"YAML parse failed for {path.name}: {result.stderr}")
    return json.loads(result.stdout)


def compose_command() -> list[str] | None:
    docker = shutil.which("docker")
    if docker is not None:
        plugin = subprocess.run(
            [docker, "compose", "version"],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if plugin.returncode == 0:
            return [docker, "compose"]
    standalone = shutil.which("docker-compose")
    if standalone is not None:
        version = subprocess.run(
            [standalone, "version"],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        if version.returncode == 0:
            return [standalone]
    return None


class WebLocalPackageTests(unittest.TestCase):
    def test_package_files_are_scoped_and_static_build_is_an_explicit_input(self):
        required = [COMPOSE, DOCKERFILE, DOCKERIGNORE, CONTAINER_NGINX, HOST_NGINX]
        missing = [path.name for path in required if not path.is_file()]
        self.assertEqual(missing, [])
        self.assertIn("COPY build/web/ /usr/share/nginx/html/", DOCKERFILE.read_text())
        self.assertIn("build/web/index.html", (ROOT / "README.md").read_text())
        self.assertFalse(any(path.name.startswith(".env") for path in ROOT.rglob("*")))
        self.assertFalse(any("secret" in path.name.lower() for path in ROOT.rglob("*")))

    def test_dockerfile_is_digest_pinned_and_docker_context_is_allowlisted(self):
        dockerfile = DOCKERFILE.read_text(encoding="utf-8")
        self.assertRegex(
            dockerfile,
            r"(?m)^FROM nginx:1\.30\.5-alpine3\.24@sha256:[0-9a-f]{64}$",
        )
        self.assertIn("COPY nginx/default.conf /etc/nginx/conf.d/mindforge.conf", dockerfile)
        self.assertIn("COPY build/web/ /usr/share/nginx/html/", dockerfile)
        ignore = DOCKERIGNORE.read_text(encoding="utf-8").splitlines()
        self.assertEqual(ignore[0], "**")
        self.assertIn("!Dockerfile", ignore)
        self.assertIn("!nginx/default.conf", ignore)
        self.assertIn("!build/web/**", ignore)
        self.assertNotIn("!src/", ignore)
        self.assertNotIn("!.env", ignore)

    def test_compose_contains_only_loopback_web_service_on_existing_network(self):
        config = parse_yaml(COMPOSE)
        self.assertEqual(config["name"], "mindforge-web-develop")
        self.assertEqual(set(config["services"]), {"web"})
        service = config["services"]["web"]
        self.assertEqual(service["ports"], ["127.0.0.1:18088:80"])
        self.assertEqual(service["networks"], ["infra"])
        self.assertEqual(service["restart"], "unless-stopped")
        self.assertTrue(service["read_only"])
        self.assertEqual(set(service["cap_drop"]), {"ALL"})
        self.assertEqual(
            set(service["cap_add"]),
            {"NET_BIND_SERVICE", "SETUID", "SETGID", "CHOWN"},
        )
        self.assertNotIn("environment", service)
        self.assertNotIn("volumes", service)
        self.assertNotIn("depends_on", service)
        self.assertEqual(
            config["networks"]["infra"],
            {"external": True, "name": "infra_default"},
        )

    def test_compose_render_is_daemon_free_and_has_no_private_configuration(self):
        command = compose_command()
        if command is None:
            self.skipTest("Docker Compose CLI unavailable")
        rendered = subprocess.run(
            [*command, "--project-directory", str(ROOT), "-f", str(COMPOSE), "config", "--format", "json"],
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
            timeout=30,
        )
        self.assertEqual(rendered.returncode, 0, "Compose render failed")
        config = json.loads(rendered.stdout)
        self.assertEqual(set(config["services"]), {"web"})
        self.assertEqual(config["services"]["web"]["ports"][0]["published"], "18088")
        self.assertEqual(config["services"]["web"]["ports"][0]["host_ip"], "127.0.0.1")
        self.assertNotIn("environment", config["services"]["web"])
        self.assertNotIn("volumes", config["services"]["web"])

    def test_container_nginx_routes_only_the_five_auth_facade_paths_to_brain_engine(self):
        config = CONTAINER_NGINX.read_text(encoding="utf-8")
        auth_paths = (
            "/api/v1/auth/authorize",
            "/api/v1/auth/callback",
            "/api/v1/auth/token",
            "/api/v1/auth/user",
            "/api/v1/auth/logout",
        )
        for path in auth_paths:
            with self.subTest(path=path):
                self.assertIn(f"location = {path}", config)
        self.assertIn("location = /auth {\n        return 404;", config)
        self.assertIn("location ^~ /auth/ {\n        return 404;", config)
        self.assertNotIn("infra-auth-develop-1", config)
        self.assertNotIn("/auth/health", config)
        self.assertNotIn("/auth/settings", config)
        self.assertNotIn("proxy_pass http://auth", config)

        raw_path_guard = 'if ($request_uri ~ "^[^?]*%") { return 404; }'
        self.assertEqual(config.count(raw_path_guard), len(auth_paths))
        for path in auth_paths:
            start = config.index(f"location = {path} {{")
            end = config.find("\n    location ", start + 1)
            block = config[start:] if end == -1 else config[start:end]
            self.assertIn(raw_path_guard, block)
            self.assertRegex(
                config,
                re.compile(
                    rf"location = {re.escape(path)}.*?proxy_pass http://{re.escape(EXPECTED_API)}/{re.escape(path.removeprefix('/api/'))};",
                    re.S,
                ),
            )

    def test_container_nginx_allows_only_the_four_brain_operations(self):
        config = CONTAINER_NGINX.read_text(encoding="utf-8")
        self.assertRegex(config, r"location = /api/v1/me/brain\s*\{\s*limit_except GET PUT")
        self.assertRegex(config, r"location = /api/v1/me/brain/layers\s*\{\s*limit_except POST")
        self.assertRegex(
            config,
            r"location ~ \^/api/v1/me/brain/layers/\[\^/\]\+\$\s*\{\s*limit_except PATCH",
        )
        self.assertIn("location /api/ {\n        return 404;", config)
        proxy_passes = re.findall(r"proxy_pass\s+([^;]+);", config)
        self.assertTrue(proxy_passes)
        self.assertTrue(all("$" not in upstream for upstream in proxy_passes))
        self.assertTrue(all(EXPECTED_API in upstream for upstream in proxy_passes))
        self.assertNotIn("infra-auth-develop-1", config)
        self.assertIn(f"proxy_pass http://{EXPECTED_API}/v1/me/brain;", config)
        self.assertIn(f"proxy_pass http://{EXPECTED_API}/v1/me/brain/layers;", config)
        self.assertIn("rewrite ^/api(/.*)$ $1 break;", config)

    def test_container_nginx_disables_sensitive_logs_and_caching_and_sets_headers(self):
        config = CONTAINER_NGINX.read_text(encoding="utf-8")
        self.assertIn("access_log off;", config)
        self.assertIn("error_log /dev/stderr crit;", config)
        self.assertNotRegex(config, r"(?m)^\s*log_format\s")
        self.assertNotRegex(config, r"(?m)^\s*proxy_cache\s+(?!off\s*;)")
        self.assertIn("proxy_cache off;", config)
        self.assertIn("proxy_no_cache 1;", config)
        self.assertIn("proxy_cache_bypass 1;", config)
        self.assertNotIn("proxy_set_header Cookie", config)
        self.assertIn('add_header Referrer-Policy "no-referrer" always;', config)
        self.assertIn('add_header X-Content-Type-Options "nosniff" always;', config)
        self.assertIn('add_header Cache-Control "no-store" always;', config)
        self.assertNotIn("Content-Security-Policy", config)
        self.assertEqual(config.count("$request_uri"), 5)
        self.assertNotIn("$args", config)
        self.assertNotRegex(config, r"(?i)access_log\s+[^;]*\$(?:args|request_uri|request)")

    def test_spa_callback_serves_index_and_host_vhost_is_lan_restricted(self):
        container = CONTAINER_NGINX.read_text(encoding="utf-8")
        self.assertIn("location = /login/callback {\n        try_files /index.html =404;", container)
        self.assertIn("try_files $uri $uri/ /index.html;", container)

        host = HOST_NGINX.read_text(encoding="utf-8")
        self.assertIn("server_name web-mind.home.arpa;", host)
        self.assertIn("allow 127.0.0.1;", host)
        self.assertIn("allow 10.0.2.2;", host)
        self.assertIn("allow 192.168.0.0/24;", host)
        self.assertIn("deny all;", host)
        self.assertIn("access_log off;", host)
        self.assertIn("error_log /var/log/nginx/error.log crit;", host)
        self.assertEqual(host.count("proxy_pass"), 1)
        self.assertIn("proxy_pass http://127.0.0.1:18088;", host)
        self.assertNotIn("proxy_set_header Cookie", host)
        self.assertNotRegex(host, r"(?m)^\s*listen\s+\[::\]")
        self.assertNotIn("auth_basic", host)
        self.assertIn("proxy_set_header X-Forwarded-Proto $scheme;", host)


if __name__ == "__main__":
    unittest.main()
