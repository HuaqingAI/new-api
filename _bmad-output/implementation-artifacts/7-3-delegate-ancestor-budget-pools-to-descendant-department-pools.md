---
baseline_commit: ab9550a95e83569c20ea818aa2b3d8777c1ea8c2
---

# Story 7.3: 支持上级预算池向后代部门预算池分配

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 部门负责人,
I want 把本部门预算池中的额度继续分配给后代部门预算池,
so that 下级部门可以在自己的治理范围内自主继续分配给成员。

## Acceptance Criteria

1. **Given** 某负责人管理祖先部门且该部门存在可用预算池  
   **When** 其选择某个后代部门作为目标  
   **Then** 系统允许从来源预算池向目标部门预算池分配额度  
   **And** 不要求目标部门必须是直属下级。
2. **Given** 目标部门不是来源部门的后代节点  
   **When** 负责人尝试发起预算分配  
   **Then** 系统拒绝操作  
   **And** 返回可测试的树关系错误。
3. **Given** 已存在一条部门到部门的预算委派记录  
   **When** 管理员调整该委派额度  
   **Then** 系统优先关闭旧委派并创建新委派  
   **And** 不静默覆盖历史分配事实。

## Tasks / Subtasks

- [x] 新增部门预算委派数据模型与迁移，建立独立 delegation ledger (AC: 1, 2, 3)
  - [x] 按架构映射新增 `model/enterprise/budget_delegation.go` 或等价命名文件，表名保持 `enterprise_` 前缀，承载至少：`tenant_id`、`source_department_id`、`source_budget_id`、`target_department_id`、`target_budget_id`、`actor_id`、`committed_quota`、`budget_type_snapshot`、`cycle_type_snapshot`、`status`、`before_source_budget_snapshot`、`after_source_budget_snapshot`、`before_target_budget_snapshot`、`after_target_budget_snapshot`、`superseded_by_id`、`processed_at`、`reason`、时间戳。[Source: _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-23: 支持上级预算池向下级部门预算池分配]
  - [x] 在 `model/enterprise/migration.go` 注册新模型迁移，使用 GORM/Migrator 三库兼容方式，不写 recursive CTE、JSONB 操作符或 DB 专属 DDL；如需额外索引，走现有 migration helper 风格。[Source: model/enterprise/migration.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [x] status 语义需与后续 Story 7.4/7.5 可衔接，至少明确 `active / superseded / revoked / expired / closed` 或等价状态；“调整委派”不能原地改行，必须通过旧记录关闭 + 新记录创建表达。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.3 上级预算池到下级预算池的治理语义]

- [x] 实现预算委派服务，严格区分“部门预算池 -> 部门预算池”与“部门预算池 -> 成员 wallet” (AC: 1, 2, 3)
  - [x] 新增 `service/enterprise/budget_delegation.go`，稳定入口建议至少包含：创建委派、调整委派（supersede）、查询委派列表/详情；不要把部门到部门委派逻辑塞进现有 `quota_allocation.go`，因为后者只负责成员 wallet 分配。[Source: _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）; service/enterprise/quota_allocation.go]
  - [x] 创建委派时必须先校验：来源预算池和目标预算池都存在、位于同一 `tenant_id`、类型仍只允许 `balance / subscription`、来源部门是目标部门的祖先、操作者对来源部门具治理权限。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-23: 支持上级预算池向下级部门预算池分配; service/enterprise/permission.go]
  - [x] 类型兼容规则要写清楚并固化在 service：同一条委派不能把 `balance` 池委派到 `subscription` 池或反之，除非业务明确允许且快照/剩余额度语义一致；若当前架构未授权跨类型转换，应拒绝并返回可测试错误，避免隐式引入第三类预算策略。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-23: 支持上级预算池向下级部门预算池分配; model/enterprise/department_budget.go]
  - [x] 委派只改变企业预算池状态，不直接创建 `subscriptions` 行，不调用 `model.CreateEnterpriseAllocationSubscriptionTx`，成员 wallet 仍由后续现有 `QuotaAllocationService.Create` 承接。[Source: _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）; service/enterprise/quota_allocation.go]

