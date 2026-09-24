---
stepsCompleted:
  - step-01-document-discovery
  - step-02-prd-analysis
  - step-03-epic-coverage-validation
  - step-04-ux-alignment
  - step-05-epic-quality-review
  - step-06-final-assessment
includedFiles:
  architecture:
    - _bmad-output/planning-artifacts/architecture.md
  epics:
    - _bmad-output/planning-artifacts/epics.md
  prd:
    - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md
    - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md
    - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/.decision-log.md
    - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/review-rubric.md
    - _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/validation-report.md
  ux: []
missingDocuments:
  - UX Design
---

# Implementation Readiness Assessment Report

**Date:** 2026-05-28
**Project:** new-api

## Step 1: Document Discovery

### PRD Files Found

**Whole Documents:**
- Not found

**Sharded Documents:**
- Folder: `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/`
  - `prd.md` (43885 bytes, 2026-05-27 19:00:29)
  - `addendum.md` (7004 bytes, 2026-05-27 11:49:58)
  - `.decision-log.md` (5462 bytes, 2026-05-27 12:51:06)
  - `review-rubric.md` (15747 bytes, 2026-05-27 12:04:48)
  - `validation-report.md` (6195 bytes, 2026-05-27 12:08:32)

### Architecture Files Found

**Whole Documents:**
- `_bmad-output/planning-artifacts/architecture.md` (109750 bytes, 2026-05-27 17:57:44)

**Sharded Documents:**
- Not found

### Epics & Stories Files Found

**Whole Documents:**
- `_bmad-output/planning-artifacts/epics.md` (37098 bytes, 2026-05-28 12:48:23)

**Sharded Documents:**
- Not found

### UX Design Files Found

**Whole Documents:**
- Not found

**Sharded Documents:**
- Not found

### Issues Found

- WARNING: UX design document not found.
- No duplicate whole/sharded document conflicts found.

## PRD Analysis

### Functional Requirements

FR1: 建立企业组织与部门层级，管理员可以查看由组织同步产生的部门树；每个部门包含稳定 ID、名称、父部门、状态和外部来源标识，并支持至少 3 层部门结构、停用/删除后的历史追溯和同步异常状态展示。

FR2: 维护用户与部门的 N:N 平等成员关系；系统在登录、同步、用量统计中按全部成员关系展开使用，不存在主部门概念，任一部门均可独立向同一用户分配 wallet。

FR3: 保持部门与现有 `Group` 语义分离；`Group` 继续用于模型倍率、可用模型、渠道选择和订阅升级，部门仅用于组织归属、部门预算、部门统计和告警聚合。

FR4: 在组织相关核心数据中预留 SaaS 租户演进字段或映射策略，但 V1 不开放 SaaS 工作空间、公共注册或跨租户管理入口。

FR5: Root 管理员可以配置钉钉企业应用凭据、回调地址、同步开关和同步范围，并执行连通性测试；缺失凭据或权限不足时必须阻止启用并返回可区分错误。

FR6: 支持员工通过钉钉 OAuth 登录；系统根据钉钉身份绑定已有账号或创建账号，并使用最近一次成功同步的本地 `department_membership` 快照读取部门集合。

FR7: 支持管理员执行钉钉部门与成员全量同步，并按配置周期执行定时同步；同步结果必须包含新增、更新、停用、失败、跳过数量，支持幂等重试和日志筛选。

FR8: 处理钉钉组织变更，包括部门改名、移动、停用、成员离职和成员转部门，并明确这些变更对部门成员关系和已分配 wallet 子钱包的影响。

FR9: 管理员可以为部门创建 `balance` 或 `subscription` 类型的部门预算池；`balance` 使用一次性额度和可选过期时间，`subscription` 使用周期类型、周期额度、可选自定义秒数和起始时间。

FR10: 以 wallet 子分配方式承接部门预算扣减；部门预算不进入 relay 路径，管理员分配预算时在目标用户名下创建 `subscriptions` 子钱包行，并通过 `source_type='enterprise_allocation'` 与 `source_allocation_id` 关联来源。

