---
status: approved
project: new-api
created: 2026-05-31
reprocessed: 2026-06-02
approved: 2026-06-02
workflow: bmad-correct-course
mode: incremental
recommended_scope: moderate
recommended_path: Direct Adjustment + MVP Exit Gate Review
section_status:
  issue_summary: approved
  impact_analysis: approved
  artifact_conflict_analysis: approved
  path_forward_evaluation: approved
  detailed_change_proposals: approved
  implementation_handoff: approved
  final_review: approved
old_conclusions: ignored
trigger_source: user request on 2026-06-02 to reprocess the 2026-05-31 Agent Platform change proposal from Issue Summary, preserving the generic Agent Platform goal while ignoring prior conclusions
artifacts_now_present:
  - _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md
  - _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md
  - _bmad-output/planning-artifacts/epics-agent-platform.md
  - _bmad-output/planning-artifacts/architecture-agent-platform.md
  - _bmad-output/planning-artifacts/ux-agent-platform.md
artifacts_modified:
  - _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md
  - _bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md
  - _bmad-output/planning-artifacts/epics-agent-platform.md
  - _bmad-output/planning-artifacts/architecture-agent-platform.md
  - _bmad-output/planning-artifacts/ux-agent-platform.md
  - _bmad-output/implementation-artifacts/sprint-status.yaml
  - docs/agent-platform-downstream-contract-spec.md
---

# Sprint Change Proposal: 通用 Agent 能力平台化重规划

> 状态说明：
> 本文档创建于 2026-05-31，并于 2026-06-02 按 Correct Course Incremental 模式重新处理。
> 本轮重处理忽略旧的 Impact Analysis、Recommended Approach、Detailed Change Proposals 和 Final Recommendation。
> 截至 2026-06-02，本提案的 Issue Summary、Impact Analysis、Artifact Conflict Analysis、Path Forward、Detailed Change Proposals、Implementation Handoff 与 Final Review 均已增量批准。

## 1. Issue Summary

### Trigger

本轮 Correct Course 由对 `_bmad-output/planning-artifacts/sprint-change-proposal-2026-05-31-agent-platform.md` 的重新处理触发。目标不变：`new-api` 需要建设通用 Agent 能力平台，而不是 Cherry Studio 单消费者接口适配。

但当前事实已发生变化：Agent Platform 独立 PRD、Epics、Architecture、轻量 UX 文档均已落地，且 AP-2.x 后端接入底座已经实现。现在需要纠偏的问题不再是“是否要新建 Agent Platform 规划”，而是：

> 已有 Agent Platform 后端能力尚未完全整理、冻结和产品化为下游可签核的公共接入规格；同时企业模型发现、完整错误/状态矩阵、mock/fixture 和运营化 onboarding 仍存在明确缺口。

### Core Problem

问题类型：**实施后规划收敛 / 公共契约冻结缺口**。

当前风险不是平台方向错误，而是平台已经从概念规划推进到部分实现后，规划文档、实现状态和下游签核需求之间出现了新的不一致：

- PRD / Architecture 已定义通用资源控制面、开放能力层、OAuth、Skill、Knowledge、Agent、freshness 和错误 envelope。
- 代码与实现产物显示 AP-2.x 已具备 OAuth、client registration、open capability discovery/detail/refresh、skill invoke、knowledge query、agent detail 等后端基础。
- 但下游仍缺一套可直接签核的正式公共规格包，尤其是 OAuth wire contract、client registration schema、discovery/detail/refresh 字段集、Skill/Knowledge request/response、错误码矩阵和状态矩阵。
- 下游需求中的企业模型发现仍未作为 Agent Platform 公共契约冻结，可能成为 Cherry Studio P0 接入的实际阻塞。
- 管理 UI / Clients 工作区 / 授权运营流程尚未完整产品化，可能导致“接口可用但运营不可交付”。

### Evidence

1. `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md`：已冻结 Agent Platform 的通用平台目标、retrieval-only Knowledge、Agent definition/template 边界、Codex 第二验证消费者。
2. `_bmad-output/planning-artifacts/architecture-agent-platform.md`：已冻结 control plane / auth plane / open capability plane、`/api/open-capabilities/**`、OAuth + PKCE、provider-neutral `http_retrieval`。
3. `_bmad-output/planning-artifacts/epics-agent-platform.md`：已有 AP-1 到 AP-5 的 story 拆分。
4. `docs/agent-platform-downstream-gap-matrix.md`：明确指出最大 gap 是模型发现公共契约未冻结、后端基础未沉淀成签核级文档、错误/状态矩阵和 mock fixture 缺失。
5. `docs/agent-platform-downstream-contract-spec.md`：说明当前后端底座已超过概念 API，但尚未 fully frozen / operator-friendly。
6. 当前代码中已存在 `controller/agentplatform/`、`service/agentplatform/`、`dto/agentplatform/`、`middleware/agent_platform_bearer.go`、`router/open_capabilities_router.go` 等实现路径。

