---
workflowStatus: 'draft'
totalSteps: 5
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
nextStep: ''
lastSaved: '2026-05-28'
inputDocuments:
  - '_bmad-output/implementation-artifacts/sprint-status.yaml'
  - '_bmad-output/planning-artifacts/epics.md'
  - '_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md'
  - '_bmad-output/planning-artifacts/architecture.md'
---

# Test Design: Epics 1-5 - 企业管控基线

**Date:** 2026-05-28  
**Author:** hth  
**Status:** Draft

## Executive Summary

**Scope:** Epic-Level test design for the enterprise control baseline covering Epic 1 through Epic 5: enterprise organization, DingTalk identity/sync, department budget wallet allocation, organization usage reporting, and content risk alerting.

**Risk Summary:**

- Total risks identified: 12
- High-priority risks (score >= 6): 9
- Blockers (score = 9): 2
- Critical categories: DATA, SEC, OPS, PERF, TECH

**Coverage Summary:**

- P0 scenarios: 9 groups, ~35-60 hours
- P1 scenarios: 8 groups, ~30-55 hours
- P2/P3 scenarios: 8 groups, ~20-42 hours
- **Total effort:** ~85-155 hours, roughly ~2-4 weeks depending on framework readiness and DB matrix automation

## Not In Scope

| Item | Reasoning | Mitigation |
| --- | --- | --- |
| SaaS multi-tenant isolation | PRD marks V1 as single enterprise/self-deployed with `tenant_id` reserved only. | Test reserved field presence, not runtime tenant isolation. |
| Feishu/WeCom identity providers | Architecture defers secondary enterprise identity sources. | Keep provider abstraction testable; DingTalk is the V1 tested source. |
| Relay-path department deduction | V1.1 explicitly forbids department logic in relay/billing path. | Add static guards and regression tests to prove relay behavior is unchanged. |
| Full Classic frontend parity | Architecture says Default is the complete delivery surface; Classic provides necessary entry points. | Cover Classic with smoke checks only unless product requires parity. |
| Real DingTalk production credentials in automated CI | External credentials make CI brittle and can leak secrets. | Use provider mocks/fakes; keep sandbox tests manual/on-demand. |

## Risk Assessment

### High-Priority Risks (Score >= 6)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner | Timeline |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| R-001 | DATA | Department budget parent/child invariants can break under concurrent allocation, revoke, expiry, or reset. | 3 | 3 | 9 | Transactional integration and concurrency tests with invariant assertions across DBs. | Backend Lead + QA | Before Epic 3 RC |
| R-002 | SEC | DingTalk secrets, webhook secrets, tokens, or sensitive risk data may leak through API/UI/log/export/error paths. | 2 | 3 | 6 | Redaction tests at controller, log, export, UI payload, and error boundaries. | Security + Backend Lead | Before Epic 2/5 merge |
| R-003 | DATA | N:N department membership may regress into a single/primary department assumption. | 3 | 3 | 9 | Shared multi-department fixtures and assertions across org, budget, usage, and risk suites. | Backend Lead + QA | Before Epic 1 complete |
| R-004 | PERF | Department usage summary may miss P95 < 3s for 30-day view with 1M logs and <=100 departments. | 2 | 3 | 6 | k6 or Go benchmark with generated data, query evidence, and snapshot-table checks. | Backend Lead + DevOps | Before Epic 4 RC |
| R-005 | TECH | Enterprise work can accidentally modify relay DTOs, relay routes, billing expression variables, or Group semantics. | 2 | 3 | 6 | Static boundary guards, existing relay/billing regression, OpenAPI placement check. | Architect + Backend Lead | Every enterprise PR |
| R-006 | OPS | Background jobs can duplicate work or fail silently in multi-node or no-Redis modes. | 2 | 3 | 6 | Job state-machine, `common.IsMasterNode`, idempotency, retry, and no-Redis fallback tests. | Backend Lead + DevOps | Per job-owning epic |
| R-007 | SEC | Department admin access could rely on frontend visibility or broad Admin role rather than server-side department scope. | 2 | 3 | 6 | API authorization matrix for Root/Admin/department admin/ordinary user. | Backend Lead + Security | Before affected epics release |
| R-008 | BUS | Multi-department duplicated attribution may be misread if disclaimers are missing from dashboards, risk views, or CSV. | 3 | 2 | 6 | Mandatory disclaimer assertions in component/API/E2E and CSV first-line checks. | Frontend Lead + QA | Before Epic 4/5 release |
| R-009 | DATA | DingTalk sync can auto-bind the wrong account or mutate member/wallet state incorrectly after org changes. | 2 | 3 | 6 | Mocked sync scenarios for stable identity, conflicts, pending state, and wallet non-mutation. | Backend Lead + QA | Before Epic 2 release |

