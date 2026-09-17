---
status: needs_decision
domain_profile: backend
scope:
  feature: brain
  layer: brain-core-and-neural-hierarchy
  authorized_work:
    - "Map the current product and implementation boundary for the single Brain and its neural-layer hierarchy."
    - "Record only evidence-backed discovery; this artifact does not approve a feature contract, schema, API, endpoint, persistence, implementation, or delivery plan."
  excluded_work:
    - "Hive Mind behavior and all cross-Brain operational design."
    - "The exact semantic relationship between Cognitive Functions and layers."
    - "Any code, test, SQL, migration, OpenAPI proposal, runtime, infrastructure, deployment, or Git action."
delivery_posture: "Incremental and documentation-first; stop at owner review before feature specification."
owner_priorities:
  - "Develop the Brain and its neural layers before Cognitive Functions."
  - "Preserve freely user-created, arbitrarily nested layers instead of a fixed taxonomy."
  - "Do not close complex lifecycle, selection, identity, persistence, or sharing semantics prematurely."
  - "Keep Hive Mind for a later product boundary."
  - "Be conservative about authorization, persistent data, deletion, hierarchy integrity, and future compatibility."
risk_tolerance: "Conservative for authorization, durable data, deletion, hierarchy integrity, and compatibility; exploratory only for unsettled product semantics."
complexity_tolerance: "Accept real complexity necessary for flexible hierarchy, but begin with the smallest safe boundary and defer nonessential hardening."

verified_facts:
  - id: BF-001
    fact: "PD-001 is owner-approved: each user has exactly one Brain, created automatically on first login; personal, work, company, and project contexts remain internal to that Brain rather than becoming separate Brains."
    evidence:
      - "docs/product-decisions.md (PD-001)"
      - "docs/product/brain/concept.md"
  - id: BF-002
    fact: "PD-005 is owner-approved: users may create layers inside layers without a fixed taxonomy or fixed depth; examples are not defaults."
    evidence:
      - "docs/product-decisions.md (PD-005)"
      - "docs/product/brain/neural-layers/concept.md"
  - id: BF-003
    fact: "PD-005 permits a future Cognitive Function to consider or act on one layer, explicit sublayers, or a group spanning branches with different parents. Selecting a parent does not impose a universal automatic inclusion rule for descendants."
    evidence:
      - "docs/product-decisions.md (PD-005)"
      - "docs/product/brain/concept.md"
      - "docs/product/brain/neural-layers/concept.md"
  - id: BF-004
    fact: "The nature of the Cognitive Function-to-layer relationship is intentionally unresolved; it is not yet context, binding, scope, participation, execution, or another defined model."
    evidence:
      - "docs/open-decisions.md (OD-010)"
      - "docs/product/brain/neural-layers/concept.md"
  - id: BF-005
    fact: "The Brain Engine is the sole authority for authentication, authorization, rules, validation, persistence, effects, and durable state. Clients and agent runtimes do not directly access authoritative data."
    evidence:
      - "AGENTS.md"
      - "docs/architecture-overview.md"
      - "docs/stack/back-end/readme.md"
  - id: BF-006
    fact: "The only current source surface under src is the completed reduced infrastructure package at src/brain-engine/features/infra/. Its approved contract explicitly excludes Brain Engine source, product tables, domain migrations, Mindforge HTTP endpoints, and Flutter integration."
    evidence:
      - "rg --files src (2026-09-12)"
      - "docs/stack/back-end/features/infra/contract.md"
      - "docs/stack/back-end/features/infra/checkpoint.yaml"
  - id: BF-007
    fact: "The infrastructure migrations inspected create only infrastructure-owned Auth state; the static source scan found no Brain, neural-layer, document, or other product-domain DDL."
    evidence:
      - "src/brain-engine/features/infra/postgres/migrations/0001_infra_init.sql"
      - "src/brain-engine/features/infra/postgres/migrations/0002_auth_bootstrap.sql"
      - "src/brain-engine/features/infra/postgres/migrations/0003_auth_schema.sql"
      - "src/brain-engine/features/infra/postgres/migrations/0004_auth_postgres_role.sql"
  - id: BF-008
    fact: "No Brain HTTP operation is approved, proposed, implemented, or runtime-verified: contracts/openapi/registry.yaml contains exactly endpoints: []."
    evidence:
      - "contracts/openapi/README.md"
      - "contracts/openapi/registry.yaml"
  - id: BF-009
    fact: "The current product documents are conceptual and expressly do not authorize schema, API, persistence, implementation, or delivery planning."
    evidence:
      - "docs/product/brain/concept.md"
      - "docs/product/brain/neural-layers/concept.md"
      - "docs/product-decisions.md"
      - "docs/open-decisions.md"
  - id: BF-010
    fact: "Hive Mind is conceptually a separately scoped connection among explicitly selected layers of different users; its consent, permission, lifecycle, identity, and data behavior are still open and are outside this discovery scope."
    evidence:
      - "docs/product-decisions.md (PD-001)"
      - "docs/open-decisions.md (OD-005)"
      - "docs/product/brain/concept.md"

