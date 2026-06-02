# Story 1.3: 实现统一生命周期、版本与可回滚治理动作

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 资源发布者,
I want 资源与资源版本按照统一生命周期流转，并通过显式治理动作改变状态,
so that publish、disable、revoke、offline、rollback 的行为是一致且可解释的。

## Acceptance Criteria

1. **Given** 某资源已存在于 registry 中  
   **When** 发布者发起治理动作改变状态  
   **Then** 平台强制使用统一生命周期词汇 `draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`  
   **And** 状态变化由显式动作触发，而不是由隐式数据写入推导。
2. **Given** 某资源历史版本已经发布过  
   **When** 有权限的操作者执行 rollback  
   **Then** 平台恢复到已知历史版本  
   **And** rollback 不依赖手工修改数据库。
3. **Given** 某次生命周期变更成功或失败  
   **When** 记录审计事件  
   **Then** 审计事件至少包含操作者、对象、动作、前后状态、请求标识与结果  
   **And** 该事件链后续可用于 publish/revoke/rollback 诊断。

## Tasks / Subtasks

- [x] 在现有 registry + resource_version + typed detail 基线上补统一生命周期治理模型 (AC: 1, 2, 3)
  - [x] 为资源与资源版本明确统一状态词汇和可执行治理动作：`draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`，并把这些词汇的校验下沉到 Agent Platform 模型/服务层，而不是散落在 controller 条件分支里。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2; _bmad-output/planning-artifacts/architecture-agent-platform.md#615-619]
  - [x] 在已有 `model/agentplatform/resource.go` 和 `resource_version.go` 基础上补充 lifecycle/update 约束，确保资源状态变化通过显式治理动作驱动，而不是靠“直接写任意状态字符串”达成。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3; _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md]
  - [x] 若需要新增治理事件或任务模型，优先使用 `model/agentplatform/admin_action.go`、`publish_task.go`（或等价命名）承载治理动作和诊断链条，不要混入 `enterprise_admin_actions`、`relay/**` 或通用日志表。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#audit--tasks; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]

- [x] 落地最小 lifecycle 服务与控制面 API，证明状态变化是显式动作而非隐式写库 (AC: 1, 3)
  - [x] 扩展 `service/agentplatform`，提供至少这些显式治理入口：publish、disable、revoke、offline、rollback（命名可等价），并统一检查当前状态是否合法、目标版本是否存在、以及状态迁移是否被允许。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2; _bmad-output/planning-artifacts/architecture-agent-platform.md#versioning]
  - [x] 为控制面增加最小管理路由，例如 `/api/agent-platform/resources/:id/publish`、`/disable`、`/revoke`、`/offline`、`/versions/:version/rollback`（或等价 REST 设计），并继续沿用 `/api/agent-platform/**` + `AdminAuth()` 的管理面权限模型。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Route naming; router/agentplatform-router.go]
  - [x] 当前故事不要求开放能力消费面感知这些状态变化，不要提前去实现 `/api/open-capabilities/**` 的 freshness/revoke 收敛逻辑；这里只需要把管理面动作和底层状态持久化链打通。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#592-595; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4]

- [x] 实现 rollback 基线，确保回退以历史版本为锚点而非手工改库 (AC: 2, 3)
  - [x] rollback 必须基于已存在的 `agent_platform_resource_versions` 历史记录执行，至少能把资源的“当前/latest_version + 生命周期状态”恢复到指定历史版本，而不是依赖手工 UPDATE 某几列。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#60; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-2]
  - [x] rollback 不需要在 1.3 引入 exposure/client 维度回滚，但必须保证后续 1.4/1.5 可以基于这条历史链扩展，而不会推翻当前的版本治理实现。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.5]
  - [x] 若 rollback 需要写任务或审计记录，确保记录包含：操作者、对象、动作、前状态、后状态、目标版本、结果和请求标识。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-14]

