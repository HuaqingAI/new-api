---
baseline_commit: 29d99f67
---

# Story 2.3: 支持钉钉 OAuth 登录与账号绑定

Status: done

## Story

As a 企业员工,
I want 使用钉钉登录系统,
So that 我可以用企业身份进入 new-api 并获得正确部门归属。

## Acceptance Criteria

1. Given 钉钉登录已启用且员工在允许范围内, When 员工完成钉钉 OAuth 回调, Then 系统绑定已有账号或按配置创建新账号, And 登录成功后复用现有 session、`setupLogin`、`UserAuth/AdminAuth/RootAuth`。
2. Given 最近一次成功同步中存在该员工部门成员关系快照, When 员工钉钉登录成功, Then 系统使用本地快照读取该员工全部部门成员关系, And 不在登录路径直接写入钉钉回调返回的部门列表。
3. Given 员工被停用或不在允许同步范围内, When 员工尝试钉钉登录, Then 系统拒绝登录并给出原因, And 不影响已有非钉钉登录方式。

## Tasks / Subtasks

- [x] 新增钉钉身份绑定模型与迁移 (AC: 1)
  - [x] 新增 `enterprise_dingtalk_identities`，包含 `tenant_id`、`corp_id`、`identity_key`、`union_id`、`open_id`、`external_user_id`、`user_id`、`status` 和时间戳。
  - [x] 使用 `identity_key` 避免空 `union_id` 在跨数据库唯一索引中冲突。
  - [x] 表名保持 `enterprise_` 前缀并预留 `tenant_id`。
- [x] 扩展 DingTalk client 与 OAuth service (AC: 1, 2, 3)
  - [x] 支持 OAuth code 换 user access token、读取 `/v1.0/contact/users/me`，并用 app token 解析 unionId 对应的通讯录 `userid`。
  - [x] 登录 service 支持已绑定身份登录、唯一邮箱安全绑定、注册开启时创建新用户。
  - [x] 已登录用户回调走显式绑定当前账号。
  - [x] 停用员工、停用绑定、本地停用用户、安全冲突、未启用登录、provider 失败均返回稳定错误。
- [x] 使用本地成员快照验证范围 (AC: 2, 3)
  - [x] 登录成功路径读取 `enterprise_user_departments` 中 `external_source=dingtalk` 的活跃成员关系。
  - [x] 登录路径不创建、不更新、不停用部门成员关系。
  - [x] 已解析到通讯录 `external_user_id` 但本地没有快照时拒绝为不在允许范围。
- [x] 新增 OAuth 回调 API 与状态暴露 (AC: 1, 3)
  - [x] 新增 `GET /api/oauth/dingtalk`，放在 `/api/oauth/:provider` 通配路由之前。
  - [x] 回调复用现有 OAuth state session 校验。
  - [x] 登录成功复用 `setupLogin`，因此后续仍走既有 `UserAuth/AdminAuth/RootAuth`。
  - [x] `/api/status` 在钉钉登录启用时返回 `dingtalk_oauth`、client id、callback URL 和授权端点。
- [x] Default 前端登录入口 (AC: 1)
  - [x] 登录页 OAuth provider 列表新增 “Continue with DingTalk”。
  - [x] 使用 `/api/oauth/state` 和配置回调地址构造钉钉 OAuth URL。
  - [x] 新增 DingTalk brand icon 与 `en/zh/fr/ja/ru/vi` 翻译。
- [x] 测试与验证 (AC: 1, 2, 3)
  - [x] service 测试覆盖已绑定登录、本地快照读取、唯一邮箱绑定、邮箱冲突、停用员工、停用绑定、范围拒绝、新用户创建、显式绑定当前账号。
  - [x] controller 测试覆盖 state 校验、登录、绑定和范围错误映射。
  - [x] 前端 typecheck、i18n sync 和 E2E 测试通过。

## Dev Notes

- 本故事不实现通讯录同步任务、同步进度、同步日志或冲突处理 UI；这些属于 Story 2.4 和 Story 2.5。
- 登录路径只使用 DingTalk 回调身份确认员工身份，部门归属读取本地最近同步快照，不从回调响应写入成员关系。
- 自动绑定仅在本地邮箱唯一且未软删除时进行；重复邮箱或软删除邮箱按冲突拒绝。
- 响应和错误不暴露 app secret、access token 或 DingTalk 原始敏感响应。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 2.3 branch `codex/story-2-3`, based on Story 2.2 branch `codex/story-2-2` at `29d99f67`.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api ./controller`.
- Validation passed after installing Default frontend dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run test:e2e` with a non-fatal optional `supports-color` warning from `debug`.
- Validation passed: `git diff --check`.
- Forbidden path scan clean for relay, billing expression, cachex, core log, relay router, and relay OpenAPI paths.

### Completion Notes List

- Added DingTalk identity binding table and OAuth login/bind service.
- Added DingTalk OAuth callback route that reuses existing session and `setupLogin`.
- Added local DingTalk membership snapshot read for login scope validation without mutating membership state.
- Added Default frontend DingTalk OAuth login button and translations.
- Added focused backend tests for binding, scope, disabled employee, conflict, and callback behavior.
