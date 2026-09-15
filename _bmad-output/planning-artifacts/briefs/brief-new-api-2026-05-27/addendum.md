---
title: new-api 企业演进简报 - 附录
status: approved
created: 2026-05-27
updated: 2026-05-27
---

# 附录：new-api 企业演进简报

> 本附录沉淀简报正文之外的深度内容：现状能力清单、技术约束、阶段排序依据、后续 PRD/架构需要带入的信息。

---

## A. 现状能力详细评估

| 功能方向 | 状态 | 关键文件 |
|---|---|---|
| 用户管理 / 3 级角色 | 已有 | `model/user.go`, `common/constants.go` |
| 用户分组（计费倍率层） | 已有 | `setting/ratio_setting/group_ratio.go`, `setting/user_usable_group.go` |
| 组织 / 多团队 | **缺失** | 无 `tenant_id` / `org_id` |
| 用户额度 + 用量统计 | 已有 | `model/usedata.go`, `model/user.go` |
| 模型/分组/分组间三级倍率 | 已有 | `setting/ratio_setting/*` |
| 计费表达式系统 | 已有 | `pkg/billingexpr/expr.md` 必读 |
| 订阅套餐 | 已有（完整） | `model/subscription.go` |
| 在线充值（4 渠道） | 已有 | `controller/topup_*.go` |
| Token / API Key 管理 | 已有（完整） | `model/token.go` |
| 模型/IP 白名单 | 已有 | `model/token.go`, `middleware/auth.go` |
| 渠道管理 + 多 Key | 已有 | `model/channel.go` |
| 渠道亲和性路由 | 已有 | `setting/operation_setting/channel_affinity_setting.go` |
| 上游同步 | 已有 | `controller/ratio_sync.go`, `controller/channel_upstream_update.go` |
| 请求日志 + 链路追踪 | 已有 | `model/log.go` |
| 管理员审计字段 | 部分已有（无独立表） | `model/log.go` RecordLogWithAdminInfo |
| 日志分库 | 已有 | LOG_DB 配置 |
| 限流（3 层） | 已有 | `middleware/rate-limit.go`, `middleware/model-rate-limit.go` |
| OIDC / 自定义 OAuth | 已有（含访问策略） | `model/custom_oauth_provider.go` |
| WebAuthn / Passkey | 已有 | `model/passkey.go` |
| 2FA / TOTP / 备用码 | 已有 | `model/twofa.go` |
| 微信 / Telegram 登录 | 已有 | `controller/wechat.go`, `controller/telegram.go` |
| **钉钉 / 飞书 / 企微登录** | **缺失** | — |
| **LDAP / SAML / SCIM** | 缺失（不做） | — |
| **多租户隔离** | **缺失** | — |
| Web 前端（双主题） | 已有 | `web/default/`（React 19），`web/classic/`（React 18） |
| Playground / WebChat | 已有（基础） | `web/default/src/features/playground/`（路径待确认） |
| 代码库 DeepWiki | **缺失** | — |
| 通用文档库 RAG | **缺失**（V2 二期可选） | — |
| 多模态生成入口 | **缺失** | — |

---

## B. 项目身份与约束

来自 `CLAUDE.md`：

- **Rule 5（受保护标识）**：`new-api`（项目名）与 `QuantumNous`（组织名）严禁修改/删除/替换。本演进所有外部材料保留这两个标识的原始形式。
- **Rule 2（DB 三库兼容）**：所有新增表（特别是 `Organization`、`KnowledgeBase`、`TenantConfig` 等）必须同时通过 SQLite / MySQL 5.7.8+ / PostgreSQL 9.6+。
- **Rule 1（JSON 包装）**：所有 JSON 操作走 `common/json.go`，新模块禁止直接 import `encoding/json`。
- **Rule 7（计费表达式）**：组织级配额若涉及表达式扩展，必读 `pkg/billingexpr/expr.md`。

---

## C. 头脑风暴中讨论但未纳入简报的细节

### C.1 池子里的方向（暂不做但有据）

- **D — API Key 管控策略增强**：强制过期、轮换提醒、管理员代管。优先级低，现有 Key 已具备模型/IP 白名单 + 额度限制，足够企业基线。
- **H — 私有化授权 License 机制**：与 SaaS 模式存在哲学冲突（License 倾向于"卖一份代码"，SaaS 倾向于"卖一份服务"）。待 V3 SaaS 定位明确后再决定是否补 License。
- **A — 专用审计日志表**：现有 `Log` 表 `Manage` 类型 + `admin_info` JSON 已记录管理员操作来源；如未来出现合规审查需要"变更前后值对比"，再单独立项。
- **通用 LDAP/SAML/SCIM**：中国企业市场实际使用率低，让位于钉钉/飞书/企微。

### C.2 头脑风暴产生但简报未明示的关键洞察

> "钉钉本身就是一个组织树（企业 → 部门 → 成员），如果用钉钉登录并对接钉钉通讯录同步，可以**免费获得组织层级的数据来源**。"

这是 V1 主题 A 设计的核心简化逻辑——**huaqing 版不必在 UI 中提供组织管理界面，钉钉管理员在钉钉侧管理组织结构即可**。本架构选择会显著减少 V1 工作量，但对"不用钉钉的企业"会失效，需在 PRD 阶段补充"无 IdP 时的本地组织管理 UI"作为 Plan B。

### C.3 主题 B 内部依赖关系

```
代码库 DeepWiki（Git 接入 + 索引 + 问答）
       │
       └── WebChat 接入 ←── 多模态输入（图像/视频生成入口）
```

任何一环单独存在都有价值，但联动起来才形成"研发团队 AI 工作台"的完整体验。V2 应按"代码库 DeepWiki → WebChat 接入 → 生成入口"的顺序逐步上线。通用企业文档库 RAG 作为 V2 二期可选扩展，需求验证通过后再立项。

