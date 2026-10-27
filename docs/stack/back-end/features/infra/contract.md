---
schema_version: 1
flow: orchestration-flow
feature: infra
revision: 6
status: approved
owner_approved_at: "2026-09-30T20:00:01Z"
tdd: yes
implementation_authorized_at: null
implementation_authorized_batches: []
repositories:
  - mindforge
---

# Feature Contract: Initial reduced Supabase infrastructure

## Outcome

Provide a reproducible reduced self-hosted Supabase environment on the owner's designated VM while preserving the existing production/main PostgreSQL database named `mindforge`, its data, and its durable volume. The physical database name `mindforge` is the explicit main target; this revision does not rename it to `mindforge_main`, copy its data, or authorize production SQL, backup, deployment, or service mutation. The approved revision-3 develop-first model is carried forward with a separate `mindforge_develop` target. This topology is approved, while any provisioning or mutation on the existing cluster still requires separate authorization. The two-Auth model is preserved: the existing main Auth service remains connected to `mindforge`, and any develop Auth service is isolated to `mindforge_develop`; identities, sessions, secrets and configuration are not copied between them. Feature migrations may be verified in develop and promoted to the production/main target only through a separately authorized operation with explicit target `mindforge`, a read-only preflight and a fresh validated backup. The stack remains movable between compatible machines through documented backup and restore without defining the Mindforge domain model now. The feature also defines a future Go-built distributable infrastructure-operations tool under `src/brain-engine/features/infra/tools/`, runnable without Go or Python installed on a third-party machine, for backup/dump, restore, preflight, apply, promote and audit; this revision authorizes only the narrowly scoped adaptation of the existing operator recorded below, not unrelated tool capabilities or any VM/database operation.

The designated VM and its PostgreSQL cluster may also host other projects. Only `mindforge` and `mindforge_develop` are Mindforge targets; an unrelated database such as `nsrdb` may coexist passively and must not become a target, be inspected beyond the minimum read-only cluster inventory, or be modified by Mindforge operations. The earlier assumption that these were the only application databases in the cluster is superseded.

## Scope

### In

- Create the first physical `src/brain-engine/features/infra/` surface only when implementation is authorized.
- Preserve the existing persistent PostgreSQL cluster and the existing main/production application database named `mindforge`, including its name and data. PostgreSQL's required system databases are excluded from the application database count.
- Keep the separate `mindforge_develop` database as the approved develop target alongside production/main `mindforge`. On the existing persistent cluster it may be provisioned only by an explicit, separately authorized additive operation after a read-only preflight; ordinary startup and restart must not silently create or replace it.
- Allow unrelated project databases to coexist in the same PostgreSQL cluster without making their presence alone a failure of Mindforge provisioning, readiness or promotion. Keep all Mindforge operations restricted to the two explicit Mindforge targets and their own roles, Auth services and evidence.
- Preserve two isolated Supabase Auth/GoTrue boundaries in the approved two-target topology: the existing main Auth service remains connected only to `mindforge`; a develop Auth service connects only to `mindforge_develop` after that target is explicitly provisioned and configured.
- Include only supporting initialization, roles, configuration or routing that is required for PostgreSQL and Auth to operate and expose their contracted connection surfaces.
- Pin every container image and compatibility-sensitive extension to explicit supported versions rather than floating tags.
- Configure durable database storage outside the container lifecycle for both databases.
- Provide healthchecks and deterministic inspection of PostgreSQL plus each Auth/database environment independently.
- Provide versionable, non-secret configuration examples and keep real credentials out of version control.
- Provide an initially empty location for future ordered SQL migrations and document a develop-first workflow whose production/main destination is explicitly named `mindforge`.
- Coordinate approved product migrations through the existing Go operator and one authoritative migration ledger per Mindforge target, while each feature remains responsible for its canonical SQL and schema-specific validation.
- Install and enable the PostgreSQL `vector` extension (`pgvector`) independently in each explicitly contracted application database through versioned infrastructure migrations without creating embeddings, indexes or product tables.
- Prepare Supabase's required Auth database schema, roles and migrations independently in each explicitly provisioned database as infrastructure-owned state; these do not authorize Mindforge product tables.
- Provide a documented PostgreSQL connection surface for the future Brain Engine and authorized administration.
- Provide a documented Auth HTTP base URL and key/configuration boundary suitable for future clients, without exposing server-only credentials.
- Provide documented backup and restore operations suitable for moving the database between compatible machines.
- Require separate production authorization, a read-only preflight and a fresh validated backup of the explicit target `mindforge` before any feature migration promotion into it.
- Specify, for future owner-authorized implementation, a Go-built distributable infrastructure-operations binary under `src/brain-engine/features/infra/tools/` that can perform backup/dump, restore, preflight, apply, promote and audit operations without requiring Go or Python on a third-party machine. This contract fixes capabilities and guardrails, not CLI/subcommands, JSON schemas or detailed file layout.
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
- Production deployment, Internet/LAN exposure changes, production SQL execution, real backup handling, or service mutation; each remains a separate owner-authorized gate. The existing `mindforge` database is already designated production/main by the owner.
- Renaming, replacing, deleting, moving, copying or merging the existing `mindforge` database or its contents as a consequence of this contract amendment.
- Reading the contents of, connecting to, backing up, restoring, migrating, changing roles or permissions for, renaming or deleting `nsrdb` or any other unrelated project database. Their presence in a read-only cluster-name inventory does not authorize an operation on them.
- Copying or merging users, sessions, tokens or feature data between `mindforge_develop` and `mindforge` as a migration-promotion mechanism.
- Local or disposable PostgreSQL/Docker runtime execution of infrastructure, Auth or feature SQL migrations.
- Adding unrelated infrastructure-operations capabilities or executing the operator against a VM, database or backup without separate owner authorization; the narrowly scoped adaptation of the existing operator is authorized below. No target, restore or promotion may be implicit.
- Applying a product migration through standalone feature `.psql` scripts outside the target-controlled develop-first operator flow, or treating a feature-local ledger as independent proof of eligibility for promotion.
- Practical testing of restoring a fresh promotion backup; this remains explicitly pending for a later phase and is not a restore-proof claim in this revision.

