---
baseline_commit: 270b9d393c9adc4a542b4e1952db0925564faa1b
---

# Story 6.1: 企业组织页重构为树驱动治理视图

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 企业组织页以“左侧可折叠组织树 + 右侧当前部门工作区”的结构工作,
so that 我可以围绕当前部门连续完成成员、预算和治理操作。

## Acceptance Criteria

1. **Given** 管理员进入企业组织页  
   **When** 页面加载完成  
   **Then** 页面左侧显示可折叠组织树  
   **And** 右侧显示当前选中部门的摘要、成员列表、预算/资源池摘要和相关操作区。
2. **Given** 管理员在组织树中选中某个部门  
   **When** 页面状态更新  
   **Then** 当前选中部门成为页面一级状态  
   **And** 右侧所有摘要、列表和操作都切换到该部门上下文，而不是进入独立 detail drill-down 页面。
3. **Given** 页面存在当前选中部门  
   **When** 管理员刷新页面或复制当前页面链接  
   **Then** 系统能够恢复当前选中部门  
   **And** 默认展开根节点与当前选中部门的祖先路径。

## Tasks / Subtasks

- [x] 以“部门选中态”为核心重构 `enterprise-organization` 页面骨架，替换当前四个割裂 tab + 手工 ID 输入工作流 (AC: 1, 2, 3)
  - [x] 在 `web/default/src/routes/_authenticated/enterprise-organization/index.tsx` 为企业组织页增加 `validateSearch`，引入搜索参数 schema（至少包含 `dept_id`，必要时增加 `budget_id` 或右侧子区段状态），并使用 TanStack Router search 作为当前部门的持久状态来源。[Source: web/default/src/routes/_authenticated/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/index.tsx]
  - [x] 在 `web/default/src/features/enterprise-organization/index.tsx` 将当前 `Tabs` 结构重构为稳定的双栏布局：左侧树导航、右侧部门工作区；当前部门必须成为页面一级状态，而不是散落在各卡片本地 state 或输入框中。[Source: web/default/src/features/enterprise-organization/index.tsx]
  - [x] 保留现有 `SectionPageLayout`、`Card`、`Empty`、`Skeleton`、`StatusBadge` 视觉体系，不新增第二套页面框架，也不要把组织页改造成新的详情路由栈。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/planning-artifacts/architecture.md#Process Patterns]

- [x] 升级部门树交互，使其支持折叠/展开、选中、高亮与祖先路径恢复，而不仅是静态只读表格 (AC: 1, 2, 3)
  - [x] 在 `web/default/src/features/enterprise-organization/components/DepartmentTree.tsx` 引入可折叠树行为，至少支持根节点和任意分支节点展开/收起，并为当前选中部门提供明确高亮状态。[Source: web/default/src/features/enterprise-organization/components/DepartmentTree.tsx; _bmad-output/planning-artifacts/epics.md#Story 6.1: 企业组织页重构为树驱动治理视图]
  - [x] 新增 `web/default/src/features/enterprise-organization/lib/tree-utils.ts` 或等价工具，负责从现有 `DepartmentTreeNode[]` 计算祖先路径、默认展开集合、节点查找和选中部门回退策略，避免把树算法写进组件 render 流程。[Source: _bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure; service/enterprise/department.go]
  - [x] 保持现有部门树 API `GET /api/enterprise/departments/tree` 不变，优先复用 `useDepartmentTree()` 与 `['enterprise','organization','department-tree']` queryKey；本故事目标是视图重构，不是新增第二棵树接口。[Source: web/default/src/features/enterprise-organization/hooks/use-department-tree.ts; web/default/src/features/enterprise-organization/api.ts; controller/enterprise/department.go]

- [x] 将右侧区域重组为“当前部门工作区”，把摘要、成员、预算和操作统一到部门上下文内 (AC: 1, 2)
  - [x] 右侧顶部提供当前部门摘要卡，至少展示部门名称、ID、父级信息、状态、来源、外部 ID、同步状态和历史名称摘要，避免管理员切换后失去上下文。[Source: web/default/src/features/enterprise-organization/components/DepartmentTree.tsx; web/default/src/features/enterprise-organization/types.ts]
  - [x] 将当前 `UserDepartmentsPanel`、`DepartmentMembersPanel`、`DepartmentBudgetPanel` 中依赖手工输入 `Department ID` / `User ID` 的主路径改为读取当前选中部门；成员列表、预算列表、预算详情和 allocation 列表都要在当前部门工作区内联切换。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]
  - [x] 保留必要的用户搜索/选择能力，但 `Department ID` / `User ID` 不能再作为默认治理入口；页面主流程必须由当前部门驱动，而不是让管理员回退到 ID 驱动的表单思路。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.3: 部门上下文驱动的成员与预算操作]
  - [x] 预算区域继续复用 Story 3.5 已交付的预算池列表、详情、wallet 明细和 allocation 撤销链路，不得重新发明预算展示模型，也不要回退成只读单卡片。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md; web/default/src/features/enterprise-organization/index.tsx]

