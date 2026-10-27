---
schema_version: 1
flow: orchestration-flow
feature: infra
task: infra-revision-6
contract_revision: 6
status: ready
tdd: yes
implementation_authorized_at: null
repositories:
  - mindforge
---

# Code Plan: infra revision 6 migration integration

## Execution summary

This frozen plan is bound to approved infra contract revision 6 and TDD `yes`.
It adds passive coexistence for unrelated databases such as `nsrdb` to the
existing Mindforge-target checks, then connects the approved Brain Core r4
canonical migration and its schema checks to the single Go-operator route and
the `infra.migration_ledger`. Brain Core plan r3 B03 owns the feature-side
preflight/verification SQL and retirement of the standalone `.psql` files;
infra plan B02 consumes those files and owns the catalog/operator/ledger side.
Infra B02 depends on `brain-core:B03`. These scopes are disjoint, the canonical
`0001_brain_core.sql` remains feature-owned, and no SQL is copied into infra. A
final batch updates the infra runbook and static assertions.

All validation is local and static, unit, or fake-backed. No test may start
Docker, PostgreSQL or Auth, execute SQL, contact the VM, handle real backups,
or claim runtime evidence. This plan grants no implementation authority.

## Tasks

```yaml
- id: B01
  goal: "Make unrelated cluster databases passive inventory while keeping every operator operation restricted to explicit Mindforge targets."
  repository: mindforge
  depends_on: []
  allowed_scope:
    - src/brain-engine/features/infra/tools/internal/operations/operations.go
    - src/brain-engine/features/infra/tools/internal/operations/operations_test.go
    - src/brain-engine/features/infra/tools/internal/operations/promote_test.go
    - src/brain-engine/features/infra/tests/test_operations.sh
  changes:
    - "Expected-red: add fake-backed preflight and additive-provisioning cases with the current Mindforge state plus passive `nsrdb`; the existing exact application-database-list guard must fail those cases. Add negative cases proving `nsrdb` cannot be selected as an operation target."
    - "Change read-only inventory validation to inspect only the minimum cluster catalog facts needed for `mindforge` and `mindforge_develop`, their configured Mindforge roles, and the requested operation. Unrelated names may appear in inventory without becoming targets or drift. Do not connect to, inspect contents of, back up, restore, migrate, or change roles/data in an unrelated database."
    - "Keep preflight bound to its explicit database and preserve fail-closed checks for target identity, owner/role, Auth binding, infra ledger, evidence, vector and migration hashes. Exclude unrelated database names from the canonical observed-state digest so their presence alone cannot stale a Mindforge preflight or promotion recheck."
    - "Keep additive provisioning limited to explicit `mindforge_develop`; require the existing `mindforge` and its expected role, require develop to be absent, recheck before the write, and create only the configured develop role/database. Preserve the explicit operator and authorization reference."
    - "Relevant-green: fake runners prove inventories containing `nsrdb` pass when Mindforge target state is eligible, `nsrdb` is never a connection or mutation target, and conflicting Mindforge state still blocks preflight, provisioning and promotion before mutation."
  validation:
    - "go test ./features/infra/tools/internal/operations"
    - "bash src/brain-engine/features/infra/tests/run.sh unit operations"
    - "bash -n src/brain-engine/features/infra/tests/test_operations.sh"
  done_when:
    - "The accepted target set remains exactly `mindforge_develop` and `mindforge`; unrelated names such as `nsrdb` are passive context only."
    - "Preflight and additive provisioning tolerate an unrelated database without querying its contents or changing it, while all inconsistent Mindforge target/role/evidence states fail closed."
    - "Promotion still targets only `mindforge` and preserves its develop evidence, authorization, fresh backup, preflight and recheck gates."
  parallelizable: false

- id: B02
  goal: "Register the approved Brain Core migration by its global infra-catalog identity and execute its feature-owned checks through the sole Go operator and infra ledger."
  repository: mindforge
  depends_on:
    - B01
    - "brain-core:B03"
  allowed_scope:
    - src/brain-engine/features/infra/postgres/feature-migrations/catalog.json
    - src/brain-engine/features/infra/postgres/feature-migrations/README.md
    - src/brain-engine/features/infra/tests/test_migrations.sh
    - src/brain-engine/features/infra/tests/test_operations.sh
    - src/brain-engine/features/infra/tests/versioned-surface.txt
    - src/brain-engine/features/infra/scripts/migrate.sh
    - src/brain-engine/features/infra/scripts/up.sh
    - src/brain-engine/features/infra/scripts/inspect.sh
    - src/brain-engine/features/infra/tools/cmd/mindforge-infra/main.go
    - src/brain-engine/features/infra/tools/cmd/mindforge-infra/main_test.go
    - src/brain-engine/features/infra/tools/internal/audit/audit.go
    - src/brain-engine/features/infra/tools/internal/audit/audit_test.go
    - src/brain-engine/features/infra/tools/internal/migrations/catalog.go
    - src/brain-engine/features/infra/tools/internal/migrations/catalog_test.go
    - src/brain-engine/features/infra/tools/internal/migrations/baseline.go
    - src/brain-engine/features/infra/tools/internal/migrations/baseline_test.go
    - src/brain-engine/features/infra/tools/internal/operations/operations.go
    - src/brain-engine/features/infra/tools/internal/operations/operations_test.go
    - src/brain-engine/features/infra/tools/internal/operations/promote_test.go
  changes:
    - "Expected-red: test that the global catalog registers migration ID `0006` after infra `0001`–`0005`, resolves Brain Core's local `0001_brain_core.sql` as the canonical payload, hashes its exact bytes, and rejects duplicate IDs, missing checks, unsafe paths and symlinks escaping the repository. The current loader only scans infra SQL files and cannot satisfy this mapping."
    - "After the Brain Core plan r3 B03 dependency completes, add a small versioned infra catalog descriptor that maps global ID `0006` to the Brain Core-owned canonical SQL and its feature-owned read-only preflight and schema-verification SQL. Resolve all sources from an explicit repository root and constrain them to registered in-repository feature paths. Do not copy or rename the canonical payload; a local filename is not its global identity."
    - "Adapt the Go operator's feature flow to run the feature preflight against the explicit target, distinguish an absent eligible schema from partial or divergent state, and verify already-applied state against the same global ID/hash. During apply, execute the canonical payload, feature schema verification, and the single `infra.migration_ledger` insert in one transaction. Any failed check must roll back payload and ledger together; a matching completed migration is a no-op."
    - "Retain target-specific attempt/evidence records and make promotion apply the identical catalog-resolved payload bytes and hash only after complete develop evidence, separate production authorization, a fresh validated backup, and successful main preflight/recheck. Do not add restore proof or change backup semantics."
    - "Consume the feature-owned checks produced by Brain Core plan r3 B03. Do not edit Brain Core migration sources or tests; the Brain plan owns the `.psql` retirement and check definitions, while this batch owns their global catalog registration and operator invocation."
    - "Update static audit and the pinned catalog baseline to verify infra `0001`–`0005` plus approved feature entry `0006`, while applying infrastructure-only SQL restrictions only to infra-owned migrations. Update CLI/script source-root handling so the catalog and its canonical feature sources resolve consistently from the repository root."
    - "Relevant-green: fake-backed tests prove absent/clean Brain Core state applies atomically with the infra ledger; identical ID/hash and complete shape is a no-op; partial objects, missing or changed source, hash drift, failed verification, privilege errors and inconsistent ledger state fail without a success record or promotion eligibility."
  validation:
    - "go test ./features/infra/tools/..."
    - "go vet ./features/infra/tools/..."
    - "bash src/brain-engine/features/infra/tests/run.sh unit migrations operations"
    - "bash src/brain-engine/features/infra/tests/run.sh static all"
    - "go build -o <temporary-path-outside-checkout> ./features/infra/tools/cmd/mindforge-infra"
    - "bash -n src/brain-engine/features/infra/scripts/migrate.sh src/brain-engine/features/infra/scripts/up.sh src/brain-engine/features/infra/scripts/inspect.sh src/brain-engine/features/infra/tests/test_migrations.sh src/brain-engine/features/infra/tests/test_operations.sh"
  done_when:
    - "The catalog assigns Brain Core one global identity after infra `0001`–`0005`, hashes the exact feature-owned `0001_brain_core.sql` bytes, and contains no second SQL payload."
    - "`infra.migration_ledger` is the only applied-migration history, and the registered Brain Core source/check paths are consumed without copying or modifying feature-owned files."
    - "Brain Core preflight, atomic application, schema verification, same-hash no-op, failure handling and develop-first promotion preserve the approved r6/r4 acceptance behavior."
    - "No HTTP/OpenAPI operation, product data, VM action, database SQL execution, backup operation, deployment, commit or push is added by local implementation."
  parallelizable: false

- id: B03
  goal: "Document the shared-cluster boundary and the one coordinated feature-migration route in the infra runbook."
  repository: mindforge
  depends_on:
    - B02
  allowed_scope:
    - src/brain-engine/features/infra/README.md
    - src/brain-engine/features/infra/postgres/feature-migrations/README.md
    - src/brain-engine/features/infra/tests/test_runbook_hygiene.sh
    - src/brain-engine/features/infra/tests/test_migrations.sh
    - src/brain-engine/features/infra/tests/versioned-surface.txt
  changes:
    - "Expected-red: add runbook/static assertions for passive coexistence of unrelated project databases, the two explicit Mindforge targets, the global Brain Core catalog entry and sole infra ledger; assertions must fail against instructions implying an exact cluster-wide application database list or a standalone Brain Core migration path."
    - "Document that cluster inventory may show `nsrdb` but Mindforge commands operate only on `mindforge_develop` or `mindforge`; unrelated databases are not inspected beyond the minimal read-only inventory and are never targets. Keep the existing separate-authorization boundaries for both Mindforge targets and VM/database/backup operations."
    - "Document the feature-owned canonical SQL plus catalog global ID/hash/check mapping, the operator-controlled develop-first apply/promotion route and the one authoritative ledger. State that the local plan does not prove VM state, SQL application, backup/restore, or runtime behavior. Preserve the no-endpoint and no-product-infra-scope boundary."
    - "Relevant-green: runbook and catalog hygiene checks reject stale exact-inventory or second-ledger guidance and accept the documented passive `nsrdb` and canonical Brain Core integration."
  validation:
    - "bash src/brain-engine/features/infra/tests/run.sh static runbook-hygiene"
    - "bash src/brain-engine/features/infra/tests/run.sh static migrations"
    - "bash -n src/brain-engine/features/infra/tests/test_runbook_hygiene.sh src/brain-engine/features/infra/tests/test_migrations.sh"
    - "git diff --check"
  done_when:
    - "The runbook explains `nsrdb` as passive shared-cluster context and restricts Mindforge operations to the two explicit Mindforge targets."
    - "The runbook and static assertions describe the canonical feature source, one global migration identity, one operator route and one infra ledger without claiming VM/runtime proof."
    - "All planned work remains local; the OpenAPI registry stays empty and no VM, PostgreSQL, Docker, backup, restore, deployment, commit or push operation is performed."
  parallelizable: false
```