supported_inferences:
  - id: BI-001
    inference: "A Brain feature cannot safely inherit an operational authorization model from the presence of Auth infrastructure: user-to-Brain binding, product authorization, and failure behavior have no approved Brain contract."
    basis:
      - "BF-005"
      - "BF-006"
      - "BF-008"
  - id: BI-002
    inference: "A first slice that excludes cross-user access, Cognitive Functions, Hive Mind, deletion, and reorganization has materially fewer unresolved semantics while preserving the approved free-nesting direction."
    basis:
      - "BF-002"
      - "BF-003"
      - "BF-004"
      - "BF-010"
  - id: BI-003
    inference: "No implementation-readiness claim is possible yet because the owner has not selected the initial layer lifecycle, defaults, product identity/authorization boundary, persistence, or operation set."
    basis:
      - "BF-006"
      - "BF-008"
      - "docs/open-decisions.md (OD-002, OD-004, OD-006, OD-009, OD-010)"

assumptions: []

sources:
  mandatory_repository_authority:
    - "AGENTS.md"
    - "docs/architecture-overview.md"
    - "docs/stack/readme.md"
    - "docs/stack/front-end/readme.md"
    - "docs/stack/back-end/readme.md"
    - "contracts/openapi/README.md"
    - "contracts/openapi/registry.yaml"
  product_authority:
    - "docs/product/brain/concept.md"
    - "docs/product/brain/neural-layers/concept.md"
    - "docs/product-decisions.md"
    - "docs/open-decisions.md"
  implementation_boundary:
    - "docs/stack/back-end/features/infra/contract.md"
    - "docs/stack/back-end/features/infra/checkpoint.yaml"
    - "src/brain-engine/features/infra/"
  method: "Read-only document and static-source inspection on 2026-09-12. No code execution, HTTP call, container, database, or runtime validation was performed for this feature."

current_flow:
  - id: CF-001
    type: verified_current_repository_flow
    flow:
      - "The checkout supplies a reduced PostgreSQL/Auth/pgvector infrastructure package only."
      - "No Brain Engine product source, Brain domain migration, client consumer, or OpenAPI operation exists in the inspected tree."
      - "Therefore no Brain request can be traced from input through authorization, hierarchy behavior, persistence, effects, output, or failure handling."
    observable_outcome: "No mounted or reachable Brain feature flow is evidenced by the repository."
  - id: CF-002
    type: conceptual_target_boundary_not_runtime
    flow:
      - "A future manual client or agent runtime requests a structured operation."
      - "The future Brain Engine authenticates, authorizes, validates, applies deterministic rules, persists authorized state, and returns a contract-defined result."
    limitation: "This is an architectural direction, not an implemented route, protocol, consumer, or runtime proof."

