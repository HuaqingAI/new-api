---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - "_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md"
  - "_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/addendum.md"
  - "_bmad-output/planning-artifacts/architecture.md"
  - "_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md"
  - "_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/addendum.md"
---

# new-api - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for new-api, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

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

FR21: 系统建立部门负责人权限模型，负责人来源同时支持钉钉同步与手动指定；两者并存时以本地显式覆盖结果作为最终生效权限。

FR22: 系统在预算 summary、用量 summary、告警 summary 等查询/汇总页面支持“包含子部门”作用域，第一版优先覆盖 summary / overview 视图。

FR23: 系统支持上级部门预算池向任意后代部门预算池分配额度，不限制为直属下级，并继续复用既有 `balance` / `subscription` 两类预算池能力。

FR24: 系统为成员额度分配补齐调整、取消和回收能力；调整优先按“关闭旧分配 + 新建新分配”建模，取消不做硬删除，保留完整状态语义与审计链路。

FR25: 当员工额度不足时，允许其在所属部门上下文中发起额度申请；员工必须选择目标部门和基于现有企业预算池能力的模式或目标池，由部门负责人执行单步审批，审批通过后系统自动复用现有预算分配链路完成分配。

FR26: 系统对部门负责人授权、跨层预算委派、成员 allocation 调整/取消/回收、员工申请审批、自动分配和回收等动作形成统一审计与钉钉通知闭环。

### NonFunctional Requirements

NFR1: 兼容性 - 新增数据库对象和查询必须同时兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+。

NFR2: 安全 - 钉钉应用密钥、Webhook、邮箱凭据等敏感配置不得明文返回前端，不得写入普通日志。

NFR3: 隐私 - 内容风险事件默认只展示追溯所需摘要，不扩大现有日志中敏感内容的可见范围。

NFR4: 可靠性 - 组织同步失败不得影响现有登录、API 调用、计费和日志链路。

NFR5: 一致性 - 部门预算 wallet 子分配的扣减必须与用户 quota 消费最终一致；铁律 `Σ children.committed_quota ≤ parent.total_quota / cycle_quota` 必须在分配/扩容前强制校验，不能出现父池负余额或用户扣费成功但部门 wallet 未同步扣减。

NFR6: 可观测性 - 同步任务、配额拦截、告警发送必须有后台可查状态和错误原因。

NFR7: 性能 - 部门用量看板默认时间范围查询不得依赖逐条扫描全量日志；V1 验收基线为 `logs` 累计 100 万条、部门数 ≤ 100 时，近 30 天部门用量首屏数据 P95 < 3 秒。

NFR8: 国际化 - 新增前端文案必须进入 `web/default/src/i18n/locales/{lang}.json`，以英文 key 作为源字符串，并保持现有多语言同步流程。

### Additional Requirements