### Medium-Priority Risks

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner |
| --- | --- | --- | --- | --- | --- | --- | --- |
| R-010 | OPS | No E2E framework config exists yet, so UI workflows may be under-tested. | 2 | 2 | 4 | API/component tests first; add Playwright before broad UI E2E. | QA Lead |
| R-011 | TECH | Enterprise frontend i18n keys may drift or remain untranslated. | 2 | 2 | 4 | `bun run i18n:sync`, locale key checks, mandatory disclaimer key test. | Frontend Lead |
| R-012 | DATA | CSV exports can emit wrong order, `null`, missing "未归属", or inconsistent sorting. | 2 | 2 | 4 | File-level API tests for header, comment row, sorting, empty strings, and unassigned row. | QA + Backend Lead |

### Risk Category Legend

- **TECH:** Architecture, integration, maintainability, compatibility boundaries
- **SEC:** Authentication, authorization, secret handling, data exposure
- **PERF:** Latency, load, scalability, query performance
- **DATA:** Data integrity, consistency, correctness, migration behavior
- **BUS:** Business interpretation and user-facing decision risk
- **OPS:** Operational reliability, jobs, retries, monitoring, environment behavior

## NFR Planning

This is a planning view for later `nfr-assess`; it does not assign final PASS/CONCERNS/FAIL from implementation evidence.

| NFR Category | Requirement / Threshold | Risk Link | Planned Validation | Evidence Needed |
| --- | --- | --- | --- | --- |
| Compatibility | SQLite, MySQL 5.7.8+, PostgreSQL 9.6+ support for new migrations and queries. | R-001, R-003, R-005 | DB matrix integration tests. | Migration and query test logs for all three DBs. |
| Security | Secrets not returned/logged/exported; department scope enforced server-side. | R-002, R-007 | API negative tests, log/export scans, authorization matrix. | Test reports and redaction assertions. |
| Privacy | Risk events expose only trace summary and obey existing log desensitization. | R-002 | API response and ordinary-user denial tests. | Response snapshots and negative assertions. |
| Reliability | Sync, aggregation, report, wallet sync, expiry, and alert jobs expose state/errors and are idempotent. | R-006, R-009 | Job state-machine and retry tests. | State table assertions and retry/final-failure logs. |
| Performance | 30-day department usage first screen P95 < 3s with 1M logs and <=100 departments. | R-004 | k6 or Go benchmark. | Benchmark/k6 report and query evidence. |
| Consistency | `Σ children.committed_quota <= parent.total_quota/cycle_quota`; wallet consumption remains consistent. | R-001 | Concurrency and ledger consistency tests. | Invariant assertion report. |
| Observability | Sync, allocation, aggregation, and alert delivery have queryable status and error reasons. | R-006 | API/integration checks for admin-visible state. | Admin API/state-table evidence. |
| Internationalization | New default frontend text enters locale files with existing sync flow. | R-008, R-011 | Static i18n checks. | i18n sync output and locale key report. |

**Unknown thresholds:**

- Hardware baseline for P95 < 3s is UNKNOWN.
- Expected concurrent dashboard/report users are UNKNOWN.
- DingTalk sync retry schedule is UNKNOWN unless finalized during implementation.
- Exact sensitive-content redaction policy for risk events is UNKNOWN beyond "summary only".
- Classic theme translation depth is UNKNOWN; plan smoke-level coverage unless product requires parity.

## Entry Criteria

