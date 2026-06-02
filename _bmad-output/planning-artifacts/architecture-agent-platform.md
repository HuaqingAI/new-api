---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
workflowType: 'architecture'
lastStep: 8
status: 'complete'
completedAt: '2026-05-31'
revisedAt: '2026-05-31'
revision: 'V1.3'
revisionNotes: |
  V1.0 (2026-05-31):
  - Complete BMad architecture artifact for the Agent Platform product line.
  - Freeze Knowledge as a provider-backed retrieval contract.
  - Freeze Agent as definition/template distribution only; MVP does not include server-side execution runtime.
  V1.1 (2026-05-31):
  - Reclassify OAuth 2.1 details, `client_instance`, and external retrieval-provider selection as ADR/pending implementation choices rather than PRD-frozen decisions.
  - Restore DG-2 as an open product decision pending explicit follow-up prioritization.
  - Promote FR-6 freshness/convergence semantics and FR-15 contract-version governance into explicit architecture constraints.
  - Clarify that MVP includes an admin management UI, with `web/default` as the primary implementation target and no Classic parity commitment.
  V1.2 (2026-05-31):
  - Freeze the MVP delegated-auth implementation profile to authorization code + PKCE for user-delegated browser flows, plus optional client credentials for trusted server-to-server consumers.
  - Clarify that `client_instance` is not a required MVP persistence object or route surface; instance-level identity remains a post-MVP architecture amendment unless explicitly pulled in.
  - Normalize open capability endpoint examples to the repository's `/resource/:id/action` route style.
  - Temporarily downgrade readiness to "READY WITH MINOR GAPS" until UX and implementation-readiness issues are resolved.
  V1.3 (2026-05-31):
  - Freeze `Knowledge` provider integration to a provider-neutral `http_retrieval` adapter boundary; engines such as LightRAG, FastGPT, and RAGFlow remain swappable implementations rather than public contract types.
  - Resolve second-consumer priority after Cherry Studio: Codex is validated before `cc switch`.
  - Align implementation planning with the readiness review by removing the Epic 1 -> Epic 2 forward dependency, splitting oversized auth work, and adding a lightweight UX artifact as an implementation input.
inputDocuments:
  - "_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md"
  - "_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md"
  - "_bmad-output/planning-artifacts/architecture.md"
  - "_bmad-output/planning-artifacts/ux-agent-platform.md"
  - "_bmad-output/planning-artifacts/implementation-readiness-report-2026-05-31.md"
  - "_bmad-output/planning-artifacts/research/technical-lightrag-knowledge-base-research-2026-05-31.md"
  - "docs/integration-architecture.md"
  - "AGENTS.md"
project_name: 'new-api'
user_name: 'hth'
date: '2026-05-31'
---

# Architecture Decision Document: Agent Platform

_This document is the BMad architecture source of truth for the `new-api` Agent Platform product line. It inherits global repository constraints from `AGENTS.md` and shared platform foundations from `_bmad-output/planning-artifacts/architecture.md`, but owns all Agent Platform specific decisions._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**

The Agent Platform is a new product line inside `new-api`, not a relay rewrite. It introduces a unified control plane for three resource types:

1. `Skill`
2. `Knowledge`
3. `Agent`

The platform must let administrators and publishers:

- manage resource definitions, versions, publish state, rollback, revoke, and diagnostics
- register downstream consumers as `client`
- distinguish product-level access (`client`) from device/runtime-level access (`client_instance`)
- expose a stable open capability layer for `discovery`, `detail`, `invoke/query`, `refresh`, and revoke handling
- provide OAuth-style delegated authorization so downstream products can obtain tokens and directly access platform data
- keep `Agent` in MVP as a template/definition object with dependency references, not a server-side execution runtime

The PRD groups the core feature set into six implementation domains:

1. unified resource control plane: FR-1, FR-2, FR-3
2. client registry and capability standard layer: FR-4, FR-5, FR-6, FR-15
3. skill library management and invoke contract: FR-7, FR-8
4. knowledge library management and retrieval contract: FR-9, FR-10
5. agent library management and dependency boundary: FR-11, FR-12
6. audit, error, and diagnostics: FR-13, FR-14

**Non-Functional Requirements:**

The architecture must satisfy the PRD governance constraints and repository rules:

- all new backend code must remain compatible with SQLite, MySQL 5.7.8+, and PostgreSQL 9.6+
- all JSON marshal/unmarshal flows must use `common/json.go`
- management APIs must be documented in `docs/openapi/api.json`
- the relay surface under `/v1/**` and related upstream DTO behavior must remain unchanged
- the open capability layer must be contract-stable and versioned
- OAuth-style direct token access must support revoke, audit, and diagnosis
- Knowledge cannot be reduced to a flat metadata list; the architecture must allow integrating an external knowledge engine without forcing `new-api` to own the full RAG pipeline
- new UI text in `web/default` must be internationalized

**Scale & Complexity:**

- Primary domain: control plane + open capability data plane + delegated auth + provider-backed retrieval
- Complexity level: high
- Architectural shape: brownfield extension inside an existing multi-surface product

The complexity comes from four boundaries that must stay clean:

1. control plane vs open capability data plane
2. admin session auth vs downstream bearer-token auth
3. resource definition vs published exposure
4. platform-managed metadata vs externally executed knowledge/agent runtime

### Technical Constraints & Dependencies

- Backend stays inside the existing Go monolith using `Router -> Controller -> Service -> Model`.
- Frontend control plane follows the `web/default` React 19 + Rsbuild + Base UI + Tailwind stack; Bun remains the preferred package manager.
- Existing session auth and upstream login/OAuth providers already live in `controller/*`, `oauth/*`, `middleware/auth.go`, and `router/api-router.go`. Agent Platform data-plane auth must coexist with them, not replace them.
- Existing route organization already reserves `/api/**` for management APIs. Agent Platform must use that namespace, not create a second gateway.
- Existing repo rules forbid direct `encoding/json` use in business code, PostgreSQL-only JSONB assumptions, and breaking relay DTO semantics.
- Existing repo already contains OAuth/OIDC concepts for user login. Those can be reused as upstream identity sources, but the Agent Platform bearer tokens must be first-party tokens issued by `new-api`, not reused relay tokens or dashboard sessions.

**Knowledge integration dependency note:**

The latest provider research and implementation-readiness review converge on one architecture rule: `Knowledge` must stay provider-neutral at the public contract boundary.

- `LightRAG` is a viable candidate because it already supports structured retrieval/query flows, graph/vector-oriented storage abstractions, and multimodal extension.
- `FastGPT`, `RAGFlow`, and similar engines are also plausible candidates, but they remain external runtimes with their own storage, operational, and product-surface tradeoffs.
- The platform therefore must not elevate any single engine into a top-level public contract type or let provider-native response fields leak into the shared capability contract.