- 使用现有 new-api monorepo 作为 brownfield starter；不得初始化新项目或引入第二套应用骨架。
- 企业模块默认通过新增表、新 service、新 controller、新 route 和 wrapper/adapter 集成，避免侵入 relay、billing、user、token、log 等核心链路。
- 企业相关表名统一使用 `enterprise_` 前缀，列名使用 snake_case；唯一例外是扩展现有 `subscriptions` 表增加 `source_type` 和 `source_allocation_id` 两列。
- 企业后端包路径使用 `model/enterprise`、`service/enterprise`、`controller/enterprise`、`dto/enterprise`，企业路由统一挂载在 `/api/enterprise/**`。
- 企业 API 归入管理后台 OpenAPI `docs/openapi/api.json`，不得写入 `docs/openapi/relay.json`。
- 不得修改 `/v1/**`、`/v1beta/**`、`/mj/**`、`/suno/**` 的协议表面，不向 relay 请求 DTO 添加企业/部门字段。
- 严禁触碰 `relay/**`、`pkg/billingexpr/**`、`pkg/cachex/**`、`model/log.go`、`controller/log.go`、`router/relay-router.go`、`docs/openapi/relay.json`、所有 `relay/channel/**`，除非先回到架构层评审。
- 所有 JSON marshal/unmarshal 必须使用 `common/json.go` 中的封装函数，不得在业务代码中直接调用 `encoding/json` marshal/unmarshal。
- 数据库实现必须三库兼容；保留字列处理沿用 `commonGroupCol`、`commonKeyCol`，企业模块不得新增名为 `group` 的组织字段。
- `tenant_id` V1 仅在新增企业相关表中预留，默认值为单租户值；不扩展到 `users`、`logs`、`quota_data` 等旧表。
- 部门层级 V1 使用邻接表建模；如性能需要可增加路径缓存字段，V1 不引入闭包表。
- 钉钉登录、access token、通讯录同步、部门成员关系、冲突处理和同步状态属于专用 DingTalk enterprise integration module，不以 `custom_oauth_providers` 作为主模型。
- 钉钉登录成功后复用现有 session、`setupLogin`、`UserAuth/AdminAuth/RootAuth`，不新增 JWT/session 体系。
- 钉钉身份绑定优先使用稳定身份标识；邮箱/手机号仅作为无冲突时辅助匹配，冲突进入待处理并由管理员确认。
- Root 管理钉钉密钥和系统级企业配置；Admin 管理全局企业能力；部门管理员权限放入新增部门角色/成员关系表，不扩展 `users.role` 枚举。
- 钉钉 app secret、client secret、access token、suite ticket 不得返回前端、进入日志、导出或错误响应。
- 部门预算池采用 `balance` 和 `subscription` 双类型；分配预算时必须在目标用户名下创建独立 wallet，落地为 `subscriptions` 新行。
- 父池和子 wallet 必须遵守铁律；部门预算池剩余额度变更使用原子 update 和防超扣 WHERE 条件。
- `enterprise_quota_allocations` 是配额分配与撤销的单一审计来源，不再向 `enterprise_admin_actions` 双写。
- 其他低频企业管理动作（部门 CRUD、钉钉配置、告警规则等）通过统一 `enterprise_admin_actions` 记录。
- 部门分配 wallet 默认排在用户主钱包之前；用户可调整顺序但不能删除，删除/撤销只能由管理员执行。
- 父预算池暂停、过期或撤销时，子 wallet 状态由 `wallet_state_sync_task` 联动，不得在 controller 层逐个手工更新。
- `balance` 型过期回收由 `balance_expiry_task` 处理，必须幂等，避免重复回收。
- 部门用量归因采用成员消费视角，运行时或聚合任务按 `enterprise_user_departments` join 展开，多部门用户重复计入；不得取单一部门或按 user_id 唯一归因。
- 未归属用户消费单独归入 `dept_id = NULL`，看板显示为“未归属”，不计入任何部门风险率分母。
- 部门用量看板、详情、风险概览和 CSV 必须展示 `enterprise.usage.multi_dept_disclaimer` 口径说明。
- 部门负责人来源同时支持钉钉同步与手动指定，推荐优先级为 `manual_deny_override > manual_grant > dingtalk_synced_owner`。
- “包含子部门”第一版优先落在 `usage / alerts / budget summary` 等汇总视图，不要求把所有 detail list 扩成整棵子树。
- 上级预算池可以向任意后代部门预算池分配，但仍只允许 `balance / subscription` 两类预算池，不新增第三类预算类型。
- 成员 allocation 调整应优先按“关闭旧分配 + 新建新分配”建模；取消不做硬删除，保留 `active / superseded / revoked / expired / closed` 等状态语义。
- 员工额度申请只做单步审批，不做多人会签、代理审批或复杂升级链；审批通过后的自动分配继续复用现有 allocation / wallet 生命周期规则。
- 审计与通知需覆盖授权、预算下拨、成员分配调整、申请提交、审批结果、自动分配和回收；通知失败不得阻塞核心治理事务。
- 通讯录同步、报表聚合、定期邮件和告警投递采用后台 job + 状态表；V1 不引入消息队列作为必需依赖。
- 后台任务必须通过 `common.IsMasterNode` 守卫，避免多节点重复执行；Redis 可增强锁、缓存、限流和去重，但无 Redis 时必须依靠 DB 状态表和幂等逻辑可用。
- 控制台 API 响应沿用 `{ success, message, data }` 与 `common.ApiSuccess/ApiError/ApiSuccessI18n/ApiErrorI18n`；业务错误继续按现有惯例返回 HTTP 200。
- DTO 可选标量字段一律使用指针类型和 `omitempty`，避免 `0` 或 `false` 被误吞。
- JSON 时间字段使用 `int64` Unix 秒；数组字段缺省返回 `[]` 而非 `null`。
- Default 前端是 V1 企业能力完整交付面，新增 `enterprise-organization`、`enterprise-dingtalk`、`enterprise-usage`、`enterprise-alerts` 等 feature。
- Default 前端使用 TanStack Query，queryKey 统一为 `['enterprise', '<feature>', '<scope>', ...]`，mutation invalidate 必须精确到 feature 级。
- Classic 前端提供必要管理入口，但不做与 Default 同等深度的重构；遵循 `pages/`、`components/`、Semi Design 既有模式。
- 不要把企业能力塞入现有 Group、User、Log 页面；应新增企业组织/组织管理模块，并在内部承载组织架构、钉钉同步、配额分配、用量报表和风险告警。
- 企业 i18n 后端 key 统一加 `enterprise.<feature>.<message>` 前缀；Default 前端继续使用英文源字符串作为 key。
- 测试必须覆盖三库迁移和查询、部门树三层以上结构、部门移动/停用、钉钉同步幂等、用户冲突匹配、部门预算并发分配/扣减、用量聚合口径、告警通道失败、权限隔离。
- V1 不交付代码库 DeepWiki、通用企业文档库 RAG、WebChat 增强、多模态生成入口、SaaS 多租户、公网注册、工作空间、租户计费、LDAP、SAML、SCIM、飞书/企业微信登录、私有化授权 License 或专用审计日志表。
- `new-api` 与 `QuantumNous` 是受保护标识，任何 README、元数据、UI、文档、导入路径、部署配置或注释中不得删除、替换、改名。

### UX Design Requirements

无独立 UX Design 文档。可执行 UX 要求已从 PRD 和 Architecture 提取到 FR13、FR14、FR15、FR20、NFR8 与 Additional Requirements 中，尤其包括部门间不可加和提示、CSV 注释、Default/Classic 管理入口、i18n 和权限可见性要求。

### FR Coverage Map

FR1: Epic 1 - 企业部门层级。
FR2: Epic 1 - 用户与部门 N:N 成员关系。
FR3: Epic 1 - 部门与 `Group` 语义分离。
FR4: Epic 1 - SaaS 租户字段预留。
FR5: Epic 2 - 钉钉企业应用配置。
FR6: Epic 2 - 钉钉 OAuth 登录。
FR7: Epic 2 - 钉钉部门与成员同步。
FR8: Epic 2 - 钉钉组织变更处理。
FR9: Epic 3 - 部门预算池。
FR10: Epic 3 - wallet 子分配。
FR11: Epic 3 - 计费表达式与倍率语义保留。
FR12: Epic 3 - 预算池与 wallet 状态展示。
FR13: Epic 4 - 部门聚合用量。
FR14: Epic 4 - 部门用量详情。
FR15: Epic 4 - 部门用量 CSV 导出。
FR16: Epic 4 - 定期用量报告。
FR17: Epic 5 - 内容风险事件记录。
FR18: Epic 5 - 告警规则与通道配置。
FR19: Epic 5 - 可追溯告警通知。
FR20: Epic 5 - 部门风险概览。
FR21: Epic 7 - 部门负责人权限模型。
FR22: Epic 7 - 包含子部门的查询作用域。
FR23: Epic 7 - 上级预算池向下级部门预算池分配。
FR24: Epic 7 - 成员 allocation 调整 / 取消 / 回收。
FR25: Epic 7 - 员工额度申请与单步审批。
FR26: Epic 7 - 审计与钉钉通知闭环。

## Epic List

### Epic 1: 企业组织与部门治理基线

管理员可以在系统中获得企业组织、部门层级、用户部门关系、部门权限和审计基础，并确保部门不破坏现有 `Group` 计费语义。

**FRs covered:** FR1, FR2, FR3, FR4

### Epic 2: 钉钉企业身份接入与通讯录同步

IT 管理员可以配置钉钉企业应用、同步组织通讯录，员工可以用钉钉登录并获得正确部门归属。

