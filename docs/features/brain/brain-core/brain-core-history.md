# Task History: brain-core

<!-- managed_by: orchestration-flow -->

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
