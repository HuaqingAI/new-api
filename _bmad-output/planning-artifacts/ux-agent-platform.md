---
title: "Agent Platform 轻量 UX 说明"
status: draft
created: 2026-05-31
updated: 2026-06-02
related_prd: "./prds/prd-agent-platform-2026-05-31/prd.md"
related_architecture: "./architecture-agent-platform.md"
revision_notes:
  - "V1.4 (2026-06-02): 补充 AP-6 下游公共契约冻结与接入签核所需的 UX 最小要求，包括 Clients onboarding、OAuth/consent 可见性、契约签核视图、错误/状态矩阵和 model discovery 状态展示。"
---

# Agent Platform 轻量 UX 说明

## 1. 目标

本说明只冻结实施前必须统一的管理端关键交互，避免前端在开发期再次反推领域规则。它不是完整视觉稿，也不替代后续详细设计。

## 2. 信息架构

管理端导航顺序固定为：

1. Overview
2. Clients
3. Skills
4. Knowledge
5. Agents
6. Publishing
7. Audit & Diagnostics

## 3. 关键交互

### 3.1 客户端注册

- 列表页显示 `client_id`、名称、状态、支持的 `contract_version`、授权模式、最近一次授权时间。
- 创建/编辑抽屉至少包含：名称、slug、grant types、allowed scopes、contract version、capability declarations、redirect URIs、namespaced extensions。
- 若客户端配置不完整，列表与详情中显示 `invalid integration` 状态，不允许进入发布目标选择。

### 3.2 资源发布与撤销

- Publishing 页以“资源版本 -> 目标客户端 -> projection 状态”为主视角。
- 发布动作必须要求用户明确选择资源版本与目标客户端，禁止隐式覆盖。
- 撤销动作必须展示影响范围：
  - 只影响当前客户端 projection
  - 不删除资源定义
  - 不影响其他客户端 projection

### 3.3 visible but not callable

- 列表页和详情页都必须分开展示 `visible` 与 `callable`。
- 当资源“可见但不可调”时，UI 必须显示明确原因标签，最少覆盖：
  - contract mismatch
  - permission denied
  - dependency invalid
  - provider offline
  - revoked
  - stale projection

### 3.4 审计与诊断 drill-down

- Audit & Diagnostics 列表页必须支持按 `request_id`、资源 ID、客户端、动作、结果状态过滤。
- 详情抽屉至少展示：操作者、对象、动作、前后状态、关联版本、错误类别、是否可重试、时间线。
- 若失败来自 provider，详情中展示“provider adapter failure”而不是 provider-native 原始报文。

### 3.5 失败重试反馈

- 对可重试失败，UI 使用明确文案 `retryable`。
- 对不可重试失败，UI 必须提示用户优先检查契约、权限、依赖或 provider 状态，而不是只显示通用错误提示。

### 3.6 AP-6 Clients onboarding

- Clients 工作区必须支持下游接入最小闭环：
  - client registration 字段查看与编辑
  - redirect URI / callback 配置
  - allowed scopes
  - contract version
  - capability declarations
  - namespaced extensions
  - allow client credentials 开关
- AP-6.2 当前冻结的是 registration schema 与 API / fixture 支撑的最小 onboarding 闭环，不代表完整 Clients 工作区已完成。
- 若完整 Clients 工作区尚未实现，UI 必须清楚标记 registration、redirect/callback、allowed scopes、contract version、capabilities、consent/OAuth prerequisites、mock fixture 中哪些步骤仍由 API / fixture 完成，避免运营人员误以为可完全自助接入。
- `invalid integration` 必须给出字段级原因，而不是只显示通用错误。

### 3.7 OAuth / consent 可见性

- 客户端详情页必须展示 OAuth 状态摘要：
  - grant types
  - redirect URI 校验状态
  - consent 是否存在
  - token revoke 状态
  - 最近一次 authorize/token/revoke 时间
- consent 可见性必须对管理员可解释，但不得显示 refresh token、access token、provider secret 或用户敏感凭据。
- callback / allowlist 未配置或不匹配时，UI 显示专门状态，而不是归入 generic failure。

### 3.8 下游契约签核视图

- AP-6 至少需要一个契约签核视图或等价页面区块，用于展示：
  - 当前 contract version
  - client registration / onboarding schema 是否冻结
  - OAuth / token / revoke 是否冻结
  - discovery / detail / refresh 字段集是否冻结
  - Skill invoke / Knowledge query spec 是否冻结
  - model discovery spec 是否冻结
  - error matrix / state matrix 是否冻结
  - mock / fixture / conformance tests 是否可用
  - Cherry Studio signoff 状态
  - Codex second-consumer review 状态
- 签核状态必须区分 `draft`、`ready for signoff`、`signed off`、`blocked`。

### 3.9 Model discovery 状态展示

- Model discovery UI 必须显示默认模型、模型状态和不可用原因。
- 至少覆盖：
  - no default model
  - multiple default models
  - default model disabled
  - provider offline
  - model unavailable
  - account / tenant mismatch
- 不得把 `/v1/models` 的 relay 列表直接展示成 enterprise model discovery 签核结果。

### 3.10 客户端状态矩阵展示

- UI 必须支持将平台错误映射为客户端可理解状态。
- 至少覆盖：
  - loginExpired
  - empty
  - loadFailed
  - networkFailed
  - noAssignedResource
  - visibleButNotCallable
  - stale
  - revoked
  - offline
- 对每个状态，UI 应展示责任边界：client、resource、permission、contract、provider 或 platform。

## 4. 页面级最低要求

### Overview

- 展示资源总量、各状态分布、最近发布/撤销动作、近期失败调用统计。

### Clients

- 展示客户端接入状态、授权模式、能力声明完整性。
- 展示 onboarding checklist：registration、redirect/callback、scopes、contract version、capabilities、consent、mock fixture。
- 若 AP-6.2 只由 API / fixture 支撑，则必须把该边界显示为后续 UI productization 缺口。
- 展示 AP-6 signoff 状态，至少覆盖 Cherry Studio first consumer 与 Codex second-consumer review。

### Skills / Knowledge / Agents

- 统一使用“列表 + 详情/编辑抽屉 + 版本标签 + 发布历史”布局。
- MVP 最低可用形态不是静态说明卡片或 narrative shell，而是与现有 `web/default` 管理页一致的真实列表工作区。
- 最低交互应至少覆盖：
  - loading / empty / error states
  - status badge 与版本标签
  - 列表级信息组织（按 tab、分组或筛选）
  - 详情入口或清晰的 forward-compatible workspace slot

### Publishing

- 展示按资源和客户端展开的 projection 视图。

### Audit & Diagnostics

- 展示完整时间线和 drill-down 入口。

### Contract Signoff

- 展示 AP-6 公共契约冻结状态。
- 展示 mock / fixture / conformance test 可用性。
- 展示 error matrix、state matrix、model discovery 的签核状态。

## 5. 文案与术语约束

- 使用 `visible` / `callable`，不要用模糊词替代。
- 使用 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated` 作为唯一状态词汇。
- `Knowledge` 对外模式统一显示为 `retrieval`，禁止在管理端把特定 provider 名称显示成平台标准模式。
