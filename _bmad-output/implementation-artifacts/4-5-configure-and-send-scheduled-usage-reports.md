---
baseline_commit: 89e5e10e2e674994368bee0b3e128fce9f2b8649
---

# Story 4.5: 配置和发送定期用量报告

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 配置部门用量报告接收人、频率和时间范围,
so that 管理层可以定期收到 AI 用量摘要。

## Acceptance Criteria

1. **Given** 管理员打开定期报告配置  
   **When** 设置接收人、频率和时间范围  
   **Then** 系统保存配置并校验邮箱格式  
   **And** 记录低频管理操作审计。
2. **Given** 定期报告任务到达发送时间  
   **When** 系统生成报告  
   **Then** 报告至少包含部门总览、Top 部门和异常高增长部门  
   **And** 报告包含部门间不可加和说明。
3. **Given** 邮件发送失败  
   **When** 任务结束  
   **Then** 系统记录失败原因和状态  
   **And** 不影响看板聚合数据生成或后续重试。

## Tasks / Subtasks

- [x] 新增定期报告配置与任务状态模型，保持三库兼容和单一状态源 (AC: 1, 2, 3)
  - [x] 新增 `model/enterprise/usage_report_job.go`，落地 `enterprise_usage_report_jobs` 表，至少覆盖：`tenant_id`、接收人列表、频率、时间范围、状态、`last_run_at`、`next_run_at`、`last_success_at`、`error_reason`、以及渲染报告所需的稳定快照字段；迁移继续走 `model/enterprise/migration.go` + `model/main.go` 既有模式，不引入数据库专属 DDL。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; _bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure; model/main.go]
  - [x] 接收人列表、时间范围和报告摘要快照如需存 JSON，业务序列化必须使用 `common.Marshal` / `common.UnmarshalJsonStr`，不要直接调用 `encoding/json`。[Source: AGENTS.md#rule-1-json-package--use-commonjsongo; model/enterprise/usage_snapshot.go]
  - [x] 模型字段设计要支持“单条配置 + 多次执行”的状态追踪，而不是每次发送都新建一套平行表。报告配置与发送状态必须能作为 `usage_report_task` 的正确性来源，便于失败重试与后台排障。[Source: _bmad-output/planning-artifacts/architecture.md#Async Work; _bmad-output/planning-artifacts/architecture.md#Background Tasks]
  - [x] 频率与时间范围只实现 Epic/FR 需要的最小集合，例如 `daily/weekly/monthly` + 对应报告窗口，不提前扩展成通用 cron 编辑器；Story 4.5 的目标是可配置、可发送、可追踪，不是做一个完整调度平台。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.5: 配置和发送定期用量报告; _bmad-output/planning-artifacts/architecture.md#Async Work]

- [x] 实现报告配置 API，并复用既有企业审计与权限边界 (AC: 1)
  - [x] 在 `dto/enterprise/usage.go` 中新增报告配置请求/响应 DTO，例如 `DepartmentUsageReportConfigRequest`、`DepartmentUsageReportConfigResponse`、`DepartmentUsageReportJobItem`；可选标量继续使用指针类型并带 `omitempty`，避免显式 `0` / `false` 被吞掉。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; dto/enterprise/usage.go]
  - [x] 在 `controller/enterprise/usage.go` 增加报告配置读写和状态查询入口，建议落在 `/api/enterprise/usage/reports` 资源下；controller 只做参数绑定、邮箱格式校验、权限校验、调用 service 和 HTTP 响应映射，不在 controller 内写调度判定、HTML 模板拼装或聚合 SQL。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; controller/enterprise/usage.go]
  - [x] 在 `router/enterprise-router.go` 注册报告配置与任务状态相关路由，继续沿用 `middleware.EnterpriseAdmin()`；Story 4.5 不自行引入“经营负责人可编辑报告配置”的新权限模型，避免与 4.2 的只读看板权限口径混淆。[Source: router/enterprise-router.go; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
  - [x] 成功保存配置后写 `enterprise_admin_actions` 低频管理审计，新增专用 `action_type` / `object_type` 常量并复用 `service/enterprise/admin_action.go` 的敏感字段净化；接收人邮箱、报告参数可以审计，但不得把 SMTP 凭据、邮件正文全文或潜在敏感配置写入 payload。[Source: _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md; service/enterprise/admin_action.go; _bmad-output/implementation-artifacts/3-1-create-department-budget-pool.md]

- [x] 实现 `usage_report_task`，复用现有聚合与邮件能力生成定期报告 (AC: 2, 3)
  - [x] 新增 `service/enterprise/usage_report_task.go`，由 `service/enterprise/scheduler.go` 在 `common.IsMasterNode` 下统一调度，不单独开新 scheduler，不在 HTTP 请求内同步等待发送完成。[Source: _bmad-output/planning-artifacts/architecture.md#Background Tasks; service/enterprise/scheduler.go]
  - [x] 报告数据必须复用 Story 4.1 的 `enterprise_usage_snapshots` 聚合结果与 Story 4.4 已建立的 summary/export 排序语义，不允许回退为默认扫描 `logs` 明细，更不要在报告链路复制第三套聚合口径。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md; service/enterprise/usage_aggregation.go; service/enterprise/usage_export.go]
  - [x] “部门总览”至少覆盖请求数、prompt/completion tokens、quota、用户数和时间窗口；“Top 部门”应基于明确的排序规则生成，优先沿用 summary 排序能力；“异常高增长部门”必须给出稳定、可测试的增长判定规则，例如当前窗口相对上一同长度窗口的请求数或 quota 增长率阈值，且在无上期数据时定义明确降级行为。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.5: 配置和发送定期用量报告; service/enterprise/usage_aggregation.go]
  - [x] 报告正文必须包含与看板/CSV 一致的“部门间不可加和”说明，口径对齐 `enterprise.usage.multi_dept_disclaimer`；不要在邮件里发明另一版文案或省略该说明。[Source: _bmad-output/planning-artifacts/architecture.md#UI Annotation Requirements; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md]
  - [x] 邮件发送复用 `common.SendEmail` 和现有 SMTP 配置，不新增新的邮件 provider 抽象；支持多接收人时沿用现有 `receiver` 以分号分隔的约定，避免引入不兼容的地址拼接规则。[Source: _bmad-output/planning-artifacts/architecture.md#External Integrations; common/email.go]
  - [x] 发送失败时仅更新报告 job 的 `status`、`error_reason`、重试计数/下次发送时间等字段，并记录摘要日志；失败不得阻塞 `usage_aggregation_task`、看板查询、CSV 导出或其他企业后台任务。[Source: _bmad-output/planning-artifacts/architecture.md#Observability; _bmad-output/planning-artifacts/architecture.md#Data Flow]

- [x] 在 Default `enterprise-usage` feature 中新增报告配置 UI，并保持现有总览/详情/导出行为不回归 (AC: 1)
  - [x] 继续在 `web/default/src/features/enterprise-usage/` 内实现，不新建平行的“reports”顶级 feature；新增配置表单、任务状态展示或最近发送记录时，应遵守现有 `api.ts` + query hooks + TanStack Router search 状态模式。[Source: _bmad-output/planning-artifacts/architecture.md#Frontend Architecture; web/default/src/features/enterprise-usage/api.ts; web/default/src/features/enterprise-usage/index.tsx]
  - [x] 新文案进入 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，英文 key 作为 source string；不要把报告专用文案硬编码在组件里。[Source: AGENTS.md#internationalization-i18n; _bmad-output/planning-artifacts/architecture.md#i18n]
  - [x] 报告配置页/面板至少允许设置接收人邮箱、频率和时间范围，并展示最近一次发送状态/错误摘要，便于管理员确认是否生效；不要把 SMTP 系统设置复制进该页面，也不要在 4.5 扩展 Classic 完整等价 UI，除非实现只需极小读写入口。[Source: _bmad-output/planning-artifacts/architecture.md#Default Theme; _bmad-output/planning-artifacts/architecture.md#Classic Theme]
  - [x] 现有 `enterprise-usage` 的总览筛选、详情 drill-down、导出按钮和 `summary_sort` / `summary_order` 语义必须保持可用。4.5 新增的报告配置状态不能污染 4.3 的 detail-only search state，也不能改写 4.4 的导出参数契约。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md; web/default/src/features/enterprise-usage/types.ts]

- [x] 补齐测试与回归，锁定报告配置、发送任务和失败状态行为 (AC: 1, 2, 3)
  - [x] 新增 `service/enterprise/usage_report_task_test.go`，覆盖：到点任务挑选、时间窗口解析、空数据/无上期数据的增长判断、多接收人拼接、成功状态落库、失败 `error_reason` 写入、以及任务失败不影响聚合任务继续运行。[Source: _bmad-output/planning-artifacts/architecture.md#Test Organization; service/enterprise/scheduler.go]
  - [x] 为 `model/enterprise/usage_report_job.go` 增加迁移/序列化测试，确保三库兼容字段默认值、JSON 空数组语义与状态字段更新正确。[Source: model/main.go; model/enterprise/usage_snapshot.go]
  - [x] 扩展 `controller/enterprise/usage_test.go`，覆盖报告配置参数校验、邮箱格式校验、企业管理员权限、审计写入、配置读取与状态查询接口。[Source: controller/enterprise/usage_test.go; service/enterprise/admin_action.go]
  - [x] 扩展 `tests/api/enterprise_usage_test.go`，覆盖真实路由上的报告配置保存/读取与一次任务执行后的状态变化；如果任务执行通过可注入时钟或 service 入口触发，测试应避免依赖真实 ticker 睡眠。[Source: _bmad-output/planning-artifacts/architecture.md#Test Organization]
  - [x] 扩展 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`，至少覆盖：报告配置入口可见、表单校验、保存后请求参数正确、错误状态可见、以及原有 disclaimer / 导出按钮 / 详情交互仍可用。[Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx; web/default/src/features/enterprise-usage/index.tsx]

## Dev Notes

- Story 4.5 应该扩展 Epic 4 现有“聚合 -> 总览/详情 -> 导出”链路，而不是新建一个与 `enterprise_usage_snapshots` 平行的报表系统。报告任务消费的是已经稳定下来的 summary 数据和部门元数据，不能重新定义第四套统计口径。[Source: _bmad-output/planning-artifacts/epics.md#Epic 4: 组织用量看板、详情与报告导出; _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; service/enterprise/usage_aggregation.go]
- 架构文档对 FR-16 已经给出权威文件归属：`model/enterprise/usage_report_job.go`、`service/enterprise/usage_report_task.go`、`controller/enterprise/usage.go`、`/api/enterprise/usage/reports`、Default `enterprise-usage/`。开发时不要把报告任务塞进 `alert_dispatch.go`、系统设置模块或新的顶层 feature。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- `service/enterprise/scheduler.go` 当前只启动 wallet 和 usage aggregation 任务，尚未纳入 `usage_report_task`。4.5 需要在同一入口补齐任务注册，但必须保持 `common.IsMasterNode` 单点守卫与 `sync.Once` 行为，避免多节点重复发送邮件。[Source: service/enterprise/scheduler.go; _bmad-output/planning-artifacts/architecture.md#Background Tasks]
- 报告配置属于低频管理动作，应写 `enterprise_admin_actions`；但报告发送执行状态本身应以 `enterprise_usage_report_jobs` 为正确性来源，避免把“配置审计”和“运行状态”混成同一表，重蹈 Epic 3 之前状态模型分裂的问题。[Source: _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md; _bmad-output/implementation-artifacts/epic-3-retro-2026-05-29.md]
- 邮件正文只需满足管理摘要和可追踪，不要在 4.5 提前扩成附件下载中心、异步文件归档或富文本模板平台。若后续需要 HTML 模板文件，可放 `service/enterprise/templates/` 并通过 Go embed 管理，不应放到 `web/` 目录。[Source: _bmad-output/planning-artifacts/architecture.md#File Organization Patterns]
- “异常高增长部门”是本故事最容易模糊实现的部分。必须在 service 层定义统一算法和阈值，并让测试锁定；不要在前端临时计算，也不要依赖肉眼排序近似判断，否则报告内容会在重跑时漂移且难以验收。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.5: 配置和发送定期用量报告]
- 报告任务失败必须是旁路失败：写状态、留错误、允许后续重试，但不能影响 `usage_aggregation_task` 的进度水位、看板接口可用性或 4.4 CSV 导出契约。这是本故事最大的回归边界之一。[Source: _bmad-output/planning-artifacts/architecture.md#Observability; _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md]
- 4.4 已经显式化了 `summary_sort` / `summary_order`，并修复了页面与导出排序漂移问题。4.5 的报告“Top 部门”如按请求数或 quota 排序，应直接复用同一排序定义，避免邮件里的 Top 排行和页面/CSV 各说各话。[Source: _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md; service/enterprise/usage_export.go]

### Project Structure Notes

- 预计新增文件：
  - `model/enterprise/usage_report_job.go`
  - `service/enterprise/usage_report_task.go`
  - `service/enterprise/usage_report_task_test.go`
- 预计修改文件：
  - `model/enterprise/migration.go`
  - `model/main.go`
  - `service/enterprise/scheduler.go`
  - `service/enterprise/admin_action.go`
  - `controller/enterprise/usage.go`
  - `controller/enterprise/usage_test.go`
  - `dto/enterprise/usage.go`
  - `router/enterprise-router.go`
  - `tests/api/enterprise_usage_test.go`
  - `web/default/src/features/enterprise-usage/api.ts`
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-usage/types.ts`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `web/default/src/i18n/locales/en.json`
  - `web/default/src/i18n/locales/fr.json`
  - `web/default/src/i18n/locales/ja.json`
  - `web/default/src/i18n/locales/ru.json`
  - `web/default/src/i18n/locales/vi.json`
  - `web/default/src/i18n/locales/zh.json`
- 保持不改或仅只读依赖：
  - `service/enterprise/usage_aggregation.go` 的聚合口径
  - `service/enterprise/usage_aggregation_task.go` 的水位推进语义
  - `service/enterprise/usage_export.go` 的 CSV 契约
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 4.5: 配置和发送定期用量报告]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 4: 组织用量看板、详情与报告导出]
- [Source: _bmad-output/planning-artifacts/architecture.md#Async Work]
- [Source: _bmad-output/planning-artifacts/architecture.md#Background Tasks]
- [Source: _bmad-output/planning-artifacts/architecture.md#Observability]
- [Source: _bmad-output/planning-artifacts/architecture.md#UI Annotation Requirements]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/planning-artifacts/architecture.md#Complete Project Directory Structure]
- [Source: _bmad-output/planning-artifacts/architecture.md#Test Organization]
- [Source: _bmad-output/planning-artifacts/architecture.md#File Organization Patterns]
- [Source: _bmad-output/planning-artifacts/architecture.md#External Integrations]
- [Source: _bmad-output/implementation-artifacts/1-4-record-enterprise-organization-audit.md]
- [Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md]
- [Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- [Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- [Source: _bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md]
- [Source: _bmad-output/implementation-artifacts/epic-3-retro-2026-05-29.md]
- [Source: AGENTS.md#rule-1-json-package--use-commonjsongo]
- [Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]
- [Source: common/email.go]
- [Source: controller/enterprise/usage.go]
- [Source: controller/enterprise/usage_test.go]
- [Source: dto/enterprise/usage.go]
- [Source: model/enterprise/usage_snapshot.go]
- [Source: model/main.go]
- [Source: router/enterprise-router.go]
- [Source: service/enterprise/admin_action.go]
- [Source: service/enterprise/scheduler.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: service/enterprise/usage_aggregation_task.go]
- [Source: service/enterprise/usage_export.go]
- [Source: web/default/src/features/enterprise-usage/api.ts]
- [Source: web/default/src/features/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-usage/types.ts]
- [Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工执行 `bmad-create-story` workflow，因为 `_bmad/scripts/resolve_customization.py` 依赖 Python 3.11+ 的 `tomllib`，当前环境按 skill fallback 规则手工解析 `customize.toml` 与 `_bmad/bmm/config.yaml`。
- 已读取 Epic 4、architecture、Story 4.4、`service/enterprise/scheduler.go`、`controller/enterprise/usage.go`、`dto/enterprise/usage.go`、`service/enterprise/usage_aggregation.go`、`service/enterprise/usage_export.go`、`common/email.go`、`router/enterprise-router.go` 与 Default `enterprise-usage` 前端文件，确认当前仓库尚未实现 `usage_report_job` / `usage_report_task`。
- 已检查近 5 个提交，Epic 4 的已有实现顺序为 4.1 聚合、4.2 总览、4.3 详情、4.4 导出；Story 4.5 需要建立在这些已完成能力之上，不能回退为新的全量统计路径。
- 已按 checklist 自检：故事包含明确 AC 映射、具体文件落点、复用约束、失败状态与测试要求，并标出最容易出错的增长判定、任务调度和审计边界。
- 已实现 `enterprise_usage_report_jobs` 模型、`usage_report_task` 调度与 `/api/enterprise/usage/reports` 读写接口，并保持 JSON 读写全部走 `common` 包包装器。
- 已为用量报告补充 backend i18n key、controller/API 测试、service/model 单测，以及 Default `enterprise-usage` 报告配置卡片与多语言文案。
- 已验证报告任务失败只写 job 状态和错误，不影响聚合任务、总览、详情与 CSV 导出链路。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 已明确 Story 4.5 的权威落点是 `usage_report_job` + `usage_report_task` + `/api/enterprise/usage/reports` + Default `enterprise-usage`，而不是新建平行系统。
- 已明确报告任务必须复用 `enterprise_usage_snapshots` 聚合和既有 summary/export 排序语义，禁止回退扫描 `logs`。
- 已明确配置审计与运行状态要分离：低频保存动作写 `enterprise_admin_actions`，发送状态以 `enterprise_usage_report_jobs` 为正确性源。
- 已明确邮件正文必须包含与看板/CSV 一致的“部门间不可加和”口径，防止三条输出链路分叉。
- 已完成 Story 4.5 的后端模型、任务服务、企业管理员配置 API、Default 前端配置面板和相关回归测试。
- 已通过 `go test`、`bun test src/features/enterprise-usage/enterprise-usage.test.tsx` 和 `bun run typecheck` 验证本故事实现。
- Senior Developer Review（AI）已修复已启用报告配置在编辑时重算 `next_run_at` 导致临近待发送任务被顺延的问题，以及报告配置表单在同作用域 refetch 时覆盖未保存输入的问题，并补充对应前后端回归测试。
- Senior Developer Review（AI）后续复核已补齐 Story 4.5 测试声明与实际覆盖之间的缺口：修复 `usage_report_task_test.go` 对 `alert.go` 辅助函数的跨文件耦合，并新增时间窗口、到点任务筛选、失败不阻塞聚合、状态字段更新时间以及 `/usage/reports` 中间件权限回归测试。

### File List

- `_bmad-output/implementation-artifacts/4-5-configure-and-send-scheduled-usage-reports.md`
- `controller/enterprise/department_membership_test.go`
- `controller/enterprise/usage.go`
- `controller/enterprise/usage_test.go`
- `dto/enterprise/usage.go`
- `i18n/keys.go`
- `i18n/locales/en.yaml`
- `i18n/locales/zh-CN.yaml`
- `i18n/locales/zh-TW.yaml`
- `model/enterprise/migration.go`
- `model/enterprise/usage_report_job.go`
- `model/enterprise/usage_report_job_test.go`
- `router/enterprise-router.go`
- `service/enterprise/admin_action.go`
- `service/enterprise/errors.go`
- `service/enterprise/scheduler.go`
- `service/enterprise/usage_report_task.go`
- `service/enterprise/usage_report_task_test.go`
- `tests/api/enterprise_usage_test.go`
- `web/default/src/features/enterprise-usage/api.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

## Change Log

- 2026-05-29: 新增 `enterprise_usage_report_jobs` 模型、`usage_report_task` 调度、`/api/enterprise/usage/reports` 配置接口，以及 Default `enterprise-usage` 报告配置卡片与回归测试。
- 2026-05-29: AI review 自动修复已启用报告配置在仅编辑接收人/时间范围时仍重算 `next_run_at`、导致临近待发送任务被顺延的问题，并补充后端回归测试。
- 2026-05-29: AI review 修复报告配置表单在同租户数据 refetch 时重置脏表单、覆盖未保存输入的问题，并补充前端静态测试锁定重置契约。
- 2026-05-31: AI review 补齐 Story 4.5 测试覆盖与故事声明的一致性，新增时间窗口/到点筛选/失败旁路聚合/状态更新时间/中间件权限回归测试，并移除 `usage_report_task_test.go` 对 `alert.go` helper 的跨文件耦合。

## Senior Developer Review (AI)

### Reviewer

- GPT-5 Codex

### Review Date

- 2026-05-29 23:16:09 +0800

### Outcome

- Changes Requested -> Fixed -> Approved

### Findings Fixed During Review

1. High: `service/enterprise/usage_report_task.go` 在编辑已启用报告配置时总是按“当前时间”重算 `next_run_at`。这会让一个原本几分钟后就该发送的待执行任务，因为管理员只改了接收人或报告窗口而被整体顺延到下一周期，破坏单一状态源对“下一次发送时间”的正确表达。已改为：仅在首次启用、频率变化或原 `next_run_at` 无效时重算，否则保留既有待执行时间。
2. Medium: `service/enterprise/usage_report_task_test.go` 之前没有覆盖“已启用任务在配置编辑后仍保留临近待发送时间”的回归场景，导致上面的调度漂移很容易再次被引入。已补充针对 `NextRunAt` 保留语义的单测。
3. Medium: `web/default/src/features/enterprise-usage/index.tsx` 在每次报告配置查询成功后都会 `reset` 表单；只要保存后失效重取、或同租户发生普通 refetch，管理员尚未提交的输入就可能被服务器返回值覆盖。已将重置条件收敛为“作用域变化”或“表单仍是 pristine”两类情况。
4. Low: `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx` 之前没有锁定上述表单重置契约，前端表单状态回归只能靠人工发现。已新增纯函数级静态测试，覆盖 dirty/pristine 与 scope change 的组合行为。

### Validation

- 已复核 AC 1-3 与当前实现一致，未发现剩余阻塞项。
- 已复核报告链路继续基于 `enterprise_usage_snapshots` 聚合结果，未回退扫描 `logs` 明细。
- 已运行：
  - `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise -run 'UsageReport' -count=1`
  - `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'UsageReport|EnterpriseUsage' -count=1`
  - `cd web/default && bun test src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `cd web/default && bun run typecheck`

### Follow-up Review Date

- 2026-05-31 11:38:09 +0800

### Follow-up Outcome

- Changes Requested -> Fixed -> Approved

### Follow-up Findings Fixed During Review

1. Medium: `service/enterprise/usage_report_task_test.go` 仍依赖 `alert.go` 中的 `boolPtr` helper，造成报告测试对告警实现的跨文件耦合；一旦无关的告警 helper 改名或移动，Story 4.5 的报告测试会被连带打断。已改为在报告测试内统一使用局部 `usageReportBoolPtr`，消除隐式依赖。
2. Medium: 故事子任务宣称 `usage_report_task_test.go` 已覆盖“到点任务挑选”和“时间窗口解析”，但原测试只验证配置保存与发送结果，没有单独锁定 `usageReportWindow` / `previousUsageReportWindow` 以及 `enabled + next_run_at <= now` 的筛选语义。已新增对应回归测试。
3. Medium: 故事子任务宣称 `model/enterprise/usage_report_job.go` 的“状态字段更新正确”已有测试，但原 `usage_report_job_test.go` 仅检查 JSON wrapper 和迁移列存在性，没有证明 `status`、`error_reason`、`failure_count` 与 `UpdatedAt` 在更新路径上会正确落库。已补充更新路径测试。
4. Medium: 故事子任务宣称 `controller/enterprise/usage_test.go` 已覆盖企业管理员权限，但 controller 级测试此前直接调用 handler，绕过了 `middleware.EnterpriseAdmin()`，无法证明 `/api/enterprise/usage/reports` 的 GET/PUT 真实会拒绝非管理员。已新增中间件挂载下的权限回归测试。
5. Medium: 故事子任务宣称“任务失败不影响聚合任务继续运行”，但原测试只证明多个 report job 之间互不阻塞，没有验证报告失败后聚合任务仍能继续推进。已新增先失败 `RunDueReports`、再执行 `RunUsageAggregationTaskOnce` 的串联回归测试。

### Follow-up Validation

- 已运行：
  - `GOCACHE=$(pwd)/.tmp/go-cache GOTMPDIR=$(pwd)/.tmp/go-tmp go test ./service/enterprise -run 'UsageReport' -count=1`
  - `GOCACHE=$(pwd)/.tmp/go-cache GOTMPDIR=$(pwd)/.tmp/go-tmp go test ./model/enterprise -run 'UsageReport' -count=1`
  - `GOCACHE=$(pwd)/.tmp/go-cache GOTMPDIR=$(pwd)/.tmp/go-tmp go test ./controller/enterprise -run 'UsageReport' -count=1`
  - `GOCACHE=$(pwd)/.tmp/go-cache GOTMPDIR=$(pwd)/.tmp/go-tmp go test ./tests/api -run 'EnterpriseUsageReport' -count=1`
