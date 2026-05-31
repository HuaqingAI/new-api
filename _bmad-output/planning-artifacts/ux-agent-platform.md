---
title: "Agent Platform 轻量 UX 说明"
status: draft
created: 2026-05-31
updated: 2026-05-31
related_prd: "./prds/prd-agent-platform-2026-05-31/prd.md"
related_architecture: "./architecture-agent-platform.md"
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

## 4. 页面级最低要求

### Overview

- 展示资源总量、各状态分布、最近发布/撤销动作、近期失败调用统计。

### Clients

- 展示客户端接入状态、授权模式、能力声明完整性。

### Skills / Knowledge / Agents

- 统一使用“列表 + 详情/编辑抽屉 + 版本标签 + 发布历史”布局。

### Publishing

- 展示按资源和客户端展开的 projection 视图。

### Audit & Diagnostics

- 展示完整时间线和 drill-down 入口。

## 5. 文案与术语约束

- 使用 `visible` / `callable`，不要用模糊词替代。
- 使用 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated` 作为唯一状态词汇。
- `Knowledge` 对外模式统一显示为 `retrieval`，禁止在管理端把特定 provider 名称显示成平台标准模式。