## Observable behavior

- On the designated VM, an authorized operator can configure the required variables and start the reduced Supabase stack through the documented command.
- PostgreSQL readiness and the readiness of each Auth service/database pair are reported independently through deterministic checks rather than inferred only from running container processes.
- The existing `mindforge` database retains its name, data, durable volume and main Auth database binding through any separately authorized infrastructure transition; revision-4 contract approval alone changes none of them.
- In the approved two-target topology, `mindforge_develop` is a distinct database and never aliases `mindforge`. On the existing persistent volume, a normal startup/restart does not create it implicitly; absence is reported as an incomplete target until separately authorized additive provisioning succeeds.
- An unrelated database in the shared PostgreSQL cluster does not by itself block an otherwise eligible Mindforge-target preflight, additive develop provisioning, startup inspection or migration promotion. An inconsistent Mindforge target, role, Auth binding, authorization or evidence still fails closed.
- Each configured Auth service reaches only its assigned database, exposes its documented health/API surface and can complete an authorized authentication flow using environment-specific identity data. The existing main Auth boundary remains on `mindforge`.
- Users, sessions, Auth configuration, secrets and tokens from `mindforge_develop` and `mindforge` are not accepted as interchangeable.
- Infrastructure migrations, Auth bootstrap/migrations and the `vector` extension can be inspected independently in each explicitly provisioned target without creating product vector data.
- Stopping and starting the stack without deleting its storage preserves the existing `mindforge` database and, if separately provisioned, the independent develop database.
- The future operations tool is distributable as a Go binary runnable without Go or Python installed on a third-party machine; operations that access a VM/database or handle a backup name an explicit target and require the corresponding owner authorization. Purely local static audit needs no VM authority; no database target, restore or promotion is inferred.
- An authorized operator can create a backup artifact outside the container. The documented restore capability can recover an explicitly identified target from its own environment-bound backup, or use another explicitly approved compatible destination, with separate authority for any replacement; practical restoration of a fresh dump remains outside this revision and is not claimed as verified.
- The migration workflow applies and verifies a feature migration first in `mindforge_develop`, then promotes the exact same bytes and hash to production/main `mindforge` only after separate production authorization, a fresh validated backup and read-only preflight.
- The operator registers one globally ordered identity and hash for each approved feature migration in the authoritative ledger of each Mindforge target. A feature may retain a local filename such as `0001_brain_core.sql`, but that local number is not the global catalog identity and cannot collide with infra migrations `0001`–`0005`.
- Each target-specific migration record links the migration ID/hash, target, authorized operator, preflight, post-application result and the behavior/access verification required by the feature; incomplete evidence blocks promotion.
- Failure, drift or uncertain evidence in `mindforge_develop`, or missing production authorization/backup/preflight for `mindforge`, blocks promotion without changing `mindforge`.
- A fresh machine can reconstruct the same database service definition from the versioned repository plus owner-supplied configuration and, when required, a backup artifact.

## Interfaces and data

