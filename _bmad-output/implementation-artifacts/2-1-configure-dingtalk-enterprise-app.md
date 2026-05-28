---
baseline_commit: 2fc5f36f14d34646fd6e78e42f8e2a92fe17a682
---

# Story 2.1: 配置钉钉企业应用

Status: done

## Story

As a Root 管理员,
I want 配置钉钉应用凭据、回调地址和同步范围,
So that 系统可以安全接入企业钉钉身份源。

## Acceptance Criteria

1. Given Root 管理员打开钉钉配置页面, When 填写 app key、app secret、corp id、回调地址、同步范围和启用开关, Then 系统保存配置并隐藏敏感字段, And API 响应、前端状态、日志和导出文件中不出现明文密钥。
2. Given 缺失必要凭据或配置格式错误, When 管理员保存或启用钉钉登录/同步, Then 系统拒绝启用并返回可区分错误, And 错误不透传钉钉原始敏感信息。

## Tasks / Subtasks

- [x] 新增钉钉配置模型与迁移 (AC: 1)
  - [x] 新增 `enterprise_dingtalk_configs` 专用表，字段包含 `tenant_id`、`corp_id`、`app_key`、`app_secret`、`callback_url`、`sync_scope`、启用开关和时间戳。
  - [x] `app_secret` 在 model JSON 输出中屏蔽，并通过配置响应的 `has_app_secret` 只暴露是否已保存。
  - [x] 在企业模块迁移中注册，保持三库兼容。
- [x] 实现安全配置 service、DTO 与 API (AC: 1, 2)
  - [x] 新增 `GET /api/enterprise/dingtalk/config` 和 `PUT /api/enterprise/dingtalk/config`。
  - [x] API 使用 `RootAuth()`，普通管理员不能保存钉钉企业应用凭据。
  - [x] 保存时支持空 `app_secret` 保留旧密钥；启用登录或同步时要求 `corp_id`、`app_key`、已保存或本次提供的 `app_secret`、有效 HTTPS 回调地址。
  - [x] 缺失凭据和回调地址格式错误返回不同 i18n 错误码。
- [x] 接入企业审计 (AC: 1)
  - [x] 成功保存配置后写入 `enterprise_admin_actions`。
  - [x] 审计 payload 复用统一敏感字段净化，确保 `app_key`、`app_secret` 等不落入明文审计。
- [x] Default 前端配置页 (AC: 1, 2)
  - [x] 新增 `enterprise-dingtalk` feature、路由和侧边栏入口。
  - [x] 新增钉钉配置表单，展示密钥是否已保存，但不回填已保存 secret。
  - [x] 组织空状态的 “Configure DingTalk sync” 入口跳转到新配置页。
  - [x] 新增前端多语言文案并同步 `en/zh/fr/ja/ru/vi` locale。
- [x] 测试与验证 (AC: 1, 2)
  - [x] service 测试覆盖密钥保存/保留、启用校验、回调地址校验和草稿配置。
  - [x] controller/API 测试覆盖响应脱敏、Root 权限、错误码区分和审计脱敏。
  - [x] 运行 Go 测试、Default i18n sync/typecheck 和 diff hygiene。

## Dev Notes

- 本故事不实现钉钉连通性测试、OAuth 登录、通讯录同步或冲突处理；这些分别属于 Story 2.2 至 Story 2.5。
- 钉钉配置使用专用 `enterprise_dingtalk_configs` 表，不复用通用 option API，避免通用设置读取路径暴露敏感凭据。
- `has_app_secret` 是安全状态字段，不是明文密钥；API 不返回 `app_secret` 字段。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 2.1 branch `codex/story-2-1`, based on pushed Epic 1 aggregate branch `codex/story-1-4` at `2fc5f36f`.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api`.
- Validation passed after installing Default frontend dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync`; sync report shows zero missing/extras/untranslated for `en/zh/fr/ja/ru/vi`.
- Validation passed: `git diff --check`.
- Boundary scan passed: no changed paths under `relay/**`, `pkg/billingexpr/**`, `pkg/cachex/**`, `model/log.go`, `controller/log.go`, `router/relay-router.go`, or `docs/openapi/relay.json`.

### Completion Notes List

- Added Root-only DingTalk enterprise app configuration API backed by `enterprise_dingtalk_configs`.
- API responses return config plus `has_app_secret`, never the plaintext secret.
- Enabling DingTalk login/sync now rejects missing credentials and invalid callback URLs with distinct i18n errors.
- Added Default frontend DingTalk Integration page with credential-safe form handling and sidebar/empty-state navigation.
- Added service, controller, and router-level tests for secret masking, validation, permissions, and audit redaction.