FR11: 保留现有计费表达式和倍率语义；单次请求 quota 继续由模型计费、`Group` 倍率和计费表达式系统计算，部门预算只通过 wallet 子钱包参与现有 subscription 扣费链路。

FR12: 管理员可以查看每个部门预算池及其派生的全部 wallet 子分配状态，包括父池剩余、累计已分配、目标用户、quota、remain_quota、周期/过期和状态，并显示预警/紧急使用率。

FR13: 管理员可以按时间范围查看部门维度请求数、token、quota、模型分布和用户数；聚合采用成员视角重复计入口径，多部门用户同一请求在所有归属部门各计一次，并强制展示部门间不可加和说明。

FR14: 管理员可以查看单个部门用量详情，包括用户排行、模型分布、时间趋势和最近消费日志入口；同一用户可出现在多个部门排行中，详情页同样展示部门间不可加和说明。

FR15: 管理员可以导出当前筛选条件下的部门用量 CSV；CSV 字段顺序固定，包含未归属行，数值与页面一致，标题前必须包含部门间不可加和注释行。

FR16: 系统可以按配置向指定邮箱发送部门用量定期报告，管理员可配置接收人、频率和时间范围；邮件失败记录原因且不影响看板数据生成。

FR17: 当请求触发现有敏感词检查或过滤规则时，系统记录内容风险事件，包含用户、请求发生时的部门成员关系快照、模型、时间、请求追踪 ID、风险类型和处理结果。

FR18: 管理员可以配置内容风险告警规则、接收人和告警通道；规则可按风险类型和部门范围启停，保存时校验接收人或 Webhook 格式并记录操作日志。

FR19: 触发告警规则时，系统发送可追溯告警通知，包含部门、用户、模型、时间、风险类型和请求追踪 ID；失败时按默认退避策略重试，并支持最终失败后的后台手动重发。

FR20: 管理员可以在组织用量或监控页面查看部门风险次数、风险率和近期趋势；风险率按部门成员请求口径计算，多部门成员重复计入，并展示公式和部门间不可加和说明。

Total FRs: 20

### Non-Functional Requirements

NFR1: 兼容性 - 新增数据库对象和查询必须同时兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。

NFR2: 安全 - 钉钉应用密钥、Webhook、邮箱凭据等敏感配置不得明文返回前端，不得写入普通日志。

NFR3: 隐私 - 内容风险事件默认只展示追溯所需摘要，不扩大现有日志中敏感内容的可见范围。

NFR4: 可靠性 - 组织同步失败不得影响现有登录、API 调用、计费和日志链路。

NFR5: 一致性 - 部门预算 wallet 子分配的扣减必须与用户 quota 消费最终一致；铁律 `Σ children.committed_quota ≤ parent.total_quota / cycle_quota` 必须在分配/扩容前强制校验，不能出现父池负余额或用户扣费成功但部门 wallet 未同步扣减。

NFR6: 可观测性 - 同步任务、配额拦截、告警发送必须有后台可查状态和错误原因。

NFR7: 性能 - 部门用量看板默认时间范围查询不得依赖逐条扫描全量日志；V1 验收基线为 `logs` 累计 100 万条、部门数 ≤ 100 时，近 30 天部门用量首屏数据 P95 < 3 秒。

NFR8: 国际化 - 新增前端文案必须进入 `web/default/src/i18n/locales/{lang}.json`，以英文 key 作为源字符串，并保持现有多语言同步流程。

Total NFRs: 8

### Additional Requirements