- The infrastructure exposes explicit PostgreSQL targets: `mindforge` is the existing physical database and production/main target; `mindforge_develop` is the separately named develop target in the approved topology. Their Auth boundaries are distinct. No Mindforge application API is part of this contract.
- Cluster inventory may report unrelated database names, but those names are passive context and never enter the accepted Mindforge target set, Auth configuration, backup/restore destination, migration catalog or promotion evidence.
- Flutter/client code may use the Auth interface and public client configuration when later authorized, but must never receive direct PostgreSQL credentials or server-only Auth credentials.
- Connection settings, Auth URLs, redirect allowlists, JWT/key material boundaries and database credentials are documented separately through non-secret examples for each environment; real secrets remain owner-supplied and unversioned.
- Auth identities, sessions, configuration, secrets and issued tokens are environment-bound and must not be copied, shared or treated as valid across `mindforge_develop` and `mindforge`.
- Real database data lives in durable storage and is transferred through backup/restore artifacts; the container volume itself is not considered a backup.
- Future product schema changes enter only through ordered SQL migration files whose bytes and hash become immutable after first application.
- A feature owns the canonical SQL source and any required schema-specific preflight/post-application checks. The operator binds those checks and the exact source bytes/hash to its explicit target and migration identity; the catalog must not silently maintain a divergent SQL copy.
- `infra.migration_ledger` is the single authoritative applied-migration history in each Mindforge database for the coordinated workflow. Brain Core's prepared `brain_core.migration_ledger` and standalone `.psql` envelopes are not a parallel execution or promotion authority; an approved implementation plan must adapt or retire them before application.
- Any change to product SQL objects or permissions—tables, functions including RPCs, views, indexes, triggers, RLS/policies, ACL/grants or product roles where applicable—must be in a versioned migration derived from an approved feature contract. A SQL function or RPC does not automatically become an HTTP/PostgREST endpoint or an OpenAPI operation.
- Each target-specific migration evidence record links the migration ID/hash, target, authorized operator, preflight, post-application result and the behavior/access verification required by the feature; missing or incomplete evidence blocks promotion.
- Promotion means applying the same migration bytes/hash already verified in `mindforge_develop` to the explicitly selected production/main database `mindforge`, only with separate production authorization; it is not a dump, restore, volume copy or data merge.
- Infrastructure migrations may independently create in each database only migration bookkeeping, Supabase Auth-required state and the `vector` extension. This does not authorize Mindforge product tables.

## Constraints

- Keep infrastructure under `src/brain-engine/features/infra/` in the root monorepo and never initialize a nested Git repository.
- Use a declarative container definition that is portable to the owner's Windows-hosted local-network VM environment and preserves two Mindforge databases in one PostgreSQL cluster, which may also contain unrelated project databases.
- Never infer authority over the VM, PostgreSQL cluster or unrelated databases from the Mindforge stack's presence. Check only Mindforge-owned target state needed for an operation; reject missing, conflicting or inconsistent Mindforge targets and roles without treating passive unrelated databases as drift.
- Default network binding must be local or otherwise non-public. LAN access is opt-in through explicit configuration and remains subject to the owner's VM and firewall controls.
- Direct PostgreSQL access is restricted to the Brain Engine, authorized operators and internal service connectivity; exposing Auth does not authorize direct client database access.
- Auth redirect URLs and origins must be explicit and fail closed for unlisted destinations; placeholder/default credentials must prevent startup rather than create an apparently usable insecure environment.
- Do not commit passwords, connection strings containing secrets, real backups or machine-specific addresses.
- Backup output must be stored outside the PostgreSQL container and excluded from version control.
- Restore must fail clearly on incompatible inputs and must not silently overwrite an unidentified database; recovery into `mindforge_develop` or `mindforge` from its own environment-specific backup requires explicit replacement authority and is not migration promotion.
- Before every promotion to production/main `mindforge`, generate a new dump/backup outside Git, verify completion, integrity, readability/listing and hash, and link that evidence to the explicit `mindforge` target. An incomplete check blocks promotion.
- Backup validation is separate from restore proof. A valid backup check does not claim that the dump was restored; practical restoration of a fresh dump is outside this revision and remains pending.
- Runtime SQL execution is allowed only in the designated VM/PostgreSQL cluster with separate owner authority for access and mutation; local or disposable PostgreSQL/Docker targets may not be used as substitutes.
- Local validation of SQL is limited to non-runtime checks such as static analysis, syntax-oriented inspection that does not execute SQL, ordering, hashing and repository consistency.
- Every feature migration must pass in `mindforge_develop` before it is eligible for production/main `mindforge`; failure, drift, mismatched hash or inconclusive evidence blocks promotion.
- Feature-specific checks must reject a partial or incompatible product schema before DDL, verify the required shape after DDL, and fail closed on hash or history mismatch. For one migration attempt, DDL and its authoritative ledger entry must commit together or both roll back; uncertain commit outcomes require read-only reconciliation before retry.
- Promotion to `mindforge` requires separate production authorization, a validated backup and a successful read-only preflight of that database before any mutation.
- Every product SQL object or permission change—tables, functions including RPCs, views, indexes, triggers, RLS/policies, ACL/grants or product roles where applicable—must be delivered through a versioned migration derived from an approved feature contract. SQL functions/RPCs do not automatically expose HTTP/PostgREST endpoints; an HTTP operation requires a separate approved contract.
- The future operations tool must be distributed as a Go binary runnable without Go or Python installed on a third-party machine. Every VM/database or backup operation must have an explicit operation, target and applicable owner authorization; missing or ambiguous authority/target fails closed, and agents may execute those operations only when explicitly authorized. Purely local static audit may run without VM/database authority.
- A migration applied to either database is immutable; corrections use a new ordered migration rather than rewriting applied bytes.
- The exact schema and every future domain migration must be derived from an approved feature contract, not from the conceptual architecture alone.

