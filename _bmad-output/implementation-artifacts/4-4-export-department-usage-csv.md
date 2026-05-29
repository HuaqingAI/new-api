# Story 4.4: 导出部门用量 CSV

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 导出当前筛选条件下的部门用量 CSV,
so that 我可以把部门用量用于管理汇报和离线分析。

## Acceptance Criteria

1. **Given** 管理员在用量看板设置筛选和排序  
   **When** 点击导出 CSV  
   **Then** CSV 行集合等于页面当前筛选条件下的部门集合  
   **And** 行排序与页面当前排序一致。
2. **Given** CSV 生成完成  
   **When** 管理员打开文件  
   **Then** 字段按 `部门 ID`、`部门名称`、`父部门`、`周期开始`、`周期结束`、`请求数`、`prompt_tokens`、`completion_tokens`、`quota`、`用户数` 顺序输出  
   **And** 空值输出为空字符串，不输出 `null`。
3. **Given** CSV 包含部门数据  
   **When** 文件生成  
   **Then** 标题行之前包含 `# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和`  
   **And** 包含“未归属”行，部门 ID 为空、部门名称为 `未归属`。

## Tasks / Subtasks

- [x] 新增后端导出服务，基于现有 summary 聚合结果生成 CSV，而不是前端直接序列化表格 (AC: 1, 2, 3)
  - [x] 新增 `service/enterprise/usage_export.go`，封装导出查询、排序、父部门补齐和 CSV 写出；导出数据源必须建立在 Story 4.1 的 `enterprise_usage_snapshots` summary 能力之上，不允许回退到 `logs` 全表在线扫描，也不要复制一套新的聚合 SQL。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; docs/enterprise/usage-aggregation.md]
  - [x] 导出服务应先复用 `service/enterprise/usage_aggregation.go` 的 summary 查询结果，再按导出需要补齐 `enterprise_departments.parent_id` 与父部门名称映射；当前 summary 返回中没有 `parent_department` 字段，因此不能只把现有 API 原样写成 CSV。[Source: service/enterprise/usage_aggregation.go; dto/enterprise/usage.go; model/enterprise/department.go]
  - [x] “未归属”行保持 `dept_id = nil` / `dept_name = "未归属"` 语义，CSV 中 `部门 ID` 与 `父部门` 输出空字符串；其它列继续使用快照聚合值，不得丢弃该行。[Source: docs/enterprise/usage-aggregation.md#归因口径; _bmad-output/planning-artifacts/epics.md#Story 4.4: 导出部门用量 CSV]
  - [x] CSV 写出使用标准库 `encoding/csv` 仅作为格式化器可以接受，但任何业务 JSON 处理仍必须走 `common/json.go`；文件内容中不得出现 `null`、`<nil>`、Go 默认零值占位文本或敏感凭据。[Source: AGENTS.md#rule-1-json-package--use-commonjsongo; _bmad-output/planning-artifacts/architecture.md#Data Exchange Formats]
  - [x] 文件名遵循架构约定 `<feature>-<scope>-<yyyymmdd>[-<yyyymmdd>].csv`，本故事应对齐类似 `usage-department-20260501-20260531.csv` 的命名；响应采用同步文件流，不要提前引入导出任务表或下载链接。[Source: _bmad-output/planning-artifacts/architecture.md#api-naming-conventions; _bmad-output/planning-artifacts/architecture.md#Nice-to-Have Gaps（V1 可选，不影响验收）]

- [x] 暴露 `/api/enterprise/usage/export` 导出接口，并保持现有企业权限与参数边界 (AC: 1, 2, 3)
  - [x] 在 `dto/enterprise/usage.go` 中新增导出查询 DTO，例如 `DepartmentUsageExportQuery`；可选标量继续使用指针类型并加 `omitempty`，避免 `0` 或 `false` 被吞掉。[Source: AGENTS.md#rule-6-upstream-relay-request-dtos--preserve-explicit-zero-values; dto/enterprise/usage.go]
  - [x] 在 `controller/enterprise/usage.go` 新增 `ExportDepartmentUsageCSV`，仅负责 query 解析、权限检查、调用导出 service、设置 `Content-Type` / `Content-Disposition` 并返回文件流；不要在 controller 中拼装 summary 聚合、拼接父部门 SQL 或手写 CSV 行逻辑。[Source: controller/enterprise/usage.go; _bmad-output/planning-artifacts/architecture.md#backend-内部分层]
  - [x] 在 `router/enterprise-router.go` 注册 `GET /api/enterprise/usage/export`，继续沿用 `middleware.EnterpriseAdmin()`；4.4 不要自行扩展“经营负责人”或部门管理员的导出权限模型。[Source: router/enterprise-router.go; middleware/enterprise_admin.go; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
  - [x] 参数校验与 summary 保持同一时间窗口语义 `[from, to)`；若增加导出排序参数，需与页面当前 summary 排序显式对齐，并避免与 4.3 已存在的详情排行 `sort` 参数语义冲突。[Source: controller/enterprise/usage.go; web/default/src/features/enterprise-usage/index.tsx; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]

- [x] 在 Default `enterprise-usage` 总览页新增导出动作，并让导出严格复用当前总览筛选/排序状态 (AC: 1, 3)
  - [x] 扩展 `web/default/src/features/enterprise-usage/index.tsx` 与 `api.ts`，在现有总览视图中增加导出按钮或导出触发器；不要新开独立页面，也不要把导出入口塞到 `usage-logs`、`dashboard` 或 Classic 特例中。[Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; web/default/src/features/enterprise-usage/api.ts]
  - [x] 导出请求必须复用当前总览筛选条件，至少包括 `from`、`to`、可选 `tenant_id`，以及总览表当前排序；若 4.4 需要补“总览显式排序”能力，应使用新的 summary search 字段（如 `summary_sort` / `summary_order`）而不是复用 4.3 已用于详情用户排行的 `sort` 字段。[Source: web/default/src/features/enterprise-usage/index.tsx; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
  - [x] 当前 `dept_id` 仅用于 detail panel 选中态，`log_user` 仅用于 recent logs drill-down；除非本故事显式把它们升级为 summary 过滤条件，否则导出范围应视为“当前总览部门集合”，不要被 detail-only search state 意外缩窄为单部门或单用户。[Source: web/default/src/features/enterprise-usage/index.tsx; web/default/src/features/enterprise-usage/types.ts]
  - [x] 导出完成的文件内容必须与页面顶部 disclaimer 语义一致；页面仍固定展示 `enterprise.usage.multi_dept_disclaimer`，CSV 首行注释不允许缺失、隐藏或与页面文案分叉成另一套口径。[Source: _bmad-output/planning-artifacts/architecture.md#Attribution Model; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]

- [x] 固定 CSV 列顺序、排序和格式化语义，避免导出文件与页面统计不一致 (AC: 1, 2, 3)
  - [x] CSV 表头固定为 `部门 ID,部门名称,父部门,周期开始,周期结束,请求数,prompt_tokens,completion_tokens,quota,用户数`；不要因前端 locale、字段别名或表格显示顺序变化而改动导出列序。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.4: 导出部门用量 CSV]
  - [x] `周期开始`、`周期结束` 应直接反映 summary 聚合窗口的 Unix 秒时间范围；若页面展示做了本地化格式化，导出层仍需使用稳定、可解析的统一格式，并在测试里固定下来，避免时区导致列值漂移。[Source: dto/enterprise/usage.go; service/enterprise/usage_aggregation.go]
  - [x] 行排序必须与页面当前总览排序一致。若 4.4 不新增显式总览排序控件，则至少要与当前 `normalizeDepartmentUsageItems()` 的默认规则保持一致：`request_count DESC`、`quota DESC`、`dept_name ASC`；若新增显式排序，则页面与导出必须共用同一排序定义而不是各自实现相近但不完全一致的规则。[Source: web/default/src/features/enterprise-usage/index.tsx]
  - [x] 空值仅允许输出为空字符串；尤其是 `dept_id = nil`、`parent_id = nil`、父部门不存在或已停用时，不得输出 `null`、`0`、`undefined`、`未设置` 等非 AC 约定值。[Source: _bmad-output/planning-artifacts/epics.md#Story 4.4: 导出部门用量 CSV; model/enterprise/department.go]

- [x] 补齐测试与回归，覆盖最容易错的排序、父部门补齐和文件响应契约 (AC: 1, 2, 3)
  - [x] 新增/扩展 `service/enterprise/usage_aggregation_test.go` 或新增 `service/enterprise/usage_export_test.go`，覆盖：未归属行、父部门名称补齐、固定列顺序、注释行、空字符串输出、以及导出排序与 summary/页面排序一致。[Source: service/enterprise/usage_aggregation_test.go; _bmad-output/planning-artifacts/architecture.md#Test Organization]
  - [x] 扩展 `controller/enterprise/usage_test.go`，覆盖导出接口参数校验、管理员权限、`Content-Type`、`Content-Disposition`、文件名规则和文件内容前两行（注释 + 表头）。[Source: controller/enterprise/usage_test.go; middleware/enterprise_admin.go]
  - [x] 扩展 `tests/api/enterprise_usage_test.go`，覆盖真实路由 `/api/enterprise/usage/export` 返回文件流且内容与快照/部门元数据一致，不读取 `logs` 明细作默认导出路径。[Source: tests/api/enterprise_usage_test.go; docs/enterprise/usage-aggregation.md#幂等与回填]
  - [x] 扩展 `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`，至少覆盖：导出按钮存在、导出请求携带当前 `from/to/tenant_id` 与 summary 排序、detail-only search state 不会错误污染 summary 导出范围、以及 disclaimer 文案仍固定可见。[Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx; web/default/src/features/enterprise-usage/api.ts]

## Dev Notes

- Story 4.4 是 Epic 4 的“总览导出”能力，不是新的聚合故事。后端必须建立在 Story 4.1 的 `enterprise_usage_snapshots` 之上，前端必须建立在 Story 4.2 的 `enterprise-usage` 总览页之上，不能各自再造统计路径。[Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- Story 4.3 已经占用了 `enterprise-usage` search state 里的 `sort` 字段作为详情页用户排行排序。如果 4.4 要给总览导出引入“当前排序”参数，必须避免与 4.3 的 detail sort 语义相撞，否则会出现页面详情排序正常、导出却按错误字段排序的回归。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; web/default/src/features/enterprise-usage/index.tsx]
- 当前总览页的部门列表排序来自 `normalizeDepartmentUsageItems()`，不是后端返回顺序，也没有单独 summary sort 参数。因此 4.4 不能默认假设“后端自然顺序 == 页面当前顺序”；要么明确固化这一默认规则到导出层，要么在本故事中把 summary 排序状态显式化并让页面与导出共用。[Source: web/default/src/features/enterprise-usage/index.tsx]
- 4.4 明确需要 `父部门` 列，但当前 summary DTO/响应没有该字段。这意味着导出层必须额外读取 `enterprise_departments` 建立 `dept_id -> parent department name` 映射，而不是简单把现有 summary API 输出转存为 CSV。[Source: dto/enterprise/usage.go; model/enterprise/department.go]
- 架构文档对 CSV 一处提到“长耗时操作可任务化”，但同一文档又把“CSV 大文件异步化”列为 V1 可选优化，并在 FR-15 映射里明确 `service/enterprise/usage_export.go` 为流式生成。因此 4.4 应先实现同步流式文件响应；异步下载链接属于后续性能演进，不要在本故事提前扩 scope。[Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping; _bmad-output/planning-artifacts/architecture.md#Nice-to-Have Gaps（V1 可选，不影响验收）]
- `enterprise.usage.multi_dept_disclaimer` 是架构级强约束。CSV 首行注释、总览页固定提示、详情页固定提示都必须共用同一业务口径，不能出现页面写“部门间不可加和”，导出文件却省略或改成另一套说明的情况。[Source: _bmad-output/planning-artifacts/architecture.md#Attribution Model; _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md; _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- 不要触碰 `model/log.go`、`controller/log.go`、relay 路径、`pkg/billingexpr/**`，也不要把 CSV 导出做成写临时表或修改快照模型字段的“反向侵入式”实现。4.4 只是读取既有 summary 快照和部门元数据，并输出文件响应。[Source: AGENTS.md#overview; _bmad-output/planning-artifacts/epics.md#Additional Requirements]

### Project Structure Notes

- 预计新增文件：
  - `service/enterprise/usage_export.go`
  - 需要时新增 `service/enterprise/usage_export_test.go`
- 预计修改文件：
  - `controller/enterprise/usage.go`
  - `controller/enterprise/usage_test.go`
  - `dto/enterprise/usage.go`
  - `router/enterprise-router.go`
  - `tests/api/enterprise_usage_test.go`
  - `web/default/src/features/enterprise-usage/api.ts`
  - `web/default/src/features/enterprise-usage/index.tsx`
  - `web/default/src/features/enterprise-usage/types.ts`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- 保持不改或仅只读依赖：
  - `service/enterprise/usage_aggregation.go` 的归因语义
  - `model/log.go`
  - `controller/log.go`
  - `relay/**`
  - `pkg/billingexpr/**`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 4.4: 导出部门用量 CSV]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic 4: 组织用量看板、详情与报告导出]
- [Source: _bmad-output/planning-artifacts/architecture.md#Attribution Model]
- [Source: _bmad-output/planning-artifacts/architecture.md#api-naming-conventions]
- [Source: _bmad-output/planning-artifacts/architecture.md#Requirements → Components Mapping]
- [Source: _bmad-output/planning-artifacts/architecture.md#Nice-to-Have Gaps（V1 可选，不影响验收）]
- [Source: _bmad-output/implementation-artifacts/4-1-aggregate-department-usage-snapshots.md]
- [Source: _bmad-output/implementation-artifacts/4-2-display-department-usage-overview-dashboard.md]
- [Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md]
- [Source: docs/enterprise/usage-aggregation.md]
- [Source: controller/enterprise/usage.go]
- [Source: controller/enterprise/usage_test.go]
- [Source: dto/enterprise/usage.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: service/enterprise/usage_aggregation_test.go]
- [Source: model/enterprise/department.go]
- [Source: router/enterprise-router.go]
- [Source: middleware/enterprise_admin.go]
- [Source: tests/api/enterprise_usage_test.go]
- [Source: web/default/src/features/enterprise-usage/api.ts]
- [Source: web/default/src/features/enterprise-usage/index.tsx]
- [Source: web/default/src/features/enterprise-usage/types.ts]
- [Source: web/default/src/features/enterprise-usage/enterprise-usage.test.tsx]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工执行 `bmad-create-story` workflow，因为 `_bmad/scripts/resolve_customization.py` 依赖 Python 3.11+ 的 `tomllib`，当前环境需按 skill fallback 规则手工解析。
- 已读取 Epic 4、architecture、Story 4.1/4.2/4.3、当前 `enterprise-usage` 前后端实现、近期提交历史与测试文件，补齐 4.4 的导出边界、排序语义和父部门映射约束。
- 运行 `mkdir -p .cache/go-build && GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage'`，使用工作区内 Go cache 绕开沙箱对默认构建缓存目录的权限限制。
- 运行 `cd web/default && bun run typecheck` 与 `cd web/default && bun test src/features/enterprise-usage/enterprise-usage.test.tsx` 验证前端类型与导出交互回归。

### Completion Notes List

- Ultimate context engine analysis completed - comprehensive developer guide created.
- 已明确 4.4 默认采用同步流式 CSV 导出，不在本故事提前引入异步导出任务。
- 已明确 `父部门` 列必须由后端导出链路额外补齐，不能直接复用现有 summary DTO。
- 已明确 4.3 detail `sort` 与 4.4 summary/export sorting 不能共用同一个 search 字段。
- 已新增独立 summary/export 排序字段 `summary_sort` / `summary_order`，并让页面表格排序与 CSV 导出共用同一排序语义。
- 已新增同步 CSV 文件流导出接口、总览导出按钮、多语言文案，以及后端/前端/路由回归测试。

### File List

- `_bmad-output/implementation-artifacts/4-4-export-department-usage-csv.md`
- `controller/enterprise/department_membership_test.go`
- `controller/enterprise/usage.go`
- `controller/enterprise/usage_test.go`
- `dto/enterprise/usage.go`
- `router/enterprise-router.go`
- `service/enterprise/usage_aggregation_test.go`
- `service/enterprise/usage_export.go`
- `tests/api/enterprise_usage_test.go`
- `web/default/src/features/enterprise-usage/api.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/features/enterprise-usage/hooks/use-department-usage-summary.ts`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

### Change Log

- 新增 `service/enterprise/usage_export.go`，基于 summary 快照与部门元数据生成同步 CSV 文件流，补齐父部门列、固定列顺序、未归属行和命名规则。
- 新增 `/api/enterprise/usage/export` 接口与导出 DTO，并将 summary/export 排序语义显式化为 `summary_sort` / `summary_order`。
- 更新 Default `enterprise-usage` 总览页，新增总览排序切换与导出按钮，确保导出仅复用总览筛选/排序，不受 detail-only state 污染。
- 扩展 service、controller、API 和前端测试，覆盖 CSV 注释/表头、文件名、权限、父部门映射、未归属行与排序一致性。
- Senior Developer Review（AI）自动修复导出错误响应处理、浏览器下载释放时序，以及前后端 `dept_name` 排序平局时的稳定性问题，并补充对应回归测试。

## Senior Developer Review (AI)

### Outcome

- Approve after auto-fix.

### Findings

1. High: 前端导出接口把 `blob` 响应一律当作成功文件处理，后端返回 JSON 业务错误时会下载一个伪 CSV，而不是向用户显示失败信息。已在 `web/default/src/features/enterprise-usage/api.ts` 中按 `Content-Type` 识别并解析 JSON 错误 blob。
2. Medium: 导出按钮点击后立即 `URL.revokeObjectURL(...)`，在部分浏览器中会与下载启动竞态，造成 CSV 偶发下载失败。已在 `web/default/src/features/enterprise-usage/index.tsx` 中改为延迟释放。
3. Medium: 前端与后端在 `summary_sort=dept_name` 下对“同名部门 / 未归属”平局排序的语义并不完全一致，页面顺序与 CSV 顺序存在漂移风险。已统一平局规则，并在 `service/enterprise/usage_export.go`、`web/default/src/features/enterprise-usage/index.tsx` 及对应测试中锁定行为。

### Validation

- 已复核 AC 1-3 与实现一致。
- 已复核导出链路继续基于 `enterprise_usage_snapshots`，未回退到 `logs` 明细扫描。
- 已运行：
  - `GOCACHE=$(pwd)/.cache/go-build go test ./service/enterprise ./controller/enterprise ./tests/api -run 'Usage|EnterpriseUsage'`
  - `cd web/default && bun test src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `cd web/default && bun run typecheck`