contracts:
  brain_feature_contract:
    status: absent
    evidence: "No approved Brain feature contract was found."
  http_contracts:
    registry: "contracts/openapi/registry.yaml"
    endpoints: []
    state: "No approved, proposed, implemented, or verified Mindforge Brain endpoint."
  persistence_contract:
    status: absent
    limitation: "PostgreSQL/Auth/pgvector infrastructure exists, but product storage, canonical identity, source of truth, synchronization, and migrations remain undecided."
  authorization_contract:
    status: absent
    limitation: "Auth infrastructure does not define product-level actor-to-Brain binding, authorization, or error behavior."
  cognitive_function_contract:
    status: absent
    limitation: "PD-005 fixes flexible selection direction but intentionally leaves its semantic relationship open."

dependencies:
  closed_product_direction:
    - "PD-001: one Brain per user and hierarchical internal organization."
    - "PD-005: freely nested layers and flexible future selection direction."
  current_operational_substrate:
    - "Reduced PostgreSQL/Auth/pgvector infrastructure is documented as complete, but it is not a Brain product implementation."
  future_dependencies_not_yet_contracted:
    - "Product identity and authentication-to-Brain binding."
    - "Product authorization and isolation policy."
    - "Layer lifecycle and hierarchy invariants."
    - "Canonical persistence/source-of-truth and synchronization direction."
    - "Exact initial operation surface and its errors/effects."
  blocked_dependencies:
    - "OD-002: initial layers, lifecycle, reorganization, and limits."
    - "OD-004: persistence, canonical identity, and source of truth."
    - "OD-006: offline and synchronization."
    - "OD-009: exact initial operations, authorization, failures, and effects."
    - "OD-010: Cognitive Function selection semantics and continuity of links."

consumers:
  current: []
  conceptual_only:
    - "A manual Flutter/Dart client."
    - "The client-side Prefrontal Cortex."
    - "The server-side Hidden Cortex through Brain Engine capabilities."
  limitation: "None of these consumers has an approved Brain contract or an implementation in the inspected tree."

closed_decisions:
  - id: PD-001
    decision: "One Brain per user; internal hierarchical layers; no separate implicit Brains for contexts; first-login automatic creation direction."
    boundary: "Does not define bootstrap transaction, schema, storage, API, product roles, synchronization, or layer lifecycle."
  - id: PD-005
    decision: "Free arbitrary nesting and future granular/transversal selection; no universal automatic descendant inclusion."
    boundary: "Does not define selection semantics, group persistence, lifecycle effects, identity strategy, or Hive Mind integration."
  - id: ARCH-AUTH-001
    decision: "Brain Engine remains the sole authority for authentication, authorization, rules, validation, persistence, effects, and durable state."
    boundary: "The exact transport, authentication scheme, product authorization rules, persistence model, and protocol remain open."

unknowns:
  - id: U-001
    question: "Does a newly initialized Brain have no default layers, or an owner-selected starter tree?"
    authority: "OD-002"
  - id: U-002
    question: "Which layer operations belong to the first release: read, create, rename, move, archive, remove, restore, or another constrained set?"
    authority: "OD-002 and OD-009"
  - id: U-003
    question: "What lifecycle, reorganization rules, depth limits, and hierarchy-integrity behavior apply to layers and sublayers?"
    authority: "OD-002"
  - id: U-004
    question: "Whether PD-003's temporary ten-day trash behavior applies to layers, and what happens to descendants, links, and future bindings when a layer is removed."
    authority: "PD-003; OD-002; OD-008; OD-010"
  - id: U-005
    question: "What canonical identity, persistence format, source of truth, and synchronization behavior applies to a Brain and its layers?"
    authority: "OD-004 and OD-006"
  - id: U-006
    question: "How an authenticated identity is bound to exactly one Brain, including first-login idempotency and recovery behavior."
    authority: "PD-001 sets the product outcome; no operational contract currently resolves it."
  - id: U-007
    question: "What product authorization and error behavior prevents a caller from reading or changing another user's Brain before Hive Mind exists."
    authority: "Architecture authority boundary; no approved product contract."
  - id: U-008
    question: "What a Cognitive Function's selection of layers means, how groups persist, and how bindings survive lifecycle changes."
    authority: "OD-010"
  - id: U-009
    question: "Hive Mind consent, visibility, ownership, revocation, lifecycle, and interaction with layer selection."
    authority: "OD-005"
    handling: "Explicitly deferred; not to be designed in this feature."