## Acceptance criteria

- [ ] The existing designated-VM PostgreSQL cluster retains the application database named `mindforge`, its data, its durable volume and its existing main Auth database binding. Contract approval alone performs no database, volume, role, Auth, service or network mutation.
- [ ] On a fresh compatible cluster, the approved two-target topology contains exactly `mindforge_develop` and `mindforge` as Mindforge application databases, apart from PostgreSQL's required system databases; it does not create a database named `mindforge_main`.
- [ ] Given an existing shared cluster containing `mindforge`, `nsrdb` and PostgreSQL system databases but no `mindforge_develop`, a read-only preflight identifies the two Mindforge target states and does not reject provisioning solely because `nsrdb` exists; no connection to `nsrdb` or mutation occurs.
- [ ] Given separate authority for additive develop provisioning and otherwise eligible Mindforge target/role state, the operator creates only the configured Mindforge develop role and `mindforge_develop`; `mindforge`, `nsrdb` and other unrelated databases, roles, data and services remain unchanged. A conflicting Mindforge target or role, missing authority or changed Mindforge state still blocks the operation.
- [ ] Given both Mindforge targets present alongside unrelated databases, startup preflight and migration promotion evaluate the two explicit Mindforge targets without accepting an unrelated database as a target or weakening Auth, backup, migration-evidence or production-authorization checks.
- [ ] On the existing persistent cluster, ordinary startup/restart does not silently create `mindforge_develop`, rename/recreate `mindforge`, or run the fresh-volume bootstrap against the protected volume. Absence of develop remains an explicit incomplete-target state until separate additive provisioning is authorized and verified.
- [ ] If separately authorized, additive provisioning creates a distinct `mindforge_develop` and its isolated Auth boundary without changing the existing `mindforge` database, data, volume or main Auth binding; read-only preflight and post-operation evidence identify both targets by physical database name.
- [ ] PostgreSQL and each configured Auth service reach healthy states, and readiness proves the existing main Auth remains connected only to `mindforge` and any develop Auth is connected only to `mindforge_develop`.
- [ ] Each explicitly provisioned database independently contains the required infrastructure ledger, Auth bootstrap/migrations and compatible `vector` extension, with no Mindforge product table created by infra.
- [ ] An authorized environment-specific authentication flow succeeds independently in each provisioned environment, while users, sessions, configuration, secrets and tokens are not interchangeable and no server-only secret is exposed to a client surface.
- [ ] Independent validation records survive an ordinary stop/start cycle without deleting the protected storage.
- [ ] Main operations name `mindforge` explicitly as the physical production/main target; preflight, backup manifests, migration evidence and sanitized results bind to `mindforge`, never to an inferred alias or physical `mindforge_main` database.
- [ ] Before each production promotion to `mindforge`, the owner separately authorizes the production mutation; a fresh backup of `mindforge` is created outside the PostgreSQL container and Git; completion, integrity, readability/listing and hash are verified and linked to `mindforge`; and a successful read-only preflight/recheck is required. Missing authority or evidence blocks mutation.
- [ ] Restore capability is documented and guarded for recovery of `mindforge_develop` or `mindforge` from its own environment-specific backup, or for another explicitly approved compatible target. It refuses missing, incompatible or ambiguous inputs and unauthorized replacement; practical restoration of a fresh dump remains outside this revision and is not claimed as verified.
- [ ] Given an authorized feature migration under the approved two-target topology, its exact bytes/hash are applied and verified first in `mindforge_develop`; target-specific evidence links the migration ID/hash, target, authorized operator, preflight, post-application result and feature-required behavior/access verification, and incomplete evidence prevents promotion.
- [ ] Given a migration verified in `mindforge_develop`, promotion refuses to mutate production/main `mindforge` until separate production authorization, fresh validated backup and successful read-only preflight/recheck exist, then applies exactly the same bytes/hash and records target-specific evidence without copying users or feature data.
- [ ] The approved Brain Core initial migration is registered once in the global feature catalog under a unique identity after infra `0001`–`0005`; its canonical feature-owned SQL bytes and hash are the same for develop and main, without a second divergent SQL payload or a feature-local ledger acting as application proof.
- [ ] On an eligible develop target, the operator's read-only preflight confirms the exact source/hash and absence of conflicting Brain Core objects; apply performs the feature checks, DDL and authoritative ledger write as one committed outcome. Partial objects, conflicting hash/history or insufficient privilege fail closed without claiming success.
- [ ] If an attempt fails before commit, the Brain Core DDL and ledger entry are both absent; if the connection fails around commit, a new read-only preflight distinguishes applied, absent and inconsistent states before any retry. Reapplication of the exact applied migration is a no-op; divergent bytes never overwrite history.
- [ ] Promotion to `mindforge` accepts only the same global migration identity and bytes/hash with complete develop evidence, separate production authorization, a fresh validated main backup and successful read-only preflight/recheck. A standalone Brain Core `.psql` result or ledger cannot substitute for this evidence.
- [ ] Re-execution of an identical applied migration is idempotent, while modification of applied bytes is rejected and a correction requires a new ordered migration.
- [ ] Real secrets and backups are absent from the versioned file set; a non-secret configuration example documents every required variable.
- [ ] The runbook documents startup, shutdown, status, logs, PostgreSQL connection, Auth connection/configuration, migration, backup, restore, secret rotation boundary and explicit recovery steps.
- [ ] Static inspection confirms there are no Mindforge domain tables, application endpoints or Supabase services beyond PostgreSQL, Auth and their strictly required support; product SQL object/permission changes are feature-derived migration artifacts, and no SQL function/RPC creates an HTTP/PostgREST endpoint without a separate approved OpenAPI contract.
- [ ] For future separately authorized implementation, the infrastructure-operations tool is a distributable Go binary runnable without Go or Python installed on a third-party machine; VM/database and backup operations require an explicit target and applicable owner authorization, while purely local static audit requires no VM authority. No target, restore or promotion is implicit.

