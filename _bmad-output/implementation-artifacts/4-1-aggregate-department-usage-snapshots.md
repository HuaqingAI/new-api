---
baseline_commit: f760d3e8690359ec00535c49880f5f57a0fc0534
---

# Story 4.1: 聚合部门用量快照

Status: done

## Story

As a 管理员,
I want 系统按部门聚合请求、token、quota 和模型用量,
so that 部门用量看板可以快速加载且口径稳定。

## Acceptance Criteria

1. **Given** 系统存在消费日志和用户部门成员关系  
   **When** 用量聚合任务执行  
   **Then** 系统按时间窗扫描日志并 join `enterprise_user_departments` 展开到部门维度  
   **And** 多部门用户同一笔消费在每个归属部门各计一次。
2. **Given** 用户没有任何部门成员关系  
   **When** 聚合任务处理该用户消费  
   **Then** 系统将消费归入 `dept_id = NULL`  
   **And** 看板显示为“未归属”，不计入任何部门风险率分母。
3. **Given** 部门用量聚合任务执行完成  
   **When** 管理员通过 `/api/enterprise/usage/department-summary` 查询同一时间范围  
   **Then** API 从 `enterprise_usage_snapshots` 或等价聚合表返回部门汇总数据  
   **And** 每行至少包含 `dept_id`、`dept_name`、`window_start`、`window_end`、`request_count`、`prompt_tokens`、`completion_tokens`、`quota`、`user_count` 和 `model_distribution`。
4. **Given** 聚合范围内存在未归属用户消费  
   **When** 查询 `/api/enterprise/usage/department-summary`  
   **Then** 响应包含 `dept_id = null`、`dept_name = "未归属"` 的行  
   **And** 数组字段缺省返回 `[]`，不返回 `null`。
5. **Given** `logs` 累计 100 万条且部门数 ≤ 100  
   **When** 查询近 30 天部门用量首屏  
   **Then** 系统 P95 响应目标 < 3 秒  
   **And** 默认查询不得依赖逐条扫描全量日志。

## Tasks / Subtasks

