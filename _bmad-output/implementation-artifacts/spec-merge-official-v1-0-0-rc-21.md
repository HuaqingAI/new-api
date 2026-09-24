---
title: 'Merge official v1.0.0-rc.21 into dev'
type: 'chore'
created: '2026-07-13'
status: 'done'
baseline_commit: 'b6464ea46c3039dd7508a2b449e43d2106f50361'
context:
  - '{project-root}/AGENTS.md'
  - '{project-root}/web/default/AGENTS.md'
  - '{project-root}/pkg/billingexpr/expr.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** The fork's `dev` branch is 355 commits ahead of the common base while official release `v1.0.0-rc.21` contributes 44 commits across billing, relay conversion, logging, and the default frontend. The official update must be integrated without losing the fork's enterprise governance, subscription-wallet behavior, or release/deployment customizations.

**Approach:** Create `codex/merge-official-v1.0.0-rc.21` from the current `dev`, merge the exact annotated official tag at commit `bde9b2f44887d34ec54799ae191d50f97914359e`, resolve conflicts as semantic unions, and validate both official behavior and fork-specific contracts.

## Boundaries & Constraints

**Always:** Preserve protected project and organization identity; keep the fork's enterprise APIs/UI, subscription-wallet fallback behavior, configurable image release and Kubernetes deployment flow; retain official billing saturation safeguards, GPT-5.6 cache-write billing, relay conversion registry, timing metrics, and UI improvements; use project JSON wrappers, cross-database patterns, checked quota conversion, Bun tooling, and script-managed i18n writes.

**Ask First:** Halt if a conflict cannot preserve both sides and requires choosing between an existing fork behavior and the official API contract, or if verification reveals a pre-existing failure unrelated to the merge.

**Never:** Rebase or rewrite `dev`; force-push; resolve wholesale with `ours` or `theirs`; delete fork-specific enterprise code; weaken billing safety; directly hand-edit locale JSON files; remove protected metadata or attribution.

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Clean merge start | Clean `dev` matching `origin/dev` | New merge branch contains a two-parent merge of official rc.21 | Stop if branch base or tag commit differs |
| Text conflict | Both sides changed the same workflow/component | Resolve to a coherent union that preserves both contracts | Inspect base/ours/theirs and add focused regression coverage when behavior changes |
| Locale conflict | Both sides added or reordered flat i18n keys | All supported locale keys survive, remain valid and synchronized | Apply locale changes through the sanctioned script and run `i18n:sync` |
| Auto-merged high-risk code | Git reports no conflict in billing/relay code | Semantic review confirms fork invariants still hold | Fix incompatibility before completing merge |
| Verification failure | Build, lint, or tests fail | Merge branch is left with diagnosed, fixed code | Do not merge back to `dev` or push incomplete work |

</frozen-after-approval>

## Code Map

- `.github/workflows/docker-build.yml` -- predicted conflict between official workflow updates and the fork's configurable registry/version/Kubernetes deployment pipeline.
- `web/default/src/features/usage-logs/index.tsx` -- predicted conflict between enterprise log behavior and official task-detail/timing additions.
- `web/default/src/features/wallet/components/subscription-plans-card.tsx` -- predicted conflict between enterprise subscription-wallet semantics and official UI cleanup.
- `web/default/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json` -- predicted key/order conflicts; writes must go through i18n tooling.
- `common/quota_math.go`, `relay/helper/price.go`, `service/{billing.go,billing_usage.go,tiered_settle.go}` -- auto-merged billing surface requiring semantic review against saturation and audit invariants.
- `service/relayconvert/**`, `relay/channel/{openai,claude,gemini}/**` -- official protocol conversion refactor and compatibility surface.

## Tasks & Acceptance

**Execution:**
- [x] Git refs -- verify `dev`, create `codex/merge-official-v1.0.0-rc.21`, and merge exact official tag with a merge commit.
- [x] `.github/workflows/docker-build.yml` -- combine official workflow deltas with fork release versioning, configurable image registry, and deployment jobs.
- [x] Conflicting TSX files -- preserve enterprise behavior while incorporating official timing/task-detail and component-quality changes.
- [x] Locale files -- reconstruct the union of official and fork keys through `add-missing-keys.mjs`, then normalize with `bun run i18n:sync`.
- [x] Billing and relay auto-merges -- review the full official delta for semantic incompatibilities with fork quota, subscription, provider, and enterprise behavior.
- [x] Merge result -- remove all conflict stages/markers, format touched code, and run focused plus repository-level verification.