## Validation boundary

- Revision-4 authority covered preparing a plan and locally adapting the existing operator under `src/brain-engine/features/infra/tools/**`, non-secret configuration, scripts, migration files (authoring only), runbook and tests for the `mindforge_main` to existing `mindforge` production/main target correction and an explicit additive path for the separate `mindforge_develop` target. It covered no unrelated tool capability or execution against a VM, database or backup.
- Local validation for this scope is non-runtime/static; SQL runtime validation may occur only on the owner-designated VM/cluster with separate authority.
- Revision-5 coexistence acceptance is documentary until a separately authorized local implementation changes the exact-inventory guard and its tests. Approval of this amendment alone does not prove that the current operator accepts `nsrdb` or authorize VM execution.
- Runtime validation must distinguish PostgreSQL readiness, each Auth/database pair, `vector` availability, persistence, migration execution, backup check, preflight, promotion and restore as separate evidence. Backup check does not prove restore; practical restore proof remains pending outside this revision.
- Migration evidence must be complete per target and link the migration ID/hash, target, authorized operator, preflight, post-application result and feature-required behavior/access verification before promotion can be considered eligible.
- The existing production/main designation of `mindforge` does not authorize production SQL, preflight, backup/dump, restore, service mutation, deployment, Internet/LAN exposure, Coolify, DNS, firewall or real-data validation. Each action requires separate owner authority and owner-provided access.
- Access to or mutation of `mindforge_develop` and `mindforge` each requires separate owner authority; authority for one target does not imply authority for the other. Creating `mindforge_develop` on the existing volume is a distinct additive database operation, not an initdb side effect.
- The prior infra plan revision 5 completed local B01–B04 under contract revision 4. Contract revision 5 coexistence and revision 6 migration policy are not covered by that plan; confirm or regenerate an executable plan against this approved revision before code work. Prior local implementation authority and its exclusions are recorded below; it authorizes no SQL, backup, VM or database action.
- Revision 6 approves one operator-controlled feature-migration route and retirement or adaptation of Brain Core's separate `.psql` ledger path. This contract approval does not authorize Brain Core VM preflight or application.

## Authority required

- The owner authorized local planning and adaptation of the existing operator under `src/brain-engine/features/infra/tools/**`, non-secret configuration, scripts, migration files (authoring only), runbook and tests under revision 4, solely to replace the stale physical target `mindforge_main` with the existing production/main database `mindforge` and provide an explicit additive provisioning path for distinct `mindforge_develop`. No unrelated tool capability was authorized. The frozen plan revision 5 defines the authorized batches B01–B04 recorded in the contract frontmatter. That authorization was limited to local repository changes and non-runtime/static code/test validation; it excluded VM access or actions, SQL execution, backup/dump/restore, production/service/network mutation, deployment, commit and push. `implementation_authorized_at` is the source-registration timestamp `2026-09-23T22:45:12Z`; the exact owner-message time was unavailable.
- Revision-5 coexistence implementation is not covered by the earlier B01–B04 authorization. Confirm or regenerate the plan against this revision and establish local implementation authority before changing the operator. This amendment grants no VM, SQL, service, backup, deployment, commit or push authority.
- Revision-6 migration integration requires a current plan and separate local implementation authority. Approval of this contract and the matching Brain Core contract authorizes no VM access, SQL execution, provisioning, backup, promotion or production mutation.
- Separate explicit owner authorization is required for each target and VM/database or backup operation, including VM access, SQL/mutation, backup/dump, restore, preflight, apply, promotion, LAN exposure, firewall changes, deployment and handling real backup data; authorization for one target or operation never implies another. Agents may execute those tool operations only when authorized; local static audit does not require VM authority.

## Failure and recovery