- PRD, epics, architecture, and sprint status are available.
- Test DB environments or containers are available for SQLite, MySQL, and PostgreSQL.
- DingTalk provider fake/mocking strategy is available before Epic 2 tests.
- Budget and usage data factories can seed users, departments, memberships, subscriptions, logs, and risk events.
- Frontend test runner decision is made before broad E2E coverage; component/static coverage can start earlier.

## Exit Criteria

- All P0 tests pass.
- P1 pass rate is >=95%, with triaged owner and dated mitigation for any failure.
- No open score-9 risk remains.
- All score >=6 risks are mitigated or formally waived with owner and expiry.
- Security and secret-redaction scenarios pass 100%.
- Three-database compatibility evidence exists before enterprise migrations are enabled by default.
- Performance evidence addresses P95 < 3s or records a documented `nfr-assess` CONCERNS decision.

## Test Coverage Plan

P0/P1/P2/P3 below are risk priorities, not execution timing.

### P0 Critical

| Test ID | Requirement / Scenario | Test Level | Risk Link | Count | Notes |
| --- | --- | --- | --- | --- | --- |
| ENT-ORG-P0-001 | N:N membership, no primary department, and "未归属" users remain usable. | API + Unit | R-003 | 6-10 | Shared fixtures for organization, usage, budget, and risk. |
| ENT-ORG-P0-002 | Enterprise migrations and core queries work on SQLite/MySQL/PostgreSQL. | Integration | R-001, R-005 | 6-9 | Include reserved-name and DB-specific SQL checks. |
| ENT-BUDGET-P0-001 | Parent budget invariant holds under >=50 concurrent allocation requests. | Integration | R-001 | 5-8 | Assert no negative parent, duplicate wallet, duplicate ledger. |
| ENT-BUDGET-P0-002 | Revoke, expiry, and subscription reset are idempotent and consistent. | Integration | R-001, R-006 | 6-10 | Repeat jobs and compare parent/wallet/allocation state. |
| ENT-BUDGET-P0-003 | Existing relay, Group, billing expression, and user subscription behavior remain unchanged. | Unit + Integration | R-005 | 4-7 | Reuse existing billing/relay regression and add static guard. |
| ENT-SEC-P0-001 | API authorization matrix rejects ordinary users and out-of-scope department admins. | API | R-007 | 8-12 | Root/Admin/department admin/ordinary user coverage. |
| ENT-SEC-P0-002 | Secrets never appear in API, UI payload, logs, CSV, or errors. | API + Integration | R-002 | 6-10 | Negative assertions at representative boundaries. |
| ENT-DING-P0-001 | DingTalk binding avoids unsafe auto-bind and uses last successful membership snapshot. | API + Integration | R-009 | 6-9 | Mock stable identity, email/phone conflict, and no snapshot cases. |
| ENT-RISK-P0-001 | Risk events persist department snapshot and hide admin-only fields from ordinary users. | API + Integration | R-002, R-003 | 5-8 | Multi-department and unassigned cases required. |

### P1 High

| Test ID | Requirement / Scenario | Test Level | Risk Link | Count | Notes |
| --- | --- | --- | --- | --- | --- |
| ENT-DING-P1-001 | Connectivity test returns stable auth, permission, network, and callback error codes. | API | R-002, R-009 | 5-7 | Mock upstream timeout and error bodies. |
| ENT-DING-P1-002 | Full sync is idempotent and records counts, failure items, and filterable logs. | Integration | R-006, R-009 | 6-10 | Include partial failure without rollback of successful objects. |
| ENT-USAGE-P1-001 | Usage aggregation duplicates multi-department consumption and isolates unassigned users. | Integration | R-003, R-008 | 6-9 | Assert department totals are intentionally non-additive. |
| ENT-USAGE-P1-002 | Dashboard/detail/risk overview always show `enterprise.usage.multi_dept_disclaimer`. | Component + E2E smoke | R-008, R-011 | 4-6 | Component first; E2E after framework exists. |
| ENT-USAGE-P1-003 | CSV export preserves filter/sort, fixed columns, disclaimer, empty strings, and "未归属". | API + File validation | R-008, R-012 | 5-8 | Validate file content and row ordering. |
| ENT-ALERT-P1-001 | Alert rules validate recipients/webhooks and audit changes without secrets. | API | R-002, R-006 | 5-8 | Include invalid recipient and secret redaction. |
| ENT-ALERT-P1-002 | Alert delivery retries, final failure is queryable, and manual resend creates a new delivery. | Integration | R-006 | 5-8 | Use fake clock or controllable scheduler. |
| ENT-FE-P1-001 | Default enterprise modules use scoped query keys, invalidation, route guards, and i18n keys. | Component + Static | R-007, R-011 | 5-8 | Classic gets smoke-level coverage. |

