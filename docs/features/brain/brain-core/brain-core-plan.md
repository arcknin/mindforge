---
schema_version: 1
flow: orchestration-flow
feature: brain
task: brain-core
revision: 3
contract_revision: 4
status: ready
tdd: yes
implementation_authorized_at: null
repositories:
  - mindforge
---

# Code Plan: Brain Core r4 migration integration

## Execution summary

This plan preserves the completed B01 and B02 work and replaces only the
remaining B03 migration work from plan revision 2. It is bound to approved
Brain Core contract r4, with TDD `yes` and no implementation authorization.
B03 prepares feature-owned schema checks around the existing canonical
`0001_brain_core.sql` payload and retires the prepared standalone `.psql`
execution and `brain_core.migration_ledger` route.

The cross-feature ownership boundary is explicit: B03 owns only files under
`src/brain-engine/features/brain/`; the infra r6 plan B02 owns the global
catalog descriptor, Go operator, `infra.migration_ledger`, and infra-side
integration tests. Infra B02 depends on `brain-core:B03` and consumes the
feature-owned payload/check paths without copying or editing them. These
disjoint scopes avoid simultaneous file ownership and do not change either
feature contract or plan other than the two authorized plans.

## Completed batches

- **B01 — completed:** deterministic Brain Core behavior and ports, with
  local ATDD-01 through ATDD-16 coverage.
- **B02 — completed:** PostgreSQL adapter and canonical
  `migrations/0001_brain_core.sql` payload with static/unit evidence. Its
  historical implementation is preserved; no claim is made that the payload
  was accepted by PostgreSQL or applied to a database.

## Tasks

```yaml
- id: B03
  goal: "Prepare Brain Core-owned read-only/schema checks for infra's coordinated migration route and retire the standalone psql ledger path."
  repository: mindforge
  depends_on:
    - B02
  allowed_scope:
    - src/brain-engine/features/brain/migrations/0001_brain_core.preflight.psql
    - src/brain-engine/features/brain/migrations/0001_brain_core.apply.psql
    - src/brain-engine/features/brain/migrations/0001_brain_core.preflight.sql
    - src/brain-engine/features/brain/migrations/0001_brain_core.verify.sql
    - src/brain-engine/features/brain/migration_test.go
  changes:
    - "Expected-red: update feature-local static tests to require the read-only preflight and schema verifier, preserve the canonical payload's approved shape, and reject standalone `.psql` envelopes or a `brain_core.migration_ledger`. The current tests still require both independent wrappers and their second ledger."
    - "Keep `0001_brain_core.sql` byte-for-byte as the feature's canonical DDL. Add a feature-owned `0001_brain_core.preflight.sql` that reads only PostgreSQL catalogs and reports whether the expected Brain Core objects are absent/eligible, complete, or partial/divergent. It must not read user-owned Brain or layer values and must not write state."
    - "Add a feature-owned `0001_brain_core.verify.sql` that checks the approved `brain_core.brains` and `brain_core.nodes` schema shape, parent/brain relationship, root/layer constraints, version and required indexes. It must fail the caller if the expected shape is incomplete. Infra's operator is responsible for running this check inside its migration transaction before writing the authoritative infra ledger."
    - "Remove `0001_brain_core.preflight.psql` and `0001_brain_core.apply.psql`, including their psql meta-command flow, payload inclusion, advisory-lock/DDL coordination and product-local ledger. The Go operator in infra r6 is the sole coordinator; this batch creates no replacement executor, CLI, ledger, or direct apply path."
    - "Update only `migration_test.go` to test the Brain Core-owned source and check files, assert the preflight is read-only and checks do not inspect user data, assert the verifier covers the approved schema, and assert both standalone envelopes are absent. Leave the infra catalog descriptor, operator code/tests, `infra.migration_ledger`, Brain Core contract/plan decisions and OpenAPI registry to their owning surfaces."
    - "Relevant-green: rerun the focused Brain Core Go tests and static checks. They demonstrate source/check consistency only; they do not execute SQL or prove PostgreSQL accepts or applies it."
  validation:
    - "cd src/brain-engine && go test -count=1 ./features/brain/..."
    - "cd src/brain-engine && go vet ./features/brain/..."
    - "gofmt -d src/brain-engine/features/brain/*.go"
    - "git diff --check"
  done_when:
    - "The canonical Brain Core payload remains feature-owned and unchanged; its local filename `0001` is not assigned a global migration identity by this plan."
    - "Brain Core supplies read-only eligibility and post-application schema checks for infra plan B02 to register and invoke; no SQL copy or feature-local application ledger exists."
    - "The standalone `.psql` route and `brain_core.migration_ledger` are absent, and feature-local tests cover the new boundaries without opening a database connection."
    - "The infra plan B02 integration remains a separate dependent task; this batch edits no files in the infra operator/catalog/test scope."
  parallelizable: false
```

## Migration ownership and sequence

1. B03 produces the Brain Core-owned check sources and removes the obsolete
   standalone envelopes. It does not add a global ID or invoke the checks.
2. After B03 completes, infra plan r6 B02 registers global ID `0006` against
   these feature-owned sources and the unchanged canonical payload, then wires
   preflight, atomic apply, verification, evidence, and same-hash promotion
   through its Go operator and `infra.migration_ledger`.
3. Only after both local plans have authorized implementations and passed their
   local validation may an owner consider a separately authorized VM operation.
   This plan itself authorizes no preflight, provisioning or SQL execution.

## TDD execution

B03 keeps one functional TDD cycle: first the revised static tests fail against
the current standalone wrappers; then the feature-owned checks replace those
wrappers; finally the same tests and focused Go checks pass. The infra operator
cycle belongs solely to infra plan B02, after this batch. There is no separate
test phase, code phase, review gate or owner decision inside B03.

## Assumptions

- Brain Core contract revision 4 is approved and its canonical migration
  payload from completed B02 remains the approved source of bytes.
- Infra contract revision 6 is approved. Its plan B02 owns global migration
  identity, path/hash registration, target policy, the sole applied-migration
  ledger, operator execution and promotion evidence.
- The existing Brain Core SQL catalog checks can be represented by a read-only
  preflight result and a post-application shape verifier that the operator can
  run under its explicit target and migration transaction.
- `implementation_authorized_at` remains null until separate local authority is
  explicitly recorded; contract approval and plan completion do not authorize
  source changes or operations on a database.

## Evidence boundary

Local Go tests, formatting and static source inspection can verify the feature
payload/check relationships and the absence of the former route. They cannot
prove PostgreSQL syntax, catalog results, transaction rollback, operator
behavior, target ledger contents, runtime, promotion, or deployment. No local or
disposable PostgreSQL/Docker test is planned.

## Residual risks

- The exact-root invariant still depends on the approved Brain Core application
  path and is not proven for privileged direct writes; this batch must not
  invent a trigger or broaden product behavior.
- Static SQL checks do not establish compatibility with the installed
  PostgreSQL version. Any future apply requires the infra operator, an explicit
  target and separate owner authorization.
- The existing designated VM checkpoint has `mindforge_develop` absent. Its
  additive provisioning, read-only preflight, migration application, main
  backup/promotion, restore, service changes and deployment remain separate
  pending operations; none is authorized here.