- Startup failure must leave a diagnosable service state and must not delete durable storage automatically.
- Failure of PostgreSQL, Auth or required initialization must not be reported as a ready stack.
- Auth must fail closed when required database connectivity, secrets, issuer/base URL or redirect configuration is absent or invalid.
- Backup failure must return a failure and must not present a partial artifact as usable.
- Restore must target an explicitly identified, separately owner-approved destination and fail closed on missing or incompatible input.
- The operations tool must fail closed when a VM/database or backup operation, target or applicable owner authorization is missing or ambiguous; it must not infer a target or perform an implicit restore or promotion.
- A backup with incomplete completion, integrity, readability/listing, hash or target-link evidence is unusable for promotion.
- Failure, drift, hash mismatch or inconclusive validation in `mindforge_develop` must stop the workflow before any production/main `mindforge` mutation.
- Missing, failed or stale production authorization, `mindforge` backup or `mindforge` preflight evidence must block promotion.
- Promotion failure must preserve diagnosable evidence and must not fall back to copying/restoring `mindforge_develop` over `mindforge`.
- If `mindforge_develop` is absent from the existing volume, startup must report that target as absent/incomplete; it must not initialize, rename, replace or otherwise mutate the protected `mindforge` database as a fallback.
- Destructive removal of volumes or replacement of an existing database is never an automatic recovery step.
- Recovery documentation must preserve the distinction between restarting a container, recreating a service, restoring data and deleting storage.

## Assumptions

- The target VM provides the designated Docker Compose-compatible runtime and PostgreSQL cluster; exact access and mutation authority remain owner-controlled.
- PostgreSQL and Supabase Auth/GoTrue are the application services required by this initial contract; narrowly required initialization or routing support may be added by the plan with an explicit dependency justification.
- The Supabase PostgreSQL distribution must provide a compatible `vector` extension; `pgvector` is enabled now, while embeddings and retrieval behavior remain future feature work.
- The implementation planner may choose the migration runner and supporting file layout as long as SQL migrations remain versioned, ordered, hash-verifiable and usable without Brain Engine source code while enforcing develop-first promotion.
- The future infrastructure-operations tool is a Go-built distributable binary that can run on a third-party machine without Go or Python installed; database/container client utilities required by particular operations and exact packaging remain implementation details to document in the plan and runbook.
- Restore remains a required guarded capability even though practical restoration of a fresh dump is outside this revision and its proof remains pending.
- Direct LAN access to PostgreSQL is not required by default; the owner can explicitly enable a restricted binding when needed.
- The owner-designated VM currently has a protected database named `mindforge`; the owner identifies it as production/main. The read-only inventory recorded in the infra checkpoint establishes database and catalog names only, not user rows, data contents, retention needs or backup validity.
- The 2026-09-29 read-only inventory found `nsrdb` in the same PostgreSQL cluster. On 2026-09-30 the owner clarified that it belongs to another project and that Mindforge work is limited to its own infrastructure; its contents and technical ownership metadata were not inspected.
- The revision-3 two-target contract is carried forward with physical targets `mindforge_develop` and `mindforge`; the owner approved this topology in revision 4. The inventory does not show `mindforge_develop` on the existing volume, so approved topology does not itself authorize provisioning it there.
- Port numbers, usernames and the exact pinned PostgreSQL version remain implementation configuration provided they are documented and portable; the approved database names are `mindforge_develop` and `mindforge`, with the existing `mindforge` assigned the main/production role.

## Owner decisions