- [x] 复用并扩展现有预算预留/回收语义，保证父池和目标池额度一致性 (AC: 1, 3)
  - [x] 参考 `QuotaAllocationService.reserveBudgetQuota` 与 `buildBudgetReservationUpdate` 的并发和快照模式，为 delegation service 提供“来源池扣减 + 目标池增加可分配额度”的事务实现；优先提炼共享 helper，避免复制两套近似但不一致的额度原子更新逻辑。[Source: service/enterprise/quota_allocation.go]
  - [x] `balance` 型委派需要从来源池 `remaining` 原子扣减，并把目标池可用额度增加到可继续分配的状态；`subscription` 型委派需要确保 `Σ delegated + Σ member allocations <= cycle_quota` 或等价铁律仍成立，不能因为祖先委派后代而绕过现有 `allocated_total` 约束。[Source: AGENTS.md#Rules; service/enterprise/quota_allocation.go; _bmad-output/planning-artifacts/epics.md#Additional Requirements]
  - [x] 目标预算池当前模型只有 `TotalQuota / Remaining / AllocatedTotal / CycleQuota` 等字段，没有“上游注资”概念；实现前需明确是复用现有字段表达 delegated capacity，还是为 `DepartmentBudget` 增加派生字段（如 `delegated_total` / `source_budget_id` / `delegated_remaining`），并在 Dev Agent Record 说明为何该做法不会破坏 Story 3.x/7.2 现有预算 summary 语义。[Source: model/enterprise/department_budget.go; service/enterprise/department_budget.go; _bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md]
  - [x] 调整委派额度时必须先关闭旧 delegation，再用新额度重新创建 delegation，并同步回退/重算来源池与目标池余额；不能直接 `UPDATE committed_quota`，否则历史快照、审计链和并发校验都会失真。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.3 上级预算池到下级预算池的治理语义]

- [x] 实现祖先/后代关系与治理权限校验，拒绝跨树越权 (AC: 1, 2)
  - [x] 复用 Story 7.2 已落地的 `ResolveDepartmentScope` 或等价树 helper 判断祖先-后代关系，不引入新的闭包表或递归 SQL；若需要“源部门是目标部门祖先”的显式 helper，放在 `service/enterprise/department_scope_resolver.go` 或同包中，避免在 controller 里重复拼树逻辑。[Source: _bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md; service/enterprise/department_scope_resolver.go]
  - [x] 使用 Story 7.1 的 `CanGovernDepartment` / `ResolveEffectiveDepartmentOwners` 作为权限事实来源；祖先负责人可以治理后代，但目标部门上的本地 deny 仍必须阻断越权委派。[Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md; service/enterprise/permission.go]
  - [x] “目标部门不是后代”与“无权限治理该目标部门”必须返回可区分错误，供 controller、前端和测试断言，不要全部折叠成通用数据库错误。[Source: service/enterprise/errors.go; controller/enterprise/department_budget.go]

