# Investigation: 预算池创建后为何不能修改

## Hand-off Brief

1. **What happened.** 当前仓库中的部门预算池只实现了“创建 + 查询/列表/详情”，没有任何更新/停用接口或前端入口；“创建后不能直接修改类型”是需求明确规定，但“其他字段也完全不能调整”并非完整需求结论，而是实现缺口叠加造成的现状。
2. **Where the case stands.** 状态为 Concluded；已确认后端路由、控制器、服务和前端 API/页面都不存在预算池编辑能力，同时 PRD/Story 仅覆盖创建，不覆盖编辑闭环。
3. **What's needed next.** 如果产品预期是“预算池可调整”，需要先补需求：定义哪些字段可编辑、哪些必须走“停用旧池并新建”；如果产品预期沿用现有 PRD，则至少要补“停用旧池”能力，否则需求写的替代路径本身无法执行。

## Case Info

| Field            | Value                                                                                           |
| ---------------- | ----------------------------------------------------------------------------------------------- |
| Ticket           | N/A                                                                                             |
| Date opened      | 2026-06-03                                                                                      |
| Status           | Concluded                                                                                       |
| System           | Windows / PowerShell / Go + React monorepo                                                      |
| Evidence sources | PRD, epics, story artifact, router, controller, service, model, frontend API, frontend page    |

## Problem Statement

用户反馈“当前一创建的预算池不能修改”，希望确认这是需求如此设计，还是代码没有实现。

## Evidence Inventory

| Source | Status | Notes |
| ------ | ------ | ----- |
| PRD | Available | 明确写了类型不可切换，变更应停用旧池并新建。 |
| Epic / Story | Available | Story 3.1 仅要求创建预算池，没有编辑 AC。 |
| Router | Available | 预算池仅注册 GET/POST，无 PUT/PATCH/停用路由。 |
| Controller | Available | 仅有 Create/Get/List/Detail，无 Update/Disable。 |
| Service | Available | 仅实现 Create/Get/List/Detail，无更新逻辑。 |
| Frontend API | Available | 仅封装 create/get/list/detail，无 update。 |
| Frontend UI | Available | 只有“Create Budget Pool”表单和列表/详情展示，无编辑或停用入口。 |

## Investigation Backlog

| # | Path to Explore | Priority | Status | Notes |
| - | --------------- | -------- | ------ | ----- |
| 1 | 确认预算池需求边界 | High | Done | 已核对 PRD / Epic / Story 3.1 |
| 2 | 确认后端是否支持更新 | High | Done | 已核对 router / controller / service |
| 3 | 确认前端是否暴露编辑入口 | High | Done | 已核对 api.ts / index.tsx |
| 4 | 确认“停用旧池并新建”是否已落地 | High | Done | 未发现对外停用能力 |

## Timeline of Events

| Time | Event | Source | Confidence |
| ---- | ----- | ------ | ---------- |
| 2026-06-03 | 调查开始，用户提出“预算池不能修改” | 用户描述 | Confirmed |
| 2026-06-03 | 确认企业预算池路由只有 GET/POST | `router/enterprise-router.go:17-20` | Confirmed |
| 2026-06-03 | 确认控制器只有 Create/Get/List/Detail | `controller/enterprise/department_budget.go:16-222` | Confirmed |
| 2026-06-03 | 确认服务层只有 Create/Get/List/Detail | `service/enterprise/department_budget.go:135-325` | Confirmed |
| 2026-06-03 | 确认前端 API 只有 create/get/list/detail | `web/default/src/features/enterprise-organization/api.ts:324-378` | Confirmed |
| 2026-06-03 | 确认前端只有创建入口，无编辑入口 | `web/default/src/features/enterprise-organization/index.tsx:2160-2183`, `2790-2821`, `4103-4215`, `4432-4547` | Confirmed |
| 2026-06-03 | 确认 PRD 规定“类型不可切换；如需变更，必须先停用旧池并新建” | `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:248-252` | Confirmed |
| 2026-06-03 | 确认 Story 3.1 只覆盖创建预算池，无编辑验收项 | `_bmad-output/planning-artifacts/epics.md:439-462`, `_bmad-output/implementation-artifacts/3-1-create-department-budget-pool.md:15-49` | Confirmed |

## Confirmed Findings

### Finding 1: 预算池“直接修改”能力未实现

**Evidence:** `router/enterprise-router.go:17-20`; `controller/enterprise/department_budget.go:16-222`; `service/enterprise/department_budget.go:135-325`

**Detail:** 后端仅支持 `GET /departments/:id/budget`、`GET /departments/:id/budgets`、`GET /departments/:id/budgets/:budget_id` 和 `POST /departments/:id/budget`。没有 `PUT` / `PATCH` / `POST .../supersede` / `.../disable` 等预算池更新路径。

### Finding 2: 前端没有预算池编辑或停用入口

**Evidence:** `web/default/src/features/enterprise-organization/api.ts:324-378`; `web/default/src/features/enterprise-organization/index.tsx:2160-2183`; `web/default/src/features/enterprise-organization/index.tsx:2790-2821`; `web/default/src/features/enterprise-organization/index.tsx:4103-4215`; `web/default/src/features/enterprise-organization/index.tsx:4432-4547`

