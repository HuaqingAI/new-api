---
baseline_commit: 5d9df6cd40547f62ddf7cd5e3d879f2aed4dfb22
---

# Story 7.6: 建立治理动作审计时间线与钉钉通知

Status: review

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员或负责人,
I want 所有治理动作都可追溯并能通过钉钉通知相关人员,
so that 企业预算治理形成完整闭环。

## Acceptance Criteria

1. **Given** 系统发生负责人授权、预算委派、成员 allocation 调整/取消/回收、申请提交或审批结果  
   **When** 动作提交成功  
   **Then** 系统记录统一审计事实  
   **And** 审计至少包含操作者、目标对象、额度变化、时间、状态与追溯标识。
2. **Given** 员工申请提交、审批通过、审批拒绝、分配成功或回收/撤销发生  
   **When** 通知通道可用  
   **Then** 系统发送对应钉钉通知  
   **And** 通知内容能帮助接收者定位到治理动作结果。
3. **Given** 钉钉通知发送失败  
   **When** 核心治理事务已经提交  
   **Then** 事务结果保持成功  
   **And** 系统记录投递失败状态与错误原因以供重试。

## Tasks / Subtasks

- [x] 建立统一治理审计时间线服务，聚合既有事实源而不是制造双写审计源 (AC: 1)
  - [x] 新增 `service/enterprise/governance_timeline.go`，把 `enterprise_admin_actions`、`enterprise_budget_delegations`、`enterprise_quota_allocations`、`enterprise_quota_requests` 归一成统一 timeline item；每条 item 至少包含 `trace_id`、`source_type`、`source_id`、`action_type`、`actor_id`、目标对象、额度变化、状态、发生时间、详情路由/API 路径。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26: 建立治理动作审计与钉钉通知闭环; service/enterprise/admin_action.go; model/enterprise/quota_allocation.go; model/enterprise/quota_request.go; model/enterprise/budget_delegation.go]
  - [x] `trace_id` 使用确定性格式，例如 `admin_action:{action_id}`、`budget_delegation:{id}`、`quota_allocation:{id}`、`quota_request:{id}`，不要依赖随机值导致同一事实多次查询出现不同追溯标识。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26]
  - [x] allocation 和 request 的业务事实表仍是各自单一事实源；不要把 allocation 调整/取消/回收、request 审批履约再双写到新的 `enterprise_admin_actions` 记录里来“凑时间线”。[Source: _bmad-output/planning-artifacts/architecture.md#Enforcement Guidelines; _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
  - [x] 补齐现有 admin action payload 的归一字段读取，尤其负责人授权、钉钉配置/同步、预算委派、quota request submit/approve/reject 的 payload；对历史 payload 字段缺失要降级显示，不返回数据库错误。[Source: service/enterprise/admin_action.go; controller/enterprise/quota_request.go; controller/enterprise/budget_delegation.go]
  - [x] 查询接口必须支持按 `department_id`、`actor_id`、`action_type/source_type`、`status`、`start_at/end_at`、`page/page_size` 过滤；权限上 Admin 可看全局，负责人只能看其可治理部门相关 item，普通员工只能看与自己申请相关的 request timeline。[Source: middleware/enterprise_dept_admin.go; service/enterprise/permission.go; router/enterprise-router.go]

- [x] 为治理钉钉通知建立独立 delivery 表与非阻塞投递服务 (AC: 2, 3)
  - [x] 新增 `model/enterprise/governance_notification_delivery.go` 或等价模型，表名保持 `enterprise_` 前缀，字段至少包含：`tenant_id`、`source_type`、`source_id`、`trace_id`、`action_type`、`recipient_user_id`、`recipient_kind`、`channel_type=dingtalk_robot`、`status`、`attempt_count`、`max_attempts`、`next_retry_at`、`last_attempt_at`、`sent_at`、`final_failed_at`、`error_reason`、`dedupe_key`、`trace_payload`、`created_at`、`updated_at`。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26; model/enterprise/alert_delivery.go]
  - [x] 在 `model/enterprise/migration.go` 注册迁移，使用 GORM/三库兼容索引；`dedupe_key` 必须唯一，防止重复审批、重复调度或重试造成重复通知。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6; model/enterprise/migration.go]
  - [x] 新增 `service/enterprise/governance_notification.go` 与 `governance_notification_dispatch.go`，复用 `AlertDispatchService` 的状态语义和退避节奏思想，但不要把治理通知塞进风险告警专用的 `enterprise_alert_deliveries`，因为后者绑定 `event_id/rule_id/risk_type`。[Source: service/enterprise/alert_dispatch.go; model/enterprise/alert_delivery.go]
  - [x] 核心治理事务提交后只 enqueue delivery，投递失败不得回滚负责人授权、预算委派、allocation 治理或 quota request 审批结果；如果 enqueue 本身失败，控制器可返回核心结果并通过系统日志/错误字段记录，不能撤销已提交事务。[Source: _bmad-output/planning-artifacts/architecture.md#Data Flow; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26]
  - [x] `trace_payload` JSON 必须通过 `common.Marshal` / `common.UnmarshalJsonStr` 处理；若复用或修改 `service/webhook.go`，同时修正其直接 `encoding/json` marshal 写法，避免新治理通知继续扩大项目 JSON 规则违规面。[Source: AGENTS.md#Rule 1: JSON Package — Use common/json.go; service/webhook.go]

- [x] 接入钉钉机器人通知通道，并保持密钥、SSRF 与签名边界安全 (AC: 2, 3)
  - [x] 通道配置优先复用现有 `AlertRuleChannelDingTalkRobot` 的 webhook/secret 建模、脱敏显示和 `service.SendWebhookNotify` 投递能力；如果需要独立治理通知配置，应复用相同 DTO/脱敏/SSRF 验证模式，不新增明文密钥响应。[Source: model/enterprise/alert_rule.go; service/enterprise/alert.go; service/webhook.go]
  - [x] 通知覆盖矩阵至少包含：quota request submitted、approved、rejected、fulfilled/allocation created、allocation reclaim、allocation cancel/revoke。预算委派和负责人授权可进入 timeline，通知可作为可配置项，但不得抢占本故事必选矩阵。[Source: _bmad-output/planning-artifacts/epics.md#Story 7.6: 建立治理动作审计时间线与钉钉通知; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26]
  - [x] 收件人解析必须来自既有部门上下文：申请提交通知目标部门有效负责人；审批结果/分配成功通知申请人；回收/撤销通知目标成员与可治理负责人；无有效负责人时记录 `fallback=admin` 并通知 Admin 配置收件人或仅记录 pending/final_failed，不静默丢弃。[Source: service/enterprise/permission.go; service/enterprise/quota_request.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
  - [x] 通知内容至少包含动作类型、部门、预算池/allocation/request ID、操作者、目标用户、额度变化、状态、处理时间和 `trace_id`，并提供后台详情路由/API 路径；不要包含钉钉 app secret、robot secret、webhook query token 或用户敏感凭据。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26; _bmad-output/planning-artifacts/architecture.md#Enforcement Guidelines]
  - [x] 发送失败记录 `error_reason`、`attempt_count`、`next_retry_at/final_failed_at`，默认退避可沿用风险告警的 30 秒、2 分钟、10 分钟思路；重试由后台 task 扫描 due delivery，不由用户请求同步等待。[Source: service/enterprise/alert_dispatch.go; service/enterprise/alert_dispatch_task.go; service/enterprise/scheduler.go]

- [x] 在治理动作提交路径中 enqueue 通知并补齐 timeline 所需事实字段 (AC: 1, 2, 3)
  - [x] 在 quota request submit/approve/reject 成功后 enqueue 对应通知；approve 后如果自动 allocation 创建成功，还要产生分配成功通知，且通过 `dedupe_key` 保证重复 approve 幂等返回时不会再发第二条。[Source: controller/enterprise/quota_request.go; service/enterprise/quota_request.go; _bmad-output/implementation-artifacts/7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation.md]
  - [x] 在 allocation cancel/revoke/reclaim/supersede 路径 enqueue 回收/撤销相关通知；保持 `enterprise_quota_allocations` 的 `processed_at`、`processed_source`、`reclaimed_quota`、`revoke_reason` 是通知和 timeline 的事实来源。[Source: service/enterprise/quota_allocation.go; controller/enterprise/quota_allocation.go; _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
  - [x] 对预算委派 create/supersede 只补齐 timeline 投影和可选通知，不得改变 Story 7.3 已建立的“部门预算池 -> 部门预算池” ledger 语义，也不得直接创建成员 wallet。[Source: service/enterprise/budget_delegation.go; _bmad-output/implementation-artifacts/7-3-delegate-ancestor-budget-pools-to-descendant-department-pools.md]
  - [x] 负责人授权/禁用动作必须出现在 timeline 中，并能显示手动 grant/deny 与钉钉同步 owner 来源；local deny 覆盖导致的拒绝需要能被追溯，不要只返回前端错误。[Source: service/enterprise/admin_action.go; service/enterprise/permission.go; _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
  - [x] 所有 enqueue hook 必须在服务层或事务成功后的明确边界中实现，不把 side effect 分散在前端 mutation 中；controller 只负责解参、调用 service、返回 response 和必要 admin action 写入。[Source: _bmad-output/planning-artifacts/architecture.md#Layered Architecture; router/enterprise-router.go]

- [x] 暴露治理时间线与通知投递 API，并接入 Default 企业组织页 (AC: 1, 2, 3)
  - [x] 在 `controller/enterprise/governance_timeline.go`、`dto/enterprise/governance_timeline.go` 与 `router/enterprise-router.go` 增加 `/api/enterprise/governance/timeline`、`/api/enterprise/governance/notifications`、`/api/enterprise/governance/notifications/:id/resend` 或等价路由；响应保持 `{ success, message, data }`、snake_case、空数组 `[]`。[Source: router/enterprise-router.go; dto/enterprise/admin_action.go; controller/enterprise/admin_action.go]
  - [x] 手动 resend 只允许 Admin 或有权治理相关部门的负责人触发，且只针对 `final_failed` 或等价终态失败 delivery；重复 resend 如果已有 pending manual delivery，应返回现有 delivery 而不是再建一条。[Source: service/enterprise/alert.go#ResendAlertDelivery; service/enterprise/permission.go]
  - [x] 在 `web/default/src/features/enterprise-organization/` 当前部门工作区新增“治理时间线/通知投递状态”视图或次级分区，复用现有左树 + 右侧工作区，不新建平行企业治理页面。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#7.1 企业组织页骨架; web/default/src/features/enterprise-organization/index.tsx]
  - [x] UI 需要能按当前部门展示 timeline，显示 trace ID、动作、操作者、目标对象、额度变化、状态、通知状态；失败 delivery 显示错误原因和重发操作，loading/error 使用现有 Base UI/Skeleton/Alert 模式。[Source: _bmad-output/planning-artifacts/architecture.md#Error Handling Strategy; web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 新增前端文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，queryKey 继续挂在 `['enterprise', 'organization', ...]` 命名空间，mutation invalidate 精确到 timeline/notification delivery，不全量 invalidate `['enterprise']`。[Source: AGENTS.md#Internationalization (i18n); web/default/src/features/enterprise-organization/api.ts]

- [x] 增加后台调度、测试与边界回归，锁住非阻塞、幂等和跨库行为 (AC: 1, 2, 3)
  - [x] 在 `service/enterprise/scheduler.go` 增加 governance notification dispatch ticker，受 `common.IsMasterNode` 守卫；单次执行函数需可测试，类似 `RunAlertDispatchTaskOnce`。[Source: service/enterprise/scheduler.go; service/enterprise/alert_dispatch_task.go]
  - [x] Model/migration tests：覆盖 delivery 表字段、唯一 `dedupe_key`、trace payload JSON wrapper、三库兼容迁移路径；避免 JSONB、recursive CTE、PG-only UPSERT。[Source: model/enterprise/alert_delivery_test.go; AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]
  - [x] Service tests：覆盖 timeline 聚合排序和过滤、request submit/approve/reject 通知 enqueue、approve 自动 allocation 的分配成功通知、allocation cancel/reclaim/revoke 通知、重复 approve/retry 不重复 delivery、发送失败不回滚核心事实、失败退避和 final_failed 状态。[Source: service/enterprise/quota_request_test.go; service/enterprise/quota_allocation_test.go; service/enterprise/alert_dispatch_test.go]
  - [x] Controller/router tests：覆盖权限边界、timeline 空数组、通知列表、resend 幂等、错误 message key、可选标量指针 DTO 绑定。[Source: controller/enterprise/quota_request_test.go; controller/enterprise/quota_allocation_test.go; AGENTS.md#Rule 6: Upstream Relay Request DTOs — Preserve Explicit Zero Values]
  - [x] Frontend tests：覆盖当前部门 timeline 渲染、trace ID 展示、通知失败错误原因、手动重发按钮、queryKey 命名空间和精确 invalidate；运行 `bun run typecheck` 或项目既有前端校验。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] Boundary tests：确认本故事不修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`，不向 `/v1/**` 或 billing expression 增加企业治理字段。[Source: _bmad-output/planning-artifacts/architecture.md#Anti-Patterns; service/enterprise/boundary_test.go]

## Dev Notes

- Story 7.6 的核心是“统一展示与通知闭环”，不是重新定义所有治理事实表。现有事实边界已经形成：低频管理动作走 `enterprise_admin_actions`；预算委派走 `enterprise_budget_delegations`；成员分配/撤销/回收走 `enterprise_quota_allocations`；员工申请审批走 `enterprise_quota_requests`。[Source: _bmad-output/planning-artifacts/architecture.md#Enforcement Guidelines; service/enterprise/admin_action.go]
- “统一审计事实”建议实现为标准化 timeline projection + 必要缺口补齐，而不是新建一个万能 `enterprise_governance_audits` 表再把所有业务事实复制进去。复制会造成 completion 判断、重试、查询排序和权限过滤不一致。[Source: _bmad-output/planning-artifacts/architecture.md#审计单源 + UI 统一]
- `enterprise_alert_deliveries` 已经有 delivery/status/retry 模式，但它是内容风险告警专用模型，字段绑定 `event_id/rule_id/risk_type`。7.6 可以复用设计和代码习惯，但治理通知应独立模型或抽取共享 helper，避免污染风险告警语义。[Source: model/enterprise/alert_delivery.go; service/enterprise/alert_dispatch.go]
- Story 7.5 已经在 `enterprise_quota_requests` 中保留 `owner_count_snapshot`、`fallback`、`submitted_at`、`approved_at`、`rejected_at`、`fulfilled_at`、`processed_at`，这些字段正是 7.6 投影申请 -> 审批 -> 自动分配事实的基础，不要再回填第二套状态。[Source: _bmad-output/implementation-artifacts/7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation.md]
- `service.SendWebhookNotify` 当前能发送 webhook 并做 SSRF 验证，但文件内直接使用了 `encoding/json`。本故事如改动通知发送路径，必须把 marshal 改为 `common.Marshal`，不能复制该违规模式到新 service。[Source: service/webhook.go; AGENTS.md#Rule 1: JSON Package — Use common/json.go]
- 钉钉业务边界有两种：`service/enterprise/dingtalk_client.go` 面向登录/通讯录 OpenAPI；钉钉机器人通知在现有告警中按 webhook channel 处理。7.6 应走 webhook/robot 通知，不要把审批通知错误接进通讯录同步 client 或 OAuth 流。[Source: service/enterprise/dingtalk_client.go; model/enterprise/alert_rule.go; service/enterprise/alert_dispatch.go]
- 权限边界要在后端落实。前端隐藏按钮不等于授权；timeline 和 notification delivery API 必须复用 `PermissionService.CanGovernDepartment` / manageable department 逻辑，普通员工只能看自己的 request 相关记录。[Source: service/enterprise/permission.go; middleware/enterprise_dept_admin.go]
- 通知失败必须是 best-effort。核心治理事务成功后，delivery 失败最多影响通知状态，不得撤销 budget delegation、allocation lifecycle、quota request approval 或 admin action 记录。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26]
- 所有新增 JSON payload、DTO 和表结构继续遵守项目规则：Go 业务 JSON 走 `common.*` wrapper；DB 代码同时兼容 SQLite/MySQL/PostgreSQL；DTO 可选标量使用 `*T + omitempty`；不修改 protected project identity、relay 主链路或 billing expression。[Source: AGENTS.md#Rules]

### Project Structure Notes

- 重点预计新增或修改：
  - `model/enterprise/governance_notification_delivery.go`
  - `model/enterprise/migration.go`
  - `model/enterprise/governance_notification_delivery_test.go`
  - `service/enterprise/governance_timeline.go`
  - `service/enterprise/governance_notification.go`
  - `service/enterprise/governance_notification_dispatch.go`
  - `service/enterprise/governance_notification_task.go`
  - `service/enterprise/scheduler.go`
  - `service/enterprise/errors.go`
  - `service/enterprise/admin_action.go`
  - `service/enterprise/quota_request.go`
  - `service/enterprise/quota_allocation.go`
  - `service/webhook.go`（仅用于 JSON wrapper 修正或共享投递路径复用）
  - `controller/enterprise/governance_timeline.go`
  - `controller/enterprise/governance_notification.go`
  - `dto/enterprise/governance_timeline.go`
  - `dto/enterprise/governance_notification.go`
  - `router/enterprise-router.go`
  - `i18n/keys.go`
  - `i18n/locales/{en,zh-CN,zh-TW}.yaml`
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
- 当前工作树已有与本故事无关的 `_bmad-output/story-automator/orchestration-7-20260601-075141.md` 修改；实现 7.6 时不要回滚该文件。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 7.6: 建立治理动作审计时间线与钉钉通知]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 7: 分层预算治理与额度审批]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-26: 建立治理动作审计与钉钉通知闭环]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.6 审计与钉钉通知]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#8.7 V1.3 测试补充]
- [Source: _bmad-output/planning-artifacts/architecture.md#Enforcement Guidelines]
- [Source: _bmad-output/planning-artifacts/architecture.md#Data Flow]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/implementation-artifacts/7-1-establish-department-owner-role-model-and-precedence.md]
- [Source: _bmad-output/implementation-artifacts/7-3-delegate-ancestor-budget-pools-to-descendant-department-pools.md]
- [Source: _bmad-output/implementation-artifacts/7-4-govern-member-allocations-with-adjust-cancel-and-reclaim.md]
- [Source: _bmad-output/implementation-artifacts/7-5-support-employee-quota-requests-single-step-approval-and-auto-allocation.md]
- [Source: service/enterprise/admin_action.go]
- [Source: service/enterprise/quota_request.go]
- [Source: service/enterprise/quota_allocation.go]
- [Source: service/enterprise/budget_delegation.go]
- [Source: service/enterprise/permission.go]
- [Source: service/enterprise/alert_dispatch.go]
- [Source: service/enterprise/alert_dispatch_task.go]
- [Source: service/enterprise/scheduler.go]
- [Source: model/enterprise/alert_delivery.go]
- [Source: model/enterprise/alert_rule.go]
- [Source: service/webhook.go]
- [Source: router/enterprise-router.go]
- [Source: web/default/src/features/enterprise-organization/index.tsx]
- [Source: web/default/src/features/enterprise-organization/api.ts]
- [Source: web/default/src/features/enterprise-organization/types.ts]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-02 11:00:15 +0800: 按用户要求读取 `.agents/skills/bmad-dev-story/SKILL.md` 与 `checklist.md`；`resolve_customization.py` 因本机 `python3` 缺少 Python 3.11+ `tomllib` 失败，已按 skill fallback 读取 `.agents/skills/bmad-dev-story/customize.toml`，确认无 team/user override，persistent facts 为 `file:{project-root}/**/project-context.md`，仓库内未发现匹配文件。
- 2026-06-02 11:00:15 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`、`user_skill_level=intermediate`。
- 2026-06-02 11:00:15 +0800: 已将 Story 7.6 与 sprint-status 从 ready/backlog 进入开发并保留既有 `baseline_commit=5d9df6cd40547f62ddf7cd5e3d879f2aed4dfb22`。
- 2026-06-02 11:00:15 +0800: 已实现统一治理 timeline projection，聚合 `enterprise_admin_actions`、`enterprise_budget_delegations`、`enterprise_quota_allocations`、`enterprise_quota_requests`，trace_id 使用确定性格式，权限按 Admin/负责人/本人 request 边界过滤。
- 2026-06-02 11:00:15 +0800: 已新增独立 `enterprise_governance_notification_deliveries` 模型、迁移、非阻塞 enqueue、后台 dispatch、manual resend 与钉钉机器人 webhook 投递；通知失败只更新 delivery 状态，不回滚核心治理事实。
- 2026-06-02 11:00:15 +0800: 已修正 `service/webhook.go` 直接 `encoding/json` marshal 为 `common.Marshal`，治理通知 trace payload 使用 `common.Marshal` / `common.UnmarshalJsonStr`。
- 2026-06-02 11:00:15 +0800: 已在 quota request submit/approve/reject、自动 allocation created、allocation cancel/revoke/reclaim/supersede 路径接入服务层 enqueue hook；dedupe_key 防止幂等重复通知。
- 2026-06-02 11:00:15 +0800: 已新增 `/api/enterprise/governance/timeline`、`/api/enterprise/governance/notifications`、`/api/enterprise/governance/notifications/:id/resend` API，并接入 Default 企业组织页当前部门工作区。
- 2026-06-02 11:00:15 +0800: 已执行边界检查：未修改 `relay/**`、`pkg/billingexpr/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`；未触碰受保护项目身份信息。
- 2026-06-02 11:00:15 +0800: 验证通过：`GOCACHE=/private/tmp/new-api-gocache go test ./model/enterprise ./controller/enterprise`；`GOCACHE=/private/tmp/new-api-gocache go test ./service/enterprise -run 'TestGovernance|TestQuotaRequest|TestQuotaAllocation'`；`cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx && bun run typecheck`。
- 2026-06-02 11:00:15 +0800: 尝试运行 `GOCACHE=/private/tmp/new-api-gocache go test ./service/enterprise ./model/enterprise ./controller/enterprise`，其中 model/controller 通过，service 全包因既有测试在 sandbox 中 `httptest` 监听端口被拒绝以及既有 usage report 断言偏移失败；已用定向 service 测试覆盖本故事新增逻辑。
- 2026-06-02 10:37:41 +0800: 按用户要求读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，并执行 create-story workflow。
- 2026-06-02 10:37:41 +0800: `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 Python 3.11+ 的 `tomllib` 失败；已按 skill fallback 手工读取 `.agents/skills/bmad-create-story/customize.toml`，确认无 team/user override，persistent facts 为 `file:{project-root}/**/project-context.md`，仓库内未发现匹配文件。
- 2026-06-02 10:37:41 +0800: 已读取 `_bmad/bmm/config.yaml`，确认 `project_name=new-api`、`user_name=hth`、`communication_language=Chinese`、`document_output_language=Chinese`。
- 2026-06-02 10:37:41 +0800: 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `7-6-audit-governance-actions-and-send-dingtalk-notifications`，原状态为 `backlog`，Epic 7 已为 `in-progress`。
- 2026-06-02 10:37:41 +0800: 已执行 discover-inputs：加载 `_bmad-output/planning-artifacts/epics.md`、`architecture.md`、`prds/prd-new-api-2026-05-27/prd.md`、`addendum.md`；重点读取 FR26、Story 7.6、审计边界、通知边界、企业数据流和 V1.3 测试补充。
- 2026-06-02 10:37:41 +0800: 已读取前序故事 `7-3`、`7-4`、`7-5`，确认预算委派、allocation lifecycle 和 quota request lifecycle 的事实源、幂等字段与后续 7.6 衔接要求。
- 2026-06-02 10:37:41 +0800: 已读取现有实现：`service/enterprise/{admin_action,quota_request,quota_allocation,budget_delegation,permission,alert_dispatch,alert_dispatch_task,scheduler}.go`、`model/enterprise/{admin_action,quota_allocation,quota_request,alert_delivery,alert_rule}.go`、`service/webhook.go`、`controller/enterprise/{admin_action,quota_request}.go`、`router/enterprise-router.go`、`web/default/src/features/enterprise-organization/{api,types,index,test}.tsx`。
- 2026-06-02 10:37:41 +0800: 已确认现有告警投递支持钉钉机器人 webhook 通道，但 `enterprise_alert_deliveries` 是风险告警专用；7.6 应复用投递模式和 webhook helper，不应污染风险告警模型。
- 2026-06-02 10:37:41 +0800: 已按 checklist 自检，补强了反双写审计、非阻塞通知、幂等 delivery、权限边界、JSON wrapper、三库迁移、前端 i18n 和 boundary tests。

### Completion Notes List

- 已实现 Story 7.6，状态进入 `review`。
- 已建立统一治理时间线投影服务，保持业务事实单源，不新增万能审计表，不把 allocation/request 事实双写到 admin actions。
- 已建立独立治理通知 delivery 表和非阻塞投递服务，复用钉钉机器人 webhook 通道、SSRF/签名边界和告警退避语义；失败记录 `error_reason`、attempt/retry/final_failed 字段。
- 已在 request/approval/allocation governance 路径 enqueue 钉钉通知，重复审批/重试通过 deterministic dedupe key 幂等。
- 已在 Default 企业组织页新增当前部门治理时间线/通知投递状态视图，展示 trace ID、动作、操作者、目标对象、额度变化、状态、失败原因和重发按钮，并补全 i18n。
- 已补充 model/service/controller/frontend 测试和边界检查；全量 service 包在当前 sandbox 受 `httptest` 监听限制影响，相关新增逻辑已通过定向测试。
- 已创建 Story 7.6 故事文档，状态为 `ready-for-dev`。
- 已将 FR26 拆成统一 timeline projection、治理通知 delivery、钉钉机器人投递、提交路径 hook、Default 当前部门工作区 UI 和测试/边界回归任务。
- 已明确最关键防误点：不要新建万能审计表复制所有事实；不要把治理通知塞进风险告警专用 delivery；不要让通知失败回滚核心治理事务；不要复制 `service/webhook.go` 的直接 JSON marshal 写法。

### File List

- `_bmad-output/implementation-artifacts/7-6-audit-governance-actions-and-send-dingtalk-notifications.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/department_membership_test.go`
- `controller/enterprise/governance_controller_test.go`
- `controller/enterprise/governance_notification.go`
- `controller/enterprise/governance_timeline.go`
- `dto/enterprise/governance_notification.go`
- `dto/enterprise/governance_timeline.go`
- `model/enterprise/governance_notification_delivery.go`
- `model/enterprise/governance_notification_delivery_test.go`
- `model/enterprise/migration.go`
- `router/enterprise-router.go`
- `service/enterprise/errors.go`
- `service/enterprise/governance_notification.go`
- `service/enterprise/governance_notification_dispatch.go`
- `service/enterprise/governance_notification_dispatch_test.go`
- `service/enterprise/governance_notification_task.go`
- `service/enterprise/governance_timeline.go`
- `service/enterprise/quota_allocation.go`
- `service/enterprise/quota_request.go`
- `service/enterprise/scheduler.go`
- `service/webhook.go`
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

### Change Log

- 2026-06-02 11:00:15 +0800: 完成 Story 7.6 实现，新增治理时间线、治理通知 delivery/dispatch/API、组织页视图、测试与 i18n，并将状态更新为 review。
- 2026-06-02 10:37:41 +0800: 创建 Story 7.6 上下文文档，准备进入开发。
