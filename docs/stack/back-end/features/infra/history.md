# Feature History

<!-- managed_by: orchestration-flow -->

Compact closed-work entries managed by `orchestration-flow`. This file preserves outcomes, evidence, limitations, and residual risk; it does not preserve agent conversation or transient reasoning.

## 2026-09-19 — Infra revision 3 local code plan completed through B06

- Outcome: B06 removed the three obsolete disposable execution/test paths and aligned configuration, shell adapters, source-level checks and runbooks with the durable two-target VM-only workflow. B01-B06 of plan revision 4 are locally complete.
- Evidence: B06 expected-red static suite failed on 30 causal assertions before implementation. Executor and root passed static all, unit all, Go tests/vet/build, Bash syntax and `git diff --check`; executor also passed render-only `docker-compose config`. Root confirmed OpenAPI registry remains empty and Git HEAD remains `0204ba4` with no new commit or push.
- Limitation: evidence is local/static/fake-backed. No current two-database/Auth VM runtime, SQL migration, real backup, restore, promotion, deployment, commit or push was executed. Contract body and plan frontmatter retain stale authorization prose.
- Resume point: separate owner authority for target-specific VM preflight/runtime and resolution of protected legacy database disposition; synchronize documentary authority without treating local code completion as runtime proof.

## 2026-09-19 — Infra revision 3 B05 local completion

- Outcome: B05 added immutable develop-first promotion to `mindforge_main` with verified develop evidence, main preflight/recheck, a fresh validated main backup, identical-hash apply, idempotent replay and sanitized target evidence.
- Evidence: causal Go expected-red preceded implementation; executor passed Go tests/vet/build, unit migrations/operations and `git diff --check`. Root repeated those checks with exit 0.
- Limitation: fake-backed/static proof only; no real backup, PostgreSQL catalog, SQL application, VM, deployment, commit or push.
- Resume point: execute authorized B06 under plan revision 4.

## 2026-09-19 — Infra revision 3 B04 local completion

- Outcome: B04 added guarded external backup artifacts, validated manifests and explicit-target restore with replacement authority; shell adapters remain narrow.
- Evidence: causal Go expected-red preceded implementation; executor passed Go tests/vet/build, unit operations, Bash syntax and `git diff --check`. Root repeated those checks; build initially hit sandbox Go cache permissions, then passed with `GOCACHE` under `/tmp`.
- Limitation: fake-backed/static proof only; no real dump, archive validation, restoration, VM, PostgreSQL, SQL execution, deployment, commit or push.
- Resume point: execute authorized B05, then B06 serially under plan revision 4.

## 2026-09-19 — Infra revision 3 B03 local completion

- Outcome: B03 added explicit-target preflight and application logic, target evidence, and thin startup, migration and inspection adapters within its plan scope.
- Evidence: causal Go expected-red preceded implementation; executor passed Go tests, vet, build, unit migrations and operations, Bash syntax and `git diff --check`. Root repeated focused Go tests/vet, Bash unit migrations/operations, Bash syntax and diff check with exit 0.
- Limitation: only local/static/fake-backed proof; no VM, PostgreSQL, SQL execution, Docker runtime, credentials, real backup, deployment, commit or push. B04 backup/restore behavior and tests remain pending.
- Resume point: execute authorized B04, then B05 and B06 serially under plan revision 4.

## 2026-09-19 — Scoped B01-B02 run completed

- Outcome: B02 added the local Go `mindforge-infra` foundation (`config validate`, `audit`), target/authority model, immutable migration catalog checks, an empty feature-migration location and infrastructure-owned `0005` target evidence SQL. B01 and B02 are complete within the owner's two-batch authorization.
- Evidence: B02 expected-red failed on absent APIs; Go tests, vet, build, local audit, Bash unit migrations, static runbook-hygiene, syntax and diff checks passed. Root repeated the focused Go and Bash checks and `git diff --check` with exit 0.
- Limitation: all proof is local/static/fake-backed. `0005` was not applied; no Docker daemon, PostgreSQL, Auth, VM, SQL runtime, backup, deployment, commit or push was performed. The legacy migration/startup script assertions await B03 replacement.
- Resume point: stop at B02. B03-B06 require separate implementation authority, and each VM/database/backup action requires its own target-specific authority.

## 2026-09-19 — Infra revision 3 B01 local completion