This leads to a single architecture constraint: `new-api` owns metadata, auth, exposure, diagnostics, and contract normalization, while any external knowledge engine is integrated behind a normalized `http_retrieval` adapter boundary.

### Cross-Cutting Concerns Identified

- **Contract stability:** the first consumer is Cherry Studio, but the core standard cannot collapse into Cherry-specific protocol branches.
- **Delegated auth correctness:** downstream tools need OAuth-style authorization and direct bearer-token access, while admin users still need dashboard session auth.
- **Revocation and convergence:** resource revoke, client disable, token revoke, and publish rollback all need deterministic convergence windows and audit trails.
- **Knowledge system boundary:** the platform must govern `Knowledge` lifecycle and visibility without taking on ingestion, chunking, embeddings, vector DB ownership, or long-running RAG orchestration in MVP.
- **Agent runtime scope control:** `Agent` must remain a definition/template with dependency semantics; server-side orchestration is explicitly out of scope.
- **Product vs device identity:** the architecture may distinguish product-level access from device/runtime-level access, but the exact persistence model for instance-level identity remains an implementation decision unless and until MVP validation proves it is necessary.
- **Observability:** every failure must be attributable to one of: resource state, exposure state, grant state, client capability mismatch, contract mismatch, provider failure, or platform internal error.

## Starter Template Evaluation

### Primary Technology Domain

Brownfield full-stack platform extension inside an existing Go + React monorepo.

The architecture does not start from a greenfield starter. It extends a production repository that already contains:

- backend request routing, auth, settings, billing, caching, and logging
- dual frontend surfaces, with `web/default` as the modern admin UI
- deployment patterns for local, server, Docker, and Electron packaging

### Starter Options Considered

**Option 1: Existing `new-api` monorepo**

Use the current repository as the only valid architectural starter.

Why it fits:

- the project already owns auth, user/session lifecycle, OpenAPI docs, DB migration patterns, and the protected project identity
- the Agent Platform must share the same DB and management backend
- the work is additive, not a re-platform

**Option 2: New standalone service**

Rejected for MVP.

Why it does not fit:

- it would split auth, settings, deployments, and operational ownership too early
- it would create duplicate admin surfaces and deployment complexity before the product line is validated

**Option 3: Embed a full knowledge/RAG framework into the Go service**

Rejected for MVP.

Why it does not fit:

- knowledge ingestion/indexing is a separate product problem
- it would force Python/runtime/storage concerns into the main Go process
- it weakens the clean provider boundary needed for future engines beyond LightRAG

### Selected Starter: Existing `new-api` Monorepo

**Rationale for Selection:**

The Agent Platform will be implemented inside the current repository, same service process, same DB, same admin site, and same deployment model. It is a new architectural domain, not a new product shell.

**Initialization Command:**

```bash
# No external starter command.
# Implement inside the existing repository.
```

**Architectural baseline inherited from the repository:**

- existing dashboard user/session auth stays in place
- `docs/openapi/api.json` remains the management API contract location
- `web/default` is the primary admin frontend for new control-plane UI
- the relay surface remains frozen unless a future explicit architecture amendment approves changes

## Core Architectural Decisions

### Decision Priority Analysis

**Critical decisions frozen by this architecture:**

- **CP-AP-1:** resource definition, version, and exposure are separate concepts
- **CP-AP-2:** control-plane APIs and open-capability APIs are separate surfaces
- **CP-AP-3:** `Knowledge` is exposed as a retrieval contract backed by a provider adapter boundary
- **CP-AP-4:** `Agent` is a template/definition resource only; no server-side execution runtime, execution state, or multi-step orchestration ownership in MVP
- **CP-AP-5:** MVP includes an admin management UI, with `web/default` as the primary implementation target
- **CP-AP-6:** client-specific metadata may only extend the standard contract through namespaced extension fields and may not redefine core semantics
- **CP-AP-7:** FR-6 freshness semantics are contract requirements, not cache hints
- **CP-AP-8:** FR-15 contract-version governance is a first-class architecture constraint

**Important decisions that shape implementation detail:**

- **CP-AP-9:** use a shared registry + typed detail model rather than a single giant polymorphic table
- **CP-AP-10:** JWT access tokens are short-lived and first-party signed; refresh tokens are opaque and stored hashed
- **CP-AP-11:** the open capability plane is organized under `/api/open-capabilities/**`, but URI layout does not replace `Contract Version` governance
- **CP-AP-12:** `web/default` is the default implementation surface for the MVP management UI; Classic parity is out of scope unless a later product decision amends that scope
- **CP-AP-13:** Codex is the second validation consumer after Cherry Studio; `cc switch` and later consumers must validate against the same standardized contract without introducing a parallel protocol branch

**Deferred decisions:**

- additional delegated-auth grant types beyond the MVP profile
- whether post-MVP delivery requires a persisted `client_instance` model beyond product-level `client`
- which external retrieval provider is used first in delivery work
- additional knowledge modes beyond `retrieval`
- embedded or server-side agent execution runtime
- non-HTTP provider transports for external knowledge engines
- Classic theme parity for the Agent Platform control plane

### Data Architecture

**Shared registry + typed detail model:**

The platform will use a registry-driven model instead of a single giant table. This preserves shared governance fields while keeping each resource type honest about its own structure.

**Core tables:**

1. `agent_platform_resources`
   - stable `resource_id`
   - `resource_type` = `skill | knowledge | agent`
   - `display_name`
   - `owner_user_id`
   - `status`
   - `latest_version`
   - `tenant_id`
   - `created_at`, `updated_at`

2. `agent_platform_resource_versions`
   - `resource_id`
   - semantic `version`
   - `contract_version`
   - `summary`
   - `schema_json`
   - `detail_json`
   - `status`
   - `created_by`
   - `published_at`
   - `created_at`, `updated_at`

3. typed detail tables
   - `agent_platform_skill_defs`
   - `agent_platform_knowledge_defs`
   - `agent_platform_agent_defs`

4. publication / visibility tables
   - `agent_platform_clients`
   - optional `agent_platform_client_instances`
   - `agent_platform_resource_exposures`

5. auth / grant tables
   - `agent_platform_authorization_grants`
   - `agent_platform_refresh_tokens`

6. audit / tasks
   - `agent_platform_admin_actions`
   - `agent_platform_publish_tasks`
   - `agent_platform_provider_health_checks`

**Typed detail shape:**