- [x] 建立治理动作审计与诊断事实源 (AC: 3)
  - [x] 为 Agent Platform 增加管理动作审计模型/服务/DTO（如 `admin_action.go`），最少记录：actor、object、action、before/after status、resource_version、request_id、result、error summary；确保这些记录属于 Agent Platform bounded context，自有表名前缀使用 `agent_platform_`。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#272-274; _bmad-output/planning-artifacts/architecture-agent-platform.md#823]
  - [x] 若治理动作具有异步或可重试特征，可引入最小 `publish_task` 记录，但 1.3 不需要完整 scheduler/worker 编排；关键是动作结果可追踪、失败可诊断。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#272-274; _bmad-output/planning-artifacts/architecture-agent-platform.md#1107]
  - [x] 审计事实源不能退回使用通用 log 或 enterprise 审计表；Agent Platform 的治理链要保持独立，方便后续 5.x 之外的 consumers 做诊断。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#Audit + diagnostics]

- [x] 锁定 1.3 的边界，不提前做 1.4/1.5/2.x 的能力 (AC: 1, 2, 3)
  - [x] 本故事不实现 projection/exposure、client visibility/callable、OAuth 授权、refresh token、open-capabilities freshness 收敛，也不做 `web/default` 控制面页面。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4; _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.5; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md#6-暂不进入-MVP-的内容]
  - [x] 本故事也不要求 provider offline 检测任务、client revoke 协调或 token version 缓存实现；这里只需要把统一生命周期语义和 rollback/action 审计基线建好。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#125-126; _bmad-output/planning-artifacts/architecture-agent-platform.md#744-793]

