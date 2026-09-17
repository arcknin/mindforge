---
schema_version: 1
flow: orchestration-flow
feature: infra
revision: 2
status: approved
owner_approved_at: "2026-09-07T14:51:46-03:00"
repositories:
  - mindforge
---

# Feature Contract: Initial reduced Supabase infrastructure

## Outcome

Provide a reproducible reduced self-hosted Supabase environment for the future Mindforge system that can be started on the owner's local-network VM. The initial stack consists of PostgreSQL, Supabase Auth/GoTrue and the `pgvector` extension, plus only the initialization and connectivity support required for those components. It must persist across ordinary container restarts, be movable between compatible machines through documented backup and restore, and be ready for feature-driven SQL migrations without defining the Mindforge domain model now.

## Scope

### In

- Create the first physical `src/brain-engine/features/infra/` surface only when implementation is authorized.
- Provide a declarative reduced Supabase stack containing PostgreSQL and Supabase Auth/GoTrue.
- Include only supporting initialization, roles, configuration or routing that is required for PostgreSQL and Auth to operate and expose their contracted connection surfaces.
- Pin every container image and compatibility-sensitive extension to explicit supported versions rather than floating tags.
- Configure durable database storage outside the disposable container lifecycle.
- Provide a healthcheck and a deterministic way to inspect readiness.
- Provide versionable, non-secret configuration examples and keep real credentials out of version control.
- Provide an initially empty location for future ordered SQL migrations and document how they will be applied.
- Install and enable the PostgreSQL `vector` extension (`pgvector`) through a versioned infrastructure migration without creating embeddings, indexes or product tables.
- Prepare Supabase's required Auth database schema, roles and migrations as infrastructure-owned state; these do not authorize Mindforge product tables.
- Provide a documented PostgreSQL connection surface for the future Brain Engine and authorized administration.
- Provide a documented Auth HTTP base URL and key/configuration boundary suitable for future clients, without exposing server-only credentials.
- Provide documented backup and restore operations suitable for moving the database between compatible machines.
- Document startup, shutdown, status, logs, backup, restore, migration and recovery procedures.
- Allow an explicit owner-configured LAN binding without making unrestricted network exposure the default.

### Out

- Domain tables, product schemas, columns, relationships, constraints, seeds or feature data.
- A first migration that invents `users`, `brains`, `neural_layers`, `documents` or any other product model.
- Brain Engine or Hidden Cortex Go source code.
- HTTP endpoints, OpenAPI operations or Flutter integration.
- PostgREST, Studio, Storage, Realtime, Edge Functions, Analytics/Logflare, Vector log collection, queues, schedulers, Model Gateway or other Supabase/application services not required by PostgreSQL or Auth.
- Choosing an embedding provider or model, creating vector columns or indexes, generating embeddings, or implementing semantic retrieval.
- A connection pooler such as Supavisor unless a later approved requirement demonstrates that direct PostgreSQL connections are insufficient.
- Coolify configuration, DNS, ingress, TLS, firewall changes or deployment to the owner's VM.
- Production or Internet exposure.

## Observable behavior

- From a machine with the documented container prerequisites, an operator can configure the required variables and start the reduced Supabase stack through the documented command.
- Readiness is reported independently for PostgreSQL and Auth through deterministic checks rather than inferred only from running container processes.
- Auth can reach its database, expose its documented health/API surface and complete a disposable local authentication flow using non-production test identity data.
- The `vector` extension is present and enabled, and its version can be inspected deterministically without creating product vector data.
- Stopping and starting the stack without deleting its storage preserves database data.
- An operator can create a backup artifact outside the container and restore it into a clean compatible database environment.
- The migration workflow can inspect an empty migration set successfully and is ready to apply future feature migrations in deterministic order.
- A fresh machine can reconstruct the same database service definition from the versioned repository plus owner-supplied configuration and, when required, a backup artifact.

## Interfaces and data

- The infrastructure exposes PostgreSQL's native connection interface for the future Brain Engine and authorized administration, and an HTTP interface for Supabase Auth. No Mindforge application API is part of this contract.
- Flutter/client code may use the Auth interface and public client configuration when later authorized, but must never receive direct PostgreSQL credentials or server-only Auth credentials.
- Connection settings, Auth URLs, redirect allowlists, JWT/key material boundaries and database credentials are documented through non-secret examples; real secrets remain owner-supplied and unversioned.
- Real database data lives in durable storage and is transferred through backup/restore artifacts; the container volume itself is not considered a backup.
- Future product schema changes enter only through ordered, immutable-after-application SQL migration files.
- Infrastructure migrations may create only migration bookkeeping, Supabase Auth-required state and the `vector` extension. This does not authorize Mindforge product tables.

## Constraints

- Keep infrastructure under `src/brain-engine/features/infra/` in the root monorepo and never initialize a nested Git repository.
- Use a declarative container definition that is portable to the owner's Windows-hosted local-network VM environment.
- Default network binding must be local or otherwise non-public. LAN access is opt-in through explicit configuration and remains subject to the owner's VM and firewall controls.
- Direct PostgreSQL access is restricted to the Brain Engine, authorized operators and internal service connectivity; exposing Auth does not authorize direct client database access.
- Auth redirect URLs and origins must be explicit and fail closed for unlisted destinations; placeholder/default credentials must prevent startup rather than create an apparently usable insecure environment.
- Do not commit passwords, connection strings containing secrets, real backups or machine-specific addresses.
- Backup output must be stored outside the disposable container and excluded from version control.
- Restore must fail clearly on incompatible inputs and must not silently overwrite an unidentified database.
- The exact schema and every future domain migration must be derived from an approved feature contract, not from the conceptual architecture alone.

