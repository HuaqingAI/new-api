---
status: draft
project: new-api
created: 2026-06-01
workflow: bmad-correct-course
mode: batch
trigger_source: local validation of Agent Platform control-plane shell on 2026-06-01 revealed route 404, control-plane API 404, missing i18n coverage, and UI parity drift from existing web/default admin modules
recommended_scope: moderate
proposed_changes_to:
  - _bmad-output/planning-artifacts/epics-agent-platform.md
  - _bmad-output/planning-artifacts/architecture-agent-platform.md
  - _bmad-output/planning-artifacts/ux-agent-platform.md
  - _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md
  - _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md
  - _bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md
  - _bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md
proposed_new_artifacts:
  - _bmad-output/implementation-artifacts/ap-5-4-stabilize-agent-platform-web-default-integration.md
---

# Sprint Change Proposal: Agent Platform 控制面集成收口

## 1. Issue Summary

### Trigger

本次 Correct Course 由 2026-06-01 的本地验证触发。触发问题不是抽象的“体验不佳”，而是 4 个明确、可复现的交付偏差：

1. Agent Platform 页面入口存在前端路由 404。
2. 页面进入后，`skills / knowledge / agents` 控制面 API 请求出现 404。
3. 页面文案未完整遵循 `web/default` i18n 约束。
4. 页面样式与交互结构明显偏离现有 `enterprise-*` 管理模块，表现为 story 占位壳而非成熟管理页。

### Core Problem

问题类型：**原始需求理解被弱化后的 story 完成口径偏差 + 验收闭环不足**。

本次排查结论不是“Agent Platform 架构主方向错了”，而是：

- 上层 PRD / Architecture / UX 对管理面要求已经写得比较清楚；
- 但在 story 落地时，这些要求被收缩成“shell / 最小真实列表承接面”；
- review 与验证又主要停留在定向单测、typecheck 和 story 内部自证，没有收口到真实 route surface、真实页面接入、真实 i18n 覆盖和与现有 admin 模块的一致性。

因此，当前问题不是单个 story 的 isolated bug，也不是需要推翻产品线架构的大改，而是 **Agent Platform 控制面在实现阶段被过度“降级交付”**。

### Evidence

#### 1. Architecture / UX 已要求完整管理面，不只是静态 shell

- Architecture 明确写明：MVP 包含完整 admin management UI，`web/default` 是主实现面。  
  Source: [architecture-agent-platform.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture-agent-platform.md:674)
- Architecture 明确要求：新增 `web/default` 文案必须国际化。  
  Source: [architecture-agent-platform.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/architecture-agent-platform.md:87)
- UX 文档明确要求：`Skills / Knowledge / Agents` 应采用“列表 + 详情/编辑抽屉 + 版本标签 + 发布历史”布局。  
  Source: [ux-agent-platform.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/planning-artifacts/ux-agent-platform.md:45)

#### 2. Story 落地口径被收缩成“最小承接面”

- `1.6` 最终完成记录只声明“控制面骨架、导航接入、基础页面结构”。  
  Source: [ap-1-6-deliver-web-default-control-plane-shell.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md:9)
- `3.1 / 4.1 / 5.1` 都把前端目标描述为“最小真实列表承接面”，没有把与现有模块一致的列表/详情/错误态/状态表达收口成强 AC。  
  Source: [ap-3-1-provide-skill-management-in-control-plane.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md:33), [ap-4-1-manage-knowledge-resources-with-provider-metadata.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md:34), [ap-5-1-provide-agent-definition-management-in-control-plane.md](/Users/hq-it/repository/github/huaqingai/new-api/_bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md:31)

#### 3. 当前实现与“已 done”状态不一致

- 前端生成路由树当前不包含 `agent-platform`，说明真实 route surface 未闭环。  
  Evidence file: [routeTree.gen.ts](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/routeTree.gen.ts:1)
- 当前 Agent Platform 页面实现仍是 story-style narrative shell，未对齐现有 enterprise 管理页结构。  
  Evidence file: [index.tsx](/Users/hq-it/repository/github/huaqingai/new-api/web/default/src/features/agent-platform/index.tsx:72)

## 2. Impact Analysis

### 2.1 Triggering Story Assessment

#### Story 1.6

