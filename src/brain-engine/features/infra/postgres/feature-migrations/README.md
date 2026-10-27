# Feature migration catalog

`catalog.json` assigns approved feature migrations their global infra-catalog
identity after infra-owned IDs `0001` through `0005`. Entry `0006` references Brain Core's canonical feature-owned
`0001_brain_core.sql` plus its read-only preflight and schema-verification checks. The descriptor pins the exact SHA-256 of all three files; the payload is not copied into this directory.

The Go operator resolves every path from the explicit repository root, checks
the registered feature boundary and hashes, runs preflight against the selected
Mindforge target, then applies the canonical payload, verification check, and
one `infra.migration_ledger` insert in a single transaction. This is the only
applied-migration ledger. A matching global ID and hash is a no-op only when
preflight confirms the complete schema. Feature migration application is
develop-only; promotion to `mindforge` uses the same catalog-resolved bytes
after develop evidence, separate production authorization, a validated fresh
backup, and successful main preflight/recheck.
Infrastructure-only SQL audit rules apply to infra-owned migrations; feature
SQL remains governed by its approved feature contract and pinned catalog entry.

Local static and fake-backed validation does not prove VM state, SQL application, backup or restore behavior, or runtime behavior.

Entry `0007` references Documents' canonical
`src/brain-engine/features/documents/migrations/0001_documents.sql` plus its
read-only preflight and verification checks. The approved backend contract is
`docs/features/brain/layer-content/spec-02-backend.md`. This is source registration,
not application evidence: fresh develop preflight and separate SQL/VM authority
are required. Existing `0001`–`0006` payload pins remain unchanged.