**FRs covered:** FR5, FR6, FR7, FR8

### Epic 3: 部门预算池与成员 Wallet 分配

管理员可以为部门创建预算池，把预算安全分配给成员 wallet，并查看预算池与子钱包状态，同时不改 relay 计费链路。

**FRs covered:** FR9, FR10, FR11, FR12

### Epic 4: 组织用量看板、详情与报告导出

管理层和运营人员可以按部门查看用量、模型分布、用户排行，导出 CSV，并接收定期报告，同时明确多部门用户重复计入口径。

**FRs covered:** FR13, FR14, FR15, FR16

### Epic 5: 内容风险事件、告警与部门风险概览

安全/合规负责人可以追踪内容风险事件，配置告警规则，接收可追溯通知，并查看部门风险趋势。

**FRs covered:** FR17, FR18, FR19, FR20

### Epic 6: 企业治理视图重构与账号标识修正

管理员可以通过树驱动的治理视图完成企业组织、预算和用量管理，并确保企业账号标识在登录、查询、统计和分配场景中可读、可维护且一致。

**FRs covered:** follow-up to FR1, FR2, FR6, FR10, FR12, FR13, FR14

### Epic 7: 分层预算治理与额度审批

管理员与部门负责人可以在既有企业组织、预算、用量和告警基础上，完成跨层预算治理、成员额度治理、员工额度申请审批和治理动作审计通知，形成企业板块的治理闭环。

**FRs covered:** FR21, FR22, FR23, FR24, FR25, FR26

### Story 6.1: 企业组织页重构为树驱动治理视图

**Requirements:** follow-up to FR1, FR2, FR12

As a 管理员,
I want 企业组织页以“左侧可折叠组织树 + 右侧当前部门工作区”的结构工作,
So that 我可以围绕当前部门连续完成成员、预算和治理操作。

**Acceptance Criteria:**

**Given** 管理员进入企业组织页
**When** 页面加载完成
**Then** 页面左侧显示可折叠组织树
**And** 右侧显示当前选中部门的摘要、成员列表、预算/资源池摘要和相关操作区。

**Given** 管理员在组织树中选中某个部门
**When** 页面状态更新
**Then** 当前选中部门成为页面一级状态
**And** 右侧所有摘要、列表和操作都切换到该部门上下文，而不是进入独立 detail drill-down 页面。

**Given** 页面存在当前选中部门
**When** 管理员刷新页面或复制当前页面链接
**Then** 系统能够恢复当前选中部门
**And** 默认展开根节点与当前选中部门的祖先路径。

### Story 6.2: 部门用量页重构为树驱动分析视图

**Requirements:** follow-up to FR13, FR14

As a 管理员或经营负责人,
I want 部门用量页以“左侧可折叠组织树 + 右侧当前部门分析区”的结构工作,
So that 我可以围绕当前部门持续分析用量，而不是先看总表再跳详情。

**Acceptance Criteria:**

**Given** 管理员进入部门用量页
**When** 页面加载完成
**Then** 页面左侧显示可折叠组织树
**And** 右侧显示当前选中部门的用量概览、趋势、模型分布、用户排行和最近日志入口。

**Given** 管理员查看当前部门分析区
**When** 页面渲染完成
**Then** 右侧除用量数据外，还包含当前部门维度下的说明或配置区域
**And** 不再把“总览”和“配置”拆成割裂的独立主视图。

**Given** 管理员在左侧切换当前部门
**When** 页面状态更新
**Then** 当前选中部门仍是页面一级状态
**And** 右侧分析区在不离开页面主骨架的前提下切换到新的部门上下文。

### Story 6.3: 部门上下文驱动的成员与预算操作

**Requirements:** follow-up to FR2, FR10, FR12

As a 管理员,
I want 从当前选中部门直接进入成员维护、预算池查看和 wallet 分配,
So that 我不需要再依赖输入 Department ID 或 User ID 才能完成治理操作。

**Acceptance Criteria:**

**Given** 管理员已选中某个部门
**When** 管理员查看该部门右侧工作区
**Then** 可以直接进入该部门成员维护、预算池列表和 wallet 分配操作
**And** 这些操作都继承当前部门上下文。

**Given** 管理员需要向成员分配预算
**When** 管理员从当前部门工作区发起分配
**Then** 系统默认通过当前部门成员列表或搜索选择目标成员
**And** `Department ID` / `User ID` 不再作为主界面默认输入路径。

**Given** 管理员需要查看某成员在当前部门的治理信息
**When** 从成员列表进入操作
**Then** 系统能在当前部门上下文中承接成员归属、预算分配和相关操作
**And** 不要求管理员跳回全局用户页或手工填写对象 ID。

### Story 6.4: 钉钉账号 username 策略与受控修改

**Requirements:** follow-up to FR6

As a 管理员,
I want 企业账号默认 username 更可读，并支持受控修改,
So that 登录、查询、统计、预算分配和运维识别不会长期暴露不可读账号名。

**Acceptance Criteria:**

**Given** 钉钉同步或 OAuth 创建新用户
**When** 系统生成默认 username
**Then** 系统优先生成可读且唯一的 username
**And** 不再以 `dt_编号` 作为主要结果。

**Given** 企业账号已创建
**When** 有权限的管理员执行 username 修改
**Then** 系统支持受控修改 username
**And** 明确校验唯一性、保留审计记录，并处理相关缓存刷新与后续查询联动。

**Given** username 已被修改
**When** 页面、查询、日志入口和企业视图再次读取该用户
**Then** 系统按照更新后的规则展示与筛选该用户
**And** 历史日志与历史统计是否保留原 username 快照必须有明确规则并可验证。

### Story 6.5: 企业页面统一用户标识展示规则

**Requirements:** follow-up to FR14, FR17, FR20

As a 管理员,
I want 企业页面统一按“可读名称优先，username / ID 兜底”展示用户,
So that 我在不同页面看到的是一致且可识别的对象。

**Acceptance Criteria:**

**Given** 管理员查看成员列表、预算分配、wallet 明细、用量排行、日志入口或风险事件
**When** 页面展示用户标识
**Then** 系统优先展示可读名称
**And** 在需要消歧时补充 username 或 user ID 作为兜底信息。