- [x] 补齐模型、服务、控制器测试与合同，证明 lifecycle/rollback/action audit 真正可用 (AC: 1, 2, 3)
  - [x] 新增 `model/agentplatform/*_test.go`，覆盖：合法/非法状态词、resource/resource_version 状态迁移、rollback 对历史版本的恢复、以及审计记录字段完整性。[Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md; _bmad-output/implementation-artifacts/ap-1-2-implement-typed-detail-storage-for-skill-knowledge-agent.md]
  - [x] 新增 `service/agentplatform/*_test.go`，覆盖：publish/disable/revoke/offline/rollback 的状态机校验、错误输入、无历史版本 rollback 拒绝、以及治理动作写审计链。[Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-2]
  - [x] 新增 `controller/agentplatform/*_test.go`，覆盖：治理 API 成功路径、非法迁移拒绝、rollback 读取指定历史版本恢复、以及最小错误响应语义；若 OpenAPI 管理面新增治理动作接口，同步更新 `docs/openapi/api.json`。[Source: docs/openapi/api.json; _bmad-output/planning-artifacts/architecture-agent-platform.md#Requirements -> Components Mapping]

## Dev Notes

- Story 1.3 的重点不是“再多建几张表”，而是把 1.1/1.2 已有 registry + version + detail 基线变成一个 **显式治理动作驱动的生命周期系统**。核心要求是：状态不能被隐式写入随意篡改，rollback 不能靠手工 SQL 完成。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3; _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2]
- 1.3 仍然属于 control plane，后续 1.4/1.5 才会把这些状态带到 projection/exposure 语义上。所以这里要先把内部治理链做扎实，但不要提前把 client/exposure 逻辑掺进来。[Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.4; _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
- 现有 `agent_platform_resource_versions` 已经是历史版本事实源，1.3 的 rollback 一定要基于它做恢复，而不是重新发明“快照表”或让操作者手工回填字段。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#245-254]
- Agent Platform 的治理动作审计应该落在自己的 bounded context 表中，例如 `agent_platform_admin_actions` / `agent_platform_publish_tasks`，不要借 `enterprise_admin_actions` 或 generic log。这样后续诊断链条才能独立演进。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#272-274; _bmad-output/planning-artifacts/architecture-agent-platform.md#1107]
- 生命周期词汇的统一性是 1.3 的显式目标：`draft`、`published`、`disabled`、`revoked`、`offline`、`deprecated`。任何其它语义都不应悄悄冒出来成为内部状态分支。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#615-619]
- 即便 1.3 引入治理动作，也还不需要把 `provider_offline` 探测、client revoke convergence 或 refresh token/token_version 一并实现；这些属于更后面的 auth/open-capability/health 章节。[Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#125-126; _bmad-output/planning-artifacts/architecture-agent-platform.md#744-793]

### Project Structure Notes

- 预计新增文件：
  - `model/agentplatform/admin_action.go`
  - `model/agentplatform/publish_task.go`（若最小任务记录确有必要）
  - `service/agentplatform/lifecycle.go`
  - `controller/agentplatform/lifecycle.go`
  - `dto/agentplatform/lifecycle.go`
  - 对应 `*_test.go`
- 预计修改文件：
  - `model/agentplatform/resource.go`
  - `model/agentplatform/resource_version.go`
  - `model/agentplatform/migration.go`
  - `router/agentplatform-router.go`
  - `docs/openapi/api.json`
- 保持不改或只读依赖：
  - `relay/**`
  - `docs/openapi/relay.json`
  - `web/default/**`
  - `oauth/**`
  - `middleware/agentplatform_*`（除非 1.3 为管理面校验显式需要）

### References

- [Source: _bmad-output/planning-artifacts/epics-agent-platform.md#Story 1.3: 实现统一生命周期、版本与可回滚治理动作]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-2]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#UJ-2]
- [Source: _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md#FR-14]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#60]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#125-126]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#245-254]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#272-274]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#319-339]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#615-619]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md#1107]
- [Source: _bmad-output/implementation-artifacts/ap-1-1-establish-shared-resource-registry-and-stable-identity.md]
- [Source: _bmad-output/implementation-artifacts/ap-1-2-implement-typed-detail-storage-for-skill-knowledge-agent.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- Story Automator 对 Agent Platform 的 `1.3 -> ap-1-3-*` key 映射沿用已修正逻辑；本 story 文件由 manual create-story takeover 补齐，用于避免 create-story 会话停留在 1.2 typed-detail 代码上下文中空转。
- 已加载 `epics-agent-platform.md`、`architecture-agent-platform.md`、`prd-agent-platform-2026-05-31/prd.md`，并参考已完成的 `ap-1-1`、`ap-1-2` story 产物作为当前实现前序上下文。
- 当前仓库在 `1.2` 后已经具备 registry、resource_version 和 typed detail 基线，因此 `1.3` 应把重点放在 lifecycle action、rollback 和审计链，而不是重建前两步的数据模型。

### Completion Notes List

- 已在 `1.1`/`1.2` 基线之上补齐统一生命周期动作：publish、disable、revoke、offline、rollback。
- 已新增 Agent Platform 自有 `admin_action` 审计事实源，用于记录治理动作、前后状态、目标版本和请求标识。
- rollback 现在基于 `agent_platform_resource_versions` 历史记录完成，不依赖手工修改数据库。
- 已完成 lifecycle 相关 model/service/controller 定向测试，并通过 `GOCACHE=/private/tmp/go-build-cache go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform` 验证。

### File List

- `_bmad-output/implementation-artifacts/ap-1-3-implement-unified-lifecycle-version-and-rollback-actions.md`
- `model/agentplatform/admin_action.go`
- `service/agentplatform/lifecycle.go`
- `service/agentplatform/lifecycle_test.go`
- `controller/agentplatform/lifecycle.go`
- `controller/agentplatform/lifecycle_test.go`
- `model/agentplatform/migration.go`
- `router/agentplatform-router.go`


## Change Log

- 2026-06-01: 完成生命周期动作、rollback 基线、治理动作审计与定向测试，并将故事推进为 done。
