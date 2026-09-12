---
baseline_commit: 4d26552d7188be0ced41f488618d849acbdc683a
---

# Story 7.5: 支持员工额度申请、单步审批与自动分配

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 员工,
I want 在额度不足时向目标部门提交申请并获得审批,
so that 我可以通过正式治理链路获得继续工作的额度。

## Acceptance Criteria

1. **Given** 员工属于一个或多个部门  
   **When** 其发起额度申请  
   **Then** 必须显式选择目标部门与目标预算池模式/池子  
   **And** 系统按目标部门的有效负责人集合路由审批。
2. **Given** 负责人收到额度申请  
   **When** 执行审批  
   **Then** 审批为单步完成  
   **And** 不需要多人会签、代理审批或升级审批链。
3. **Given** 审批通过  
   **When** 系统执行后续处理  
   **Then** 直接复用现有预算分配链路完成自动分配  
   **And** 重复审批或重复回调不会创建重复 allocation。

## Tasks / Subtasks

- [x] 建立员工额度申请的持久化模型、状态机与幂等约束，避免把申请流硬塞进现有 allocation/delegation 表 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/quota_request.go` 与迁移，落地专用 `enterprise_quota_requests`（或等价命名）表，至少保存：`tenant_id`、`requester_id`、`target_department_id`、`target_budget_id`、`target_budget_mode`、`requested_quota`、`approved_quota`、`status`、`request_reason`、`approval_comment`、`approver_id`、`allocation_id`、`idempotency_key`、`owner_count_snapshot`、`fallback`、`submitted_at`、`approved_at`、`rejected_at`、`fulfilled_at`、`processed_at`、`created_at`、`updated_at`。[Source: _bmad-output/planning-artifacts/architecture.md#额度申请审批流（V1.3）; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.5 员工申请与单步审批]
  - [x] 状态只覆盖本故事真正需要的单步流，例如 `pending -> approved -> fulfilled`、`pending -> rejected` 与少量失败/重试语义；不要提前实现多人会签、转签、代理审批、升级审批等 V1.3 明确不做的状态。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-25: 支持员工额度申请与单步审批]
  - [x] 以三库兼容的 GORM 迁移建立幂等唯一约束，例如 `(tenant_id, requester_id, idempotency_key)` 或等价键，保证重复提交返回同一 request 事实而不是新建第二条申请；不要依赖 PostgreSQL 专属 UPSERT/JSONB 方案。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [x] `enterprise_quota_allocations` 继续是“已实际分配额度”的唯一审计源，`enterprise_quota_requests` 只承载“申请/审批/履约”事实；不要把申请记录混写到 delegation ledger、admin action payload 或 `subscriptions` 扩展列里充当主存储。[Source: _bmad-output/planning-artifacts/architecture.md#额度申请审批流（V1.3）; service/enterprise/quota_allocation.go; service/enterprise/admin_action.go]

- [x] 在服务层实现提交、审批、拒绝和收件箱查询，并把审批路由绑定到既有负责人解析结果 (AC: 1, 2, 3)
  - [ ] 新增 `service/enterprise/quota_request.go`，提供稳定入口，例如 `Submit`, `Approve`, `Reject`, `ListMine`, `ListInbox` 或等价命名；不要把业务逻辑散落到 controller 或前端 mutation 中。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
  - [ ] 提交申请时必须复用现有成员关系能力校验“当前登录用户属于目标部门”，并显式校验目标预算池属于目标部门且处于可分配状态；员工不能替别的用户发起申请，也不能给不属于自己的部门发起申请。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#UJ-6. 员工额度不足时发起申请并由部门负责人审批。; service/enterprise/quota_allocation.go]
  - [ ] 审批路由必须复用 `PermissionService.ResolveEffectiveDepartmentOwners` 的统一解析结果；当 `owner_count=0` 时记录明确的 `fallback=admin` 路由快照，供管理员处理，而不是静默让任何普通用户可审批。[Source: service/enterprise/permission.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
  - [ ] 单步审批允许“通过较小额度”或“拒绝”，以覆盖预算不足时的简化处理；但不要扩展多人会签、二次确认、代理审批、升级审批链或 SLA 计时器。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.5 员工申请与单步审批]
  - [ ] 审批通过后的自动分配必须直接复用 `QuotaAllocationService`，不得复制预算预留、wallet 创建、SQLite 重试或 budget invariant 逻辑。若需要把 request 状态更新与 allocation 创建放进同一事务，应优先把 `QuotaAllocationService.Create` 内部提炼为 tx-aware helper，而不是在 quota request service 再写一份“伪 Create”逻辑。[Source: service/enterprise/quota_allocation.go; _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
  - [ ] 重复审批、重复回调或重试不得创建重复 allocation。若 request 已绑定 `allocation_id` 或已处于终态，后续 approve/retry 应返回现有结果并保持幂等。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-25: 支持员工额度申请与单步审批; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]

- [x] 暴露 quota request API，并按员工提交与负责人审批的不同权限边界设计路由 (AC: 1, 2, 3)
  - [ ] 在 `controller/enterprise/quota_request.go` 与 `router/enterprise-router.go` 增加 `/api/enterprise/quota-requests` 相关接口，至少覆盖：员工提交申请、员工查看自己的申请、负责人/管理员查看待处理 inbox、审批通过、审批拒绝、读取单条详情。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; router/enterprise-router.go]
  - [ ] 员工提交接口不能直接复用 `EnterpriseDepartmentAdmin` 中间件，因为当前 story 的发起者是普通员工；应使用 `UserAuth` + service 内部的成员归属校验。负责人审批接口也不要只靠前端隐藏按钮，服务层必须再次校验审批者是目标部门有效负责人或 Admin fallback 处理者。[Source: middleware/enterprise_dept_admin.go; _bmad-output/planning-artifacts/architecture.md#企业 API（新增）]
  - [ ] 为员工 UI 提供“我可申请的部门/预算池”窄读接口，或新增仅返回当前用户所属部门及可申请预算池的查询；不要简单放宽现有 `/departments/:id/budgets`、`/quota-allocations` 等 owner/admin 接口给所有员工。[Source: router/enterprise-router.go; web/default/src/features/enterprise-organization/api.ts]
  - [ ] DTO 继续使用 snake_case；可选标量使用指针 + `omitempty`；响应保持 `{ success, message, data }` 和空数组 `[]` 约定，错误继续走项目既有 message key 风格而不是暴露 GORM/SQL 文本。[Source: AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values; controller/enterprise/quota_allocation.go; controller/enterprise/budget_delegation.go]
  - [ ] 错误语义至少区分：用户不在目标部门、目标预算池不存在/不可分配、无有效负责人但已回退 Admin、审批者无权限、请求已处理、批准额度非法、自动分配预算不足、幂等重放返回既有结果等场景，便于前端做准确提示。[Source: service/enterprise/errors.go; i18n/keys.go]

- [x] 在 Default 企业组织页补齐“员工申请入口 + 负责人审批收件箱”，但保持现有上下文驱动工作台结构 (AC: 1, 2, 3)
  - [ ] 继续基于 `web/default/src/features/enterprise-organization/` 现有上下文驱动页面实现，而不是新做第二个企业治理页；员工申请入口可以与当前“当前部门/预算/成员”工作区并存，但只暴露当前登录用户可用的部门和预算池，不要求手输 `department_id`/`budget_id`。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md; web/default/src/features/enterprise-organization/index.tsx]
  - [ ] 员工发起申请时必须显式选择目标部门与目标池子/模式；若用户属于多个部门，UI 不能默认替用户猜部门；若某部门无可申请预算池，要显示明确空态，而不是把所有 owner/admin 明细都暴露出来。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.5: 支持员工额度申请、单步审批与自动分配]
  - [ ] 负责人/管理员在审批收件箱中需要看到申请人、目标部门、目标预算池、申请额度、可批准额度输入和审批结果；通过时允许批准较小额度，拒绝时要求或至少允许填写原因。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-25: 支持员工额度申请与单步审批]
  - [ ] 审批成功后精确失效/刷新当前部门预算、allocation 列表、钱包明细和 request inbox/mine queryKey；继续遵守 `['enterprise', 'organization', ...]` 命名空间，不要全量 invalidate `['enterprise']`。[Source: web/default/src/features/enterprise-organization/api.ts; _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
  - [ ] 所有新增前端文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`；Classic 仅做必要可见性/兼容，不把本故事扩张成 Classic 审批台重写。[Source: AGENTS.md#Internationalization (i18n); _bmad-output/planning-artifacts/architecture.md#Important Decisions]

- [x] 为 Story 7.6 的统一审计与钉钉通知留出连续性，但不要提前把 7.5 扩 scope 成完整通知系统 (AC: 2, 3)
  - [ ] request 记录中要保留足够的生命周期事实和追溯字段，使 Story 7.6 可以无损投影“申请提交 -> 审批通过/拒绝 -> 自动分配结果”；不要等 7.6 再回填基础状态。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26: 建立治理动作审计与钉钉通知闭环]
  - [ ] 如果本故事需要补最小通知 hook/status，也必须保持 best-effort 和非阻塞，不让通知失败回滚核心申请/审批/自动分配事务；真正的钉钉发送矩阵、时间线查询和交付状态管理属于 Story 7.6。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.6 审计与钉钉通知; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
  - [ ] 低频治理审计可参考 Story 7.3 的 `enterprise_admin_actions` 写法补充必要 action type，但不能把 admin action 当成 quota request 的唯一真源；request 表仍应是审批流主事实表。[Source: service/enterprise/admin_action.go; _bmad-output/implementation-artifacts/7-3-delegate-ancestor-budget-pools-to-descendant-department-pools.md]