**Given** 用户没有可用的可读名称
**When** 页面渲染用户标识
**Then** 系统回退到 username
**And** 在 username 仍不足以消歧时显示 user ID。

**Given** 同一用户出现在多个企业页面
**When** 管理员跨页面查看该用户
**Then** 组织页、预算页、用量页和风险页遵循同一展示规则
**And** Default 前端优先完整落地该规则，Classic 主题至少保持语义一致。

### Story 7.1: 建立部门负责人权限模型与生效优先级

**Requirements:** FR21

As a 管理员,
I want 同时使用钉钉同步负责人和手动指定负责人，并明确最终生效优先级,
So that 部门治理权限既能跟随组织事实，又能被企业本地规则覆盖。

**Acceptance Criteria:**

**Given** 某部门同时存在钉钉同步负责人和本地手动授权记录
**When** 系统计算该部门的有效负责人集合
**Then** 按 `manual_deny_override > manual_grant > dingtalk_synced_owner` 的顺序得出生效结果
**And** 原始来源事实与覆盖结果都可审计。

**Given** 某负责人被本地显式禁用
**When** 该用户尝试执行该部门预算治理或审批动作
**Then** 系统拒绝其操作
**And** 错误与审计中能说明其被本地规则覆盖。

**Given** 某部门不存在任何有效负责人
**When** 员工向该部门发起额度申请或管理员触发需负责人处理的治理动作
**Then** 系统明确提示“无有效负责人”
**And** 可按规则回退给 Admin 处理。

### Story 7.2: 为预算/用量/告警 summary 增加包含子部门作用域

**Requirements:** FR22

As a 管理员或经营负责人,
I want 在汇总视图中切换“仅当前部门 / 包含子部门”作用域,
So that 我可以按治理树层级查看预算、用量和风险概览。

**Acceptance Criteria:**

**Given** 管理员位于预算 summary、用量 summary 或告警 summary 页面
**When** 切换 `include_descendants=true`
**Then** 系统返回当前部门及其全部后代部门的聚合结果
**And** 页面明确展示当前作用域。

**Given** 页面未显式指定子部门作用域
**When** 首次加载
**Then** 默认保持“仅当前部门”
**And** 与既有口径保持兼容。

**Given** 管理员查看 detail list 或逐笔明细
**When** 页面处于 V1.3 首轮交付
**Then** 不要求自动扩展整棵子树
**And** 该限制在文案或交互上可被理解。

### Story 7.3: 支持上级预算池向后代部门预算池分配

**Requirements:** FR23

As a 部门负责人,
I want 把本部门预算池中的额度继续分配给后代部门预算池,
So that 下级部门可以在自己的治理范围内自主继续分配给成员。

**Acceptance Criteria:**

**Given** 某负责人管理祖先部门且该部门存在可用预算池
**When** 其选择某个后代部门作为目标
**Then** 系统允许从来源预算池向目标部门预算池分配额度
**And** 不要求目标部门必须是直属下级。

**Given** 目标部门不是来源部门的后代节点
**When** 负责人尝试发起预算分配
**Then** 系统拒绝操作
**And** 返回可测试的树关系错误。

**Given** 已存在一条部门到部门的预算委派记录
**When** 管理员调整该委派额度
**Then** 系统优先关闭旧委派并创建新委派
**And** 不静默覆盖历史分配事实。

### Story 7.4: 补齐成员 allocation 的调整、取消与回收治理

**Requirements:** FR24

As a 部门负责人,
I want 对已分配给成员的 allocation 执行调整、取消和回收,
So that 成员额度分配具备完整的后续治理能力。

**Acceptance Criteria:**

**Given** 某成员已有有效 allocation
**When** 负责人执行额度调整
**Then** 系统关闭旧 allocation 并新建新 allocation
**And** 历史记录与状态链保持可追溯。

**Given** 负责人取消某个成员 allocation
**When** 取消生效
**Then** 系统不物理删除 wallet 或 allocation
**And** 记录 `revoked`、`closed` 或等价状态语义。

**Given** 系统对某 allocation 执行回收
**When** 回收成功
**Then** 记录回收额度、操作者、处理时间和来源 allocation
**And** 幂等重试不会产生双重回收。

### Story 7.5: 支持员工额度申请、单步审批与自动分配

**Requirements:** FR25

As a 员工,
I want 在额度不足时向目标部门提交申请并获得审批,
So that 我可以通过正式治理链路获得继续工作的额度。

**Acceptance Criteria:**

**Given** 员工属于一个或多个部门
**When** 其发起额度申请
**Then** 必须显式选择目标部门与目标预算池模式/池子
**And** 系统按目标部门的有效负责人集合路由审批。

**Given** 负责人收到额度申请
**When** 执行审批
**Then** 审批为单步完成
**And** 不需要多人会签、代理审批或升级审批链。

**Given** 审批通过
**When** 系统执行后续处理
**Then** 直接复用现有预算分配链路完成自动分配
**And** 重复审批或重复回调不会创建重复 allocation。

### Story 7.6: 建立治理动作审计时间线与钉钉通知

**Requirements:** FR26

As a 管理员或负责人,
I want 所有治理动作都可追溯并能通过钉钉通知相关人员,
So that 企业预算治理形成完整闭环。

**Acceptance Criteria:**

**Given** 系统发生负责人授权、预算委派、成员 allocation 调整/取消/回收、申请提交或审批结果
**When** 动作提交成功
**Then** 系统记录统一审计事实
**And** 审计至少包含操作者、目标对象、额度变化、时间、状态与追溯标识。

**Given** 员工申请提交、审批通过、审批拒绝、分配成功或回收/撤销发生
**When** 通知通道可用
**Then** 系统发送对应钉钉通知
**And** 通知内容能帮助接收者定位到治理动作结果。

**Given** 钉钉通知发送失败
**When** 核心治理事务已经提交
**Then** 事务结果保持成功
**And** 系统记录投递失败状态与错误原因以供重试。

## Epic 1: 企业组织与部门治理基线

管理员可以在系统中获得企业组织、部门层级、用户部门关系、部门权限和审计基础，并确保部门不破坏现有 `Group` 计费语义。

### Story 1.1: 查看企业部门树

**Requirements:** FR1, FR4

As a 管理员,
I want 查看企业部门树和部门状态,
So that 我可以确认组织结构已被系统正确承载。

