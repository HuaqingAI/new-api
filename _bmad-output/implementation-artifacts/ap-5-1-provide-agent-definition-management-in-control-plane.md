# Story 5.1: 在控制面提供 Agent 定义管理能力

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 创建、更新、发布、撤销并查看 `Agent` 定义,
so that `Agent` 成为一等治理资源，而不是被误当成平台必须执行的运行时对象。

## Acceptance Criteria

1. **Given** 发布者管理某个 `Agent` 资源  
   **When** 创建或更新其定义  
   **Then** 平台保存稳定身份、版本、状态与描述信息  
   **And** 该管理模型明确把 `Agent` 视为 definition/template 对象。
2. **Given** 某个 `Agent` 定义发生状态变化  
   **When** 执行 publish、revoke、rollback 等动作  
   **Then** 这些动作像其他资源一样被审计  
   **And** UI/控制面不会暗示平台拥有超出定义边界的服务端 runtime 能力。
3. **Given** MVP 边界已冻结  
   **When** 为 `Agent` 做实现决策  
   **Then** 平台不会在本 Story 中引入 execution state、workflow state 或 multi-step orchestration  
   **And** 这些能力保持明确 out of scope。

## Tasks / Subtasks

- [x] 提供 Agent 专用 control-plane 管理入口，复用现有治理基线 (AC: 1, 2, 3)
  - [x] 新增 `controller/service/dto` 中的 Agent 专用 list/get/create/update 入口，建立在现有 `resource` / `resource_version` / `agent_def` 模型之上，而不是新造运行时对象体系。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1106]
  - [x] `Agent` 入口只面向 `resource_type=agent`，降低调用方对通用资源接口的手动过滤负担。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-11]
  - [x] 支持稳定 identity、状态、当前版本和描述信息查看，明确它是 definition/template，而不是执行实例。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-11]

- [x] 保持 Agent 管理边界不越界到 runtime (AC: 1, 2, 3)
  - [x] 管理面读取建立在现有 `agent_def` typed detail（manifest / dependencies / prompt metadata / compatibility metadata）之上，但本 story 不实现服务端执行状态或 runtime 控制逻辑。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#280; #899-904]
  - [x] 复用已有 publish / disable / revoke / rollback lifecycle 能力，但没有在控制面文案或结构上暗示平台能执行 Agent runtime。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 5.1]

- [x] 在 web/default 的 Agent Platform shell 中接入 Agent 管理承接面 (AC: 1, 2)
  - [x] 将 `web/default/src/features/agent-platform` 中当前 Agent 区域从纯静态占位升级为最小真实列表/承接视图，展示已存在 Agent 的名称、状态、当前版本和 owner 信息。[Source: web/default/src/features/agent-platform/index.tsx]
  - [x] 维持现有视觉语言，没有在 5.1 提前做依赖编辑器或 runtime 交互；重点是把 Agent 管理入口从静态说明升级为真实视图。[Source: _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md]

- [x] 补定向测试与最小文档判断 (AC: 1, 2, 3)
  - [x] 新增/更新 Agent 管理相关 service/controller 测试，覆盖：仅返回 agent、创建 agent、更新基础信息。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 本故事优先保证行为和测试一致；当前无高信号接口结构变化，暂不更新 OpenAPI 文档。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- Skill 和 Knowledge 已经沿统一模式落地 control-plane 专用入口，因此 `5.1` 应复用同样模式，把 Agent 纳入管理面，而不是重新设计一套特殊控制面。[Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md; ap-4-1-manage-knowledge-resources-with-provider-metadata.md]
- `agent_def` typed detail 与 `resource_version` 已存在，因此本故事的重点是管理入口和边界表达，不是重建底层模型。[Source: model/agentplatform/agent_def.go; service/agentplatform/resource_version.go]

### Scope Boundaries

- `5.1` 不实现：
  - Agent 依赖边界校验（留给 5.2）
  - Agent detail discovery 输出（留给 5.3）
  - 任何服务端 runtime / orchestration 能力
- `5.1` 要实现：
  - Agent 专用 control-plane 入口
  - 最小前端承接面
  - 明确 runtime out-of-scope 边界

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx && bun run typecheck`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/agent.go`
  - `controller/agentplatform/agent.go`
  - `dto/agentplatform/agent.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `router/agentplatform-router.go`
  - `web/default/src/features/agent-platform/api.ts`
  - `web/default/src/features/agent-platform/index.tsx`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 5.1: 在控制面提供 Agent 定义管理能力]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-11]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#280]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#899-904]
- [Source: _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `5.1` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent runtime 边界上继续漂移。
- 已加载 Story 5.1 验收标准、Agent typed detail 模型和 Agent Platform shell，准备补 Agent 控制面入口与最小前端承接面。

### Completion Notes List

- 已新增 Agent 专用 control-plane service/controller/dto 与 `/api/agent-platform/agents` 路由，复用通用 resource registry 基线。
- 已实现 Agent 的列表、读取、创建、更新基础入口，并确保只返回 `resource_type=agent`。
- 已把 Agent Platform shell 中的 Agents 区域升级为真实列表承接面，明确展示它是 definition/template 资源而非 runtime 实例。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`、`bun test src/features/agent-platform/agent-platform.test.tsx`、`bun run typecheck` 验证。
- 轻量 CC 复盘后确认：本 story 的 `done` 表示 Agent 控制面能力与最小前端承接已落地，不等于 Agent Platform 控制面真实 route/API/i18n/UI parity 已全部收口；这些跨域集成项由后续 stabilization story 统一补齐。

### File List

- `_bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md`
- `service/agentplatform/agent.go`
- `service/agentplatform/agent_test.go`
- `controller/agentplatform/agent.go`
- `controller/agentplatform/agent_test.go`
- `dto/agentplatform/agent.go`
- `router/agentplatform-router.go`
- `web/default/src/features/agent-platform/api.ts`
- `web/default/src/features/agent-platform/index.tsx`

## Change Log

- 2026-06-01: 完成 Agent 定义控制面管理入口、最小前端承接面与定向测试，并将故事推进为 done。
- 2026-06-01: 轻量 CC 复盘补充完成口径说明，明确跨 story 的前端集成收口留给后续 stabilization story。