- [x] 补齐测试，锁住员工申请、单步审批、自动分配复用和幂等语义 (AC: 1, 2, 3)
  - [ ] Model/migration tests：覆盖 `enterprise_quota_requests` 新表结构、幂等唯一约束、状态默认值和三库兼容迁移路径；至少避免使用 MySQL/PostgreSQL 专属 DDL 才能通过。[Source: model/enterprise/migration.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [ ] Service tests：覆盖员工只能为自己且仅对所属部门提交申请、多部门用户必须显式选部门、无有效负责人时记录 `fallback=admin`、负责人可批准较小额度、审批拒绝、审批通过自动创建 allocation、重复 approve 不重复创建 allocation、重复 submit 命中 idempotency key 返回同一 request、预算不足时可拒绝或小额批准。[Source: service/enterprise/permission.go; service/enterprise/quota_allocation_test.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
  - [ ] Controller/router tests：覆盖员工提交与 owner/admin 审批的权限边界、message key、错误 reason、空数组返回、已处理 request 的幂等返回，以及 employee 窄读接口不会泄露无权部门预算信息。[Source: controller/enterprise/quota_allocation_test.go; controller/enterprise/department_role_test.go; router/enterprise-router.go]
  - [ ] Frontend tests：覆盖员工申请表部门/预算池显式选择、无可申请预算池空态、负责人审批收件箱、批准较小额度、审批成功后 allocation/预算 UI 更新，以及 queryKey 精确失效。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/src/features/enterprise-organization/api.ts]
  - [ ] Boundary tests：继续确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`，且不把企业申请流带入 `/v1/**` relay 或 billing 主链路。[Source: _bmad-output/planning-artifacts/architecture.md#Anti-Patterns]

## Dev Notes

- 当前仓库尚无 `quota_request` 相关 model/service/controller/dto 实现；本故事应新增一条完整但最小的 request workflow 纵切片，而不是把审批状态偷偷塞进 `budget_delegation`、`quota_allocation` 或前端本地状态里。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- `PermissionService.ResolveEffectiveDepartmentOwners` 已经给出 `effective_owners`、`owner_count` 和 `fallback`，这是 7.5 审批路由的统一真源。不要在 controller 或前端再重写一套“谁可以批”的判断。[Source: service/enterprise/permission.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
- `EnterpriseDepartmentAdmin` 适合负责人/管理员治理入口，不适合员工提交申请。若直接把 submit API 挂在该中间件后面，员工会天然被挡在门外，违背 story 角色设定。[Source: middleware/enterprise_dept_admin.go]
- `QuotaAllocationService.Create` 目前自己管理预算锁定、wallet 创建和事务；7.5 最大技术风险是开发者复制这一段逻辑来“顺手实现自动分配”。正确方向是提炼 tx-aware helper 或等价复用点，让 request 状态更新和 allocation 创建可以原子完成，但预算铁律仍只有一套实现。[Source: service/enterprise/quota_allocation.go]
- 现有 `enterprise-organization` 页面已经有 owner、budget delegation 和 quota allocation 工作区，且 Story 6.3 已把它收敛为当前部门上下文驱动。7.5 应在这个 feature 上扩展“员工申请”和“审批 inbox”，不要新建平行治理页面，但员工可见的数据面必须比负责人更窄。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md; web/default/src/features/enterprise-organization/index.tsx]
- Story 7.4 已经把 allocation 的 supersede/cancel/reclaim 状态语义补齐；7.5 自动分配复用的就是这条既有 allocation 链路，而不是新建“request allocation”第四种 lifecycle。[Source: _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
- Story 7.6 才是统一审计时间线和钉钉通知闭环。7.5 需要把 request/approval/fulfillment 的基础事实存全、为后续通知留 hook，但不应在本故事里扩成完整消息投递编排系统。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.6: 建立治理动作审计时间线与钉钉通知]
- 最近 Story 7.3 和 7.4 的实现模式已经统一了企业治理 mutation 的习惯：服务层收敛业务铁律、controller 返回 message key 与 reason、低频治理动作可写 `enterprise_admin_actions`、UI 在当前部门工作区内完成。这些模式应直接复用到 7.5。[Source: git log -5; controller/enterprise/budget_delegation.go; controller/enterprise/quota_allocation.go]
- 所有 JSON 继续使用 `common.Marshal` / `common.Unmarshal*`，所有表结构和迁移保持 SQLite/MySQL/PostgreSQL 兼容，不引入 JSONB、专属 UPSERT 或依赖数据库特性的审批状态实现。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]

### Project Structure Notes

- 重点预计新增或修改：
  - `model/enterprise/quota_request.go`
  - `model/enterprise/migration.go`
  - `model/enterprise/quota_request_test.go`
  - `service/enterprise/quota_request.go`
  - `service/enterprise/quota_request_test.go`
  - `service/enterprise/errors.go`
  - `service/enterprise/quota_allocation.go`
  - `controller/enterprise/quota_request.go`
  - `controller/enterprise/quota_request_test.go`
  - `dto/enterprise/quota_request.go`
  - `router/enterprise-router.go`
  - `i18n/keys.go`
  - `i18n/locales/{en,zh-CN,zh-TW}.yaml`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
- 应保持不改或仅只读依赖：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - `relay/channel/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.5: 支持员工额度申请、单步审批与自动分配]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#UJ-6. 员工额度不足时发起申请并由部门负责人审批。]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-25: 支持员工额度申请与单步审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26: 建立治理动作审计与钉钉通知闭环]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.5 员工申请与单步审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.6 审计与钉钉通知]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
- [Source: _bmad-output/planning-artifacts/architecture.md#额度申请审批流（V1.3）]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
- [Source: _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
- [Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
- [Source: service/enterprise/quota_allocation.go]
- [Source: service/enterprise/permission.go]
- [Source: service/enterprise/errors.go]
- [Source: service/enterprise/admin_action.go]
- [Source: controller/enterprise/quota_allocation.go]
- [Source: controller/enterprise/budget_delegation.go]
- [Source: middleware/enterprise_dept_admin.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-organization/api.ts]
- [Source: web/default/src/features/enterprise-organization/types.ts]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 09:28:35 +0800: 已按用户要求读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，并执行手工 fallback，因为 `_bmad/scripts/resolve_customization.py` 受本地 `python3` 版本限制无法运行 `tomllib`。
- 2026-06-02 09:28:35 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`。
- 2026-06-02 09:28:35 +0800: 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation`，当前状态已是 `ready-for-dev`。
- 已执行 discover-inputs：加载 `_bmad-output/planning-artifacts/epics.md`、`architecture.md`、`prds/prd-new-api-2026-05-27/prd.md`、`addendum.md`；未发现仓库内 `project-context.md`。
- 已读取前序故事与相关代码：`7-1`、`7-4`、`6-3`、`service/enterprise/permission.go`、`quota_allocation.go`、`budget_delegation.go`、`controller/enterprise/*`、`router/enterprise-router.go`、`web/default/src/features/enterprise-organization/*`。
- 已确认当前仓库还不存在 `quota_request` 相关实现，本故事需要新增完整 request workflow 纵切片，并严格复用既有 owner 解析与 allocation 服务。
- 2026-06-02 09:48:30 +0800: 已完成 `enterprise_quota_requests` 模型、迁移、service、controller、DTO、路由、i18n、Admin fallback 能力口与 Default 前端申请/审批入口；审批通过复用 `QuotaAllocationService` 的 tx-aware helper，重复审批保持 allocation 幂等。
- 2026-06-02 09:48:30 +0800: 已运行 `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./service/enterprise ./controller/enterprise -run 'QuotaRequest'` 通过，并运行 `bun run typecheck`（目录 `web/default/`）通过。
- 2026-06-02 10:11:47 +0800: 执行 story-automator review workflow，自动修复 request 主事实表字段缺口、inactive budget 提交校验、capability 路由顺序、DTO/前端类型同步和新增回归测试。

### Completion Notes List

- 已完成员工额度申请纵切片：专用 request 真源、单步审批、负责人/管理员审批边界、窄读 capability 与自动分配复用。
- 已完成 Default 前端同页入口：显式选择目标预算池提交申请、在部门工作区中查看 request 状态并执行较小额度审批或拒绝。
- 已完成 7.5 定向后端测试与前端类型检查；未在本次执行里跑完整前端测试矩阵和全仓 `service/enterprise` 非 7.5 噪声测试。
- Senior review 已修复 4 个确认问题：request 表缺少 owner/fallback 与生命周期快照字段；提交申请未提前拒绝 inactive budget；capability 窄读路由被 `:id` 详情路由遮挡风险；前端/DTO 未暴露新增 request 审计事实字段。

### File List

- `controller/enterprise/department.go`
- `controller/enterprise/department_membership_test.go`
- `controller/enterprise/quota_request.go`
- `controller/enterprise/quota_request_test.go`
- `dto/enterprise/quota_request.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `i18n/locales/zh-TW.yaml`
- `model/enterprise/migration.go`
- `model/enterprise/quota_request.go`
- `model/enterprise/quota_request_test.go`
- `router/enterprise-router.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/errors.go`
- `service/enterprise/quota_allocation.go`
- `service/enterprise/quota_allocation_test.go`
- `service/enterprise/quota_request.go`
- `service/enterprise/quota_request_test.go`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`
- `_bmad-output/implementation-artifacts/7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

### Change Log

- 2026-06-02 09:28:35 +0800: 依据 create-story workflow 重新整理 Story 7.5，上下文补强为可开发版本。
- 2026-06-02 09:48:30 +0800: 完成 7.5 后端与 Default 前端实现，补齐定向测试与类型检查，并将故事推进到 review。
- 2026-06-02 10:11:47 +0800: Senior Developer Review 自动修复已完成，补齐 request 审计字段、路由与校验回归测试，并将故事推进到 done。

### Senior Developer Review (AI)

#### Review Summary

- Outcome: Approve
- Story Status: done
- Git vs Story File List: 发现新增 `model/enterprise/quota_request_test.go` 未在 File List 中记录，已补齐。

#### Findings Fixed

- HIGH: `enterprise_quota_requests` 没有保存 `owner_count_snapshot`、`fallback`、`submitted_at`、`approved_at`、`rejected_at`、`processed_at`，不满足 AC1/AC3 对审批路由快照和生命周期事实的要求。已补齐模型、迁移、DTO、service 映射与前端类型。
- HIGH: 员工提交申请只校验预算池存在，未拒绝 paused/revoked/expired 预算池。已在 service submit 校验中返回 inactive budget 错误，并添加回归测试。
- MEDIUM: `/quota-requests/:id` 注册在 `/quota-requests/capability/:department_id` 前时存在 capability 窄读接口被详情路由吞掉的风险。已调整路由顺序并添加 controller 测试。
- MEDIUM: request 响应和前端类型未暴露审批路由/生命周期快照字段，Story 7.6 无法无损投影申请 -> 审批 -> 自动分配事实。已同步 DTO、controller map 与 Default 类型/fixture。

#### Validation

- `GOCACHE=$(pwd)/.tmp/go-build-cache go test ./model/enterprise ./service/enterprise ./controller/enterprise -run 'QuotaRequest'`
- `bun run typecheck`（目录：`web/default/`）

#### Remaining Risk

- 未运行完整前端 `bun run test:e2e`；本轮验证覆盖了新增类型面和 7.5 后端定向行为。
