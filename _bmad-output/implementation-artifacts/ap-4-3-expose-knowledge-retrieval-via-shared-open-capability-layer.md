# Story AP-4.3: 通过共享能力层暴露 Knowledge 检索行为

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want 像使用其他资源一样，通过共享标准层发现并查询 `Knowledge`,
so that retrieval 能成为可复用的平台能力，而不是消费者特例。

## Acceptance Criteria

1. **Given** 客户端已授权且某个 `Knowledge` 已对其发布  
   **When** 客户端执行 discovery 与 detail  
   **Then** 该资源会通过共享能力层返回标准化元数据  
   **And** exposure 规则继续决定其 visibility 与 callability。
2. **Given** 客户端发起 retrieval query  
   **When** 查询合法且资源可调用  
   **Then** 平台返回结构化 retrieval 结果，其中包含来源元数据与排序信息  
   **And** 响应使用统一 success/error 语义。
3. **Given** 查询因为契约错误、权限问题、撤销、下线、超时或 provider 失败而失败  
   **When** 平台返回失败结果  
   **Then** 失败映射到共享错误 envelope  
   **And** 诊断信息不会泄露 provider secret 或敏感配置。

## Tasks / Subtasks

- [x] 将 Knowledge detail / visibility 规则接入真正的 query 路径 (AC: 1, 2, 3)
  - [x] 在 `service/agentplatform` 中新增 Knowledge query 服务，读取 `resource_version` + `knowledge_def` + `exposure`，以 open capability 授权与发布状态为前提执行业务查询。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1105]
  - [x] query 前强制执行 visibility / callability / contract version / retrieval-only 契约校验，避免未发布、已撤销、已下线或契约不完整的 Knowledge 被查询。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md; ap-4-2-freeze-knowledge-mvp-contract-to-standardized-retrieval.md]
  - [x] 继续沿用统一 error envelope，没有为 Knowledge query 单独定义另一套错误语义。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#313]

- [x] 为 MVP 提供最小 provider-neutral retrieval adapter (AC: 2, 3)
  - [x] 基于 `knowledge_def` 中的 provider binding，先提供了一个最小 query adapter 入口，返回标准化 retrieval 结果结构：包含 `items` / `citations` 的公共语义。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#996-1001; #1060]
  - [x] 在 4.3 范围内，query adapter 先支持最小内建/mock 路径，真正的外部 `http_retrieval` provider 接入继续留给 4.4。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.4]
  - [x] 调用成功时返回标准 success envelope；调用失败时根据场景映射到 `contractInvalid`、`permissionDenied`、`resourceRevoked`、`resourceOffline`、`timeout`、`upstreamFailed` 等稳定类别。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#656-661]

- [x] 扩展 open capability controller / DTO / 测试 (AC: 1, 2, 3)
  - [x] 将 `controller/agentplatform/open_capabilities.go` 中 `OpenCapabilityKnowledgeQuery` 从占位 `contractInvalid` 改成真实查询入口。[Source: controller/agentplatform/open_capabilities.go]
  - [x] 当前直接复用了标准 open capability success envelope + `items/citations` 结构，保持 MVP 简洁可诊断。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-10]
  - [x] 更新 `controller/agentplatform/open_capabilities_test.go` 并新增 `service/agentplatform/knowledge_query_test.go`，覆盖：成功 query、契约错误、统一结果结构。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]

## Dev Notes

### Previous Story Insights

- `4.1` 已建立 Knowledge 专用 control-plane 管理入口，`4.2` 已冻结 retrieval-only 契约，因此 `4.3` 只需要把查询路径接通，不必重做管理与契约边界。[Source: _bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md; ap-4-2-freeze-knowledge-mvp-contract-to-standardized-retrieval.md]
- 当前 `/api/open-capabilities/knowledge-bases/:id/query` 仍是占位 `contractInvalid`，这是本故事的直接切入点。[Source: controller/agentplatform/open_capabilities.go]

### Scope Boundaries

- `4.3` 不实现：
  - 外部 `http_retrieval` provider 真正接入
  - provider 切换矩阵
  - provider-specific response normalization beyond MVP retrieval result
- `4.3` 要实现：
  - Knowledge query 的最小真实闭环
  - retrieval result 的统一结构
  - 统一错误语义

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/knowledge_query.go`
  - `service/agentplatform/knowledge_query_test.go`
  - 如需要补 DTO 文件
- 预计修改文件：
  - `controller/agentplatform/open_capabilities.go`
  - `controller/agentplatform/open_capabilities_test.go`
  - `service/agentplatform/open_capability_errors.go`
  - `dto/agentplatform/open_capabilities.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.3: 通过共享能力层暴露 Knowledge 检索行为]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-10]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#996-1001]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1060]
- [Source: _bmad-output/implementation-artifacts/ap-4-2-freeze-knowledge-mvp-contract-to-standardized-retrieval.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `4.3` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Knowledge query runtime 边界上继续漂移。
- 已加载 Story AP-4.3 验收标准、retrieval-only 契约和现有 open capability 层，准备接通 Knowledge query 最小闭环。

### Completion Notes List

- 已新增 `service/agentplatform/knowledge_query.go`，提供 retrieval-only 的最小 Knowledge query 闭环。
- 已将 `/api/open-capabilities/knowledge-bases/:id/query` 从占位实现接成真实查询入口，并复用现有授权、发布状态和 contract version 校验。
- 已返回标准化 retrieval 结果结构，至少包含 `items` 和 `citations` 两类公共语义字段。
- 已完成 service/controller 定向测试，覆盖成功 query 和统一结果结构。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-4-3-expose-knowledge-retrieval-via-shared-open-capability-layer.md`
- `service/agentplatform/knowledge_query.go`
- `service/agentplatform/knowledge_query_test.go`
- `controller/agentplatform/open_capabilities.go`
- `controller/agentplatform/open_capabilities_test.go`

## Change Log

- 2026-06-01: 完成 Knowledge 的开放能力检索 MVP、统一 retrieval result 结构与定向测试，并将故事推进为 done。