### Reframed Issue

因此，本轮变更应从“新建通用 Agent Platform 产品线”改为：

> 将已落地的 Agent Platform 后端能力与下游 P0 接入需求重新对齐，冻结正式公共契约，补齐模型发现、错误/状态矩阵、mock fixture 与运营化 onboarding 缺口，确保 Cherry Studio 可以作为首个消费者签核接入，同时不破坏 Codex / cc switch 等后续消费者的通用平台目标。

## 2. Impact Analysis

### 2.1 Triggering Story Assessment

触发点不是单个 AP story 的验收失败，而是 **AP-1 到 AP-5 已基本完成后，发现“实现底座”与“下游可签核接入规格”之间仍有缺口**。

根据 `_bmad-output/implementation-artifacts/sprint-status.yaml`，截至 2026-06-01：

- Agent Platform Epic AP-1 到 AP-5 均为 `done`
- AP-2 OAuth / client / open capability / freshness 已完成
- AP-3 Skill、AP-4 Knowledge、AP-5 Agent 也已完成 MVP 闭环

因此本轮不应回滚 AP-1 到 AP-5，也不应把它们重新定义为失败；更准确的判断是：

> 现有 AP epics 完成了平台后端与最小管理面基线，但没有完全覆盖“下游 P0 签核级公共契约、模型发现、状态矩阵、mock fixture、运营化 onboarding”这组交付物。

### 2.2 Current Epic Impact

#### Agent Platform Epic AP-1: 资源治理基线

状态：不建议重开。

影响：低。

Epic 1 的 registry、typed detail、lifecycle、exposure、web/default shell 仍然有效。后续只需要复用其资源身份、版本、exposure 与 audit 基线来支撑公共规格和状态矩阵，不需要重做基础模型。

#### Agent Platform Epic AP-2: 客户端接入与开放能力标准层

状态：主要受影响。

影响：高。

Epic 2 已完成 OAuth、client registration、token、open capability、freshness / contract convergence，但当前缺的是“对下游可签核”的冻结口径：

- OAuth wire contract / callback / allowlist / consent UX
- client registration schema 示例与运营流程
- discovery/detail/refresh 字段集
- freshness / revoke / stale / retry 的正式状态矩阵
- 错误码与客户端态映射
- mock / fixture / conformance tests

Epic 2 不必推翻，但需要新增 follow-up story 或新 epic 承接“公共契约冻结”。

#### Agent Platform Epic AP-3: Skill 库管理与调用

状态：保留完成态。

影响：中。

Skill 管理与 invoke 已完成，但下游仍缺正式 metadata + invoke request/response spec。若 Cherry Studio P0 需要 async skill、任务状态或取消接口，需要明确 P0 是否只支持 sync；否则容易在接入期扩大范围。

#### Agent Platform Epic AP-4: Knowledge 库管理与检索契约

状态：保留完成态。

影响：中。

Knowledge retrieval-only 和 provider-neutral `http_retrieval` 边界有效。后续需要把 query request/response、citations/items、provider failure、metadata 字段冻结成公共规格。当前不建议把 Knowledge 扩成完整 RAG / ingestion / chat grounding。

#### Agent Platform Epic AP-5: Agent 定义库与依赖边界

状态：保留完成态。

影响：低到中。

Agent definition/template 边界仍然正确。后续只需要在公共规格里强调 Agent 是 discovery/detail/reference-boundary 对象，不是服务端 runtime。

### 2.3 Enterprise Governance Epic Impact

企业治理 Epic 1-6 仍然是基础设施底座，不需要因本轮 CC 回滚或重写。

Epic 7 当前为 `backlog`，与本轮问题没有直接阻塞关系。但下游需求中出现的 `accountId`、`tenantId`、`enterpriseName`、企业账号展示 DTO 等字段，可能需要与企业治理身份/组织模型建立轻量映射。这个影响应作为接口规格问题处理，不应把 Agent Platform 下游接入强行并入 Epic 7。

### 2.4 New Epic-Level Change Needed

建议新增一个 follow-up epic，而不是重开 AP-1 到 AP-5：

