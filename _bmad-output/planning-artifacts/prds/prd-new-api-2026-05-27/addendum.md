---
title: "{NEW_PRODUCT_NAME} V1 企业管控基线 PRD Addendum"
status: draft
created: 2026-05-27
updated: 2026-05-27
---

# Addendum: V1 企业管控基线技术与决策上下文

本文保存不适合放入 PRD 主体、但会影响架构和 story 拆分的技术细节。主 PRD 定义“要什么”，本文记录“已有基线、约束、待架构决策”。

## 1. 输入来源

- `_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md`
- `_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/addendum.md`
- `_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/.decision-log.md`
- `AGENTS.md`
- `pkg/billingexpr/expr.md`
- 代码基线抽查：`model/user.go`、`model/log.go`、`model/usedata.go`、`controller/custom_oauth.go`、`router/api-router.go`

## 2. 棕地现状摘要

### 2.1 用户与分组

- `model.User` 当前包含 `Group string`，用于计费倍率、模型/渠道可用性和订阅升级等业务。
- `Group` 是单维字符串标签，不代表组织层级。
- 用户缓存中也包含 `Group`，相关变更会影响缓存一致性。
- 因为 `group` 是部分数据库的保留字，现有代码通过 `commonGroupCol` 处理 raw SQL 场景；新增组织字段不要继续使用 `group` 命名。

### 2.2 OAuth

- 已有内置 OAuth provider 和 `CustomOAuthProvider` 管理能力。
- `CustomOAuthProvider` 适合标准 OAuth/OIDC 字段映射，但钉钉通讯录同步不是单纯 OAuth 登录，需要额外同步任务、权限校验和外部组织 ID 映射。
- 架构阶段需要决定：钉钉登录复用 `CustomOAuthProvider` 的登录部分，还是新增专用 `dingtalk` provider。

### 2.3 用量与日志

- `model.Log` 记录消费日志，包含 `UserId`、`Username`、`ModelName`、`Quota`、`PromptTokens`、`CompletionTokens`、`Group`、`RequestId` 等字段。
- `model.QuotaData` 提供小时级数据看板聚合，当前按用户、用户名、模型和时间聚合。
- 部门用量看板如果直接扫 `logs` 可能有性能风险，架构阶段应优先评估新增部门快照或聚合表。
- 历史日志如果没有部门快照，后续用当前部门反推历史归属会产生错误口径。

### 2.4 计费表达式

- `pkg/billingexpr/expr.md` 明确计费表达式系统原则：单个表达式定义模型计费，表达式输出通过 quota conversion 转为内部 quota，并应用 groupRatio。
- V1 部门配额不得新增模型计费语义，也不应改变 `p`、`c`、`len` 等变量。
- 部门配额应消费最终 quota，定位为治理额度，不是模型定价层。

### 2.5 内容监控

- 现有 `setting/sensitive.go` 提供敏感词配置和 prompt 检查开关。
- `controller/relay.go` 中已有 relay 前的敏感词检查入口。
- V1 应基于现有检查补事件记录和告警，不应承诺复杂语义审核或第三方安全模型。

### 2.6 通知能力

- `common/email.go` 已有邮件发送能力。
- 系统设置里已有监控/告警相关页面和邮件告警概念。
- 钉钉机器人/Webhook 告警是否作为 V1 必选仍未确认。

## 3. 架构待决策项

### A-1. 组织层级建模方式

候选：
- 邻接表：每个部门记录 `parent_id`，实现简单，适合 V1。
- 物化路径：查询子树方便，但部门移动需要维护路径。
- 闭包表：复杂查询强，但 V1 成本较高。

建议：V1 优先邻接表，必要时增加路径缓存字段，避免一开始引入闭包表复杂度。

### A-2. `tenant_id` 预留范围

候选：
- 只在新增组织、部门、配额、同步、告警表中预留。
- 同步扩展到 `users`、`logs`、`quota_data` 等旧表。

取舍：
- 只在新增表预留更利于 V1 交付。
- 扩展旧表更利于 V3 SaaS，但会显著增加迁移、查询和索引改造成本。

PRD 默认假设：只在新增组织相关数据中预留，是否扩旧表作为 OQ-3。

### A-3. 用户身份匹配规则

需要明确优先级：
- 钉钉 unionid/openid 与本地绑定表。
- 邮箱匹配。
- 手机号匹配。
- 管理员手工确认。

建议：高置信标识自动绑定；邮箱/手机号冲突进入待处理，不自动合并。

### A-4. 部门配额扣减位置

需要在架构阶段确认：
- 预消费阶段先判断部门剩余额度。
- 实际结算阶段写入部门用量。
- 预估与实际差额如何补偿。
- 并发消费如何避免超扣。

验收风险最高的是 FR-10，应优先出技术设计和并发测试。

### A-5. 部门用量统计口径

需要决定：
- 消费发生时写入部门快照，还是查询时 join 当前用户部门。
- 用 `logs` 扩字段，还是新增 `department_usage_data` 聚合表。
- `quota_data` 是否扩展部门字段。

建议：消费时保存部门快照或异步聚合时保存部门维度，避免历史归属漂移。

### A-6. 同步任务调度

需要决定：
- 全量同步是否阻塞 UI。
- 定时同步间隔与失败重试策略。
- 同步任务状态存储表结构。
- 是否支持手动重试单个失败项。

### A-7. 本地组织管理 UI

简报指出钉钉组织树是 V1 的工作量简化来源。若 V1 同时支持无钉钉企业的完整本地组织管理 UI，将显著扩大 UX、权限、审计和数据变更范围。

建议：默认不纳入 V1；仅提供只读部门树和必要的冲突处理。无钉钉 Plan B 作为后续补充 PRD 或 V1.1。

## 4. 建议的表/模块方向（非最终设计）

以下只是架构输入，不是 PRD 强制实现：

- `organizations`：企业实例级组织记录，V1 可单条。
- `departments`：部门节点，含外部来源、外部部门 ID、父部门、本地状态。
- `user_departments`：用户与部门关系，含主部门标记、外部成员 ID。
- `department_quota_policies`：部门周期配额配置。
- `department_quota_usage`：部门周期用量聚合或快照。
- `dingtalk_sync_jobs`：同步任务。
- `dingtalk_sync_items`：同步明细与失败原因。
- `content_risk_events`：内容风险事件。
- `alert_deliveries`：告警发送记录。

## 5. 测试重点

- SQLite、MySQL、PostgreSQL 三库迁移和查询。
- 部门树三层以上结构、部门移动、部门停用。
- 钉钉同步幂等性：重复全量同步不得重复建部门或用户。
- 用户冲突匹配：邮箱/手机号冲突不能自动绑定错误账号。
- 部门配额并发扣减：同部门多用户同时请求不得超额。
- 计费表达式模型的部门扣减：最终部门用量与用户日志 quota 一致。
- 历史部门快照：用户换部门后历史用量不漂移。
- 告警通道失败：事件落库不受通知失败影响。
- 权限：普通用户不能查看其他部门风险事件或管理员字段。

## 6. 后续 PRD 分界

- V2 代码库 DeepWiki、WebChat 增强、多模态入口应单独 PRD。
- 通用企业文档库 RAG 是 V2 二期可选扩展，不应混入 V1。
- V3 SaaS 多租户应独立立项，不能反向扩大 V1。