- [x] 用路由 search 和默认展开逻辑保证刷新/分享链接后能恢复当前部门上下文 (AC: 3)
  - [x] 进入页面时，如果 search 中存在合法 `dept_id`，则恢复该部门选中态，并自动展开根节点与该节点祖先链；如果 `dept_id` 缺失，则选择稳定默认节点（优先首个根节点或第一条可见节点），同时把 search 同步到 URL。[Source: web/default/src/features/enterprise-usage/index.tsx; service/enterprise/department.go]
  - [x] 如果 search 中的 `dept_id` 已不存在、无权限或当前树数据未包含该节点，页面必须做稳定降级：回退到首个可见部门并更新 search，而不是保持损坏状态或白屏。[Source: controller/enterprise/department.go; router/enterprise-router.go]
  - [x] 祖先路径展开逻辑必须基于当前树数据动态计算，不能硬编码“展开全部”；Epic 明确要求默认展开根节点与当前节点祖先路径，而不是把大树一次性全展开。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.1: 企业组织页重构为树驱动治理视图]

- [x] 为当前组织页数据请求补齐部门上下文驱动的 query 组织方式，避免粗粒度失效和状态漂移 (AC: 1, 2, 3)
  - [x] 为成员、预算、预算详情、allocation 等查询抽出基于当前 `dept_id` 的 queryKey/查询 hooks，继续遵守 `['enterprise', '<feature>', '<scope>', ...]` 规则，不允许用 `invalidateQueries({ queryKey: ['enterprise'] })` 粗暴刷新整个企业模块。[Source: _bmad-output/planning-artifacts/architecture.md#Communication Patterns; web/default/src/features/enterprise-organization/api.ts]
  - [x] 保留 Story 3.5 中预算 detail 与 allocation 的联动，但要确保部门切换时能够重置无效的 `selectedBudgetId`、避免将上一个部门的预算池残留到下一个部门上下文。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]
  - [x] 任何新引入的本地 UI state 都应围绕“当前部门上下文”服务，并与 search/query 数据关系清晰，避免 render 期间写 state 或形成多个相互竞争的部门真源。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md#Senior Developer Review (AI)]

