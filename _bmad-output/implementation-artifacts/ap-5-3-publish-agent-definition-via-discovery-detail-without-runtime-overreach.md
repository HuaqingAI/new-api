# Story AP-5.3: 通过 discovery/detail 发布 Agent 定义而不越界到 runtime

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want 通过标准能力层发现并读取 `Agent` 定义,
so that 我可以消费可复用的 `Agent` 元数据，而不要求平台替我执行该 `Agent`。

## Acceptance Criteria

1. **Given** 某个 `Agent` 定义已经对客户端发布  
   **When** 客户端执行 discovery 与 detail  
   **Then** 该 `Agent` 通过与其他资源一致的标准能力层暴露  
   **And** 返回数据能清晰表达其定义与依赖边界。
2. **Given** 客户端或平台需要判断该 `Agent` 是否 callable  
   **When** 依赖状态或兼容规则不满足  
   **Then** 平台可以将其判定为 visible but not callable  
   **And** 判定原因使用共享、可诊断的语义，而不是泛化失败提示。
3. **Given** MVP 边界需要被最终验证  
   **When** 本 Epic 完成  
   **Then** 首个 `Agent` 验证闭环覆盖 publish -> discovery -> detail -> dependency/reference-boundary validation -> audit trail  
   **And** 不要求平台交付任何服务端 `Agent` runtime 或 orchestration engine。

## Tasks / Subtasks

- [x] 将 Agent detail 升级为依赖边界友好的开放能力输出 (AC: 1, 2, 3)
  - [x] 在 `service/agentplatform/discovery.go` 中对 `resource_type=agent` 的 detail 读取增加了专用 Agent 边界数据映射，清晰返回 manifest、dependencies、compatibility metadata。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1004-1008]
  - [x] 保持 Agent detail 是 definition/template 语义，没有在任何响应里暗示服务端 runtime 或 orchestration ownership。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#899-904]
  - [x] `/api/open-capabilities/agents/:id` 不再只是复用通用 resource detail，而是带着 Agent 依赖边界语义输出更清晰的 detail 结构。[Source: controller/agentplatform/open_capabilities.go]

- [x] 把 visible-but-not-callable 判定接入 Agent 依赖边界 (AC: 2, 3)
  - [x] 当 Agent 的某个依赖未发布、被撤销、状态不兼容或对当前客户端不可见时，平台会将该 Agent 判定为 visible but not callable，而不是简单 permission denied。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-5.3]
  - [x] 复用现有 `callable_state` / diagnostics 语义表达 `contract_invalid`、依赖未满足等原因，让客户端和平台都能读出边界失败来源。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337; #631]
  - [x] 没有引入任何服务端 Agent 执行能力；依赖边界校验仅服务于 definition 可读性和可用性判定。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#SM-6]

- [x] 补开放能力测试，锁住最终闭环 (AC: 1, 2, 3)
  - [x] 更新 `controller/agentplatform/open_capabilities_test.go`，覆盖：已发布 Agent 的 discovery/detail、依赖满足时 detail 正常返回、依赖不满足时 visible-but-not-callable/contractInvalid 诊断。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 保持当前统一 error envelope、freshness、contract version 语义继续生效。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md; ap-2-5-enforce-freshness-revoke-and-contract-version-convergence.md]

## Dev Notes

### Previous Story Insights

- `5.1` 已建立 Agent control-plane 管理入口，`5.2` 已完成依赖声明与存在性校验，因此 `5.3` 的重点是把这些边界真正通过 open capability detail 暴露给客户端。[Source: _bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md; ap-5-2-implement-agent-dependency-declaration-and-boundary-validation.md]
- 当前 `/api/open-capabilities/agents/:id` 只是复用通用 resource detail，这是本 story 要升级的直接入口。[Source: controller/agentplatform/open_capabilities.go]

### Scope Boundaries

- `5.3` 不实现：
  - Agent runtime / orchestration
  - Agent invoke/query 运行时
  - 额外的 runtime state surface
- `5.3` 要实现：
  - Agent discovery/detail 的边界语义
  - dependency-aware callable 判定
  - 最终 Epic AP-5 的验证闭环

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计修改文件：
  - `service/agentplatform/discovery.go`
  - `controller/agentplatform/open_capabilities.go`
  - `controller/agentplatform/open_capabilities_test.go`
  - 如需要，`dto/agentplatform/open_capabilities.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-5.3: 通过 discovery/detail 发布 Agent 定义而不越界到 runtime]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1004-1008]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#899-904]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#SM-6]
- [Source: _bmad-output/implementation-artifacts/ap-5-2-implement-agent-dependency-declaration-and-boundary-validation.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `5.3` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent detail/runtime 边界上继续漂移。
- 已加载 Story AP-5.3 验收标准、Agent dependency validation 结果与现有 open capability 层，准备接通 Agent detail 的最终边界输出。

### Completion Notes List

- 已将 Agent detail 升级为依赖边界友好的开放能力输出，明确返回 manifest、dependencies 与 compatibility metadata。
- 已把 Agent 的依赖状态接入 callable 判定；当依赖不可见或不可调时，Agent 会表现为 visible but not callable，并返回共享 diagnostics 原因。
- 已完成开放能力层的最终 Agent 测试闭环，覆盖已发布 detail 和依赖不满足时的诊断输出。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-5-3-publish-agent-definition-via-discovery-detail-without-runtime-overreach.md`
- `service/agentplatform/discovery.go`
- `controller/agentplatform/open_capabilities_test.go`

## Change Log

- 2026-06-01: 完成 Agent discovery/detail 的最终边界闭环与依赖可调用性判定，并将故事推进为 done。