- 原本目的是交付控制面骨架和导航结构。
- 但它被标记为 `done` 时，没有验证真实 route surface 是否已经进入生成路由树，也没有把 i18n 和管理页模式一致性收进 story 完成定义。
- 结论：`1.6` 不是功能方向错误，而是 story “done” 口径过松。

#### Story 3.1 / 4.1 / 5.1

- 这三个 story 的后端控制面入口方向基本正确。
- 问题在于前端侧被统一描述为“最小真实列表承接面”，这与 architecture/UX 对完整管理体验的要求存在张力。
- 结论：这些 story 的 scope 可以保留，但其完成说明与后续承接 story 需要修正。

### 2.2 Epic Impact Assessment

#### Current Epic Viability

- Epic 1-5 的总体拆分仍然成立。
- 不需要推翻 Agent Platform epics，也不需要重新定义产品线。

#### Required Epic-Level Changes

- 不建议改 epic 主体结构。
- 建议在 `epics-agent-platform.md` 中新增一个 **integration hardening / control-plane stabilization** 类 follow-up story，专门收口：
  - route surface
  - control-plane API wiring
  - i18n compliance
  - UI parity with existing `web/default` admin modules

#### Future Epic Impact

- 后续 `Publishing` 和 `Audit & Diagnostics` 页面如果继续沿“最小承接面”思路落地，会重复出现同类偏差。
- 因此需要在 epics 或 architecture note 中明确：Agent Platform UI story 的“完成”不能只等于 narrative shell + static cards。

### 2.3 PRD Impact

- PRD 主方向无需修改。
- 现有 PRD 已定义通用 Agent Platform、统一管理面、客户端消费边界与平台治理目标。
- 当前问题不构成 PRD 目标冲突。

结论：PRD 无需结构性改动。

### 2.4 Architecture Impact

- Architecture 主体方向正确。
- 但建议补一条 clarifying note：
  - “full admin management UI” 对 `web/default` story 的最低交付标准，至少包括真实 route surface、真实控制面请求链路、i18n 合规、与既有 admin 模块一致的信息架构与状态表达。

结论：仅需轻量补充说明，不需大改架构。

### 2.5 UX Impact

- UX 文档已写明页面级最低要求和关键交互。
- 当前实现明显没有达到 `Skills / Knowledge / Agents` 的统一管理布局要求。

结论：
- 需要把 UX 文档中的最低要求正式回灌到实现 story 的 AC 或 follow-up story 中。

### 2.6 Secondary Artifact Impact

- `web/default` route generation / build validation 需要被纳入验证步骤。
- Agent Platform 前端测试不能只停留在 `renderToStaticMarkup` 文案存在性断言。
- 需要补更接近真实页面接入的 smoke 或 build-time route verification。

## 3. Recommended Approach

### Selected Path

**Option 1: Direct Adjustment（轻量 CC + 新增修复 story）**

### Why This Path

这是风险最低且最符合当前事实的路径：

- 不推翻 Agent Platform 产品线架构；
- 不回滚已完成的底层后端治理模型；
- 只修正“story 完成定义 / 验收闭环 / UI 交付口径”；
- 再通过一个明确的 stabilization story 收口代码和文档。

### Alternatives Considered

#### Option 2: Potential Rollback

不推荐。

- 当前后端 `skills / knowledge / agents` 专用入口和模型分层方向是对的。
- 回滚只会让问题重新回到更弱的通用资源入口，不会帮助解决 route / i18n / UI parity。

#### Option 3: PRD MVP Review

不推荐。

- 当前问题不证明 MVP 不可达。
- 问题在实现质量和 story 口径，不在产品目标本身。

## 4. Detailed Change Proposals

### 4.1 Epics Update

Artifact: `_bmad-output/planning-artifacts/epics-agent-platform.md`

Change:
- 在 Epic 5 之后新增一个 follow-up stabilization story，建议编号：
  - `Story 5.4: 收口 Agent Platform web/default 控制面集成与体验一致性`

Rationale:
- 该问题横跨 `1.6 / 3.1 / 4.1 / 5.1`，直接塞回任一原 story 都会让责任边界模糊。
- 以新的 integration hardening story 统一收口更清晰。

### 4.2 Architecture Clarification

Artifact: `_bmad-output/planning-artifacts/architecture-agent-platform.md`

