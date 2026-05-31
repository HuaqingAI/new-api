---
baseline_commit: 378a640bf91ac5a3c0f7d91672a5663ab1b88416
---

# Story 6.2: 部门用量页重构为树驱动分析视图

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或经营负责人,
I want 部门用量页以“左侧可折叠组织树 + 右侧当前部门分析区”的结构工作,
so that 我可以围绕当前部门持续分析用量，而不是先看总表再跳详情。

## Acceptance Criteria

1. **Given** 管理员进入部门用量页  
   **When** 页面加载完成  
   **Then** 页面左侧显示可折叠组织树  
   **And** 右侧显示当前选中部门的用量概览、趋势、模型分布、用户排行和最近日志入口。
2. **Given** 管理员查看当前部门分析区  
   **When** 页面渲染完成  
   **Then** 右侧除用量数据外，还包含当前部门维度下的说明或配置区域  
   **And** 不再把“总览”和“配置”拆成割裂的独立主视图。
3. **Given** 管理员在左侧切换当前部门  
   **When** 页面状态更新  
   **Then** 当前选中部门仍是页面一级状态  
   **And** 右侧分析区在不离开页面主骨架的前提下切换到新的部门上下文。

## Tasks / Subtasks

- [ ] 将 `enterprise-usage` 页面重构为“左侧组织树 + 右侧当前部门分析区”的双栏骨架，替换当前先看总表再 drill-down 的主流程 (AC: 1, 2, 3)
  - [ ] 在 `web/default/src/routes/_authenticated/enterprise-usage/index.tsx` 继续使用 `validateSearch` / `useSearch` / `navigate({ search })`，把 `dept_id` 作为页面一级状态真源；如需要保留当前详情状态，可额外承载 `budget_id`、排序或明细用户等子状态，但不能让这些状态绕过当前部门真源。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; web/default/src/routes/_authenticated/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/index.tsx]
  - [ ] 右侧分析区必须在同一主骨架内承接部门概览、趋势、模型分布、用户排行、最近日志入口，以及已存在的报告配置/说明区域，而不是把“总览”和“配置”拆成独立主 tab 或完全不同页面。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图; _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]
  - [ ] 保持现有 `SectionPageLayout`、`Card`、`Alert`、`Skeleton`、`StatCard` 等 Default 设计体系，不引入第二套页面框架或新的详情路由栈。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]

- [ ] 复用组织树语义，把部门切换变成用量页的主导航，而不是总览表内的次级动作 (AC: 1, 3)
  - [ ] 优先复用 Story 6.1 已交付的组织树数据源、树工具和 URL 恢复策略，包括 `GET /api/enterprise/departments/tree`、`useDepartmentTree()`、祖先路径展开、无效 `dept_id` 回退、默认根节点展开等行为；不要为用量页新建第二棵树接口或复制一套不同的树状态算法。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/hooks/use-department-tree.ts; web/default/src/features/enterprise-organization/lib/tree-utils.ts]
  - [ ] 左侧组织树切换后，右侧所有概览、趋势、模型分布、用户排行和最近日志入口都应自动跟随当前部门刷新；不得要求用户先回到总览表点击“View Details”才能进入部门分析。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图]
  - [ ] `dept_id` 缺失或失效时，页面必须稳定回退到首个可见部门，并清理无效子状态；不能保持损坏 URL 或白屏。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; controller/enterprise/department.go]

- [ ] 将现有用量总览、详情、趋势、模型分布和用户排行整合为“当前部门分析区”，不破坏既有口径 (AC: 1, 2, 3)
  - [ ] 继续复用 Story 4.2 / 4.3 已建立的 `enterprise-usage` 查询层、summary/detail API、时间范围解析与图表/表格组件，不允许回退为新建平行 feature，也不要把主数据源改成扫描 `logs`。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; web/default/src/features/enterprise-usage/api.ts; controller/enterprise/usage.go]
  - [ ] 右侧顶部至少提供当前部门维度的分析摘要与固定 disclaimer，内容应覆盖部门用量概览、趋势、模型分布、用户排行和最近日志入口；详情区不应要求用户通过“返回总览”才能切换部门上下文。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
  - [ ] “最近日志入口”仍然只传筛选上下文，不得修改 `model/log.go`、`controller/log.go` 或 usage logs 核心语义；Story 6.2 的目标是视图重排与上下文统一，不是日志系统改造。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; model/log.go; controller/log.go]

