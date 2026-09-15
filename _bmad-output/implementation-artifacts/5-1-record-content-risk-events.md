---
baseline_commit: 1a2769c8644ac687d51411d61761bcafc11f51d3
---

# Story 5.1: 记录内容风险事件

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 安全/合规负责人,
I want 系统记录触发敏感词或过滤规则的风险事件,
so that 后续可以追踪风险来源和请求上下文。

## Acceptance Criteria

1. **Given** 请求触发现有敏感词检查或过滤规则  
   **When** 风险事件产生  
   **Then** 系统记录用户、部门成员关系快照、模型、时间、请求追踪 ID、风险类型和处理结果  
   **And** 风险事件落库失败不得扩大到普通日志敏感内容可见范围。
2. **Given** 用户属于多个部门  
   **When** 系统记录风险事件  
   **Then** 风险事件保存请求发生时的部门成员关系列表  
   **And** 后续按“包含某部门”语义筛选命中。
3. **Given** 普通用户查看日志或事件  
   **When** 页面或 API 返回数据  
   **Then** 不暴露管理员专用风险字段  
   **And** 遵守现有日志脱敏策略。

## Tasks / Subtasks

- [x] 新增企业风险事件模型与迁移，作为 Epic 5 的单一事件源 (AC: 1, 2)
  - [x] 新增 `model/enterprise/alert_event.go`，表名使用 `enterprise_alert_events`，至少覆盖：`id`、`tenant_id`、`user_id`、`username`、`request_id`、`model_name`、`risk_type`、`action_result`、`department_snapshot`、`summary`、`created_at`、`updated_at`。`department_snapshot` 和其他数组/对象字段使用 `TEXT` 存储并通过 `common.Marshal` / `common.UnmarshalJsonStr` 做序列化，不使用 `JSONB`、`GROUP_CONCAT` 或数据库专属类型。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; _bmad-output/planning-artifacts/architecture.md#Data Architecture; AGENTS.md#rule-1-json-package--use-commonjsongo; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] `department_snapshot` 必须保存“请求发生时”的部门成员关系列表，而不是查询时再 join 当前 `enterprise_user_departments`。快照元素至少应包含 `department_id`、`department_name`，必要时可附带 `external_source` / `status` 等最小追溯字段，但不要把整个部门对象或无关审计字段塞进去。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-17-记录内容风险事件-v11-修订; _bmad-output/planning-artifacts/epics.md#Story 5.1: 记录内容风险事件]
  - [x] 文本字段遵循现有 enterprise 模型测试约束：`type:text` 字段不要声明数据库默认值，应用层在 `BeforeCreate` / `normalize` 中兜底为空 JSON 字符串（如 `[]` / `{}` / 空摘要），并新增对齐 `model/enterprise/text_default_test.go`、`model/enterprise/usage_snapshot_test.go` 的测试模式。[Source: model/enterprise/text_default_test.go; model/enterprise/usage_snapshot_test.go]
  - [x] 在 `model/enterprise/migration.go` 注册 `AlertEvent` 迁移，保持与现有 `Department`、`UsageSnapshot`、`UsageReportJob` 一致的 GORM `AutoMigrate` 路径；不要为 Story 5.1 直接引入 `alert_rules` / `alert_deliveries` 表，这些属于后续 5.3/5.4/5.5。[Source: model/enterprise/migration.go; _bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure]

