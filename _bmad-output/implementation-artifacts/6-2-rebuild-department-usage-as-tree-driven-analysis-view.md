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

- [x] 将 `enterprise-usage` 页面重构为“左侧组织树 + 右侧当前部门分析区”的双栏骨架，替换当前先看总表再 drill-down 的主流程 (AC: 1, 2, 3)
  - [x] 在 `web/default/src/routes/_authenticated/enterprise-usage/index.tsx` 继续使用 `validateSearch`，并由 `enterpriseUsageSearchSchema` 承载 `dept_id`、`include_descendants`、`log_user`、排序与时间范围等 URL search 状态。[Source: web/default/src/routes/_authenticated/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/index.tsx]
  - [x] 右侧分析区在同一主骨架内承接当前部门概览、趋势、模型分布、用户排行、最近日志入口、说明区和报告配置卡片，避免继续把“总览”和“配置”拆成独立主视图。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图]
  - [x] 保持现有 `SectionPageLayout`、`Card`、`Alert`、`Skeleton`、`StatCard`、Base UI/Tailwind 组件体系，没有引入第二套页面框架或新的详情路由栈。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]

- [x] 复用组织树语义，把部门切换变成用量页的主导航，而不是总览表内的次级动作 (AC: 1, 3)
  - [x] 复用 Story 6.1 已交付的 `useDepartmentTree()`、`DepartmentTree`、`resolveDepartmentSelection`、`syncExpandedDepartmentIds` 和 `toggleExpandedDepartmentId`，并继续消费 `GET /api/enterprise/departments/tree`；未新建第二棵树接口。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/hooks/use-department-tree.ts; web/default/src/features/enterprise-organization/lib/tree-utils.ts]
  - [x] 左侧组织树切换时通过 `navigate({ search })` 更新 `dept_id` 并清理 `log_user`，右侧 summary/detail/report 区域留在同一页面骨架内按当前部门上下文刷新。[Source: web/default/src/features/enterprise-usage/index.tsx]
  - [x] `dept_id` 缺失或失效时通过 `normalizeEnterpriseUsageSearch()` 回退到首个可见部门；当组织树为空时清理 `dept_id`、`include_descendants` 和 `log_user`，避免损坏 URL 或白屏。[Source: web/default/src/features/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]

- [x] 将现有用量总览、详情、趋势、模型分布和用户排行整合为“当前部门分析区”，不破坏既有口径 (AC: 1, 2, 3)
  - [x] 继续复用 Story 4.2 / 4.3 / 4.5 已建立的 `enterprise-usage` 查询层、summary/detail/report API、时间范围解析和图表/表格组件；未新建平行 feature，也未把主数据源改成扫描 `logs`。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md; web/default/src/features/enterprise-usage/api.ts]
  - [x] 右侧顶部固定显示 `enterprise.usage.multi_dept_disclaimer`，并在当前部门上下文卡、分析说明和详情面板中持续表达“多部门重复计入、部门间不可加和”口径。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增); web/default/src/features/enterprise-usage/index.tsx]
  - [x] 最近日志入口继续只传筛选上下文到 usage logs；本故事未修改 `model/log.go`、`controller/log.go` 或日志写入/查询核心语义。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; web/default/src/features/enterprise-usage/index.tsx]

- [x] 保留并内联当前部门维度下的说明/配置区域，避免分析页继续被拆成割裂主视图 (AC: 2)
  - [x] `Analysis Notes` 在当前部门分析区内说明多部门口径、报告配置作用域和详情/日志作用域，防止用户误以为需要离开页面才能理解或配置用量报告。[Source: web/default/src/features/enterprise-usage/index.tsx]
  - [x] `Scheduled Usage Reports` 配置卡片保留在当前部门分析骨架内，并明确“除非配置卡片说明，否则定期报告仍是租户级作用域”。[Source: _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md; web/default/src/features/enterprise-usage/index.tsx]

