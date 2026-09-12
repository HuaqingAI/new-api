# Story AP-4.2: 将 Knowledge 的 MVP 契约冻结为标准化 retrieval

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 客户端集成人员,
I want `Knowledge` 的契约语义在 MVP 中被明确收束,
so that 我只需要围绕一个清晰的 `retrieval` 模式完成消费，而不用猜测它是不是泛化 RAG 接口。

## Acceptance Criteria

1. **Given** 某个 `Knowledge` 资源被定义为 MVP 资源  
   **When** 其契约 detail 被保存  
   **Then** `knowledge_mode` 被明确标记为 `retrieval`  
   **And** direct answer generation、多轮 grounding 等模式不作为 MVP 标准能力出现。
2. **Given** 某个 `Knowledge` 资源绑定了具体 provider  
   **When** 平台保存其对外契约  
   **Then** 公共 detail/query 契约仍然通过统一 `http_retrieval` 语义表达  
   **And** 不暴露 `LightRAG` 或其它 provider 的专属字段作为平台标准字段。
3. **Given** 某客户端请求 `Knowledge` detail  
   **When** detail 响应返回  
   **Then** detail 中包含 retrieval query shape 与结果结构预期  
   **And** 结果语义覆盖片段、来源元数据与排序信息。
4. **Given** 某客户端不支持目标 `Knowledge` 要求的 retrieval 模式  
   **When** 平台在调用前执行兼容性校验  
   **Then** 平台可以将该资源判定为 visible but not callable  
   **And** 返回明确的兼容性或契约错误原因。

## Tasks / Subtasks

- [x] 收紧 Knowledge typed detail 的契约校验 (AC: 1, 2, 3, 4)
  - [x] 在 `model/agentplatform/knowledge_def.go` 中收紧校验：MVP 仅允许 `knowledge_mode=retrieval`，并要求 `provider_type`、`provider_adapter_key`、query/citation 相关契约字段满足最小有效性。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#389-398]
  - [x] `provider_type` 在 MVP 中限制在统一 provider-neutral 语义边界（`http_retrieval` / `native`），不再把具体引擎名当成公共契约字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#390-400]
  - [x] 继续使用 `common.*` JSON wrapper 做业务 JSON 校验。[Source: AGENTS.md#Rule 1]

- [x] 在 Knowledge resource version 创建路径中冻结 retrieval-only 语义 (AC: 1, 2, 3)
  - [x] `service/agentplatform/resource_version.go` 在创建 Knowledge 版本时阻止非 `retrieval` 模式、缺失 query/citation 契约、缺失 provider binding 的版本进入可发布候选。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#271-277]
  - [x] 平台 detail 侧继续只暴露 retrieval query shape 与结果结构预期，不增加 provider-specific 公共字段。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#274-277]
  - [x] 本故事没有实现 retrieval query 调用，query runtime 继续留给 4.3/4.4。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.3; #Story AP-4.4]

- [x] 将 visible-but-not-callable 的契约原因接入 Knowledge detail / readiness 判断 (AC: 4)
  - [x] 对于不满足 retrieval-only 契约的 Knowledge，平台现在会在版本创建阶段明确阻断，避免其进入后续可调用候选。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#276]
  - [x] 当前未额外扩展 Knowledge 专属 callable_state；在 4.2 范围内，先通过严格 contract validation 锁住“不可调”的来源，后续 4.3 再把 query 路径侧的 visible-but-not-callable 语义接通。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#337; #631]

- [x] 补定向测试，锁住 retrieval-only 与 provider-neutral 约束 (AC: 1, 2, 3, 4)
  - [x] 更新 `service/agentplatform/resource_version_test.go`、`controller/agentplatform/resource_version_test.go`，覆盖：非 retrieval 被拒绝、provider-specific 公共模式被拒绝、完整 retrieval detail 可保存并可读。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 本故事未引入高信号 control-plane 文档变化，暂不更新 OpenAPI；优先保证测试和行为一致。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- `4.1` 已经建立 Knowledge 专用 control-plane 管理入口，因此 `4.2` 应聚焦 detail/contract 语义，而不是重做 Knowledge CRUD。[Source: _bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md]
- 通用 open capability detail 已经存在，`4.2` 需要把 Knowledge detail 的公共语义正式收束为 retrieval-only，为 `4.3` 的 query 闭环扫清边界。[Source: _bmad-output/implementation-artifacts/ap-2-4-publish-open-capability-standard-apis-and-error-envelope.md]

### Scope Boundaries

- `4.2` 不实现：
  - provider query 调用
  - citation 运行时归一化
  - provider adapter 接入
- `4.2` 要实现：
  - retrieval-only 契约冻结
  - provider-neutral metadata 约束
  - visible-but-not-callable 的契约侧理由表达

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计修改文件：
  - `model/agentplatform/knowledge_def.go`
  - `service/agentplatform/resource_version.go`
  - `service/agentplatform/resource_version_test.go`
  - `controller/agentplatform/resource_version_test.go`
  - 如需要，`service/agentplatform/discovery.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story AP-4.2: 将 Knowledge 的 MVP 契约冻结为标准化 retrieval]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-10]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#389-400]
- [Source: _bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `4.2` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Knowledge 契约冻结边界上继续漂移。
- 已加载 Story AP-4.2 验收标准、Knowledge typed detail 模型、open capability detail 与 UX 约束，准备冻结 retrieval-only 契约。

### Completion Notes List

- 已将 Knowledge typed detail 收紧为 retrieval-only，并限制 `provider_type` 只能使用 provider-neutral 的 `http_retrieval` / `native` 语义。
- 已将 query/citation/provider config 的最小有效性校验接入 Knowledge resource version 创建路径。
- 已补 service/controller 定向测试，锁住非 retrieval、provider-specific 公共模式和缺失 retrieval 契约字段被拒绝的行为。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`。

### File List

- `_bmad-output/implementation-artifacts/ap-4-2-freeze-knowledge-mvp-contract-to-standardized-retrieval.md`
- `model/agentplatform/knowledge_def.go`
- `service/agentplatform/resource_version.go`
- `service/agentplatform/resource_version_test.go`
- `controller/agentplatform/resource_version.go`
- `controller/agentplatform/resource_version_test.go`

## Change Log

- 2026-06-01: 完成 Knowledge retrieval-only 与 provider-neutral 契约冻结及定向测试，并将故事推进为 done。