- [x] 暴露委派 API，并保持现有预算/成员分配入口边界清晰 (AC: 1, 2, 3)
  - [x] 在 `router/enterprise-router.go` 新增 `/api/enterprise/budget-delegations` 或等价 REST 路由，至少覆盖：创建委派、调整委派、查询当前部门相关委派；不要把部门到部门委派混进 `/quota-allocations`，因为该接口的参数和鉴权都假设目标是成员 wallet。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; router/enterprise-router.go]
  - [x] 新增 `controller/enterprise/budget_delegation.go` 与 `dto/enterprise/budget_delegation.go`。DTO 可选标量一律使用指针 + `omitempty`，请求/响应字段保持 snake_case，数组缺省返回 `[]`。[Source: AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values]
  - [x] 控制台 API 继续使用 `{ success, message, data }`，业务错误按现有 enterprise i18n message key 输出 HTTP 200；树关系错误、预算不足、类型不匹配、无权限需可被前端单独翻译展示。[Source: controller/enterprise/department_budget.go; _bmad-output/planning-artifacts/architecture.md#Response Format]

- [x] 把委派可视化接入 Default 企业组织页预算工作区 (AC: 1, 2, 3)
  - [x] 复用 `web/default/src/features/enterprise-organization/` 现有“左树 + 右侧当前部门工作区”骨架，在预算工作区内增加“委派到后代部门”入口与委派记录展示；不要新做独立页面或要求用户手填 Department ID 作为主路径。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#UJ-5. 上级部门负责人向下级部门委派预算。]
  - [x] UI 需要把目标部门选择限制在当前部门后代范围内，并明确显示“当前部门 -> 目标后代部门 -> 目标预算池”的链路；若 7.2 已提供包含子部门预算列表，可复用其 `department_name` / scope 文案帮助用户辨认预算池归属。[Source: _bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md; web/default/src/features/enterprise-organization/types.ts]
  - [x] 调整委派时，前端交互必须呈现“旧委派将关闭并新建新委派”的语义，避免用户误以为是在编辑同一条记录；历史记录列表应保留 superseded/closed 状态可见性。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.3: 支持上级预算池向后代部门预算池分配]
  - [x] 新增前端文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，queryKey 继续挂在 `['enterprise', 'organization', ...]` 命名空间下，mutation invalidate 精确到当前部门预算工作区和 delegation 列表，不误伤无关 summary query。[Source: AGENTS.md#Internationalization (i18n); web/default/src/features/enterprise-organization/api.ts; web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]

- [x] 记录低频治理审计，并为 Story 7.6 钉钉通知留好事实基础 (AC: 3)
  - [x] 在 `service/enterprise/admin_action.go` 增加预算委派相关 action type，例如 create / supersede / revoke / reject；payload 至少包含 source/target department、source/target budget、committed quota、budget type snapshot、delegation status、actor、错误原因或 supersede 关系。[Source: service/enterprise/admin_action.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26]
  - [x] 当前 Story 7.3 不需要真正发送钉钉通知，但审计数据结构要足够支撑 Story 7.6 后续通知闭环；不要把通知 side effect 写进本故事核心事务里。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.6 审计与钉钉通知]