- Outcome: one-cluster/two-database Compose and two isolated Auth service definitions, strict target-aware private configuration, and a fresh-cluster bootstrap script were implemented within B01 scope.
- Evidence: causal static-config red preceded implementation; static config/compose, Bash syntax, render-only `docker-compose config`, and `git diff --check` passed. Root repeated the static config/compose checks and diff check.
- Limitation: no Docker daemon, PostgreSQL, Auth, SQL, VM or installed-cluster topology was exercised; the protected legacy volume was not altered.
- Resume point: execute authorized B02 with a distinct leaf; B03-B06 and VM/database/backup work remain separately gated.

## 2026-09-19 — Infra revision 3 code plan frozen

- Outcome: `plan.md` revision 4 binds to approved contract revision 3, preserves `tdd: yes` and `implementation_authorized_at: null`, and defines serial B01-B06 for topology, Go tool, target operations, backup/restore, promotion, and runbook/static validation.
- Evidence: structural plan checks and `git diff --check` passed. No source implementation, Docker, PostgreSQL, SQL, VM, backup or deployment was executed.
- Limitation: the existing protected single-database VM state is not reconciled by planning; the plan fails closed on extra legacy application databases pending an owner-controlled disposition.
- Resume point: obtain matching authorization for local implementation before B01. VM/database/backup actions require separate target-specific authority.

## 2026-09-19 — Owner approved infrastructure contract revision 3

- Outcome: the owner explicitly approved revision 3; `contract.md` now records `status: approved` and the approval registration timestamp. The existing `tdd: yes` choice remains in force.
- Authority: approval authorizes planning in this turn. Implementation, VM access, SQL, backup handling, deployment, commit and push remain separate.
- Evidence: contract frontmatter and `git diff --check` passed; no runtime action occurred.
- Resume point: replace the revision-2-bound plan with a frozen code plan for revision 3, including the two-database topology and guarded `infra/tools` binary.

## 2026-09-19 — Revision 3 SQL and operations-tool decisions formalized

- Outcome: pending contract revision 3 now covers feature-approved versioned SQL for product objects and permissions, per-target migration evidence, a fresh checked `mindforge_main` dump before promotion, and a future Go binary under `infra/tools/` for explicit backup/dump, restore, preflight, apply, promote and audit actions.
- Boundary: authorized agents may run VM/database and backup operations only for the specifically approved action and target; local static audit needs no VM authority. Restore may recover an environment from its own backup, but a practical fresh-dump restore test is deferred and no restore proof is claimed.
- Evidence: owner clarifications in the current conversation were formalized in `contract.md`; no tool, VM, database or SQL operation was executed.
- Resume point: obtain explicit owner approval of contract revision 3, then route a separate new code plan. Existing revision-2 implementation and disposable-validation tactics remain stale.

## 2026-09-18 — Contract revision 3 formalized for owner approval

- Outcome: contract revision 3 is `ready_for_owner_approval` and formalizes one persistent PostgreSQL cluster with separate `mindforge_develop` and `mindforge_main` databases, isolated GoTrue/Auth per database and immutable develop-first migration promotion.
- Authority: this interaction authorized only the contract amendment and orchestration-state reconciliation; revision 3 implementation, VM access, SQL execution, database mutation, commit and push remain unauthorized.
- Evidence: `INFRA-A02` records the owner decisions; `tdd: yes` was preserved from the active checkpoint; contract-only diff inspection and `git diff --check` passed.
- Limitation: the current plan revision 3 and implementation still describe one application database and disposable validation, so they become stale after approval and require a new code plan.
- Resume point: owner approves revision 3 or requests exact changes; after approval, route a separate `$code-plan` interaction before any implementation or Brain Core SQL execution.

## 2026-09-18 — Owner migration-execution boundary changed

- Decision: SQLs and migrations continue to be authored and versioned locally, but execution and validation must occur only on infrastructure or a VM explicitly designated by the owner.
- Prohibition: agents must not suggest, create or use temporary/disposable Docker, PostgreSQL or databases as substitute migration targets.
- Security: endpoint, username, port and credentials remain outside Git in local ignored configuration; absence of access is a blocker, not authorization for a local fallback.
- Contract impact: infra contract revision 2, plan revision 3 and disposable runners are now stale for future migration execution. Their prior evidence remains historical; changing them requires a separate contract amendment.
- Resume point: amend the infra contract before executing Brain Core SQL or changing the validation runners. No SQL, SSH, VM, database or runtime action occurred in this registration.