risks:
  - id: R-001
    risk: "Cross-user or direct-data access could be introduced if infrastructure Auth is mistaken for a complete product authorization boundary."
    likelihood: medium
    impact: critical
    current_mitigation: "Brain Engine sole-authority rule; no Brain endpoint or consumer exists."
    minimum_safe_action: "Specify a private first slice with Brain Engine-enforced actor ownership and no cross-user surface."
    recovery: "Stop at contract divergence or authorization ambiguity; do not add a fallback access path."
    cost_to_address: high
  - id: R-002
    risk: "Deletion, move or archive behavior could orphan descendants or future references; rename now requires stable identity and references but still needs validation and concurrency semantics."
    likelihood: high
    impact: high
    current_mitigation: "PD-007 closes rename identity continuity while OD-002 and OD-010 keep the remaining lifecycle behavior open; no implementation exists."
    minimum_safe_action: "Include rename only with stable identity; exclude destructive and reorganization operations from the first slice."
    recovery: "Do not expose the operation; revisit after lifecycle and reference behavior are approved."
    cost_to_address: high
  - id: R-003
    risk: "A starter taxonomy could be treated as mandatory despite PD-005 and the concept documents declaring examples non-default."
    likelihood: medium
    impact: medium
    current_mitigation: "Documentation consistently labels examples as illustrative."
    minimum_safe_action: "Choose an empty root by default unless the owner explicitly approves an exact starter tree."
    recovery: "Keep defaults out of the product contract until selected by the owner."
    cost_to_address: low
  - id: R-004
    risk: "Cognitive Function selection semantics could be locked into hierarchy behavior before the owner decides whether it is context, binding, scope, participation, execution, or another model."
    likelihood: high
    impact: high
    current_mitigation: "PD-005 and OD-010 explicitly preserve the uncertainty."
    minimum_safe_action: "Do not add Cognitive Function selection, saved groups, or descendant propagation in the Brain core slice."
    recovery: "Retain only the closed flexibility direction and return to owner review when Cognitive Functions are explored."
    cost_to_address: high
  - id: R-005
    risk: "Premature cross-Brain behavior could leak content or permissions because Hive Mind governance is unresolved."
    likelihood: medium
    impact: critical
    current_mitigation: "PD-001 scopes Hive Mind conceptually; OD-005 remains open; this discovery excludes it."
    minimum_safe_action: "Make the first Brain contract single-user/private with no Hive Mind behavior."
    recovery: "Reject cross-Brain scope as out of contract and require a later owner-approved Hive Mind feature."
    cost_to_address: high
  - id: R-006
    risk: "Future sync, offline edits, or concurrency can break hierarchy integrity if product identity and source of truth are assumed from infrastructure alone."
    likelihood: medium
    impact: high
    current_mitigation: "OD-004 and OD-006 are open; no client or product persistence exists."
    minimum_safe_action: "Keep offline/sync and concurrent reorganization outside the first feature contract."
    recovery: "Specify them in a future decision and contract before exposing such behavior."
    cost_to_address: high

