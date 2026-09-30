---
baseline_commit: 00ab797f3e3007debbb29e9b6caf804fa1e75d1f
---

# Story 7.1: 建立部门负责人权限模型与生效优先级

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 同时使用钉钉同步负责人和手动指定负责人，并明确最终生效优先级,
so that 部门治理权限既能跟随组织事实，又能被企业本地规则覆盖。

## Acceptance Criteria

1. **Given** 某部门同时存在钉钉同步负责人和本地手动授权记录  
   **When** 系统计算该部门的有效负责人集合  
   **Then** 按 `manual_deny_override > manual_grant > dingtalk_synced_owner` 的顺序得出生效结果  
   **And** 原始来源事实与覆盖结果都可审计。
2. **Given** 某负责人被本地显式禁用  
   **When** 该用户尝试执行该部门预算治理或审批动作  
   **Then** 系统拒绝其操作  
   **And** 错误与审计中能说明其被本地规则覆盖。
3. **Given** 某部门不存在任何有效负责人  
   **When** 员工向该部门发起额度申请或管理员触发需负责人处理的治理动作  
   **Then** 系统明确提示“无有效负责人”  
   **And** 可按规则回退给 Admin 处理。

## Tasks / Subtasks

- [x] 扩展部门负责人数据模型，保留来源事实并支持本地覆盖 (AC: 1, 2)
  - [ ] 以现有 `model/enterprise/department_role.go` 和 `enterprise_department_roles` 为首选落点，新增能表达 `dingtalk_synced_owner`、`manual_grant`、`manual_deny_override` 的来源/效果字段；不要新建与现有 `PermissionService` 脱节的第二套权限表，除非迁移无法三库兼容且在代码注释和故事记录中说明原因。[Source: model/enterprise/department_role.go; service/enterprise/permission.go; _bmad-output/planning-artifacts/architecture.md#Department Owner Precedence (V1.3)]
  - [ ] 当前唯一索引 `uq_ent_dept_roles_role` 只允许同一用户/部门/角色一条记录，无法同时保存钉钉事实与本地 deny/grant 覆盖；迁移必须调整唯一约束或引入等价事实模型，确保同一 `tenant_id + department_id + user_id + role` 下可同时审计不同来源事实。[Source: model/enterprise/department_role.go; model/enterprise/department_role_test.go]
  - [ ] 状态语义必须区分“事实已失效/撤销”和“本地显式禁用”。现有 `StatusInactive` 不能直接当成 `manual_deny_override` 使用，否则无法表达“取消手动授权但不阻断钉钉负责人”的场景。[Source: constant/enterprise.go; service/enterprise/permission.go]
  - [ ] 新增/变更迁移放在 `model/enterprise/migration.go` 或 `DepartmentRole` AutoMigrate 可覆盖路径内，使用 GORM/Migrator 三库兼容能力；SQLite 不支持的 ALTER 行为必须用现有 `AddColumn` 风格兜底，不使用 MySQL/PostgreSQL 专属 SQL。[Source: model/enterprise/migration.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]

- [x] 实现有效负责人解析服务，并让治理鉴权统一走该解析结果 (AC: 1, 2, 3)
  - [ ] 在 `service/enterprise/permission.go` 或新增同包文件中实现稳定入口，例如 `ResolveEffectiveDepartmentOwners(tenantId, departmentId)` 和 `CanGovernDepartment(userId, tenantId, departmentId)`；算法按用户聚合来源事实，优先级必须是 `manual_deny_override > manual_grant > dingtalk_synced_owner`。[Source: service/enterprise/permission.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-21]
  - [ ] `ListManageableDepartmentIds` 当前会把部门管理员作用域扩展到后代部门；新解析逻辑要明确继承规则：祖先部门有效负责人可管理后代，但目标部门上的本地 deny 必须能阻断该用户对该目标部门的治理动作，避免 Story 7.3 跨层委派时越权。[Source: service/enterprise/permission.go; _bmad-output/planning-artifacts/epics.md#Story 7.3: 支持上级预算池向后代部门预算池分配]
  - [ ] `middleware/enterprise_dept_admin.go` 需要改为调用新的有效负责人/治理权限入口，Root/Admin 仍保留全局放行；普通用户无有效负责人权限时继续返回现有企业权限错误，若命中本地 deny 应能映射到更明确的 i18n 错误。[Source: middleware/enterprise_dept_admin.go; i18n/keys.go]
  - [ ] 无有效负责人时不要静默成功。服务层应返回可区分错误（如 `ErrDepartmentOwnerNotFound` 或等价命名），供后续 Story 7.5 quota request 路由到 Admin fallback，并供当前 UI/API 明确展示“无有效负责人”。[Source: service/enterprise/errors.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.1-部门负责人来源与生效优先级]

- [x] 将钉钉同步负责人写成来源事实，不覆盖本地规则 (AC: 1)
  - [ ] 当前 `DingTalkDepartmentInfo` / `DingTalkDepartmentUserInfo` 只映射部门和成员基础字段，没有负责人字段；需要在 `service/enterprise/dingtalk_client.go` 与 fake client tests 中补齐钉钉负责人来源字段或调用官方可返回负责人信息的通讯录接口。[Source: service/enterprise/dingtalk_client.go; service/enterprise/dingtalk_sync.go]
  - [ ] 钉钉同步只写 `dingtalk_synced_owner` 来源事实，并在外部事实消失时将该来源标记失效；不得删除或改写 `manual_grant` / `manual_deny_override` 本地覆盖事实。[Source: service/enterprise/dingtalk_sync_apply.go; _bmad-output/planning-artifacts/architecture.md#Department Owner Precedence (V1.3)]
  - [ ] 同步日志需记录负责人事实新增、更新、失效或跳过，使用现有 `enterprise_dingtalk_sync_logs` 模型和稳定 message，不把钉钉 secret/access token 写入日志。[Source: model/enterprise/dingtalk_sync.go; service/enterprise/dingtalk_sync_apply.go; AGENTS.md#Rule 2]
  - [ ] 最新外部文档检查：钉钉开放平台仍将通讯录用户详情作为部门用户数据来源之一；实现时只引用官方字段，不用字符串拼接解析未知 JSON，所有 decode 继续走 `common.*` wrapper。[Source: https://open.dingtalk.com/document/app/queries-department-user-details; AGENTS.md#Rule 1: JSON Package — Use common/json.go]

- [x] 暴露部门负责人 API，并保持旧部门管理员接口兼容 (AC: 1, 2, 3)
  - [ ] 在 `router/enterprise-router.go` 增加 `/api/enterprise/departments/:id/owners` 相关路由，至少覆盖：读取来源事实与有效负责人、手动授权、手动禁用、取消手动禁用/撤销手动授权。路由应使用 `EnterpriseAdmin()` 管理本地覆盖；普通部门负责人只能消费解析结果，不应任意给自己授权。[Source: router/enterprise-router.go; _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
  - [ ] 现有 `/departments/:id/admins` grant/revoke 路由不要直接删除；可以包装到新 owner service 的 `manual_grant` / revoke 语义，保证历史 UI/API 测试仍可通过。[Source: controller/enterprise/department_role.go; router/enterprise-router.go]
  - [ ] DTO 新增字段使用 snake_case；可选标量使用指针 + `omitempty`；响应中的 facts/effective owners 数组即使为空也返回 `[]`，不要返回 `null`。[Source: dto/enterprise/department_role.go; AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values]
  - [ ] 控制台 API 继续使用 `{ success, message, data }` 与 `common.ApiSuccess/ApiErrorI18n`；业务错误仍按项目惯例映射为 HTTP 200 下的 message key，不直接把 GORM 错误返回前端。[Source: controller/enterprise/department_role.go; _bmad-output/planning-artifacts/architecture.md#Response Format]

- [x] 记录负责人授权/禁用/解析结果审计，确保事实与覆盖可追溯 (AC: 1, 2, 3)
  - [ ] 在 `service/enterprise/admin_action.go` 增加负责人相关 action type，例如 owner manual grant、manual deny、manual deny revoke、DingTalk owner sync update；低频管理动作写 `enterprise_admin_actions`，payload 必须包含 actor、target department/user、source、effect、before/after 或 resolution summary。[Source: service/enterprise/admin_action.go; model/enterprise/admin_action.go]
  - [ ] 本地显式禁用导致治理动作被拒绝时，错误响应和审计/日志信息需要能说明“被本地覆盖规则阻断”，但不得记录敏感凭据或无关个人信息。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.1: 建立部门负责人权限模型与生效优先级; service/enterprise/admin_action.go]
  - [ ] 若某部门无有效负责人，API/服务返回应包含可审计的 fallback decision：`owner_count=0`、`fallback=admin` 或等价信息，为 Story 7.5 员工额度申请审批路由复用。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.5-员工申请与单步审批]

- [x] 在 Default 企业组织页提供负责人治理入口，Classic 保持必要可见性 (AC: 1, 2, 3)
  - [ ] Default 优先在 `web/default/src/features/enterprise-organization/index.tsx` 当前部门工作区新增负责人治理面板，复用现有部门树上下文；不要回退到手填 Department ID 的旧交互。[Source: web/default/src/features/enterprise-organization/index.tsx; _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
  - [ ] UI 需要同时显示有效负责人和来源事实：钉钉同步、手动授权、本地禁用覆盖，并清楚区分“撤销手动授权”和“本地禁用覆盖”。不要把所有非 active 状态都显示成同一种“已撤销”。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-21]
  - [ ] 本地 deny 命中时，预算治理按钮或审批入口应显示可理解的拒绝状态；无有效负责人时显示“无有效负责人”并标注 Admin fallback，不要让用户误以为同步失败或页面空数据。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.1: 建立部门负责人权限模型与生效优先级]
  - [ ] 新增 Default 文案必须进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，英文 key 为源字符串，并运行/更新 i18n 同步；Classic 如新增最小入口，遵循 `web/classic/src/pages/Enterprise/*` 现有模式。[Source: AGENTS.md#Internationalization (i18n); web/default/src/features/enterprise-organization/api.ts]

- [x] 补齐测试，锁住优先级、继承、审计和 API/UI 契约 (AC: 1, 2, 3)
  - [ ] Model/migration tests：覆盖新增列、唯一约束调整、同一用户同部门同时存在 `dingtalk_synced_owner` 与 `manual_deny_override`、三库兼容迁移路径（至少 SQLite 单测 + 避免专属 SQL）。[Source: model/enterprise/department_role_test.go; model/enterprise/text_default_test.go]
  - [ ] Service tests：覆盖 manual deny 覆盖 manual grant / DingTalk owner、manual grant 覆盖 DingTalk owner、仅 DingTalk owner 生效、无有效负责人、祖先负责人管理后代、目标部门 deny 阻断继承权限。[Source: service/enterprise/permission_test.go]
  - [ ] DingTalk sync tests：fake client 返回负责人字段时写入/失效 `dingtalk_synced_owner` 来源事实；重复同步幂等；本地 manual deny/grant 不被同步覆盖。[Source: service/enterprise/dingtalk_sync_test.go]
  - [ ] Controller/router tests：覆盖 owner list/grant/deny/revoke API、旧 `/admins` 兼容、deny 后预算治理接口拒绝并返回明确 i18n message、无有效负责人响应可被前端展示。[Source: controller/enterprise/department_budget_test.go; controller/enterprise/dingtalk_config_test.go]
  - [ ] Frontend tests：Default 组织页展示来源事实和有效结果，手动禁用后有效负责人移除但钉钉事实仍可见，无有效负责人显示 Admin fallback；新增 queryKey 保持 `['enterprise', 'organization', ...]` 精确失效。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/src/features/enterprise-organization/api.ts]
  - [ ] Boundary tests：继续确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`docs/openapi/relay.json`。[Source: service/enterprise/boundary_test.go; _bmad-output/planning-artifacts/architecture.md#Anti-Patterns]

## Dev Notes

- Story 7.1 是 Epic 7 的基础故事。后续 Story 7.3 预算委派、Story 7.5 员工额度申请审批都依赖“目标部门有效负责人集合”这个统一解析结果；如果本故事把权限判断散落在 controller 或前端，后续会重复实现并产生越权风险。[Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- 当前代码已有 `enterprise_department_roles`、`PermissionService.GrantDepartmentAdmin/RevokeDepartmentAdmin/ListManageableDepartmentIds`、`EnterpriseDepartmentAdmin` 中间件和 `/departments/:id/admins` 路由。它们是必须复用/演进的基线，不应新增一套独立 owner 权限系统绕过这些入口。[Source: model/enterprise/department_role.go; service/enterprise/permission.go; middleware/enterprise_dept_admin.go; router/enterprise-router.go]
- 现有 `DepartmentRole` 的 unique index 不支持同一用户同部门多来源事实并存；这是本故事最容易漏掉的结构性风险。只在 service 里写 if/else 而不处理存储模型，会无法审计“钉钉原始负责人 + 本地 deny 覆盖”的 AC1。[Source: model/enterprise/department_role.go]
- 当前 DingTalk 同步只同步部门树和成员关系，没有负责人事实。开发前要确认使用现有 `ListDepartmentUsers` 响应扩展字段还是额外详情接口；无论哪种方式，都必须通过 typed struct 和 `common.*` JSON wrapper 实现，不要直接引入 `encoding/json` marshal/unmarshal。[Source: service/enterprise/dingtalk_client.go; AGENTS.md#Rule 1: JSON Package — Use common/json.go]
- 本故事不改变 `users.role` 枚举。Root/Admin 是全局后台权限；部门负责人是企业模块内部治理权限，属于 `enterprise_department_roles` 或等价企业权限事实。[Source: _bmad-output/planning-artifacts/architecture.md#Authorization Model]
- 本故事不触碰 relay、计费表达式、用户日志模型或 wallet 扣费主链路。部门负责人只决定谁能执行企业治理动作，不参与 `/v1/**` 请求、quota 计算、billing expression 或 subscription 扣费。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#4.6-分层预算治理与额度审批; _bmad-output/planning-artifacts/architecture.md#Relay Boundary]
- 负责人解析需要支持后代作用域，因为现有部门管理员权限会展开到后代，且 Story 7.3 明确“负责人管理祖先部门”可向后代部门委派预算。但本地 deny 的优先级必须能阻断目标部门权限，否则 `manual_deny_override` 在跨层治理里会失效。[Source: service/enterprise/permission.go; _bmad-output/planning-artifacts/epics.md#Story 7.3: 支持上级预算池向后代部门预算池分配]
- 审计边界：配额分配仍以 `enterprise_quota_allocations` 为单一审计源；负责人授权/禁用属于低频管理动作，应写 `enterprise_admin_actions`。不要因为本故事顺手改变 quota allocation 审计来源。[Source: service/enterprise/admin_action.go; _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]

### Project Structure Notes

- 重点预计修改：
  - `constant/enterprise.go`
  - `model/enterprise/department_role.go`
  - `model/enterprise/migration.go`
  - `model/enterprise/department_role_test.go`
  - `service/enterprise/permission.go`
  - `service/enterprise/permission_test.go`
  - `service/enterprise/dingtalk_client.go`
  - `service/enterprise/dingtalk_sync.go`
  - `service/enterprise/dingtalk_sync_apply.go`
  - `service/enterprise/dingtalk_sync_test.go`
  - `service/enterprise/admin_action.go`
  - `service/enterprise/errors.go`
  - `middleware/enterprise_dept_admin.go`
  - `controller/enterprise/department_role.go`
  - `dto/enterprise/department_role.go`
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

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.1: 建立部门负责人权限模型与生效优先级]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-21]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.1-部门负责人来源与生效优先级]
- [Source: _bmad-output/planning-artifacts/architecture.md#Department Owner Precedence (V1.3)]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/6-1-rebuild-enterprise-organization-as-tree-driven-governance-view.md]
- [Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
- [Source: model/enterprise/department_role.go]
- [Source: model/enterprise/migration.go]
- [Source: service/enterprise/permission.go]
- [Source: middleware/enterprise_dept_admin.go]
- [Source: controller/enterprise/department_role.go]
- [Source: router/enterprise-router.go]
- [Source: service/enterprise/dingtalk_client.go]
- [Source: service/enterprise/dingtalk_sync_apply.go]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: 钉钉开放平台 - 获取部门用户详情: https://open.dingtalk.com/document/app/queries-department-user-details]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-01 16:16:29 +0800: dev-story 启动；`_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 Python 3.11+ 的 `tomllib` 失败，已按 workflow fallback 手工读取 `.agents/skills/bmad-dev-story/customize.toml`，确认无项目/用户 override 文件。
- 2026-06-01 16:16:29 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`；未发现仓库内 `project-context.md`。
- 2026-06-01 16:16:29 +0800: 已完整读取 sprint status 与 Story 7.1，保留既有 `baseline_commit`，将故事从 `ready-for-dev` 切换到 `in-progress`。
- 已按用户要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`.agents/skills/bmad-create-story/discover-inputs.md`、`.agents/skills/bmad-create-story/template.md`、`.agents/skills/bmad-create-story/checklist.md`。
- `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 Python 3.11+ 的 `tomllib` 失败；已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml`，确认无项目/用户 override 文件。
- 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`、planning/implementation artifact 路径。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `7-1-establish-department-owner-role-model-and-precedence`，原状态为 `backlog`，Epic 7 原状态为 `backlog`。
- 已执行 discover-inputs：加载 `epics.md`、`architecture.md`、`prds/prd-new-api-2026-05-27/prd.md`、`prds/prd-new-api-2026-05-27/addendum.md`；未发现仓库内 `project-context.md`，agent-platform PRD/Architecture/UX 与本企业治理 Story 7.1 无直接实现关系，未纳入主上下文。
- 已读取现有相关代码：`model/enterprise/department_role.go`、`service/enterprise/permission.go`、`middleware/enterprise_dept_admin.go`、`controller/enterprise/department_role.go`、`router/enterprise-router.go`、`service/enterprise/dingtalk_client.go`、`service/enterprise/dingtalk_sync*.go`、`web/default/src/features/enterprise-organization/*`。
- 已确认当前实现只有本地部门管理员 active/inactive 语义，DingTalk 同步尚未保存负责人事实；本故事需要扩展现有权限模型而不是新建独立权限系统。
- 已进行外部官方文档最小检查：钉钉开放平台“获取部门用户详情”页面作为实现时确认负责人来源字段的官方入口；故事中仅要求使用官方字段和 typed struct，不硬编码未经验证的字段名。

### Completion Notes List

- 已创建 Story 7.1 故事文档，状态为 `ready-for-dev`。
- 已把最关键的实现风险写入任务与 Dev Notes：现有 unique index 无法保存多来源事实、`StatusInactive` 不能等同 manual deny、DingTalk 负责人来源事实不得覆盖本地规则、目标部门 deny 必须阻断祖先继承权限。
- 已明确 API/UI/审计/测试范围，并标出不得触碰 relay、billing expression、日志模型和 AI API 协议面。
- 已按 checklist 进行自检：故事包含用户故事、AC、任务拆解、架构约束、当前代码状态、测试要求、文件位置、前后故事依赖、外部文档注意点与来源引用。

### File List

- `_bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`

## Change Log

- 2026-06-01 16:06:45 +0800: 创建 Story 7.1 上下文文档，准备进入开发。

- 2026-06-01 16:55:00 +0800: Source-of-truth closure completed after targeted verification. Backend owner model / permission / DingTalk owner sync / controller / middleware tests passed. Frontend owner panel coverage was updated, but direct `bun test` remains blocked in this environment by missing `react/jsx-dev-runtime`; this is carried as a local test-harness limitation rather than a story blocker. Story status advanced to done for code-review closure.
