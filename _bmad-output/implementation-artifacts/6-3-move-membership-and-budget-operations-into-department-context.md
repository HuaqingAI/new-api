---
baseline_commit: 0e95ffa732cfaf23b7a90ad12efa2d977392112d
---

# Story 6.3: 部门上下文驱动的成员与预算操作

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 从当前选中部门直接进入成员维护、预算池查看和 wallet 分配,
so that 我不需要再依赖输入 Department ID 或 User ID 才能完成治理操作。

## Acceptance Criteria

1. **Given** 管理员已选中某个部门  
   **When** 管理员查看该部门右侧工作区  
   **Then** 可以直接进入该部门成员维护、预算池列表和 wallet 分配操作  
   **And** 这些操作都继承当前部门上下文。
2. **Given** 管理员需要向成员分配预算  
   **When** 管理员从当前部门工作区发起分配  
   **Then** 系统默认通过当前部门成员列表或搜索选择目标成员  
   **And** `Department ID` / `User ID` 不再作为主界面默认输入路径。
3. **Given** 管理员需要查看某成员在当前部门的治理信息  
   **When** 从成员列表进入操作  
   **Then** 系统能在当前部门上下文中承接成员归属、预算分配和相关操作  
   **And** 不要求管理员跳回全局用户页或手工填写对象 ID。

## Tasks / Subtasks

- [x] 把组织页右侧工作区真正收敛为“当前部门上下文驱动的成员与预算操作台”，移除手工 Department/User ID 作为主流程入口 (AC: 1, 2, 3)
  - [x] 在 `web/default/src/features/enterprise-organization/index.tsx` 中继续沿用 Story 6.1 已交付的当前部门真源，不再让 `DepartmentMembersPanel`、`DepartmentBudgetPanel`、allocation 表单依赖用户自己输入 `Department ID` 才能工作。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/index.tsx]
  - [x] 当前部门切换后，成员列表、预算池列表、预算详情、allocation 列表和 wallet 分配表单都必须自动切换到新的部门上下文，而不是保留旧部门残留状态或要求用户重置表单。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/api.ts]
  - [x] 继续保持组织页“左树 + 右工作区”骨架，不退回老式 tab 分裂流，也不新建第二个成员/预算详情路由栈。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.3: 部门上下文驱动的成员与预算操作; _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]

- [x] 将成员维护入口锚定在当前部门上下文内，避免用户回到全局 ID 驱动页面 (AC: 1, 3)
  - [x] 基于 Story 1.2 已交付的成员关系 API 与 `DepartmentMembersPanel` 现有能力，把成员列表直接绑定当前部门，不再把 `Department ID` 输入框作为主操作入口。[Source: _bmad-output/implementation-artifacts/1-2-maintain-user-department-memberships.md; web/default/src/features/enterprise-organization/api.ts; web/default/src/features/enterprise-organization/index.tsx]
  - [x] 新增/整理“从当前部门查看某成员治理信息”的 UI 流程，例如在成员表中选中成员后直接驱动预算/分配区域，或在同一工作区内展开成员侧栏/子面板；但不能要求管理员跳回全局用户页或重新输入对象 ID。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.3: 部门上下文驱动的成员与预算操作]
  - [x] 当前部门无成员或目标成员不属于当前部门时，应给出明确空态/错误提示，并保留当前部门上下文，不把页面打回无部门状态。[Source: _bmad-output/implementation-artifacts/1-2-maintain-user-department-memberships.md; web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]

- [x] 把预算池查看与 wallet 分配表单绑定到当前部门和当前成员上下文 (AC: 1, 2, 3)
  - [x] 继续复用 Story 3.5 的预算池列表/详情/阈值状态展示，以及 Story 3.2 的 quota allocation 链路，但 allocation 创建时默认使用当前部门和从当前部门成员列表选中的目标成员，不再要求用户手工填 `Department ID` 或 `Target User ID` 作为主路径。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md; _bmad-output/implementation-artifacts/3-2-allocate-department-budget-to-member-wallet.md; web/default/src/features/enterprise-organization/index.tsx]
  - [x] 预算池切换、成员切换和 allocation 表单之间的状态关系必须清晰：切换部门时清空无效成员/预算/已填 allocation 表单；切换预算池时保留当前部门与可用成员，但重置不再适用的钱包分配上下文。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md#Senior Developer Review (AI)]
  - [x] 若当前部门无预算池、成员未选中、或成员不属于当前部门，allocation 表单必须展示明确的上下文缺失说明，而不是继续裸露 ID 输入框让用户自行纠错。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.3: 部门上下文驱动的成员与预算操作]

