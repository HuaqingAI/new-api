# Story 3.3: 通过开放能力层暴露 Skill 的 discovery/detail/invoke

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want 通过共享能力层发现、查看并调用 `Skill`,
so that 首个消费者和后续消费者都能走同一条集成路径。

## Acceptance Criteria

1. **Given** 客户端已授权且目标 `Skill` 已对其发布  
   **When** 客户端执行 discovery 与 detail  
   **Then** 它可以发现该 `Skill` 并读取标准化契约数据  
   **And** 平台按客户端规则强制执行 visibility/callability 校验。
2. **Given** 客户端调用某个 `Skill`  
   **When** 调用请求合法且已授权  
   **Then** 平台通过标准化 invoke 面执行业务调用  
   **And** 成功与失败都使用统一响应/错误语义。
3. **Given** 调用因参数校验、权限不足、资源撤销、资源下线、超时或上游失败而失败  
   **When** 平台返回失败响应  
   **Then** 失败会映射到稳定的标准错误类别  
   **And** 结果可以被审计和诊断，而不是只能依赖零散日志。

## Tasks / Subtasks

- [x] 将 Skill detail 接入真正的 open-capabilities invoke 路径 (AC: 1, 2, 3)
  - [x] 在 `service/agentplatform` 中新增 Skill invoke 服务，读取 `resource_version` + `skill_def` + `exposure`，以 open capability 授权与发布状态为前提执行调用。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1104]
  - [x] invoke 前强制执行 visibility / callability / contract version / readiness 校验，避免未发布、已撤销、已下线或契约不完整的 Skill 被调用。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.3; _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md]
  - [x] 继续沿用统一 error envelope，没有为 Skill invoke 单独定义另一套错误模型。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#313]

- [x] 为 MVP 提供最小可运行的 Skill binding adapter (AC: 2, 3)
  - [x] 基于 `binding_config_json` 提供了一个受控的 HTTP invoke 适配形态，支持最小 `method` / `url` / `headers` / `timeout_seconds` 组合，让 Skill invoke 具备真实调用闭环。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#278]
  - [x] 调用成功时返回标准 success envelope；调用失败时根据场景映射到 `contractInvalid`、`permissionDenied`、`resourceRevoked`、`resourceOffline`、`timeout`、`upstreamFailed` 等稳定类别。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#656-661]
  - [x] Skill invoke adapter 只服务 Skill MVP，没有扩展成 Knowledge/Agent 通用 runtime。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.3]

- [x] 扩展 open capability controller / DTO / 测试 (AC: 1, 2, 3)
  - [x] 将 `controller/agentplatform/open_capabilities.go` 中 `OpenCapabilitySkillInvoke` 从占位 `contractInvalid` 改成真实调用入口。[Source: controller/agentplatform/open_capabilities.go]
  - [x] 当前直接复用了标准 open capability success envelope + `output` 结构，未额外引入复杂 DTO，保持 MVP 简洁可诊断。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-8]
  - [x] 更新 `controller/agentplatform/open_capabilities_test.go` 并新增 `service/agentplatform/skill_invoke_test.go`，覆盖：成功 invoke、未发布/不可调拦截语义、timeout、upstream failed。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]

## Dev Notes

### Previous Story Insights

- `2.4/2.5` 已经把 Skill 的 discovery/detail、bearer auth、统一 envelope 和 freshness / contract-version 规则搭好，所以 `3.3` 只需要把 invoke 真正接通，不要重造开放能力层。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md; ap-2-5-enforce-freshness-revoke-and-contract-version-convergence.md]
- `3.2` 已经保证 Skill contract readiness，所以 `3.3` 可以把 invoke 前置校验建立在 `skill_def` 有效这一前提上。[Source: _bmad-output/implementation-artifacts/ap-3-2-define-skill-detail-schema-and-invoke-readiness.md]

### Scope Boundaries

- 本故事目标是 Skill invoke MVP，不实现：
  - Knowledge query
  - Agent runtime
  - 复杂 workflow orchestration
- 允许采用最小 HTTP invoke adapter 完成首个闭环，但保持统一协议面和错误语义稳定。

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/skill_invoke.go`
  - `service/agentplatform/skill_invoke_test.go`
  - 如需要可补 DTO 文件
- 预计修改文件：
  - `controller/agentplatform/open_capabilities.go`
  - `controller/agentplatform/open_capabilities_test.go`
  - `service/agentplatform/open_capability_errors.go`
  - `docs/openapi/api.json`（如 invoke payload 结构需要最小同步）

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.3: 通过开放能力层暴露 Skill 的 discovery/detail/invoke]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-8]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#278]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#570]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#656-661]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `3.3` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Skill invoke runtime 边界上继续漂移。
- 已加载 Story 3.3 验收标准、现有 open capability 层、Skill contract readiness 和统一错误 envelope，准备接通 Skill invoke 最小闭环。

### Completion Notes List

- 已新增 `service/agentplatform/skill_invoke.go`，通过最小 HTTP binding adapter 让 Skill invoke 具备真实执行闭环。
- 已将 `/api/open-capabilities/skills/:id/invoke` 从占位实现接成真实调用入口，并复用现有 open capability 授权、发布状态和 contract version 校验。
- 已将 timeout / upstream failed 映射到统一 error envelope 中的稳定错误码。
- 已完成 service/controller 定向测试，覆盖成功调用、上游失败等主路径。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-3-3-expose-skill-discovery-detail-and-invoke-via-open-capabilities.md`
- `service/agentplatform/skill_invoke.go`
- `service/agentplatform/skill_invoke_test.go`
- `service/agentplatform/open_capability_errors.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`

## Change Log

- 2026-06-01: 完成 Skill 的开放能力 invoke MVP、HTTP binding adapter、统一错误映射与定向测试，并将故事推进为 done。