**AP-6: 下游公共契约冻结与接入签核**

建议覆盖：

1. 冻结 OAuth / token / revoke / callback / allowlist wire contract
2. 冻结 client registration schema 与 onboarding 运营流程
3. 冻结 discovery / detail / refresh 公共字段集
4. 冻结 Skill invoke 与 Knowledge query request/response spec
5. 补齐 enterprise model discovery / default model / model status 公共契约
6. 冻结统一错误码矩阵与客户端状态矩阵
7. 提供 mock server / fixture / contract conformance 测试包
8. 明确 Cherry Studio first、Codex second 的签核路径

### 2.5 Epic Order / Priority Impact

优先级应调整为：

1. 先做 AP-6 公共契约冻结，避免下游接入继续靠实现猜测。
2. 再根据 AP-6 结论决定是否需要补代码 story。
3. 最后推进更完整 Clients UI / onboarding 运营工作区。

这能保护 AP-1 到 AP-5 的完成态，同时把真正缺失的“签核层”补上。

## 3. Artifact Conflict And Impact Analysis

### 3.1 PRD Impact

#### Agent Platform PRD

需要小幅修订 / addendum 补充。

当前 PRD 的平台方向仍然正确，不需要重写。但它对“AP-6 / 标准能力层”的表达存在新旧语境混杂：

- PRD 和 addendum 已提到 AP-6 是标准能力面，不是 Cherry Studio 专属适配。
- 但当前 `epics-agent-platform.md` 和 `sprint-status.yaml` 中只有 AP-1 到 AP-5，且均已 done。
- 新发现的缺口是“下游公共契约冻结与接入签核”，并不等同于旧语境里的“是否要建设 AP-6 标准层”。

建议：在 PRD addendum 中新增 2026-06-02 修订说明，把 AP-6 重新定义为 follow-up：

> AP-6 不再表示“新建标准层本身”，而是表示“在 AP-1 到 AP-5 已完成后，将已有能力冻结成下游可签核公共契约，并补齐模型发现、错误/状态矩阵、mock fixture 与 onboarding”。

#### 企业治理 PRD

不建议直接修改主体。

企业治理 PRD 仍负责组织、身份、预算、用量、风险治理。只需要在接口规格中引用企业账号展示字段，例如 `accountId`、`tenantId`、`enterpriseName` 的来源语义；不应把 Agent Platform 下游接入并入 Epic 7。

### 3.2 Epics / Stories Impact

#### `epics-agent-platform.md`

需要新增 AP-6。

AP-1 到 AP-5 不应重开。新增 AP-6 更清晰：

- AP-6.1 冻结 OAuth / token / revoke / callback / allowlist wire contract
- AP-6.2 冻结 client registration schema 与 onboarding 流程
- AP-6.3 冻结 discovery / detail / refresh 公共字段集
- AP-6.4 冻结 Skill invoke 与 Knowledge query 公共 request/response
- AP-6.5 冻结 enterprise model discovery / default model / model status 契约
- AP-6.6 冻结错误码矩阵与客户端状态矩阵
- AP-6.7 提供 mock / fixture / contract conformance 套件
- AP-6.8 完成 Cherry Studio first / Codex second 签核路径

#### `sprint-status.yaml`

批准后需要新增 AP-6 条目。

当前 AP-1 到 AP-5 均为 `done`。批准 AP-6 后，应新增：

- `ap-epic-6: backlog`
- `ap-6-*` story keys: `backlog`

不改 AP-1 到 AP-5 状态。

### 3.3 Architecture Impact

`architecture-agent-platform.md` 需要架构 amendment，不需要重写。

现有架构仍然有效：control plane / auth plane / open capability plane、provider-neutral Knowledge、Agent definition-only 都应该保留。

需要补的架构点是：

- enterprise model discovery 应归属哪个 surface：`/api/open-capabilities/**`、`/api/agent-platform/**`，还是单独扩展现有 `/api/models` 的下游投影。
- 下游公共契约包的 source of truth：OpenAPI、docs spec、fixture，还是三者组合。
- model discovery 的 stable ID、default model、disabled reason、provider stable ID、account/tenant 字段来源。
- 错误码矩阵如何映射到当前 open capability envelope。
- mock / fixture / conformance test 的目录、生成方式和发布方式。
- Clients onboarding 与 consent visibility 是否作为 AP-6 产品化范围，还是后续 AP-7。

### 3.4 UX / Frontend Impact

`ux-agent-platform.md` 需要补充。

