# Deferred Work

## 2026-07-06

- Enterprise wallet reorder / billing-priority semantics appear to have drifted outside GH-32 scope in `model/subscription.go` and `service/task_billing_test.go`. Review separately to decide whether department-managed wallets should remain movable and whether `subscription_first` may legally consume ordinary subscriptions ahead of enterprise allocation wallets.
- `model/enterprise/migration.go` currently includes a startup data-normalization path for department budget remaining/quota values that was not part of GH-32 approval. Review as a separate data-governance change before it ships with an unrelated bugfix.
