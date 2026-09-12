# Story AP-5.2: 实现 Agent 依赖声明与边界校验

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 让 `Agent` 显式声明其 `Skill` 与 `Knowledge` 依赖,
so that 平台与消费者都能在使用前理解其组合边界。

## Acceptance Criteria

1. **Given** 某个 `Agent` 定义正在编辑  
   **When** 发布者声明依赖项  
   **Then** 平台保存对 `Skill` 和 `Knowledge` 的依赖引用  
   **And** 这些依赖在控制面中可被查看与检查。
2. **Given** 某个依赖缺失、被撤销、不兼容或未发布给目标客户端  
   **When** 平台校验该 `Agent` 的发布或可用性  
   **Then** 平台阻止它被视为稳定可调用资源  
   **And** 校验失败会明确指出是哪条依赖边界不满足。
3. **Given** 某客户端请求 `Agent` detail  
   **When** detail 响应返回  
   **Then** 客户端能够理解该 `Agent` 的依赖边界与组合元数据  
   **And** 不需要自行猜测平台并不拥有的隐式 runtime 行为。

## Tasks / Subtasks

- [x] 收紧 Agent typed detail 的依赖声明结构 (AC: 1, 2, 3)
  - [x] 在 `model/agentplatform/agent_def.go` 中补充最小业务约束：`manifest_json`、`dependencies_json`、`compatibility_meta_json` 不再只是“任意文本”；空依赖声明或非法 JSON 会被阻止进入受治理版本。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#280]
  - [x] 依赖声明必须显式表达对 `Skill` / `Knowledge` 的引用，而不是隐式字符串约定或外部文档。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-12]

- [x] 在 Agent resource version / detail 路径中增加依赖边界校验 (AC: 1, 2, 3)
  - [x] `service/agentplatform/resource_version.go` 在创建 Agent 版本时校验依赖声明结构，拒绝缺失依赖类型、缺失目标 resource id、或引用不存在资源的 Agent detail。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-5.2]
  - [x] 对已存在但结构不满足要求的依赖给出明确 validation failure，避免 Agent 被视为稳定可调用资源；更深的发布/兼容状态校验继续留给 5.3 的开放能力层判定。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1004-1008]
  - [x] 兼容性元数据继续保留在 definition/template 语义内，没有越界为 runtime 执行状态。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#899-904]

- [x] 让 Agent detail 能表达依赖边界给控制面与后续开放能力层 (AC: 1, 3)
  - [x] Agent detail 查询结果继续稳定返回 dependencies 与 compatibility metadata，供控制面和后续 discovery/detail 消费。[Source: service/agentplatform/resource_version.go]
  - [x] 本故事没有实现 open capability Agent detail 路由改造，但已经让后续 5.3 能直接复用这些依赖边界数据。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-5.3]

- [x] 补定向测试，锁住依赖校验行为 (AC: 1, 2, 3)
  - [x] 更新 `service/agentplatform/resource_version_test.go`、`controller/agentplatform/resource_version_test.go`，覆盖：合法依赖可保存、依赖缺失/类型错误/资源不存在被拒绝。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 当前 control-plane 响应结构无高信号变化，暂不更新 OpenAPI 文档，优先保证行为与测试一致。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- `5.1` 已经建立 Agent 专用 control-plane 管理入口，因此 `5.2` 应聚焦依赖声明和边界校验，不需要再扩写 Agent CRUD 主流程。[Source: _bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md]
- 现有 `agent_def.dependencies_json` 已存在，但尚未成为真正的受校验依赖边界，因此这是本 story 的直接落点。[Source: model/agentplatform/agent_def.go]

### Scope Boundaries

- `5.2` 不实现：
  - open capability Agent detail 路由改造
  - Agent runtime / orchestration
  - dependency-aware callable state 的最终对外暴露（留给 5.3）
- `5.2` 要实现：
  - 受治理的依赖声明结构
  - 依赖存在性/状态基础校验
  - control-plane 可读的边界数据

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计修改文件：
  - `model/agentplatform/agent_def.go`
  - `service/agentplatform/resource_version.go`
  - `service/agentplatform/resource_version_test.go`
  - `controller/agentplatform/resource_version_test.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-5.2: 实现 Agent 依赖声明与边界校验]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-12]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#280]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1004-1008]
- [Source: _bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `5.2` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent 依赖边界语义上继续漂移。
- 已加载 Story AP-5.2 验收标准、Agent typed detail 模型和依赖边界相关架构约束，准备补依赖声明与边界校验。

### Completion Notes List

- 已收紧 `agent_def` 的最小业务校验，要求 manifest / dependencies / compatibility metadata 均为有效结构化 JSON。
- 已将 Agent 依赖声明校验接入 `resource_version` 创建路径，依赖缺失、类型错误、资源不存在等情况会被阻断为 `ErrAgentDependencyInvalid`。
- 已补 service/controller 定向测试，锁住合法依赖可保存、非法依赖被拒绝的行为。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-5-2-implement-agent-dependency-declaration-and-boundary-validation.md`
- `model/agentplatform/agent_def.go`
- `service/agentplatform/resource_version.go`
- `service/agentplatform/resource_version_test.go`
- `controller/agentplatform/resource_version.go`
- `controller/agentplatform/resource_version_test.go`

## Change Log

- 2026-06-01: 完成 Agent 依赖声明与边界校验及定向测试，并将故事推进为 done。