- Start with database infrastructure before Brain Engine and client implementation.
- Do not design product tables before their corresponding features are explored and contracted.
- Keep the infrastructure inside `src/brain-engine/features/infra/` within the root monorepo.
- Prepare migrations now as an operational path, but create domain migrations only from future approved feature work.
- Prioritize reproducible startup, persistence, backup, restore and migration between machines.
- Use a reduced self-hosted Supabase stack with PostgreSQL, Auth/GoTrue and `pgvector` from the first infrastructure delivery.
- Prepare the connection boundaries needed by the future Brain Engine and clients while preserving the Brain Engine as the only authority over Mindforge product data and effects.
- Preserve the existing production/main database `mindforge`, its name, data, durable volume and current main Auth binding; do not rename, replace, delete, move or copy its contents through this contract amendment.
- Retain revision 3's separate develop database as `mindforge_develop`, alongside the existing `mindforge` production/main database. Its absence on the existing volume does not authorize automatic initialization; additive provisioning requires separate explicit authority.
- Preserve two isolated Auth boundaries: existing main Auth stays on `mindforge`; any develop Auth stays only on `mindforge_develop`. Do not copy identities, sessions, secrets or configuration.
- Promote only immutable migration bytes/hash verified in `mindforge_develop` into explicit production/main target `mindforge`, and only after separate production authorization, read-only preflight/recheck and a fresh validated backup; never promote by copying users, data, a database or a volume.
- Put every product SQL object or permission change—tables, functions including RPCs, views, indexes, triggers, RLS/policies, ACL/grants and product roles where applicable—in a versioned migration derived from an approved feature contract; a SQL function/RPC does not automatically become an HTTP/PostgREST endpoint.
- Record target-specific migration evidence linking migration ID/hash, target, authorized operator, preflight, post-application result and feature-required behavior/access verification; incomplete evidence blocks promotion.
- Before every promotion to physical production/main target `mindforge`, generate a new backup/dump outside Git, verify completion, integrity, readability/listing and hash, and link it to that target. This backup check is separate from restore proof.
- Keep the restore capability and its explicit-target/authorization guardrails, while leaving practical fresh-dump restoration testing explicitly pending outside this revision.
- Provide the future infrastructure-operations capability as a Go distributable binary runnable without Go or Python on third-party machines, with explicit operation, target and owner authorization for VM/database and backup actions; agents may execute those actions when authorized, local static audit needs no VM authority, and no target, restore or promotion is implicit.
- Execute migration SQL only in the owner-designated VM/cluster, never in local or disposable PostgreSQL/Docker environments.
- The prior revision-3 decision to host future production separately and treat `mindforge_main` on the current VM as non-production is superseded for the existing database by INFRA-A03; this does not authorize deployment, exposure or production operations.
- Preserve the recorded TDD choice as `yes`; no prior implementation authority carried forward into revision 4. The owner's separate, scoped local authorization is recorded under Authority required.
- Revision-6 decision: use the Go operator as the sole coordinator and the infra ledger in each target as the sole applied-migration history; keep the SQL and schema-specific checks owned by the feature. The prepared Brain Core `.psql` scripts must be adapted or retired before execution, not run as a second workflow.
- Other projects may share the VM and PostgreSQL cluster. `nsrdb` belongs to another project and must coexist without becoming a Mindforge target or being touched by Mindforge operations; only `mindforge` and `mindforge_develop` are Mindforge databases.

## Amendments

### INFRA-A01 — Reduced Supabase foundation

- **Resulting revision:** 2
- **Reason:** the owner confirmed that Auth and vector support are required in the initial infrastructure rather than deferred.
- **Evidence:** owner clarification on 2026-09-05; prior recorded direction for reduced Supabase with PostgreSQL and Auth; current architecture direction for PostgreSQL plus `pgvector`.
- **Changed clauses:** outcome, in/out scope, observable behavior, interfaces, constraints, acceptance criteria, validation, failure handling, assumptions and owner decisions.
- **Preserved clauses:** no Mindforge domain model, no Brain Engine/Hidden Cortex source, no Mindforge HTTP endpoints, no deployment/VM mutation, no production exposure and separate authority for real data or destructive operations.
- **Owner approval:** approved by the owner on 2026-09-07 at 14:51:46-03:00.

### INFRA-A02 — Durable develop-first database promotion

- **Resulting revision:** 3
- **Reason:** the owner replaced disposable migration runtime validation with a durable two-database workflow on the designated VM, separating first execution/verification from promotion while avoiding a second PostgreSQL container.
- **Evidence:** owner decisions recorded on 2026-09-18 for one persistent PostgreSQL cluster with `mindforge_develop` and `mindforge_main`, separate Auth services and immutable develop-first migration promotion.
- **Changed clauses:** outcome, in/out scope, observable behavior, interfaces and data, constraints, acceptance criteria, validation boundary, authority, failure/recovery, assumptions and owner decisions.
- **Preserved clauses:** reduced PostgreSQL/Auth/pgvector stack; Brain Engine authority; no infrastructure-owned product tables; no Mindforge HTTP API; secrets and real backups outside Git; no production deployment or Internet exposure; restore portability; no automatic destructive recovery.
- **Superseded after approval:** contract revision 2 clauses authorizing disposable local PostgreSQL/Docker migration runtime validation, and code plan revision 3 tactics bound to that validation model, become stale and require a new plan before implementation.
- **Owner approval:** approved by the owner on 2026-09-19 at 18:58:04Z.

#### Additional decisions recorded before revision 3 approval

- **Status:** incorporated into revision 3 without incrementing it; the owner approved revision 3 on 2026-09-19 at 18:58:04Z. These records do not themselves establish runtime evidence.
- **SQL change boundary:** every product SQL object or permission change (including tables, functions/RPCs, views, indexes, triggers, RLS/policies, ACL/grants and applicable product roles) must be delivered by a versioned migration derived from an approved feature contract. A SQL function/RPC does not automatically create an HTTP/PostgREST endpoint.
- **Evidence boundary:** each target requires linked migration ID/hash, explicit target, authorized operator, preflight, post-application result and feature-required behavior/access verification; incomplete evidence blocks promotion.
- **Backup/restore boundary at revision 3:** each promotion required a new `mindforge_main` dump/backup outside Git with verified completion, integrity, readability/listing and hash linked to the target. INFRA-A03 replaces that physical target name with `mindforge`; the backup check remains separate from restore proof, and practical restoration testing remains pending.
- **Tool boundary:** the future infrastructure tool is a Go-built distributable binary runnable without Go or Python installed on third-party machines, with backup/dump, restore, preflight, apply, promote and audit capabilities. VM/database and backup operations require explicit operation, target and owner authorization; local static audit needs no VM authority. The tool is not implemented or authorized to access a VM/database in this revision; no implicit target, restore or promotion is allowed.