- `Skill` detail owns invoke schema, output schema, invoke mode, timeout, and binding config.
- `Knowledge` detail owns `knowledge_mode`, provider type, provider adapter key, provider binding config, normalized query/citation schema, freshness rules, and provider capability declarations.
- `Agent` detail owns manifest, dependency references, prompt/template metadata, and compatibility metadata. It does not own runtime execution state in MVP.

**Client model:**

`agent_platform_clients` is the MVP product-level principal:

- `client_id` (stable opaque id)
- `slug`
- `display_name`
- `client_type`
- `status`
- `allowed_grant_types`
- `redirect_uris_json`
- `allowed_scopes_json`
- `contract_version`
- `capabilities_json`
- `extensions_json`
- `allow_client_credentials`

`agent_platform_client_instances` is a post-MVP extension point for device/runtime-level identity if implementation validation later proves instance-level revoke and telemetry are required:

- `instance_id` (stable opaque id)
- `client_id`
- `display_name`
- `instance_type`
- `status`
- `metadata_json`
- `token_version`
- `last_seen_at`
- `last_ip`
- `revoked_at`

If instance-level identity is introduced in a later architecture amendment, the separation is deliberate:

- `client` answers "what product/application is authorized?"
- `client_instance` answers "which installation/device/runtime currently holds access?"

**Exposure model:**

`agent_platform_resource_exposures` is the published projection table. It must not be inferred from registry status alone.

Key fields:

- `resource_id`
- `resource_version`
- `client_id`
- optional `client_instance_scope`
- `visibility_state`
- `callable_state`
- `freshness_ttl_seconds`
- `etag`
- `extensions_json`
- `published_at`
- `revoked_at`

This enables:

- visible but not callable
- callable for one client but not another
- revoked for one client instance while still visible at the product level

**OAuth / grant model:**

The platform requires a dedicated bearer-token auth plane for downstream consumers. For MVP, the delegated-auth profile is frozen to:

- authorization code + PKCE for user-delegated browser flows
- optional client credentials for trusted server-to-server consumers explicitly enabled per `client`

All other grant types remain post-MVP decisions unless this document is amended.

`agent_platform_authorization_grants` records first-party grants:

- `grant_id`
- `client_id`
- nullable `client_instance_id`
- nullable `user_id`
- `grant_type`
- `scope_text`
- `status`
- `token_version`
- `contract_version`
- `consented_at`
- `revoked_at`

`agent_platform_refresh_tokens` stores:

- `grant_id`
- hashed `refresh_token`
- `expires_at`
- `rotated_from_id`
- `revoked_at`
- `last_used_at`

Access tokens are not stored as plaintext rows. They are signed JWTs validated with:

1. signature
2. expiry
3. `client` status
4. optional `client_instance` status
5. current `grant.token_version`

This design gives immediate revoke without turning access tokens into fully stateful session rows.

**Knowledge provider model:**

The platform must allow external knowledge engines without making them first-class data owners of the control plane.

`agent_platform_knowledge_defs` includes:

- `knowledge_mode` = `retrieval` in MVP
- `provider_type` = `native | http_retrieval`
- `provider_adapter_key` = opaque adapter id such as `lightrag`, `fastgpt`, `ragflow`, `haystack`, or `custom`
- `provider_config_json`
- `query_schema_json`
- `citation_schema_json`
- `freshness_strategy`
- `max_staleness_seconds`

**External retrieval provider decision:**

The architecture explicitly supports integrating external retrieval providers, but only through the normalized `http_retrieval` adapter contract.

- providers are not embedded into the Go process
- providers are not the mandatory default for all tenants
- provider adapter choice does not change the public `Knowledge` contract shape
- providers do not own the control-plane registry/version/exposure model

That means `new-api` owns metadata, auth, exposure, diagnostics, and query/citation normalization, while the provider owns ingestion/indexing/retrieval internals.

**Cross-DB guidance:**

- use GORM for all tables and migrations
- JSON-bearing columns use `TEXT` with `common.Marshal` / `common.Unmarshal`
- no PostgreSQL-only JSONB operators
- no raw SQL unless unavoidable and cross-DB safe

### Authentication & Security

**Two auth planes:**

1. **Control plane auth**
   - existing dashboard session auth
   - existing user login methods remain the source of authenticated admin identity
   - applies to `/api/agent-platform/**`

2. **Open capability data-plane auth**
   - first-party bearer-token auth
   - applies to `/api/open-capabilities/**`

These are intentionally separate.

**Data-plane auth profile:**

The product requirement is a dedicated delegated-authorization mechanism for downstream consumers. The MVP implementation uses an OAuth-style profile, but it is intentionally narrower than a full generic OAuth server.

Guardrails:

- do not reuse relay tokens or dashboard sessions as downstream bearer tokens
- do not let auth choices redefine the core resource contract
- do not make the second consumer decision implicit through auth design alone
- do not add non-MVP grant types without an architecture amendment

**Authorization flow design:**

- browser-based downstream product starts at `/api/agent-platform/oauth/authorize`
- if the user is not logged in, the platform uses the existing `new-api` login/session flow first
- after login, the user consents to the requested scopes for the target `client`
- token exchange happens at `/api/agent-platform/oauth/token`
- downstream then directly calls `/api/open-capabilities/**` with bearer token

For trusted server-to-server consumers, MVP may additionally allow `client_credentials` token exchange only when the `client` record explicitly enables it. That path does not create end-user delegated grants or consent records.

This satisfies the user requirement: downstream products can receive delegated authorization and then directly use tokens to access data.

**Token form:**

- access token: signed JWT, short-lived, first-party issuer
- refresh token: opaque random token, stored hashed

Access token claims include:

- `iss`
- `sub`
- `aud`
- `exp`
- `jti`
- `client_id`
- optional `client_instance_id`
- optional `user_id`
- `scope`
- `contract_version`
- `token_version`

**Revoke behavior:**

Revoke must work at these levels:

1. resource exposure revoke
2. client revoke
3. grant/token revoke
4. optional client instance revoke only if a later architecture amendment introduces instance identity

Implementation rule:

- access token validation must consult current `token_version` from cache/DB
- rotating or revoking a grant/client increments or invalidates the current version
- optional `client_instance` version invalidation applies only if instance identity is later introduced
- refresh tokens are individually revocable and rotated on use

**Scope model:**

MVP scopes are explicit and resource-oriented:

- `ap.resources.read`
- `ap.skills.invoke`
- `ap.knowledge.query`
- `ap.agents.read`
- `ap.clients.self`

No scope may imply admin control-plane rights.

**Extension policy for client-specific fields:**

Client-specific metadata is allowed only under namespaced extension objects, for example:

```json
{
  "extensions": {
    "cherry_studio": {
      "ui_variant": "enterprise"
    }
  }
}
```

