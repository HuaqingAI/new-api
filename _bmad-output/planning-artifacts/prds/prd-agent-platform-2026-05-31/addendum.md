---
title: "Agent Platform PRD Addendum"
status: draft
created: 2026-05-31
updated: 2026-06-02
revision_notes:
  - "V1.4 (2026-06-02): Correct Course 重处理后，AP-6 从早期“标准层定位说明”收敛为 AP-1 到 AP-5 完成后的 follow-up epic：下游公共契约冻结与接入签核。AP-6 成为 Cherry Studio / Codex 正式接入前的 MVP exit gate。"
---

# Addendum: 通用 Agent 能力平台技术与规划上下文

本文记录不适合放入 PRD 主体、但会影响架构和 Epic 拆分的技术上下文。

## 1. 输入来源

- `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-31-agent-platform.md`
- `_bmad-output/planning-artifacts/briefs/brief-new-api-2026-05-27/brief.md`
- `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`
- `_bmad-output/planning-artifacts/architecture.md`
- `docs/integration-architecture.md`
- `AGENTS.md`

## 2. 关键边界

### 2.1 与企业治理底座的边界

- 企业治理底座继续负责组织、身份、预算、用量、风险治理。
- Agent Platform 负责通用资源控制面：
  - skill
  - knowledge base
  - agent
  - client exposure / publish projection

### 2.2 与 relay / 模型转发链路的边界

- 本规划默认不改写 `relay/**` 的核心协议与转发语义。
- 管理 API 进入 `docs/openapi/api.json`。
- 面向下游工具的标准能力发现 / 获取 / 调用接口，属于开放能力扩展层，不属于传统 relay API 面。

### 2.3 与 Cherry Studio 的边界

- Cherry Studio 是首个验证消费者。
- AP-6 不是 Cherry Studio 专属接口层。
- 标准应先于消费者，消费者只验证标准是否足够。

## 3. 必须显式设计的控制面对象

- 资源定义（Resource Definition）
- 资源版本（Version）
- 发布投影（Published Projection）
- 客户端可见性（Client Visibility）
- 可调用状态（Callable State）
- 撤销状态（Revocation State）
- 缓存新鲜度（Freshness / TTL / ETag）
- 统一错误包（Error Envelope）
- 审计记录（Admin / Publish / Revoke / Exposure Audit）

## 4. 建议的后台信息架构

- 平台总览
- 客户端管理
- Skill 库
- Knowledge 库
- Agent 库
- 发布中心
- 审计与任务
- 契约与 Mock

## 5. AP-6 的具体定位

### 5.1 2026-05-31 原始定位

AP-6 最初用于强调：它不是“做 Cherry Studio 适配器”，而是在既有 OpenAPI / open 能力体系上扩出统一标准层，定义 discovery、detail、invoke / execute、refresh、revoke / offline handling、error/status mapping 等消费者侧通用契约，并用 Cherry Studio 先验证这套标准。

### 5.2 2026-06-02 Correct Course 重处理后的定位

AP-1 到 AP-5 已经完成资源治理、客户端接入、开放能力、Skill、Knowledge 与 Agent 的 MVP 基线。因此 AP-6 的当前定位不再是“是否要建设标准层”，而是：

- 将 AP-1 到 AP-5 已实现能力冻结成下游可签核公共契约。
- 补齐 enterprise model discovery / default model / model status 公共契约。
- 冻结 OAuth / token / revoke / callback / allowlist wire contract。
- 冻结 discovery / detail / refresh 公共字段集。
- 冻结 Skill invoke 与 Knowledge query request / response。
- 冻结统一错误码矩阵与客户端状态矩阵。
- 提供 mock / fixture / contract conformance 套件。
- 明确 Cherry Studio first consumer 与 Codex second consumer 的签核路径。

### 5.3 AP-6 MVP Exit Gate

Cherry Studio 或 Codex 进入正式接入签核前，至少需要满足：

1. AP-6 公共契约文档完成，并标注 MUST / SHOULD / MAY 字段。
2. Enterprise model discovery 契约完成，不能用现有 `/api/models`、`/api/user/models` 或 `/v1/models` 模糊替代。
3. OAuth / token / revoke / callback / allowlist 行为完成签核。
4. Discovery / detail / refresh、Skill invoke、Knowledge query 示例 payload 完成。
5. 错误码矩阵和客户端状态矩阵完成。
6. Mock / fixture / conformance tests 可供下游验证。
7. Cherry Studio 专属字段只通过 namespaced extensions 承载。
8. Codex 作为第二消费者评审时，不需要平行资源模型或核心协议主干。

## 6. 暂不进入 MVP 的内容

- 多企业账号并存
- 本地离线副本执行
- 知识库 ingestion / embedding / RAG pipeline 全链路
- 复杂 agent workflow engine
- 客户端端到端双向同步冲突解决