- [x] 实现风险事件记录 service，并以“旁路写入、不阻塞 relay”为核心约束 (AC: 1, 2)
  - [x] 新增 `service/enterprise/alert.go` 或等价命名的服务，提供统一 `RecordRiskEvent(...)` 入口，封装：上下文提取、部门快照构建、字段净化、落库和错误处理。后续 5.2/5.4/5.6 都应复用这一事件源，不要把记录逻辑散落在 controller、middleware 或多个 service 内。[Source: _bmad-output/planning-artifacts/architecture.md#Content Monitoring Alerting; _bmad-output/planning-artifacts/architecture.md#Integration Patterns]
  - [x] 事件记录至少从请求上下文中提取：`common.RequestIdKey`、用户 ID / username、模型名、发生时间、风险类型、处理结果；并查询当前有效的 `enterprise_user_departments` 生成快照。未归属用户必须保存为空快照或等价空列表，不能伪造一个“未归属部门 ID”。[Source: controller/relay.go; model/enterprise/user_department.go; service/enterprise/department_membership.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-17-记录内容风险事件-v11-修订]
  - [x] 风险事件只保留追溯所需摘要，不存原始 prompt、完整命中内容、完整消息数组或其他会扩大敏感信息可见范围的数据。若需要记录命中信息，限制为风险类型、摘要说明、处理结果和可审计的最小上下文。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-17-记录内容风险事件-v11-修订; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#421; _bmad-output/test-artifacts/test-design-progress.md#Privacy]
  - [x] 落库失败必须是旁路失败：只记录脱敏 warning/error，不得改变 `Relay` 对客户端的响应语义，不得把原始敏感词列表重新写进普通日志，也不得反向影响预扣费/退款/重试链路。[Source: controller/relay.go; _bmad-output/planning-artifacts/architecture.md#Data Flow; _bmad-output/test-artifacts/test-design-progress.md#Security]

- [x] 将当前已存在的敏感词拦截链路接入风险事件记录点，并为后续过滤规则预留统一入口 (AC: 1, 2)
  - [x] 在 `controller/relay.go` 现有 `needSensitiveCheck` / `CheckSensitiveText` 命中分支中接入 enterprise 风险事件记录。当前仓库已明确存在的触发点是敏感词阻断；Story 5.1 至少要把这个路径接通，不能只建表不接入口。[Source: controller/relay.go; service/sensitive.go; setting/sensitive.go]
  - [x] 接入时保持“先判定风险，再旁路写事件，再返回既有错误”的顺序，确保 `types.ErrorCodeSensitiveWordsDetected`、普通日志和上层调用方行为不回归。不要把事件写入放到 `model/log.go`、`controller/log.go` 或 `relay/**` 内部。[Source: controller/relay.go; AGENTS.md#overview; _bmad-output/planning-artifacts/epics.md#Story 5.1: 记录内容风险事件]
  - [x] “过滤规则”在当前仓库里尚未看到与 Story 5.1 等价的第二条明确实现路径，因此 service API 需要支持未来从其他过滤/拦截点复用，但本故事不要为了抽象而提前引入复杂事件总线、消息队列或独立 scheduler。[Source: _bmad-output/planning-artifacts/architecture.md#Async Work; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#443]

- [x] 明确读写边界，防止 Story 5.1 提前把查询/UI 做成错误形态 (AC: 3)
  - [x] Story 5.1 的核心是“记录事件”，不是完成完整事件查询台。若为了验证最小可用性引入 `dto/enterprise/alert.go`、`controller/enterprise/alert.go` 或 `/api/enterprise/alerts/events` 只读入口，必须挂在 `/api/enterprise/**` 并使用 `middleware.EnterpriseAdmin()`；完整筛选、分页和页面交互由 Story 5.2 负责。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; router/enterprise-router.go; _bmad-output/planning-artifacts/epics.md#Story 5.2: 查询和筛选风险事件]
  - [x] 不要修改 `controller/log.go`、`model/log.go`、Default `usage-logs` 模块去暴露管理员专用风险字段。普通用户现有日志查询仍应保持既有脱敏行为，风险事件不能混入通用 usage logs 结果集。[Source: controller/log.go; AGENTS.md#overview; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#330]
  - [x] 若新增面向管理员的 DTO，返回字段只包含 Story 5.1 所需的最小追溯信息和脱敏摘要，不把完整部门对象、原文 prompt、命中词原串或通知/规则字段混入 5.1 的事件模型；规则、通道、投递状态属于后续故事。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道; _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知]

- [x] 补齐模型、service、控制器/集成测试，锁定隐私、快照和旁路失败语义 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/alert_event_test.go`，覆盖迁移成功、文本字段无数据库默认值、JSON 空列表归一化、部门快照解析，以及三库兼容所需的 schema/索引约束。[Source: model/enterprise/text_default_test.go; model/enterprise/usage_snapshot_test.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 新增 `service/enterprise/alert_test.go`，覆盖：单部门与多部门用户快照、未归属用户空快照、摘要字段不含原始敏感内容、落库失败不向上冒泡为 relay 业务错误、以及后续“按包含某部门筛选”所需的快照结构可用性。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.1: 记录内容风险事件; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#328]
  - [x] 为 `controller/relay.go` 增加定向回归测试或 `tests/api` 集成测试，至少验证：敏感词命中时会产生一条风险事件；普通响应仍返回既有敏感词错误；风险事件失败不会让接口额外泄露敏感词或改变 HTTP/业务响应形态。[Source: controller/relay.go; _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
  - [x] 若 Story 5.1 增加任何管理员读接口，补齐普通用户/无企业权限用户拒绝访问测试，并验证响应中不出现管理员专用风险字段。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.1: 记录内容风险事件; _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md]

## Dev Notes

- Story 5.1 的权威落点是“在现有敏感词/过滤拦截链路旁路写 `enterprise_alert_events`”，不是改造 relay 协议，也不是把风险数据并入现有 `logs` 表。架构已经明确 FR-17 对应 `enterprise_alert_events`，而普通日志语义和 `/v1/**` 协议表面必须保持不变。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; AGENTS.md#overview]
- 当前仓库可确认的现成触发点是 `controller/relay.go` 中 `needSensitiveCheck` 命中后直接返回 `types.ErrorCodeSensitiveWordsDetected`。这里是 Story 5.1 的最小必须接入点。开发时不要跳过现有触发链路，只做“未来也许会用到”的抽象层。[Source: controller/relay.go; service/sensitive.go]
- 风险事件必须保存“请求发生时”的部门快照，而不是当前部门关系。原因有两个：一是 AC2 明确要求快照；二是 Epic 5 后续的查询、告警和风险概览都要和这份事件事实源对齐。用当前 `enterprise_user_departments` 回查会让部门变更污染历史事件语义。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#328; model/enterprise/user_department.go]
- 隐私是本故事的最高回归边界之一。PRD 与测试设计都只授权“追溯所需摘要”，没有授权记录原始 prompt 或完整命中内容。因此设计字段时宁可少，也不要为了排障方便扩大敏感信息落库或普通日志可见范围。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#421; _bmad-output/test-artifacts/test-design-epic-1-5.md#102]
- Story 5.1 不应抢 Story 5.2/5.3/5.4/5.5 的范围。不要在这里提前做完整筛选台、规则 DSL、告警投递状态表、重试调度或 Default `enterprise-alerts` 整页。5.1 的正确完成态是：事件模型稳、入口接通、隐私和权限边界锁住、后续故事有可复用的数据事实源。[Source: _bmad-output/planning-artifacts/epics.md#Story 5.2: 查询和筛选风险事件; _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道; _bmad-output/planning-artifacts/epics.md#Story 5.4: 发送可追溯告警通知]
- Epic 4 retrospective 已经给 Epic 5 明确了两个 guardrail：一是 enterprise 模块现在是“任务 + 查询 API + 文档合同”的完整子系统，不适合用轻量 CRUD 心态实现；二是接口合同/文档同步必须成为完成标准。若 5.1 新增任何管理员 API，请同步 `docs/openapi/api.json`，不要把文档滞后留给后续故事补洞。[Source: _bmad-output/implementation-artifacts/epic-4-retro-2026-05-29.md]
- 现有 enterprise 后台任务统一走 `service/enterprise/scheduler.go`，但 Story 5.1 不需要 scheduler。风险事件记录发生在请求处理路径，必须是同步旁路写入或可测试的轻量封装，不要平地再起一套队列/轮询机制。[Source: service/enterprise/scheduler.go; _bmad-output/planning-artifacts/architecture.md#Background Tasks]
- 审计模式上，风险事件与企业管理动作是两条不同事实流。`enterprise_admin_actions` 适合记录低频管理操作；Story 5.1 的请求级风险事件不应双写到 `enterprise_admin_actions`。两者职责必须分离，避免后续 5.2/5.4 查询口径混乱。[Source: _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md; service/enterprise/admin_action.go]
- 本次 story 创建未发现可用的 `project-context.md` 持久事实文件，因此本故事主要以上游 skill、AGENTS 规则、PRD、Architecture、Epic 5、Epic 4 retrospective 与当前源码实现为上下文依据。

### Project Structure Notes

- 预计新增文件：
  - `model/enterprise/alert_event.go`
  - `model/enterprise/alert_event_test.go`
  - `service/enterprise/alert.go`
  - `service/enterprise/alert_test.go`
  - `dto/enterprise/alert.go`（仅当本故事确实需要最小管理员读接口时）
  - `controller/enterprise/alert.go`（仅当本故事确实需要最小管理员读接口时）
- 预计修改文件：
  - `model/enterprise/migration.go`
  - `service/enterprise/errors.go`
  - `controller/relay.go`
  - `router/enterprise-router.go`（仅当新增最小管理员读接口时）
  - `tests/api/*` 或相关 relay/controller 测试文件
- 保持不改或只读依赖：
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`
  - `service/enterprise/scheduler.go`
  - `web/default/src/features/usage-logs/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.1: 记录内容风险事件]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.2: 查询和筛选风险事件]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.3: 配置内容风险告警规则和通道]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 5: 内容风险事件、告警与部门风险概览]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/planning-artifacts/architecture.md#Data Architecture]
- [Source: _bmad-output/planning-artifacts/architecture.md#Content Monitoring Alerting]
- [Source: _bmad-output/planning-artifacts/architecture.md#Background Tasks]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-17-记录内容风险事件-v11-修订]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#421]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#Privacy]
- [Source: _bmad-output/test-artifacts/test-design-progress.md#ENT-SEC-P0-002]
- [Source: _bmad-output/test-artifacts/test-design-epic-1-5.md#102]
- [Source: _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md]
- [Source: _bmad-output/implementation-artifacts/epic-4-retro-2026-05-29.md]
- [Source: AGENTS.md#rule-1-json-package--use-commonjsongo]
- [Source: AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- [Source: controller/log.go]
- [Source: controller/relay.go]
- [Source: model/enterprise/migration.go]
- [Source: model/enterprise/text_default_test.go]
- [Source: model/enterprise/usage_snapshot_test.go]
- [Source: model/enterprise/user_department.go]
- [Source: router/enterprise-router.go]
- [Source: service/enterprise/admin_action.go]
- [Source: service/enterprise/department_membership.go]
- [Source: service/enterprise/errors.go]
- [Source: service/enterprise/scheduler.go]
- [Source: service/sensitive.go]
- [Source: setting/sensitive.go]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 按用户要求先读取 `.agents/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`，并依据 skill 指令执行 create-story。
- `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 `tomllib` 无法运行，已按 skill fallback 规则手工读取 `.agents/skills/bmad-create-story/customize.toml` 与 `_bmad/bmm/config.yaml`；确认 prepend/append 为空，persistent fact 仅为可选 `project-context.md`。
- 已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`，确认目标 story key 为 `5-1-record-content-risk-events`，原状态为 `backlog`，且因这是 Epic 5 的首个 story，需要同步将 `epic-5` 从 `backlog` 更新为 `in-progress`。
- 已加载并分析 Epic 5、PRD FR-17/FR-20、Architecture 的 alert/event 映射、Epic 4 retrospective、测试设计文档，以及现有 `service/sensitive.go`、`controller/relay.go`、`model/enterprise/migration.go`、`router/enterprise-router.go`、`service/enterprise/admin_action.go`、`service/enterprise/permission.go`、`controller/log.go`、`service/enterprise/scheduler.go`、`model/enterprise/user_department.go`、`service/enterprise/department_membership.go`。
- 已确认当前仓库尚未实现 `model/enterprise/alert_event.go`、`service/enterprise/alert.go`、`controller/enterprise/alert.go`、`enterprise_alert_events` 迁移和任何 `alerts/events` 路由；5.1 的主要实现工作仍待开发。
- 已按 checklist 自检：故事包含明确 AC 映射、文件落点、跨库/JSON/隐私护栏、现有入口文件说明、前后续 story 边界、以及模型/service/controller/集成测试建议。
- 已按 `bmad-dev-story` workflow 将故事状态切换为 `in-progress`，并确认 `baseline_commit` 继续沿用 `1a2769c8644ac687d51411d61761bcafc11f51d3`。
- 本轮实现计划：先补 `AlertEvent` 模型/迁移与 service、再把 `controller/relay.go` 的敏感词阻断路径接入、最后补模型/service/API 回归测试并完成 story 勾选与 DoD 校验。
- 已完成 `AlertEvent` 模型、`enterprise_alert_events` 迁移注册、`AlertService.RecordRiskEvent` 统一入口，以及 `controller/relay.go` 敏感词拦截路径的旁路写入接线。
- 已新增 `model/enterprise/alert_event_test.go`、`service/enterprise/alert_test.go`、`controller/relay_test.go`，覆盖快照保存、摘要脱敏、旁路失败和敏感词响应不回归。
- 已验证本故事未新增管理员事件查询 API，也未修改 `controller/log.go`、`model/log.go` 或通用 usage logs 返回结构，保持 Story 5.2 之前的读写边界。
- 已运行定向与相关回归测试：`go test ./model/enterprise -run 'AlertEvent|EnterpriseTextFieldsDoNotDeclareDatabaseDefaults' -count=1`、`go test ./service/enterprise -run 'AlertService' -count=1`、`go test ./controller -run 'RelaySensitiveWords' -count=1`、`go test ./model/enterprise ./service/enterprise ./controller ./tests/api -run 'Alert|RelaySensitiveWords|Enterprise' -count=1`。
- 已尝试 `go test ./... -count=1`；当前工作区存在与本 story 无关的既有失败：根包缺少 `web/classic/dist` 嵌入资源，且 `relay/channel/claude`、`relay/helper`、`service/enterprise` 中有现存失败/受限环境失败，未由本次改动引入。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 已明确 Story 5.1 的最小必须入口是 `controller/relay.go` 中的敏感词阻断路径，不能只建表不接线。
- 已明确风险事件是新的 enterprise 事实源，不能复用 `enterprise_admin_actions`，也不能并入 `model/log.go` / `controller/log.go`。
- 已明确部门成员关系必须在事件产生时快照保存，不能在查询阶段回查当前部门关系。
- 已明确隐私边界：仅存追溯摘要，不存原始 prompt、完整命中内容或会扩大普通日志敏感可见范围的字段。
- 已明确 Story 5.1 不应提前实现完整筛选台、规则 DSL、投递状态表或完整 `enterprise-alerts` UI，这些由 5.2+ 继续完成。
- 已完成 `enterprise_alert_events` 事实源建模与迁移注册，文本字段默认值继续遵循 enterprise 现有 `TEXT` 归一化模式。
- 已完成统一 `AlertService.RecordRiskEvent` 入口，保存请求时部门快照、请求 ID、用户、模型、风险类型、处理结果和脱敏摘要；未归属用户落空快照。
- 已完成敏感词阻断路径接线：先命中现有敏感词检查，再旁路写风险事件，最后返回既有 `sensitive_words_detected` 错误。
- 已收紧普通日志暴露面：敏感词 warning 不再打印命中词原文，仅记录命中数量，避免扩大敏感信息可见范围。
- 已完成模型、service、controller 定向回归测试，并通过 enterprise/API 相关回归。
- `go test ./... -count=1` 未能全绿，原因是工作区既有根包静态资源缺失以及其他非本 story 测试失败/环境受限失败；与本故事直接相关的测试已通过。

### File List

- `_bmad-output/implementation-artifacts/5-1-record-content-risk-events.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/relay.go`
- `controller/relay_test.go`
- `model/enterprise/alert_event.go`
- `model/enterprise/alert_event_test.go`
- `model/enterprise/migration.go`
- `service/enterprise/alert.go`
- `service/enterprise/alert_test.go`

## Change Log

- 2026-05-31: 新增 `enterprise_alert_events` 模型与迁移注册，使用 JSON wrapper + `TEXT` 存储部门快照，建立 Epic 5 风险事件单一事实源。
- 2026-05-31: 新增 `service/enterprise/alert.go`，统一旁路记录风险事件，保存请求时部门快照并对摘要/日志做最小化脱敏处理。
- 2026-05-31: 在 `controller/relay.go` 的敏感词阻断路径接入风险事件记录，并补充模型、service、controller 回归测试。
- 2026-05-31: Senior Developer Review (AI) 自动修复 relay 租户归属丢失、敏感词事件摘要可被调用方文本污染、以及缺失的租户作用域回归测试；故事状态更新为 done。

## Senior Developer Review (AI)

Reviewer: GPT-5 Codex
Date: 2026-05-31
Outcome: Approved after automatic fixes.

### Findings and Fixes

- [HIGH] `controller/relay.go` 在敏感词命中路径中把 `TenantId` 固定写成 `0`，导致非默认租户的风险事件会落到错误租户，AC2 要求的“请求发生时部门成员关系快照”也会随之失真。已修复为从 relay 请求中的 `tenant_id` 保留显式租户作用域，并补充租户作用域回归测试。
- [HIGH] `service/enterprise/alert.go` 原先信任调用方传入的 `Summary`，只在命中词级别做替换；未来其他过滤入口若把原始 prompt 或片段塞进 `Summary`，就会把敏感内容带入 `enterprise_alert_events`，违反 AC1/AC3 的最小暴露要求。已修复为对敏感词风险统一归一化为计数型摘要，不再持久化调用方原文。
- [MEDIUM] 风险事件 `RiskType` / `ActionResult` 未做空值归一化，未来复用入口若漏传字段会写入空字符串，削弱事件表作为单一事实源的可筛选性。已新增 `unknown` 兜底归一化。
- [MEDIUM] 敏感词摘要长度此前没有上限，后续非敏感词风险类型若传入超长摘要会把表当作自由文本存储。已增加摘要长度裁剪，保持“追溯所需最小上下文”的边界。
- [MEDIUM] 原测试只覆盖 tenant `0` 成功路径，无法发现“所有租户都被写成 0”这一实现偏差。已在 `service/enterprise/alert_test.go` 与 `controller/relay_test.go` 新增非零租户快照断言。

### Validation

- Passed: `env GOCACHE=/private/tmp/go-cache go test ./model/enterprise -run 'AlertEvent|EnterpriseTextFieldsDoNotDeclareDatabaseDefaults' -count=1`
- Passed: `env GOCACHE=/private/tmp/go-cache go test ./service/enterprise -run 'AlertService' -count=1`
- Passed: `env GOCACHE=/private/tmp/go-cache go test ./controller -run 'RelaySensitiveWords' -count=1`
- Passed: `git diff --check -- controller/relay.go controller/relay_test.go service/enterprise/alert.go service/enterprise/alert_test.go model/enterprise/alert_event.go model/enterprise/alert_event_test.go model/enterprise/migration.go`

### Review Notes

- Story与实现文件清单已交叉检查；应用源码层面的改动集中在 `controller/relay.go`、`service/enterprise/alert.go`、`model/enterprise/alert_event.go` 及对应测试。
- 当前工作区还有 `.agents/skills/bmad-story-automator/**/__pycache__`、`_bmad-output/story-automator/**` 等非应用源码变更，按 workflow 规则未纳入应用实现审查，也未修改。
- 无 CRITICAL 问题残留；故事状态已更新为 `done`，可作为 Epic 5 后续查询/告警故事的数据事实源继续推进。