## 2026-09-12 — Local structural reorganization

- Outcome: the stack documentation moved from `stack/` to `docs/stack/`; the real Brain Engine project moved under `src/`; and the complete operational infrastructure surface moved to `src/brain-engine/features/infra/`.
- Documentation: the active contract, plan, checkpoint and backend stack references now resolve under `docs/stack/`; the infra feature documents remain alongside this history at `docs/stack/back-end/features/infra/`.
- Preservation: all 28 infra files and their executable modes were preserved by directory moves. `.DS_Store`, `mf-backup-test`, `mf-accept-opstest1234-restore` and other checkout residues were left untouched.
- Validation boundary: this was a local structural/documentation change only; no Docker runtime, SSH, VM, remote, deployment, database or endpoint work was performed. `contracts/openapi/registry.yaml` remains `endpoints: []`.
- Historical-reference rule: entries below retain material outcomes and evidence boundaries from their original runs; machine-specific paths, addresses, SSH usernames, ports and credential-store names are redacted.

## 2026-09-07 — Contract approval, post-reboot observation and plan review

- Outcome: contract revision 2 approved; plan revision 2 frozen; observational post-reboot validation passed; consolidated planning review returned `CHANGES_REQUIRED`.
- Scope: `stack/back-end/features/infra/{contract.md,plan.md,checkpoint.yaml,history.md}` and read-only observation of the existing stack on the identified private test VM.
- Acceptance evidence: PostgreSQL/Auth healthy after host reboot, Auth `/health` HTTP 200, PostgreSQL 16.15, `vector` 0.8.6, `uuid-ossp` 1.1, `citext` 1.6 and loopback-only bindings.
- Limitations: no signup/signin/token, vector rollback probe, controlled persistence marker, backup, isolated restore or migration replay was executed; no production proof; at this public-repo checkpoint the root Git exists on `main` with no commits before this checkpoint, and no upstream or publication proof exists.
- Residual risk: Auth configuration coverage, restore/replay isolation, deterministic migration execution, backup destination/exclusion, SCM evidence and restore documentation require owner-authorized correction before functional validation.

## 2026-09-08 — TDD correction and disposable runtime acceptance

- Outcome: plan revision 3 executed with `tdd: yes`; B01-B05 completed; integrated validation findings `MIG-LEDGER-001` and `BACKUP-ARCHIVE-001` closed.
- Scope: `brain-engine/infra/`, `stack/readme.md`, `stack/back-end/readme.md` and root-owned orchestration artifacts; no Mindforge product endpoint or domain model added.
- Acceptance evidence: final static/unit/syntax exit 0; local disposable run `190c679c1a48afb4` and VM disposable run `e8a64a60d2a2d1a6` passed Auth signup/password grant, vector rollback, persistence, external backup, isolated restore, deterministic replay and absence of product domain objects.
- Final VM observation: zero disposable containers active, three final-run volumes preserved and the original observed stack remained healthy by read-only `inspect.sh`.
- Limitations: corrected artifacts were not deployed over the original VM checkout; no production, network, real-data or Git publication proof; at this public-repo checkpoint the root Git exists on `main` with no commits before this checkpoint.
- Residual risk: 45 local disposable volumes, 11 VM disposable volumes and 5 owned VM temporary packages remain preserved pending explicit destructive-cleanup authority.
- Owner-authorized cleanup later removed exactly those 45 local volumes, 11 VM volumes and 5 exact temporary package directories. Post-cleanup checks found no `mf-accept` resources; the original `infra_mindforge-postgres-data`, `mindforge-postgres` and `mindforge-auth` remained present and healthy, and the original VM `inspect.sh` exited 0.

## 2026-09-08 — Original VM deployment authorized and blocked at preflight

