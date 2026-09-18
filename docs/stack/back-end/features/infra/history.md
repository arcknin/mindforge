# Feature History

<!-- managed_by: orchestration-flow -->

Compact closed-work entries managed by `orchestration-flow`. This file preserves outcomes, evidence, limitations, and residual risk; it does not preserve agent conversation or transient reasoning.

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
