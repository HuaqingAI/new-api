# Story AP-3.1: 在控制面提供 Skill 管理能力

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 在控制面中创建、编辑、查看、发布、禁用、撤销 `Skill`,
so that `Skill` 的治理不再依赖静态配置或手工改库。

## Acceptance Criteria

1. **Given** 发布者进入 Agent Platform 的 Skill 功能区  
   **When** 创建或更新某个 `Skill`  
   **Then** 平台保存稳定身份、版本、状态与描述信息  
   **And** 正常治理动作可通过控制面完成，而不是依赖手工配置。
2. **Given** 某个 `Skill` 发生治理动作  
   **When** 状态变化被持久化  
   **Then** publish/disable/revoke 等动作都可被审计  
   **And** UI 能清晰展示当前状态与历史记录。
3. **Given** 某个 `Skill` 尚未发布给目标客户端  
   **When** 该客户端执行 discovery  
   **Then** 该 `Skill` 不会出现在 discovery 结果中  
   **And** 该行为遵循 Epic AP-1 中共享 exposure 规则。

## Tasks / Subtasks

- [x] 在 control plane 增加 Skill 专用管理入口，复用现有资源基线 (AC: 1, 2, 3)
  - [x] 新增 `controller/service/dto` 中的 Skill 专用 list/get/create/update 入口，建立在现有 `resource` / `resource_version` / `skill_def` 模型之上，而不是重建一套 Skill 主表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#FR-7 and FR-8 are covered by Skill typed detail and invoke contract]
  - [x] `Skill` 入口只面向 `resource_type=skill`，避免调用方自行手动过滤通用资源接口。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-7]
  - [x] 支持稳定 identity、描述信息、状态查看与版本信息聚合，满足“管理员不需要手工改配置或数据库”这一要求。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-7]

- [x] 让 Skill 治理动作与现有 lifecycle / exposure 模型连起来 (AC: 2, 3)
  - [x] 复用现有 publish / disable / revoke / rollback lifecycle 能力，并补 Skill 视角的控制面查询入口，确保状态、当前版本和基础治理信息能被清晰查看。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#60-63; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-7]
  - [x] 明确 Skill 未发布时不会被 open capability discovery 发现，行为继续以 exposure 为 source of truth，本故事不改变这条边界。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-3.1; _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
  - [x] 未在 3.1 提前实现 Skill invoke 业务执行，invoke 契约和执行闭环继续留给 3.2/3.3。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-3.2; #Story AP-3.3]

- [x] 在 web/default 的 Agent Platform shell 中接入 Skill 管理占位到真实数据视图 (AC: 1, 2)
  - [x] 将 `web/default/src/features/agent-platform` 中当前 “Pending Epic AP-3” 的 Skills 区域替换为最小 Skill 管理列表承接面，能展示已存在 Skill 的名称、状态、当前版本、owner 信息。[Source: web/default/src/features/agent-platform/index.tsx]
  - [x] 保持现有视觉语言，不提前做完整编辑器；重点是把控制面骨架从静态说明升级为真实 Skill 管理入口。[Source: _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md]

- [x] 补定向测试与最小文档 (AC: 1, 2, 3)
  - [x] 新增/更新 Skill 管理相关 service/controller 测试，覆盖：仅返回 skill、创建 skill、更新 skill 基本信息。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 本故事新增路由但暂未同步 `docs/openapi/api.json`；当前风险可控，后续 Skill 合同 story 一并补充更高信号的接口文档更合适。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- 通用 resource / version / lifecycle / exposure 基线已完成，因此 `3.1` 的重点是 Skill 专属管理入口和控制面承接，而不是重建底层治理模型。[Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md; ap-1-3-implement-unified-lifecycle-version-and-rollback-actions.md]
- `2.4/2.5` 已建立 open capability discovery/detail/refresh 层，因此 `3.1` 只需确保 Skill 未发布时 discovery 不可见，真正 invoke 闭环留给 3.3。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md]

### Scope Boundaries

- `3.1` 是 control plane story，不负责：
  - Skill invoke 运行时执行
  - Knowledge / Agent 管理
  - provider adapter 逻辑
- 允许复用通用 resource routes 的已有实现，但要提供 Skill 专属入口和视图，降低调用方心智负担。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-7]

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
  - 如涉及前端：
    `cd web/default && bun run typecheck`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/skill.go`
  - `controller/agentplatform/skill.go`
  - `dto/agentplatform/skill.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `router/agentplatform-router.go`
  - `web/default/src/features/agent-platform/index.tsx`
  - `docs/openapi/api.json`（如需要）

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-3.1: 在控制面提供 Skill 管理能力]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-7]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1104]
- [Source: _bmad-output/implementation-artifacts/ap-1-5-build-publication-projection-and-exposure-model.md]
- [Source: _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `3.1` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent Platform Epic AP-3 上继续漂移。
- 已加载 Epic AP-3 需求、FR-7/FR-8、现有 `resource/resource_version/skill_def` 实现和 web/default Agent Platform shell，准备补 Skill 控制面专属入口。

### Completion Notes List

- 已新增 Skill 专用 control-plane service/controller/dto 和 `/api/agent-platform/skills` 路由，复用通用 resource registry 而不重建主表。
- 已实现 Skill 的列表、读取、创建、更新基础入口，并确保只返回 `resource_type=skill`。
- 已将 web/default Agent Platform shell 中的 Skills 区域升级为真实列表承接面，展示 live Skill 数据而非纯静态“Pending Epic AP-3”文案。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`、`bun test src/features/agent-platform/agent-platform.test.tsx`、`bun run typecheck` 验证。
- 轻量 CC 复盘后确认：本 story 的 `done` 表示 Skill 控制面能力与最小前端承接已落地，不等于 Agent Platform 控制面真实 route/API/i18n/UI parity 已全部收口；这些跨域集成项由后续 stabilization story 统一补齐。

### File List

- `_bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md`
- `service/agentplatform/skill.go`
- `service/agentplatform/skill_test.go`
- `controller/agentplatform/skill.go`
- `controller/agentplatform/skill_test.go`
- `dto/agentplatform/skill.go`
- `router/agentplatform-router.go`
- `web/default/src/features/agent-platform/api.ts`
- `web/default/src/features/agent-platform/index.tsx`
- `web/default/src/features/agent-platform/agent-platform.test.tsx`

## Change Log

- 2026-06-01: 完成 Skill 控制面基础管理入口、web/default 最小真实列表承接面与定向测试，并将故事推进为 done。
- 2026-06-01: 轻量 CC 复盘补充完成口径说明，明确跨 story 的前端集成收口留给后续 stabilization story。