**Acceptance Criteria:**

**Given** 企业模块已启用
**When** 管理员打开企业组织页面
**Then** 系统显示部门树、部门名称、父子关系、状态、外部来源标识和同步异常状态
**And** 支持至少 3 层部门结构，部门停用或删除后不物理删除，历史名称可追溯。

**Given** 企业模块已启用但尚未完成任何部门同步或手工创建
**When** 管理员打开企业组织页面
**Then** API 返回空数组而非 `null`
**And** Default 前端显示空状态和可执行入口，引导有权限的管理员进入钉钉配置或同步流程。

**Given** 新增企业部门表
**When** 执行迁移
**Then** 表结构兼容 SQLite、MySQL 5.7.8+、PostgreSQL 9.6+
**And** 表名使用 `enterprise_` 前缀，预留 `tenant_id`，不新增 `group` 字段。

### Story 1.2: 维护用户与部门 N:N 成员关系

**Requirements:** FR2, FR4

As a 管理员,
I want 查看和维护用户所属的多个部门关系,
So that 一个用户可以被多个部门独立治理和统计。

**Acceptance Criteria:**

**Given** 已存在用户和部门
**When** 管理员查看用户详情或部门成员列表
**Then** 系统展示该用户的全部部门成员关系
**And** 不存在主部门字段或唯一部门假设。

**Given** 用户没有任何部门成员关系
**When** 管理员查看组织维度归属
**Then** 系统显示该用户为“未归属”
**And** 不影响现有用户登录、API 调用、计费和 `User.Group`。

### Story 1.3: 保持部门权限与 Group 语义分离

**Requirements:** FR3

As a 系统管理员,
I want 部门权限和现有 `Group` 计费体系互不干扰,
So that 企业组织能力不会破坏现有模型、渠道和倍率行为。

**Acceptance Criteria:**

**Given** 用户属于一个或多个部门
**When** 用户部门关系发生变化
**Then** `User.Group`、`Token.Group`、渠道分组和模型倍率行为保持不变
**And** relay 请求 DTO、计费表达式和模型定价变量不增加部门字段。

**Given** 部门管理员权限被配置
**When** 部门管理员访问企业组织接口
**Then** 后端按企业权限服务校验可管理部门范围
**And** 前端导航/按钮可见性只作为辅助，不作为权限来源。

### Story 1.4: 记录企业组织管理审计

**Requirements:** FR1, FR2, FR3

As a Root 或管理员,
I want 查看企业组织相关管理动作,
So that 部门、成员和权限变更具备可追溯记录。

**Acceptance Criteria:**

**Given** 管理员执行部门、成员关系或部门权限变更
**When** 操作成功保存
**Then** 系统写入 `enterprise_admin_actions` 审计记录
**And** 记录包含 actor、动作类型、对象、时间、必要字段差异，不写入敏感凭据。

**Given** 管理员查看企业组织操作记录
**When** 按时间、动作类型或对象筛选
**Then** 系统返回分页列表
**And** 每条记录至少包含 action_id、actor_id、action_type、object_type、object_id、created_at 和 diff_summary。

**Given** 普通用户或无企业管理权限用户访问企业组织操作记录
**When** 请求 `enterprise_admin_actions` 列表或详情
**Then** 系统拒绝访问并返回 `error.enterprise.permission.admin_required`
**And** 不暴露管理员专用审计字段。

## Epic 2: 钉钉企业身份接入与通讯录同步

IT 管理员可以配置钉钉企业应用、同步组织通讯录，员工可以用钉钉登录并获得正确部门归属。

### Story 2.1: 配置钉钉企业应用

**Requirements:** FR5

As a Root 管理员,
I want 配置钉钉应用凭据、回调地址和同步范围,
So that 系统可以安全接入企业钉钉身份源。

**Acceptance Criteria:**

**Given** Root 管理员打开钉钉配置页面
**When** 填写 app key、app secret、corp id、回调地址、同步范围和启用开关
**Then** 系统保存配置并隐藏敏感字段
**And** API 响应、前端状态、日志和导出文件中不出现明文密钥。

**Given** 缺失必要凭据或配置格式错误
**When** 管理员保存或启用钉钉登录/同步
**Then** 系统拒绝启用并返回可区分错误
**And** 错误不透传钉钉原始敏感信息。

### Story 2.2: 测试钉钉连通性与权限

**Requirements:** FR5

As a Root 管理员,
I want 测试钉钉应用连通性和通讯录权限,
So that 我能在启用前发现配置或权限问题。

**Acceptance Criteria:**

**Given** 已保存钉钉配置
**When** 管理员执行连通性测试
**Then** 系统返回 `auth_success`、`auth_invalid_credentials`、`auth_permission_insufficient`、`network_unreachable` 或 `callback_misconfigured` 之一
**And** 前端按错误码展示对应引导文案。

**Given** 钉钉接口超时或返回错误
**When** 连通性测试结束
**Then** 系统记录摘要错误原因
**And** 不把 app secret、access token 或原始敏感响应写入日志。

### Story 2.3: 支持钉钉 OAuth 登录与账号绑定

**Requirements:** FR6

As a 企业员工,
I want 使用钉钉登录系统,
So that 我可以用企业身份进入 new-api 并获得正确部门归属。

**Acceptance Criteria:**

**Given** 钉钉登录已启用且员工在允许范围内
**When** 员工完成钉钉 OAuth 回调
**Then** 系统绑定已有账号或按配置创建新账号
**And** 登录成功后复用现有 session、`setupLogin`、`UserAuth/AdminAuth/RootAuth`。

**Given** 最近一次成功同步中存在该员工部门成员关系快照
**When** 员工钉钉登录成功
**Then** 系统使用本地快照读取该员工全部部门成员关系
**And** 不在登录路径直接写入钉钉回调返回的部门列表。

**Given** 员工被停用或不在允许同步范围内
**When** 员工尝试钉钉登录
**Then** 系统拒绝登录并给出原因
**And** 不影响已有非钉钉登录方式。

### Story 2.4: 执行钉钉部门与成员同步

**Requirements:** FR7

As a 管理员,
I want 手动或定时同步钉钉部门与成员,
So that 本地组织结构和成员关系能跟随钉钉更新。

