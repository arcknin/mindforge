# Mindforge Infra — durable PostgreSQL, Auth and pgvector

This package implements the infrastructure boundary in approved infra contract revision 6. The production/main database is `mindforge`;
`mindforge_develop` is a distinct database, with an independent Supabase
Auth/GoTrue service for each target. It creates no Mindforge domain tables,
HTTP endpoints, OpenAPI operation, PostgREST service, Studio, Storage,
Realtime, queue or production deployment.

The cluster inventory may include unrelated databases such as `nsrdb`.
Mindforge operations accept only `mindforge_develop` or `mindforge`; unrelated
databases are passive shared-cluster context. The operator does not inspect the contents of unrelated databases, and unrelated databases are never operation targets. Target operations still require their own owner authorization.

SQL execution and real backup handling remain VM-only and require separate
owner authorization. VM access and database mutation also need their own
authorization and owner-supplied private configuration. Local validation is
static, unit or fake-backed; it does not start a PostgreSQL runtime or execute
SQL.

## Prerequisites

The designated VM operator needs:

- Docker Engine with Compose v2 for the stack lifecycle;
- Bash 3.2 or later;
- Go only when building the distributable operator binary;
- PostgreSQL client utilities matching the pinned server major version:
  `psql`, `pg_dump` and `pg_restore`;
- an existing external backup directory for backup artifacts and sanitized evidence.

The Compose file pins the PostgreSQL image by digest and the GoTrue image by
version. The `vector` extension is accepted in the contracted `0.8.x` range.
The host PostgreSQL utilities are prerequisites for the binary operations;
they are not a local database runtime.

## Private configuration

Create the owner-private configuration and validate it before any VM action:

```bash
cp .env.example .env
# Edit .env with owner-supplied values. Never commit it.
./scripts/validate-config.sh .env
```

The configuration names production/main and develop database targets and both Auth boundaries:
`DEVELOP_DB_*`, `MAIN_DB_*`, `DEVELOP_AUTH_*` and `MAIN_AUTH_*`. The validator
requires the exact `mindforge_develop` and `mindforge` names, distinct
roles and secrets, internal `db:5432` Auth URLs with `search_path=auth`,
explicit redirect lists and non-public default binds. `GOTRUE_API_EXTERNAL_URL`
must identify each environment's Auth base URL. `GOTRUE_MAILER_AUTOCONFIRM` is
an owner choice for the target environment; it is not a local test switch.

## Build and distribute the operator

Build for the declared destination platform and keep the generated binary
outside the checkout:

```bash
GOOS=linux GOARCH=amd64 go build \
  -o /absolute/path/to/mindforge-infra \
  ./src/brain-engine/features/infra/tools/cmd/mindforge-infra
export MINDFORGE_INFRA_BIN=/absolute/path/to/mindforge-infra
```

The resulting Go binary does not require Go or Python when executed on the
selected platform. Database operations still require the compatible external
PostgreSQL client utilities listed above.

## VM-only lifecycle and explicit develop provisioning

Use a named Compose project for the owner-designated VM stack and provide an
operator plus a non-secret authorization reference on every operation:

```bash
./scripts/up.sh .env infra operator-id authorization-ref
./scripts/inspect.sh .env infra operator-id authorization-ref
./scripts/lib/compose.sh .env infra ps
./scripts/lib/compose.sh .env infra logs -f db
./scripts/lib/compose.sh .env infra logs -f auth-develop
./scripts/lib/compose.sh .env infra logs -f auth
./scripts/lib/compose.sh .env infra down
```

The Compose project name `infra` preserves the existing
`infra_mindforge-postgres-data` volume on the designated cluster. The existing
`auth` service remains bound to `mindforge`; `auth-develop` has its own
credentials and binds only to `mindforge_develop`.

On an existing initialized cluster, normal startup first checks both
Mindforge targets without mutation. If develop is absent or incomplete,
startup reports an incomplete result, blocks develop migration/Auth startup,
and checks main Auth health read-only. It does not stop, recreate, rebind or
otherwise mutate the existing PostgreSQL cluster or main Auth service. Normal
startup and restart never create, replace, rename, or copy either database.

Adding develop to the existing volume is an explicit, separately authorized
provisioning action. After authorizing that target-specific operation, invoke
the fixed-target adapter:

```bash
./scripts/provision-develop.sh .env mindforge_develop operator-id authorization-ref
```

The adapter delegates to the existing operator, which performs a read-only
cluster inventory and recheck before adding only the configured develop role
and `mindforge_develop`. It requires main to exist and develop to be absent;
it refuses unexpected or partial state and never mutates `mindforge`. This
command is not part of startup. On a truly fresh empty cluster only, the
separate initdb bootstrap creates both databases; it is not used to provision
an existing volume.

When both targets are complete, startup applies their independent
infrastructure migrations and Auth migrations/readiness checks. Inspection
reports the database and each database/Auth pair independently. Stopping the
project with `down` preserves the durable volume; `down -v`, volume pruning
and direct volume removal are not recovery procedures.

Compose lifecycle commands are for the designated VM stack. They do not
authorize a local or disposable database substitute.

`POSTGRES_PORT`, `DEVELOP_AUTH_PORT` and `MAIN_AUTH_PORT` are host port values;
the containers use `db:5432` and Auth `:9999` internally. A separate authorized
host-side Auth health check may use `curl` against the configured Auth host port.