当前 UX 是轻量说明，已覆盖导航、发布/撤销、visible/callable、审计 drill-down。但 readiness report 已指出 consent visibility 偏弱；实际前端也只有 `AgentPlatformShell`，不是完整 Clients 工作区。

需要补：

- Clients 工作区：创建、编辑、能力声明、redirect URI、scopes、contract version、状态校验
- OAuth onboarding：callback / allowlist / consent 可见性 / revoke 操作
- 下游签核视图：契约版本、字段矩阵、状态矩阵、mock fixture 下载或查看
- 错误态与客户端态：`loginExpired`、`empty`、`loadFailed`、`networkFailed`、`noAssignedResource` 等如何展示
- model discovery 默认模型 / disabled reason / unavailable 状态的 UI 解释

### 3.5 OpenAPI / Docs Impact

#### `docs/openapi/api.json`

需要新增或扩展 AP-6 相关契约。

当前 OpenAPI 有传统 `/api/models`、`/api/user/models`、`/v1/models`、channel model 管理等接口，但没有面向 Agent Platform 下游的 enterprise model discovery 公共契约。不能简单把现有管理/relay 模型接口等同为下游 P0 model discovery。

#### `docs/agent-platform-downstream-contract-spec.md`

需要升级为正式签核规格。

当前它是 v0.1 草案，已经是很好的基线，但需要从“当前状态说明”升级为“正式公共契约”：

- 明确哪些字段是 MUST / SHOULD / MAY
- 明确版本号和 breaking change 规则
- 明确示例 payload
- 明确错误码和状态矩阵
- 明确 mock / fixture 使用方式

#### `docs/agent-platform-downstream-gap-matrix.md`

可保留为分析依据。

它不应成为最终签核物，但可以作为 AP-6 story 的输入材料。

### 3.6 Secondary Artifacts

需要新增或更新：

- AP-6 PRD addendum section
- `epics-agent-platform.md`
- `sprint-status.yaml`
- `architecture-agent-platform.md` amendment
- `ux-agent-platform.md` addendum
- `docs/agent-platform-downstream-contract-spec.md`
- `docs/openapi/api.json`
- mock / fixture / conformance test artifacts

不建议修改：

- `relay/**`
- `/v1/**`
- `docs/openapi/relay.json`
- AP-1 到 AP-5 已完成 story 状态
- 企业治理 PRD 主体

## 4. Path Forward Evaluation

### 4.1 Option 1: Direct Adjustment

结论：Viable，但不是“改几个已有 story”那么小。

适用方式：

- 不重开 AP-1 到 AP-5
- 新增 AP-6 follow-up epic
- 在 AP-6 内产出正式公共契约、model discovery、状态/错误矩阵、mock fixture
- 仅当 AP-6 明确发现代码缺口时，再追加最小实现 story

优点：

- 保护已完成的 AP-1 到 AP-5 成果
- 不扰动 relay、企业治理主线和已有开放能力接口
- 让缺口集中在“签核层 / 合同层 / onboarding 层”，边界清楚
- 能最快支撑 Cherry Studio P0 接入评审

风险：

- 如果 AP-6 只写文档、不补 model discovery 或 fixture，仍会停留在“说明性规格”
- 如果 AP-6 把完整 Clients UI 也一次性吃下，范围会膨胀

Effort: Medium

Risk: Medium

Timeline impact: 1 个 follow-up epic，优先做文档/契约，代码补丁按缺口追加

### 4.2 Option 2: Potential Rollback

结论：Not viable / 不推荐。

不建议回滚 AP-1 到 AP-5，因为：

- sprint-status 显示 AP-1 到 AP-5 均已完成
- 当前缺口不是“实现方向错误”，而是“实现后公共签核规格不足”
- 回滚会破坏已完成的 resource registry、exposure、OAuth、open capability、Skill、Knowledge、Agent 基线
- 下游仍然需要这些能力，回滚不能解决 model discovery / 状态矩阵 / mock fixture 缺口

Effort: High

Risk: High

Timeline impact: 负面，且不会解决核心问题

### 4.3 Option 3: PRD MVP Review

结论：Viable，建议作为 Direct Adjustment 的组成部分。

需要复审的不是 AP 平台 MVP 是否成立，而是 **AP-6 是否应作为 Cherry Studio / Codex 接入前的 MVP exit gate**。

建议新增 MVP exit gate：