- [ ] 保留并内联当前部门维度下的说明/配置区域，避免分析页继续被拆成割裂主视图 (AC: 2)
  - [ ] 已有的 `Scheduled Usage Reports` 配置卡片、部门间不可加和说明或同等级的分析说明，必须保留在当前部门分析骨架中，而不是被移回单独主 tab、独立全屏配置页或与当前部门完全脱钩的全局区域。[Source: _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md; web/default/src/features/enterprise-usage/index.tsx]
  - [ ] 如果某些配置本质仍是租户级而非部门级，UI 也应在当前部门分析区里清楚表达其作用域，避免用户误以为离开页面才可配置报告或说明内容。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图; web/default/src/features/enterprise-usage/types.ts]

- [ ] 重构 `enterprise-usage` 查询组织方式，使其围绕当前部门上下文工作且不会污染现有总览/详情语义 (AC: 1, 2, 3)
  - [ ] 对 summary、detail、report config、最近日志 drill-down 等查询/本地 UI state 做清晰分层：当前部门由 search 驱动，次级排序/子面板状态围绕当前部门工作，避免多个互相竞争的部门真源。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
  - [ ] 切换部门时应重置与旧部门绑定的无效子状态，例如 detail 用户、选中的明细段、可能的报告相关局部状态，但不得把时间范围、租户过滤等跨部门共享分析上下文无意义地重置。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
  - [ ] 所有失效与刷新继续遵守 `['enterprise', 'usage', ...]` 及 feature 级 query key 约束，不用 `invalidateQueries({ queryKey: ['enterprise'] })` 粗暴全刷。[Source: _bmad-output/planning-artifacts/architecture.md#Communication Patterns; web/default/src/features/enterprise-usage/api.ts]

- [ ] 维持 Classic 最小可用入口，但不把 Story 6.2 扩张成 Classic 深度重构 (AC: 1, 2)
  - [ ] 检查 Classic 企业用量入口是否受 Default 端 API / 字段组织方式变化影响；如需最小修正，保持 `web/classic/src/services/enterprise.js` 和既有 `pages/` 模式，不复制 Default 的树驱动全套交互。[Source: _bmad-output/planning-artifacts/architecture.md#Important Decisions; web/classic/src/services/enterprise.js]
  - [ ] Story 6.2 的主要实现面仍然是 Default `enterprise-usage` feature；Classic 只做必要兼容，不追求同等深度。[Source: _bmad-output/planning-artifacts/architecture.md#Classic Theme]

- [ ] 补齐测试、i18n 与回归，锁定当前部门分析区语义 (AC: 1, 2, 3)
  - [ ] 扩展 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`，至少覆盖：组织树驱动的 `dept_id` 恢复/回退、切换部门后右侧分析区整体切换、说明/配置区域仍保留在主骨架内、无效 `dept_id` 降级、以及报告配置卡片在新骨架中的存在。[Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx; _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]
  - [ ] 如复用 6.1 的树工具或 search 归一化逻辑，补对应的静态测试，确保部门切换、祖先展开和 URL 恢复在用量页同样成立。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [ ] 所有新增文案同步到 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，并运行 `bun run i18n:sync`；特别是 6.2 新增的“当前部门分析区”“说明/配置区域”类文案不能只留在英文源里。[Source: AGENTS.md#Internationalization i18n]
  - [ ] 运行最小前端回归：`cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`；若为复用树上下文新增了服务端依赖读取方式，补充最小 API 回归以确认现有 `department-summary` / `department-detail` 契约未破坏。[Source: AGENTS.md#Rule 3: Frontend — Prefer Bun; controller/enterprise/usage.go]

## Dev Notes

- Story 6.2 本质上是把 Story 4.2、4.3、4.5 已落地的 `enterprise-usage` 总览、详情、报告配置重排为“左侧树导航 + 右侧当前部门分析区”，而不是新建一套第四条用量分析链路。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]
- Story 6.1 已经证明 TanStack Router search + 组织树工具可以稳定承接 `dept_id` 恢复、祖先展开和无效回退；6.2 应优先复用这些工具和行为，不再复制一套不同的部门真源逻辑。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/lib/tree-utils.ts]
- 当前 `enterprise-usage/index.tsx` 已经同时承载总览表、详情 panel 和报告配置卡片；6.2 需要重排这些区域的主从关系，而不是把报告配置或 disclaimer 从主骨架里移除。[Source: web/default/src/features/enterprise-usage/index.tsx]
- `enterprise.usage.multi_dept_disclaimer` 仍然是架构级硬约束，切到树驱动分析区后也必须默认可见，不能因为“当前部门分析”就隐藏或折叠。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增); web/default/src/i18n/locales/en.json]
- “总览”和“配置”不再是割裂主视图，意味着 6.2 不应继续把报告配置看作平级主 tab 或需要离开分析骨架的配置流；如果保留局部 section，也必须共享当前部门上下文和同一页面骨架。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图]
- Story 4.3 的最近日志入口语义和 usage-logs query 风格已经确定，6.2 只能在当前部门分析区里保留/加强该入口，不得发明新的日志路由或改变日志后端字段语义。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; web/default/src/features/usage-logs/]
- 当前权限边界仍以后端 `EnterpriseAdmin()` 为真源；Story 6.2 不自行扩展“经营负责人”鉴权模型，只在现有管理员入口上重构分析体验。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]

### Project Structure Notes

- 重点修改文件：
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`
  - 需要时复用/引用 `web/default/src/features/enterprise-organization/lib/tree-utils.ts`
  - 需要时补充 `web/default/src/features/enterprise-usage/api.ts` / `types.ts`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/vi.json`
- 保持不改或仅只读依赖：
  - `controller/enterprise/usage.go`
  - `dto/enterprise/usage.go`
  - `service/enterprise/usage_aggregation.go`
  - `service/enterprise/usage_report_task.go`
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图]
- [Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- [Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- [Source: _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]
- [Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
- [Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- [Source: web/default/src/features/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-usage/api.ts]
- [Source: web/default/src/features/enterprise-usage/types.ts]
- [Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]
- [Source: web/default/src/routes/_authenticated/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-organization/lib/tree-utils.ts]
- [Source: web/default/src/features/enterprise-organization/hooks/use-department-tree.ts]
- [Source: web/default/src/i18n/locales/en.json]
- [Source: controller/enterprise/usage.go]
- [Source: dto/enterprise/usage.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: router/enterprise-router.go]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工接管 `create-story`，因为 story-automator 连续两次 `6.2` create 会话都没有把 story 文件落地，也没有把 `sprint-status` 从 `backlog` 推进到 `ready-for-dev`。
- 已读取 Epic 6 对 Story 6.2 的 AC、Story 4.2/4.3/4.5 已完成交付，以及当前 `web/default/src/features/enterprise-usage/index.tsx` 结构，确认 6.2 是对既有 `enterprise-usage` 总览/详情/报告区的树驱动重排，而非新建 feature。
- 已确认 6.2 应直接复用 Story 6.1 的组织树 search/回退/祖先展开语义，以及 Story 4.x 既有免责声明、报告配置与最近日志入口能力。

### Completion Notes List

- 手工补齐 Story 6.2 故事文档，明确当前部门为页面一级状态、左树右分析区骨架、说明/配置区域内联和 URL 恢复约束。
- 明确 6.2 的主要修改面集中在 Default `enterprise-usage` feature、其测试和 i18n，不扩展后端 usage 聚合、日志语义或 Classic 深度交互。
- 记录两次自动 create 会话未落地 story artifact 的情况，供后续 automator 优化 create-step 停滞处理。

### File List

- `_bmad-output/implementation-artifacts/6-2-rebuild-department-usage-as-tree-driven-analysis-view.md`