### P2 Medium

| Test ID | Requirement / Scenario | Test Level | Risk Link | Count | Notes |
| --- | --- | --- | --- | --- | --- |
| ENT-ORG-P2-001 | Department tree shows 3+ levels, disabled/deleted states, sync anomaly, and empty array. | API + Component | R-003 | 4-7 | Include empty/loading/error UI states. |
| ENT-AUDIT-P2-001 | Admin actions list supports pagination and filters without sensitive fields. | API | R-002 | 3-5 | Allocation ledger remains its own audit source. |
| ENT-BUDGET-P2-001 | Budget list/detail sorting and warning/urgent threshold validation. | API + Component | R-001 | 4-7 | Verify `0 < warning < urgent <= 100`. |
| ENT-REPORT-P2-001 | Scheduled report validates recipients and records send failures without affecting dashboards. | API + Integration | R-006 | 4-6 | Use email fake/spool. |
| ENT-I18N-P2-001 | Enterprise frontend copy syncs across locale files and mandatory disclaimer key exists. | Static | R-011 | 2-4 | `bun run i18n:sync` plus key check. |

### P3 Low / Exploratory

| Test ID | Requirement / Scenario | Test Level | Count | Notes |
| --- | --- | --- | --- | --- |
| ENT-EXPL-P3-001 | Browser exploratory pass for enterprise navigation, table density, and visual coherence. | E2E exploratory | 2-4 | Run after frontend routes exist. |
| ENT-PERF-P3-001 | Extended background-job soak beyond acceptance windows. | Long-running | 1-3 | Weekly/on-demand only. |
| ENT-REPORT-P3-001 | Optional webhook/DingTalk robot alert channel smoke if enabled post-MVP. | API smoke | 1-3 | Out of V1 mandatory scope unless product decides otherwise. |

## Execution Strategy

Run everything in PRs when the suite stays under ~15 minutes. Defer only expensive, external, or long-running work.

| Cadence | Scope |
| --- | --- |
| PR | Go unit tests, enterprise-focused integration tests without large datasets, frontend typecheck/lint/static i18n checks, component tests where configured, relay/core boundary guard. |
| Nightly | Three-database matrix if too heavy for PR, usage performance baseline, CSV/report validation, alert/job retry tests with controlled scheduler. |
| Weekly / On-demand | Long job soak, large concurrency burn-in, browser exploratory passes, and any real DingTalk sandbox tests. |

## Resource Estimates

| Priority | Scope | Estimated QA/Test Dev Effort |
| --- | --- | --- |
| P0 | Data integrity, security, authz, budget concurrency, relay boundary guards | ~35-60 hours |
| P1 | DingTalk sync, usage aggregation/export, alert delivery, main frontend checks | ~30-55 hours |
| P2 | Admin audit, UI states, i18n, scheduled reports, sorting/threshold edges | ~15-30 hours |
| P3 | Exploratory, soak, optional channels | ~5-12 hours |
| Total | Enterprise control baseline | ~85-155 hours |

## Quality Gate Criteria

- P0 pass rate: 100%.
- P1 pass rate: >=95%.
- P2/P3 failures are informational unless they expose a score >=6 risk.
- No open score-9 risk at release.
- All high risks (score >=6) mitigated or formally waived.
- Security and secret-redaction tests pass 100%.
- Critical path coverage target: >=80%.
- NFR evidence source exists for each in-scope NFR category before final QA sign-off.
- Full NFR evidence decision is deferred to `nfr-assess` after implementation evidence exists.

## Mitigation Plans

### R-001: Budget Invariant Corruption

