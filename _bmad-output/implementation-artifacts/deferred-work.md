# Deferred Work

## 2026-07-06

- Enterprise wallet reorder / billing-priority semantics appear to have drifted outside GH-32 scope in `model/subscription.go` and `service/task_billing_test.go`. Review separately to decide whether department-managed wallets should remain movable and whether `subscription_first` may legally consume ordinary subscriptions ahead of enterprise allocation wallets.
- `model/enterprise/migration.go` currently includes a startup data-normalization path for department budget remaining/quota values that was not part of GH-32 approval. Review as a separate data-governance change before it ships with an unrelated bugfix.

## 2026-07-10

- `TestNewBillingSessionSubscriptionFirstHonorsReorderedEnterpriseWalletOrder` in `service/task_billing_test.go` cannot run because the shared service test fixture does not create or seed `enterprise_quota_allocations`. Repair the enterprise subscription fixture in a separate change; this predates and is outside the subscription-first wallet fallback hotfix.

## 2026-07-13

- Repository-wide Go tests still depend on shared SQLite and permission fixtures that do not consistently initialize abilities, top-ups, tasks, subscriptions, system tasks, or enterprise API state. The same controller, middleware, model, service, and `tests/api` failures reproduce on `dev@b6464ea46`; repair the shared fixtures in a focused test-infrastructure change.
- Repository-wide frontend lint and format checks report broad pre-existing style debt across fork and upstream files. Normalize that debt separately so future release merges can use full-repository lint/format as blocking gates without obscuring merge-specific regressions.
