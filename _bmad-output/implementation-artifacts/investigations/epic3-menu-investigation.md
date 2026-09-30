# Investigation: Epic 3 Menu Visibility

## Hand-off Brief

1. **What happened.** 用户认为 Epic 3 的“菜单”在当前页面消失了，但已确认企业组织一级菜单、前端路由、后端预算/分配 API 仍然存在，未发现合并后被删除的证据。
2. **Where the case stands.** 当前最强结论是：这不是“功能代码被 merge 丢了”，而是入口位置认知偏差，或 `企业组织` 页面内 `Department Budget` Tab 的可见性/布局问题。
3. **What's needed next.** 优先在实际页面确认 `企业组织` 顶部 Tab 是否被横向挤出可视区；若是，则修复 Tab 容器的窄屏展示策略。

## Case Info

| Field            | Value                                                                 |
| ---------------- | --------------------------------------------------------------------- |
| Ticket           | N/A                                                                   |
| Date opened      | 2026-05-31                                                            |
| Status           | Active                                                                |
| System           | Windows workspace, Go backend + React default frontend                |
| Evidence sources | Source code, route generation, git history, user-provided screenshot |

## Problem Statement

用户描述：Epic 3 的功能“之前版本是有的，但现在页面上没有这个菜单了”，希望判断是前端问题，还是代码合并时直接丢失。

## Evidence Inventory

| Source | Status | Notes |
| ------ | ------ | ----- |
| 用户截图 | Available | 左侧一级菜单中仍可见 `企业组织`、`部门用量总览`、`企业风险事件`、`钉钉集成`。 |
| 后端路由 | Available | `router/enterprise-router.go` 中 Epic 3 预算池与分配接口仍在。 |
| 默认前端路由 | Available | `/_authenticated/enterprise-organization/` 路由仍注册到 `EnterpriseOrganization` 页面。 |
| 默认前端菜单 | Available | 左侧 Admin 导航仍包含 `Enterprise Organization`。 |
| 页面实现 | Available | `EnterpriseOrganization` 页面仍包含 `Department Budget` Tab 和预算/分配面板。 |
| Git 历史 | Available | 未发现将 Epic 3 入口移除的提交；相反可追溯到添加入口与预算 Tab 的提交。 |
| 浏览器运行态截图 | Missing | 尚未直接看到 `企业组织` 页面内容区，因此未对 Tab 是否被挤出可视区做运行态确认。 |

## Investigation Backlog

| # | Path to Explore | Priority | Status | Notes |
| - | --------------- | -------- | ------ | ----- |
| 1 | 运行态确认 `企业组织` 页顶部 Tab 可见性 | High | Open | 需要实际页面截图或本地运行态验证。 |
| 2 | 若 Tab 被遮挡，检查 `TabsList` 窄屏样式 | High | Open | 当前实现无换行或横向滚动策略。 |

## Timeline of Events

| Time | Event | Source | Confidence |
| ---- | ----- | ------ | ---------- |
| 2026-05-28 22:54 +0800 | 默认前端加入 `Enterprise Organization` 侧边栏入口与页面主体 | commit `0b244f23d`, `0978fc6e3` | Confirmed |
| 2026-05-29 13:21 +0800 | Epic 3.1 加入 `Department Budget` Tab 与预算面板入口 | commit `c1d6f4b52` | Confirmed |
| 2026-05-29 17:35 +0800 | Epic 3.5 扩展预算池、wallet 状态与 allocation 展示 | commit `f760d3e8` | Confirmed |
| 2026-05-31 | 用户报告“菜单没了”，并提供当前左侧菜单截图 | user message | Confirmed |

## Confirmed Findings

### Finding 1: 左侧一级菜单 `企业组织` 仍然存在

**Evidence:** 用户截图；`web/default/src/hooks/use-sidebar-data.ts:137`

**Detail:** 当前默认前端 Admin 菜单仍声明 `Enterprise Organization`，URL 为 `/enterprise-organization`。

### Finding 2: Epic 3 后端 API 没有被删

**Evidence:** `router/enterprise-router.go:17`, `router/enterprise-router.go:18`, `router/enterprise-router.go:19`, `router/enterprise-router.go:20`, `router/enterprise-router.go:21`, `router/enterprise-router.go:22`, `router/enterprise-router.go:23`

**Detail:** 部门预算池查询、预算列表、预算详情、创建预算池、quota allocation 查询、创建、撤销接口全部仍在注册。

### Finding 3: 默认前端页面路由没有被删

**Evidence:** `web/default/src/routes/_authenticated/enterprise-organization/index.tsx:23`

**Detail:** `/enterprise-organization` 仍然路由到 `EnterpriseOrganization` 组件。

### Finding 4: Epic 3 预算功能入口仍在页面源码中

**Evidence:** `web/default/src/features/enterprise-organization/index.tsx:300`, `web/default/src/features/enterprise-organization/index.tsx:305`, `web/default/src/features/enterprise-organization/index.tsx:319`, `web/default/src/features/enterprise-organization/index.tsx:1038`, `web/default/src/features/enterprise-organization/index.tsx:1317`

**Detail:** `EnterpriseOrganization` 页面顶部仍有四个 Tab，其中最后一个是 `Department Budget`；预算面板和 `Create wallet allocation` 表单仍然挂载。

### Finding 5: 当前代码相对 Epic 3.5 没有移除预算入口

**Evidence:** `git diff f760d3e8..HEAD -- web/default/src/features/enterprise-organization/index.tsx`

**Detail:** 现有差异只有格式化与局部排版调整，未删除 `TabsTrigger value='budgets'` 或 `DepartmentBudgetPanel`。

