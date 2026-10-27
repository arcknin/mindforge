from __future__ import annotations

import os
import re
import stat
from pathlib import Path
from urllib.parse import unquote, urlsplit


class ConfigError(ValueError):
    """A configuration error whose message never contains a supplied value."""


GOOGLE_KEYS = {
    "GOOGLE_AUTH_API_EXTERNAL_URL",
    "GOOGLE_AUTH_SITE_URL",
    "GOOGLE_AUTH_ADDITIONAL_REDIRECT_URLS",
    "GOOGLE_AUTH_URI_ALLOW_LIST",
    "GOOGLE_AUTH_EXTERNAL_EMAIL_ENABLED",
    "GOOGLE_AUTH_EXTERNAL_PHONE_ENABLED",
    "GOOGLE_AUTH_EXTERNAL_ANONYMOUS_USERS_ENABLED",
    "GOOGLE_AUTH_EXTERNAL_GOOGLE_ENABLED",
    "GOOGLE_AUTH_GOOGLE_CLIENT_ID",
    "GOOGLE_AUTH_GOOGLE_SECRET",
    "GOOGLE_AUTH_GOOGLE_REDIRECT_URI",
}
RUNTIME_KEYS = {
    "MINDFORGE_ENVIRONMENT",
    "MINDFORGE_DATABASE_URL",
    "MINDFORGE_AUTH_USER_URL",
    "MINDFORGE_AUTH_CALLBACK_URL",
    "MINDFORGE_AUTH_REDIRECT_URLS",
    "MINDFORGE_LISTEN_ADDR",
}
SSH_KEYS = {"VM_SSH_HOST", "VM_SSH_PORT", "VM_SSH_USER"}
PLACEHOLDER = re.compile(
    r"(?:replace[-_ ]?with|change[-_ ]?me|placeholder|dummy|sample|example|your[-_ ]|<[^>]+>|\b(?:todo|tbd)\b)",
    re.IGNORECASE,
)

GOOGLE_API_EXTERNAL_URL = "http://127.0.0.1:11001/v1/auth"
GOOGLE_CALLBACK_URL = "http://127.0.0.1:11001/v1/auth/callback"
GOOGLE_FINAL_CALLBACKS = (
    "mindforge-dev://auth/callback",
    "http://web-mind.home.arpa/login/callback",
)
GOOGLE_FINAL_CALLBACKS_VALUE = ",".join(GOOGLE_FINAL_CALLBACKS)


def _fail(message: str) -> None:
    raise ConfigError(message)


def read_private_env(path: str | Path, expected_keys: set[str]) -> dict[str, str]:
    file_path = Path(path)
    file_descriptor = None
    try:
        flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0)
        file_descriptor = os.open(file_path, flags)
        metadata = os.fstat(file_descriptor)
    except OSError:
        _fail("private environment file is unavailable")
    if not stat.S_ISREG(metadata.st_mode):
        os.close(file_descriptor)
        _fail("private environment file must be a regular non-symlink file")
    if metadata.st_mode & 0o077:
        os.close(file_descriptor)
        _fail("private environment file permissions must be 0600 or stricter")
    if metadata.st_size > 16 * 1024:
        os.close(file_descriptor)
        _fail("private environment file exceeds the size limit")

    try:
        stream = os.fdopen(file_descriptor, "r", encoding="utf-8")
        file_descriptor = None
        with stream:
            text = stream.read(16 * 1024 + 1)
    except (OSError, UnicodeError):
        _fail("private environment file is unreadable or is not UTF-8")
    finally:
        if file_descriptor is not None:
            os.close(file_descriptor)
    if len(text) > 16 * 1024:
        _fail("private environment file exceeds the size limit")
    values: dict[str, str] = {}
    for line_number, raw_line in enumerate(text.splitlines(), start=1):
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        if "=" not in line:
            _fail(f"invalid private environment line {line_number}")
        key, value = line.split("=", 1)
        if not re.fullmatch(r"[A-Z][A-Z0-9_]*", key):
            _fail(f"invalid private environment key at line {line_number}")
        if key not in expected_keys:
            _fail(f"unknown private environment key {key}")
        if key in values:
            _fail(f"duplicate private environment key {key}")
        if not value or value != value.strip():
            _fail(f"empty or whitespace-padded value for {key}")
        if any(ord(char) < 0x20 or ord(char) == 0x7F for char in value):
            _fail(f"control character in {key}")
        if any(char in value for char in ("$", "`", "\\", "'", '"')):
            _fail(f"shell or interpolation syntax is not allowed in {key}")
        values[key] = value

    missing = expected_keys - values.keys()
    if missing:
        _fail("missing private environment keys: " + ", ".join(sorted(missing)))
    return values


def reject_placeholder(key: str, value: str) -> None:
    if PLACEHOLDER.search(value):
        _fail(f"placeholder value is not allowed for {key}")


