---
create_story_refreshed_at: 2026-06-03 03:00 +0800
source_status_note: "Enterprise Epic 6 in epics.md and sprint-status.yaml contains stories 6.1-6.5 only; this 6.6 artifact is a YOLO-derived follow-up guardrail story for the completed Epic 6 account identity work."
baseline_commit: 17b86fe2929ea31bb451c0e618df4cd2a9339c4e
---

# Story 6.6: 固化企业账号标识契约与回归护栏

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 企业管理员和后续维护开发者,
I want 固化企业账号标识、历史快照、筛选值和展示值的统一契约,
so that Story 6.4 / 6.5 已完成的 username 可读化与跨页面展示规则不会在后续治理、用量、风险和预算迭代中回归。

## Acceptance Criteria

1. **Given** 系统已经支持可读 username 生成、受控 username 修改和“可读名称优先，username / ID 兜底”的展示规则  
   **When** 开发者查看企业账号标识契约文档或 API 契约  
   **Then** 能清楚区分当前身份字段、历史 snapshot 字段、UI 展示字段和底层筛选字段  
   **And** 文档明确 `display_name` 只用于展示，日志/风险历史快照不得被回写或重算。
2. **Given** 用户被 rename 或 display name 与 username 不一致  
   **When** 组织、预算、用量、风险、治理时间线和 Classic 必要入口展示该用户  
   **Then** 主展示遵循“可读名称 -> username -> user ID”顺序  
   **And** 辅助信息只展示真实存在的 username / user ID，不得出现 `#-`、空 ID、把 display name 当 username 查询等伪线索。
3. **Given** 开发者后续修改企业 DTO、service lookup、OpenAPI、Default/Classic 展示 helper 或查询入口  
   **When** 运行本故事新增的回归测试矩阵  
   **Then** 测试会覆盖当前身份字段透传、历史 snapshot 保留、日志/风险筛选值不变、helper fallback 边界和多语言文案  
   **And** 任一页面回到 username-only 主展示或 display-name-as-filter 的实现会被测试阻止。

## Tasks / Subtasks

- [x] 梳理并落文档化企业账号标识契约，明确当前身份、历史快照、展示值和筛选值的边界 (AC: 1, 3)
  - [x] 新增或更新企业文档，例如 `docs/enterprise/user-identity-display-contract.md`，用表格定义字段语义：`user_id`、`username`、`display_name`、`username_snapshot`、`department_snapshot`、`target_username`、`target_display_name`、`requester_display_name`、`trace.display_name`、recent logs `user_options`。[Source: service/enterprise/user_lookup.go; dto/enterprise/usage.go; dto/enterprise/alert.go; dto/enterprise/department_budget.go; dto/enterprise/governance_timeline.go]
  - [x] 文档必须写清硬边界：`logs.username`、风险事件写入时的 `event.Username` / `username_snapshot` 是历史快照；rename 后当前治理视图可以展示新值，但不得批量回写历史日志、风险事件或用量快照。[Source: _bmad-output/implementation-artifacts/6-4-fix-dingtalk-username-strategy-and-controlled-rename.md; model/log.go; service/enterprise/alert.go]
  - [x] 文档必须写清 UI 底层筛选值：最近日志入口、风险 username 筛选、任何现有日志模块查询仍使用 `username` / `user_id` 语义，不允许把 `display_name` 作为筛选参数传入。[Source: _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md; web/default/src/features/enterprise-usage/index.tsx; web/default/src/features/enterprise-alerts/index.tsx]

