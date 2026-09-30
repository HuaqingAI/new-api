---
baseline_commit: a1025acf
---

# Story 2.5: 处理同步冲突与组织变更

Status: done

## Story

As a 管理员,
I want 处理钉钉账号匹配冲突和组织变更,
So that 错绑账号、部门变更和离职不会污染本地治理数据。

## Acceptance Criteria

1. Given 钉钉成员无法通过稳定身份自动匹配本地账号, When 邮箱或手机号存在冲突, Then 系统将该成员标记为待处理冲突, And 不自动绑定到可能错误的账号。
2. Given 钉钉部门改名、移动、停用或删除, When 同步任务处理变更, Then 本地部门状态和父子关系按规则更新, And 历史用量仍可按旧部门名称或名称快照追溯。
3. Given 成员离职或转部门, When 同步任务处理成员变更, Then 系统更新成员关系状态, And 不静默删除或篡改已分配 wallet 子钱包，需标记待处理或交由预算回收策略处理。

## Tasks / Subtasks

- [x] 新增同步冲突记录模型 (AC: 1)
  - [x] 新增 `enterprise_dingtalk_sync_conflicts`，记录外部用户、邮箱、手机号、候选用户、冲突类型和 pending/resolved/ignored 状态。
  - [x] 迁移纳入 `model/enterprise.AutoMigrate`，保持 SQLite/MySQL/PostgreSQL 兼容。
  - [x] `enterprise_dingtalk_identities` 补充 `mobile` 快照字段，用于后续同步/OAuth 的手机号冲突判断。
- [x] 扩展同步安全匹配规则 (AC: 1)
  - [x] 稳定身份 `identity_key` 命中时继续更新既有绑定。
  - [x] 未命中稳定身份时，如果邮箱命中本地用户或手机号命中既有钉钉身份，则写入 pending conflict。
  - [x] 冲突成员跳过自动创建用户、身份绑定和部门成员关系，并写 `conflict_pending` warning 日志。
- [x] 处理组织变更留痕 (AC: 2, 3)
  - [x] 部门改名时将旧名称追加到 `Department.NameHistory`，用于历史用量按名称快照追溯。
  - [x] 部门移动继续通过 `parent_id` 更新，停用/删除继续以本轮未见的钉钉部门标记 disabled/warning。
  - [x] 本轮未见的钉钉成员关系改为 `EnterpriseMembershipStatusLeft` 并写 `left_at`，不删除历史关系或 wallet 相关数据。
- [x] 新增 Root-only 查询和前端可见性 (AC: 1)
  - [x] 新增 `GET /api/enterprise/dingtalk/sync/conflicts`，支持 status 和分页查询。
  - [x] DingTalk Integration 页展示待处理冲突列表，提示冲突成员不会自动绑定。
  - [x] 新增 `en/zh/fr/ja/ru/vi` 前端文案。
- [x] 测试与验证 (AC: 1, 2, 3)
  - [x] service 测试覆盖邮箱冲突、手机号冲突、部门改名/移动历史、离职/转部门成员关系状态。
  - [x] controller 测试覆盖冲突列表 API。
  - [x] 前端 SSR 测试覆盖冲突列表渲染和敏感信息不泄漏。

## Dev Notes

- 本故事只提供冲突检测与只读可见性，不实现人工 resolved/ignored 操作；后续若需要人工绑定，应另开 story 并补审计动作。
- 手机号冲突基于 `enterprise_dingtalk_identities.mobile` 快照；老身份数据没有手机号时不会误判，新的 OAuth/同步会持续刷新。
- 成员离职/转部门只更新企业成员关系状态，不触碰 wallet 或 relay/billing 链路，给 Epic 3 的预算回收策略保留单独处理空间。
- 同步冲突写入 warning 日志并累计 skipped，不把可人工处理的数据当作系统失败。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 2.5 branch `codex/story-2-5`, based on Story 2.4 branch `codex/story-2-4` at `a1025acf`.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api ./controller`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun install --frozen-lockfile`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run test:e2e` with the known non-fatal optional `supports-color` warning from `debug`.

### Completion Notes List

- Added pending DingTalk sync conflict detection and listing.
- Prevented unsafe auto-binding when email or mobile conflicts exist.
- Added department name history append on DingTalk rename.
- Marked stale DingTalk memberships as left with `left_at`, preserving history for future wallet recovery policies.
