---
baseline_commit: 10217e3746e30faf699e1905ec5fbca15ef98fb6
---

# Story 4.3: 查看部门用量详情

Status: done

## Story

As a 管理员,
I want 进入单个部门查看用户排行、模型分布和趋势,
so that 我可以定位某部门主要 AI 消耗来源。

## Acceptance Criteria

1. **Given** 管理员打开某部门详情  
   **When** 选择时间范围  
   **Then** 页面展示该部门用户排行、模型分布、时间趋势和最近消费日志入口  
   **And** 用户排行可按 `quota`、请求数、`token` 排序。
2. **Given** 同一用户属于多个部门  
   **When** 管理员查看任一部门详情  
   **Then** 该用户可出现在多个部门排行中  
   **And** 每个部门按本部门口径独立计算，不跨部门加和。
3. **Given** 管理员点击最近消费日志入口  
   **When** 跳转到日志页面  
   **Then** 系统带入部门、时间范围和用户筛选条件  
   **And** 不修改现有 `model/log.go` 或 `controller/log.go` 的核心日志语义。

## Tasks / Subtasks

- [x] 扩展 enterprise usage 后端查询，提供部门详情聚合接口 (AC: 1, 2, 3)
  - [x] 在 `dto/enterprise/usage.go` 中新增 `department-detail` 所需的响应结构，至少覆盖用户排行、模型分布、时间趋势和最近消费日志入口所需的筛选上下文；数组字段默认输出 `[]`，可选标量继续保持指针类型。[Source: _bmad-output/planning-artifacts/architecture.md#State Management Patterns; controller/enterprise/usage.go]
  - [x] 在 `service/enterprise/usage_aggregation.go` 增加单部门详情查询能力，基于 `enterprise_usage_snapshots` 与现有成员关系/日志只读链路聚合用户排行、模型分布和趋势；不得回退为直接扫描全量 `logs` 作为默认查询路径。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14; docs/enterprise/usage-aggregation.md]
  - [x] 同一用户多部门重复计入语义必须在详情层保持一致：部门详情里的用户排行只按当前部门口径计算，但允许同一用户在多个部门详情中分别出现。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]

- [x] 提供 `/api/enterprise/usage/department-detail` API，并保持现有权限边界 (AC: 1, 2, 3)
  - [x] 在 `controller/enterprise/usage.go` 新增 `GetDepartmentUsageDetail`，只负责参数解析、权限校验、调用 service 和返回 `{ success, message, data }`，不在 controller 层拼 SQL 或聚合趋势数据。[Source: _bmad-output/planning-artifacts/architecture.md#backend-内部分层]
  - [x] 在 `router/enterprise-router.go` 注册 `/api/enterprise/usage/department-detail`，继续沿用 `middleware.EnterpriseAdmin()` 的现有权限边界，不擅自引入“经营负责人”或部门管理员的细粒度新模型。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
  - [x] 最近消费日志入口只需要产生正确的筛选上下文/链接，不得修改 `model/log.go`、`controller/log.go` 或现有日志语义。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14; model/log.go]

- [x] 在 Default `enterprise-usage` feature 中新增部门详情视图和 drill-down 交互 (AC: 1, 2, 3)
  - [x] 扩展 `web/default/src/features/enterprise-usage/`，在总览页的基础上新增单部门详情面板或二级视图，展示用户排行、模型分布、时间趋势和最近消费日志入口。[Source: _bmad-output/planning-artifacts/architecture.md#FR-14 | 查看部门用量详情; web/default/src/features/enterprise-usage/index.tsx]
  - [x] 详情页继续固定展示 `enterprise.usage.multi_dept_disclaimer`，不可因为已经进入单部门视图就隐藏；部门间不可加和的口径在详情层同样成立。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
  - [x] 用户排行需支持至少 `quota`、请求数、`token` 三种排序切换；趋势展示优先复用现有仓库中的图表/统计组件，不新增新的图表库。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14; web/default/package.json]

- [x] 接通最近消费日志入口，但保持日志模块语义零侵入 (AC: 3)
  - [x] 在详情页生成带部门、时间范围和用户筛选条件的日志页跳转链接；优先复用现有 `usage-logs` 路由和 query 参数风格，不在本故事中重构日志模块。[Source: web/default/src/routes/_authenticated/usage-logs/index.tsx; web/default/src/features/usage-logs/]
  - [x] 跳转仅携带筛选上下文，不新增企业专属日志表，也不修改日志写入链路。[Source: model/log.go; controller/log.go]