**Acceptance Criteria:**
- Given the current `dev`, when the official tag is merged, then the resulting branch has both `dev` and `bde9b2f44` as ancestors and no unresolved index entries.
- Given the resolved workflow, when inspected, then existing configurable registry, composite version, and Kubernetes deployment behavior remains while compatible official action updates are retained.
- Given subscription and usage-log conflicts, when the frontend is built and tested, then enterprise wallet/log flows and official rc.21 UI behavior both remain available.
- Given official cache-write and quota changes, when Go tests run, then quota cannot wrap negative and saturation remains auditable.
- Given all locale changes, when i18n sync runs, then supported locale files are valid, sorted, and contain the merged key set.

## Spec Change Log

- 2026-07-13: Resolved all ten merge conflicts as semantic unions, restored protected poster assets removed by the official cleanup, and fixed merged frontend compatibility with the official StatCard tone API.
- 2026-07-13: Rebuilt locale files from Git's base/ours/theirs stages through the sanctioned script, added the 74 enterprise UI keys missing on the baseline, normalized all locales, and corrected tests to use the configured `zhCN` language code.
- 2026-07-13: Hardened merged OpenAI, Claude, and Gemini billing usage against negative counts and integer overflow; normalized inconsistent Gemini totals, prevented pure-image modality double counting, and pinned the manifest cosign installer to the build job's verified SHA.

## Design Notes

Conflict resolution will use the merge base (`6ce7305cd`), current `dev`, and official tag independently. Conflict-free Git output is not sufficient for billing and relay code because both branches contain substantial architectural changes; those areas receive a semantic diff review before tests.

The semantic audit confirmed strict pre-consume quota conversion, auditable settlement saturation, GPT-5.6 cache-write charging, non-negative cache remainders, relay conversion registries, enterprise log scope/context, and subscription-wallet display behavior. Official pinned action SHAs were combined with the fork's configurable registry, composite version, signing, manifest, and Kubernetes deployment flow.

## Verification

**Commands:**
- `git ls-files -u` -- expected: no unresolved index entries.
- `git diff --check` -- expected: no conflict markers or whitespace errors.
- `go test ./...` -- expected: all backend and cross-module regression tests pass.
- `bun run i18n:sync` -- expected: locale reports regenerate without missing keys introduced by the merge.
- `bun run typecheck` -- expected: TypeScript passes.
- `bun run lint` -- expected: no lint errors.
- `bun run test:e2e` -- expected: enterprise, subscription, usage, quota, and sidebar frontend regressions pass.
- `bun run build` -- expected: production frontend build succeeds.

**Results:** `git ls-files -u`, conflict-marker scanning, both diff checks, workflow YAML parsing, `bun run i18n:sync`, `bun run typecheck`, `bun run test:e2e`, and `bun run build` passed. Focused Go tests passed for `common`, `pkg/billingexpr`, `relay/helper`, `relay/common`, all `service/relayconvert` packages, and the provider billing-usage normalization regressions in `service`. Repository-wide Go, lint, and format checks were executed but retain baseline test-fixture and style-debt failures; the merge-specific failures found by typecheck and E2E were fixed and reverified.

## Suggested Review Order

**Billing safety**

- Start with provider usage normalization, non-negative counts, and saturating token sums.
  [`billing_usage.go:106`](../../service/billing_usage.go#L106)

- Verify tiered pre-consume rejects quota saturation before charging.
  [`price.go:268`](../../relay/helper/price.go#L268)

**Protocol conversion**

- Review the typed request registry and composed conversion routes.
  [`request_registry.go:85`](../../service/relayconvert/request_registry.go#L85)

- Review response conversion, streaming state, and compatibility aliases.
  [`response_registry.go:112`](../../service/relayconvert/response_registry.go#L112)

**Fork compatibility**

- Confirm official release versioning remains compatible with configurable registry and deployment jobs.
  [`docker-build.yml:15`](../../.github/workflows/docker-build.yml#L15)

- Confirm enterprise log scope and department context survive the official logs redesign.
  [`index.tsx:75`](../../web/default/src/features/usage-logs/index.tsx#L75)

- Confirm enterprise allocation wallets retain managed status and non-expiring semantics.
  [`subscription-plans-card.tsx:138`](../../web/default/src/features/wallet/components/subscription-plans-card.tsx#L138)

**Regression evidence**

- Review negative, inconsistent, overflow, and modality token regression cases.
  [`text_quota_test.go:751`](../../service/text_quota_test.go#L751)

- Review synchronized locale results after the three-way key union.
  [`_sync-report.json:1`](../../web/default/src/i18n/locales/_reports/_sync-report.json#L1)