**Strategy:** Build DB-backed concurrency tests for allocation, expand, revoke, expiry, and subscription reset. Assert parent pool, child wallet, and allocation ledger after each operation and after repeated jobs.  
**Owner:** Backend Lead + QA  
**Timeline:** Before Epic 3 release candidate  
**Status:** Planned  
**Verification:** P0 integration suite passes on the supported DB matrix.

### R-003: N:N Membership Semantic Drift

**Strategy:** Standardize multi-department and unassigned fixtures, then require all organization, budget, usage, and risk tests to use them. Add negative assertions against primary-department assumptions.  
**Owner:** Backend Lead + QA  
**Timeline:** Before Epic 1 complete  
**Status:** Planned  
**Verification:** P0 organization/usage/risk tests pass with multi-department fixtures.

### R-002/R-007: Security And Authorization Exposure

**Strategy:** Add API-level authz matrix tests and secret redaction tests before UI coverage. Treat frontend visibility as secondary only.  
**Owner:** Security + Backend Lead  
**Timeline:** Before affected enterprise endpoints merge  
**Status:** Planned  
**Verification:** API matrix and redaction tests pass 100%.

### R-004/R-006: Performance And Job Reliability

**Strategy:** Use generated data and controllable schedulers/fake clocks. Capture benchmark output, job state transitions, retry counts, and final failure states.  
**Owner:** Backend Lead + DevOps  
**Timeline:** Before Epic 4/5 release candidates  
**Status:** Planned  
**Verification:** Nightly performance/job suites produce retained artifacts.

## Assumptions And Dependencies

### Assumptions

1. Enterprise APIs will use `/api/enterprise/**` and current `{ success, message, data }` response conventions.
2. DingTalk can be represented by a deterministic fake for CI; real sandbox tests remain optional/on-demand.
3. Default frontend is the primary UI delivery surface; Classic receives necessary smoke coverage.
4. New enterprise modules can add test factories without refactoring unrelated relay/provider code.

### Dependencies

1. DB matrix environment or containers for SQLite/MySQL/PostgreSQL.
2. Data factories for departments, memberships, users, subscriptions, logs, allocations, and risk events.
3. Fake DingTalk provider and controllable scheduler/fake clock utilities.
4. Decision on frontend E2E framework setup before broad UI workflow coverage.

### Risks To Plan

- **Risk:** Performance target lacks hardware/concurrency baseline.  
  **Impact:** P95 evidence may be disputed even if tests are automated.  
  **Contingency:** Record baseline environment in benchmark artifact and raise `nfr-assess` CONCERNS if unresolved.

- **Risk:** No E2E framework is configured today.  
  **Impact:** UI journey automation may lag API/backend coverage.  
  **Contingency:** Start with component/static coverage and add Playwright before enterprise navigation workflow coverage.

## Interworking & Regression

| Service / Component | Impact | Regression Scope |
| --- | --- | --- |
| `relay/**` | Must remain department-unaware. | Existing relay tests plus static guard for department terms in relay DTOs/routes. |
| `pkg/billingexpr/**` and subscription billing | Department wallet uses existing subscription deduction semantics. | Billing expression and tiered settlement regression tests. |
| `model/log.go` / usage logs | Usage aggregation reads logs but must not change core log semantics. | Existing log behavior plus enterprise aggregation tests. |
| `users`, `tokens`, `Group` semantics | Department membership must not alter Group behavior. | User/group regression and P0 boundary tests. |
| Default frontend | New enterprise management surface. | Typecheck, lint/static i18n, component tests, targeted E2E smoke. |
| Classic frontend | Necessary management entry points only. | Smoke checks for access and basic rendering. |

## Appendix

### Knowledge Base References

- `risk-governance.md`
- `probability-impact.md`
- `test-levels-framework.md`
- `test-priorities-matrix.md`
- `nfr-criteria.md`
- `contract-testing.md`
- `playwright-cli.md`

### Related Documents

- PRD: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`
- Epics: `_bmad-output/planning-artifacts/epics.md`
- Architecture: `_bmad-output/planning-artifacts/architecture.md`
- Sprint status: `_bmad-output/implementation-artifacts/sprint-status.yaml`

**Generated by:** BMad TEA Agent - Test Architect Module  
**Workflow:** `bmad-testarch-test-design`  
**Version:** 4.0 (BMad v6)
