# Task History: brain-core

<!-- managed_by: orchestration-flow -->

## 2026-09-25 — Continuation reconciled against infra revision 4

- Outcome: local Brain Core plan completion remains valid, but its old direct-to-preflight resume point is superseded by the explicit migration-policy gate in infra revision 4 / INFRA-A03.
- Evidence: read-only reconciliation of Brain Core checkpoint/plan and infra contract revision 4, plan revision 5 and checkpoint; two bounded evidence scouts gathered Brain Core and infra/Git state.
- Current direction: decide the separate Brain Core psql envelope/ledger versus infra feature catalog before any Brain Core VM preflight/application. Infra local B01-B04 completion does not prove develop provisioning or runtime.
- Limits: no tests, database, SSH, VM, SQL, commit or push executed in this continuation; the 2026-09-23 VM inventory is historical evidence. Existing worktree changes were preserved. Stack overview/readmes and older infra checkpoint bullets contain stale revision/target references.
- Resume: owner migration-policy decision, then authorized artifact reconciliation; VM operations remain separately authorized per target and operation. Only orchestration checkpoint/history were updated, manually.

## 2026-09-18 — Foundation published and SQL plan revision 2 frozen

- Outcome: the existing Brain Core foundation was committed and pushed to `origin/feat/brain-core-foundation` at `0204ba4218bf963b8d2497e6ddb04b4a93803a95`.
- Planning: plan revision 2 preserves completed B01/B02 and adds only B03 for a local transactional, ledgered and fail-closed SQL/psql migration envelope.
- Boundary: B03 may edit only the listed Brain Core migration and test files and must not access PostgreSQL, Docker, SSH or the VM. Applying the migration remains a later owner gate.
- Evidence: unit/race, vet, infra static and `git diff --check` passed during planning without a database target.
- Resume point: assign B03 to a `$code-implement` executor, validate the local package, then stop before any VM preflight or apply.

## 2026-09-18 — B03 local SQL package complete

- Outcome: `0001_brain_core.preflight.psql` and `0001_brain_core.apply.psql` now wrap the unchanged DDL payload with read-only catalog classification, a product-local ledger, SHA-256 binding, transaction and advisory-lock barriers, and fail-closed recovery behavior.
- TDD: causal static reds were observed before the psql artifacts and disposable-harness removal; Brain Core unit/race, vet, gofmt, forbidden-pattern checks, infra static and `git diff --check` then passed.
- Hash: payload SHA-256 `1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9` matches both envelopes.
- Boundary: no PostgreSQL, Docker, network, SSH, VM or SQL execution occurred. Local evidence does not prove psql syntax, catalog shape, advisory locking, rollback or an installed schema.
- Resume point: stop before VM access. The next gate is an owner-authorized read-only preflight on the designated VM; applying `0001` remains a separate explicit authorization after an `absent_eligible` result.

Compact closed-work entries managed by `orchestration-flow`. This file preserves outcomes, evidence, limitations, and residual risk; it does not preserve agent conversation or transient reasoning.

## 2026-09-17 — Contract revision 2 approved

- Outcome: owner decisions were formalized; 16 behavioral ATDD scenarios received an independent PASS; contract revision 2 was approved with `tdd: yes` and conditional local implementation authority.
- Affected paths: `docs/features/brain/brain-core/brain-core-contract.md`.
- Evidence: approved contract frontmatter, amendment ledger, independent ATDD conformity PASS, and clean `git diff --check`.
- Limitations: documentary/static validation only; no executable test, source implementation, database, HTTP, runtime, migration, deployment or publication evidence.
- Residual risk: technical persistence, API, authorization mechanism, conflict representation and implementation tactics remain for the frozen code plan.

## 2026-09-17 — Code plan revision 1 frozen

- Outcome: a complete sequential implementation plan was frozen with B01 for the deterministic Go core and B02 for PostgreSQL persistence and disposable integration.
- Affected paths: `docs/features/brain/brain-core/brain-core-plan.md`.
- Evidence: contract revision 2, `tdd: yes`, matching implementation authority, explicit allowlists, ATDD-01 through ATDD-16 coverage, and clean `git diff --check`.
- Limitations: no source, executable test, database, migration, Docker, HTTP or runtime work was performed during planning.
- Residual risk: authentication integration, HTTP contracts and persistent-environment migration execution remain outside the approved plan.

## 2026-09-17 — B01 deterministic Brain Core complete

- Outcome: B01 completed one test-first cycle from an expected compile failure to a deterministic Go Engine, injectable ports, opaque typed errors, stable UUID identities and test-only concurrent Store covering ATDD-01 through ATDD-16.
- Affected paths: `src/brain-engine/go.mod` and `src/brain-engine/features/brain/{model.go,errors.go,ports.go,ids.go,engine.go,engine_test.go,memory_store_test.go}`.
- Evidence: the initial `go test ./features/brain/...` failed on absent Brain Core types; orchestrator verification passed focused unit tests, race tests, `go vet`, `gofmt -d`, and the forbidden HTTP/PostgreSQL import check.
- Limitations: local/unit evidence only; the test Store is not production persistence and no PostgreSQL, authentication integration, HTTP, runtime, migration, deployment or publication was proven.
- Residual risk: B02 still must implement the isolated product migration and PostgreSQL adapter and may add integration evidence only against an explicitly local, exclusive and disposable target.

## 2026-09-17 — B02 PostgreSQL persistence surface complete

- Outcome: B02 completed one test-first cycle from missing `PostgresStore` symbols to an isolated `brain_core` product migration and pgx adapter implementing owner-scoped bootstrap, observation, layer insertion and rename CAS without HTTP or automatic migration execution.
- Affected paths: `src/brain-engine/go.mod`, `src/brain-engine/go.sum` and `src/brain-engine/features/brain/{migrations/0001_brain_core.sql,postgres_store.go,migration_test.go,postgres_store_test.go}`.
- Evidence: accumulated Brain Core unit tests, race tests and `go vet ./...` passed; migration shape/infra-ledger checks, infra static/unit regressions, formatting, empty OpenAPI registry and `git diff --check` passed.
- Limitations: `TestPostgresStoreIntegrationMatrix` skipped because no loopback `BRAIN_CORE_DATABASE_URL` explicitly marked with `BRAIN_CORE_DATABASE_DISPOSABLE=1` was available; no PostgreSQL schema was applied and no runtime, persistent database, VM, production, deploy or publication is proven.
- Residual risk: PostgreSQL transaction and constraint behavior still requires a separately authorized disposable integration run before it can be claimed as integration-verified; authentication and transport contracts remain outside this task.

## 2026-09-30 — Revision 4 migration check adaptation complete locally

- Outcome: owner-authorized Brain Core B03 replaced standalone `.psql` preflight/apply and its product-local ledger with feature-owned read-only preflight and schema verification SQL for the infra r6 operator. The canonical `0001_brain_core.sql` payload remained byte-for-byte unchanged.
- Affected paths: Brain Core migration check SQL and `migration_test.go`; no Brain Core endpoint or domain behavior changed.
- Evidence: expected-red for missing preflight SQL, then focused Go tests, `go vet`, formatting and `git diff --check` passed; canonical payload SHA-256 remained `1c8e0e960f728662062bd73bf791bce7b6cee092cff099375fe4455660f0f7b9`.
- Limits: checks were inspected statically, not executed in PostgreSQL; no VM, SQL execution, Docker, commit, push or deployment occurred.
- Exact continuation: infra r6 B02 consumed these checks in its local operator/catalog integration. PostgreSQL behavior and develop-first application remain separate authorized runtime work.