**Acceptance Criteria:**

**Given** 钉钉配置可用
**When** 管理员触发全量同步
**Then** 系统创建同步任务并返回任务状态
**And** 前端不长时间阻塞请求，可查看任务进度和结果。

**Given** 同步任务执行完成
**When** 管理员查看同步结果
**Then** 系统展示新增、更新、停用、失败、跳过数量
**And** 同步日志可按时间、状态和对象类型筛选。

**Given** 同一钉钉部门或成员被重复同步
**When** 同步任务重试或定时任务再次运行
**Then** 系统幂等更新，不重复创建部门、用户或成员关系
**And** 失败项记录错误原因且不回滚已成功的无关对象。

### Story 2.5: 处理同步冲突与组织变更

**Requirements:** FR8

As a 管理员,
I want 处理钉钉账号匹配冲突和组织变更,
So that 错绑账号、部门变更和离职不会污染本地治理数据。

**Acceptance Criteria:**

**Given** 钉钉成员无法通过稳定身份自动匹配本地账号
**When** 邮箱或手机号存在冲突
**Then** 系统将该成员标记为待处理冲突
**And** 不自动绑定到可能错误的账号。

**Given** 钉钉部门改名、移动、停用或删除
**When** 同步任务处理变更
**Then** 本地部门状态和父子关系按规则更新
**And** 历史用量仍可按旧部门名称或名称快照追溯。

**Given** 成员离职或转部门
**When** 同步任务处理成员变更
**Then** 系统更新成员关系状态
**And** 不静默删除或篡改已分配 wallet 子钱包，需标记待处理或交由预算回收策略处理。

## Epic 3: 部门预算池与成员 Wallet 分配

管理员可以为部门创建预算池，把预算安全分配给成员 wallet，并查看预算池与子钱包状态，同时不改 relay 计费链路。

### Story 3.1: 创建部门预算池

**Requirements:** FR9

As a 管理员,
I want 为部门创建 balance 或 subscription 类型预算池,
So that 部门可以承接可治理的 AI 额度。

**Acceptance Criteria:**

**Given** 管理员进入部门预算页面
**When** 为部门创建 `balance` 型预算池并填写 quota 和可选 `expires_at`
**Then** 系统保存一次性预算池
**And** 预算池不周期重置，类型创建后不可切换。

**Given** 管理员创建 `subscription` 型预算池
**When** 填写 cycle_type、cycle_quota、起始时间和必要的 custom_seconds
**Then** 系统保存周期预算池
**And** 周期语义与现有用户 subscription 同构。

**Given** 预算池保存请求包含非法 quota、cycle_quota 或周期配置
**When** 管理员提交
**Then** 系统拒绝保存并返回可测试错误
**And** 记录低频管理操作审计。

### Story 3.2: 将部门预算分配为成员 wallet

**Requirements:** FR10

As a 管理员,
I want 从部门预算池向成员分配 wallet,
So that 成员 API 消费可以优先使用部门分配额度。

**Acceptance Criteria:**

**Given** 部门预算池有可分配额度且目标用户属于该部门
**When** 管理员提交分配额度
**Then** 系统在目标用户名下创建一行 `subscriptions` 子钱包
**And** 子钱包带 `source_type='enterprise_allocation'` 与 `source_allocation_id`。

**Given** 创建 wallet 子分配
**When** 系统写入分配结果
**Then** `enterprise_quota_allocations` 记录父预算池、目标用户、目标 wallet、金额、类型快照、周期/过期快照、操作者和状态
**And** 该记录是配额分配审计单一来源，不再双写 `enterprise_admin_actions`。

**Given** 用户已有主钱包或自有钱包
**When** 部门 wallet 创建成功
**Then** 部门 wallet 默认排在主钱包之前参与现有 wallet 优先级
**And** 用户可调整顺序但不能自行删除部门 wallet。

### Story 3.3: 强制预算铁律与并发安全

**Requirements:** FR10, FR11

As a 管理员,
I want 系统阻止超额分配和并发穿透,
So that 部门预算不会出现父池负余额或重复分配。

**Acceptance Criteria:**

**Given** 管理员创建、扩容或变更子分配
**When** `Σ children.committed_quota + new_quota` 超过父池 `total_quota` 或 `cycle_quota`
**Then** 系统拒绝操作
**And** 返回明确的预算超额错误。

**Given** 多个管理员或后台任务并发分配同一父预算池
**When** ≥50 个分配请求同时提交，且请求总额超过父池 `total_quota` 或 `cycle_quota`
**Then** 系统只允许不违反铁律的请求成功
**And** 失败请求返回 `enterprise_budget_insufficient`，并区分 `balance_remaining_insufficient` 与 `subscription_cycle_allocated_exceeded`
**And** 数据库中不出现父池负余额、重复 wallet、重复 allocation 或 `Σ children.committed_quota` 超过父池额度的状态。

**Given** 已存在部门预算子 wallet，且系统同时处理固定计费模型与 `tiered_expr` 模型请求
**When** ≥50 并发负载连续运行 ≥10 分钟
**Then** 每个子 wallet 满足 `consumed = quota - remain_quota`
**And** 父池铁律始终成立，`enterprise_quota_allocations`、`subscriptions` 子 wallet 与父池快照一致
**And** `relay/**` 中不新增部门查询、部门变量或企业 DTO 字段。

**Given** 没有创建任何部门预算池或子钱包
**When** 用户继续发起 API 调用
**Then** 现有用户额度、订阅和计费行为完全不变
**And** relay、计费表达式、模型倍率不新增部门判断。

### Story 3.4: 撤销和联动部门 wallet 状态

**Requirements:** FR10, FR12

As a 管理员,
I want 暂停、过期或撤销部门预算时正确联动子钱包,
So that 已分配额度能按规则失效或回收且保留审计。

**Acceptance Criteria:**

**Given** 管理员撤销 `balance` 型子分配
**When** wallet 仍有未消费余额
**Then** 父池回收金额等于 `min(wallet.remain_quota, max(0, allocated_quota - consumed))`
**And** allocation 状态标记为 `revoked`，wallet 状态标记为 inactive/revoked，不物理删除。

