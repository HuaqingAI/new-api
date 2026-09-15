---
title: 'GH-32 提权旧会话恢复与企业钱包只读/状态纠偏'
type: 'bugfix'
created: '2026-07-06'
status: 'done'
baseline_commit: '3a1dad0880b31b9985dceced5c375f869b13db22'
context:
  - '{project-root}/AGENTS.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** Issue #32 仍有 3 个稳定回归：提权后旧会话不能在下一次受保护请求中直接恢复管理员权限；enterprise allocation 钱包仍可从通用订阅管理入口作废；钱包页对 enterprise allocation 的来源和 `end_time <= 0` 语义仍会误判。

**Approach:** 让受保护请求和 `getSelf()` 都以数据库当前用户事实为准并刷新旧 session；把 enterprise wallet 在通用订阅入口收敛为只读保护；统一展示层对 enterprise allocation 的来源、状态与无正过期时间语义。

## Boundaries & Constraints

**Always:** `AdminAuth`/`UserAuth` 必须按数据库当前 `role`、`status`、`group` 判定并刷新旧 session；用户更新必须真实写入非 Root 角色变更；enterprise wallet 在通用订阅管理入口只能是只读语义，不能变成另一条治理写路径；enterprise allocation 且 `status='active'`、`end_time <= 0` 时必须保持 active/长期有效；普通订阅与企业治理账务链路不得被顺带改坏；后端继续保持跨 SQLite/MySQL/PostgreSQL 兼容。

**Ask First:** 如果现网 enterprise wallet 来源值不是 `enterprise_allocation`；如果要扩大角色编辑边界；如果需要把通用入口改成直接跳企业治理页面。

**Never:** 不要求重新登录才让提权生效；不在通用作废 API 内偷偷转调企业治理取消；不把所有普通订阅的 `end_time <= 0` 一律解释为永不过期；不改 enterprise allocation 的账务/审计事实模型。

</frozen-after-approval>

## Code Map

- `middleware/auth.go` -- 旧 session 是否按数据库当前事实恢复权限。
- `controller/user.go`, `model/user.go` -- `GetSelf` 权限快照与角色持久化。
- `web/default/src/routes/_authenticated/route.tsx` -- 受保护路由的前端会话刷新。
- `controller/subscription.go`, `model/subscription.go` -- enterprise wallet 的通用作废保护。
- `web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx`, `web/default/src/features/wallet/components/subscription-plans-card.tsx` -- enterprise wallet 的只读按钮、来源、状态、有效期展示。

## Tasks & Acceptance

**Execution:**
- [x] `model/user.go`, `controller/user.go`, `middleware/auth.go`, `web/default/src/routes/_authenticated/route.tsx` -- 让角色更新真正落库，且受保护请求与 `getSelf()` 都使用数据库最新用户事实 -- 修复提权后旧会话仍被当成普通用户或被跳回登录页。
- [x] `model/subscription.go`, `controller/subscription.go`, `web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx` -- enterprise wallet 的通用作废/删除入口统一为只读受保护语义，不再代办治理取消 -- 保持企业治理入口是唯一写入口。
- [x] `web/default/src/features/wallet/components/subscription-plans-card.tsx` -- 统一 enterprise allocation 的来源识别、状态判定和 `end_time <= 0` 展示语义 -- 修正“未知来源/active 却 expired”。
- [x] `controller/user_self_test.go`、邻近鉴权测试、`controller/subscription_invalidate_enterprise_test.go`、`web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.test.tsx`、`web/default/src/features/wallet/components/subscription-plans-card.test.tsx` -- 补回归覆盖提权旧会话恢复、enterprise wallet 通用入口保护和无正过期时间语义。

**Acceptance Criteria:**
- Given 已登录普通用户被管理员提升为管理员且未重新登录, when 下一次访问 `AdminAuth` 资源并触发受保护路由校验, then 请求按数据库新角色通过，session 与本地 auth 缓存同步为管理员状态，且不会跳登录页。
- Given 管理员查看某用户的订阅列表且其中包含 enterprise allocation 钱包, when 渲染操作列或直调通用 invalidate API, then UI 只显示只读受保护语义，API 返回受保护错误，allocation / wallet 生命周期事实不变。
- Given 钱包页存在 enterprise allocation 且 `end_time <= 0`, when 页面渲染该记录, then active 记录显示企业分配与长期有效语义，cancelled 记录显示 Cancelled 与无 epoch 时间语义，且普通订阅既有语义保持兼容。

## Spec Change Log

## Verification

**Commands:**
- `go test ./controller -run 'TestGetSelf|TestAdminInvalidateUserSubscription'` -- expected: 提权权限快照与 enterprise wallet 通用入口保护回归通过
- `cd web/default && bun test src/features/subscriptions/components/dialogs/user-subscriptions-dialog.test.tsx src/features/wallet/components/subscription-plans-card.test.tsx` -- expected: enterprise wallet 来源/状态/按钮文案回归通过
- `cd web/default && bun run build` -- expected: Default 前端类型检查与打包通过

## Suggested Review Order

**Session Truth**

- Entry point refreshes stale session role, status, and group from the database.
  [`auth.go:36`](../../middleware/auth.go#L36)

- User edits now persist role changes instead of silently dropping them.
  [`user.go:539`](../../model/user.go#L539)

- Self payload now computes permissions from persisted role, not stale context.
  [`user.go:434`](../../controller/user.go#L434)

- Protected route refreshes user state but avoids logging out on transient failures.
  [`route.tsx:24`](../../web/default/src/routes/_authenticated/route.tsx#L24)

**Enterprise Wallet Guards**

- Normalize source-only and source_type-based enterprise wallets before write protection.
  [`subscription.go:339`](../../model/subscription.go#L339)

- Invalidate and delete both block enterprise wallets through the common admin API.
  [`subscription.go:1251`](../../model/subscription.go#L1251)

- Controller now returns a protected message instead of silently delegating cancellation.
  [`subscription.go:433`](../../controller/subscription.go#L433)

- Admin dialog uses the same enterprise detection for labels and disabled actions.
  [`user-subscriptions-dialog.tsx:100`](../../web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx#L100)

- Source-only enterprise rows are also treated as read-only in the actions column.
  [`user-subscriptions-dialog.tsx:391`](../../web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.tsx#L391)

**Wallet Presentation**

- Card title and managed note now honor source-only enterprise wallets too.
  [`subscription-plans-card.tsx:121`](../../web/default/src/features/wallet/components/subscription-plans-card.tsx#L121)

- Status and expiry helpers share the normalized enterprise detection path.
  [`subscription-plans-card.tsx:145`](../../web/default/src/features/wallet/components/subscription-plans-card.tsx#L145)

**Regression Coverage**

- Middleware test proves promoted users regain admin access without re-login.
  [`auth_test.go:20`](../../middleware/auth_test.go#L20)

- Controller test proves self permissions come from database role snapshots.
  [`user_self_test.go:52`](../../controller/user_self_test.go#L52)

- Model tests cover source-only enterprise wallets for invalidate and delete guards.
  [`subscription_enterprise_wallet_test.go:467`](../../model/subscription_enterprise_wallet_test.go#L467)

- Dialog tests lock source-only enterprise labels and localized source rendering.
  [`user-subscriptions-dialog.test.tsx:59`](../../web/default/src/features/subscriptions/components/dialogs/user-subscriptions-dialog.test.tsx#L59)

- Wallet tests lock source-only enterprise title, note, and non-positive expiry semantics.
  [`subscription-plans-card.test.tsx:84`](../../web/default/src/features/wallet/components/subscription-plans-card.test.tsx#L84)