- 不能只以 AP-1 到 AP-5 done 作为“可下游签核”标准
- 必须追加 AP-6 签核门槛：
  - enterprise model discovery 已冻结
  - OAuth / token / revoke wire contract 已冻结
  - discovery/detail/refresh 字段集已冻结
  - Skill / Knowledge 请求响应已冻结
  - 错误码矩阵和客户端状态矩阵已冻结
  - mock / fixture / conformance tests 可用
  - Cherry Studio first consumer 完成 contract sign-off
  - Codex second consumer 不需要新建平行协议主干

Effort: Low to Medium

Risk: Low

Timeline impact: 会延后“正式签核”，但减少接入返工

### 4.4 Recommended Path

推荐路径：**Direct Adjustment + MVP Exit Gate Review**

具体决策：

1. 不回滚 AP-1 到 AP-5。
2. 不把本轮变更升级为重新规划整条 Agent Platform 产品线。
3. 新增 AP-6：下游公共契约冻结与接入签核。
4. 把 AP-6 设为 Cherry Studio / Codex 正式接入前的 MVP exit gate。
5. AP-6 先产出契约和测试资产，再按发现的真实缺口补最小代码 story。
6. 完整 Clients UI / onboarding 工作区可以纳入 AP-6 的最小签核面；若范围过大，拆到后续 AP-7 产品化工作区。

### 4.5 Rationale

这个路径最适合当前状态，因为：

- 当前平台方向正确，已完成的底座不应推翻。
- 真正缺的是公共契约冻结和签核资产，不是新的资源治理基础设施。
- Model discovery 是明确缺口，必须进入 AP-6，而不能靠现有 `/api/models` 或 `/v1/models` 模糊替代。
- 错误/状态矩阵和 mock fixture 是下游接入质量门槛，应该比完整 UI 产品化更优先。
- Cherry Studio 可以继续作为 first consumer，但不能成为协议特例；Codex second consumer 用来验证通用性。

### 4.6 Scope Classification

**Moderate**

原因：

- 需要新增 epic、更新多份规划/规格/UX/OpenAPI/test artifacts。
- 但不需要推翻 PRD、重写架构、回滚已完成实现，也不要求立即大规模代码重构。
- 代码改动应由 AP-6 规格冻结后按缺口局部追加。

## 5. Detailed Change Proposals

### Proposal 1: 新增 AP-6 下游公共契约冻结与接入签核

Artifact: `_bmad-output/planning-artifacts/epics-agent-platform.md`

Status: approved and applied

Change:

- 在 Epic List 中新增 `Epic 6: 下游公共契约冻结与接入签核`
- 在文档尾部追加 AP-6 详细 story 拆分
- AP-1 到 AP-5 保持完成态，不重开

New stories:

1. `Story 6.1: 冻结 OAuth、Token、Revoke 与 Callback Wire Contract`
2. `Story 6.2: 冻结 Client Registration Schema 与 Onboarding 最小流程`
3. `Story 6.3: 冻结 Discovery、Detail 与 Refresh 公共字段集`
4. `Story 6.4: 冻结 Skill Invoke 与 Knowledge Query Request/Response Spec`
5. `Story 6.5: 冻结 Enterprise Model Discovery 公共契约`
6. `Story 6.6: 冻结错误码矩阵与客户端状态矩阵`
7. `Story AP-6.7: 提供 Mock、Fixture 与 Contract Conformance 套件`
8. `Story AP-6.8: 完成 Cherry Studio First / Codex Second 签核路径`

Rationale:

- 把本轮 CC 的核心问题落到可执行 backlog。
- 已完成的 AP-1 到 AP-5 保持稳定。
- 缺失的签核级契约、model discovery、错误/状态矩阵、mock fixture 有明确承接位置。
- Cherry Studio first 与 Codex second 的验证顺序被写进 story，而不是停留在口头策略。

### Proposal 2: 将 AP-6 加入 sprint status 跟踪

Artifact: `_bmad-output/implementation-artifacts/sprint-status.yaml`

Status: approved and applied

Change:

- 新增 `ap-epic-6: backlog`
- 新增 AP-6.1 到 AP-6.8 story keys，全部为 `backlog`
- 新增 `ap-epic-6-retrospective: optional`
- AP-1 到 AP-5 状态保持不变

New status entries:

```yaml
ap-epic-6: backlog
ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract: backlog
ap-6-2-freeze-client-registration-schema-and-onboarding-flow: backlog
ap-6-3-freeze-discovery-detail-and-refresh-public-fields: backlog
ap-6-4-freeze-skill-invoke-and-knowledge-query-specs: backlog
ap-6-5-freeze-enterprise-model-discovery-contract: backlog
ap-6-6-freeze-error-code-and-client-state-matrices: backlog
ap-6-7-provide-mock-fixture-and-contract-conformance-suite: backlog
ap-6-8-complete-cherry-studio-first-and-codex-second-signoff: backlog
ap-epic-6-retrospective: optional
```