## Migration workflow

Infrastructure migrations are ordered, hash-checked and target-specific. The
approved feature migration catalog maps Brain Core to global infra-catalog ID `0006`, after infrastructure IDs `0001` through `0005`. The descriptor at
`src/brain-engine/features/infra/postgres/feature-migrations/catalog.json`
points to the canonical feature-owned
`src/brain-engine/features/brain/migrations/0001_brain_core.sql`, its
`0001_brain_core.preflight.sql`, and `0001_brain_core.verify.sql`. It pins the
SHA-256 hashes of the canonical payload and both checks; the payload is not
copied into infra. `infra.migration_ledger` is the only applied-migration ledger.

Apply infrastructure state to an explicit target through the binary adapter:

```bash
./scripts/migrate.sh .env mindforge_develop operator-id authorization-ref infrastructure
```

The Brain Core feature migration is applied through the same operator, first
in `mindforge_develop`:

```bash
./scripts/migrate.sh .env mindforge_develop operator-id authorization-ref feature verification/brain-core-run-id
```

The Go operator resolves the registered feature-owned sources from the
repository root and performs the read-only preflight. It applies the canonical
payload, runs schema verification and writes the ledger record in the same transaction. A matching global ID and source hash is a no-op only when
preflight confirms the complete schema. Feature migrations cannot target
`mindforge`; promotion applies the same catalog-resolved bytes to main only
after complete develop evidence, separate production authorization, a fresh
validated backup and successful main preflight/recheck. Altered source bytes,
an unknown target, omitted authority, partial schema or incomplete evidence
fail closed. A correction is a new ordered migration.

## Promotion and backup

Promotion never copies users, sessions, feature data, a database or a volume.
It verifies the exact feature catalog and successful develop evidence, performs
a read-only main preflight, creates and validates a fresh main backup outside
Git, rechecks main for drift, applies the same bytes and hash, and records
target evidence:

```bash
"$MINDFORGE_INFRA_BIN" promote \
  --config .env \
  --source-root /absolute/path/to/mindforge \
  --target mindforge \
  --operator-id operator-id \
  --authorization-ref authorization-ref \
  --output-dir /absolute/external/infra-evidence \
  --verification-ref verification/feature-id
```

The promotion command requires an external backup directory. It validates
`pg_dump` completion, archive listing through `pg_restore --list`, the archive
hash and target-bound manifest before any main mutation. Failed or stale
preflight, changed source bytes, drift, incomplete develop evidence or a
missing backup preserves a sanitized failure record and leaves main untouched.
An identical already-applied hash is an idempotent outcome.

The adapter form for an independent backup check is:

```bash
./scripts/backup.sh .env mindforge operator-id authorization-ref \
  /absolute/external/infra-evidence
```

Backup artifacts, manifests and promotion evidence remain outside this
checkout. A backup check proves completion, integrity, listing and target
identity. Backup validation does not prove restoration.

## Guarded restore and recovery

Restore is a separate, explicit recovery capability. It requires an
environment-bound manifest, matching archive hash and listing, an identified
target, operator authorization and a target-bound replacement confirmation:

```bash
./scripts/restore.sh \
  /absolute/external/infra-evidence/mindforge-infra-*.dump \
  /absolute/external/infra-evidence/mindforge-infra-*.manifest.json \
  .env mindforge operator-id authorization-ref \
  replace/mindforge/owner-confirmation \
  /absolute/external/infra-evidence
```

Restore never silently selects a database, volume or source target and never
serves as promotion. Practical restoration of a fresh dump remains pending;
there is no restore proof claim in local or static evidence. Restarting a
container, recreating a service, restoring a backup and deleting storage are
separate actions with separate authority.

## VM evidence sequence

The owner-run evidence sequence is separate for each gate:

1. authorize VM access, target access and mutation independently;
2. validate the private configuration and the pinned Compose render;
3. record PostgreSQL readiness and independent Auth readiness for both pairs;
4. run read-only preflight for `mindforge_develop` and `mindforge`;
5. verify infrastructure ledger, Auth schema markers and `vector` in both targets;
6. verify environment-isolated Auth flows and stop/start persistence;
7. before promotion, create and validate the fresh `mindforge` backup;
8. after separate authorization, apply and verify Brain Core global migration
   `0006` in develop;
9. promote the immutable bytes to main only after the sequence above is green.

Every result must identify its target, operator, authority reference and
sanitized evidence class. Runtime status is not inferred from documentation,
mock output, static checks, unit tests or a build.

## Secrets and operational limits

Real credentials, Auth tokens, connection URLs containing secrets, database
artifacts and machine-specific addresses stay in owner-private configuration or
external storage. The repository ignores `.env`, generated binaries, backup
artifacts, manifests and sanitized operational evidence. Secret rotation means
changing the private values and matching Auth URLs, validating them, and then
performing an owner-approved controlled restart.

This package exposes no Mindforge endpoint. `contracts/openapi/registry.yaml`
remains empty until a separate approved Brain Engine HTTP contract exists.
Local plan and static or fake-backed evidence does not prove VM state, SQL application, backup or restore behavior, or runtime behavior.