- 使用现有 new-api monorepo 作为 brownfield starter，不初始化新项目或引入第二套应用骨架。
- 企业模块默认通过新增表、新 service、新 controller、新 route 和 wrapper/adapter 集成，避免侵入 relay、billing、user、token、log 等核心链路。
- 企业相关表名统一使用 `enterprise_` 前缀，列名使用 snake_case；仅扩展现有 `subscriptions` 表增加 `source_type` 和 `source_allocation_id` 两列。
- 企业后端包路径使用 `model/enterprise`、`service/enterprise`、`controller/enterprise`、`dto/enterprise`，企业路由统一挂载在 `/api/enterprise/**`。
- 企业 API 归入管理后台 OpenAPI `docs/openapi/api.json`，不得写入 `docs/openapi/relay.json`。
- 不得修改 `/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**` 的协议表面，不向 relay 请求 DTO 添加企业/部门字段。
- 严禁触碰 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`，除非先回到架构层评审。
- 所有 JSON marshal/unmarshal 必须使用 `common/json.go` 中的封装函数。
- 数据库实现必须三库兼容；企业模块不得新增名为 `group` 的组织字段。
- `tenant_id` V1 仅在新增企业相关表中预留，默认值为单租户值；不扩展到 `users`、`logs`、`quota_data` 等旧表。
- 钉钉登录成功后复用现有 session、`setupLogin`、`UserAuth/AdminAuth/RootAuth`，不新增 JWT/session 体系。
- 钉钉 app secret、client secret、access token、suite ticket 不得返回前端、进入日志、导出或错误响应。
- 部门用量看板、详情、风险概览和 CSV 必须展示 `enterprise.usage.multi_dept_disclaimer` 口径说明。
- 后台任务必须通过 `common.IsMasterNode` 守卫，Redis 可增强但无 Redis 时必须依靠 DB 状态表和幂等逻辑可用。
- Default 前端是 V1 企业能力完整交付面；Classic 前端提供必要管理入口但不做同等深度重构。
- 测试必须覆盖三库迁移和查询、部门树三层以上结构、部门移动/停用、钉钉同步幂等、用户冲突匹配、部门预算并发分配/扣减、用量聚合口径、告警通道失败和权限隔离。
- V1 不交付代码库 DeepWiki、通用企业文档库 RAG、WebChat 增强、多模态入口、SaaS 多租户、公网注册、工作空间、租户计费、LDAP、SAML、SCIM、飞书/企业微信登录、私有化授权 License 或专用审计日志表。
- `new-api` 与 `QuantumNous` 是受保护标识，不得删除、替换或改名。

### PRD Completeness Assessment

PRD is materially complete for requirements extraction: it defines 20 FRs, 8 NFRs, user journeys, non-goals, success metrics, assumptions, risks, release gates, and open questions. Remaining caveats: UX exists only as derived requirements rather than a standalone UX design artifact; OQ-1/OQ-2/OQ-3/OQ-4/OQ-5 remain open or deferred, while OQ-6/OQ-7 are closed in V1.1.

## Epic Coverage Validation

### Epic FR Coverage Extracted

FR1: Covered in Epic 1 - 企业部门层级; Story 1.1, Story 1.4

FR2: Covered in Epic 1 - 用户与部门 N:N 成员关系; Story 1.2, Story 1.4

FR3: Covered in Epic 1 - 部门与 `Group` 语义分离; Story 1.3, Story 1.4

FR4: Covered in Epic 1 - SaaS 租户字段预留; Story 1.1, Story 1.2

FR5: Covered in Epic 2 - 钉钉企业应用配置; Story 2.1, Story 2.2

FR6: Covered in Epic 2 - 钉钉 OAuth 登录; Story 2.3

FR7: Covered in Epic 2 - 钉钉部门与成员同步; Story 2.4

FR8: Covered in Epic 2 - 钉钉组织变更处理; Story 2.5

FR9: Covered in Epic 3 - 部门预算池; Story 3.1

FR10: Covered in Epic 3 - wallet 子分配; Story 3.2, Story 3.3, Story 3.4

FR11: Covered in Epic 3 - 计费表达式与倍率语义保留; Story 3.3

FR12: Covered in Epic 3 - 预算池与 wallet 状态展示; Story 3.4, Story 3.5

FR13: Covered in Epic 4 - 部门聚合用量; Story 4.1, Story 4.2

FR14: Covered in Epic 4 - 部门用量详情; Story 4.3

FR15: Covered in Epic 4 - 部门用量 CSV 导出; Story 4.4

FR16: Covered in Epic 4 - 定期用量报告; Story 4.5

FR17: Covered in Epic 5 - 内容风险事件记录; Story 5.1, Story 5.2

FR18: Covered in Epic 5 - 告警规则与通道配置; Story 5.3

FR19: Covered in Epic 5 - 可追溯告警通知; Story 5.4, Story 5.5

FR20: Covered in Epic 5 - 部门风险概览; Story 5.6

Total FRs in epics: 20

### Coverage Matrix

| FR Number | PRD Requirement | Epic Coverage | Status |
| --------- | --------------- | -------------- | ------ |
| FR1 | 建立企业组织与部门层级 | Epic 1; Stories 1.1, 1.4 | Covered |
| FR2 | 维护用户与部门 N:N 平等成员关系 | Epic 1; Stories 1.2, 1.4 | Covered |
| FR3 | 保持部门与现有 `Group` 语义分离 | Epic 1; Stories 1.3, 1.4 | Covered |
| FR4 | 预留 SaaS 租户演进字段或映射策略 | Epic 1; Stories 1.1, 1.2 | Covered |
| FR5 | 配置钉钉企业应用凭据、回调地址、同步开关和同步范围 | Epic 2; Stories 2.1, 2.2 | Covered |
| FR6 | 支持员工通过钉钉 OAuth 登录并读取本地部门快照 | Epic 2; Story 2.3 | Covered |
| FR7 | 执行钉钉部门与成员全量/定时同步 | Epic 2; Story 2.4 | Covered |
| FR8 | 处理钉钉组织变更 | Epic 2; Story 2.5 | Covered |
| FR9 | 创建 `balance` 或 `subscription` 类型部门预算池 | Epic 3; Story 3.1 | Covered |
| FR10 | 以 wallet 子分配方式承接部门预算扣减 | Epic 3; Stories 3.2, 3.3, 3.4 | Covered |
| FR11 | 保留计费表达式和倍率语义 | Epic 3; Story 3.3 | Covered |
| FR12 | 查看预算池及派生 wallet 子分配状态 | Epic 3; Stories 3.4, 3.5 | Covered |
| FR13 | 按部门聚合用量并展示不可加和说明 | Epic 4; Stories 4.1, 4.2 | Covered |
| FR14 | 查看单个部门用量详情 | Epic 4; Story 4.3 | Covered |
| FR15 | 导出部门用量 CSV | Epic 4; Story 4.4 | Covered |
| FR16 | 发送部门用量定期报告 | Epic 4; Story 4.5 | Covered |
| FR17 | 记录内容风险事件 | Epic 5; Stories 5.1, 5.2 | Covered |
| FR18 | 配置内容风险告警规则和通道 | Epic 5; Story 5.3 | Covered |
| FR19 | 发送可追溯告警通知并支持重试/重发 | Epic 5; Stories 5.4, 5.5 | Covered |
| FR20 | 展示部门风险概览 | Epic 5; Story 5.6 | Covered |

### Missing Requirements

No PRD FRs are missing from the epic/story coverage map.

No FRs appear in the epics that are absent from the PRD baseline.

### Coverage Statistics

- Total PRD FRs: 20
- FRs covered in epics: 20
- Coverage percentage: 100%

## UX Alignment Assessment

### UX Document Status

Not found. No standalone whole or sharded UX design document was found under `_bmad-output/planning-artifacts`.

UX is nevertheless implied and required. PRD, Architecture, and Epics define multiple user-facing/admin-facing surfaces: enterprise organization, DingTalk configuration/sync status, department budget allocation, organization usage dashboard, usage detail, CSV export, usage report configuration, alert/risk event management, and Default/Classic frontend entry points.

### Alignment Issues

- No standalone UX artifact exists to define page-level IA, screen states, empty/error/loading states, visual hierarchy, navigation placement, table/filter behavior, or form interaction details for the enterprise management experience.
- PRD and Epics do include executable UX requirements for department usage disclaimer visibility, CSV note behavior, i18n coverage, Default/Classic frontend scope, permissions visibility, and dashboard/report workflows.
- Architecture supports the implied UX needs through `/api/enterprise/**`, Default enterprise feature modules, Classic pages, TanStack Query usage, file export support, i18n conventions, loading/error handling guidance, long-running task polling, and OpenAPI placement.
- Architecture carries the responsiveness requirement for usage dashboards: 100 万 `logs` / ≤100 departments / recent 30 days first screen P95 < 3 seconds.

### Warnings

- WARNING: UX is required but not independently specified. Implementation can proceed only if stories and architecture are treated as the source of UX acceptance, but design consistency risk remains for complex admin surfaces.
- WARNING: Default frontend is the full enterprise management experience while Classic receives necessary entry points only; this product distinction must be made explicit in story acceptance and QA expectations.
- WARNING: The multi-department disclaimer (`enterprise.usage.multi_dept_disclaimer`) is a load-bearing UX requirement and must appear in dashboard, detail, CSV, and risk overview surfaces; it cannot be left to visual polish later.

## Epic Quality Review

### Overall Epic Structure

The epic structure is mostly aligned with create-epics-and-stories best practices:

- Epic 1 delivers enterprise organization governance value rather than a pure setup milestone.
- Epic 2 delivers DingTalk identity and directory sync value using Epic 1 organization entities.
- Epic 3 delivers department budget and wallet allocation value using prior organization membership data.
- Epic 4 delivers usage dashboard/export/reporting value using prior organization membership data and existing logs.
- Epic 5 delivers risk event, alert, and risk overview value using prior organization and usage surfaces.

No epic is a pure "database setup", "API development", or "infrastructure setup" milestone. No Epic N requires Epic N+1 to function.

### Critical Violations

None found at epic level.

### Major Issues

1. Story 1.4 contains a forward dependency on future risk-event functionality.

   Evidence: Story 1.4 is in Epic 1 and includes acceptance text that ordinary users cannot view "other department risk events or admin-only fields". Risk events are introduced in Epic 5. This makes the Story 1.4 acceptance suite partially dependent on a future epic.

   Recommendation: Move risk-event visibility checks to Story 5.1 / 5.2, or reword Story 1.4 to cover only enterprise organization/admin-action audit permissions.

2. Story 3.3 weakens the PRD's quantitative concurrency acceptance.

   Evidence: PRD FR10 requires a measurable concurrent allocation/billing validation baseline: >=50 concurrency, mixed `tiered_expr` and fixed-price models, >=10 minutes, no parent negative balance, and iron-rule consistency. Story 3.3 only says transactions and atomic update prevent over-deduction.

   Recommendation: Add explicit load/concurrency acceptance criteria to Story 3.3 matching PRD FR10, including the required concurrency level, duration, model mix, and final invariants.

3. Story 3.4 omits the precise balance-expiry recovery formula and idempotency acceptance.

   Evidence: PRD FR10 defines balance expiry recovery as `min(wallet.remain_quota_at_expiry, max(0, allocated_quota - consumed))` and requires repeated task runs not to recover twice. Story 3.4 only says the parent pool recovers the unused part "按规则".

   Recommendation: Add the exact recovery formula and idempotent rerun acceptance to Story 3.4.

4. Story 4.1 is implementation-heavy and needs an admin-observable completion boundary.

   Evidence: Story 4.1 is phrased around aggregation tasks and snapshots. It has clear performance AC, but its user-observable result is indirect unless the story also exposes a query/API/state that the dashboard can consume.

   Recommendation: Add acceptance that the generated department usage snapshot can be queried by the enterprise usage API and returns the fields needed by Story 4.2.

### Minor Concerns

- Several ACs use placeholders like "可测试错误", "必要字段差异", "必要追溯摘要", and "可读错误原因". These are directionally clear but should be replaced with named error keys, field lists, or response examples before implementation.
- Epic 1 Story 1.1 should explicitly cover the empty-state path for a newly enabled enterprise module before DingTalk sync exists, so Epic 1 remains independently demonstrable.
- The lack of standalone UX documentation increases reliance on story ACs for page/state behavior; Story 4.2, 4.3, 4.4, 5.6 should keep UX-critical disclaimer placement explicit.

### Best Practices Checklist

| Epic | Delivers User Value | Independent From Future Epics | Stories Sized Reasonably | No Forward Dependencies | Clear ACs | Traceability |
| ---- | ------------------- | ----------------------------- | ------------------------ | ----------------------- | --------- | ------------ |
| Epic 1 | Pass | Pass | Pass | Needs fix in Story 1.4 | Mostly pass | Pass |
| Epic 2 | Pass | Pass | Pass | Pass | Pass | Pass |
| Epic 3 | Pass | Pass | Pass | Pass | Needs stronger quantitative ACs | Pass |
| Epic 4 | Pass | Pass | Mostly pass | Pass | Mostly pass | Pass |
| Epic 5 | Pass | Pass | Pass | Pass | Mostly pass | Pass |

### Quality Recommendations

- Fix Story 1.4 forward reference before implementation starts.
- Restore PRD-level quantitative acceptance into Story 3.3.
- Restore PRD-level expiry/idempotency acceptance into Story 3.4.
- Add a query/API completion boundary to Story 4.1.
- Tighten vague AC wording into concrete error keys, field lists, and observable responses during story finalization.

## Summary and Recommendations

### Overall Readiness Status

NEEDS WORK

The artifacts are close to implementation-ready, but should not be treated as fully ready until the major story-quality issues are fixed. PRD and architecture are materially strong; FR coverage is complete; the remaining risk is that several story-level acceptance criteria do not yet preserve the PRD's most important measurable safeguards.

### Critical Issues Requiring Immediate Action

No critical epic-level blockers were found.

Major issues requiring action before implementation:

1. Story 1.4 references risk-event visibility before risk events exist in Epic 5. This is a forward dependency and should be moved or reworded.
2. Story 3.3 does not carry PRD FR10's quantitative concurrency/load acceptance for department budget wallet allocation and billing invariants.
3. Story 3.4 does not carry PRD FR10's exact balance-expiry recovery formula or idempotency requirement.
4. Story 4.1 needs a clear observable completion boundary: generated usage snapshots must be queryable by the enterprise usage API and sufficient for the dashboard.
5. No standalone UX document exists despite multiple complex admin surfaces; implementation must either create a lightweight UX spec or strengthen UX acceptance directly in stories.

### Recommended Next Steps

1. Patch `epics.md` Story 1.4, Story 3.3, Story 3.4, and Story 4.1 before starting implementation.
2. Add a compact UX specification or story-level UX acceptance for the enterprise navigation, list/detail/filter states, empty/error/loading states, and disclaimer placement.
3. Replace vague AC terms such as "可测试错误", "必要字段差异", and "可读错误原因" with concrete error keys, response fields, or examples.
4. Treat PRD FR10, NFR1, NFR2, NFR5, and NFR7 as non-negotiable implementation gates: concurrency, three-database compatibility, secret handling, consistency, and dashboard performance.
5. Keep the architecture guardrails intact: no relay DTO changes, no `relay/**` modifications, enterprise APIs only under `/api/enterprise/**`, enterprise OpenAPI only in `docs/openapi/api.json`.

### Issue Count

This assessment identified 8 issues across 4 categories:

- Document discovery/validation: 1 warning (no standalone UX document).
- UX alignment: 3 warnings.
- Epic quality: 4 major/minor issues.
- FR coverage: 0 missing FRs.

### Final Note

The implementation plan is directionally sound and traceable: PRD has 20 FRs and 8 NFRs, epics cover all 20 FRs, and architecture supports the intended backend/frontend split. The project should proceed to implementation only after the story-level acceptance gaps above are corrected, because those gaps are exactly where budget consistency, concurrency safety, and UX consistency can otherwise be lost.

**Assessment Date:** 2026-05-28

**Assessor:** BMAD Implementation Readiness workflow