### INFRA-A03 — Preserve the existing production/main database name

- **Resulting revision:** 4
- **Reason:** the owner clarified that the existing `mindforge` database on the designated VM is production/main and must retain both its name and data; the revision-3 physical target name `mindforge_main` does not describe that existing database.
- **Evidence:** owner clarification on 2026-09-23; the checkpoint's sanitized read-only inventory records the physical databases `mindforge` and `postgres`, existing Auth pointed to `mindforge`, and infrastructure migrations 0001–0004. The inventory did not inspect user rows or data contents and did not authorize mutation.
- **Changed clauses:** outcome, scope, observable behavior, interfaces and data, constraints, acceptance criteria, validation boundary, authority, failure/recovery, assumptions and owner decisions.
- **Preserved clauses:** the persistent cluster, isolated Auth boundaries, approved develop-first flow, immutable migrations, target-specific evidence, backup/restore safeguards, no Mindforge product schema or HTTP endpoints, and separate authorization for every VM/database/production operation.
- **Superseded:** `mindforge_main` as the physical main database name and the decision that future production must be hosted separately from the current VM. Existing `mindforge` is the production/main target; no production operation is authorized by this amendment.
- **Carry-forward decision:** revision 4 retains the previously approved separate `mindforge_develop` target alongside `mindforge` as production/main. The current VM inventory shows that `mindforge_develop` is absent, so provisioning it on the existing volume remains a separately authorized additive operation. No accidental UI interaction is treated as an owner decision.
- **Separate unresolved gate:** the Brain Core migration envelope/ledger versus the infra feature-migration catalog remains undecided and blocks Brain Core VM preflight/application.
- **Plan/code state:** plan revision 4 and its current physical target assumptions are stale against this approved revision. A new plan bound to revision 4 and separate implementation authorization are required before implementation.
- **Owner approval:** approved. `owner_approved_at` records `2026-09-23T22:40:28Z` as the source-registration timestamp because the exact approval-message timestamp was unavailable. This approval grants no implementation, VM, SQL, backup, deployment or production-operation authority.

### INFRA-A04 — Coexist with unrelated project databases

- **Resulting revision:** 5
- **Reason:** the exact cluster-wide database inventory assumption blocks Mindforge operations when another project's database shares the PostgreSQL cluster.
- **Evidence:** the 2026-09-29 read-only inventory found `nsrdb`; on 2026-09-30 the owner confirmed it belongs to another project, must remain untouched, and asked that this boundary be recorded in the authoritative contract.
- **Changed clauses:** outcome, scope, observable behavior, interfaces and data, constraints, acceptance criteria, validation boundary, authority, assumptions and owner decisions.
- **Preserved clauses:** the two explicit Mindforge targets, isolated Auth, develop-first migration and promotion, fail-closed checks for Mindforge state, protected `mindforge` data, separate authorization for VM/SQL/production operations and the unresolved Brain Core migration policy.
- **Superseded:** any requirement that `mindforge` or `mindforge,mindforge_develop` be the complete list of application databases in the PostgreSQL cluster; the two names remain the complete list of Mindforge targets.
- **Owner approval:** approved through the 2026-09-30 owner clarification and explicit request to make this decision effective in the contract. `owner_approved_at` is the source-registration timestamp because the exact message time was unavailable. Approval grants no new implementation or VM authority.

### INFRA-A05 — Single coordinated product-migration history

- **Resulting revision:** 6.
- **Reason:** Brain Core's prepared `.psql` envelope records `0001` in a feature-local ledger, while the Go operator requires a globally ordered catalog, target ledger and develop evidence before promotion. Either standalone path leaves the other unable to establish a complete application state.
- **Evidence:** local Brain Core `0001_brain_core.sql`, `0001_brain_core.preflight.psql` and `0001_brain_core.apply.psql`; infra catalog and Go operator source; no Brain Core SQL runtime evidence.
- **Changed clauses:** scope, observable behavior, interfaces and data, constraints, acceptance criteria, validation boundary, authority and owner decisions.
- **Preserved clauses:** feature ownership of product SQL, two explicit Mindforge targets, shared-cluster coexistence, develop-first verification, separate main authorization and backup, immutable hashes, fail-closed drift handling and all VM/SQL authority boundaries.
- **Decision:** one operator-controlled migration identity/history per target; feature-specific checks remain mandatory, but a standalone feature ledger or `.psql` run cannot establish promotion eligibility.
- **Owner approval:** approved by the owner in the current conversation. `owner_approved_at` is the source-registration timestamp because the exact message time was unavailable. TDD remains `yes`; prior B01–B04 implementation authority does not cover revision-6 work.