- [x] 重构 `enterprise-usage` 查询组织方式，使其围绕当前部门上下文工作且不会污染现有总览/详情语义 (AC: 1, 2, 3)
  - [x] 当前部门由 URL search 驱动，summary/detail/report、排序、最近日志用户筛选等子状态围绕当前部门工作，避免多个互相竞争的部门真源。[Source: web/default/src/features/enterprise-usage/index.tsx]
  - [x] 切换部门时清理旧部门绑定的 `log_user`；时间范围、租户过滤、summary 排序和 `include_descendants` 等跨部门分析上下文保持稳定。[Source: web/default/src/features/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]
  - [x] 查询键继续遵守 `['enterprise', 'usage', ...]` feature 级约束，summary/detail/report 分别使用可序列化参数键；未使用 `invalidateQueries({ queryKey: ['enterprise'] })` 粗暴全刷。[Source: web/default/src/features/enterprise-usage/api.ts; _bmad-output/planning-artifacts/architecture.md#API & Communication Patterns]

- [x] 维持 Classic 最小可用入口，但不把 Story 6.2 扩张成 Classic 深度重构 (AC: 1, 2)
  - [x] 当前重构集中在 Default `enterprise-usage` feature；未改动企业 usage 后端契约，因此 Classic 既有 `web/classic/src/services/enterprise.js` / `pages/` 最小入口不需要复制 Default 的树驱动全套交互。[Source: _bmad-output/planning-artifacts/architecture.md#Classic Theme; web/classic/src/services/enterprise.js]
  - [x] Story 6.2 未扩展 Classic 深度 UI、未新增 Classic 专属 API，也未改变 `/api/enterprise/usage/**` 字段组织方式。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]

- [x] 补齐测试、i18n 与回归，锁定当前部门分析区语义 (AC: 1, 2, 3)
  - [x] 扩展 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`，覆盖 `dept_id` 恢复/回退、空树清理、当前部门分析区渲染、说明/配置区域保留、最近日志上下文和报告配置卡片存在。[Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]
  - [x] 新增/保留静态测试锁定 query key、export 参数、search schema、summary/detail normalization、用户排行排序和 report form reset 语义，防止树驱动重排破坏 Story 4.x 既有能力。[Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]
  - [x] 新增文案已同步到 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，包括 `Current Department Analysis`、`Analysis Notes`、`Use as Current Context` 等 6.2 相关文案。[Source: web/default/src/i18n/locales/en.json; web/default/src/i18n/locales/zh.json]
  - [x] 已执行并通过前端验证：`cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`；本故事未新增服务端 usage 契约，因此未要求额外 Go API 回归。[Source: _bmad-output/story-automator/orchestration-6-20260531-065455.md]

## Dev Notes

### Context Loaded

- 已按用户要求先读取 `.claude/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- `_bmad/scripts/resolve_customization.py` 因当前 Python 缺少 3.11+ `tomllib` 无法运行，已按 skill fallback 手工解析 `.claude/skills/bmad-create-story/customize.toml`：无 activation prepend/append，persistent facts 只包含 `file:{project-root}/**/project-context.md`，但仓库中未发现 `project-context.md`。
- 已读取 `_bmad/bmm/config.yaml`：`user_name=hth`、`project_name=new-api`、沟通语言/文档输出语言为中文，规划目录为 `_bmad-output/planning-artifacts`，实现目录为 `_bmad-output/implementation-artifacts`。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认 `6-2-rebuild-department-usage-as-tree-driven-analysis-view` 当前为 `done`。本次 create-story 复核不把已完成故事降级为 `ready-for-dev`。
- 已加载并分析 `epics.md`、`architecture.md`、Story 4.2、4.3、4.5、Story 6.1、当前 `enterprise-usage` 真实代码、前端测试和 i18n 词条。未发现独立 UX 文档；UX 约束来自 Epic 6、Architecture 与既有 Default 企业页面模式。

### Story Intent and Current Gap

- Story 6.2 是 Epic 6 的树驱动治理视图跟进故事，直接 follow-up FR13 / FR14。目标不是新增用量聚合能力，而是把 Story 4.x 已完成的总览、详情和报告配置重排为“左侧可折叠组织树 + 右侧当前部门分析区”。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.2: 部门用量页重构为树驱动分析视图]
- 当前真实代码已经实现该目标：`EnterpriseUsageOverview` 使用组织树、URL search 的 `dept_id`、当前部门 summary/detail 查询和右侧分析区组件，`EnterpriseUsageContent` 渲染 `Department Tree` 与 `Current Department Analysis` 双栏布局。[Source: web/default/src/features/enterprise-usage/index.tsx]
- 本故事必须保留 4.x 已确定的多部门重复计入口径、未归属行、summary/detail/report API、最近日志入口和报告配置；任何“为了树驱动重排而重造聚合/日志/报表”的实现都是回归。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]

### Previous Story Intelligence

- Story 6.1 已建立可复用的组织树交互、`dept_id` URL 恢复、祖先展开、无效部门回退和树工具。6.2 应直接复用这些工具，而不是复制另一套树状态算法。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
- Story 4.2 建立了 `enterprise-usage` feature、认证路由、summary API 消费、时间范围解析、固定 disclaimer 和未归属部门展示。6.2 的树驱动分析区必须在这些能力上重排，不得回退到 dashboard 或 usage-logs 页面。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- Story 4.3 增加了部门详情、用户排行、模型分布、趋势和最近日志入口；6.2 应把详情变成当前部门分析区的核心内容，而不是继续依赖“总览表 -> View Details”的 drill-down 思路。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- Story 4.5 增加了 `Scheduled Usage Reports` 配置卡片；6.2 需要把它内联保留在分析骨架中，并明确其当前租户/报告作用域，不能拆回独立主 tab。[Source: _bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md]

