# Story 3.2: 定义 Skill 契约 detail、schema 与调用准备度

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 每个 `Skill` 都具备明确的 schema、invoke mode 与版本化契约 detail,
so that 客户端无需猜测请求和返回结构就能准备调用。

## Acceptance Criteria

1. **Given** 某个 `Skill` detail 被创建  
   **When** detail 被保存  
   **Then** 平台保存输入 schema、输出 schema、timeout/binding metadata 与版本化契约 detail  
   **And** 这些字段存放在 typed `Skill` detail 结构中，而不是消费者私有文档里。
2. **Given** 某个 `Skill` 契约不完整或不合法  
   **When** 发布者尝试发布该 `Skill`  
   **Then** 平台阻止发布并返回 contract validation error  
   **And** 失败原因可从控制面反馈和审计记录中定位。
3. **Given** 某客户端读取已发布的 `Skill` detail  
   **When** detail 响应返回  
   **Then** 契约信息足以支持调用前准备  
   **And** 可选扩展字段不能替代核心 invoke 契约。

## Tasks / Subtasks

- [x] 强化 Skill typed detail 的结构校验 (AC: 1, 2, 3)
  - [x] 在 `model/agentplatform/skill_def.go` 中补充最小业务约束：`invoke_schema_json`、`output_schema_json`、`invoke_mode`、`timeout_seconds`、`binding_config_json` 不再只是“可空文本”；MVP 现在会阻止明显缺失或非法的 Skill 契约入库。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#278]
  - [x] 继续使用 `common.*` JSON wrapper 做业务 JSON 校验，不在业务逻辑里直接使用 `encoding/json`。[Source: AGENTS.md#Rule 1]
  - [x] 保持 Skill detail 仍作为 typed table 存储，而不是退回 consumers 私有文档或松散 comment 字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Typed detail shape]

- [x] 在 Skill resource version 创建与发布前增加 invoke readiness 校验 (AC: 1, 2)
  - [x] `service/agentplatform/resource_version.go` 在创建 Skill 版本时明确校验：输入/输出 schema 存在、invoke mode 合法、timeout 为正值、binding config 具备最小可调用形态。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.2]
  - [x] 当 Skill 契约不完整时，平台阻止其被视为可发布或可调用候选，并返回 `contract validation error`，而不是让问题拖到 invoke 时才暴露。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-8]
  - [x] 本故事聚焦“准备度”与契约有效性，没有提前实现真实 invoke 执行；真实 invoke 继续留给 3.3。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.3]

- [x] 提升 Skill control-plane 读取结果对调用准备的表达能力 (AC: 1, 3)
  - [x] Skill detail 查询结果继续稳定返回 invoke schema、output schema、invoke mode、timeout 与 binding metadata，足以支持客户端“调用前准备”。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.2]
  - [x] 当前未额外引入 Skill 专用 readiness 字段；现有控制面 detail + 严格 validation 已能表达 invoke readiness，避免在 3.2 提前扩散过多专用状态字段。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-8]

- [x] 补定向测试，锁住 contract validation 行为 (AC: 1, 2, 3)
  - [x] 更新 `service/agentplatform/resource_version_test.go`、`controller/agentplatform/resource_version_test.go`，覆盖：完整 Skill detail 可保存、缺失 schema/非法 timeout/非法 invoke mode 被拒绝。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Testing organization]
  - [x] 本故事未引入高信号 control-plane 文档变更，暂不更新 OpenAPI；优先保证测试和行为一致。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Docs organization]

## Dev Notes

### Previous Story Insights

- `3.1` 已经提供 Skill 专用 control-plane 管理入口，因此 `3.2` 不需要再扩写列表/创建主流程，而应聚焦契约 detail 和 readiness 校验。[Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md]
- 通用 resource version 与 typed detail 基线已存在，Skill detail 的自然落点就是 `skill_def` + `resource_version`，不是新增另一层 Skill contract 表。[Source: service/agentplatform/resource_version.go; model/agentplatform/skill_def.go]

### Scope Boundaries

- `3.2` 不实现：
  - Skill 真实 invoke 执行
  - open capability invoke 路由的业务逻辑
  - Knowledge / Agent 合同深化
- `3.2` 要实现：
  - Skill detail 的明确校验
  - 发布/调用前 readiness 阻断
  - 充足的 control-plane 读取信息

### Testing

- 推荐至少运行：
  - `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform`

### Project Structure Notes

- 预计修改文件：
  - `model/agentplatform/skill_def.go`
  - `service/agentplatform/resource_version.go`
  - `service/agentplatform/resource_version_test.go`
  - `controller/agentplatform/resource_version_test.go`
  - 可能涉及 `dto/agentplatform/resource_version.go`

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 3.2: 定义 Skill 契约 detail、schema 与调用准备度]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-8]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#278]
- [Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 继续作为总编排状态源，但 `3.2` 采用 manual create/dev takeover，避免 create-story / dev-story 子进程在 Skill contract 语义上继续漂移。
- 已加载 Story 3.2 验收标准、Skill typed detail 模型和 resource version 实现，准备补 Skill 契约校验与 invoke readiness。

### Completion Notes List

- 已强化 `skill_def` 的最小业务校验，要求 invoke schema、output schema、invoke mode、timeout、binding config 全部满足最小可调用条件。
- 已将 Skill contract validation 接入 `resource_version` 创建路径，缺失 schema、非法 invoke mode、非法 timeout 等情况会被阻断为 `ErrSkillContractInvalid`。
- 已补 service/controller 定向测试，锁住非法 Skill 契约被拒绝的行为。
- 已通过 `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-3-2-define-skill-detail-schema-and-invoke-readiness.md`
- `model/agentplatform/skill_def.go`
- `service/agentplatform/resource_version.go`
- `service/agentplatform/resource_version_test.go`
- `controller/agentplatform/resource_version.go`
- `controller/agentplatform/resource_version_test.go`

## Change Log

- 2026-06-01: 完成 Skill 契约校验、invoke readiness 阻断与定向测试，并将故事推进为 done。
