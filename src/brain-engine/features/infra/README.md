# Mindforge Infra — reduced PostgreSQL, Auth and pgvector

This package implements only the infrastructure authorized by infra contract revision 2: PostgreSQL, Supabase Auth/GoTrue, `pgvector`, their required migration support, and safe disposable operations. It creates no Mindforge domain table, product API, client, PostgREST, Studio, Storage, Realtime, queue, DNS, TLS, firewall, Coolify, or production deployment.

The package is intentionally fail-closed. An apparently healthy container is not a completed functional acceptance: database readiness, GoTrue bootstrap, Auth health, migration state, extensions, persistence, backup and isolated restore are separate checks.

The operational package lives at `src/brain-engine/features/infra/` in the monorepo. The commands below assume that the current directory is this package directory.

## Prerequisites

- Docker Engine with Compose v2, available either as `docker compose` or a compatible `docker-compose` command.
- Bash 3.2 or later, `shasum` or `sha256sum`, and `curl` for the documented host-side Auth HTTP check.
- `pg_restore` on the operator host for the read-only backup preflight before any restore target is created.
- An existing, explicit external backup directory for backup output. It must be outside this checkout.

The database service is pinned to `pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b` for `linux/amd64`. The installed `vector` extension is compatible when its strict three-part version is greater than or equal to `0.8.0` and less than `1.0.0`; the current VM image reports `0.8.6`. Inspection accepts that compatibility range rather than requiring one patch release. The infrastructure migration remains `CREATE EXTENSION IF NOT EXISTS "vector";` and never alters or downgrades an installed extension.

## Configure and validate

From this directory:

```bash
cp .env.example .env
# Edit .env locally. Do not commit it.
./scripts/validate-config.sh .env
```

Every `replace-with-*` value is rejected. `POSTGRES_PASSWORD`, `GOTRUE_JWT_SECRET`, and `GOTRUE_OPERATOR_TOKEN` must be distinct, non-placeholder values; the password must be URI-safe because `GOTRUE_DB_DATABASE_URL` must match the PostgreSQL values exactly and retain `search_path=auth` for the Auth catalog. `GOTRUE_API_EXTERNAL_URL` is required and must be the externally reachable Auth base URL for this environment.

The default binds are `127.0.0.1`. To request LAN binding, set the relevant bind address to `0.0.0.0` **and** set `ALLOW_LAN_BINDING=yes`; firewall and LAN changes remain an owner-controlled operation outside this package.

## Start and operate one explicit project

Every supported command requires an explicit Compose project name. Choose a unique lowercase name, for example `mf-local-dev`:

```bash
./scripts/up.sh .env mf-local-dev
./scripts/inspect.sh .env mf-local-dev
./scripts/lib/compose.sh .env mf-local-dev ps
./scripts/lib/compose.sh .env mf-local-dev logs -f db
./scripts/lib/compose.sh .env mf-local-dev logs -f auth
./scripts/lib/compose.sh .env mf-local-dev down
```

`down` stops containers and removes the project network while preserving the database volume. Do not add `-v`, run a volume-prune command, or remove a volume as routine recovery. Destructive storage replacement requires a validated backup and an explicit owner decision.

## Ports and connections

`POSTGRES_PORT` and `AUTH_PORT` are host port values. The containers always use the fixed internal endpoints `db:5432` and `auth:9999`; changing a host port never changes GoTrue's internal healthcheck or API port.

- PostgreSQL for the future Brain Engine or authorized administration: host bind from `DB_BIND_ADDR:${POSTGRES_PORT}`, database/user from `.env`.
- Auth HTTP: host bind from `AUTH_BIND_ADDR:${AUTH_PORT}`, normally `http://127.0.0.1:${AUTH_PORT}`. A manual host-side check is `curl -fsS "http://127.0.0.1:${AUTH_PORT}/health"`.
- No client receives direct PostgreSQL credentials. Future client Auth use needs a separately approved client contract.

The historical logical Compose volume is `mindforge-postgres-data`; Docker resolves its actual name from the explicit project, normally `<project-name>_mindforge-postgres-data`. The original `infra` project therefore reuses `infra_mindforge-postgres-data`, while disposable projects remain isolated by their own project prefix. Do not hardcode a project-prefixed volume name or use a global `name:` override.

## Migrations and Auth bootstrap

`up.sh` always executes this order: configuration validation, healthy database, deterministic infrastructure migrations, the exact pinned `gotrue migrate` command, then `gotrue serve` and Auth readiness. A failed GoTrue migration prevents Auth startup.