minimum_safe_recommendation:
  status: partially_approved_pending_default_policy
  boundary: "Start one private Brain-core slice before Cognitive Functions: an authenticated owner can observe the one Brain promised by PD-001, create user-named layers at the top level or under an existing same-Brain layer, and rename them, allowing arbitrary nesting without a prescribed taxonomy."
  recommended_initial_behavior:
    - "Brain initialization is idempotent for the authenticated user and produces one Brain, while the exact transactional mechanism remains a later implementation choice."
    - "The initial default-layer policy remains pending; Inbox is not a layer and cannot be used as an implicit default."
    - "A layer can be created under the Brain root or another layer belonging to that same Brain."
    - "A layer can be renamed without changing its stable identity or references."
    - "The Brain Engine enforces ownership/isolation and validates hierarchy containment; no client or agent gets direct persistence authority."
    - "The product contract requires references to remain distinct from mutable display labels and hierarchy position, without choosing a storage technology or identifier format here."
  explicit_exclusions:
    - "Cognitive Functions, selection groups, and descendant-selection semantics."
    - "Hive Mind and every cross-Brain action."
    - "Move, archive, delete, restore, retention, and descendant-handling behavior. Rename is included only with identity continuity; its remaining validation semantics must be contracted."
    - "Offline operation, synchronization, conflict handling, retrieval, graph, embeddings, and documents."
    - "Schema, SQL, API method/path, transport, framework, and implementation plan."
  rationale: "This is the smallest boundary that gives the owner a real, private, freely nestable Brain hierarchy while avoiding unresolved lifecycle, sharing, and Cognitive Function semantics."

required_now:
  - "Owner review and approval of the intended first Brain-core behavior and its explicit exclusions."
  - "An owner decision on default-layer policy and initial layer lifecycle surface before a feature specification."
  - "Owner approval of the draft Brain-core contract, including the remaining rename validation semantics and ATDD scenarios."
  - "An explicit TDD yes/no choice before contract closure; only then can planning begin."

deferred_hardening:
  - "Move, archive, deletion, temporary-trash applicability, restore, purge, and handling of descendants/references."
  - "Depth/size quotas, cycle prevention during future reorganization, optimistic concurrency, versioning, and audit history."
  - "Offline replicas, synchronization, conflicts, and source-of-truth policy."
  - "Cognitive Function selection semantics, saved/transversal groups, descendant inclusion/exclusion modes, and continuity of bindings."
  - "Hive Mind consent, ACLs, ownership, visibility, revocation, propagation, and cross-Brain relationship rules."
  - "Document/retrieval/graph/vector behavior and data-model integration."
  - "Exact persistence technology, schema, migration, transport, API, error model, observability, rate limits, and recovery automation."

alternatives:
  - id: A-001
    option: "Narrow private Brain root plus read/create/rename nested layers."
    strengths:
      - "Directly exercises the closed one-Brain and free-nesting decisions."
      - "Avoids choosing defaults, destructive lifecycle, Cognitive Function semantics, or sharing behavior."
      - "Minimizes authorization surface to one owner's private hierarchy."
    tradeoffs:
      - "Users cannot yet move, archive, delete, restore or group existing layers."
      - "A later lifecycle feature must extend the initial contract deliberately."
  - id: A-002
    option: "Owner-defined starter tree, using explicitly approved layer names, in the first slice."
    strengths:
      - "Can improve initial orientation and demonstrate an intended workflow sooner."
    tradeoffs:
      - "Requires an explicit default/lifecycle decision and increases migration, customization, and reset questions."
      - "Risks converting examples into a de facto taxonomy."
  - id: A-003
    option: "Full hierarchy CRUD in the first slice, including rename, move, archive, and deletion."
    strengths:
      - "Offers richer manual organization immediately."
    tradeoffs:
      - "Requires owner decisions for lifecycle, references, descendants, trash, recovery, authorization, concurrency, and synchronization before it is safe."
      - "Is not recommended under the stated conservative risk posture."
  - id: A-004
    option: "Bundle Cognitive Function selection or Hive Mind with the hierarchy foundation."
    strengths:
      - "Would demonstrate granular/transversal selection or sharing early."
    tradeoffs:
      - "Contradicts the requested delivery order and depends on unresolved OD-005 and OD-010 semantics."
      - "Is outside this feature boundary and not recommended."