- [x] 冻结后端读侧身份字段与 API 契约，避免 6.5 只在 UI 层生效 (AC: 1, 2, 3)
  - [x] 复核并补齐 `docs/openapi/api.json` 中企业相关响应字段，至少覆盖部门成员、预算 wallet/allocation、用量详情排行与 recent logs user options、风险事件、投递 trace、治理时间线、额度申请列表里已经暴露或应暴露的 `display_name` / `username_snapshot` 字段。[Source: docs/openapi/api.json; dto/enterprise/department_membership.go; dto/enterprise/department_budget.go; dto/enterprise/usage.go; dto/enterprise/alert.go; dto/enterprise/governance_timeline.go; dto/enterprise/quota_request.go]
  - [x] 继续以 `service/enterprise/user_lookup.go` 的 `loadCurrentUserIdentities` 作为当前身份 lookup 入口；如发现 service 仍自行拼接 `username/display_name` 查询，优先收敛复用，不新增重复 helper。[Source: service/enterprise/user_lookup.go; service/enterprise/usage_aggregation.go; service/enterprise/alert.go]
  - [x] 保持 DB 查询三库兼容，不引入 JSONB、数据库特有函数或 raw SQL 拼接；如果需要 raw SQL，按项目规则处理 MySQL/SQLite/PostgreSQL 差异。[Source: AGENTS.md#Rule 2: Database Compatibility — SQLite, MySQL >= 5.7.8, PostgreSQL >= 9.6]

- [x] 加强 Default 与 Classic 展示 helper 的边界测试，锁定“可读名优先 + 真实辅助信息”语义 (AC: 2, 3)
  - [x] Default `web/default/src/features/enterprise-organization/lib/user-display.ts` 增加或补齐纯函数测试，覆盖：`displayName + username + userId`、`displayName + username + no userId`、`username only`、`userId only`、空输入，确认不会渲染 `User ID #-`。[Source: web/default/src/features/enterprise-organization/lib/user-display.ts; _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md#Follow-up Review - 2026-06-03]
  - [x] Classic `web/classic/src/pages/Enterprise/alertHelpers.js` 增加同等 smoke / helper 测试，覆盖中文标签下不会出现 `用户 ID #-`，且 Department / Alerts 表格仍消费同一 helper。[Source: web/classic/src/pages/Enterprise/alertHelpers.js; web/classic/src/pages/Enterprise/Alerts.smoke.test.js; web/classic/src/pages/Enterprise/Department.smoke.test.js]
  - [x] 若新增可见文案，Default 必须同步 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json` 并运行 `bun run i18n:sync`；Classic 只补必要语言包，不做深度 UI 重构。[Source: AGENTS.md#Internationalization (i18n); _bmad-output/planning-artifacts/architecture.md#Classic Theme]

- [x] 补齐企业主链路回归矩阵，覆盖 rename 后当前身份展示与历史快照保留 (AC: 1, 2, 3)
  - [x] 后端测试覆盖用量详情：用户 rename 后，`display_name` / 当前 `username` 随当前用户表返回，但 recent logs 查询参数仍传 username 语义，历史统计快照不被回写。[Source: service/enterprise/usage_aggregation.go; controller/enterprise/usage_test.go; model/log.go]
  - [x] 后端测试覆盖风险事件：`username_snapshot` 保留事件写入时用户名，当前 `display_name` / `username` 由 lookup 补齐，department snapshot 不被展示逻辑改写。[Source: service/enterprise/alert.go; controller/enterprise/alert_test.go]
  - [x] 前端测试覆盖组织/预算/用量/风险页面：主展示优先可读名，辅助信息不伪造 ID，最近日志按钮仍以 username 查询，风险页仍显示历史 username snapshot。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx; web/default/src/features/enterprise-usage/enterprise-usage.test.tsx; web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx]
  - [x] 如治理时间线或额度申请列表已有用户展示字段，补最小 service/controller/UI 测试，确认 `requester_display_name`、`target_display_name` 等字段与 6.5 规则一致。[Source: service/enterprise/governance_timeline.go; service/enterprise/quota_request.go; dto/enterprise/governance_timeline.go; dto/enterprise/quota_request.go]

- [x] 保持范围收敛，不重新设计账号、日志、relay 或企业治理入口 (AC: 1, 2, 3)
  - [x] 不修改 DingTalk OAuth/session 主链路、`custom_oauth_providers`、relay 请求 DTO、billing expression、`model/log.go` 持久化语义或 `/v1/**` 协议表面。[Source: AGENTS.md; _bmad-output/planning-artifacts/architecture.md#DingTalk Integration Boundary; _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
  - [x] 不把 Epic 7A 的员工额度申请入口迁移、治理工作台 i18n 或预算池类型约束纠偏混入本故事；这些已有独立 follow-up scope。[Source: _bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md; _bmad-output/planning-artifacts/epics.md#Epic 7A: 企业治理体验纠偏与预算池约束校正]
  - [x] 如果发现真实代码已经满足某项契约，只补文档和测试，不做无意义重构；优先让回归护栏覆盖已完成事实。[Source: _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md]

## Dev Notes

- 目标 Story 6.6 未出现在当前 `_bmad-output/planning-artifacts/epics.md` 的企业 Epic 6 章节，也未出现在 `_bmad-output/implementation-artifacts/sprint-status.yaml`。用户要求 `#YOLO` 且输出路径为 `6-6-*.md`，因此本故事按企业 Epic 6 的已完成成果派生为“契约冻结 + 回归护栏”后续，不更新 sprint status 中不存在的 key。[Source: _bmad-output/planning-artifacts/epics.md#Epic 6: 企业治理视图重构与账号标识修正; _bmad-output/implementation-artifacts/sprint-status.yaml]
- Story 6.4 已完成可读 username 生成、受控 rename、审计、缓存刷新和历史日志/风险 snapshot 保留规则。6.6 不重新实现 rename，而是把 rename 后“当前身份”和“历史快照”并存的契约固化为文档、OpenAPI 和测试。[Source: _bmad-output/implementation-artifacts/6-4-fix-dingtalk-username-strategy-and-controlled-rename.md]
- Story 6.5 已完成跨企业页面用户展示规则，并在 review 中修复过 `User ID #-` / `用户 ID #-` 边界。6.6 的重点是防止后续页面或 DTO 修改重新引入这个回归。[Source: _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md#Follow-up Review - 2026-06-03]
- 当前 Default helper `formatEnterpriseUserPrimary/Secondary` 与 Classic helper `formatClassicEnterpriseUserPrimary/Secondary` 已有相同语义，但处于不同前端主题中；本故事不强制跨主题共享代码，只要求行为和测试矩阵一致。[Source: web/default/src/features/enterprise-organization/lib/user-display.ts; web/classic/src/pages/Enterprise/alertHelpers.js]
- 当前后端已有 `loadCurrentUserIdentities`，用量与风险读侧已开始透传 `display_name`。后续如发现治理时间线、额度申请、预算 wallet 还有局部字段遗漏，应优先复用这个 lookup，而不是每个 service 发明自己的字段拼接。[Source: service/enterprise/user_lookup.go; service/enterprise/usage_aggregation.go; service/enterprise/alert.go]
- 这不是新 UX story，也不是 Epic 7A correct-course story。不要把员工额度申请入口迁移、预算池多类型并存、治理 label i18n 修复塞进 6.6；那些工作已有 `7a-*` story 候选和 sprint-status 项。[Source: _bmad-output/planning-artifacts/sprint-change-proposal-2026-06-02-enterprise-governance-followup-cc.md; _bmad-output/implementation-artifacts/sprint-status.yaml]

### Project Structure Notes

- 预计修改或重点复核：
  - `docs/enterprise/user-identity-display-contract.md`（新增）
  - `docs/openapi/api.json`
  - `service/enterprise/user_lookup.go`
  - `service/enterprise/usage_aggregation.go`
  - `service/enterprise/alert.go`
  - `service/enterprise/governance_timeline.go`
  - `service/enterprise/quota_request.go`
  - `controller/enterprise/usage_test.go`
  - `controller/enterprise/alert_test.go`
  - `web/default/src/features/enterprise-organization/lib/user-display.ts`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
  - `web/classic/src/pages/Enterprise/alertHelpers.js`
  - `web/classic/src/pages/Enterprise/Alerts.smoke.test.js`
  - `web/classic/src/pages/Enterprise/Department.smoke.test.js`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`（仅当新增可见文案）
- 保持不改或只读依赖：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `router/relay-router.go`
  - `docs/openapi/relay.json`
  - DingTalk OAuth/session 主流程，除非发现与契约测试直接相关的 bug

### Validation Notes

- 本故事创建阶段未运行实现测试；它是 ready-for-dev 上下文文档。
- Review 自动修复后已通过：
  - `GOCACHE=/private/tmp/new-api-go-build-cache go test ./controller/enterprise -run 'Test(UsageDetail|AlertEvents|GovernanceTimeline|QuotaRequest)' -count=1`
  - `cd web/default && bun test ./src/features/enterprise-organization/enterprise-organization.test.tsx ./src/features/enterprise-usage/enterprise-usage.test.tsx ./src/features/enterprise-alerts/enterprise-alerts.test.tsx`
  - `cd web/default && bun test ./src/features/enterprise-usage/enterprise-usage.test.tsx`
  - `cd web/default && bun run typecheck`
  - `cd web/classic && bun test ./src/pages/Enterprise/Alerts.smoke.test.js ./src/pages/Enterprise/Department.smoke.test.js`
  - `node -e "JSON.parse(require('fs').readFileSync('docs/openapi/api.json','utf8')); console.log('api.json ok')"`
- 如果运行仓库级 `go test ./...` 或 Default 全量 `bun test`，需先对照 6.5 completion notes 区分既有非企业身份契约失败，不要把无关 agent-platform 文案或 relay/channel 断言误归因到本故事。[Source: _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md#Completion Notes List]

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 6: 企业治理视图重构与账号标识修正]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-6: 支持钉钉 OAuth 登录]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14: 查看部门用量详情]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#7.5 账号标识策略]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md#7.6 企业页面统一展示规则]
- [Source: _bmad-output/planning-artifacts/architecture.md#DingTalk Integration Boundary]
- [Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
- [Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
- [Source: _bmad-output/implementation-artifacts/6-4-fix-dingtalk-username-strategy-and-controlled-rename.md]
- [Source: _bmad-output/implementation-artifacts/6-5-unify-enterprise-user-display-rules.md]
- [Source: service/enterprise/user_lookup.go]
- [Source: service/enterprise/usage_aggregation.go]
- [Source: service/enterprise/alert.go]
- [Source: web/default/src/features/enterprise-organization/lib/user-display.ts]
- [Source: web/classic/src/pages/Enterprise/alertHelpers.js]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 03:00 +0800：执行 `.claude/skills/bmad-create-story` workflow for Story 6.6；已读取 `SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-03 03:00 +0800：`python3 _bmad/scripts/resolve_customization.py --skill .claude/skills/bmad-create-story --key workflow` 因当前 Python 缺少 `tomllib` 失败，已按 skill fallback 手工读取 `.claude/skills/bmad-create-story/customize.toml`；无 team/user override。
- 2026-06-03 03:00 +0800：已读取 `_bmad/bmm/config.yaml`，确认 `user_name=hth`、`project_name=new-api`、沟通语言/文档输出语言为中文，规划目录为 `_bmad-output/planning-artifacts`，实现目录为 `_bmad-output/implementation-artifacts`。
- 2026-06-03 03:00 +0800：已完整读取 `_bmad-output/implementation-artifacts/sprint-status.yaml`；确认企业 Epic 6 只有 6.1-6.5，`6-6-*` key 不存在；本次不新增或伪造 sprint-status 项。
- 2026-06-03 03:00 +0800：已按 discover-inputs 读取/分析 epics、PRD/addendum、architecture、approved correct-course、Story 6.3/6.4/6.5、最近 git commit 标题与关键代码文件。
- 2026-06-03 03:00 +0800：未找到 `project-context.md` persistent fact 文件；本故事以上述规划工件、AGENTS.md 和真实代码扫描为准。
- 2026-06-03 03:00 +0800：未进行外部技术检索；本故事聚焦仓库内既有契约、文档和测试护栏，没有发现依赖时效性外部 API 或库升级的信息需求。
- 2026-06-03：执行 `.claude/skills/bmad-dev-story` workflow for Story 6.6；`resolve_customization.py` 因当前 Python 缺少 `tomllib` 失败，已按 fallback 手动读取 `.claude/skills/bmad-dev-story/customize.toml`，无 team/user override；未找到 `project-context.md`。
- 2026-06-03：捕获 `baseline_commit=17b86fe2929ea31bb451c0e618df4cd2a9339c4e`，story 状态从 `ready-for-dev` 更新为 `in-progress`；sprint-status 中不存在 6.6 key，未新增或伪造 sprint-status 项。
- 2026-06-03：新增企业账号标识与展示契约文档，并用 `rg` 验证 `user_id`、`username_snapshot`、`department_snapshot`、`target_display_name`、`requester_display_name`、`trace.display_name`、recent logs `user_options`、`display_name` 只展示、不得批量回写和不得作为筛选参数等边界存在。
- 2026-06-03：补齐 `docs/openapi/api.json` 中 `DepartmentUsageUserRankItem.display_name`、`DepartmentUsageLogUserOption`、`DepartmentUsageLogFilters.user_options`、`AlertEventItem.display_name/username_snapshot`、`AlertDeliveryTraceItem.user_id/display_name`；用 Node 解析 OpenAPI JSON 验证相关字段无缺失。
- 2026-06-03：运行 `GOCACHE=/private/tmp/new-api-go-build-cache go test ./controller/enterprise -run 'Test(UsageDetail|ListAlert|AlertDeliver)' -count=1`，结果通过。
- 2026-06-03：补充 Default 与 Classic 用户展示 helper 边界测试，覆盖无 userId 和空输入时不产生 `User ID #-` / `用户 ID #-`；运行 `bun test ./src/features/enterprise-organization/enterprise-organization.test.tsx`、`bun test ./src/pages/Enterprise/Alerts.smoke.test.js ./src/pages/Enterprise/Department.smoke.test.js`，结果通过。
- 2026-06-03：执行 `.claude/skills/bmad-story-automator-review` workflow for Story 6.6；按用户要求自动修复，不弹出人工菜单。
- 2026-06-03：Review 发现 `docs/openapi/api.json` 缺少额度申请和治理时间线路径/schema，导致 `requester_display_name`、治理目标 `display_name` 等契约未被冻结；已补齐相关 OpenAPI path 和 schema。
- 2026-06-03：Review 发现 Default 用量页 `resolveRecentLogsUserOptions` 在只存在 `username_options` 时伪造 `user_id=index+1`，可能产生虚假 `User ID #1` 辅助信息；已移除伪造 ID 并补测试。
- 2026-06-03：Review 发现 story File List 未覆盖实际变更的后端测试、Default 用量测试和 OpenAPI 补充范围；已同步 File List 和 completion notes。

### Completion Notes List

- 已新增 `docs/enterprise/user-identity-display-contract.md`，用表格冻结当前身份、历史快照、展示值和筛选值边界，明确 `display_name` 只用于展示，日志/风险/用量历史快照不得被回写或重算。
- 已补齐已登记企业 OpenAPI schema 的当前身份展示字段和历史 username snapshot 字段，确保用量详情、recent logs user options、风险事件和投递 trace 的 API 契约不只停留在 UI 层。
- Review 已补齐 OpenAPI 中遗漏的额度申请和治理时间线路径/schema，冻结 `requester_username`、`requester_display_name`、治理目标 `username/display_name/user_id` 等身份字段。
- 已加强 Default 与 Classic 用户展示 helper 的纯函数/smoke 测试，锁定“可读名优先 + 真实 username/user ID 辅助信息”语义，并确认不会回归到 `User ID #-` / `用户 ID #-`。
- Review 已修复 Default 用量页最近日志用户选项 fallback，不再从 `username_options` 伪造 `user_id`；只展示真实存在的 username/user ID。
- 已创建 Story 6.6 artifact，按用户要求输出到 `_bmad-output/implementation-artifacts/6-6-freeze-enterprise-user-identity-contract-and-regression-guardrails.md`。
- 因企业 Epic 6 规划和 sprint-status 中没有 6.6，本故事明确记录为 YOLO-derived follow-up guardrail，不修改 sprint-status。
- 已把 6.4/6.5 的关键经验转成可执行护栏：当前身份和历史 snapshot 分离、display name 不作为筛选值、OpenAPI/DTO 字段冻结、Default/Classic helper 边界测试、组织/预算/用量/风险页面回归矩阵。
- 已对照 create-story checklist 做自检：故事包含来源说明、AC、任务、技术边界、前后端文件位置、测试建议和不可触碰范围，避免 dev agent 误改 relay/billing/log 主链路或混入 Epic 7A scope。

### File List

- `docs/enterprise/user-identity-display-contract.md`
- `docs/openapi/api.json`
- `controller/enterprise/alert_test.go`
- `controller/enterprise/governance_controller_test.go`
- `controller/enterprise/quota_request_test.go`
- `controller/enterprise/usage_test.go`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/features/enterprise-usage/index.tsx`
- `web/default/src/features/enterprise-usage/types.ts`
- `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx`
- `web/default/src/features/enterprise-alerts/enterprise-alerts.test.tsx`
- `web/classic/src/pages/Enterprise/Department.smoke.test.js`
- `_bmad-output/implementation-artifacts/6-6-freeze-enterprise-user-identity-contract-and-regression-guardrails.md`

### Senior Developer Review (AI)

Reviewer: hth on 2026-06-03

Outcome: Approve after auto-fixes. No CRITICAL issues remain.

Findings fixed:

- HIGH: OpenAPI did not document the quota request and governance timeline identity-bearing endpoints, despite AC1/AC3 requiring API contract freeze for `requester_display_name`, current target identity fields, and related user display fields. Fixed by adding `/api/enterprise/quota-requests`, `/api/enterprise/quota-requests/{id}`, `/api/enterprise/quota-requests/{id}/decision`, `/api/enterprise/governance/timeline`, and the related response/request schemas in `docs/openapi/api.json`.
- HIGH: `resolveRecentLogsUserOptions` fabricated `user_id: index + 1` when falling back from `username_options`, which could render fake `User ID #1` style auxiliary information. Fixed in `web/default/src/features/enterprise-usage/index.tsx` and covered by a direct regression test.
- MEDIUM: Story File List was incomplete compared with git reality; backend governance/quota/usage/alert tests, Default usage/alert tests, and OpenAPI contract changes were not fully listed. Fixed by synchronizing the File List.
- LOW: Story status was still `in-progress` when review started. Fixed after review because all blocking findings were auto-fixed and validation passed.

Checklist:

- [x] Story file loaded from `_bmad-output/implementation-artifacts/6-6-freeze-enterprise-user-identity-contract-and-regression-guardrails.md`
- [x] Story Status verified; invocation explicitly requested review despite initial `in-progress`
- [x] Epic and Story IDs resolved: 6.6
- [x] Story Context warning recorded: this is a YOLO-derived follow-up not present in Epic 6 sprint status
- [x] Epic Tech Spec / architecture docs loaded as available
- [x] Tech stack detected: Go/Gin/GORM backend, React/TypeScript Default frontend, React Classic frontend
- [x] MCP doc search/web fallback not needed; review used repo-local story, architecture, DTO, OpenAPI and source files
- [x] Acceptance Criteria cross-checked against implementation
- [x] File List reviewed and corrected
- [x] Tests identified and mapped to ACs
- [x] Code quality review performed on changed files
- [x] Security review performed on changed files and dependencies
- [x] Outcome decided: Approve after auto-fixes
- [x] Review notes appended under `Senior Developer Review (AI)`
- [x] Change Log updated with review entry
- [x] Status updated to `done`
- [x] Sprint status sync checked; `6-6-freeze-enterprise-user-identity-contract-and-regression-guardrails` is not present in sprint-status.yaml, so no sprint key was changed
- [x] Story saved successfully

### Change Log

- 2026-06-03: Created Story 6.6 follow-up guardrail artifact and implemented identity contract documentation, OpenAPI fields, and regression tests.
- 2026-06-03: Senior review auto-fixed missing OpenAPI quota/governance identity contract entries and removed fabricated recent-log fallback user IDs.