Infrastructure migration files have zero-padded numeric IDs and are replayed lexically by `scripts/migrate.sh`. The `infra.migration_ledger` stores ID, SHA-256, and application time. Exact replays are skipped; a changed applied file, duplicate ID, non-canonical name, or backfill before the highest applied ID fails before new migration DDL.

Run only the infrastructure migration step when needed:

```bash
./scripts/migrate.sh .env mf-local-dev
```

The migration set may contain infrastructure bookkeeping, approved extensions, and the GoTrue bootstrap boundary only. It must not create Mindforge domain objects.

## Inspect

```bash
./scripts/inspect.sh .env mf-local-dev
```

The command independently checks database and Auth containers, PostgreSQL readiness, Auth health, the infrastructure ledger, the GoTrue schema/migration marker, and the approved extension set. It returns non-zero if any one of those checks is red; it does not treat `/health` alone as Auth acceptance.

## Backup

Use an existing external backup directory, never a directory inside this checkout:

```bash
./scripts/backup.sh .env mf-local-dev /absolute/external-backup-directory
```

The script uses owner-private permissions, writes a temporary artifact, and only atomically promotes a non-empty final dump after `pg_dump` succeeds. Dumps, backups, `.env`, partial files, and temporary validation artifacts are ignored locally and are not part of the versioned-surface manifest.

## Restore only to an exclusive disposable target

Restore is deliberately not a generic recovery shortcut. The disposable acceptance runner generates the target ID, environment, project, database, and volume relationship. Its restore project format is `mf-accept-<target-id>-restore`.

```bash
./scripts/restore.sh \
  /absolute/path/to/backup.dump \
  /absolute/path/to/generated-restore.env \
  mf-accept-<target-id>-restore \
  <target-id> \
  RESTORE:<target-id>
```

Before Docker mutation, restore validates the dump with local `pg_restore --list`, the exact target-bound confirmation, generated environment target ID, exclusive project/database/volume, and absence of existing target containers or volume. Auth remains stopped. Restore uses a single PostgreSQL transaction and never selects an observed or existing stack.

## Disposable acceptance matrix

The acceptance runner creates unpredictable IDs and three separate loopback-only projects: `mf-accept-<run-id>-src`, `mf-accept-<run-id>-restore`, and `mf-accept-<run-id>-replay`. It rejects a primary environment, non-loopback bind, reused volume or project, divergent target ID, and VM mode before a local green matrix.

```bash
bash tests/run.sh static all
bash tests/run.sh unit all
bash tests/run.sh integration --local-disposable
```

The local matrix uses only synthetic identities and data. It validates database/Auth readiness, sign-up and password grant without printing a token, vector transaction rollback, a synthetic infra marker across controlled restart, external backup, isolated restore, clean migration replay, and absence of product tables. It stops containers and networks with ordinary `down`, never `-v`; the three exclusive database volumes deliberately remain and are recorded in a protected temporary evidence file.

VM validation is separately authorized only after a fresh local matrix succeeds. Use an owner-authorized key explicitly: `MF_VM_SSH_IDENTITY_FILE=/absolute/path/to/authorized-key bash tests/run.sh integration --vm-disposable`. The runner reruns the local matrix first, then copies only this corrected infra package to a unique temporary directory on the authorized test VM, where it runs the same disposable matrix. It does not use the observed stack, production, firewall/LAN changes, DNS, TLS, Coolify, real data, or volume deletion.

## Secret rotation and recovery boundary

Secrets remain owner-supplied, unversioned configuration. Rotation means choosing new credentials, updating the private environment file and its matching GoTrue database URL, validating it, then performing an owner-approved controlled restart. It never means printing a secret, editing a backup, deleting storage, or treating a volume as a backup.

If startup, migration, Auth, backup, restore, or inspect fails, retain the diagnosable service state and use its specific error. Restarting a container, recreating a service, restoring a disposable target, and deleting storage are different operations with different authority. There is no automatic destructive recovery command in this package.

## Evidence limits

The versioned-surface manifest and a temporary SHA-256 manifest can prove the observed local file surface without exposing configuration values. This checkout has no `.git` metadata, so they do not prove tracking, a commit, a branch, publication, or clean Git state. Static/unit checks do not prove Docker runtime, the test VM, production, Internet/LAN exposure, real data, firewall, DNS, TLS, or Coolify behavior.
