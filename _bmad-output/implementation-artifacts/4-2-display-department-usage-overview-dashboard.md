---
baseline_commit: a73d0055ec88960b426939fc84c90977c5cf64a0
---

# Story 4.2: 展示部门用量总览看板

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或经营负责人,
I want 按时间范围查看部门用量总览,
so that 我可以快速判断各部门 AI 使用规模和模型分布。

## Acceptance Criteria

1. **Given** 部门用量聚合数据已生成  
   **When** 管理员选择今天、昨天、近 7 天、近 30 天或自定义时间范围  
   **Then** 页面展示部门维度请求数、`prompt_tokens`、`completion_tokens`、`quota`、模型分布和用户数  
   **And** 同一笔消费可在多个部门出现，这是业务上正确的重复计入。
2. **Given** 管理员打开部门用量看板  
   **When** 页面加载完成  
   **Then** 顶部固定展示“部门间数值不可加和”说明  
   **And** 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`，不可被管理员隐藏。

## Tasks / Subtasks

- [x] 为部门用量总览补齐前端 feature 骨架与路由入口，严格落在 `enterprise-usage` 领域内 (AC: 1, 2)
  - [x] 新增 `web/default/src/features/enterprise-usage/`，至少包含 `api.ts`、`types.ts`、`index.tsx`，沿用现有 feature-sliced 结构，不把企业用量页面塞进 `dashboard/`、`usage-logs/` 或 `enterprise-organization/`。[Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture; _bmad-output/planning-artifacts/architecture.md#File Structure Patterns; _bmad-output/planning-artifacts/architecture.md#组织用量看板与导出]
  - [x] 新增 `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`，遵循现有 `createFileRoute('/_authenticated/.../')` 路由模式和 `_authenticated` 目录约定；不要手写独立 router，也不要复用 `/dashboard/*` 或 `/usage-logs/*` 路径。[Source: web/default/src/routes/_authenticated/enterprise-organization/index.tsx; web/default/src/routes/_authenticated/enterprise-dingtalk/index.tsx; _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]
  - [x] 在 `web/default/src/hooks/use-sidebar-data.ts` 新增 `/enterprise-usage` 管理入口，并同步更新 `web/default/src/hooks/use-sidebar-config.ts` 的 `DEFAULT_SIDEBAR_MODULES` 与 `URL_TO_CONFIG_MAP`，保证 sidebar 模块裁剪与导航可见性一致；不能只加菜单不补映射。[Source: web/default/src/hooks/use-sidebar-data.ts; web/default/src/hooks/use-sidebar-config.ts]

- [x] 基于现有 summary API 建立部门总览查询层，不重造后端聚合逻辑 (AC: 1)
  - [x] 在 `web/default/src/features/enterprise-usage/api.ts` 复用现有 `GET /api/enterprise/usage/department-summary`，定义 queryKey 为 `['enterprise', 'usage', 'department-summary', { from, to, tenantId? }]`，保持参数可序列化并按 feature 级精确失效。[Source: controller/enterprise/usage.go; dto/enterprise/usage.go; _bmad-output/planning-artifacts/architecture.md#Frontend Architecture; _bmad-output/planning-artifacts/architecture.md#Process Patterns]
  - [x] 新增前端类型以承接 `dept_id` 可空、`model_distribution` 必返数组、`window_start/window_end`、`user_count` 等字段，并保持未归属部门可用 `dept_id = null` / `dept_name = "未归属"` 正常渲染，而不是当成异常值丢弃。[Source: dto/enterprise/usage.go; service/enterprise/usage_aggregation.go; docs/enterprise/usage-aggregation.md#归因口径]
  - [x] 若需要最小化后端补充，只允许在 `controller/enterprise/usage.go` / `dto/enterprise/usage.go` 内做展示友好的非破坏性扩展；禁止改动 `usage_aggregation.go` 的聚合语义、回退到 `logs` 直查，或新增第二条在线聚合路径。[Source: controller/enterprise/usage.go; service/enterprise/usage_aggregation.go; docs/enterprise/usage-aggregation.md#幂等与回填]

- [x] 实现时间范围过滤与总览布局，确保时间口径和聚合窗口语义一致 (AC: 1)
  - [x] 页面必须支持“今天、昨天、近 7 天、近 30 天、自定义时间范围”五类筛选；优先复用现有日期范围交互模式，如 `CompactDateTimeRangePicker` 或同级交互，而不是重新发明一套时间组件。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.2: 展示部门用量总览看板; web/default/src/features/usage-logs/components/compact-date-time-range-picker.tsx]
  - [x] 时间计算必须与后端 summary 的 `[from, to)` 过滤语义和 1 小时快照窗口保持一致。特别是“今天/昨天”不要因为前端把结束时间算成闭区间或本地分钟级截断而少算/多算一个窗口。[Source: service/enterprise/usage_aggregation.go; docs/enterprise/usage-aggregation.md#默认规则]
  - [x] 页面主体至少展示部门行级表格和可扫描的摘要信息。摘要卡片、表格和 loading/empty/error 态应复用现有 `SectionPageLayout`、`StatCard`、`DataTablePage`、`Skeleton`、`Alert` 等项目模式，不要为本故事引入新表格框架或自制全局布局体系。[Source: _bmad-output/planning-artifacts/architecture.md#Loading State Patterns; web/default/src/components/layout/index.ts; web/default/src/features/dashboard/components/ui/stat-card.tsx; web/default/src/components/data-table/data-table-page.tsx]

- [x] 正确呈现“多部门重复计入”口径与未归属部门，不允许被 UI 弱化或隐藏 (AC: 1, 2)
  - [x] 在页面顶部固定渲染 `t('enterprise.usage.multi_dept_disclaimer')`，它必须默认可见、不可关闭、不可折叠到次级弹窗，也不能只放在 tooltip 或空状态里。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.2: 展示部门用量总览看板; _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
  - [x] 为 Default 前端六种语言字典补齐 `enterprise.usage.multi_dept_disclaimer` 及本故事新增文案，并运行 `bun run i18n:sync` 维持词条一致性；这里允许使用扁平业务 key，仓库已有 `enterprise.organization.*` 前例，不必强行改成英文句子 key。[Source: web/default/src/i18n/locales/en.json; _bmad-output/planning-artifacts/architecture.md#i18n; AGENTS.md#internationalization-i18n]
  - [x] 总览表必须保留“未归属”行，不得因 `dept_id = null` 被前端过滤；同时要把多部门重复计入视为正确业务结果，不做跨部门总和、去重总计或“修正”显示。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增); docs/enterprise/usage-aggregation.md#归因口径]

- [x] 明确当前权限边界和角色假设，避免在 Story 4.2 擅自扩展权限模型 (AC: 1, 2)
  - [x] 当前后端 `department-summary` 路由受 `middleware.EnterpriseAdmin()` 保护，真实生效门槛是 `role >= Admin`。故事文案里的“经营负责人”在现有代码中没有独立权限模型，因此 4.2 默认只实现对现有管理员能力的 UI 消费，不自行发明新 role、middleware 或绕过后端鉴权。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
  - [x] 前端可根据现有 auth/导航状态控制入口可见性，但权限真源必须仍以后端为准；不要因为想满足“经营负责人”字样而在前端单独放开页面访问。[Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]

- [x] 完成测试与回归，覆盖总览关键交互和最易出错的口径展示 (AC: 1, 2)
  - [x] 新增 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx` 或等价 Node 静态渲染测试，至少覆盖：固定 disclaimer、未归属行、时间预设文案、模型分布展示、空数据态和错误态；测试风格对齐 `enterprise-organization.test.tsx` / `enterprise-dingtalk.test.tsx`。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/src/features/enterprise-dingtalk/enterprise-dingtalk.test.tsx; web/default/rsbuild.test.config.ts]
  - [x] 更新 `web/default/rsbuild.test.config.ts` 与 `web/default/package.json` 的 `test:e2e` 入口，把新 feature 测试纳入现有静态测试流水，避免新增测试文件却不执行。[Source: web/default/rsbuild.test.config.ts; web/default/package.json]
  - [x] 运行前端最小回归：`bun run typecheck`、`bun run test:e2e`，必要时补 `bun run i18n:sync`；如果需要后端配合验证接口契约，至少覆盖 `controller/enterprise/usage_test.go` 或等价用例，确保 summary 返回字段没有被前端假设错读。[Source: controller/enterprise/usage.go; web/default/package.json; AGENTS.md#rule-3-frontend--prefer-bun]

## Dev Notes

- Story 4.2 不是新建聚合链路，而是消费 Story 4.1 已落地的 `enterprise_usage_snapshots` + `GET /api/enterprise/usage/department-summary`。开发重点在前端总览看板、时间筛选和展示正确性，不在重新实现聚合或把 summary API 改成查 `logs`。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; controller/enterprise/usage.go; service/enterprise/usage_aggregation.go]
- 当前后端 summary API 已返回 `dept_id/dept_name/window_start/window_end/request_count/prompt_tokens/completion_tokens/quota/user_count/model_distribution`，且空数组已归一化。优先直接消费这些字段；除非显示层存在明确阻塞，不要先改后端再写页面。[Source: dto/enterprise/usage.go; controller/enterprise/usage.go]
- `usage_aggregation.go` 的 summary 查询对时间范围采用 `window_start >= from AND window_end <= to`，并对多个窗口合并模型分布和去重用户数。因此前端时间预设必须给出与整点快照兼容的范围，不要用模糊自然语言范围或本地浏览器时区副作用去猜测后端会帮你纠正。[Source: service/enterprise/usage_aggregation.go; docs/enterprise/usage-aggregation.md]
- 顶部免责说明是架构级硬约束，不是 UX 建议。它同样会被 Story 4.3、4.4、5.6 复用，所以 `4.2` 应把文案和展示模式做成清晰、可复用的实现，而不是只在本页临时硬编码一段裸文本。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- 架构明确要求 Default 企业能力按独立 feature 切分，并指出不要把企业能力塞进现有 Group/User/Log 页面。`4.2` 应新增 `enterprise-usage` feature 和独立路由，而不是在 `usage-logs/` 或 `dashboard/` 下加一个特例 section，因为那会把企业统计和现有日志/全局分析语义混在一起。[Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture; _bmad-output/planning-artifacts/architecture.md#File Structure Patterns]
- 现有 sidebar 配置系统对 URL 有显式映射；如果只加 route 不补 `useSidebarData()` 和 `useSidebarConfig()`，某些管理员或被裁剪导航的用户会出现菜单不一致、入口消失或权限收窄失效的问题。这个问题不会在本地“直接输入 URL”测试中自然暴露，必须显式处理。[Source: web/default/src/hooks/use-sidebar-data.ts; web/default/src/hooks/use-sidebar-config.ts]
- 当前真实权限边界仍是 `EnterpriseAdmin()`。Story 文案里的“经营负责人”暂时应理解为业务角色描述，而不是实现层的新鉴权角色。4.2 不应单方面扩展 middleware、role 常量、数据库权限表或 department-admin 视图范围；如要支持更细粒度权限，应在后续故事/架构修订中单独设计。[Source: middleware/enterprise_admin.go; router/enterprise-router.go]
- Default 前端技术栈固定为 React 19 + TanStack Query/Router + Base UI + Tailwind，前端测试通过 `rsbuild.test.config.ts` 编到 Node 运行。不要为这个故事引入额外图表库、状态管理库或独立测试框架；如需图形表达，先复用仓库现有依赖和布局原子。[Source: web/default/package.json; _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]

### Project Structure Notes

- 预计新增文件：
  - `web/default/src/features/enterprise-usage/api.ts`
  - `web/default/src/features/enterprise-usage/types.ts`
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`
- 预计修改文件：
  - `web/default/src/hooks/use-sidebar-data.ts`
  - `web/default/src/hooks/use-sidebar-config.ts`
  - `web/default/rsbuild.test.config.ts`
  - `web/default/package.json`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/vi.json`
- 可能需要最小化确认但不应大改的后端文件：
  - `controller/enterprise/usage.go`
  - `dto/enterprise/usage.go`
  - `controller/enterprise/usage_test.go`
- 明确不要触碰：
  - `service/enterprise/usage_aggregation.go` 的聚合口径
  - `model/log.go`、`controller/log.go`
  - `web/default/src/features/usage-logs/`
  - `web/default/src/features/dashboard/`

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:539)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:343)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:399)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:572)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:673)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:838)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:1152)
- [4-1-aggregate-department-usage-snapshots.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md:1)
- [docs/enterprise/usage-aggregation.md](/Users/hq-it/repository/github/huaqingai/new-api/docs/enterprise/usage-aggregation.md:1)
- [controller/enterprise/usage.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/usage.go:14)
- [dto/enterprise/usage.go](/Users/hq-it/repository/github/huaqingai/new-api/dto/enterprise/usage.go:3)
- [service/enterprise/usage_aggregation.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/usage_aggregation.go:23)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:9)
- [middleware/enterprise_admin.go](/Users/hq-it/repository/github/huaqingai/new-api/middleware/enterprise_admin.go:9)
- [web/default/src/hooks/use-sidebar-data.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/hooks/use-sidebar-data.ts:47)
- [web/default/src/hooks/use-sidebar-config.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/hooks/use-sidebar-config.ts:38)
- [web/default/src/routes/_authenticated/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routes/_authenticated/enterprise-organization/index.tsx:20)
- [web/default/src/routes/_authenticated/enterprise-dingtalk/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routes/_authenticated/enterprise-dingtalk/index.tsx:20)
- [web/default/src/features/usage-logs/components/compact-date-time-range-picker.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/usage-logs/components/compact-date-time-range-picker.tsx:1)
- [web/default/src/components/data-table/data-table-page.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/components/data-table/data-table-page.tsx:1)
- [web/default/src/features/dashboard/components/ui/stat-card.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/dashboard/components/ui/stat-card.tsx:1)
- [web/default/src/features/enterprise-organization/enterprise-organization.test.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/enterprise-organization.test.tsx:1)
- [web/default/src/features/enterprise-dingtalk/enterprise-dingtalk.test.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-dingtalk/enterprise-dingtalk.test.tsx:1)
- [web/default/rsbuild.test.config.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/rsbuild.test.config.ts:1)
- [web/default/package.json](/Users/hq-it/repository/github/huaqingai/new-api/web/default/package.json:1)
- [web/default/src/i18n/locales/en.json](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/i18n/locales/en.json:1189)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 已按要求读取 `.agents/skills/bmad-dev-story/SKILL.md` 与 `checklist.md`，并手工回退解析 workflow 配置，因为 `_bmad/scripts/resolve_customization.py` 受限于当前 Python 版本缺少 `tomllib`。
- 已完整读取 Story 4.2、`sprint-status.yaml`、4.1 交付物、`controller/enterprise/usage.go`、`dto/enterprise/usage.go`、`docs/enterprise/usage-aggregation.md`、`web/default` 路由与 sidebar 配置，实现保持在既有 summary API 和 Default 主题范围内。
- 新增 `enterprise-usage` feature、query hook、认证路由、sidebar 入口与 sidebar config 映射，并让 URL search 驱动“今天 / 昨天 / 近 7 天 / 近 30 天 / 自定义”时间范围。
- 通过 `enterprise-usage.test.tsx`、`rsbuild.test.config.ts` 与 `package.json` 将新看板测试纳入现有 Node 静态测试流水，覆盖 disclaimer、未归属行、时间范围、空态、错误态和模型摘要。
- 补充并验证多语言词条，包含必需的 `enterprise.usage.multi_dept_disclaimer`，随后运行 `bun run i18n:sync` 统一词条。
- 补强 `controller/enterprise/usage_test.go` 与 `tests/api/enterprise_usage_test.go`，确认 summary API 继续返回数组而非 `null`，并验证管理员权限与相邻时间窗口空结果。

### Completion Notes List

- 已完成 Default `enterprise-usage` 总览看板，实现独立 feature、认证路由、sidebar 入口和 sidebar 模块裁剪映射。
- 已完成 summary API 前端消费层，queryKey 使用 `['enterprise', 'usage', 'department-summary', { from, to }]` 精确范围键，并保持未归属部门和模型分布数组正常渲染。
- 已完成固定 disclaimer、时间范围筛选、未归属行展示、模型分布摘要、空态/错误态/重试，以及自定义时间范围前端校验。
- 已补齐 `en/zh/fr/ru/ja/vi` 词条并运行 `bun run i18n:sync`。
- 已通过 `bun run typecheck`、`bun run test:e2e`、`env GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api -run 'Test(UsageSummary|EnterpriseUsageSummary)' -count=1`。

### File List

- `_bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/usage_test.go`
- `tests/api/enterprise_usage_test.go`
- `web/default/package.json`
- `web/default/rsbuild.test.config.ts`
- `web/default/src/features/enterprise-usage/api.ts`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/hooks/use-department-usage-summary.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`
- `web/default/src/hooks/use-sidebar-config.ts`
- `web/default/src/hooks/use-sidebar-data.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/routeTree.gen.ts`

## Change Log

- 2026-05-29: 新增 Default `enterprise-usage` 总览看板、时间范围查询层、认证路由、sidebar 接入、六语言词条和静态前端测试，并补充 summary API 回归测试后将故事状态更新为 `review`。
- 2026-05-29: AI review 自动修复时间范围开区间边界、摘要卡片缺失、`tenant_id` 查询透传与错误信息翻译展示问题，并将故事状态更新为 `done`。

## Senior Developer Review (AI)

### Review Date

- 2026-05-29 19:17:01 +0800

### Findings

- HIGH: `web/default/src/features/enterprise-usage/index.tsx` 的预设与自定义时间范围把结束时间实现成闭区间日末秒值，但后端 summary 使用 `[from, to)` 且要求 `window_end <= to`，会稳定漏掉最后一个完整小时窗口。
- HIGH: 页面主体只有部门表格，没有任务要求中的“可扫描摘要信息”，AC 1 只能算部分完成。
- MEDIUM: `web/default/src/features/enterprise-usage/api.ts` 与 `hooks/use-department-usage-summary.ts` 没有覆盖故事要求的可选 `tenantId` 查询维度，query key 也未纳入该参数。
- MEDIUM: 错误态直接渲染原始 `error.message`，像 `common.invalid_params` 这类后端 i18n key 不能按前端词典正确翻译展示。

### Fixes Applied

- 已把预设与自定义时间范围统一改为与后端 summary 一致的开区间结束边界，并同步修正范围标签和自定义日期回显。
- 已补充基于现有 `StatCard` 的摘要卡片区，展示请求数、Token、额度、模型数、主力模型、未归属部门数与用户数。
- 已为部门总览查询层补充可选 `tenant_id` 透传，并将其纳入 feature 级 query key。
- 已将错误态消息切到 `t(errorMessage)`，与项目既有 i18n 展示方式保持一致。
- 已更新静态测试，覆盖新的时间边界、摘要展示与 `tenant_id` query key。

### Validation

- `bun run typecheck`
- `bun run test:e2e`
- `env GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api -run 'Test(UsageSummary|EnterpriseUsageSummary)' -count=1`