## Validation boundary

Planned reds/greens are Go tests, shell unit/static checks, source/hash/path
inspection, build, Bash syntax and `git diff --check`. They do not execute SQL
or start Docker, PostgreSQL or Auth. SQL files are inspected and hashed as source
only. No VM or network operation, backup handling, database provisioning,
application, promotion, restore, service change, deployment, commit or push is
part of this plan.

## Assumptions

- Infra contract revision 6 and Brain Core contract revision 4 are approved; the
  Brain Core contract is an integration dependency, while its contract, plan and
  canonical SQL remain outside this plan's mutation scope.
- The approved Brain Core schema checks can be represented as feature-owned SQL
  checks that the Go operator runs at the existing target-specific preflight and
  within the atomic migration transaction.
- Existing operator target/auth/backup/evidence guards remain the authority for
  all operations; the catalog descriptor only identifies canonical bytes and
  checks and does not create new operations or targets.
- Local implementation requires a separate explicit owner authorization after
  this plan. The `implementation_authorized_at` value therefore remains null.

## TDD execution

Each batch owns its expected-red tests, causal implementation, relevant-green
reruns and focused accumulated checks. B01 proves passive unrelated-database
coexistence without weakening Mindforge target checks. B02 proves canonical
source/hash binding, feature checks and the single atomic operator/ledger route
with fake runners. B03 proves runbook accuracy with static assertions. No test
or batch is a separate approval, VM, runtime, review or owner-decision gate.

## Residual risks

- Static and fake-backed evidence cannot prove that `nsrdb` remains untouched on
  the live shared cluster, that either Mindforge target is currently ready, or
  that feature SQL works against the installed PostgreSQL version.
- Local completion cannot prove the Brain Core schema was applied, the target
  ledger/evidence is present, migration promotion is eligible, or a fresh
  backup can be restored. Practical restore proof remains pending under infra
  revision 6.
- Additive `mindforge_develop` provisioning, VM access, any PostgreSQL
  preflight/application, production/main backup and promotion, restore, service
  mutation, network exposure and deployment all remain separate owner-authorized
  VM operations. No such operation is authorized by this plan.
- Code implementation of these new r6 batches also remains pending separate
  local implementation authorization; prior revision-4 authorization does not
  carry forward.