Rules:

- extension fields may add hints
- extension fields may not redefine lifecycle, visibility, error, or auth semantics
- core consumers must ignore unknown extensions safely

**Secret handling:**

- OAuth signing keys live in environment/config, not in frontend payloads
- refresh tokens are stored hashed
- provider secrets for external knowledge engines are never returned to the browser
- no bearer tokens in logs or admin audit payloads

### API & Communication Patterns

**Surface split:**

1. control plane
   - `/api/agent-platform/**`
   - session auth
   - CRUD, publish, revoke, audit, client management

2. auth plane
   - `/api/agent-platform/oauth/**`
   - mixed unauthenticated + authenticated browser flow depending on step
   - authorization, token exchange, consent, revoke

3. open capability plane
   - `/api/open-capabilities/**`
   - bearer token auth
   - discovery, detail, invoke/query, refresh

**Control-plane endpoints:**

- `/api/agent-platform/resources`
- `/api/agent-platform/resources/:id/versions`
- `/api/agent-platform/skills`
- `/api/agent-platform/knowledge-bases`
- `/api/agent-platform/agents`
- `/api/agent-platform/clients`
- `/api/agent-platform/exposures`
- `/api/agent-platform/publish-tasks`
- `/api/agent-platform/admin-actions`

**Auth endpoints:**

- `/api/agent-platform/oauth/authorize`
- `/api/agent-platform/oauth/token`
- `/api/agent-platform/oauth/revoke`
- `/api/agent-platform/oauth/consents`

**Open capability endpoints:**

- `/api/open-capabilities/discovery`
- `/api/open-capabilities/resources/:id`
- `/api/open-capabilities/skills/:id/invoke`
- `/api/open-capabilities/knowledge-bases/:id/query`
- `/api/open-capabilities/agents/:id`
- `/api/open-capabilities/refresh`

**Capability contract policy:**

- `discovery` returns published projections only
- `detail` returns normalized contract data plus supported extension namespaces
- `invoke/query` must be normalized behind a shared status/error envelope
- `refresh` lets the consumer reconcile ETag/version/freshness state

**Freshness and convergence:**

MVP exposure freshness rule:

- default `TTL` upper bound: 300 seconds
- responses return `version`, `etag`, and `freshness`
- `refresh` and revoke flows must converge within that window

This is a contract-level requirement derived from FR-6, not an implementation hint. Specifically:

- every published projection must expose `TTL`, `freshness`, `version`, and/or `etag` semantics in a consumer-readable way
- `freshness` must at minimum support `fresh`, `stale`, `offline`, and `revoked`
- if a projection passes its `TTL` and the client has not reconciled to a matching `etag` or `version`, the projection must be treated as `stale`
- after `revoke`, `rollback`, `disable`, or transition to `offline`, consumers must converge by the next `refresh` or within the 300-second upper bound
- continuing to use a stale projection beyond that window is a client non-compliance condition, not a silent best-effort scenario

**Contract versioning and extension governance:**

FR-15 is also a hard architecture constraint. The open capability layer must define:

- where `contract_version` is declared in client registration, resource detail, and token/context metadata
- what constitutes a breaking change and when `Contract Version` must be incremented
- backward-compatibility expectations for optional fields and deprecated fields
- a namespaced extension model such as `extensions.<namespace>` for consumer-specific metadata
- predictable behavior for unknown optional fields, namespaced extensions, and deprecated fields on both platform and client sides

URI layout such as `/api/open-capabilities/**` organizes the surface but does not replace version governance.

**Status model:**

Registry/resource status:

- `draft`
- `published`
- `disabled`
- `revoked`
- `offline`
- `deprecated`

Exposure status:

- `visible`
- `hidden`
- `revoked`
- `stale`

Callable status:

- `callable`
- `contract_invalid`
- `auth_denied`
- `provider_offline`
- `revoked`

**Error envelope:**

The open capability layer uses a stable normalized error envelope:

```json
{
  "success": false,
  "error": {
    "code": "contractInvalid",
    "message": "Human readable summary",
    "retryable": false,
    "request_id": "req_xxx",
    "resource_id": "res_xxx",
    "resource_version": "1.2.0"
  }
}
```

Required normalized codes:

- `permissionDenied`
- `resourceRevoked`
- `resourceOffline`
- `quotaOrRateLimited`
- `timeout`
- `upstreamFailed`
- `contractInvalid`

**Contract-version policy:**

- breaking changes require `contract_version` change
- non-breaking additive changes may remain in the same contract version
- client-specific extra metadata must stay in extensions

### Frontend Architecture

**Primary surface:**

MVP includes a full admin management UI for the Agent Platform, with `web/default` as the primary implementation target.

Rationale:

- it already hosts the modern admin experience
- it already uses the preferred toolchain
- it is where new control-plane complexity belongs

**Classic policy:**

`web/classic` is not planned for dedicated Agent Platform parity in MVP. This is a scope-cut decision for the first delivery slice, not a permanent product principle.

**Default feature areas:**

- `agent-platform-overview`
- `agent-platform-clients`
- `agent-platform-skills`
- `agent-platform-knowledge`
- `agent-platform-agents`
- `agent-platform-publishing`
- `agent-platform-audit`

**Information architecture:**

Navigation order:

1. Agent Platform Overview
2. Clients
3. Skills
4. Knowledge
5. Agents
6. Publishing
7. Audit & Diagnostics

**Query key pattern:**

```ts
['agent-platform', 'clients', ...]
['agent-platform', 'skills', ...]
['agent-platform', 'knowledge', ...]
['agent-platform', 'agents', ...]
['agent-platform', 'publishing', ...]
['agent-platform', 'audit', ...]
```

**Admin OAuth UX:**

The browser admin UI manages:

- client registration
- redirect URIs and scopes
- consent visibility
- publish/revoke state
- diagnostics

It does not itself become a bearer-token consumer for the open capability data plane except for testing tools inside the control plane.

### Infrastructure & Deployment

**Deployment shape:**

- same Go service process
- same DB
- same admin frontend
- same operational ownership

No microservice split is introduced for the core platform in MVP.

**Optional infrastructure additions:**

- Redis for token-version cache, grant revoke cache, and exposure freshness invalidation
- background jobs for token cleanup, provider health checks, and publish-task convergence

**Knowledge provider deployment rule:**

External knowledge engines are deployed out-of-process. If a tenant chooses a provider such as LightRAG, FastGPT, or RAGFlow:

- the provider runs as its own service/runtime
- its indexing/storage concerns stay outside `new-api`
- `new-api` stores only provider config, exposure rules, health state, and normalized retrieval contract metadata

