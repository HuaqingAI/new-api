---
baseline_commit: 07d86a0d
---

# Story 2.4: 执行钉钉部门与成员同步

Status: done

## Story

As a 管理员,
I want 手动或定时同步钉钉部门与成员,
So that 本地组织结构和成员关系能跟随钉钉更新。

## Acceptance Criteria

1. Given 钉钉配置可用, When 管理员触发全量同步, Then 系统创建同步任务并返回任务状态, And 前端不长时间阻塞请求，可查看任务进度和结果。
2. Given 同步任务执行完成, When 管理员查看同步结果, Then 系统展示新增、更新、停用、失败、跳过数量, And 同步日志可按时间、状态和对象类型筛选。
3. Given 同一钉钉部门或成员被重复同步, When 同步任务重试或定时任务再次运行, Then 系统幂等更新，不重复创建部门、用户或成员关系, And 失败项记录错误原因且不回滚已成功的无关对象。

## Tasks / Subtasks

- [x] 新增同步任务与日志模型 (AC: 1, 2)
  - [x] 新增 `enterprise_dingtalk_sync_tasks` 保存任务状态、进度、计数器、错误摘要和时间戳。
  - [x] 新增 `enterprise_dingtalk_sync_logs` 保存对象类型、外部 ID、动作、状态、消息和时间。
  - [x] 迁移纳入 `model/enterprise.AutoMigrate`，兼容 SQLite/MySQL/PostgreSQL。
- [x] 扩展 DingTalk client 通讯录读取 (AC: 1, 3)
  - [x] 支持列出子部门、分页列出部门用户、按用户 ID 获取用户详情。
  - [x] 测试可注入 fake client，不依赖真实钉钉环境。
- [x] 实现全量同步 service (AC: 1, 2, 3)
  - [x] `StartFullSync` 创建任务并返回任务状态，支持后台执行和测试用 inline 执行。
  - [x] 按 `sync_scope` 起点递归同步部门与成员；空 scope 默认从根部门 `1` 开始。
  - [x] 幂等 upsert 钉钉部门、本地用户、钉钉身份绑定和部门成员关系。
  - [x] 同步后停用本轮未出现的钉钉部门与成员关系。
  - [x] 单个对象失败写日志并累计失败数，不回滚其他已成功对象。
- [x] 新增 Root-only API 与审计 (AC: 1, 2)
  - [x] 新增 `POST /api/enterprise/dingtalk/sync/full`。
  - [x] 新增 `GET /api/enterprise/dingtalk/sync/tasks/:id`。
  - [x] 新增 `GET /api/enterprise/dingtalk/sync/logs`，支持 task、status、object_type 和时间分页筛选。
  - [x] 启动同步写入 `enterprise_admin_actions`，payload 不包含凭据或 token。
- [x] Default 前端同步入口 (AC: 1, 2)
  - [x] 在 DingTalk Integration 页新增 Address Book Sync 区块。
  - [x] 支持启动全量同步、刷新最近日志、查看任务计数器和失败摘要。
  - [x] 新增 `en/zh/fr/ja/ru/vi` 文案。
- [x] 测试与验证 (AC: 1, 2, 3)
  - [x] service 测试覆盖首次同步、幂等重跑、停用 stale 记录、失败项日志和部分成功不回滚。
  - [x] 前端 SSR 测试覆盖同步面板任务计数和日志脱敏。

## Dev Notes

- 本故事不实现冲突待处理状态、重命名历史追溯增强、离职/转部门细粒度冲突规则；这些属于 Story 2.5。
- 同步创建的成员关系统一写入 `external_source=dingtalk`，供 Story 2.3 登录路径只读本地快照。
- API 和日志只返回稳定摘要，不暴露 app secret、access token 或 DingTalk 原始敏感响应。

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Started from Story 2.4 branch `codex/story-2-4`, based on Story 2.3 branch `codex/story-2-3` at `07d86a0d`.
- Validation passed: `go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api ./controller`.
- Validation passed after installing Default frontend dependencies with `/Users/hq-it/.bun/bin/bun install --frozen-lockfile`: `cd web/default && /Users/hq-it/.bun/bin/bun run i18n:sync && /Users/hq-it/.bun/bin/bun run typecheck`.
- Validation passed: `cd web/default && /Users/hq-it/.bun/bin/bun run test:e2e` with a non-fatal optional `supports-color` warning from `debug`.

### Completion Notes List

- Added DingTalk full sync task/log models, API, service, and frontend control panel.
- Added idempotent department/user/membership upsert for DingTalk sync.
- Added stale DingTalk department and membership deactivation after each full sync.
- Added sync task counters and filterable logs for progress/result visibility.