**Given** `balance` 型子 wallet 到达 `expires_at`
**When** `balance_expiry_task` 执行
**Then** 父池回收金额等于 `min(wallet.remain_quota_at_expiry, max(0, allocated_quota - consumed))`
**And** allocation 状态标记为 `expired`，wallet 状态失效。

**Given** 同一 `balance` 型过期或撤销事件已处理
**When** `balance_expiry_task` 或撤销逻辑重复执行
**Then** 父池 `remaining` 不会再次增加
**And** allocation 保留 processed_at 或等价幂等标记用于审计。

**Given** 管理员撤销 `subscription` 型子分配
**When** 撤销生效
**Then** 父池释放承诺额，子 wallet 状态标记为 revoked
**And** 下个周期不再 reset，该周期未用部分自然失效。

**Given** 父预算池暂停、过期或撤销
**When** `wallet_state_sync_task` 执行
**Then** 系统批量同步子 wallet 状态
**And** controller 层不逐个手工更新子 wallet。

### Story 3.5: 展示预算池与 wallet 状态

**Requirements:** FR12

As a 管理员,
I want 查看部门预算池和所有派生 wallet 的状态,
So that 我可以判断预算是否健康并追踪每笔分配。

**Acceptance Criteria:**

**Given** 部门存在一个或多个预算池
**When** 管理员打开预算池列表
**Then** 系统展示类型、剩余额度、累计已分配、使用率、生效状态
**And** 支持按使用率、剩余额度、类型和状态排序。

**Given** 管理员打开预算池详情
**When** 系统加载派生子钱包列表
**Then** 展示目标用户、quota、remain_quota、周期/过期、wallet 状态、来源 allocation
**And** 单一审计源可定位到目标用户、目标 wallet、来源父池。

**Given** 预算池使用率达到阈值
**When** 使用率 ≥ 80% 或 ≥ 95%
**Then** 页面分别显示预警或紧急状态
**And** 阈值可配置且保存时校验 `0 < 预警 < 紧急 ≤ 100`。

## Epic 4: 组织用量看板、详情与报告导出

管理层和运营人员可以按部门查看用量、模型分布、用户排行，导出 CSV，并接收定期报告，同时明确多部门用户重复计入口径。

### Story 4.1: 聚合部门用量快照

**Requirements:** FR13

As a 管理员,
I want 系统按部门聚合请求、token、quota 和模型用量,
So that 部门用量看板可以快速加载且口径稳定。

**Acceptance Criteria:**

**Given** 系统存在消费日志和用户部门成员关系
**When** 用量聚合任务执行
**Then** 系统按时间窗扫描日志并 join `enterprise_user_departments` 展开到部门维度
**And** 多部门用户同一笔消费在每个归属部门各计一次。

**Given** 用户没有任何部门成员关系
**When** 聚合任务处理该用户消费
**Then** 系统将消费归入 `dept_id = NULL`
**And** 看板显示为“未归属”，不计入任何部门风险率分母。

**Given** 部门用量聚合任务执行完成
**When** 管理员通过 `/api/enterprise/usage/department-summary` 查询同一时间范围
**Then** API 从 `enterprise_usage_snapshots` 或等价聚合表返回部门汇总数据
**And** 每行至少包含 dept_id、dept_name、window_start、window_end、request_count、prompt_tokens、completion_tokens、quota、user_count 和 model_distribution。

**Given** 聚合范围内存在未归属用户消费
**When** 查询 `/api/enterprise/usage/department-summary`
**Then** 响应包含 `dept_id = null`、`dept_name = "未归属"` 的行
**And** 数组字段缺省返回 `[]`，不返回 `null`。

**Given** `logs` 累计 100 万条且部门数 ≤ 100
**When** 查询近 30 天部门用量首屏
**Then** 系统 P95 响应目标 < 3 秒
**And** 默认查询不得依赖逐条扫描全量日志。

### Story 4.2: 展示部门用量总览看板

**Requirements:** FR13

As a 管理员或经营负责人,
I want 按时间范围查看部门用量总览,
So that 我可以快速判断各部门 AI 使用规模和模型分布。

**Acceptance Criteria:**

**Given** 部门用量聚合数据已生成
**When** 管理员选择今天、昨天、近 7 天、近 30 天或自定义时间范围
**Then** 页面展示部门维度请求数、prompt_tokens、completion_tokens、quota、模型分布和用户数
**And** 同一笔消费可在多个部门出现，这是业务上正确的重复计入。

**Given** 管理员打开部门用量看板
**When** 页面加载完成
**Then** 顶部固定展示“部门间数值不可加和”说明
**And** 文案进入 i18n key `enterprise.usage.multi_dept_disclaimer`，不可被管理员隐藏。

### Story 4.3: 查看部门用量详情

**Requirements:** FR14

As a 管理员,
I want 进入单个部门查看用户排行、模型分布和趋势,
So that 我可以定位某部门主要 AI 消耗来源。

**Acceptance Criteria:**

**Given** 管理员打开某部门详情
**When** 选择时间范围
**Then** 页面展示该部门用户排行、模型分布、时间趋势和最近消费日志入口
**And** 用户排行可按 quota、请求数、token 排序。

**Given** 同一用户属于多个部门
**When** 管理员查看任一部门详情
**Then** 该用户可出现在多个部门排行中
**And** 每个部门按本部门口径独立计算，不跨部门加和。

**Given** 管理员点击最近消费日志入口
**When** 跳转到日志页面
**Then** 系统带入部门、时间范围和用户筛选条件
**And** 不修改现有 `model/log.go` 或 `controller/log.go` 的核心日志语义。

### Story 4.4: 导出部门用量 CSV

**Requirements:** FR15

As a 管理员,
I want 导出当前筛选条件下的部门用量 CSV,
So that 我可以把部门用量用于管理汇报和离线分析。

**Acceptance Criteria:**

**Given** 管理员在用量看板设置筛选和排序
**When** 点击导出 CSV
**Then** CSV 行集合等于页面当前筛选条件下的部门集合
**And** 行排序与页面当前排序一致。

**Given** CSV 生成完成
**When** 管理员打开文件
**Then** 字段按 `部门 ID`、`部门名称`、`父部门`、`周期开始`、`周期结束`、`请求数`、`prompt_tokens`、`completion_tokens`、`quota`、`用户数` 顺序输出
**And** 空值输出为空字符串，不输出 `null`。