**Detail:** 前端只封装了创建、查询、列表、详情接口；页面上只有 “Create Budget Pool” 提交动作以及预算池列表/概览，没有编辑按钮、停用按钮、二次提交修改表单或替代流程入口。

### Finding 3: 需求明确限制“类型不可切换”，但没有要求“支持直接编辑”

**Evidence:** `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md:248-252`; `_bmad-output/planning-artifacts/epics.md:449-462`

**Detail:** PRD 写明“单个部门预算池实例一经创建不可在 `balance` 与 `subscription` 间切换；如需变更，必须先停用旧池并新建。”Story 3.1 的验收标准也只覆盖创建与非法输入拒绝，没有“编辑预算池”验收项。

### Finding 4: “停用旧池并新建”这条替代路径本身也未完整实现

**Evidence:** `model/enterprise/department_budget.go:14-17`; `router/enterprise-router.go:17-20`; `controller/enterprise/department_budget.go:16-222`; `web/default/src/features/enterprise-organization/index.tsx:2790-2821`

**Detail:** 模型定义了 `paused` / `revoked` / `expired` 状态，但对外没有对应预算池停用管理接口，也没有前端停用操作。因此 PRD 中“如需变更，必须先停用旧池并新建”目前无法通过正式治理 UI/API 完成。

## Deduced Conclusions

### Deduction 1: 当前“预算池完全不能改”是需求约束与实现缺口共同造成的

**Based on:** Finding 1, Finding 3, Finding 4

**Reasoning:** 需求只明确禁止类型切换，并要求用“停用旧池并新建”替代；但实际实现既没有直接编辑，也没有停用旧池动作，所以用户感知到的是“创建后完全不可改”。

**Conclusion:** 这不是纯粹“产品明确要求完全不允许修改”，也不是单纯“忘了做一个编辑按钮”；更准确地说，V1 只交付了创建能力，没有把变更治理闭环做完。

## Hypothesized Paths

### Hypothesis 1: 产品原始意图是允许部分字段调整，但未进入 Story 范围

**Status:** Open

**Theory:** 例如过期时间、状态等字段可能本应允许治理层调整，但 Epic 3 / Story 3.1 只做了最小创建闭环。

**Supporting indicators:** PRD 只硬性禁止“类型切换”，并未逐项声明所有其他字段都不可编辑。

**Would confirm:** 出现补充 PRD / 产品说明，明确哪些字段可 edit、哪些只能 supersede/new。

**Would refute:** 产品方明确声明预算池一旦创建除查看外全部不可改，仅允许额外新建。

**Resolution:** 暂无，需产品确认。

## Missing Evidence

| Gap | Impact | How to Obtain |
| --- | ------ | ------------- |
| 产品对“非类型字段是否允许编辑”的明确规则 | 决定后续应该补编辑还是补停用/新建流程 | 让 PM/需求补一条 FR 或 addendum |

## Source Code Trace

| Element | Detail |
| ------- | ------ |
| Error origin | 非运行时报错；是功能缺失，体现在预算池路由/控制器/前端入口缺席 |
| Trigger | 用户在企业组织预算治理页面创建预算池后尝试再次调整 |
| Condition | 系统仅提供 create/list/detail，无 update/disable 路径 |
| Related files | `router/enterprise-router.go`, `controller/enterprise/department_budget.go`, `service/enterprise/department_budget.go`, `web/default/src/features/enterprise-organization/api.ts`, `web/default/src/features/enterprise-organization/index.tsx` |

## Conclusion

**Confidence:** High

可以明确分成两层：

1. **需求上已明确的部分：** 预算池类型不能在 `balance` 和 `subscription` 之间直接切换，若要变更类型，应该“停用旧池并新建”。  
2. **实现上缺失的部分：** 当前系统没有实现预算池编辑，也没有实现“停用旧池”这一替代治理动作，因此最终用户体验成了“预算池创建后完全不能改”。

所以更准确的答案是：**不是单纯没需求，也不是简单一个编辑按钮漏做；当前交付只做了创建，没有把预算池变更闭环做完。**

## Recommended Next Steps

### Fix direction

先确认产品策略，再实现对应能力：

- 如果只允许“类型不可编辑，其他字段可编辑”，补预算池 update API/UI，并对类型字段做后端硬校验。
- 如果坚持“预算池整体不可编辑，只允许停用后新建”，至少补预算池停用动作、审计、前端入口和替换流程提示。
- 如果希望两者兼有，则需要明确字段级规则，例如：`type` 不可改，`expires_at` / `status` 可改，`quota` 是否允许改要单独定义。

### Diagnostic

无需更多技术诊断；当前阻塞点主要是产品规则未细化，以及变更治理动作未实现。

## Reproduction Plan

1. 打开企业组织预算治理页。
2. 创建任意一个预算池。
3. 观察页面仅提供列表/详情查看，不提供编辑或停用入口。
4. 检查网络层与后端路由，可见只存在创建与查询接口，没有预算池 update/disable 接口。

## Side Findings

- 预算池模型已经具备 `paused` / `revoked` / `expired` 状态定义，但这些状态目前更多被内部同步或测试使用，不是用户可直接操作的治理入口。证据：`model/enterprise/department_budget.go:14-17`。
