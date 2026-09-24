---
baseline_commit: 8a280638
---

# Story 2.2: 测试钉钉连通性与权限

Status: done

## Story

As a Root 管理员,
I want 测试钉钉应用连通性和通讯录权限,
So that 我能在启用前发现配置或权限问题。

## Acceptance Criteria

1. Given 已保存钉钉配置, When 管理员执行连通性测试, Then 系统返回 `auth_success`、`auth_invalid_credentials`、`auth_permission_insufficient`、`network_unreachable` 或 `callback_misconfigured` 之一, And 前端按错误码展示对应引导文案。
2. Given 钉钉接口超时或返回错误, When 连通性测试结束, Then 系统记录摘要错误原因, And 不把 app secret、access token 或原始敏感响应写入日志。

## Tasks / Subtasks

- [x] 新增钉钉 OpenAPI client 抽象 (AC: 1, 2)
  - [x] 支持可注入 HTTP client 与 base URL，便于用 fake server 覆盖 token、权限、网络失败场景。
  - [x] token 获取调用不把 `access_token` 透传给上层响应。
  - [x] 通讯录权限探测使用低影响部门子列表接口，仅返回稳定错误摘要。
- [x] 实现连通性测试 service 与 DTO (AC: 1, 2)
  - [x] 输出固定 code 集合：`auth_success`、`auth_invalid_credentials`、`auth_permission_insufficient`、`network_unreachable`、`callback_misconfigured`。
  - [x] 缺失配置/凭据、回调 URL 无效、DingTalk token 错误、权限错误、网络错误均映射为稳定 code。
  - [x] 响应只包含 `code`、`stage`、`summary`、`http_status`、`checked_at` 等摘要字段。
- [x] 新增 Root-only API 与审计 (AC: 1, 2)
  - [x] 新增 `POST /api/enterprise/dingtalk/connectivity-test`。
  - [x] 成功完成测试后写入 `enterprise_admin_actions`，payload 只记录稳定 code、阶段、摘要和 HTTP 状态。
  - [x] 审计不保存 app secret、access token 或 DingTalk 原始敏感响应。
- [x] Default 前端测试入口与引导 (AC: 1)
  - [x] 在 DingTalk Integration 页新增 “Test connection” 按钮。
  - [x] 按五个稳定 code 显示对应处理建议。
  - [x] 未保存配置或表单有未保存变更时禁用测试并提示先保存。
  - [x] 新增前端多语言文案并同步 `en/zh/fr/ja/ru/vi` locale。
- [x] 测试与验证 (AC: 1, 2)
  - [x] service fake server 测试覆盖成功、凭据错误、通讯录权限不足、网络失败和回调配置错误。
  - [x] controller/API 测试覆盖响应和审计脱敏。
  - [x] router/API 测试覆盖 Root 权限。
  - [x] 前端 SSR 测试覆盖五个稳定 code 的引导文案。

## Dev Notes

- 本故事不实现 OAuth 登录、账号绑定、通讯录同步任务或冲突处理；这些属于 Story 2.3 至 Story 2.5。
- `DingTalkClient` 作为后续 Story 2.3/2.4 的共享基础，生产默认 base URL 为 DingTalk OpenAPI，测试可注入 fake server。
- 连通性测试以稳定 code 作为前后端契约，不向前端或审计暴露 DingTalk 原始 `errmsg`。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 2.2 branch `codex/story-2-2`, based on Story 2.1 branch `codex/story-2-1` at `8a280638`.
- Validation passed: `go test ./service/enterprise ./controller/enterprise ./tests/api`.
- Validation passed after installing Default frontend dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run test:e2e` with a non-fatal optional `supports-color` warning from `debug`.

### Completion Notes List

- Added Root-only DingTalk connectivity test API with stable response codes.
- Added injectable DingTalk OpenAPI client and connectivity service.
- Added sanitized enterprise admin audit records for connectivity tests.
- Added Default frontend “Test connection” workflow and per-code guidance.
- Added backend and frontend tests for connectivity code mapping and redaction.