- [x] 补齐后端、前端与边界测试，锁住委派链路的正确性 (AC: 1, 2, 3)
  - [x] Model/migration tests：覆盖 delegation ledger 迁移、必要索引、三库兼容路径，以及 `DepartmentBudget` 如新增字段时的向后兼容默认值。[Source: model/enterprise/migration.go; model/enterprise/text_default_test.go]
  - [x] Service tests：覆盖祖先 -> 直属子部门、祖先 -> 孙部门、非后代拒绝、manual deny 阻断、balance/subscription 两类预算池、预算不足、调整委派走 supersede、并发下不突破铁律。[Source: service/enterprise/*_test.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
  - [x] Controller/router tests：覆盖新 API 参数绑定、错误 message、权限校验、创建委派成功返回快照字段、调整委派不会原地覆盖历史记录。[Source: controller/enterprise/*_test.go; router/enterprise-router.go]
  - [x] Frontend tests：覆盖组织页预算工作区的后代部门选择、目标预算池展示、调整委派提示、queryKey/invalidate 命名空间保持在 `enterprise-organization` 下。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] Boundary tests：继续确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`。[Source: service/enterprise/boundary_test.go; _bmad-output/planning-artifacts/architecture.md#Anti-Patterns]

## Dev Notes

- Story 7.3 只解决 FR23 的“部门预算池 -> 后代部门预算池”治理动作，不实现成员 allocation 调整/取消/回收，也不实现员工额度申请审批。这两块分别属于 Story 7.4 和 7.5，当前不要顺手扩 scope。[Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- 现有预算核心只有两条路径：`DepartmentBudgetService` 管部门预算池的创建/列表/详情，`QuotaAllocationService` 管“部门预算池 -> 成员 wallet”。仓库内尚无 delegation ledger、controller、DTO 或前端委派 UI；本故事必须补完整链路，而不是在 `quota_allocation.go` 上打补丁模拟部门委派。[Source: service/enterprise/department_budget.go; service/enterprise/quota_allocation.go; router/enterprise-router.go]
- `QuotaAllocationService` 已经提供了预算预留、SQLite 锁重试、快照记录和回收逻辑，这些并发与一致性模式应尽量复用到 delegation service。否则 Story 7.3 很容易做出一套和成员分配行为不一致的预算原子更新语义。[Source: service/enterprise/quota_allocation.go]
- 当前 `DepartmentBudget` 模型并未表达预算池之间的上游/下游关系，只有 `ParentStatus` 这类钱包状态联动字段，不能误用它来存 delegation 事实。委派历史需要独立 ledger，预算池是否需要新增字段应基于“能否表达 delegated capacity 且不破坏 Story 3.x/7.2 现有 API”来决定。[Source: model/enterprise/department_budget.go; service/enterprise/department_budget.go]
- Story 7.2 已落地 descendant scope resolver 和预算列表中的 `department_name` / `include_descendants` 能力。7.3 应复用这套树解析和预算归属展示，而不是再实现一份不同的后代部门判定与前端下拉筛选逻辑。[Source: _bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md]
- Story 7.1 已把部门负责人解析收敛到 `PermissionService.ResolveEffectiveDepartmentOwners` / `CanGovernDepartment`。7.3 的权限校验必须继续使用这套结果，特别要注意目标部门上的 local deny 仍然可以阻断祖先负责人的跨层治理权限。[Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md; service/enterprise/permission.go]
- 预算委派不改变 relay、billing expression、`/v1/**` 协议面，也不直接写成员 wallet。后续成员消费仍由现有 wallet 扣费链路承接，这是这个故事最重要的边界之一。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#4.6 分层预算治理与额度审批; _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）]
- 三库兼容和 JSON wrapper 规则仍然是硬约束。如果 delegation 快照或 payload 需要 JSON，必须用 `common.Marshal` / `common.Unmarshal*`，不要在业务代码里直接调 `encoding/json`。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go]

### Project Structure Notes

- 重点预计修改：
  - `model/enterprise/budget_delegation.go`（新增）
  - `model/enterprise/migration.go`
  - `service/enterprise/budget_delegation.go`（新增）
  - `service/enterprise/errors.go`
  - `service/enterprise/admin_action.go`
  - `service/enterprise/department_scope_resolver.go` 或等价 helper
  - `service/enterprise/department_budget.go`
  - `service/enterprise/quota_allocation.go`（仅在抽取共享预算预留 helper 时小心演进，不改变“成员 wallet 分配”职责边界）
  - `controller/enterprise/budget_delegation.go`（新增）
  - `dto/enterprise/budget_delegation.go`（新增）
  - `router/enterprise-router.go`
  - `i18n/keys.go`
  - `i18n/locales/{en,zh-CN,zh-TW}.yaml`
  - `docs/openapi/api.json`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
- 保持不改：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - `relay/channel/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.3: 支持上级预算池向后代部门预算池分配]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#UJ-5. 上级部门负责人向下级部门委派预算。]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-23: 支持上级预算池向下级部门预算池分配]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.3 上级预算池到下级预算池的治理语义]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
- [Source: _bmad-output/planning-artifacts/architecture.md#预算委派流（V1.3）]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
- [Source: _bmad-output/implementation-artifacts/7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md]
- [Source: model/enterprise/department_budget.go]
- [Source: model/enterprise/migration.go]
- [Source: service/enterprise/department_budget.go]
- [Source: service/enterprise/quota_allocation.go]
- [Source: service/enterprise/permission.go]
- [Source: service/enterprise/admin_action.go]
- [Source: service/enterprise/errors.go]
- [Source: controller/enterprise/department_budget.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-organization/api.ts]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-organization/types.ts]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-01 22:35:00 +0800: 按用户要求启动 create-story，已先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-01 22:35:00 +0800: `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 Python 3.11+ 的 `tomllib` 失败；已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml`，确认无 team/user override，persistent facts 仅为 `file:{project-root}/**/project-context.md`，仓库内未发现匹配文件。
- 2026-06-01 22:35:00 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`。
- 2026-06-01 22:35:00 +0800: 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `7-3-delegate-ancestor-budget-pools-to-descendant-department-pools`，原状态为 `backlog`，Epic 7 已为 `in-progress`。
- 2026-06-01 22:35:00 +0800: 已执行 discover-inputs：加载 `epics.md`、`architecture.md`、`prds/prd-new-api-2026-05-27/prd.md`、`prds/prd-new-api-2026-05-27/addendum.md`，重点读取 FR23、UJ-5、V1.3 addendum 8.3、预算委派流和架构映射。
- 2026-06-01 22:35:00 +0800: 已读取前序故事 `_bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md`、`7-2-add-descendant-inclusive-scope-to-budget-usage-and-alert-summaries.md`，确认 7.3 必须复用 owner 权限解析与 descendant scope resolver。
- 2026-06-01 22:35:00 +0800: 已读取现有实现：`model/enterprise/department_budget.go`、`model/enterprise/migration.go`、`service/enterprise/{department_budget,quota_allocation,permission,admin_action,errors}.go`、`controller/enterprise/department_budget.go`、`router/enterprise-router.go`、`web/default/src/features/enterprise-organization/{api,types,index}.tsx` 以及相关测试文件。
- 2026-06-01 22:35:00 +0800: 已确认仓库当前不存在 budget delegation ledger / service / controller / DTO / 前端入口；现有预算路径只有“部门预算池”与“成员 wallet 分配”，因此 Story 7.3 需要补完整链路而非修改现有 allocation 语义。
- 2026-06-01 23:53:14 +0800: 启动 dev-story 实施；由于当前 `python3` 不满足 3.11+，已按 skill fallback 手工解析 `.agents/skills/bmad-dev-story/customize.toml`，确认无 team/user override，persistent facts 仅为 `file:{project-root}/**/project-context.md`，仓库内未发现匹配文件。
- 2026-06-01 23:53:14 +0800: 已完整读取 story、checklist、`_bmad/bmm/config.yaml` 与 `sprint-status.yaml`，确认 `communication_language=Chinese`、`document_output_language=Chinese`、story key 为 `7-3-delegate-ancestor-budget-pools-to-descendant-department-pools`，并开始按任务顺序实施。
- 2026-06-02 00:48:30 +0800: 已补齐 `service/enterprise/budget_delegation_test.go` 的直属子部门与 subscription 并发铁律覆盖，并执行 story 相关 Go/Bun 验证；同时记录更大范围 `./service/enterprise` 全包回归中存在与 7.3 无关的既有失败与本机端口绑定限制。

### Implementation Plan

- 先补后端 delegation ledger、迁移与共享预算预留 helper，用红绿测试锁定祖先/后代校验、预算不变量、supersede 审计链。
- 再补 controller/DTO/router/i18n/openapi，确保错误语义可区分，且 `{ success, message, data }` 响应格式不偏离现有 enterprise 约定。
- 最后在 Default 组织页预算工作区接 delegation 表单与历史列表，补 queryKey、invalidate、6 语种文案与前端测试，然后跑 Go/Bun 测试并回填勾选。

### Completion Notes List

- 已创建 Story 7.3 故事文档，状态为 `ready-for-dev`。
- 已将 FR23 明确拆成独立 delegation ledger、service、API、Default 组织页预算工作区、审计和测试任务，避免开发阶段把“预算池 -> 预算池”误做成“预算池 -> 成员 wallet”。
- 已把最容易出错的当前代码事实写入 Dev Notes：现有 `DepartmentBudget` 不表达上下游关系、`QuotaAllocationService` 只负责成员 wallet、7.1 的 local deny 会影响跨层治理、7.2 的 descendant resolver 应被复用。
- 已按 checklist 自检并补强：接受标准、任务分解、架构边界、现有文件位置、回归风险、测试落点和来源引用均已覆盖。
- 已完成首版预算委派实现：独立 delegation ledger、共享预算原子更新 helper、预算委派 service / controller / router、后端 i18n 错误语义和企业组织页预算工作区入口。
- 目标预算池未新增派生字段；当前实现复用 `total_quota / remaining / cycle_quota / allocated_total` 表达 delegated capacity。对 balance 池增加 `total_quota + remaining`，对 subscription 池增加 `cycle_quota + remaining`，因此不会破坏 Story 3.x/7.2 现有 usage ratio 与 summary 计算语义。
- 已补齐 service 覆盖缺口：新增“祖先 -> 直属子部门”与“subscription 并发下不突破 cycle_quota/allocated_total 铁律”测试，连同既有“祖先 -> 孙部门、非后代拒绝、manual deny、预算不足、类型不匹配、supersede”一起锁住委派 service 行为。
- 已通过 story 相关验证：`go test ./service/enterprise -run 'Test(.*BudgetDelegation.*|.*Boundary.*)' -count=1`、`go test ./controller/enterprise -run 'Test.*BudgetDelegation.*' -count=1`、`go test ./model/enterprise -run 'Test.*BudgetDelegation.*|Test.*TextDefault.*' -count=1`、`bun test web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`、`bun run build:check`。
- 已额外执行 `go test ./model/enterprise ./controller/enterprise ./service/enterprise -count=1` 作为更大范围回归；其中 `model/enterprise` 与 `controller/enterprise` 通过，但 `service/enterprise` 仍存在与 Story 7.3 无关的既有失败（`usage_report_task` 断言、`dingtalk_client_test` 端口绑定受当前环境限制），因此该结果仅作为仓库现状记录，不阻断 7.3 本故事交付。

### File List

- `_bmad-output/implementation-artifacts/7-3-delegate-ancestor-budget-pools-to-descendant-department-pools.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/budget_delegation.go`
- `controller/enterprise/budget_delegation_test.go`
- `dto/enterprise/budget_delegation.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `i18n/locales/zh-TW.yaml`
- `model/enterprise/budget_delegation.go`
- `model/enterprise/budget_delegation_migration_test.go`
- `model/enterprise/migration.go`
- `model/enterprise/text_default_test.go`
- `router/enterprise-router.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/budget_capacity.go`
- `service/enterprise/budget_delegation.go`
- `service/enterprise/budget_delegation_test.go`
- `service/enterprise/department_scope_resolver.go`
- `service/enterprise/errors.go`
- `service/enterprise/quota_allocation.go`
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

## Change Log

- 2026-06-01 22:35:00 +0800: 创建 Story 7.3 上下文文档，准备进入开发。
- 2026-06-02 00:35:00 +0800: 完成预算委派 ledger / service / controller / router / i18n / Default 前端工作区首版实现，并补充 focused Go/Bun 测试；由于 OpenAPI 与部分 service 覆盖未完成，故事状态保持 `in-progress`。
- 2026-06-02 00:48:30 +0800: 补齐 Story 7.3 剩余 service 测试矩阵并完成 story 相关 Go/Bun 验证，故事状态更新为 `review`。