Change:
- 在 Frontend Architecture / Implementation Rules 区域补一条 clarifying note：
  - “full admin management UI” 的最低完成标准至少包括：
    - route surface 已真实接入并可访问
    - control-plane API 已真实连通
    - 页面新增文案完成 i18n
    - 页面结构与现有 `web/default` admin modules 一致，不以 narrative shell 视作完成

Rationale:
- 避免未来 stories 再把“完整管理 UI”降格成静态承接页。

### 4.3 UX Clarification

Artifact: `_bmad-output/planning-artifacts/ux-agent-platform.md`

Change:
- 在 “页面级最低要求” 下补充：
  - `Skills / Knowledge / Agents` 的 MVP 最低可用形态不是静态说明卡片，而是与现有管理页一致的真实列表工作区，至少包含：
    - loading / empty / error states
    - status badge
    - list filtering or summary grouping
    - detail entry or forward-compatible workspace slot

Rationale:
- 让 UI parity 从“风格建议”提升为可执行要求。

### 4.4 Story Completion Notes Revision

Artifacts:
- `_bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md`
- `_bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md`
- `_bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md`
- `_bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md`

Change:
- 在这些 story 的 review / change log 区域补充已知偏差说明：
  - 当前 done 状态代表“bounded-context capability landed”，不等于 Agent Platform 控制面已完成最终前端集成收口。
  - 后续由 stabilization story 统一补齐 route/API/i18n/UI parity。

Rationale:
- 防止后续把这些 story 的 `done` 误读成“用户已可无偏差使用”。

### 4.5 New Follow-up Story

New Artifact:
- `_bmad-output/implementation-artifacts/ap-5-4-stabilize-agent-platform-web-default-integration.md`

Suggested scope:
- 修复真实前端路由 404
- 修复控制面 API 请求 404 与 fallback 策略
- 补齐 Agent Platform 文案 i18n
- 将 Agent Platform 页面收口为与 `enterprise-*` 模块一致的管理页风格
- 补前端 smoke / route verification / typecheck / i18n sync 验证

## 5. Implementation Handoff

### Scope Classification

**Moderate**

原因：
- 不需要重做 PRD 或架构主线；
- 但需要修改 planning artifacts、story artifacts、前端实现与验证策略；
- 属于 backlog reorganization + implementation coordination。

### Recommended Handoff

- **PM / Architect**
  - 批准这次轻量 CC
  - 确认新增 stabilization story 的归属与编号
- **PO / Developer**
  - 更新受影响 stories 的完成口径
  - 创建 follow-up story
- **Developer**
  - 实施 route/API/i18n/UI 修复
  - 补 smoke / route / i18n 验证

### Success Criteria

- Agent Platform 页面在 `web/default` 中可真实访问，无前端 404
- `skills / knowledge / agents` 页面数据请求成功，无控制面 API 404
- 新增页面文案完成 i18n
- 页面风格与现有 admin 模块一致，不再是 narrative shell
- 相关 story / architecture / UX artifacts 对交付标准表达一致

## 6. Checklist Summary

- [x] 1.1 Triggering story identified
- [x] 1.2 Core problem defined
- [x] 1.3 Evidence gathered
- [x] 2.1 Current epic viability assessed
- [x] 2.2 Epic-level changes identified
- [x] 2.3 Future epic impact reviewed
- [x] 2.4 Need for new story confirmed
- [x] 2.5 Priority / sequencing checked
- [x] 3.1 PRD conflict checked
- [x] 3.2 Architecture conflict checked
- [x] 3.3 UX conflict checked
- [x] 3.4 Secondary artifact impact checked
- [x] 4.1 Direct adjustment evaluated
- [x] 4.2 Rollback evaluated
- [x] 4.3 PRD MVP review evaluated
- [x] 4.4 Recommended path selected
- [x] 5.1 Issue summary drafted
- [x] 5.2 Impact documented
- [x] 5.3 Recommendation documented
- [x] 5.4 High-level action plan defined
- [x] 5.5 Handoff plan defined

## 7. Approval Request

建议批准本次 **轻量 CC**：

- 不推翻 Agent Platform 架构主线；
- 只修正 story 完成口径与前端集成标准；
- 新增一个明确的 stabilization story 统一收口实现。