- [x] 建立部门用量快照模型、DTO 和迁移入口，形成可查询的聚合存储 (AC: 1, 2, 3, 4, 5)
  - [x] 新增 `model/enterprise/usage_snapshot.go`，定义 `enterprise_usage_snapshots` 的 GORM 模型，至少覆盖 `tenant_id`、`dept_id`、`window_start`、`window_end`、`request_count`、`prompt_tokens`、`completion_tokens`、`quota`、`user_count`、`model_distribution`、`created_at`、`updated_at`；时间字段统一 `int64` Unix 秒，`model_distribution` 使用 `TEXT`/字符串存 JSON，不使用 `JSONB`、`json_extract` 等数据库专属能力。[Source: _bmad-output/planning-artifacts/architecture.md#naming-patterns; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 为 `model_distribution` 提供读写辅助方法并统一走 `common.Marshal` / `common.UnmarshalJsonStr`，保证空值落库和出参都归一为 `[]`，而不是 `null` 或空字符串。[Source: AGENTS.md#rule-1-json-package--use-commonjsongo; _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照]
  - [x] 更新 `model/enterprise/migration.go`，把 `UsageSnapshot` 纳入企业 AutoMigrate；如为 SQLite/MySQL/PostgreSQL 需要补索引或默认值兼容处理，遵循现有 `model/main.go` 的三库迁移模式，不引入数据库专属 DDL。[Source: model/enterprise/migration.go; model/main.go; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
  - [x] 新增 `dto/enterprise/usage.go`，定义 summary 查询与返回结构；所有数组型字段默认输出 `[]`，可选标量继续用指针类型以避免 `0`/`false` 被吞掉。[Source: _bmad-output/planning-artifacts/architecture.md#api-naming-conventions; AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values]

- [x] 实现增量聚合服务，按成员关系有效期把消费日志展开到部门维度 (AC: 1, 2, 5)
  - [x] 新增 `service/enterprise/usage_aggregation.go`，仅从 `model.Log` / `logs` 表做 `SELECT` 读取，使用 `UserId`、`Username`、`ModelName`、`Quota`、`PromptTokens`、`CompletionTokens`、`CreatedAt` 等现有字段聚合，不新增 `logs` 字段、不改 `RecordConsumeLog` / relay 消费链路。[Source: model/log.go; _bmad-output/planning-artifacts/architecture.md#service-boundaries]
  - [x] 聚合时按 `enterprise_user_departments` 的生效关系展开用户消费，必须把 `status`、`joined_at`、`left_at` 纳入归因判断：同一条日志应归入消费发生时处于生效区间的全部部门，而不是简单按“当前仍在部门内”的关系回算历史数据。[Source: model/enterprise/user_department.go; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-13-按部门聚合用量-v11-修订]
  - [x] 对无任何匹配部门成员关系的日志，落入 `dept_id = NULL` 桶，并在聚合结果里统一输出 `dept_name = "未归属"`；该桶仅用于用量展示，不为任何部门风险率分母提供基数。[Source: _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照; _bmad-output/planning-artifacts/architecture.md#attribution-model]
  - [x] 采用“增量窗口 + 幂等 upsert”而非全量重扫：持久化一个聚合 watermark（可放在现有 Option/config 体系或等价企业配置），任务每次只处理未聚合时间段，并对同一 `(tenant_id, dept_id, window_start, window_end)` 做覆盖式更新，避免重复执行把数值累加两次。[Source: _bmad-output/planning-artifacts/architecture.md#critical-gaps阻塞实现需在-epicstory-前关闭; _bmad-output/planning-artifacts/architecture.md#performance100-万-logs--30-天-p95--3s]
  - [x] `model_distribution` 在 service 层聚合为稳定的结构切片并按模型名排序输出，避免 API 每次临时从明细字符串重算；后续 Story 4.2/4.3 直接复用该字段。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-13-按部门聚合用量-v11-修订; _bmad-output/planning-artifacts/architecture.md#组织用量看板与导出]

- [x] 接入企业后台调度器，确保用量聚合只在主节点启动且不破坏现有 wallet 任务 (AC: 1, 5)
  - [x] 新增 `service/enterprise/usage_aggregation_task.go`，封装 ticker 驱动的聚合入口，重用 `common.IsMasterNode` 守卫和现有 enterprise scheduler 风格，不在 service 内部写无限重试死循环。[Source: service/enterprise/scheduler.go; _bmad-output/planning-artifacts/architecture.md#backend-任务受-commonismasternode-守卫]
  - [x] 扩展 `service/enterprise/scheduler.go`，把当前只负责 wallet 的启动逻辑升级为包含 `usage_aggregation_task` 的统一 enterprise task 启动点，同时保留 `ExpireBalanceAllocations` / `SyncWalletStates` 的现有行为与 tick 模式。[Source: service/enterprise/scheduler.go; service/enterprise/balance_expiry_task.go; service/enterprise/wallet_state_sync_task.go]
  - [x] 更新 `main.go` 的 enterprise scheduler 启动调用，保持只在 master 节点上启动后台任务，不影响已有订阅重置、Codex 刷新和其它全局任务。[Source: main.go; _bmad-output/planning-artifacts/architecture.md#requirements--components-mapping]

- [x] 提供部门 summary 查询 API，确保首屏只命中聚合表 (AC: 3, 4, 5)
  - [x] 新增 `service/enterprise/usage_aggregation.go` 的 summary 查询能力，按请求时间范围从 `enterprise_usage_snapshots` 汇总返回部门级结果；查询层禁止回退到 `logs` 明细扫描，保证 30 天首屏不依赖逐条日志读取。[Source: _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照; _bmad-output/planning-artifacts/architecture.md#performance100-万-logs--30-天-p95--3s]
  - [x] 新增 `controller/enterprise/usage.go`，实现 `GET /api/enterprise/usage/department-summary`；controller 只负责参数解析、权限校验、调用 service 和返回 `{ success, message, data }`，不要在 controller 拼聚合 SQL 或重算 `model_distribution`。[Source: _bmad-output/planning-artifacts/architecture.md#backend-内部分层; controller/enterprise/department_budget.go]
  - [x] 在 `router/enterprise-router.go` 注册 usage summary 路由。Story 4.1 的入口面向“管理员”，默认使用 `middleware.EnterpriseAdmin()`，避免在没有部门过滤条件时向普通部门管理员暴露全组织汇总；如后续要支持部门管理员受限视图，在 Story 4.2/4.3 中单独扩展。[Source: router/enterprise-router.go; middleware/enterprise_admin.go; _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照]
  - [x] 响应中为 `dept_id = null` 的聚合行补齐 `dept_name = "未归属"`，并保证 `model_distribution`、未来可能扩展的数组字段全部返回 `[]`，不返回 `null`。[Source: _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照]

- [x] 完成测试、文档和回归，防止跨库/幂等/权限回退 (AC: 1, 2, 3, 4, 5)
  - [x] 新增 `service/enterprise/usage_aggregation_test.go`，覆盖：多部门用户单笔消费重复计入、未归属桶、成员关系时间窗口生效、同一窗口重复执行幂等、不读写 `logs` 以外的 relay 数据结构、`model_distribution` 空值输出 `[]`。[Source: _bmad-output/planning-artifacts/architecture.md#test-organization]
  - [x] 新增 `controller/enterprise/usage_test.go` 或等价 controller 测试，覆盖参数校验、管理员权限、`dept_id = null` 输出、数组字段非 `null`、非法时间范围错误映射。[Source: controller/enterprise/dingtalk_config_test.go; middleware/enterprise_admin.go]
  - [x] 视现有测试组织补充 `tests/api/enterprise_usage_test.go` 或等价 API 层回归，验证近 30 天 summary 调用命中聚合数据而不是明细日志路径。[Source: _bmad-output/planning-artifacts/architecture.md#test-organization]
  - [x] 新增 `docs/enterprise/usage-aggregation.md`，把本故事实际选择的默认聚合窗口、watermark 存储方式、回填策略和“未归属”口径写清楚，避免 Story 4.2/4.3/4.4/4.5 各自再发明一套规则。[Source: _bmad-output/planning-artifacts/architecture.md#critical-gaps阻塞实现需在-epicstory-前关闭]

## Dev Notes

- Epic 4 的后续故事 4.2/4.3/4.4/4.5 都建立在 Story 4.1 的聚合口径之上，因此 4.1 不能只做“后台定时任务”，必须一并落地 `enterprise_usage_snapshots` 聚合表、summary API、调度接入和稳定的 `model_distribution` 返回格式，否则后续看板/详情/CSV/定时报表都会重复造聚合逻辑。[Source: _bmad-output/planning-artifacts/epics.md#epic-4-组织用量看板详情与报告导出; _bmad-output/planning-artifacts/architecture.md#requirements--components-mapping]
- 当前仓库还没有任何 usage 相关 enterprise 文件：`controller/enterprise/`、`service/enterprise/`、`model/enterprise/`、`dto/enterprise/` 中都缺少 `usage*.go`。Story 4.1 是该能力域的起点，文件归属要严格落在 architecture 规定的 usage slice 中，不要把统计逻辑散落到 `controller/` 根目录或现有 budget 文件里。[Source: _bmad-output/planning-artifacts/architecture.md#project-organization; controller/enterprise; service/enterprise; model/enterprise; dto/enterprise]
- `service/enterprise/scheduler.go` 当前只启动 wallet 相关任务，`main.go` 也只调用 `StartEnterpriseWalletTasks()`。本故事需要把 scheduler 扩展为统一企业任务入口，但必须保留现有钱包过期和状态联动行为，不能因为接入 usage 聚合而中断 Epic 3 的定时任务。[Source: service/enterprise/scheduler.go; main.go]
- `model/log.go` 已提供聚合所需的关键字段，且 architecture 明令企业模块只读 `logs`。请直接复用 `model.Log` 和现有日志表，不要给 `logs` 表新增 `department_id`、`allocation_id` 或其它回填字段，也不要改 `RecordConsumeLog` / `LogConsume` 语义。[Source: model/log.go; _bmad-output/planning-artifacts/architecture.md#service-boundaries]
- `enterprise_user_departments` 不是纯当前态映射，它有 `status`、`joined_at`、`left_at`。如果聚合只按“当前激活成员关系”回算历史日志，用户调部门后历史统计会漂移，因此归因必须考虑消费发生时的关系有效期；这是 Story 4.1 最容易被实现忽略的正确性约束。[Source: model/enterprise/user_department.go]
- 企业模块已有的一致性模式应继续沿用：Router → Controller → Service → Model 单向分层；service 返回 sentinel error，controller 做 i18n 映射；GORM 为主，避免裸 SQL；确需原生 SQL 时必须同时给出 SQLite/MySQL/PostgreSQL 的兼容路径。[Source: _bmad-output/planning-artifacts/architecture.md#backend-内部分层; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96; service/enterprise/errors.go]
- `model_distribution` 和任何聚合快照的 JSON 列都必须按项目规则走 `common/json.go` 包装函数，不要在 business code 直接 `encoding/json.Marshal` / `Unmarshal`。同样，列类型保持 `TEXT`，不能为了省事引入 PostgreSQL 专属 `JSONB` 操作符。[Source: AGENTS.md#rule-1-json-package--use-commonjsongo; AGENTS.md#rule-2-database-compatibility--sqlite-mysql--578-postgresql--96]
- 近 30 天首屏 P95 < 3 秒的约束意味着：summary API 的默认路径只能查 `enterprise_usage_snapshots`，不能在 controller/service 中写“若快照缺失则回退扫 logs”的兜底逻辑。快照缺失应通过后台任务或显式回填机制修复，而不是把性能债转嫁给在线查询。[Source: _bmad-output/planning-artifacts/epics.md#story-41-聚合部门用量快照; _bmad-output/planning-artifacts/architecture.md#performance100-万-logs--30-天-p95--3s]
- Recent implementation pattern from Epic 3: 企业能力优先以同目录 `*_test.go` 做 service/controller 回归，controller 保持薄层，权限中间件挂在 `router/enterprise-router.go`，管理类变更通常补 `enterprise_admin_actions` 审计。Story 4.1 的聚合任务本身不需要写低频管理审计，但 API 和 scheduler wiring 要保持同样的项目结构与测试粒度。[Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md; controller/enterprise/dingtalk_config_test.go; service/enterprise/permission.go]

### Project Structure Notes

- 预计新增文件：
  - `model/enterprise/usage_snapshot.go`
  - `dto/enterprise/usage.go`
  - `service/enterprise/usage_aggregation.go`
  - `service/enterprise/usage_aggregation_task.go`
  - `controller/enterprise/usage.go`
  - `service/enterprise/usage_aggregation_test.go`
  - `controller/enterprise/usage_test.go`
  - `tests/api/enterprise_usage_test.go` 或等价 API 回归
  - `docs/enterprise/usage-aggregation.md`
- 预计修改文件：
  - `model/enterprise/migration.go`
  - `service/enterprise/scheduler.go`
  - `router/enterprise-router.go`
  - `main.go`
- 当前代码现状与架构的差异：
  - 架构规划了 `usage_snapshot.go`、`usage_aggregation.go`、`usage_aggregation_task.go`、`controller/enterprise/usage.go`，但仓库当前尚未落地这些文件。
  - 当前 enterprise scheduler 仅覆盖 wallet 任务，尚未成为 architecture 里描述的统一企业任务入口。
  - 当前 enterprise 路由没有 `/api/enterprise/usage/**` 任何端点，因此 Story 4.1 必须先完成 summary API 的最小闭环。

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 4: 组织用量看板、详情与报告导出]
- [Source: _bmad-output/planning-artifacts/epics.md#Story 4.1: 聚合部门用量快照]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-13: 按部门聚合用量 (V1.1 修订)]
- [Source: _bmad-output/planning-artifacts/architecture.md#Attribution Model]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/planning-artifacts/architecture.md#Service Boundaries]
- [Source: _bmad-output/planning-artifacts/architecture.md#Test Organization]
- [Source: _bmad-output/planning-artifacts/architecture.md#Critical Gaps（阻塞实现，需在 epic/story 前关闭）]
- [Source: model/log.go]
- [Source: model/enterprise/user_department.go]
- [Source: model/enterprise/migration.go]
- [Source: service/enterprise/scheduler.go]
- [Source: router/enterprise-router.go]
- [Source: main.go]
- [Source: middleware/enterprise_admin.go]
- [Source: service/enterprise/errors.go]
- [Source: _bmad-output/implementation-artifacts/3-5-display-budget-pool-and-wallet-state.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 已按要求读取 `.agents/skills/bmad-dev-story/SKILL.md`、`checklist.md`、`_bmad/bmm/config.yaml`、完整 story 文件与 `_bmad-output/implementation-artifacts/sprint-status.yaml`。
- `_bmad/scripts/resolve_customization.py` 因当前 `python3` 缺少 `tomllib` 无法执行，已按 skill fallback 手工读取 `.agents/skills/bmad-dev-story/customize.toml`，确认 prepend/append 为空，persistent fact 仅为可选 `project-context.md`。
- 已检查现有 enterprise 分层与接线点：`model/log.go`、`model/enterprise/user_department.go`、`model/enterprise/migration.go`、`service/enterprise/scheduler.go`、`router/enterprise-router.go`、`main.go`、权限中间件与现有 controller/service/test 模式。
- 已读取并复用工作区中已有的 usage 红测草稿（`model/enterprise/usage_snapshot_test.go`、`service/enterprise/usage_aggregation_test.go`、`controller/enterprise/usage_test.go`、`tests/api/enterprise_usage_test.go`），在其约束下补齐实现。
- 定向测试通过：`GOCACHE=$(pwd)/.cache/go-build go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage'`。
- 范围回归结果：`GOCACHE=$(pwd)/.cache/go-build go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api` 仅因现有 `service/enterprise/dingtalk_connectivity_test.go` 需要本地监听端口而在当前沙箱失败；usage 相关用例均通过。
- 项目级空编译 `go test ./... -run '^$'` 受仓库当前缺少 `web/classic/dist` embed 产物影响失败，与本故事代码无直接关系。

### Completion Notes List

- 已新增 `enterprise_usage_snapshots` 聚合表模型与 JSON/TEXT 归一化辅助，`model_distribution` 落库与读取统一归一为 `[]`。
- 已新增 `dto/enterprise/usage.go`、`service/enterprise/usage_aggregation.go`、`service/enterprise/usage_aggregation_task.go`、`controller/enterprise/usage.go`，完成快照聚合、watermark 幂等更新、summary 查询和管理员 API 闭环。
- 聚合服务仅从 `logs` 读取消费日志，按成员关系生效区间归因；多部门用户单笔消费会在每个有效部门各计一次，无归属成员关系时落到 `dept_id = NULL` / `未归属` 桶。
- summary API 只读 `enterprise_usage_snapshots`，不回退扫描 `logs`；`model_distribution` 在 service 层聚合并按模型名排序返回。
- enterprise scheduler 已扩展为统一任务入口，保留原 wallet 任务行为，并在 master 节点下附加 usage aggregation task。
- 已补充 `docs/enterprise/usage-aggregation.md`，明确 1 小时窗口、watermark 存储键、回填方式和“未归属”口径。
- 验证说明：
  - usage 定向测试全部通过。
  - enterprise 范围全量测试受现有 DingTalk connectivity 测试监听端口限制影响，当前沙箱无法完整跑通。
  - 根模块空编译受现有前端 embed 产物缺失影响，当前工作区无法完成 `go test ./... -run '^$'`。

### File List

- `_bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `controller/enterprise/usage.go`
- `controller/enterprise/usage_test.go`
- `docs/enterprise/usage-aggregation.md`
- `dto/enterprise/usage.go`
- `main.go`
- `model/enterprise/migration.go`
- `model/enterprise/usage_snapshot.go`
- `model/enterprise/usage_snapshot_test.go`
- `router/enterprise-router.go`
- `service/enterprise/scheduler.go`
- `service/enterprise/usage_aggregation.go`
- `service/enterprise/usage_aggregation_task.go`
- `service/enterprise/usage_aggregation_test.go`
- `tests/api/enterprise_usage_test.go`

## Change Log

- 2026-05-29: 新增部门用量快照模型、增量聚合服务、后台聚合任务、管理员 summary API、路由与文档；补充 usage 相关 model/service/controller/API 测试，并将 story 状态更新为 review。
- 2026-05-29: Senior Developer Review (AI) 自动修复聚合 watermark 回填起点、整窗重算陈旧桶清理、跨窗口 `user_count` 去重统计，并补充对应回归测试；故事状态更新为 done。

## Senior Developer Review (AI)

### 结论

- Outcome: Approve
- 审查范围：`controller/enterprise/usage.go`、`model/enterprise/usage_snapshot.go`、`service/enterprise/usage_aggregation*.go`、`service/enterprise/scheduler.go`、`tests/api/enterprise_usage_test.go` 及对应测试/文档。
- Git vs Story 差异：未发现会阻塞本故事验收的源码遗漏；story `File List` 与实际 usage 代码改动基本一致。

### 已验证并修复的问题

1. `service/enterprise/usage_aggregation_task.go`
   首次运行且 watermark 为空时，任务原本只从“最近一个完整小时”开始处理，历史更早的消费窗口会被永久跳过，违反“按窗口顺序追赶历史缺口”的回填要求。已修复为从最早一条未聚合消费所在的完整小时窗口开始追赶。
2. `service/enterprise/usage_aggregation.go`
   整窗重算时原实现只覆盖当前仍存在的桶，不会删除该窗口下已经失效的旧部门桶；成员关系或部门归因变化后会残留陈旧快照，导致 summary 返回错误部门行。已修复为窗口级先删后写，保证覆盖式重算。
3. `service/enterprise/usage_aggregation.go`
   `user_count` 在 summary 聚合时直接累加各小时快照的 `UserCount`，同一用户跨多个窗口活跃会被重复计数，和“部门级用户数”口径不一致。已修复为在快照内持久化窗口级去重用户集合，summary 合并时再做跨窗口去重。
4. `service/enterprise/usage_aggregation.go`
   watermark 校验原先禁止对已处理前缀窗口做显式补算，导致文档宣称的“历史回填使用 AggregateWindow 顺序补窗”与实际行为不一致。已修复为允许从已知 watermark 之前的窗口做重算，同时仍阻止跳过缺口的前向写入。

### 验证

- 已通过：`GOCACHE=$(pwd)/.cache/go-build go test ./model/enterprise ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage'`
- 新增回归覆盖：
  - 整窗重算后陈旧部门桶被清理
  - 跨窗口 summary 的 `user_count` 去重
  - 首次后台聚合任务从最早缺失窗口开始回填

### 残余风险

- 未执行 `go test ./...` 全量回归；仓库现有非本故事问题仍存在：
  - `service/enterprise/dingtalk_connectivity_test.go` 依赖本地监听端口
  - 根模块空编译受 `web/classic/dist` 缺失影响