- Outcome: owner authorized deploying the corrected `brain-engine/infra/` package over the original VM checkout and recreating PostgreSQL/Auth services while preserving the existing volume and data.
- Local evidence: the 28-file versioned surface is complete; static, unit and Bash syntax validation passed; no `.env`, backup or runtime data is part of the transfer surface.
- Remote preflight: a configured SSH alias resolved to a redacted historical target and timed out; the redacted historical target rejected the available SSH keys. No remote mutation was attempted.
- Blocker: the checkout, Compose project, protected volume, free space and current service state cannot be verified. Historical evidence reports `vector` 0.8.6, while the corrected package pins and validates 0.8.0, so service recreation cannot proceed safely without compatibility confirmation.
- Exact resume point: restore or provide working SSH access, perform the read-only preflight, preserve the remote `.env`, create and validate an external backup, transfer only the versioned surface, then reconcile services under the confirmed original Compose project without `down -v`, volume deletion, prune or restore over the source volume.
- Owner clarification: the identified private test VM uses password-authenticated SSH. The credential remains outside the project; the plaintext password is intentionally excluded from workflow artifacts.
- Authenticated preflight then confirmed the infrastructure checkout on the identified private test VM, Compose project `infra`, protected volume `infra_mindforge-postgres-data`, healthy PostgreSQL 16.15/Auth, and installed/available `vector` 0.8.6. The current image is `pgvector/pgvector@sha256:ccc6e83d6e35e931dc7c5def2022729d5a6c370318d099181995567ff1fb4d6b` on `linux/amd64`.
- Compatibility gate: the corrected local package pins and expects `vector` 0.8.0. It must not be deployed over the 0.8.6 volume until the owner authorizes pinning the exact current image digest and aligning inspect/tests/documentation to 0.8.6; no remote mutation occurred.

## 2026-09-08 — Vector compatibility correction and original VM deployment

- Outcome: owner authorized `VECTOR-COMPAT-001`; the package now accepts `vector >=0.8.0,<1.0.0`, pins the exact healthy VM image digest for `linux/amd64`, and preserves `CREATE EXTENSION IF NOT EXISTS` without downgrade or `ALTER EXTENSION`.
- Validation: focused TDD, static/unit/Bash/Compose checks and local disposable runs passed; integrated review returned PASS with no P0-P3 findings.
- Deployment safety: external dump `mindforge-infra-20260909T010436Z-validated.dump` passed `pg_restore --list`; snapshot `mindforge-infra-20260909T010501Z-predeploy` preserved the prior checkout and `.env`.
- `VOLUME-IDENTITY-001`: the first deploy attached a new `infra_postgres-data` volume. The protected original `infra_mindforge-postgres-data` remained intact; TDD restored the logical volume name, disposable run `1cfd5b80c54c693f` passed, and targeted review returned PASS.
- Final runtime: the corrected 28-file surface is deployed on the identified private test VM; `up.sh .env infra` and `inspect.sh` passed; PostgreSQL/Auth are healthy, PostgreSQL mounts `infra_mindforge-postgres-data`, `vector` is 0.8.6, Auth `/health` is 200, migrations 0001-0004 have no drift/orphan, domain tables are absent, and host bindings remain loopback-only.
- Residuals: unreferenced `infra_postgres-data` and retained disposable local volumes remain preserved pending separate destructive-cleanup authority. This does not block the completed infra delivery; production, public networking, real-data guarantees and Git publication remain outside the evidence boundary.
- Scoped cleanup later removed exactly the unreferenced regression volume `infra_postgres-data` after verifying it was distinct from `infra_mindforge-postgres-data` and had zero references. The protected volume remained mounted read-write with one reference, DB/Auth remained healthy, and `inspect.sh` exited 0. Retained disposable local volumes remain outside this cleanup.

## 2026-09-23 — Revision 4 main/develop target adaptation, local completion

- Outcome: owner approved contract revision 4 with the existing physical `mindforge` database as production/main and separate `mindforge_develop`; plan revision 5 B01–B04 implemented locally with TDD. Existing `mindforge` name, main Auth service identity and project-scoped protected volume remain the required boundary. Develop provisioning is an explicit additive operator action, never startup behavior.
- Scope: infra contract, plan, checkpoint, history, configuration, Compose/bootstrap, Go operator and pending infrastructure migration `0005`, lifecycle scripts, runbook and their local tests. Brain Core migration policy and source remain unchanged.
- Acceptance evidence: expected-red/green cycles for B01–B04; final `static all`, `unit all`, Go infra tool tests/vet, Bash syntax across 17 scripts and `git diff --check` passed. Compose was rendered only; active infra source has no physical `mindforge_main` target.
- Limitations: no new code or SQL was executed on the designated VM; no database creation, Auth split, real backup, restore, promotion, deployment, commit or push occurred. The read-only inventory still showed `mindforge` and `postgres` only. Local static/fake-backed evidence does not prove runtime.
- Exact continuation: obtain separate owner authority for any VM provisioning/runtime operation. Decide Brain Core's feature-local `.psql` envelope/ledger versus the infra feature-migration catalog before Brain Core VM preflight or application.
## 2026-09-28 — VM inventory before requested develop provisioning