Rationale:

- 让 AP-6 进入 BMad sprint tracking。
- 为后续 create-story / dev-story / review 流程提供稳定 story keys。
- 不重开或降级已经完成的 AP-1 到 AP-5。

### Proposal 3: 更新 AP PRD / Addendum 的 AP-6 定位

Artifacts:

- `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/prd.md`
- `_bmad-output/planning-artifacts/prds/prd-agent-platform-2026-05-31/addendum.md`

Status: approved and applied

Change:

- 在 PRD `revision_notes` 中追加 V1.4 说明。
- 将 addendum `updated` 更新为 `2026-06-02`。
- 在 addendum 中新增 V1.4 `revision_notes`。
- 将 `## 5. AP-6 的具体定位` 拆为：
  - `5.1 2026-05-31 原始定位`
  - `5.2 2026-06-02 Correct Course 重处理后的定位`
  - `5.3 AP-6 MVP Exit Gate`

Rationale:

- 不重写 PRD 主体，避免扰动已稳定的平台愿景和 FR。
- 明确 AP-6 是 AP-1 到 AP-5 完成后的签核 gate，不是重建标准层，也不是 Cherry Studio 专属适配。
- 为后续 architecture amendment、UX、docs 和 story 实施提供统一引用点。

### Proposal 4: 追加 AP-6 Architecture Amendment

Artifact: `_bmad-output/planning-artifacts/architecture-agent-platform.md`

Status: approved and applied

Change:

- 在 `## Workflow Completion` 前追加 `## V1.4 Architecture Amendment: AP-6 Public Contract Freeze`
- 明确 AP-6 是 contract/signoff layer，不新增第四个 runtime plane
- 冻结 enterprise model discovery 必须作为 downstream public projection
- 定义公共契约 source of truth 为 docs + OpenAPI + fixtures
- 将错误码矩阵和客户端状态矩阵定义为 contract artifacts
- 明确 fixture 不得包含 secrets 或真实租户数据
- 明确 Cherry Studio first / Codex second 的复用验证规则

Rationale:

- 解决 AP-6 的架构归属，避免后续实现直接复用 `/api/models` 或 `/v1/models` 作为下游公共契约。
- 保持既有 control plane / auth plane / open capability plane 不变。
- 为后续 OpenAPI、docs、mock / fixture、conformance tests 提供架构约束。

### Proposal 5: 补充 AP-6 UX 最小要求

Artifact: `_bmad-output/planning-artifacts/ux-agent-platform.md`

Status: approved and applied

Change:

- 将 `updated` 更新为 `2026-06-02`
- 新增 V1.4 `revision_notes`
- 新增 AP-6 Clients onboarding 最小交互
- 新增 OAuth / consent 可见性要求
- 新增下游契约签核视图要求
- 新增 model discovery 状态展示要求
- 新增客户端状态矩阵展示要求
- 扩展 Clients 页面最低要求
- 新增 `Contract Signoff` 页面级最低要求

Rationale:

- 补齐 readiness report 中指出的 consent visibility 弱点。
- 让 AP-6 的签核资产在 UI / 运营层有可解释位置。
- 不要求现在做完整高保真视觉设计，但冻结最小交互规则，避免前端继续只停留在 shell。

### Proposal 6: 将下游 contract spec 升级为 AP-6 冻结候选

Artifact: `docs/agent-platform-downstream-contract-spec.md`

Status: approved and applied

Change:

- 将状态更新为 `draft / AP-6 contract-freeze candidate`
- 将版本更新为 `v0.2`
- 将更新日期改为 `2026-06-02`
- 新增对应规划：`AP-6 下游公共契约冻结与接入签核`
- 将文档用途从“当前实现状态说明”升级为“AP-6 公共契约冻结候选稿”
- 新增 `3A. AP-6 签核原则`
- 新增 `11. AP-6 待冻结契约清单`
- 将原 readiness verdict / 审阅建议顺延为第 12 / 13 节

AP-6 待冻结契约清单覆盖：

1. OAuth / Token / Revoke Wire Contract
2. Client Registration Contract
3. Discovery / Detail / Refresh Contract
4. Skill Invoke and Knowledge Query Contracts
5. Enterprise Model Discovery Contract
6. Error Matrix and Client State Matrix
7. Mock / Fixture / Conformance
8. Consumer Signoff