## Acceptance criteria

- [ ] A clean checkout plus documented prerequisites can start the pinned reduced Supabase stack using the documented command.
- [ ] PostgreSQL and Auth each reach a healthy state and their exact readiness commands succeed.
- [ ] Auth completes a disposable local sign-up/sign-in or equivalent token issuance flow and no server-only secret is exposed to a client surface.
- [ ] The `vector` extension is enabled at the pinned/compatible version and a disposable vector operation succeeds without leaving product schema or data behind.
- [ ] A disposable validation record survives an ordinary stop/start cycle without deleting storage.
- [ ] Backup creates a non-empty artifact outside the container, without committing it to Git.
- [ ] Restore into a separate clean compatible database recovers the disposable validation record.
- [ ] The migration workflow applies infrastructure-owned Auth and `vector` setup deterministically, then handles an empty Mindforge product migration set without creating any Mindforge product table.
- [ ] Real secrets and backups are absent from the versioned file set; a non-secret configuration example documents every required variable.
- [ ] The runbook documents startup, shutdown, status, logs, PostgreSQL connection, Auth connection/configuration, migration, backup, restore, secret rotation boundary and explicit recovery steps.
- [ ] Static inspection confirms there are no Mindforge domain tables, application endpoints or Supabase services beyond PostgreSQL, Auth and their strictly required support.

## Validation boundary

- Authorized now: documentation and contract consistency checks only.
- After owner approval and an executable plan: static configuration validation and disposable local container validation may be planned.
- Runtime validation must use disposable test data and must distinguish PostgreSQL readiness, Auth readiness/flow, `vector` availability, persistence, backup and restore as separate evidence.
- Deployment or execution on the owner's VM, LAN/firewall changes and handling of real data require separate owner authority and owner-provided access.
- No production, Internet-facing, Coolify, DNS or real-data validation is authorized by this contract.

## Authority required

- Owner approval of this revision before planning or implementation.
- Separate owner authorization before connecting to or changing the target VM, opening LAN ports, changing firewall rules, deploying services, restoring over a non-disposable database or handling real backup data.

## Failure and recovery

- Startup failure must leave a diagnosable service state and must not delete durable storage automatically.
- Failure of PostgreSQL, Auth or required initialization must not be reported as a ready stack.
- Auth must fail closed when required database connectivity, secrets, issuer/base URL or redirect configuration is absent or invalid.
- Backup failure must return a failure and must not present a partial artifact as usable.
- Restore must target an explicitly identified disposable or owner-approved destination and fail closed on missing or incompatible input.
- Destructive removal of volumes or replacement of an existing database is never an automatic recovery step.
- Recovery documentation must preserve the distinction between restarting a container, recreating a service, restoring data and deleting storage.

## Assumptions

- The target VM will provide a Docker Compose-compatible container runtime; exact VM access and host configuration will be supplied later by the owner.
- PostgreSQL and Supabase Auth/GoTrue are the application services required by this initial contract; narrowly required initialization or routing support may be added by the plan with an explicit dependency justification.
- The Supabase PostgreSQL distribution must provide a compatible `vector` extension; `pgvector` is enabled now, while embeddings and retrieval behavior remain future feature work.
- The implementation planner may choose the migration runner and supporting file layout as long as SQL migrations remain versioned, ordered and usable without Brain Engine source code.
- Direct LAN access to PostgreSQL is not required by default; the owner can explicitly enable a restricted binding when needed.
- Port numbers, database names, usernames and the exact pinned PostgreSQL version are implementation configuration rather than product behavior, provided they are documented and portable.

## Owner decisions

- Start with database infrastructure before Brain Engine and client implementation.
- Do not design product tables before their corresponding features are explored and contracted.
- Keep the infrastructure inside `src/brain-engine/features/infra/` within the root monorepo.
- Prepare migrations now as an operational path, but create domain migrations only from future approved feature work.
- Prioritize reproducible startup, persistence, backup, restore and migration between machines.
- Use a reduced self-hosted Supabase stack with PostgreSQL, Auth/GoTrue and `pgvector` from the first infrastructure delivery.
- Prepare the connection boundaries needed by the future Brain Engine and clients while preserving the Brain Engine as the only authority over Mindforge product data and effects.

## Amendments

### INFRA-A01 — Reduced Supabase foundation

- **Resulting revision:** 2
- **Reason:** the owner confirmed that Auth and vector support are required in the initial infrastructure rather than deferred.
- **Evidence:** owner clarification on 2026-09-05; prior recorded direction for reduced Supabase with PostgreSQL and Auth; current architecture direction for PostgreSQL plus `pgvector`.
- **Changed clauses:** outcome, in/out scope, observable behavior, interfaces, constraints, acceptance criteria, validation, failure handling, assumptions and owner decisions.
- **Preserved clauses:** no Mindforge domain model, no Brain Engine/Hidden Cortex source, no Mindforge HTTP endpoints, no deployment/VM mutation, no production exposure and separate authority for real data or destructive operations.
- **Owner approval:** approved by the owner on 2026-09-07 at 14:51:46-03:00.