- Outcome: SSH access to the owner-designated VM succeeded for read-only inspection. `infra-db-1` and `infra-auth-1` were healthy; `infra-db-1` used Compose project `infra` and mounted `infra_mindforge-postgres-data`.
- Evidence: a read-only query of `pg_database` in `infra-db-1` returned `mindforge`, `nsrdb` and `postgres`; `mindforge_develop` was absent. The session was closed without mutation.
- Blocker: the approved additive `provision-develop` operation requires exactly `mindforge` among existing application databases. `nsrdb` makes the preflight fail closed. Its origin and intended coexistence need owner clarification before changing topology or operator behavior.
- Limits: no user data, SQL mutation, backup, service change, migration, commit or push; no claim about the other Coolify or project containers beyond the observed Docker listing.

## 2026-09-30 — Revision 6 local correction complete

- Outcome: with explicit owner authorization for local code/tests, infra r6 B01–B03 and dependent Brain Core r4 B03 completed with TDD. Shared-cluster inventory tolerates passive `nsrdb`; the global feature catalog registers Brain Core as `0006` from its unchanged canonical SQL; the Go operator uses feature checks and the infra ledger as the sole migration route. The runbook reflects these boundaries.
- Affected paths: infra operator/catalog/scripts/tests/runbook and Brain Core feature-owned migration checks. The standalone Brain Core `.psql` ledger route was removed. Existing dirty work was preserved.
- Evidence: expected-red/green cycles; infra Go tests/vet/build, shell unit/static, Bash syntax, local audit and `git diff --check`; Brain Core focused Go tests/vet and unchanged canonical payload SHA-256 `1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9`.
- Limits: all evidence is local/static/fake-backed. No VM, PostgreSQL, SQL execution, Docker, real backup, restore, commit, push or deployment occurred; no endpoint was affected. The planned `static migrations` selector was unavailable, so the underlying `unit migrations` check ran and passed.
- Exact continuation: obtain separate authority for identified-VM read-only preflight and each target/operation. `mindforge_develop` was absent in the last recorded VM inventory; additive provisioning, Auth readiness, migration application in develop, backup and any main promotion remain unverified and separately authorized.

## 2026-09-30 — Authorized VM read-only preflight blocked before login

- Outcome: owner authorized a read-only VM preflight. Network access and strict host-key verification succeeded, but SSH key and one non-disclosing password attempt were rejected. Further login attempts stopped to avoid lockout.
- Evidence limit: no authenticated session or remote command occurred; containers, volume, database names, roles, Auth, extension and ledger were not rechecked. Historical VM inventory is not current runtime proof.
- Exact continuation: obtain a working SSH authentication method through a secure channel, then repeat only the authorized read-only inventory. VM mutation, SQL application, provisioning, backup, deployment and service changes remain separately authorized.

## 2026-09-30 — Authorized VM read-only preflight completed

- Outcome: a bounded password-authenticated SSH method succeeded with strict host-key verification. Read-only inspection found healthy `infra-db-1` and `infra-auth-1` in Compose project `infra`; the database container mounts protected volume `infra_mindforge-postgres-data`, and main Auth points to `mindforge`.
- Database evidence: `pg_database` lists `mindforge`, `nsrdb` and `postgres`; `mindforge_develop` is absent. The `mindforge_develop` role and develop Auth container are absent. PostgreSQL is 16.15, `vector` is 0.8.6, and the `mindforge` infra ledger has IDs `0001`–`0004` with hashes matching the local baseline. Infra evidence tables from `0005` and schema `brain_core` are absent.
- Limits: `nsrdb` appeared only as a name in cluster inventory; no connection to it, user-data read, SQL write, file transfer, backup, deployment, service change or VM mutation occurred. Auth login behavior, remote source/operator revision, migration 0005/0006 runtime and backup/restore remain unverified. A login attempt as PostgreSQL role `postgres` was denied; catalog queries used the configured `mindforge` role.
- Exact continuation: prepare and separately authorize delivery of the corrected operator/source and additive develop provisioning. Applying migrations to develop and any main promotion require their own target-specific SQL/production authority; preserve `mindforge`, `nsrdb` and the protected volume.
