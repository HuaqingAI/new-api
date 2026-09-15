# Story AP-4.1: 建立带 provider 元数据的 Knowledge 管理能力

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 管理 `Knowledge` 资源、版本、发布状态与 provider 绑定元数据,
so that `Knowledge` 成为平台治理对象，而不要求平台承担 ingestion 或索引职责。

## Acceptance Criteria

1. **Given** 发布者创建或编辑 `Knowledge` 资源  
   **When** 定义被保存  
   **Then** 平台保存该资源的元数据、状态、版本与发布信息  
   **And** 即便 provider 侧 ingestion/indexing 在平台外部，治理仍然有效。
2. **Given** 某个 `Knowledge` 需要 provider-backed 配置  
   **When** 元数据被持久化  
   **Then** provider type 与 provider binding config 作为平台治理的一部分被保存  
   **And** 外部 provider 不会变成控制面生命周期的 source of truth。
3. **Given** 某个 `Knowledge` 被发布或撤销  
   **When** 查询审计与可见性数据  
   **Then** 平台能追踪其发布历史、撤销历史与客户端可见范围  
   **And** 它复用与其他资源一致的 exposure 模型。

## Tasks / Subtasks

- [x] 提供 Knowledge 专用 control-plane 管理入口，复用现有治理基线 (AC: 1, 2, 3)
  - [x] 新增 `controller/service/dto` 中的 Knowledge 专用 list/get/create/update 入口，建立在现有 `resource` / `resource_version` / `knowledge_def` 模型之上，而不是重建另一套主表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1105]
  - [x] `Knowledge` 入口只面向 `resource_type=knowledge`，减少调用方对通用资源接口的手动过滤负担。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-9]
  - [x] 支持稳定 identity、状态、当前版本和 provider-backed 元数据聚合查看，满足“治理有效但不承担 provider runtime”的目标。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-9]

- [x] 把 provider 元数据纳入 Knowledge 管理面而不越界到 provider runtime (AC: 1, 2)
  - [x] 复用现有 `knowledge_def` typed detail，Knowledge 管理面建立在其已有 `knowledge_mode`、`provider_type`、`provider_adapter_key`、`provider_config_json` 等字段之上，为后续 4.2/4.3 铺路。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#387-398]
  - [x] 控制面把 provider 元数据视为平台治理的一部分，但没有把 provider 变成 control plane 生命周期的 source of truth。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.1]
  - [x] 本故事没有实现 query/runtime 侧 provider 调用，Knowledge query 闭环继续留给 4.3/4.4。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.3; #Story AP-4.4]

- [x] 衔接 publish / revoke / exposure 语义到 Knowledge 视角 (AC: 3)
  - [x] 复用已有 lifecycle / exposure 模型，让 Knowledge 的发布历史、撤销历史和对客户端的可见范围继续走统一治理基线。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-9]
  - [x] 保持“已发布/未发布/已撤销”的外显行为继续以 exposure 为 source of truth，没有在 4.1 另造 Knowledge 专属发布状态体系。[Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]

- [x] 补定向测试与最小前端承接面 (AC: 1, 2, 3)
  - [x] 新增/更新 Knowledge 管理相关 service/controller 测试，覆盖：仅返回 knowledge、创建 knowledge、更新基础信息。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 将 `web/default` 的 Agent Platform Knowledge 区域从纯静态占位升级为最小真实列表/承接视图，延续 `3.1` 对 Skills 的处理方式。[Source: web/default/src/features/agent-platform/index.tsx]

## Dev Notes

### Previous Story Insights

- `3.1` 已经给 Skill 做了 control-plane 专用入口和最小前端承接面，因此 `4.1` 可以沿用同样模式为 Knowledge 增加专用入口。[Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md]
- 通用 resource version 与 `knowledge_def` 基线已经存在，因此 `4.1` 的重点是 Knowledge 管理面和 provider 元数据的可治理视角，不是重做底层模型。[Source: model/agentplatform/knowledge_def.go; service/agentplatform/resource_version.go]

### Scope Boundaries

- `4.1` 不实现：
  - Knowledge query runtime
  - 外部 provider 真正调用
  - retrieval 统一契约冻结（留给 4.2）
- `4.1` 要实现：
  - Knowledge 专用 control-plane 管理入口
  - provider-backed 元数据管理视图
  - 最小前端承接面

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
  - 如涉及前端：
    `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx && bun run typecheck`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/knowledge.go`
  - `controller/agentplatform/knowledge.go`
  - `dto/agentplatform/knowledge.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `router/agentplatform-router.go`
  - `web/default/src/features/agent-platform/index.tsx`
  - `web/default/src/features/agent-platform/api.ts`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.1: 建立带 provider 元数据的 Knowledge 管理能力]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-9]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#387-398]
- [Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `4.1` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Knowledge 管理边界上继续漂移。
- 已加载 Story AP-4.1 验收标准、Knowledge typed detail 模型和现有 Agent Platform shell，准备补 Knowledge 控制面入口与 provider 元数据视图。

### Completion Notes List

- 已新增 Knowledge 专用 control-plane service/controller/dto 与 `/api/agent-platform/knowledge-bases` 路由，复用通用 resource registry 基线。
- 已实现 Knowledge 的列表、读取、创建、更新基础入口，并确保只返回 `resource_type=knowledge`。
- 已把 Agent Platform shell 中的 Knowledge 区域升级为真实列表承接面，开始展示 live Knowledge 管理入口而非纯静态占位。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`、`bun test src/features/agent-platform/agent-platform.test.tsx`、`bun run typecheck` 验证。
- 轻量 CC 复盘后确认：本 story 的 `done` 表示 Knowledge 控制面能力与最小前端承接已落地，不等于 Agent Platform 控制面真实 route/API/i18n/UI parity 已全部收口；这些跨域集成项由后续 stabilization story 统一补齐。

### File List

- `_bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md`
- `service/agentplatform/knowledge.go`
- `service/agentplatform/knowledge_test.go`
- `controller/agentplatform/knowledge.go`
- `controller/agentplatform/knowledge_test.go`
- `dto/agentplatform/knowledge.go`
- `router/agentplatform-router.go`
- `web/default/src/features/agent-platform/api.ts`
- `web/default/src/features/agent-platform/index.tsx`

## Change Log

- 2026-06-01: 完成 Knowledge 控制面基础管理入口、最小前端承接面与定向测试，并将故事推进为 done。
- 2026-06-01: 轻量 CC 复盘补充完成口径说明，明确跨 story 的前端集成收口留给后续 stabilization story。