- [x] 完成测试与回归，保护详情语义和查询契约 (AC: 1, 2, 3)
  - [x] 新增/扩展 `service/enterprise/usage_aggregation_test.go`，覆盖单部门详情的人排行、模型分布、趋势窗口和多部门用户重复出现语义。[Source: _bmad-output/planning-artifacts/architecture.md#Test Organization]
  - [x] 新增/扩展 `controller/enterprise/usage_test.go` 与 `tests/api/enterprise_usage_test.go`，覆盖 `department-detail` 参数校验、权限、空数组、最近日志入口筛选上下文和相邻窗口空结果。[Source: controller/enterprise/usage_test.go; tests/api/enterprise_usage_test.go]
  - [x] 扩展 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`，覆盖详情页的 disclaimer、排序切换、未归属处理、趋势展示和日志入口链接参数。[Source: web/default/rsbuild.test.config.ts; web/default/package.json]

## Dev Notes

- Story 4.3 是在 4.2 总览看板基础上继续 drill-down，不要重复实现一套新的 usage feature 或 query 体系；优先扩展现有 `enterprise-usage` 模块内的详情查询、详情组件和测试。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- 4.3 的详情查询仍必须建立在 4.1 的 `enterprise_usage_snapshots` 聚合表之上，而不是回退到对 `logs` 的在线全量扫描。最近消费日志入口只是链接/筛选上下文，不是详情主数据源。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; docs/enterprise/usage-aggregation.md]
- 同一用户在多部门详情里重复出现是业务上正确的，不要在排行层做跨部门去重；但在单个部门详情内的用户统计必须只按该部门口径计算。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14; _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- 详情页顶部同样必须固定展示 `enterprise.usage.multi_dept_disclaimer`，因为用户很容易把单部门详情与企业总量混淆。[Source: _bmad-output/planning-artifacts/architecture.md#FR-14 | 查看部门用量详情]
- `usage-logs` 现有模块和 `model/log.go` / `controller/log.go` 的语义是受保护的；4.3 只做“带筛选参数跳转”，不做日志语义改造。[Source: model/log.go; controller/log.go; web/default/src/features/usage-logs/]
- 详情视图要复用 4.2 已引入的 `enterprise-usage` 路由、query key 和 i18n 体系，避免再散到 `dashboard/` 或 `usage-logs/`。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]

### Project Structure Notes

- 预计新增/修改文件：
  - `controller/enterprise/usage.go`
  - `dto/enterprise/usage.go`
  - `service/enterprise/usage_aggregation.go`
  - `service/enterprise/usage_aggregation_test.go`
  - `tests/api/enterprise_usage_test.go`
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `web/default/src/features/enterprise-usage/types.ts`
  - 需要时补充 `api.ts` / detail hook
- 继续不要触碰：
  - `model/log.go`
  - `controller/log.go`
  - `service/enterprise/usage_aggregation.go` 的 4.1 已定聚合口径，只做详情能力增量扩展

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 4.3: 查看部门用量详情]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14]
- [Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR-14 | 查看部门用量详情]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md]
- [Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- [Source: docs/enterprise/usage-aggregation.md]
- [Source: controller/enterprise/usage.go]
- [Source: dto/enterprise/usage.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: router/enterprise-router.go]
- [Source: middleware/enterprise_admin.go]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工从 Epic 4 / FR14 / architecture 的 usage-detail 段补齐 story 上下文，因为自动 `create-story` 多轮未能把 4.3 文件落地。
- 复用 4.1 / 4.2 已确定的 usage 聚合契约、免责声明和前端 feature 边界，避免在 4.3 重造总览层。
- 手工回退执行 `bmad-dev-story` workflow，因为 `_bmad/scripts/resolve_customization.py` 依赖 Python 3.11+ 的 `tomllib`，当前环境只能按 skill 说明手工解析 workflow。
- 先补红灯测试再落实现：后端新增 `department-detail` DTO / service / controller / router，前端在既有 `enterprise-usage` feature 内扩展 detail query、drill-down 面板、排序切换、趋势图和日志跳转上下文。
- 已确认 story automator 的 Codex child `CODEX_HOME` 现在同时链接 `auth.json` 与 `config.toml`；当前 4.3 dev 会话实际启动命令也已包含两者，本次实现阻塞不再是早期 `invalid key` 根因。
- 运行 `bun run i18n:sync` 同步 4.3 新文案，并补齐 `fr/ja/ru/vi` 中的 `Top Model` 译文；当前 `_sync-report.json` 已全部归零。

### Completion Notes List

- 已完成 `GET /api/enterprise/usage/department-detail`，返回部门范围内的用户排行、模型分布、按小时趋势和最近日志入口筛选上下文；主查询继续只读 `enterprise_usage_snapshots`，没有引入日志全表回退路径。
- 已完成 Default `enterprise-usage` drill-down 详情面板：固定展示免责说明、支持 `quota` / 请求数 / `token` 排序、复用现有 chart/table 组件渲染趋势和模型分布，并通过 search state 保持部门详情与排序状态。
- 已完成最近消费日志入口跳转，沿用 `/usage-logs/common` 与既有 `startTime` / `endTime` / `username` 查询参数风格，只传筛选上下文，不修改日志模块后端语义；同时保留 `departmentId` / `departmentName` 作为来源上下文，并在日志页二次筛选/重置时持续保留。
- 已补齐并通过 4.3 相关测试：`service/enterprise/usage_aggregation_test.go`、`controller/enterprise/usage_test.go`、`tests/api/enterprise_usage_test.go`、`web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`。
- 已运行 `bun run i18n:sync`、`bun run typecheck`、`bun run test:e2e`、`bun run build:check`、`env GOCACHE=/private/tmp/new-api-go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage' -count=1`。

### Senior Developer Review (AI)

- Reviewer: GPT-5 Codex
- Date: 2026-05-29 20:39:17 +0800
- Outcome: Changes Requested -> Fixed -> Approved
- Verified against implementation and tests, excluding `_bmad/` and `_bmad-output/` source review scope per workflow.
- Findings fixed during review:
  - `web/default/src/routes/_authenticated/usage-logs/$section.tsx`: usage-logs route search schema used `z.number()` instead of coercion, so deep-linked recent-log drill-down query params from the URL could fail to parse on reload and silently drop filters/context.
  - `web/default/src/features/usage-logs/components/common-logs-filter-bar.tsx`: reset handler captured stale `departmentContext`, so after entering from department detail the reset flow could lose required department context from AC 3.
  - `web/default/src/features/usage-logs/components/task-logs-filter-bar.tsx`: same stale-closure bug as common logs, causing source department context to disappear after reset/filter reuse.
  - `web/default/src/features/usage-logs/index.tsx`: non-common page title metadata incorrectly collapsed drawing logs into the task title path, producing wrong section labeling in the touched logs surface.
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`: time-range assertions were pinned to a different timezone assumption than the implementation, creating unstable regression coverage.
- Validation:
  - `env GOCACHE=/private/tmp/new-api-go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage' -count=1`
  - `bun test web/default/src/features/enterprise-usage/enterprise-usage.test.tsx web/default/src/features/usage-logs/usage-logs.test.tsx`

### File List

- `_bmad-output/implementation-artifacts/4-3-view-department-usage-details.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/implementation-artifacts/tests/test-summary.md`
- `_bmad-output/story-automator/orchestration-3-20260529-041341.md`
- `controller/enterprise/department_membership_test.go`
- `controller/enterprise/usage.go`
- `controller/enterprise/usage_test.go`
- `dto/enterprise/usage.go`
- `router/enterprise-router.go`
- `service/enterprise/usage_aggregation.go`
- `service/enterprise/usage_aggregation_test.go`
- `tests/api/enterprise_usage_test.go`
- `web/default/src/features/enterprise-usage/api.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/features/enterprise-usage/hooks/use-department-usage-detail.ts`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/features/usage-logs/components/common-logs-filter-bar.tsx`
- `web/default/src/features/usage-logs/components/task-logs-filter-bar.tsx`
- `web/default/src/features/usage-logs/index.tsx`
- `web/default/src/features/usage-logs/lib/filter.ts`
- `web/default/src/features/usage-logs/usage-logs.test.tsx`
- `web/default/src/features/usage-logs/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/_reports/_sync-report.json`
- `web/default/src/routes/_authenticated/usage-logs/$section.tsx`

## Change Log

- 2026-05-29: 新增部门详情聚合接口 `/api/enterprise/usage/department-detail`、Default `enterprise-usage` drill-down 面板、用户排行排序、趋势图和最近日志入口筛选上下文，并补齐前后端回归测试与六语言词条。
- 2026-05-29: 补强日志页部门上下文保留能力，确保 recent logs 跳转在后续筛选/重置后仍保留来源部门信息，同时保持日志查询语义零侵入。
- 2026-05-29: AI review 自动修复 usage-logs drill-down 的 query coercion、重置后部门上下文丢失、drawing section 标题回退错误，并新增对应回归测试。