### Finding 6: Classic 旧页本来就没有独立“预算池菜单”

**Evidence:** `web/classic/src/pages/Enterprise/Department.js:80`, `web/classic/src/pages/Enterprise/Department.js:107`

**Detail:** Classic 的企业组织页只包含“用户所属部门”和“部门成员”两块，不存在单独的 Epic 3 左侧菜单项。

## Deduced Conclusions

### Deduction 1: “代码合并把 Epic 3 功能直接丢了”不成立

**Based on:** Finding 1, Finding 2, Finding 3, Finding 4, Finding 5

**Reasoning:** 如果 Epic 3 被 merge 丢失，应至少看到路由、页面组件、预算 Tab、或后端 API 中的一条链路缺失；但当前四条链路都仍完整存在。

**Conclusion:** 问题不属于“合并删除实现”，而属于前端入口呈现或对入口位置的认知偏差。

### Deduction 2: 用户所说“菜单”更可能指 `企业组织` 页面里的预算 Tab

**Based on:** Finding 1, Finding 4, Finding 6

**Reasoning:** 左侧一级菜单 `企业组织` 还在，而 Epic 3 在默认前端的真实入口是 `企业组织` 页内的 `Department Budget` Tab，不是独立左侧菜单；Classic 也没有该独立菜单传统。

**Conclusion:** 用户很可能在找一个“不再独立存在的左侧菜单”，而当前实现把功能放进了页内 Tab。

## Hypothesized Paths

### Hypothesis 1: `Department Budget` Tab 在当前布局下被挤出可视区

**Status:** Open

**Theory:** `TabsList` 使用 `inline-flex w-fit`，未提供换行或横向滚动；在较窄内容区中，第四个 Tab 可能被裁掉或需要横向滚动但没有可见 affordance。

**Supporting indicators:** `web/default/src/components/ui/tabs.tsx:42` 到 `web/default/src/components/ui/tabs.tsx:66` 未定义 wrap/overflow 策略；`web/default/src/features/enterprise-organization/index.tsx:301` 直接使用默认 `TabsList`。

**Would confirm:** 运行态截图显示 `企业组织` 页顶部仅见前三个 Tab，或第四个 Tab 被截断/不可见。

**Would refute:** 运行态中四个 Tab 全部清晰可见，且点击正常。

**Resolution:** 待运行态验证。

### Hypothesis 2: 用户把“一级菜单”和“页内 Tab”混为同一层级

**Status:** Open

**Theory:** 用户预期 Epic 3 预算能力应出现在左侧独立菜单，但当前产品实现从一开始就是 `企业组织` 下的页内 Tab。

**Supporting indicators:** 当前一级菜单正常存在；Classic 历史中无独立预算菜单；默认前端源码把预算能力置于 `EnterpriseOrganization` 内部。

**Would confirm:** 用户打开 `企业组织` 后能看到 `部门预算池` Tab，或在接受入口位置说明后确认“原来找的是这个”。

**Would refute:** 用户提供 `企业组织` 页面截图，显示该 Tab 确实曾是一级/二级导航而现在被删。

**Resolution:** 待用户确认页面截图。

## Missing Evidence

| Gap | Impact | How to Obtain |
| --- | ------ | ------------- |
| 当前 `企业组织` 页面截图 | 能直接确认是入口误解还是 Tab 被布局遮挡 | 让用户补充页面内容区截图，或本地启动前端验证 |

## Source Code Trace

| Element | Detail |
| ------- | ------ |
| Error origin | `web/default/src/features/enterprise-organization/index.tsx:305` |
| Trigger | 用户进入 `/enterprise-organization` 页面后查看顶部 Tab |
| Condition | 若内容区宽度不足，`TabsList` 可能无法完整显示四个 Tab |
| Related files | `web/default/src/components/ui/tabs.tsx`, `web/default/src/hooks/use-sidebar-data.ts`, `router/enterprise-router.go` |

## Conclusion

**Confidence:** Medium

已确认这不是 Epic 3 代码在 merge 中丢失。左侧 `企业组织` 菜单、默认前端路由、预算页面实现、后端预算/分配 API 都仍然存在。当前更可能的解释是：Epic 3 功能入口在默认前端一直位于 `企业组织` 页面内的 `部门预算池` Tab，而不是独立左侧菜单；如果用户在该页面里仍看不到预算入口，则下一优先级问题是 Tab 在当前布局/宽度下被挤出可视区。

## Recommended Next Steps

### Fix direction

若运行态确认 `部门预算池` Tab 被遮挡，优先修复 `TabsList` 的窄屏策略，例如增加 `overflow-x-auto`、可滚动提示，或在小屏下允许换行。

### Diagnostic

打开 `企业组织` 页面，确认顶部是否存在以下 4 个 Tab：`部门树`、`用户所属部门`、`部门成员`、`部门预算池`。若缺少最后一个，进一步用浏览器开发者工具检查 Tab 容器实际宽度与裁剪情况。

## Reproduction Plan

1. 以管理员身份进入默认前端。
2. 打开左侧 `企业组织`。
3. 观察页面顶部 Tab 是否完整显示 4 项。
4. 若只见前三项，缩放浏览器或检查容器宽度，验证 `部门预算池` 是否被裁剪。

## Side Findings

- `SidebarModulesAdmin` 的默认配置未显式声明 `enterprise` / `enterprise_alerts` / `enterprise_usage` / `enterprise_dingtalk`，但当前前端过滤逻辑对未映射 URL 默认可见，因此这不是本次“菜单消失”的主因。证据：`web/default/src/hooks/use-sidebar-config.ts:175`