decisions_requested:
  - id: D-001
    decision: "Choose the initial default-layer policy."
    options:
      - "Recommended: initialize only the required Brain root; users create their own first layer."
      - "Approve an exact named starter tree and define whether it can be changed or removed."
    why_now: "PD-005 rejects implicit defaults, while OD-002 leaves defaults open."
  - id: D-002
    decision: "Resolved: initial manual hierarchy operation set."
    options:
      - "Owner decision: observe the private hierarchy; create top-level or nested same-Brain layers without fixed product depth; rename while preserving identity. Defer move, archive, delete, restore, and purge."
    why_now: "Resolved in PD-007; rename validation details remain for the draft contract."
  - id: D-003
    decision: "Resolved: private authorization boundary for the first slice."
    options:
      - "Owner decision: only the authenticated owner may access or change that owner's Brain; no delegation, cross-user access, or Hive Mind behavior in Brain Core."
    why_now: "Resolved in PD-007; technical authorization remains for later contract and implementation design."
  - id: D-004
    decision: "Resolved: first-login expectation and reference-continuity invariant."
    options:
      - "Owner decision: repeated or concurrent first access converges idempotently on one Brain; layer identity and references do not depend on mutable name or position."
    why_now: "Resolved in PD-007; mechanism and identifier format remain open."
  - id: D-005
    decision: "Resolved: Cognitive Function semantics and Hive Mind execution stay excluded from Brain Core."
    options:
      - "Owner decision: defer both to their own future contracts; preserve compatibility with PD-008 without implementing sharing."
    why_now: "Resolved for Brain Core scope; PD-008 records the future Hive Mind subtree direction."

owner_decisions:
  status: partial_d001_pending
  decisions:
    - "D-002 resolved by PD-007: observe, create nested layers without fixed product depth, and rename; other lifecycle operations deferred."
    - "D-003 resolved by PD-007: owner-only private Brain Core."
    - "D-004 resolved by PD-007: idempotent/convergent bootstrap and stable identity independent of name or position."
    - "D-005 resolved for Brain Core: Cognitive Functions and Hive Mind execution remain out of scope; PD-008 records the future subtree boundary."

layer_recommendation:
  status: one_product_layer_recommended
  layer: "brain-core-and-neural-hierarchy"
  boundary: "Keep one specification boundary while the owner selects the default-layer policy and closes the draft contract. Do not split into technical implementation batches or add Cognitive Function/Hive Mind behavior before contract approval."

spec_readiness:
  status: not_ready
  blockers:
    - "D-001 default-layer policy remains unresolved."
    - "The Brain-core contract exists only as a draft; its ATDD scenarios, rename validation semantics, TDD choice and owner approval remain pending."
    - "contracts/openapi/registry.yaml must remain endpoints: [] until a future operation is explicitly proposed and approved."
  readiness_after_owner_review: "After D-001, ATDD, TDD selection and explicit contract approval, invoke orchestration for planning; do not proceed directly to OpenAPI, schema or implementation."

validation:
  performed:
    - "Read all six mandatory repository sources in AGENTS.md order."
    - "Read the Brain concept, neural-layer concept, product-decision ledger, open-decision ledger, and infrastructure boundary artifacts."
    - "Statically inspected the src tree and infrastructure migration surface."
    - "Confirmed contracts/openapi/registry.yaml remains endpoints: []."
  boundary: "Documentary and static-source evidence only. No runtime, database, HTTP, container, client, deployment, Git, or production claim is made."

next_action: close_brain_core_draft_with_owner
---

# Brain discovery

This artifact deliberately separates product decisions already approved from the
unresolved operational semantics that must not be invented during discovery.
It is a gate record, not an implementation or interface contract.

## Local source navigation

- [Brain concept](concept.md)
- [Neural-layer concept](neural-layers/concept.md)
- [Product decisions](../../product-decisions.md)
- [Open decisions](../../open-decisions.md)
- [Architecture overview](../../architecture-overview.md)
- [OpenAPI registry](../../../contracts/openapi/registry.yaml)

## Evidence limit

The repository has no Git metadata at its root, so branch, commit, upstream,
and worktree-cleanliness evidence is unavailable. This discovery does not use
that absence as product evidence. Its product conclusion comes only from the
current documentation, contract registry, and static `src` inspection listed
above.