**External provider tradeoff summary:**

Benefits:

- provider diversity without public-contract drift
- clear provider boundary
- future path to graph-aware or multimodal extension

Costs:

- extra runtime and deployment surface
- separate health/latency/failure modes
- provider-specific normalization work

This tradeoff is acceptable because it preserves the core architectural boundary: `new-api` governs access and lifecycle, not end-to-end knowledge engineering.

**Environment/config additions:**

- `AGENT_PLATFORM_JWT_PRIVATE_KEY`
- `AGENT_PLATFORM_JWT_KID`
- `AGENT_PLATFORM_TOKEN_ISSUER`
- `AGENT_PLATFORM_ACCESS_TOKEN_TTL`
- `AGENT_PLATFORM_REFRESH_TOKEN_TTL`

Provider endpoint and secret metadata that vary by tenant/client live in DB-backed config, not hard-coded env-only wiring.

### Decision Impact Analysis

**Why this architecture is deliberately conservative:**

- it reuses the monolith rather than splitting services early
- it adds a new auth plane without disturbing existing session auth or relay tokens
- it treats knowledge engines as providers, not core runtime
- it constrains Agent to definition/template semantics until the product proves that runtime orchestration belongs here
- it keeps several consumer- and auth-specific implementation choices at ADR level until MVP validation closes the remaining product decision gates

**Main tradeoffs accepted:**

- per-request token version checks add some complexity, but they buy immediate revoke
- external knowledge providers add integration work, but they prevent `new-api` from owning an entire RAG stack prematurely
- Default-only admin UI reduces MVP scope and avoids dual-frontend drag

## Implementation Patterns & Consistency Rules

### Pattern Categories Defined

The Agent Platform needs strong implementation rules because three different kinds of resources, two auth planes, and external providers can drift quickly without discipline.

This section defines mandatory patterns for:

- naming
- structure
- format
- communication
- process

### Naming Patterns

**Backend package naming:**

- `model/agentplatform/`
- `service/agentplatform/`
- `controller/agentplatform/`
- `dto/agentplatform/`
- `middleware/agentplatform_*.go`

**Table naming:**

All Agent Platform tables must use the prefix `agent_platform_`.

Examples:

- `agent_platform_resources`
- `agent_platform_resource_versions`
- `agent_platform_clients`
- `agent_platform_client_instances` (post-MVP only, if later activated)

**Route naming:**

- control plane: `/api/agent-platform/**`
- auth plane: `/api/agent-platform/oauth/**`
- open capability plane: `/api/open-capabilities/**`

**Frontend feature naming:**

- `agent-platform-*`

**Query key naming:**

- `['agent-platform', ...]`

**Option / config key naming:**

- `agent_platform.<feature>.<param>`

**i18n key naming:**

- `agentPlatform.clients.*`
- `agentPlatform.skills.*`
- `agentPlatform.knowledge.*`
- `agentPlatform.agents.*`
- `agentPlatform.audit.*`

**Identity naming:**

- `client_id` is the product identifier
- `instance_id` is the device/runtime identifier
- never overload one field to mean both

### Structure Patterns

**Control plane vs open plane separation:**

- control plane controllers do not serve open-capability routes
- open-capability handlers do not read dashboard session auth

**Definition vs exposure separation:**

- definition tables own canonical metadata and versioning
- exposure tables own per-client visibility/callability/freshness
- callers must not infer exposure from definition status alone

**Client vs client_instance separation:**

- product-level policy belongs to `client`
- device/runtime-level revoke and telemetry may belong to `client_instance` only after a later architecture amendment explicitly activates instance identity

**Knowledge provider boundary:**

The provider adapter boundary must stay explicit.

Minimal service contract:

```go
type KnowledgeProvider interface {
    ValidateBinding(ctx context.Context, binding Binding) error
    Query(ctx context.Context, req QueryRequest) (QueryResponse, error)
    Refresh(ctx context.Context, binding Binding) (RefreshState, error)
    Health(ctx context.Context, binding Binding) (HealthState, error)
}
```

Do not let provider-specific raw payloads become the public open-capability contract.

**Agent runtime boundary:**

`Agent` detail may declare:

- prompt/template metadata
- dependencies
- compatible clients
- invocation hints

It may not declare or imply that `new-api` owns:

- multi-step orchestration execution
- runtime session state
- long-running execution state
- server-side workflow ownership in MVP

### Format Patterns

**JSON rules:**