- [x] 收敛 `enterprise-organization` 查询与本地状态，使其真正围绕“当前部门 + 当前成员”工作 (AC: 1, 2, 3)
  - [x] 继续遵守 `['enterprise', 'organization', ...]` 的 queryKey 组织方式，为成员列表、用户部门、预算池、预算详情和 allocation 列表保持精确失效；不要用 `invalidateQueries({ queryKey: ['enterprise'] })` 全刷。[Source: _bmad-output/planning-artifacts/architecture.md#Communication Patterns; web/default/src/features/enterprise-organization/api.ts]
  - [x] 如需新增 `member_id` / `user_id` 的 URL 或局部状态，必须说明它与当前部门真源的关系，并保证无效成员在部门切换后会被归一化清理。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
  - [x] 任何依赖当前部门/成员的 `useEffect` 或本地状态同步都不能在 render 期间直接写值；继续避免 Story 3.5 曾出现的 render-phase state update 问题。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md#Senior Developer Review (AI)]

- [x] 保持 Classic 最小可用入口，但不把 Story 6.3 扩张成 Classic 深度治理台重写 (AC: 1, 2, 3)
  - [x] 检查 `web/classic/src/services/enterprise.js` 与 Classic 企业入口是否受当前组织页上下文收敛影响；如需补最小兼容，仅保持必要读写入口，不复制 Default 的上下文驱动完整交互。[Source: web/classic/src/services/enterprise.js; _bmad-output/planning-artifacts/architecture.md#Important Decisions]
  - [x] Story 6.3 的主要实现面仍然是 Default `enterprise-organization` feature；Classic 只做必要兼容，不追求同等深度。[Source: _bmad-output/planning-artifacts/architecture.md#Classic Theme]

- [x] 补齐测试与 i18n，锁定“当前部门 + 当前成员”治理路径 (AC: 1, 2, 3)
  - [x] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`，至少覆盖：切换部门后成员列表与预算区同步切换、无成员/无预算池空态、从成员列表选中目标成员后 allocation 表单自动承接上下文、以及切换部门时旧成员/旧预算状态被清理。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 如新增成员上下文的 search/schema 或归一化逻辑，补纯函数级测试覆盖无效成员、部门切换后 member reset、预算池切换后 allocation reset 等边界。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
  - [x] 所有新增文案同步到 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，并运行 `bun run i18n:sync`；特别是“当前部门成员上下文”“当前成员钱包分配”类文案不能只留在英文源里。[Source: AGENTS.md#Internationalization i18n]
  - [x] 运行最小前端回归：`cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`；如改动企业 API 消费方式，补最小 API 回归确认成员与预算接口契约未破坏。[Source: AGENTS.md#Rule 3: Frontend — Prefer Bun; controller/enterprise/department.go; controller/enterprise/quota_allocation.go]

## Dev Notes

- Story 6.3 并不是新增成员关系、预算池或 quota allocation 后端核心能力，而是把 Story 1.2、3.2、3.5 已交付的成员与预算能力收拢到当前部门工作区里。[Source: _bmad-output/implementation-artifacts/1-2-maintain-user-department-memberships.md; _bmad-output/implementation-artifacts/3-2-allocate-department-budget-to-member-wallet.md; _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]
- Story 6.1 已经把组织页变成“左树 + 右工作区”，因此 6.3 的重点是让工作区里的成员维护与预算操作真正共享当前部门上下文，而不是继续沿用旧的 ID 驱动表单。[Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md; web/default/src/features/enterprise-organization/index.tsx]
- 当前 `DepartmentMembersPanel`、`DepartmentBudgetPanel`、quota allocation 区已经存在，但还保留了大量手工 ID/target user 入口。6.3 要把这些入口降级为次级辅助或彻底移除，而不是仅在 UI 上藏一下输入框。[Source: web/default/src/features/enterprise-organization/index.tsx]
- 成员上下文切换和预算上下文切换会相互影响，最容易引发的回归就是 render 期间写状态、旧部门旧成员残留、或分配表单带着上一部门的预算池提交。实现时应优先用纯函数归一化 + `useEffect` 同步，而不是在 render 分支里直接 `setState`。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md#Senior Developer Review (AI)]
- `enterprise_quota_allocations` 是配额分配的单一审计源，`UserSubscription` 则是实际参与 wallet 优先级的运行实体。6.3 的 UI 应让这两条信息链在“当前部门 -> 当前成员”路径里可理解，但不要发明新的中间实体或平行台账。[Source: _bmad-output/implementation-artifacts/3-2-allocate-department-budget-to-member-wallet.md]
- 当前权限边界仍以后端 `EnterpriseDepartmentAdmin` / `EnterpriseAdmin` 为真源，6.3 不应发明新的“成员治理只读角色”或修改 relay/billing/log 内核语义。[Source: router/enterprise-router.go; middleware/enterprise_dept_admin.go; AGENTS.md]

### Project Structure Notes

- 重点修改文件：
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - 需要时复用 `web/default/src/features/enterprise-organization/lib/tree-utils.ts`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/zh.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/vi.json`
- 保持不改或仅只读依赖：
  - `controller/enterprise/department_membership.go`
  - `controller/enterprise/quota_allocation.go`
  - `service/enterprise/department_membership.go`
  - `service/enterprise/quota_allocation.go`
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.3: 部门上下文驱动的成员与预算操作]
- [Source: _bmad-output/implementation-artifacts/1-2-maintain-user-department-memberships.md]
- [Source: _bmad-output/implementation-artifacts/3-1-create-department-budget-pool.md]
- [Source: _bmad-output/implementation-artifacts/3-2-allocate-department-budget-to-member-wallet.md]
- [Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]
- [Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-organization/api.ts]
- [Source: web/default/src/features/enterprise-organization/types.ts]
- [Source: dto/enterprise/department_membership.go]
- [Source: dto/enterprise/quota_allocation.go]
- [Source: controller/enterprise/department_membership.go]
- [Source: controller/enterprise/quota_allocation.go]
- [Source: router/enterprise-router.go]
- [Source: middleware/enterprise_dept_admin.go]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工接管 `create-story`，因为 story-automator 的 `6.3` create 会话长时间停在“已收集足够上下文，下一步开始写 story 文件”附近，但一直没有把 story artifact 落地，也没有把 `sprint-status` 从 `backlog` 推到 `ready-for-dev`。
- 已读取 Epic 6 中 Story 6.3 的 AC、Story 1.2 成员关系交付、Story 3.1/3.2/3.5 预算与 wallet 交付，以及当前 `enterprise-organization` 页面实现，确认 6.3 是对既有成员与预算操作台的上下文收敛，而不是新建后端能力。
- 已确认 6.3 的主要实现面应集中在 Default `enterprise-organization` feature，把当前部门和当前成员作为联合上下文驱动成员维护、预算池查看与 wallet 分配。
- 2026-05-31 17:07:44 +0800：运行 `cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`，均通过；额外尝试 `bun run lint` 时命中仓库当前 ESLint 依赖异常 `brace_expansion_1.expand is not a function`，未作为本 Story 阻塞项。
- 2026-06-03 01:34 +0800：按 dev-story 重新校验 Story 6.3，确认 story 与 sprint 状态均已为 `done`，Tasks/Subtasks 无剩余 `[ ]` 项；本轮未改业务代码或任务勾选。
- 2026-06-03 01:34 +0800：运行 `cd web/default && bun run typecheck`、`bun run i18n:sync`、`bun run test:e2e` 均通过；`bun run lint` 仍失败于 ESLint 依赖链 `brace_expansion_1.expand is not a function`；`GOCACHE=/private/tmp/new-api-go-build-cache go test ./...` 暴露既有非 6.3 范围失败（model 测试表缺失、Claude 文件内容转换断言、stream scanner 断言、usage report 计数、httptest 端口权限等）。
- 2026-06-03 01:52 +0800：story-automator-review 复核 Story 6.3；发现成员添加主路径仍暴露裸 `User ID` 输入，不满足 AC2/任务“User ID 不再作为主界面默认输入路径”，已自动修复为当前部门内的用户搜索选择流程。
- 2026-06-03 01:52 +0800：运行 `cd web/default && bun run i18n:sync`、`bun test src/features/enterprise-organization/enterprise-organization.test.tsx`、`bun run typecheck`，均通过；新增成员搜索文案已补齐 en/zh/fr/ja/ru/vi。

### Completion Notes List

- 手工补齐 Story 6.3 故事文档，明确“当前部门 + 当前成员”作为工作区真源，并把成员维护、预算池与 quota allocation 的上下文承接要求写清楚。
- 明确 6.3 的主要修改面集中在 `enterprise-organization` 前端工作区与测试/i18n，不扩展新的 relay、billing 或日志后端语义。
- 记录 `6.3` 自动 create 会话没有落 story artifact 的情况，供后续 automator 优化 create-step 长时间停在写入前的识别逻辑。
- 工作区现在由当前部门驱动成员列表、成员治理卡片、预算池列表和当前成员钱包分配表单，主流程不再暴露全局 `Membership Lookup` 或手工 `Target User ID` 输入。
- 部门切换、预算池切换、成员切换都会通过纯函数归一化和 `useEffect` 同步重置无效 allocation draft，避免旧部门/旧成员残留。
- 已补充 enterprise-organization 纯函数与静态渲染测试，覆盖成员归一化、allocation reset、当前成员治理空态与上下文承接；并同步新增中英法俄日越文案。
- 本轮复核未发现 Story 6.3 剩余未完成任务；前端 story 相关回归通过，仓库级 lint/go test 仍存在与本 Story 无关的既有环境或测试失败。
- Senior review 自动修复成员维护入口：`DepartmentMembersPanel` 现在通过用户搜索结果选择目标用户并添加到当前部门，query key 包含当前部门和租户上下文，不再把裸 `User ID` 输入作为添加成员默认路径。

### Senior Developer Review (AI)

Reviewer: GPT-5 Codex on 2026-06-03 01:52 +0800

Outcome: Approved after auto-fix. Story status remains `done`; sprint-status already has `6-3-move-membership-and-budget-operations-into-department-context: done`.

Findings:

- HIGH fixed: 成员维护任务标记为完成，但 `DepartmentMembersPanel` 的新增成员主路径仍是手工数字 `User ID` 输入，和 AC2 / “`User ID` 不再作为主界面默认输入路径”不一致。已改为当前部门卡片内的用户搜索、候选选择、再添加成员流程；重复成员会禁用候选项。
- MEDIUM fixed: 新增成员搜索流程必须遵守精确 query key 约束。已将用户搜索缓存键限定在 `['enterprise', 'organization', 'member-user-search', departmentId, tenantId, keyword]`，避免跨部门/租户上下文混用审查证据。
- MEDIUM fixed: 新增搜索文案需要覆盖所有 Default locale。已补齐 `en/zh/fr/ja/ru/vi` 并运行 i18n 同步。

Validation:

- `cd web/default && bun run i18n:sync`
- `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
- `cd web/default && bun run typecheck`

### File List

- `_bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`

### Change Log

- 2026-05-31：将 enterprise organization 工作区收敛为“当前部门 + 当前成员”驱动的治理流，移除主路径中的全局 Membership Lookup 与手工 Target User ID 输入，补齐状态归一化、测试与多语言文案。
- 2026-06-03：Senior review 自动修复成员添加入口，将裸 `User ID` 输入改为当前部门上下文内的用户搜索选择流程，并补齐测试与多语言文案。
