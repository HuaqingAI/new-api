---
title: '修复优先订阅耗尽后无法回退钱包'
type: 'bugfix'
created: '2026-07-10'
status: 'done'
baseline_commit: '40877b06c0e490a94bdb238b4d8acdeee0b7eafa'
context:
  - '{project-root}/_bmad-output/implementation-artifacts/investigations/subscription-wallet-fallback-investigation.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** `subscription_first` 已逐一尝试所有活跃订阅，但全部订阅无法承担请求时，任一订阅的 `allow_wallet_overflow=false` 会全局阻断钱包回退，导致用户在钱包额度充足时仍收到订阅额度不足错误。

**Approach:** 移除 `subscription_first` 初始预扣额度不足分支中的全局溢出许可门槛，订阅额度不足时直接尝试钱包；保留其他偏好和非额度错误的现有控制流，并以确定性服务层测试锁定契约。

## Boundaries & Constraints

**Always:** `subscription_first` 必须先让订阅模型依现有排序逐一尝试全部活跃订阅；仅在订阅返回额度不足错误后尝试钱包；钱包尝试的成功或失败结果必须成为最终结果；测试使用显式固定数据并断言资金来源、错误类型及余额变化；保持 SQLite、MySQL、PostgreSQL 兼容；使用 `testify/require` 和 `testify/assert`。

**Ask First:** 若实现需要改变初始预扣以外的 Reserve/Settle/Refund 行为、订阅排序或合并规则、数据库结构、公开 API、前端行为，必须暂停并征得确认。

**Never:** 不得让单条 `allow_wallet_overflow=false` 全局阻止 `subscription_first` 回退；不得让 `subscription_only` 使用钱包；不得改变 `wallet_first` 或 `wallet_only`；不得把计划查询、额度重置、数据库读写、token 额度等非订阅额度错误转换成钱包回退；不得修改无关代码或顺手修复既有企业订阅测试夹具。

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| 多订阅不足后钱包成功 | `subscription_first`；多条活跃订阅均无法单独承担预扣，其中一条禁止溢出；钱包充足 | 返回 wallet 会话，只扣钱包，订阅用量不变 | 无错误 |
| 多订阅与钱包均不足 | `subscription_first`；全部订阅不足；钱包也不足 | 钱包被实际尝试，订阅和钱包余额均不变 | 返回钱包额度不足的 403 `insufficient_user_quota`，不得返回订阅不足文案 |
| 仅订阅模式不足 | `subscription_only`；订阅不足；钱包充足 | 不创建会话，不扣钱包 | 返回订阅额度不足的 403 `insufficient_user_quota` |
| 非额度类订阅错误 | `subscription_first`；活跃订阅的计划解析产生确定性非额度错误；钱包充足 | 不尝试钱包，余额不变 | 保留 `update_data_error` 和 500 |

</frozen-after-approval>

## Code Map

- `service/billing_session.go` -- `NewBillingSession` 根据计费偏好选择订阅或钱包；当前缺陷位于 `subscription_first` 的额度不足回退分支。
- `model/subscription.go` -- `PreConsumeUserSubscription` 已排序并遍历全部活跃订阅；其非额度错误会立即返回，全部不足才产生额度不足错误，本次不修改。
- `service/task_billing_test.go` -- 服务包共享 SQLite 夹具、用户/订阅种子与现有 `NewBillingSession` 测试所在文件。

## Tasks & Acceptance

**Execution:**
- [x] `service/billing_session.go` -- 将 `subscription_first` 的订阅额度不足处理收敛为直接调用现有钱包路径，同时原样返回所有非额度错误；不触碰其他偏好分支。
- [x] `service/task_billing_test.go` -- 增加四个确定性回归测试，分别验证钱包成功回退、钱包不足成为最终错误、`subscription_only` 禁止回退、非额度错误禁止回退；复用现有普通订阅夹具，避免引入企业订阅依赖。

**Acceptance Criteria:**
- Given 多个活跃订阅均不足且至少一条 `allow_wallet_overflow=false`、钱包充足，when 以 `subscription_first` 创建计费会话，then 会话资金来源为 wallet、钱包扣除预扣额且订阅用量不变。
- Given 所有活跃订阅均不足且钱包也不足，when 以 `subscription_first` 创建计费会话，then 返回钱包额度不足的 403 错误且所有余额不变。
- Given 活跃订阅不足但钱包充足，when 以 `subscription_only` 创建计费会话，then 返回订阅额度不足且钱包不发生扣减。
- Given 订阅预扣产生非额度类错误且钱包充足，when 以 `subscription_first` 创建计费会话，then 原样返回非额度错误且不使用钱包。
- Given 修复和测试已完成，when 对改动文件执行 `gofmt` 并运行定向 Go 测试，then 格式检查无差异且全部契约测试通过。

## Spec Change Log

## Design Notes

`PreConsumeUserSubscription` 已完成“依次尝试所有活跃订阅”的职责；修复应留在偏好编排层。额度不足由 `BillingSession.preConsume` 映射为 `ErrorCodeInsufficientUserQuota`，因此 `subscription_first` 只需在该错误码上调用 `tryWallet()`。钱包不足自然覆盖此前的订阅错误，非额度错误继续原样返回。

## Verification

**Commands:**
- `gofmt -w service/billing_session.go service/task_billing_test.go` -- expected: 改动 Go 文件格式化完成。
- `go test ./service -run '^(TestNewBillingSessionSubscriptionFirstFallsBackToWallet|TestNewBillingSessionSubscriptionFirstReturnsWalletInsufficient|TestNewBillingSessionSubscriptionOnlyDoesNotFallbackToWallet|TestNewBillingSessionSubscriptionFirstDoesNotFallbackOnNonQuotaError)$' -count=1` -- expected: 四个产品契约回归测试全部通过。
- `git diff --check` -- expected: 无空白错误。

## Suggested Review Order

**资金源回退**

- 额度不足只切换钱包，其他订阅错误保持原样。
  [`billing_session.go:415`](../../service/billing_session.go#L415)

**契约回归测试**

- 锁定 strict 订阅存在时仍可成功回退钱包。
  [`task_billing_test.go:438`](../../service/task_billing_test.go#L438)

- 确认钱包不足成为最终错误且所有余额不变。
  [`task_billing_test.go:498`](../../service/task_billing_test.go#L498)

- 保护 `subscription_only` 永不使用钱包。
  [`task_billing_test.go:561`](../../service/task_billing_test.go#L561)

- 以计划解析失败保护非额度错误不回退。
  [`task_billing_test.go:607`](../../service/task_billing_test.go#L607)

**后续工作**

- 单独记录既有企业订阅测试夹具缺陷。
  [`deferred-work.md:9`](deferred-work.md#L9)