### Architecture Guardrails

- 部门用量归因采用成员消费视角：按 `enterprise_user_departments` 展开，多部门用户重复计入，部门间不可直接相加；UI 必须固定展示 `enterprise.usage.multi_dept_disclaimer`。[Source: _bmad-output/planning-artifacts/architecture.md#Department Usage Attribution (V1.1 新增)]
- 用量看板必须依赖 `enterprise_usage_snapshots` 聚合表和 `/api/enterprise/usage/department-summary` / `department-detail`，不允许为了页面重排改成前端扫全量 `logs` 或服务端在线全量扫描。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- Default 是企业能力完整交付面；Classic 只提供必要管理入口，不做与 Default 等价的深度树驱动重构。[Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture]
- TanStack Query queryKey 使用 feature 级键，如 `['enterprise', 'usage', ...]`；mutation/invalidation 保持精确，不粗暴刷新整个 `['enterprise']` 模块。[Source: _bmad-output/planning-artifacts/architecture.md#State & Data Fetching]
- 权限真源仍是后端 `middleware.EnterpriseAdmin()`。Story 文案中的“经营负责人”是业务角色描述，6.2 不自行扩展 role、middleware 或前端绕权逻辑。[Source: router/enterprise-router.go; middleware/enterprise_admin.go]
- 不得修改或移除任何 `new-api` / `QuantumNous` 受保护标识。[Source: AGENTS.md#Rule 5: Protected Project Information — DO NOT Modify or Delete]

### Existing Code Touchpoints

- `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`
  - 已挂载 `enterpriseUsageSearchSchema` 作为 `validateSearch`，并保留 Admin 访问守卫。
- `web/default/src/features/enterprise-usage/index.tsx`
  - 已导入 `DepartmentTree`、`useDepartmentTree()` 和 6.1 树工具；`EnterpriseUsageOverview` 负责 search 归一化、组织树展开、summary/detail/report 查询和当前部门切换。
  - `EnterpriseUsageContent` 已采用双栏布局：左侧 `Department Tree`，右侧 `Current Department Analysis`，并在右侧内联当前部门上下文、摘要卡片、详情面板、peer snapshot、分析说明和报告配置卡片。
- `web/default/src/features/enterprise-usage/api.ts`
  - 已保持 summary/detail/report/export API 包装与 `['enterprise', 'usage', ...]` query key；summary/export 支持 `departmentId`、`includeDescendants`、排序和 `tenantId`。
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - 已覆盖 search schema、当前部门归一化、query key、固定 disclaimer、树驱动分析区文案、报告配置卡、最近日志入口和回归边界。
- `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
  - 已包含 6.2 新增 UI 文案，继续使用项目既有扁平 i18n key 风格。

### Implementation Guidance for Future Maintenance

- 保持 `dept_id` 为当前部门真源；不要重新引入“总览行选中 + 详情面板本地 department state”双真源。
- 组织树数据加载完成前不要过早清理 URL search；只有在真实树数据为空或目标节点不存在时才回退/清理，以免刷新分享链接时丢失上下文。
- 切换部门应清理旧部门绑定的 `log_user`，但不应重置时间范围、租户、summary 排序和 include-descendants 这类跨部门分析上下文。
- `Scheduled Usage Reports` 当前仍按租户作用域工作；若未来改为部门级报告，需要在 report API、query key、UI 说明和测试中一起显式升级，不能只改显示文案。
- 最近日志入口只能传筛选上下文；禁止在 6.2 范围内修改 `model/log.go`、`controller/log.go` 或 relay/计费路径。

### Scope Boundaries

- 本故事聚焦 Default `enterprise-usage` 页面结构与当前部门上下文。
- 本故事不新增部门用量后端聚合模型，不新增 usage detail API，不新增 report job API；这些均已由 Story 4.1-4.5 覆盖。
- 本故事不实现新的 Classic 深度 UI、不改变日志系统、不修改 relay 协议、不触碰 `pkg/billingexpr/**`。
- 本故事不扩展“经营负责人”权限模型；如需部门负责人查看能力，应通过后续权限设计处理。

### Testing Requirements

- 已有测试应持续覆盖：
  - `dept_id` 缺失、无效和空树场景下的 search 归一化；
  - 当前部门分析区、部门树、分析说明和报告配置卡片的静态渲染；
  - summary/detail/report/export query key 与参数不互相污染；
  - 最近日志入口保留部门上下文并优先使用有效 `log_user`；
  - i18n sync 后 `en/zh/fr/ru/ja/vi` 缺失数为 0。
- 最低验证命令：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run test:e2e`
  - `cd web/default && bun run i18n:sync`
- 如果未来改动 usage 后端契约，补跑：
  - `GOCACHE=$(pwd)/.cache/go-build go test ./controller/enterprise ./service/enterprise ./tests/api -run 'Usage|EnterpriseUsage'`

### Checklist Validation Notes

- Critical misses fixed: 原故事文件此前 `Status: done` 但任务仍为 `[ ]`，且缺少真实实现记录、代码触点、验证结果和 checklist 复核说明；本次已对齐为已完成故事记录。
- Reinvention prevention: 明确复用 4.x usage 能力与 6.1 树工具，禁止新建第二套树接口、usage 聚合链路、日志语义或报告系统。
- File structure prevention: 明确主要修改面为 `web/default/src/features/enterprise-usage/` 与其 route/test/i18n，不扩张到 `dashboard/`、`usage-logs/`、relay 或 billing。
- Regression prevention: 明确保留 `enterprise.usage.multi_dept_disclaimer`、未归属行、多部门重复计入、最近日志筛选上下文、report 租户作用域和 feature-scoped query key。
- LLM optimization: 将要求按 AC、任务、代码触点、边界和验证命令组织，去掉模糊“重排页面”表述，改为可执行的文件级约束。

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:893)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:288)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:429)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:869)
- [4-2-display-department-usage-overview-dashboard.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md:1)
- [4-3-view-department-usage-details.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/4-3-view-department-usage-details.md:1)
- [4-5-configure-and-send-scheduled-usage-reports.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md:1)
- [6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md:1)
- [web/default/src/routes/_authenticated/enterprise-usage/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routes/_authenticated/enterprise-usage/index.tsx:1)
- [web/default/src/features/enterprise-usage/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-usage/index.tsx:1)
- [web/default/src/features/enterprise-usage/api.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-usage/api.ts:1)
- [web/default/src/features/enterprise-usage/enterprise-usage.test.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-usage/enterprise-usage.test.tsx:1)
- [web/default/src/features/enterprise-organization/lib/tree-utils.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/lib/tree-utils.ts:1)
- [web/default/src/features/enterprise-organization/hooks/use-department-tree.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/hooks/use-department-tree.ts:1)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:1)
- [middleware/enterprise_admin.go](/Users/hq-it/repository/github/huaqingai/new-api/middleware/enterprise_admin.go:1)
- [model/log.go](/Users/hq-it/repository/github/huaqingai/new-api/model/log.go:1)
- [controller/log.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/log.go:1)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-05-31：story-automator 记录两次自动 create-story 会话未落地 6.2 文件；随后手工接管 create-story，写入 `6-2-rebuild-department-usage-as-tree-driven-analysis-view.md` 并推进 sprint 状态。[Source: _bmad-output/story-automator/orchestration-6-20260531-065455.md]
- 2026-05-31：Story 6.2 implementation 已完成，story-automator 记录 commit `0e95ffa732cfaf23b7a90ad12efa2d977392112d` 并确认 sprint 状态完成。[Source: _bmad-output/story-automator/orchestration-6-20260531-065455.md]
- 2026-06-03：按用户要求重新执行 `.claude/skills/bmad-create-story` workflow 的读取、发现、模板和 checklist 复核；当前 story/sprint 均为 `done`，因此只补强故事上下文与验证记录，不做状态降级。
- `python3 _bmad/scripts/resolve_customization.py --skill .claude/skills/bmad-create-story --key workflow` 因 Python 版本缺少 `tomllib` 失败，已按 skill fallback 手工解析 customization。
- 已读取并复核真实实现：`enterpriseUsageSearchSchema`、`normalizeEnterpriseUsageSearch()`、组织树复用、`EnterpriseUsageContent` 双栏布局、`DepartmentUsageDetailPanel`、`DepartmentUsageReportCard`、query key、export 参数和最近日志入口。
- 2026-06-03：按用户要求执行 `.claude/skills/bmad-dev-story` workflow；所有 Tasks/Subtasks 已为 `[x]`，无剩余未完成实现项，复核 AC 1/2/3 与当前实现一致。
- 2026-06-03：验证通过 `cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`。额外执行 `bun run lint` 未进入代码检查阶段，ESLint 因现有 `brace-expansion` override 与 `minimatch@10` 期望的 `expand` 导出不兼容而启动失败。
- 2026-06-03：按 `.claude/skills/bmad-story-automator-review` workflow 执行自动 review；发现并修复 search 归一化边界问题，验证通过 `cd web/default && bun run rsbuild build --config rsbuild.test.config.ts && node .tmp/tests/enterprise-usage.mjs`、`cd web/default && bun run typecheck`。

### Completion Notes List

- 已将部门用量页重构为树驱动分析视图：左侧组织树作为主导航，右侧当前部门分析区承接用量概览、趋势、模型分布、用户排行和最近日志入口。
- 已将 `dept_id` 作为页面一级状态保存在 URL search 中，支持缺失/失效回退、树展开同步和部门切换后清理旧部门 `log_user`。
- 已保留并内联当前部门分析说明与 `Scheduled Usage Reports` 配置卡片，避免总览/配置继续作为割裂主视图。
- 已保持 Story 4.x 的用量归因、summary/detail/report API、最近日志跳转和 report 租户作用域；未修改日志、relay、billing 或 usage 聚合后端语义。
- 已同步 Default 六语言文案，并通过前端 typecheck、静态测试和 i18n sync；本故事未引入新的后端 API 改动。
- 本次 dev-story 复核未发现需要追加业务代码修改；Story 6.2 已处于 `done`，任务勾选、文件清单和验证记录已对齐。

### File List

- `_bmad-output/implementation-artifacts/6-2-rebuild-department-usage-as-tree-driven-analysis-view.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `_bmad-output/story-automator/orchestration-6-20260531-065455.md`
- `web/default/src/routes/_authenticated/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/api.ts`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/features/enterprise-usage/hooks/use-department-usage-summary.ts`
- `web/default/src/features/enterprise-usage/hooks/use-department-usage-detail.ts`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/features/enterprise-organization/components/DepartmentTree.tsx`
- `web/default/src/features/enterprise-organization/hooks/use-department-tree.ts`
- `web/default/src/features/enterprise-organization/lib/tree-utils.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`

### Change Log

- 2026-05-31: 手工接管自动 create-story 停滞，创建 Story 6.2 文档并明确树驱动分析视图的文件范围、AC、复用约束和测试要求。
- 2026-05-31: 完成 Story 6.2 实现，重构 `enterprise-usage` 为左侧组织树 + 右侧当前部门分析区，保留 summary/detail/report/最近日志能力并验证通过。
- 2026-06-03: 按 `.claude/skills/bmad-create-story` workflow 重新复核 Story 6.2；修正故事文件中 `done` 状态与未勾选任务不一致的问题，补齐真实代码触点、已完成记录、checklist validation notes 和最终上下文。
- 2026-06-03: 按 `.claude/skills/bmad-dev-story` workflow 复核 Story 6.2，确认无剩余 `[ ]` 任务并补记本次 typecheck、e2e、i18n 与 lint 启动失败原因。
- 2026-06-03: 按 `.claude/skills/bmad-story-automator-review` workflow 完成自动 review；修复组织树加载/空态下 URL search 归一化问题，并补充回归测试。

### Senior Developer Review (AI)

- Review status: auto-fixed; no critical issues remain. Story remains `done`; sprint-status already records `6-2-rebuild-department-usage-as-tree-driven-analysis-view: done`.
- [MEDIUM][Fixed] `normalizeEnterpriseUsageSearch()` cleared `dept_id` and `log_user` whenever the tree array was empty, including the initial loading/error window before `GET /api/enterprise/departments/tree` had resolved. This could drop shared `dept_id` links on refresh and violate the current-department-as-page-state behavior. Fixed by passing tree resolution state and preserving search while the tree is unresolved. [web/default/src/features/enterprise-usage/index.tsx]
- [MEDIUM][Fixed] The empty-tree cleanup task claimed `include_descendants` was cleared, but the URL sync effect only wrote `dept_id` and `log_user`, so a true empty tree could leave stale descendant scope in the URL. Fixed with explicit search-sync helpers and tests covering the cleanup payload. [web/default/src/features/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]
- [LOW][Noted] Git/story file-list reality is noisy because prior automation left additional story/test-summary/orchestration/test changes uncommitted. The application source fix in this review is limited to `web/default/src/features/enterprise-usage/index.tsx` plus targeted frontend test coverage.
- Validation: `cd web/default && bun run rsbuild build --config rsbuild.test.config.ts` passed with the existing optional `supports-color` warning from `debug`; `cd web/default && node .tmp/tests/enterprise-usage.mjs` passed 25/25; `cd web/default && bun run typecheck` passed.
