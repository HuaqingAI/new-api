# Story 2.5: 强制 freshness、revoke 与契约版本收敛规则

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want freshness、`ETag` / `Version`、refresh 和 revoke 行为具备明确契约语义,
so that 客户端可以一致地收敛发布投影，平台也能诊断陈旧状态问题。

## Acceptance Criteria

1. **Given** 客户端拿到某个 published projection  
   **When** 它读取契约元数据  
   **Then** 投影包含面向消费者可读的 `TTL`、`freshness`、`version`、`etag` 等信息  
   **And** freshness 状态至少支持 `fresh`、`stale`、`offline`、`revoked`。
2. **Given** 某投影已经超过 `TTL` 且尚未完成对账  
   **When** 客户端继续使用旧投影  
   **Then** 该投影必须被视为 `stale`  
   **And** 超过收敛窗口继续使用会被诊断为客户端不合规，而不是静默 best effort。
3. **Given** 资源发生 revoke、rollback、disable 或 offline  
   **When** 平台或客户端执行 refresh  
   **Then** 收敛必须在下一次 refresh 或 300 秒上限内完成  
   **And** 平台能解释问题来自缓存时序、刷新时序还是客户端实现错误。
4. **Given** 平台引入 breaking contract change  
   **When** 新契约准备发布  
   **Then** 平台必须要求显式 contract version 变更  
   **And** 兼容判断基于客户端声明的支持版本与能力集合完成。

## Tasks / Subtasks

- [x] 把 2.4 的 freshness 字段升级为可诊断收敛契约 (AC: 1, 2, 3)
  - [x] 在 open capability detail/refresh 响应中明确返回 `resource_version`、`etag`、`freshness_ttl_seconds`、`freshness`，并新增面向客户端对账的请求字段（如上次 `etag` / `resource_version` / `observed_at`）。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#582-596]
  - [x] 当投影超过 `TTL` 且客户端未持有匹配 `etag` / `version` 时，平台将其判定为 `stale`，不继续默认为 `fresh`。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#221]
  - [x] refresh 结果区分 `fresh`、`stale`、`offline`、`revoked`，并给出最小 diagnostics 字段说明当前不一致来自哪一层。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#593-596]

- [x] 增强 revoke / offline / rollback 后的收敛诊断 (AC: 2, 3)
  - [x] 当 exposure/resource 状态导致投影不再可用时，refresh/detail 稳定映射到 `revoked` / `offline` 等语义，不再只返回模糊 permission denied。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#582-596]
  - [x] 增加“超出 300 秒收敛窗口仍未对齐”的诊断信号，标记为客户端不合规或 stale reconciliation failure。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#595-596]
  - [x] 继续保持统一 error envelope，不把收敛诊断退回成散乱字符串。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#638-670]

- [x] 把 contract version 兼容判断接入开放能力层 (AC: 4)
  - [x] open capability detail/refresh 前校验 client 声明的 `contract_version` 与资源版本的 `contract_version` 是否兼容；breaking change 不兼容时返回 `contractInvalid`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#602-608; #666-667]
  - [x] 平台不让 breaking contract change 在不变更 `contract_version` 的情况下发布为“兼容”；至少显式阻断不兼容 client 读取 detail/invoke/query。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.5]
  - [x] 保持 client-specific 扩展仍在 namespaced `extensions` 下，不让扩展字段绕开 contract version 治理。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#608]

- [x] 扩展测试与最小文档 (AC: 1, 2, 3, 4)
  - [x] 更新 `service/agentplatform/discovery_test.go`、`controller/agentplatform/open_capabilities_test.go`，覆盖 stale 判定、revoke/offline 诊断、contract version 不兼容拦截、refresh 对账输入输出。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 最小同步 `docs/openapi/api.json` 中 refresh/detail 结构，反映新的 diagnostics / reconciliation 字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- `2.4` 已经完成 open capability bearer auth、discovery/detail/refresh 和统一 error envelope，因此 `2.5` 应在该基础上补“强约束的 freshness / contract-version 语义”，而不是新建第二套开放能力层。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md]
- `2.3` 已经完成 JWT access token 与 client 主模型，因此 contract-version 兼容判断可以同时参考 token claims、client registration 与 resource version。[Source: _bmad-output/implementation-artifacts/ap-2-3-implement-token-lifecycle-and-revocation.md]

### Architecture Constraints

- `refresh` / revoke 的收敛窗口是 hard contract，不是缓存提示：
  - default TTL upper bound = 300 seconds
  - stale beyond convergence window = client non-compliance
- `contract_version` 是显式治理语义，不是可忽略展示字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#582-608]

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform ./middleware ./router`

### Project Structure Notes

- 预计修改文件：
  - `service/agentplatform/discovery.go`
  - `service/agentplatform/open_capability_errors.go`
  - `dto/agentplatform/open_capabilities.go`
  - `controller/agentplatform/open_capabilities.go`
  - `middleware/agent_platform_bearer.go`（若要补 client contract context）
  - `docs/openapi/api.json`
- 预计新增文件：
  - 如需要可新增 `service/agentplatform/token_validate_test.go` 或等价测试文件

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 2.5: 强制 freshness、revoke 与契约版本收敛规则]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#582-608]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#620-667]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#218-222]
- [Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `2.5` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Agent Platform Epic 2 上继续漂移。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并基于 `2.4` 的 open capability 层继续补 freshness / contract-version 收敛规则。

### Completion Notes List

- 已将 open capability detail/refresh 升级为带 diagnostics 的收敛契约，支持客户端提交 `observed_etag`、`observed_resource_version`、`observed_at` 做最小对账。
- 已实现 stale / revoked / offline 的稳定 freshness 语义，并在超过收敛窗口时返回 `client_non_compliant_stale` 诊断。
- 已将 client `contract_version` 与 resource version `contract_version` 的兼容判定接入 detail/refresh，不兼容时稳定返回 `contractInvalid`。
- 已扩展 service/controller 定向测试，验证 stale 判定、contract version 不兼容拦截和 refresh diagnostics。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform ./middleware ./router`。

### File List

- `_bmad-output/implementation-artifacts/ap-2-5-enforce-freshness-revoke-and-contract-version-convergence.md`
- `service/agentplatform/discovery.go`
- `service/agentplatform/discovery_test.go`
- `dto/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`

## Change Log

- 2026-06-01: 完成 freshness stale 判定、refresh diagnostics、contract-version 兼容校验与定向测试，并将故事推进为 done。