Rationale:

- 把 v0.1 从“实现状态说明”提升为 AP-6 合同冻结候选。
- 不强行一次性填完所有字段细节，但明确每个 AP-6 story 的 contract output。
- 给 OpenAPI / fixture / conformance 后续工作一个明确对照表。

### Proposal 7: OpenAPI、Mock / Fixture、Conformance 资产延后到 AP-6 story 生成

Artifacts:

- `docs/openapi/api.json`
- mock / fixture / conformance test artifacts

Status: approved by implication through AP-6 scope; not applied in this CC pass

Decision:

- 本轮 Correct Course 不直接修改 `docs/openapi/api.json`。
- 本轮 Correct Course 不直接创建 mock / fixture / conformance test artifacts。
- 这些资产必须由 AP-6 story 基于冻结后的 contract spec 生成，并与 `docs/agent-platform-downstream-contract-spec.md` 保持一致。

Rationale:

- `docs/openapi/api.json` 需要具体 endpoint、schema 和 response shape；在 AP-6 规格尚未逐项冻结前直接写入会制造二次返工。
- Mock / fixture / conformance tests 必须与最终 contract version、字段等级、错误矩阵和 model discovery 契约一致。
- Architecture amendment 已明确 source of truth 是 docs + OpenAPI + fixtures，三者不一致时 AP-6 story 不得 done。

## 6. Implementation Handoff

### 6.1 Scope Classification

**Moderate**

本轮不是重做 Agent Platform，也不是回滚 AP-1 到 AP-5；它是一次规划与签核层补齐：

- 新增 AP-6 epic
- 更新 PRD addendum / architecture / UX / contract spec
- 更新 sprint-status
- 后续由 AP-6 stories 继续生成 OpenAPI、mock / fixture、conformance tests 和必要代码补丁

### 6.2 Handoff Recipients

#### Product / Planning Owner

负责确认 AP-6 的签核范围和验收顺序：

- AP-6 是否作为 Cherry Studio / Codex 正式接入前的 MVP exit gate
- Cherry Studio first consumer 的 signoff 标准
- Codex second-consumer review 的最小要求
- 完整 Clients UI 是否留在 AP-6，还是拆到后续 AP-7

#### Architect

负责 AP-6 架构约束落地：

- model discovery downstream projection surface
- docs + OpenAPI + fixtures source-of-truth 一致性
- 错误/状态矩阵
- contract version / namespaced extension 规则
- 禁止污染 `/v1/**` 和 `relay/**`

#### Developer / Tech Writer

负责按 AP-6 story 推进：

1. `ap-6-1` OAuth wire contract
2. `ap-6-2` client registration / onboarding
3. `ap-6-3` discovery / detail / refresh fields
4. `ap-6-4` Skill / Knowledge specs
5. `ap-6-5` enterprise model discovery
6. `ap-6-6` error/state matrices
7. `ap-6-7` mock / fixture / conformance
8. `ap-6-8` consumer signoff

#### UX / Frontend Owner

负责把 AP-6 的运营与签核状态变成可解释体验：

- Clients onboarding checklist
- OAuth / consent 可见性
- Contract Signoff 视图
- model discovery 状态展示
- 客户端错误/状态矩阵展示

### 6.3 Success Criteria

本轮 Correct Course 后的成功标准：

1. AP-6 出现在 `epics-agent-platform.md` 和 `sprint-status.yaml`。
2. PRD addendum 明确 AP-6 是 contract-freeze / signoff gate。
3. Architecture 明确 AP-6 不新增 runtime plane，model discovery 不能直接等同现有模型接口。
4. UX 明确 Clients onboarding、consent visibility、Contract Signoff、model/status 展示要求。
5. Contract spec 升级为 AP-6 freeze candidate。
6. AP-1 到 AP-5 保持 done，不回滚、不重开。
7. 后续 OpenAPI / fixture / conformance 由 AP-6 stories 生成，不在本轮提前硬写。

### 6.4 Recommended Next Step

下一步建议创建：

`ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract`

它是 AP-6 的入口 story，先冻结 OAuth / token / revoke / callback / allowlist，再让后续 discovery、model discovery、fixture 等 story 有稳定鉴权基础。

## 7. Final Review And Approval

### 7.1 Checklist Status