**Given** CSV 包含部门数据
**When** 文件生成
**Then** 标题行之前包含 `# 注意：用量按用户当前所属部门重复计入，部门间数值不可加和`
**And** 包含“未归属”行，部门 ID 为空、部门名称为 `未归属`。

### Story 4.5: 配置和发送定期用量报告

**Requirements:** FR16

As a 管理员,
I want 配置部门用量报告接收人、频率和时间范围,
So that 管理层可以定期收到 AI 用量摘要。

**Acceptance Criteria:**

**Given** 管理员打开定期报告配置
**When** 设置接收人、频率和时间范围
**Then** 系统保存配置并校验邮箱格式
**And** 记录低频管理操作审计。

**Given** 定期报告任务到达发送时间
**When** 系统生成报告
**Then** 报告至少包含部门总览、Top 部门和异常高增长部门
**And** 报告包含部门间不可加和说明。

**Given** 邮件发送失败
**When** 任务结束
**Then** 系统记录失败原因和状态
**And** 不影响看板聚合数据生成或后续重试。

## Epic 5: 内容风险事件、告警与部门风险概览

安全/合规负责人可以追踪内容风险事件，配置告警规则，接收可追溯通知，并查看部门风险趋势。

### Story 5.1: 记录内容风险事件

**Requirements:** FR17

As a 安全/合规负责人,
I want 系统记录触发敏感词或过滤规则的风险事件,
So that 后续可以追踪风险来源和请求上下文。

**Acceptance Criteria:**

**Given** 请求触发现有敏感词检查或过滤规则
**When** 风险事件产生
**Then** 系统记录用户、部门成员关系快照、模型、时间、请求追踪 ID、风险类型和处理结果
**And** 风险事件落库失败不得扩大到普通日志敏感内容可见范围。

**Given** 用户属于多个部门
**When** 系统记录风险事件
**Then** 风险事件保存请求发生时的部门成员关系列表
**And** 后续按“包含某部门”语义筛选命中。

**Given** 普通用户查看日志或事件
**When** 页面或 API 返回数据
**Then** 不暴露管理员专用风险字段
**And** 遵守现有日志脱敏策略。

### Story 5.2: 查询和筛选风险事件

**Requirements:** FR17

As a 安全/合规负责人,
I want 按部门、用户、模型、风险类型和时间范围筛选风险事件,
So that 我可以快速定位和复盘具体风险。

**Acceptance Criteria:**

**Given** 已存在风险事件
**When** 管理员按部门筛选
**Then** 系统返回部门成员关系快照中包含该部门的事件
**And** 多部门用户事件可在多个部门筛选结果中出现。

**Given** 管理员按用户、模型、风险类型或时间范围筛选
**When** 查询风险事件列表
**Then** 系统返回分页结果和必要追溯摘要
**And** 不返回敏感原文或超过追溯所需的信息。

### Story 5.3: 配置内容风险告警规则和通道

**Requirements:** FR18

As a 管理员,
I want 配置内容风险告警规则、接收人和通道,
So that 风险事件可以自动通知合适负责人。

**Acceptance Criteria:**

**Given** 管理员打开告警规则配置
**When** 创建或更新规则
**Then** 系统支持按风险类型和部门范围启用或停用规则
**And** 保存时校验接收人或 Webhook 格式。

**Given** 告警规则保存成功
**When** 配置发生变化
**Then** 系统记录低频管理操作审计
**And** 不把 Webhook secret 或邮箱凭据写入普通日志。

**Given** V1 仅保证邮件告警为必选
**When** 钉钉机器人或 Webhook 未配置
**Then** 系统仍可通过邮件和后台事件完成 V1 告警闭环
**And** 可选通道不阻塞风险事件记录。

### Story 5.4: 发送可追溯告警通知

**Requirements:** FR19

As a 安全/合规负责人,
I want 收到包含追溯信息的风险告警,
So that 我可以直接定位后台事件详情。

**Acceptance Criteria:**

**Given** 风险事件命中启用的告警规则
**When** 告警投递任务执行
**Then** 通知内容至少包含部门、用户、模型、时间、风险类型、请求追踪 ID
**And** 提供能定位后台事件详情的追溯信息。

**Given** 告警通道返回失败
**When** 系统执行重试
**Then** 按默认策略重试 3 次：30 秒、2 分钟、10 分钟
**And** 3 次后标记为最终失败并保留错误原因。

**Given** 同一风险事件在 5 分钟内重复触发同一规则
**When** 系统准备发送通知
**Then** 聚合为一次告警发送
**And** 避免重复通知噪声。

### Story 5.5: 管理告警发送状态和人工重发

**Requirements:** FR19

As a 管理员,
I want 查看告警发送状态并重新发送失败告警,
So that 通知失败可以被追踪和人工补救。

**Acceptance Criteria:**

**Given** 告警发送记录存在
**When** 管理员查看告警发送列表
**Then** 系统区分待发送、发送成功、发送失败、最终失败、已重发状态
**And** 失败状态包含可读错误原因。

**Given** 告警最终失败
**When** 管理员点击重新发送
**Then** 系统创建新一轮投递并标记为人工触发
**And** 不重置原风险事件追溯信息。

**Given** 通知投递失败
**When** 风险事件已产生
**Then** 通知失败不会阻止风险事件落库
**And** 后台仍可查看事件详情。

### Story 5.6: 展示部门风险概览

**Requirements:** FR20

As a 安全/合规负责人,
I want 查看部门风险次数、风险率和趋势,
So that 我可以识别高风险部门和近期变化。

**Acceptance Criteria:**

**Given** 存在风险事件和部门用量数据
**When** 管理员打开部门风险概览
**Then** 页面展示部门风险次数、风险率、近期趋势、Top 风险部门和风险事件入口
**And** 部门风险统计与风险事件明细数量一致。

**Given** 用户属于多个部门
**When** 计算部门风险率
**Then** 同一笔请求按成员所属部门重复计入多个部门
**And** 页面展示风险率公式和 `enterprise.usage.multi_dept_disclaimer`。

**Given** 用户未归属任何部门
**When** 计算风险率
**Then** 该用户请求不纳入任何部门风险率
**And** 单独显示为“未归属”。