def validate_google(path: str | Path) -> dict[str, str]:
    values = read_private_env(path, GOOGLE_KEYS)
    fixed_values = {
        "GOOGLE_AUTH_API_EXTERNAL_URL": GOOGLE_API_EXTERNAL_URL,
        "GOOGLE_AUTH_SITE_URL": "http://web-mind.home.arpa",
        "GOOGLE_AUTH_ADDITIONAL_REDIRECT_URLS": GOOGLE_FINAL_CALLBACKS_VALUE,
        "GOOGLE_AUTH_URI_ALLOW_LIST": GOOGLE_FINAL_CALLBACKS_VALUE,
        "GOOGLE_AUTH_EXTERNAL_EMAIL_ENABLED": "false",
        "GOOGLE_AUTH_EXTERNAL_PHONE_ENABLED": "false",
        "GOOGLE_AUTH_EXTERNAL_ANONYMOUS_USERS_ENABLED": "false",
        "GOOGLE_AUTH_EXTERNAL_GOOGLE_ENABLED": "true",
        "GOOGLE_AUTH_GOOGLE_REDIRECT_URI": GOOGLE_CALLBACK_URL,
    }
    for key, expected in fixed_values.items():
        if values[key] != expected:
            _fail(f"unsupported value for {key}")
    for key in ("GOOGLE_AUTH_GOOGLE_CLIENT_ID", "GOOGLE_AUTH_GOOGLE_SECRET"):
        reject_placeholder(key, values[key])
    if not re.fullmatch(r"[A-Za-z0-9_-]+\.apps\.googleusercontent\.com", values["GOOGLE_AUTH_GOOGLE_CLIENT_ID"]):
        _fail("invalid Google OAuth client ID format")
    if not re.fullmatch(r"[A-Za-z0-9._~+-]{16,}", values["GOOGLE_AUTH_GOOGLE_SECRET"]):
        _fail("invalid Google OAuth client secret format")
    return values


def validate_runtime(path: str | Path) -> dict[str, str]:
    values = read_private_env(path, RUNTIME_KEYS)
    fixed_values = {
        "MINDFORGE_ENVIRONMENT": "develop",
        "MINDFORGE_AUTH_USER_URL": "http://auth-develop:9999/user",
        "MINDFORGE_AUTH_CALLBACK_URL": GOOGLE_CALLBACK_URL,
        "MINDFORGE_AUTH_REDIRECT_URLS": GOOGLE_FINAL_CALLBACKS_VALUE,
        "MINDFORGE_LISTEN_ADDR": ":8080",
    }
    for key, expected in fixed_values.items():
        if values[key] != expected:
            _fail(f"unsupported value for {key}")

    raw_url = values["MINDFORGE_DATABASE_URL"]
    reject_placeholder("MINDFORGE_DATABASE_URL", raw_url)
    try:
        parsed = urlsplit(raw_url)
        username = parsed.username or ""
        password = parsed.password or ""
        port = parsed.port
    except ValueError:
        _fail("invalid MINDFORGE_DATABASE_URL format")
    if (
        parsed.scheme != "postgres"
        or parsed.hostname != "db"
        or port != 5432
        or parsed.path != "/mindforge_develop"
        or parsed.query != "sslmode=disable"
        or parsed.fragment
        or parsed.netloc.count("@") != 1
    ):
        _fail("MINDFORGE_DATABASE_URL must target only the develop database service")
    if not re.fullmatch(r"[A-Za-z][A-Za-z0-9_.-]{0,62}", username):
        _fail("invalid database username in MINDFORGE_DATABASE_URL")
    if username != "mindforge_develop":
        _fail("MINDFORGE_DATABASE_URL must use the provisioned develop database role")
    if not re.fullmatch(r"(?:[A-Za-z0-9._~-]|%[A-Fa-f0-9]{2})+", password):
        _fail("invalid database password encoding in MINDFORGE_DATABASE_URL")
    decoded_password = unquote(password)
    reject_placeholder("MINDFORGE_DATABASE_URL", decoded_password)
    if len(decoded_password) < 16:
        _fail("database password in MINDFORGE_DATABASE_URL is too short")
    return values


def validate_ssh(path: str | Path) -> dict[str, str]:
    values = read_private_env(path, SSH_KEYS)
    for key, value in values.items():
        reject_placeholder(key, value)
    host = values["VM_SSH_HOST"]
    labels = host.split(".")
    if (
        not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9.-]{0,252}", host)
        or any(not label or label.startswith("-") or label.endswith("-") for label in labels)
        or ".." in host
    ):
        _fail("invalid VM_SSH_HOST")
    if not re.fullmatch(r"[A-Za-z_][A-Za-z0-9_.-]{0,31}", values["VM_SSH_USER"]):
        _fail("invalid VM_SSH_USER")
    if not re.fullmatch(r"[0-9]{1,5}", values["VM_SSH_PORT"]):
        _fail("invalid VM_SSH_PORT")
    port = int(values["VM_SSH_PORT"])
    if not 1 <= port <= 65535:
        _fail("invalid VM_SSH_PORT")
    return values


def validate(kind: str, path: str | Path) -> dict[str, str]:
    validators = {
        "google": validate_google,
        "runtime": validate_runtime,
        "ssh": validate_ssh,
    }
    try:
        validator = validators[kind]
    except KeyError:
        _fail("unknown configuration kind")
    return validator(path)