| Checklist Item | Status | Notes |
| --- | --- | --- |
| 1.1 Triggering story identified | Done | 触发点不是单个失败 story，而是 AP-1 到 AP-5 完成后发现公共契约签核缺口。 |
| 1.2 Core problem defined | Done | 问题被重新定义为实施后规划收敛 / 公共契约冻结缺口。 |
| 1.3 Evidence collected | Done | 证据来自 PRD、Architecture、Epics、sprint-status、gap matrix、contract spec 与现有 Agent Platform 实现路径。 |
| 2.1 Current epic assessed | Done | AP-1 到 AP-5 保持 done，不重开、不回滚。 |
| 2.2 Epic-level changes determined | Done | 新增 AP-6 承接公共契约冻结与接入签核。 |
| 2.3 Remaining epics reviewed | Done | 企业治理 epics 保持不变，Agent Platform 后续工作从 AP-6 进入。 |
| 2.4 New epics needed | Done | AP-6 已加入 `epics-agent-platform.md`。 |
| 2.5 Epic order and priority considered | Done | AP-6 作为 Cherry Studio / Codex 正式接入前的 MVP exit gate。 |
| 3.1 PRD conflict check | Done | PRD 主体保留，addendum 增补 V1.4 AP-6 定位。 |
| 3.2 Architecture conflict check | Done | 追加 V1.4 architecture amendment，明确 AP-6 不新增 runtime plane。 |
| 3.3 UX conflict check | Done | UX 增补 Clients onboarding、consent visibility、Contract Signoff 与状态展示要求。 |
| 3.4 Other artifact impact | Done | sprint-status 与 downstream contract spec 已更新；OpenAPI、mock、fixture、conformance 延后到 AP-6 stories。 |
| 4.1 Direct Adjustment | Viable | 通过新增 AP-6 与更新规划产物完成，不推翻既有平台底座。 |
| 4.2 Potential Rollback | Not viable | 回滚 AP-1 到 AP-5 不会降低签核缺口风险，反而破坏已完成底座。 |
| 4.3 PRD MVP Review | Viable | AP-6 被设为正式下游接入前的 MVP exit gate。 |
| 4.4 Recommended path selected | Done | 采用 Direct Adjustment + MVP Exit Gate Review。 |
| 5.1 Issue summary | Done | 已重写并批准。 |
| 5.2 Epic/artifact adjustments | Done | AP-6、PRD addendum、Architecture、UX、contract spec、sprint-status 已更新。 |
| 5.3 Recommended path rationale | Done | 已记录取舍、范围、风险与时间影响。 |
| 5.4 MVP impact and action plan | Done | AP-6 成为 contract-freeze / signoff gate；后续从 AP-6.1 开始。 |
| 5.5 Agent handoff plan | Done | 已路由到 Product / Planning、Architect、Developer / Tech Writer、UX / Frontend Owner。 |
| 6.1 Checklist completion reviewed | Done | 所有适用项均已处理。 |
| 6.2 Proposal accuracy verified | Done | 提案与当前 Agent Platform artifacts、sprint-status 和 downstream contract spec 一致。 |
| 6.3 Explicit user approval | Done | 用户在 Incremental 审阅中以 `a` / `A` 批准各段与最终收尾。 |
| 6.4 sprint-status updated | Done | AP-6 epic 与 AP-6.1 到 AP-6.8 story keys 已加入 backlog。 |
| 6.5 Handoff plan confirmed | Done | Scope 为 Moderate，路由到 Product Owner / Developer，并需要 Architect、UX / Frontend、Tech Writer 协同。 |

### 7.2 Final Decision

本 Sprint Change Proposal 已批准进入实施路由。

最终决策：

1. 保留 AP-1 到 AP-5 的完成态。
2. 新增 AP-6：下游公共契约冻结与接入签核。
3. AP-6 作为 Cherry Studio first consumer 和 Codex second consumer 正式接入前的 MVP exit gate。
4. 本轮不直接修改 `docs/openapi/api.json`，不直接生成 mock / fixture / conformance assets。
5. 后续从 `ap-6-1-freeze-oauth-token-revoke-and-callback-wire-contract` 开始创建 story 并执行。

### 7.3 Workflow Completion

- Issue addressed: Agent Platform 后端能力已落地后，缺少下游 P0 可签核公共契约、model discovery、错误/状态矩阵、mock fixture 与 onboarding 收敛。
- Change scope: Moderate。
- Artifacts modified: PRD、PRD addendum、Epics、Architecture、UX、sprint-status、downstream contract spec。
- Routed to: Product Owner / Developer；同时需要 Architect、UX / Frontend Owner、Developer / Tech Writer 协同。

Correct Course workflow complete.