- all JSON business-code operations use `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, or `common.DecodeJson`
- no direct `encoding/json` marshal/unmarshal in business logic

**Payload naming:**

- snake_case JSON fields

**Optional scalar rule:**

All optional scalar request fields that are parsed from JSON and may be re-marshaled must use pointers with `omitempty`.

Example:

```go
type CreateClientRequest struct {
    DisplayName       string   `json:"display_name" validate:"required"`
    Description       *string  `json:"description,omitempty"`
    AccessTokenTTL    *int     `json:"access_token_ttl,omitempty"`
    RefreshTokenTTL   *int     `json:"refresh_token_ttl,omitempty"`
    AllowClientCredentials *bool `json:"allow_client_credentials,omitempty"`
    RedirectURIs      []string `json:"redirect_uris,omitempty"`
}
```

**JWT claim format rule:**

- use standard claims where possible
- custom claims must be stable and documented
- clients must not be required to parse internal-only claims to function

**Extension object rule:**

- client-specific extensions live under `extensions.<namespace>`
- top-level fields remain standardized

### Communication Patterns

**Browser admin traffic:**

- session-authenticated
- talks only to control-plane routes

**Downstream product traffic:**

- bearer-token authenticated
- talks only to open-capability routes

**Provider traffic:**

- server-to-server only
- invoked from service layer adapters
- never directly from browser code

**Publish/revoke signaling:**

- publish writes exposure state and version metadata
- refresh/revoke reconciliation uses exposure freshness + ETag/version comparison
- provider-health changes may downgrade callable state to `provider_offline` without deleting the resource definition

### Process Patterns

**Publish process:**

1. validate resource version
2. validate target client contract compatibility
3. create/update exposure rows
4. write publish task + admin audit
5. expose via discovery/detail

**Authorization process:**

1. browser-based downstream consumer redirects user to `/oauth/authorize`
2. platform ensures user session exists
3. consent is recorded against `client`
4. the selected MVP delegated-auth flow exchanges credentials for tokens
5. downstream calls open-capability routes directly

**Knowledge query process:**

1. authorize token scope and target exposure
2. validate contract compatibility
3. call provider adapter
4. normalize provider output into retrieval result + citations
5. write request/audit diagnostics

**Agent detail process:**

1. authorize read scope
2. resolve agent version
3. include dependency references and compatibility metadata
4. do not execute runtime workflow

### Enforcement Guidelines

**All implementation agents must follow these rules:**

- do not modify relay DTOs or `/v1/**` behavior for Agent Platform work
- do not embed any external knowledge engine into the Go process for MVP
- do not reuse existing relay token auth as Agent Platform bearer auth
- do not let client-specific fields leak into core top-level contract semantics
- do not add PostgreSQL-only storage types or operators
- do not skip audit rows for publish, revoke, token revoke, or client-instance revoke

**Repository rule inheritance:**

- protected `new-api` and `QuantumNous` identifiers remain untouched
- frontend package management under `web/default` uses Bun
- OpenAPI management docs live in `docs/openapi/api.json`

### Pattern Examples

**Route registration example:**

```go
func RegisterAgentPlatformRouter(apiRouter *gin.RouterGroup) {
    ap := apiRouter.Group("/agent-platform")
    ap.Use(middleware.UserAuth())
    {
        clients := ap.Group("/clients")
        clients.Use(middleware.AdminAuth())
        {
            clients.GET("", controllerAP.ListClients)
            clients.POST("", controllerAP.CreateClient)
        }
    }

    oauth := ap.Group("/oauth")
    {
        oauth.GET("/authorize", controllerAP.AuthorizeClient)
        oauth.POST("/token", controllerAP.ExchangeToken)
        oauth.POST("/revoke", controllerAP.RevokeToken)
    }
}
```

**Provider adapter example:**

```go
type QueryResponse struct {
    Version   string              `json:"version"`
    Freshness string              `json:"freshness"`
    Citations []KnowledgeCitation `json:"citations"`
    Chunks    []KnowledgeChunk    `json:"chunks"`
}
```

**Frontend query-key example:**

```ts
useQuery({
  queryKey: ['agent-platform', 'clients', params],
  queryFn: () => api.get('/api/agent-platform/clients', { params }).then(r => r.data.data),
})
```

**Extension example:**

```json
{
  "resource_id": "skill_writer_v1",
  "display_name": "Writer",
  "extensions": {
    "cherry_studio": {
      "category_icon": "pen"
    }
  }
}
```

**Anti-patterns:**

- putting product-level policy onto `client_instance`
- using one `access_token` table row as a long-lived session substitute
- returning provider-native raw output directly as public API contract
- exposing `Agent` as if the platform already owned full workflow runtime
- duplicating the same AP management UI in `web/classic` during MVP

## Project Structure & Boundaries

### Requirements -> Components Mapping

| Domain | PRD Scope | Backend | Frontend | Open Surface |
|---|---|---|---|---|
| Resource control plane | FR-1, FR-2, FR-3 | `model/service/controller/dto/agentplatform` | `agent-platform-skills`, `agent-platform-knowledge`, `agent-platform-agents`, `agent-platform-publishing` | `discovery`, `detail` |
| Client + OAuth + capability standard | FR-4, FR-5, FR-6, FR-15 | `client.go`, `oauth_*.go`, `grant.go`, `refresh_token.go` | `agent-platform-clients` | `/api/agent-platform/oauth/**`, `/api/open-capabilities/**` |
| Skill contract | FR-7, FR-8 | `skill.go`, `skill_invoke.go` | `agent-platform-skills` | `skills/:id/invoke` |
| Knowledge contract | FR-9, FR-10 | `knowledge.go`, `knowledge_provider.go`, `knowledge_query.go` | `agent-platform-knowledge` | `knowledge-bases/:id/query` |
| Agent definition | FR-11, FR-12 | `agent.go`, `agent_dependency.go` | `agent-platform-agents` | `agents/:id` |
| Audit + diagnostics | FR-13, FR-14 | `admin_action.go`, `publish_task.go`, `provider_health.go` | `agent-platform-audit`, `agent-platform-overview` | request/diagnostic metadata on all open responses |

### Complete Project Directory Structure

Only new or directly touched paths are shown:

```text
new-api/
├─ controller/
│  └─ agentplatform/
│     ├─ client.go
│     ├─ oauth.go
│     ├─ open_capabilities.go
│     ├─ skill.go
│     ├─ knowledge.go
│     ├─ agent.go
│     └─ audit.go
├─ dto/
│  └─ agentplatform/
│     ├─ client.go
│     ├─ oauth.go
│     ├─ exposure.go
│     ├─ skill.go
│     ├─ knowledge.go
│     └─ agent.go
├─ middleware/
│  ├─ agentplatform_bearer_auth.go
│  ├─ agentplatform_scope_auth.go
│  └─ agentplatform_client_instance.go
├─ model/
│  └─ agentplatform/
│     ├─ resource.go
│     ├─ resource_version.go
│     ├─ skill_def.go
│     ├─ knowledge_def.go
│     ├─ agent_def.go
│     ├─ client.go
│     ├─ client_instance.go
│     ├─ exposure.go
│     ├─ authorization_grant.go
│     ├─ refresh_token.go
│     ├─ admin_action.go
│     ├─ publish_task.go
│     └─ migration.go
├─ router/
│  ├─ api-router.go                 # register AP routes
│  └─ agentplatform-router.go
├─ service/
│  └─ agentplatform/
│     ├─ client.go
│     ├─ client_instance.go
│     ├─ oauth_authorize.go
│     ├─ oauth_token.go
│     ├─ token_validate.go
│     ├─ exposure.go
│     ├─ discovery.go
│     ├─ skill_invoke.go
│     ├─ knowledge_provider.go
│     ├─ knowledge_query.go
│     ├─ agent_read.go
│     ├─ publish.go
│     ├─ admin_action.go
│     ├─ provider_health.go
│     ├─ scheduler.go
│     └─ errors.go
├─ docs/
│  └─ openapi/
│     └─ api.json                   # add Agent Platform tags and routes
└─ web/
   └─ default/
      └─ src/
         ├─ features/
         │  ├─ agent-platform-overview/
         │  ├─ agent-platform-clients/
         │  ├─ agent-platform-skills/
         │  ├─ agent-platform-knowledge/
         │  ├─ agent-platform-agents/
         │  ├─ agent-platform-publishing/
         │  └─ agent-platform-audit/
         ├─ routes/
         │  └─ _authenticated/
         │     ├─ agent-platform.overview.tsx
         │     ├─ agent-platform.clients.tsx
         │     ├─ agent-platform.skills.tsx
         │     ├─ agent-platform.knowledge.tsx
         │     ├─ agent-platform.agents.tsx
         │     ├─ agent-platform.publishing.tsx
         │     └─ agent-platform.audit.tsx
         └─ i18n/
            └─ locales/
               ├─ en.json
               ├─ zh.json
               ├─ fr.json
               ├─ ru.json
               ├─ ja.json
               └─ vi.json
```

### Architectural Boundaries

**Frozen boundaries:**

- `relay/**` remains out of scope for Agent Platform MVP
- `/v1/**` contract remains out of scope
- existing dashboard session auth remains the control-plane auth source
- existing login/OAuth providers remain upstream identity providers only

**New domain boundaries:**

- `agentplatform` package tree is the new bounded context
- all data-plane bearer auth belongs there
- all provider adapters for Knowledge belong there

**Knowledge boundary:**

The platform owns:

- knowledge resource metadata
- versioning
- publish visibility
- auth
- contract normalization
- health state

The provider owns:

- ingestion
- chunking
- embeddings
- vector/graph indexes
- internal retrieval implementation

**Agent boundary:**

The platform owns:

- agent definition
- dependency declaration
- publish visibility
- read contract

The platform does not own:

- server-side execution runtime
- multi-step orchestration engine
- long-running task execution for agents

### Integration Points

**Internal flow:**

```text
Browser Admin
  -> /api/agent-platform/**
  -> controller/agentplatform/*
  -> service/agentplatform/*
  -> model/agentplatform/*

Downstream Client
  -> /api/agent-platform/oauth/*
  -> obtain bearer token
  -> /api/open-capabilities/**
  -> service/agentplatform/discovery|invoke|query|read

Knowledge Query
  -> service/agentplatform/knowledge_query.go
  -> provider adapter
  -> external HTTP retrieval provider
  -> normalized response
```

**External integrations:**

| External System | Purpose | Boundary |
|---|---|---|
| Existing user login providers (OIDC, GitHub, DingTalk, etc.) | Upstream identity for admin/end-user auth before consent | existing login/session flow |
| Redis | token-version cache, revoke cache, freshness invalidation | optional optimization, not source of truth |
| HTTP retrieval provider candidates (LightRAG, FastGPT, RAGFlow, Haystack service, etc.) | knowledge retrieval backend | external provider adapter |

### File Organization Patterns

**Config:**

- OAuth signing keys come from env/config
- client/provider secrets that vary per tenant/product live in DB-backed config

**Source organization:**

- backend stays layer-first, domain-second
- frontend stays feature-first inside `web/default/src/features`

**Testing organization:**

Expected first-wave test targets:

- `service/agentplatform/oauth_token_test.go`
- `service/agentplatform/token_validate_test.go`
- `service/agentplatform/discovery_test.go`
- `service/agentplatform/knowledge_query_test.go`
- `service/agentplatform/publish_test.go`
- `controller/agentplatform/*_test.go`

**Docs organization:**

- management API changes must update `docs/openapi/api.json`
- this file stays the AP architecture source of truth

### Development Workflow Integration

**Implementation order mandated by this architecture:**

1. scaffold registry, client, grant, and exposure models
2. implement delegated-auth token flow and bearer validation middleware
3. implement discovery/detail for published projections
4. implement Skill invoke and Knowledge query contracts
5. implement Agent detail/dependency contract
6. add publish/revoke/audit UI in `web/default`

**Primary verification commands:**

```bash
go test ./...
go build ./...
cd web/default && bun run typecheck
cd web/default && bun run build
```

**Next BMad artifact dependency:**

Epic and Story generation must reference this document, not the earlier AP draft.

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**

The architecture is internally coherent:

- the monolith deployment choice matches the existing repo and avoids premature service split
- a dedicated data-plane auth model coexists cleanly with existing session auth
- product-level and optional device-level identity concerns are separated without forcing MVP to overcommit before validation
- Knowledge as provider-backed retrieval avoids conflicting responsibilities between control plane and knowledge runtime
- Agent template-only semantics avoid promising a runtime the system does not yet own

**Pattern Consistency:**

- package structure follows current repository conventions
- OpenAPI placement follows current repository policy
- frontend feature naming follows `web/default` feature organization
- JSON and DB compatibility rules are inherited from the repository and explicitly carried forward here

**Structure Alignment:**

- the proposed folders fit the current layered codebase
- no decision requires breaking relay paths or replacing existing auth/session systems
- external provider integrations are isolated behind service adapters

### Requirements Coverage Validation ✅

**Epic/Feature Coverage:**

All PRD feature groups are architecturally supported:

- unified resource control plane
- client registration and capability standard layer
- skill management and invoke
- knowledge management and retrieval
- agent definition and dependency declaration
- audit and diagnostics

**Functional Requirements Coverage:**

- FR-1 to FR-3 are covered by the registry/version/exposure model
- FR-4 to FR-6 and FR-15 are covered by client registration, delegated auth, and explicit freshness/version policy
- FR-7 and FR-8 are covered by Skill typed detail and invoke contract
- FR-9 and FR-10 are covered by Knowledge typed detail and provider-backed retrieval contract
- FR-11 and FR-12 are covered by Agent typed detail and dependency contract
- FR-13 and FR-14 are covered by normalized error envelope, audit tables, publish tasks, and diagnostics

**Non-Functional Requirements Coverage:**

- security: separate auth planes, hashed refresh tokens, revoke support, no secret leakage
- compatibility: GORM + TEXT JSON + repository JSON wrapper
- scalability: same-process MVP with optional Redis and external provider scaling
- observability: request ids, audit records, provider health, token versioning

### Implementation Readiness Validation ✅

**Decision Completeness:**

Critical architectural choices are now explicit:

- auth model
- data model
- route surface
- provider boundary
- deployment mode
- UI scope

**Structure Completeness:**

The architecture names the required models, services, controllers, middleware, frontend features, and OpenAPI integration points.

**Pattern Completeness:**

Consistency rules are explicit for:

- naming
- resource lifecycle
- client vs optional client_instance usage
- JSON pointer semantics
- provider adapters
- auth boundaries

### Gap Analysis Results

**Critical Gaps:**

- none

**Important Gaps:**

- post-MVP grant-type expansion is intentionally deferred beyond the MVP delegated-auth profile
- `client_instance` remains a post-MVP extension point and should not be implemented as a required route surface or persistence model without an architecture amendment

**Nice-to-Have Gaps:**

- publish a provider certification matrix after the first external retrieval-provider integration
- add a dedicated Codex consumer validation appendix after Codex validation lands
- evaluate Classic read-only parity only if demanded by product scope

### Validation Issues Addressed

- The earlier AP draft treated auth only conceptually. This version freezes the MVP delegated-auth profile while still deferring broader grant expansion to future amendments.
- The earlier AP draft recognized Knowledge complexity but did not resolve the boundary. This version freezes a provider-backed retrieval architecture while keeping the first provider choice open.
- The earlier AP draft said `Agent` is not a Cherry Studio adapter layer but did not fully constrain runtime scope. This version freezes `Agent` as definition/template only for MVP.
- The earlier AP draft did not finish the BMad workflow structure. This version completes the full architecture artifact and handoff.

### Architecture Completeness Checklist

**Requirements Analysis**

- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed
- [x] Technical constraints identified
- [x] Cross-cutting concerns mapped

**Architectural Decisions**

- [x] Critical decisions documented with versions
- [x] Technology stack fully specified
- [x] Integration patterns defined
- [x] Performance considerations addressed

**Implementation Patterns**

- [x] Naming conventions established
- [x] Structure patterns defined
- [x] Communication patterns specified
- [x] Process patterns documented

**Project Structure**

- [x] Complete directory structure defined
- [x] Component boundaries established
- [x] Integration points mapped
- [x] Requirements to structure mapping complete

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** high

**Key Strengths:**

- clear product vs device auth boundary without forcing instance identity into MVP
- clean separation between control plane and data plane
- disciplined Knowledge provider boundary
- explicit protection against scope creep into agent runtime orchestration
- strong alignment with current repository constraints

**Areas for Future Enhancement:**

- additional knowledge modes after retrieval-only MVP
- later-consumer hardening for `cc switch` and other post-Codex consumers
- richer provider diagnostics and certification tooling

### Implementation Handoff

**AI Agent Guidelines:**

- follow this document for all Agent Platform work
- treat `_bmad-output/planning-artifacts/architecture.md` as shared platform foundation, not as the AP source of truth
- do not invent alternate auth schemes, alternate route roots, instance-identity surfaces, or embedded knowledge runtimes
- keep Classic parity and agent runtime execution out of MVP unless the architecture is formally amended

**First Implementation Priority:**

Scaffold the core AP bounded context first: resource registry, client, delegated-auth token flow, bearer validation middleware, and discovery/detail. Do not start with UI-only work or provider-specific retrieval-provider wiring before the core control-plane and token model exists.

## V1.4 Architecture Amendment: AP-6 Public Contract Freeze

### Amendment Context

2026-06-02 Correct Course 重处理后，AP-1 到 AP-5 已完成 MVP 基线。AP-6 的架构目标不是重建 control plane / auth plane / open capability plane，而是冻结下游公共契约，并补齐当前签核缺口：enterprise model discovery、错误/状态矩阵、mock fixture、conformance tests 与 onboarding 最小闭环。

### AP-6 Architectural Decisions

**AP6-AD-1: AP-6 is a contract/signoff layer, not a new runtime plane.**

AP-6 不新增第四个 runtime plane。它复用既有：

- control plane: `/api/agent-platform/**`
- auth plane: `/api/agent-platform/oauth/**`
- open capability plane: `/api/open-capabilities/**`

任何新增 API 必须归属到上述平面之一，并说明鉴权方式。

**AP6-AD-2: Enterprise model discovery must be a downstream public projection.**

Enterprise model discovery 不能直接等同于现有 `/api/models`、`/api/user/models` 或 `/v1/models`。

AP-6 必须定义一个面向下游消费者的模型投影契约，至少冻结：

- `modelId`
- `providerStableId`
- `displayName`
- `isDefault`
- `status`
- `disabledReason`
- `capabilities`
- `accountId` / `tenantId` 来源语义

该契约可以复用现有模型元数据、用户可用模型和 channel 状态作为数据来源，但对外必须是 Agent Platform 下游公共契约，而不是 dashboard 管理接口或 relay 兼容接口的直接暴露。

**AP6-AD-3: Public contract source of truth is docs + OpenAPI + fixtures.**

AP-6 签核不得只依赖 markdown 说明。公共契约的 source of truth 为三件套：

1. `docs/agent-platform-downstream-contract-spec.md`
2. `docs/openapi/api.json`
3. mock / fixture / conformance test artifacts

三者不一致时，AP-6 story 不得标记为 done。

**AP6-AD-4: Error and client-state matrices are contract artifacts.**

当前 open capability error envelope 继续保留，但 AP-6 必须冻结平台错误码到客户端状态的映射矩阵。

至少覆盖平台错误：

- `permissionDenied`
- `resourceRevoked`
- `resourceOffline`
- `quotaOrRateLimited`
- `timeout`
- `upstreamFailed`
- `contractInvalid`

至少覆盖客户端状态：

- `loginExpired`
- `empty`
- `loadFailed`
- `networkFailed`
- `noAssignedResource`
- `visibleButNotCallable`
- `stale`
- `revoked`
- `offline`

**AP6-AD-5: Mock / fixture artifacts must not contain secrets or tenant data.**

Fixture 可以包含代表性 payload，但不得包含真实 token、provider secret、tenant secret、真实用户信息或真实企业数据。所有 fixture 必须使用 synthetic IDs。

**AP6-AD-6: Cherry Studio and Codex validate contract reuse.**

Cherry Studio 作为 first consumer，可以通过 `extensions.cherry_studio` 承载私有展示提示，但不得修改核心字段语义。

Codex 作为 second consumer，必须验证不需要新建平行协议主干。若 Codex 需要扩展，应通过：

- `contract_version`
- backward-compatible optional fields
- `extensions.codex`

完成治理。

### AP-6 Implementation Handoff

AP-6 实施顺序：

1. 冻结 downstream contract spec。
2. 更新 OpenAPI。
3. 生成 mock / fixture。
4. 增加 conformance tests。
5. 根据规格缺口最小化补代码。
6. 完成 Cherry Studio signoff。
7. 完成 Codex second-consumer review。

AP-6 不得修改：

- `relay/**`
- `/v1/**`
- `docs/openapi/relay.json`
- AP-1 到 AP-5 的完成状态

## Workflow Completion

### Workflow Status

- Architecture workflow: complete
- Document status: complete
- This file is now the formal BMad architecture artifact for the Agent Platform line

### Deliverables Produced

- complete BMad architecture document
- frozen auth model for downstream token access
- frozen Knowledge provider strategy
- frozen Agent runtime boundary
- implementation structure and handoff guidance

### Mandatory Next Step

Use `bmad-create-epics-and-stories` against this document and the AP PRD.

### Recommended Next Commands

```bash
# Review available BMad guidance
bmad-help

# Continue into implementation planning
# Use the AP PRD + this architecture artifact as the only source of truth
```

### Ongoing Maintenance Rule

Any future change to:

- auth grant types
- token semantics
- knowledge modes
- agent runtime ownership
- route surface roots

must be treated as an architecture amendment to this file before implementation changes are merged.
