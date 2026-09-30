# Story AP-4.4: 接入首个 `http_retrieval` provider 适配器

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 平台管理员,
I want 通过 provider-neutral 的 `http_retrieval` adapter 接入首个外部检索引擎,
so that 平台可以验证 Knowledge 契约闭环，而不会把公共接口绑定到单一知识引擎实现。

## Acceptance Criteria

1. **Given** 团队在 `LightRAG`、`FastGPT`、`RAGFlow` 等候选中选择首个试点 provider  
   **When** 适配器被实现  
   **Then** 平台通过统一 adapter 接口完成 query 调用、错误映射与 citation 归一化  
   **And** 切换到另一个 `http_retrieval` provider 不要求修改公共 API 契约。
2. **Given** provider 发生超时、鉴权失败或响应结构变化  
   **When** 平台处理失败  
   **Then** 平台能把失败归因到 provider 适配层  
   **And** 对客户端保持稳定的错误 envelope 与诊断语义。

## Tasks / Subtasks

- [x] 抽出 provider-neutral 的 Knowledge query adapter 接口 (AC: 1, 2)
  - [x] 在 `service/agentplatform` 中新增 `KnowledgeProvider` 抽象，覆盖：binding 校验、query、refresh、health 等最小契约边界。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#889-893]
  - [x] `KnowledgeQueryService` 不再内嵌 mock 数据生成逻辑，而是通过 provider adapter 获取 retrieval 结果，再做公共结果封装。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#996-1001]
  - [x] provider adapter 的选择继续建立在 `knowledge_def` 的 `provider_type=http_retrieval` / `provider_adapter_key` 上，没有把具体引擎名泄露到公共 API 契约中。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#277]

- [x] 实现首个 `http_retrieval` provider adapter (AC: 1, 2)
  - [x] 基于 `provider_config_json` 实现了最小 HTTP query 调用，支持 endpoint、method、headers 等必要配置，并把 query payload 发给外部 provider。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#400-407]
  - [x] 将 provider 返回结果归一化成共享 retrieval 结果结构，输出 `items` 与 `citations` 的统一语义，不直接暴露 provider 原始结构。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1001; #1060]
  - [x] provider secrets 仍只保留在后端配置和调用层，没有进入 query/detail 响应。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#525]

- [x] 强化 provider 层错误映射与测试 (AC: 2)
  - [x] provider 超时、响应结构异常、非 2xx 状态会映射为稳定的 `timeout` / `upstreamFailed` / `contractInvalid` 等共享错误语义。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#313]
  - [x] 新增 provider adapter 与 Knowledge query 的 service/controller 测试，覆盖：成功 query、超时、HTTP 失败、无效 provider config、citation 归一化结果。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]

## Dev Notes

### Previous Story Insights

- `4.2` 已冻结 Knowledge 为 retrieval-only 契约，`4.3` 已把 `/knowledge-bases/:id/query` 接成最小闭环，因此 `4.4` 的重点是把 mock/内建结果替换成真正的 provider adapter，而不是再改 query 路由协议。[Source: _bmad-output/implementation-artifacts/ap-4-2-freeze-knowledge-mvp-contract-to-standardized-retrieval.md; ap-4-3-expose-knowledge-retrieval-via-shared-open-capability-layer.md]

### Scope Boundaries

- `4.4` 不实现：
  - 多 provider 管理后台矩阵
  - provider 自动发现/证书体系
  - 非 HTTP transport
- `4.4` 要实现：
  - 首个 `http_retrieval` provider adapter
  - provider-neutral 调用边界
  - 错误映射与 citation 归一化

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计新增文件：
  - `service/agentplatform/knowledge_provider.go`
  - `service/agentplatform/http_retrieval_provider.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `service/agentplatform/knowledge_query.go`
  - `controller/agentplatform/open_capabilities_test.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.4: 接入首个 http_retrieval provider 适配器]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#889-893]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#996-1001]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#277]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `4.4` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 provider adapter 边界上继续漂移。
- 已加载 Story AP-4.4 验收标准、Knowledge query MVP 和架构中的 `KnowledgeProvider` 契约建议，准备抽出 provider-neutral adapter 并接入首个 `http_retrieval` 实现。

### Completion Notes List

- 已新增 `KnowledgeProvider` 接口与首个 `HTTPRetrievalProvider` 实现，把 Knowledge query 从 mock/内建逻辑切换到 provider-neutral adapter 边界。
- 已将 `KnowledgeQueryService` 接入 provider adapter，并复用 retrieval-only 契约、授权、发布状态和统一 error envelope。
- 已把 provider 返回结果归一化为统一 `items` / `citations` 结构，同时保证 provider secrets 不进入公共响应。
- 已补 service/controller 定向测试，覆盖成功 query、provider 超时、HTTP 失败与错误映射。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-4-4-integrate-first-http-retrieval-provider-adapter.md`
- `service/agentplatform/knowledge_provider.go`
- `service/agentplatform/http_retrieval_provider.go`
- `service/agentplatform/knowledge_query.go`
- `service/agentplatform/knowledge_query_test.go`

## Change Log

- 2026-06-01: 完成首个 `http_retrieval` provider adapter、provider-neutral query 边界与定向测试，并将故事推进为 done。
