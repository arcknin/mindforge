# Brain Engine develop package

This package contains one Linux binary and one API service definition. It has no Google credential, database password, database, Auth service, or persistent volume.

The service expects the external Compose network `infra_default` and an existing develop Auth service named `auth-develop`. It publishes only `127.0.0.1:18087` and uses `mindforge_develop` through the existing `db` service. Its public runtime keys are `MINDFORGE_ENVIRONMENT`, `MINDFORGE_DATABASE_URL`, `MINDFORGE_AUTH_USER_URL`, `MINDFORGE_AUTH_CALLBACK_URL`, `MINDFORGE_AUTH_REDIRECT_URLS`, and `MINDFORGE_LISTEN_ADDR`. The API facade owns the public Auth paths; this package does not expose GoTrue directly.

After the owner supplies the develop DB connection details and separately authorizes the VM delivery, copy `.env.runtime.example` to `.env.runtime`, restrict it to mode `0600`, and fill the approved DB URL using role and database `mindforge_develop`. Run `python3 scripts/validate-config.py runtime --file .env.runtime`; it prints no URL or password. Do not pass secret values as command arguments or record rendered Compose output.

The future service operation must use this package's Compose file and `.env.runtime` only. It must not invoke the infrastructure `up.sh`, create or attach a volume, or start another DB/Auth service. This package is not deployment or runtime evidence; the authorized B04 operation must confirm the container and real API behavior separately.

Rollback removes only the dedicated API package/container after the owner coordinates the Web/Auth release and callback/tunnel configuration. Preserve main, NSR, Studio, the existing `db` and Auth services, PostgreSQL data, credentials and volumes. Never use `down -v`, `--remove-orphans`, a migration downgrade or a broad infrastructure cleanup.