---

## D. 阶段排序依据

### V1 优先做企业管控的理由

1. **直接对应商业化第一类客户（中型企业 IT）**，是回本周期最短的方向
2. **B + C + E 三个子方向技术耦合度高**，一次性做完比拆分做更经济
3. **F（组织看板）严格依赖 B**，必须排在后面
4. **G（内容监控告警）独立、改动小**，作为 V1 的快赢补充
5. **不依赖任何新增的开源大组件**，研发可控

### V2 平台扩展放在 V1 之后的理由

1. **知识库/生成接入会引入新的开源依赖**（向量库、RAG 框架、ComfyUI 等），风险更高
2. **V2 的价值释放需要 V1 的组织/用量基础**——例如知识库的访问权限要按部门控制
3. **更适合在企业自部署版稳定后再上**，避免"基础不稳就堆功能"

### V3 SaaS 后置的理由

1. **多租户改造是数据模型大动作**，最好在 V1 的 Organization 模型上自然扩展
2. **企业自部署版的稳定运营是 SaaS 销售背书**
3. **资源约束下，先打透中型企业市场比同时铺两条线更稳**

---

## E. 后续 PRD/架构需要决策的技术议题

> 以下议题不在简报层决策，但在 PRD/架构阶段需要明确：

### E.1 数据模型

- `Organization` 表结构：层级方式（邻接表 / 闭包表 / 物化路径）
- `tenant_id` 是否在 V1 就引入所有相关表（即使值固定为 1）
- 用户↔组织↔分组三者关系的最终模型

### E.2 钉钉集成

- 钉钉 OAuth 与现有 `CustomOAuthProvider` 是复用还是独立
- 通讯录同步：增量/全量、定时/事件触发
- 钉钉部门变更对已分配配额的影响（部门合并、人员离职）

### E.3 代码库 DeepWiki

- Git 平台接入范围：GitHub / GitLab / Gitee / 自建 GitLab；认证方式（PAT / OAuth / SSH key）
- 代码索引策略：是否在用户侧推送，还是 `{NEW_PRODUCT_NAME}` 主动 clone+解析；多大仓库需特殊处理
- AI 调用方式：复用 new-api 渠道（用户 Key 计费）还是平台预付费
- 可视化形态：模块图 / 调用图 / 架构图，使用什么开源依赖（mermaid 已内置，但缺乏代码图谱专用库）
- 通用文档库（二期）：向量库选型（pgvector / Milvus / Qdrant / sqlite-vss）

### E.4 SaaS 多租户

- 隔离方式：单库多 schema / 单库租户列 / 多库
- 用量统计跨租户聚合的成本
- Token 路由：每租户独立渠道，还是共享池

### E.5 内容监控

- 现有过滤的实现位置（请求拦截点、关键词来源）
- 告警通道（钉钉机器人 / 邮件 / Webhook）
- 监控指标的存储与查询

---

## F. 商业模式开放项（已根据用户决策初步落定）

简报标记的开源核心 + 企业版增值边界，按用户决策的初步划分如下：

| 能力 | `{NEW_PRODUCT_NAME}` 独占 / 上推 QuantumNous | 理由 |
|---|---|---|
| 钉钉登录 | 上推 | 通用性强，社区可受益 |
| 飞书 / 企微登录 | 上推 | 同上 |
| 组织/部门模型 | 上推 | 基础设施性质，应进开源核心 |
| 部门级用量看板 | 上推 | 与组织模型配套 |
| 内容监控告警 | 上推 | 通用安全能力 |
| 代码库 DeepWiki | **独占** | 商业化主打能力 |
| 通用企业文档库 RAG | **独占** | 同上 |
| WebChat 基础改进 | 上推 | 通用 UI/交互优化可上推 |
| WebChat × DeepWiki 集成 | **独占** | 与 DeepWiki 深度耦合，作为商业差异化 |
| 图像/视频生成入口 | 上推 | 通用能力 |
| SaaS 多租户 | **独占** | 商业模式核心 |
| Token 批发 / 运维 / 培训 | **独占** | 服务层，无开源意义 |

> 上推节奏不设硬指标，按 case 决定；企业专属功能（独占类）默认不上推。

---

## G. 完整战略风险清单

简报正文仅展示 Top 3 风险，完整清单如下：

| 风险 | 影响 | 缓解 |
|---|---|---|
| **代码库 DeepWiki 形态新颖，需求验证不足** | V2 主打能力卡壳 | V2 启动前做 1-2 家试点企业的需求访谈；优先支持主流 Git 平台 |
| **多租户晚做付出重构成本** | V3 启动时数据模型大改 | V1 的 `Organization` 模型预留 `tenant_id` 字段 |
| **双轨模式拉扯研发资源** | V1/V2 拖期 | V3 单独立项；前两版严格不为 SaaS 让步设计 |
| **新产品名 TBD 拖延品牌建设** | 对外材料/销售延迟 | 在 V1 启动前敲定产品名 |
| **品牌定位混淆**（new-api vs `{NEW_PRODUCT_NAME}`） | 客户认知模糊 | 文档/UI/对外材料明确"`{NEW_PRODUCT_NAME}` 基于 new-api 构建"的关系；受保护标识不动 |
| **平台扩展三方向分散投入** | 主题 B 三个子方向都做不深 | V2 内严格序列化：DeepWiki → WebChat → 生成入口；前一个未达成不启动下一个 |
| **上游开源协同节奏失配** | 功能贡献回 QuantumNous 时合并冲突或路线分歧 | 早期与上游沟通；企业专属功能默认不上推 |

---

_此附录在简报定稿后会随 PRD/架构进度持续扩展。_