- [x] 让 Classic 主题具备最小可用的一致入口，但不要把 6.1 的深度重构范围扩张到 Classic 全量重写 (AC: 1, 2)
  - [x] 检查 `web/classic/src/services/enterprise.js` 与 Classic 企业入口现状，若已有组织页入口，则至少保证它不会因为 Default 端的 API/字段复用调整而失效；如需补入口，保持 Classic 既有 `pages/` / `services/` 模式做最小接线。[Source: web/classic/src/services/enterprise.js; _bmad-output/planning-artifacts/architecture.md#Structure Patterns]
  - [x] 不在本故事中对 Classic 复制 Default 的树驱动深度交互；架构已说明 Classic 只提供必要管理入口，不做同等深度重构。[Source: _bmad-output/planning-artifacts/architecture.md#Important Decisions]

- [x] 补齐 i18n、测试与回归，保护现有组织/预算/成员链路不被页面重构破坏 (AC: 1, 2, 3)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，至少覆盖：树节点折叠/展开、选中高亮、刷新后恢复 `dept_id`、祖先路径默认展开、部门切换后右侧上下文切换、无效 `dept_id` 降级回退。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 为新增 route search schema 和 URL 恢复逻辑补前端测试，确认组织页与仓库既有 `validateSearch` / `useSearch` / `navigate({ search })` 用法一致。[Source: web/default/src/routes/_authenticated/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/index.tsx]
  - [x] 所有新增文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，继续使用 `t('English source')` 或项目已有扁平 key 风格，不得只改英文源文件后漏同步其他 locale。[Source: AGENTS.md#Internationalization i18n]
  - [x] 运行最小前端回归：`cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`；如涉及企业接口契约读取方式变化，补充 `controller/enterprise/department_test.go` 或等价 API 测试验证现有树接口未回归。[Source: AGENTS.md#Rule 3: Frontend — Prefer Bun; controller/enterprise/department.go]

## Dev Notes

### Context Loaded

- 已加载 `{epics_content}`：`_bmad-output/planning-artifacts/epics.md`。
- 已加载 `{architecture_content}`：`_bmad-output/planning-artifacts/architecture.md`。
- 已加载 `sprint-status.yaml`，确认目标 story key 为 `6-1-rebuild-enterprise-organization-as-tree-driven-governance-view`，原状态为 `backlog`。
- 未发现独立 UX 文档；当前 UX 约束来自 Epic 6、Architecture 与既有企业功能实现。
- 未发现 `project-context.md` 持久事实文件，因此本故事以 AGENTS.md、规划文档、既有故事与当前真实代码为准。

### Story Intent and Current Gap

- 当前 `EnterpriseOrganization` 页面仍是四个 tab：`Department Tree`、`User Departments`、`Department Members`、`Department Budget`。组织树只是静态展示，其余治理动作依赖手工输入 `User ID`、`Department ID`，这与 Story 6.1 要求的“当前部门成为页面一级状态”明显不符。[Source: web/default/src/features/enterprise-organization/index.tsx]
- Story 6.1 不是新增企业后端核心能力，而是把 Epic 1 和 Epic 3 已有组织/成员/预算能力重排成树驱动治理视图。因此优先复用现有企业组织树 API、成员 API、预算列表/详情 API 和 allocation API，而不是再造服务层/路由层。[Source: web/default/src/features/enterprise-organization/api.ts; router/enterprise-router.go]
- 这次重构的成功标准不是“树能点开”，而是“当前部门”成为组织页真源，右侧所有摘要、成员、预算与操作都围绕这个真源切换，并且刷新或分享链接后可恢复到相同上下文。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.1: 企业组织页重构为树驱动治理视图]

### Previous Story Intelligence

- Story 1.1 已经建立了企业组织树的后端模型、树接口、Default `enterprise-organization` feature、`useDepartmentTree()` 和基础静态树组件。本故事应直接在这条链路上增强可交互树和组织页布局，而不是新建第二个组织树 feature。[Source: _bmad-output/implementation-artifacts/1-1-view-enterprise-department-tree.md]
- Story 3.5 已经把预算区重构为“预算池列表 + 详情 + wallet 明细 + allocation 表”，并强调 query invalidation 必须精确到 feature 级。本故事应复用这些预算组件/查询行为，只把它们嵌入当前部门工作区，不要回退成老式表单/单卡片。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]
- Story 4.2 已经在 `enterprise-usage` 中证明了 TanStack Router search 驱动页面一级状态的实现模式：`validateSearch`、`useSearch({ from })`、`navigate({ search })`、搜索参数持久化到 URL。本故事应直接借鉴这一路径实现 `dept_id` 恢复逻辑。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; web/default/src/routes/_authenticated/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/index.tsx]
- Story 3.5 的审查还暴露过 React 渲染期直接写 state 的问题；Story 6.1 在做“当前部门切换 -> 预算/成员联动”时必须避免在 render 期间同步 `selectedBudgetId`、`dept_id` 等状态。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md#Senior Developer Review (AI)]

### Architecture Guardrails

- Default 前端是企业能力的主要交付面，企业功能按独立 feature 切分；不要把组织治理能力塞进现有 `users`、`usage-logs` 或 `dashboard` 页面。[Source: _bmad-output/planning-artifacts/architecture.md#Structure Patterns]
- TanStack Query queryKey 统一使用 `['enterprise', '<feature>', '<scope>', ...]`，mutation invalidate 精确到 feature 级；禁止全量失效 `['enterprise']`。[Source: _bmad-output/planning-artifacts/architecture.md#Communication Patterns]
- Classic 主题只提供必要管理入口，不做与 Default 同等深度的重构，因此本故事的主要实现面应集中在 `web/default/src/features/enterprise-organization/`。[Source: _bmad-output/planning-artifacts/architecture.md#Important Decisions]
- 企业组织相关 API 已固定在 `/api/enterprise/**`，并由 `router/enterprise-router.go` 注册；6.1 不需要扩展 relay、billing、usage log 或新 session 体系。[Source: router/enterprise-router.go; AGENTS.md]
- 不得修改或移除任何 `new-api` / `QuantumNous` 受保护标识。[Source: AGENTS.md#Rule 5: Protected Project Information — DO NOT Modify or Delete]

### Existing Code Touchpoints

- `web/default/src/routes/_authenticated/enterprise-organization/index.tsx`
  - 当前只有简单的登录校验和 `EnterpriseOrganization` 组件挂载，没有 `validateSearch`；Story 6.1 的 URL 恢复逻辑应从这里接入。
- `web/default/src/features/enterprise-organization/index.tsx`
  - 当前顶层是 `Tabs`，其中 `DepartmentMembersPanel` 与 `DepartmentBudgetPanel` 大量依赖 `Department ID` / `User ID` 输入框；这是本故事最核心的重构面。
- `web/default/src/features/enterprise-organization/components/DepartmentTree.tsx`
  - 当前组件按表格递归渲染树，只展示层级和状态，不支持折叠、选中和展开状态管理；需要升级为可交互树而不改变底层 API 数据结构。
- `web/default/src/features/enterprise-organization/hooks/use-department-tree.ts`
  - 已封装树接口查询并返回 `response.data ?? []`，可继续作为树数据真源。
- `web/default/src/features/enterprise-organization/api.ts`
  - 已具备成员、预算、allocation 等所需接口。Story 6.1 更像是重新组织这些查询和页面结构，而不是新增大量 API。
- `controller/enterprise/department.go` / `service/enterprise/department.go`
  - 后端树接口已经支持管理员/部门管理员视角的树返回，并在服务层处理孤儿节点与循环保护。本故事应尊重现有 API 语义，不需要新增树形后端状态接口。

### Implementation Guidance

- 推荐先定义 `enterpriseOrganizationSearchSchema`，至少支持 `dept_id?: number`。随后在组织页内部通过 `useSearch` 读取当前部门，使用 `navigate({ search })` 更新它，形成页面真源。
- 建议把树状态拆成两类：
  - URL 持久状态：当前选中 `dept_id`
  - 本地 UI 状态：`expandedIds`
  其中 `expandedIds` 初始值基于树数据和 `dept_id` 计算，必须至少包含根节点与当前部门祖先链。
- 对于部门切换后的预算联动，应该显式清理已无效的 `selectedBudgetId`，并让预算详情基于当前部门的预算池列表重新选择默认项，避免跨部门残留。
- 右侧工作区可以保留局部 section/tabs，但这些 section 必须共享当前部门上下文；不能再出现“切换到 members/budgets tab 后需要重新输入 Department ID”这种割裂流程。
- 部门管理员权限也必须可用：当前 `GET /api/enterprise/departments/tree` 已按角色收窄到可管理部门子树；前端不要假定所有管理员都能看到全树。

### Scope Boundaries

- 本故事聚焦企业组织页重构与当前部门上下文建立。
- 本故事不实现新的部门 CRUD、钉钉同步、部门用量分析页重构、账号命名规则修正或企业权限模型变更。
- 本故事不应修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`。
- 若需要为 Story 6.3 预留“从成员进入预算操作”的页面结构，可以在 6.1 中建立当前部门工作区骨架，但不要提前扩展成完整 Story 6.3 范围。

### Testing Requirements

- 前端测试必须覆盖：
  - 三层以上树在折叠/展开后仍可选中正确节点；
  - URL 中已有 `dept_id` 时页面可恢复并展开祖先链；
  - 无效 `dept_id` 时回退到首个可见节点；
  - 部门切换后右侧成员/预算区域切换到新上下文；
  - 空树状态仍保留现有空态与钉钉入口。
- 最低验证命令：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run test:e2e`
  - `cd web/default && bun run i18n:sync`
- 如果对树接口消费方式有收敛调整，补跑：
  - `GOCACHE=$(pwd)/.cache/go-build go test ./controller/enterprise ./service/enterprise`

### References

- [epics.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/epics.md:186)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:508)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:575)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:677)
- [architecture.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture.md:838)
- [1-1-view-enterprise-department-tree.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/1-1-view-enterprise-department-tree.md:1)
- [3-5-display-budget-pool-and-wallet-state.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md:1)
- [4-2-display-department-usage-overview-dashboard.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md:1)
- [web/default/src/routes/_authenticated/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routes/_authenticated/enterprise-organization/index.tsx:1)
- [web/default/src/features/enterprise-organization/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/index.tsx:274)
- [web/default/src/features/enterprise-organization/components/DepartmentTree.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/components/DepartmentTree.tsx:1)
- [web/default/src/features/enterprise-organization/hooks/use-department-tree.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/hooks/use-department-tree.ts:1)
- [web/default/src/features/enterprise-organization/api.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-organization/api.ts:1)
- [web/default/src/features/enterprise-usage/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/enterprise-usage/index.tsx:320)
- [web/default/src/routes/_authenticated/enterprise-usage/index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routes/_authenticated/enterprise-usage/index.tsx:1)
- [controller/enterprise/department.go](/Users/hq-it/repository/github/huaqingai/new-api/controller/enterprise/department.go:1)
- [service/enterprise/department.go](/Users/hq-it/repository/github/huaqingai/new-api/service/enterprise/department.go:1)
- [router/enterprise-router.go](/Users/hq-it/repository/github/huaqingai/new-api/router/enterprise-router.go:1)
- [web/classic/src/services/enterprise.js](/Users/hq-it/repository/github/huaqingai/new-api/web/classic/src/services/enterprise.js:1)

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 已按要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- `_bmad/scripts/resolve_customization.py` 受当前 Python 版本限制无法运行，已按 skill fallback 手工解析 `.agents/skills/bmad-create-story/customize.toml` 与 `_bmad/bmm/config.yaml`。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `6-1-rebuild-enterprise-organization-as-tree-driven-governance-view`，并确认本次需把 `epic-6` 从 `backlog` 调整为 `in-progress`。
- 已选择性加载并分析 `epics.md`、`architecture.md`、`docs/api-contracts-server.md`、既有实现文件和相关已完成故事（1.1、3.5、4.2）。
- 已检查当前真实实现，确认组织页的主要差距是静态树 + tab 分裂 + 手工 ID 输入，而不是后端能力缺失。
- 按 `bmad-dev-story` workflow 补齐前端实现与回归：完成路由 search schema、树驱动双栏工作区、树工具、精确 query key 和预算上下文联动。
- 执行 `cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`，全部通过；`i18n` 报告显示 `en/zh/fr/ru/ja/vi` 缺失数均为 `0`。
- 为恢复全量前端回归，顺手校正了既有 `enterprise-alerts` 与 `enterprise-usage` 测试中的 search/time-window 断言，使其与当前实现保持一致。

### Completion Notes List

- 已将企业组织页重构为“左侧树导航 + 右侧当前部门工作区”，当前部门由 TanStack Router search (`dept_id` / `budget_id`) 驱动并支持刷新恢复。
- 已为部门树补齐折叠/展开、选中高亮、祖先路径默认展开、无效部门回退和默认根节点展开逻辑，并抽离到 `tree-utils.ts`。
- 已把成员、预算、预算详情和 allocation 查询收紧为按部门上下文组织的精确 query key，同时修复预算选择在部门切换时的跨部门残留问题。
- 已扩展组织页前端测试，覆盖 search schema、树恢复/回退、展开状态同步、预算选择归一化和工作区上下文渲染。
- 已完成前端验证：`bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync` 全部通过；本故事未改动企业后端树接口，因此未补充 Go API 回归。

### File List

- `_bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/components/DepartmentTree.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/lib/tree-utils.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/routes/_authenticated/enterprise-organization/index.tsx`

### Senior Developer Review (AI)

- 发现并自动修复 2 个高/中优先级问题：
  - 组织树为空时，页面此前不会清理 URL 中残留的 `dept_id` / `budget_id`，分享或刷新后会保留失效状态；现已在组织页 search 归一化逻辑中修复，并补充测试覆盖空树与无效部门回退。
  - Story 6.1 新增的组织页文案此前没有写入 `en/zh/fr/ja/ru/vi` locale 文件；现已补齐 19 个新增键并重新执行 `bun run i18n:sync`。
- 针对 `enterprise-alerts` 与 `enterprise-usage` 测试改动做了范围审查：这些断言修正对应的是各自 feature 既有实现与旧测试之间的偏差，基线提交中实现已包含相关行为，因此它们不是由 6.1 组织页重构直接引起的必要 fallout，更接近顺带修复的既有测试漂移。
- 未发现剩余 CRITICAL 问题；Acceptance Criteria 1/2/3 与组织页当前实现一致，故事可标记为 `done`。

### Change Log

- 2026-05-31 15:38 +0800: 完成 Story 6.1 组织页树驱动重构、精确 query key/预算上下文联动、组织页测试扩展，并校正全量前端回归中的既有断言以恢复 `typecheck`/`test:e2e`/`i18n:sync` 绿灯。
- 2026-05-31 15:51 +0800: 执行 story-automator review，修复空树 URL 残留状态与 6.1 漏同步 i18n 文案问题；确认 `enterprise-alerts` / `enterprise-usage` 测试调整属于独立测试修正而非组织页重构必然 fallout，并重新验证 `typecheck` / `test:e2e` / `i18n:sync`。
