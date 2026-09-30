# Story AP-5.4: 收口 Agent Platform `web/default` 控制面集成与体验一致性

Status: done

## Story

As a 平台管理员,
I want Agent Platform 控制面在 `web/default` 中以真实可访问、可取数、可国际化且与既有管理模块一致的方式交付,
so that 这个产品线的控制面不是 story 占位壳，而是可持续扩展的正式管理入口。

## Acceptance Criteria

1. **Given** 侧边栏已出现 Agent Platform 导航入口  
   **When** 管理员点击进入  
   **Then** 真实 route surface 可访问且不会因生成路由缺失而 404  
   **And** 该入口与 `web/default` 其余 admin 模块处于同一导航与布局体系中。

2. **Given** Skill、Knowledge、Agent 控制面入口已经存在  
   **When** 页面请求对应数据  
   **Then** 页面能够连通 live `/api/agent-platform/**` 控制面接口或显式批准的兼容 fallback  
   **And** 不会因前后端路由接线不一致而出现默认 404。

3. **Given** Agent Platform 页面在 `web/default` 中渲染  
   **When** 页面展示文案、状态、空态和错误态  
   **Then** 新增文案遵循 frontend i18n 约束  
   **And** 页面结构、状态表达与现有 `enterprise-*` 模块一致，不以 narrative shell 视作完成。

## Tasks / Subtasks

- [x] 修复 Agent Platform 前端 route surface 与生成路由接入 (AC: 1)
  - [x] 确认 `/_authenticated/agent-platform/` 进入生成路由树，并消除点击侧边栏后的前端 404。
  - [x] 保持与现有 admin 导航、鉴权 guard、layout 结构一致。

- [x] 收口 Skill / Knowledge / Agent 数据请求链路 (AC: 2)
  - [x] 修复前端 API 请求与后端控制面路由的接线问题。
  - [x] 如专用接口不可达，提供显式、受控的兼容 fallback，而不是隐式失败。

- [x] 将 Agent Platform 页面改造成与现有 admin 模块一致的管理页 (AC: 3)
  - [x] 使用现有 `web/default` 的 loading / empty / error / status 表达方式。
  - [x] 将 narrative shell 收口成真实列表工作区，并为后续详情/抽屉扩展保留清晰入口。

- [x] 补齐前端 i18n 与定向验证 (AC: 3)
  - [x] 页面新增文案全部走 `t(...)` 并同步 locale。
  - [x] 运行 typecheck、i18n sync 与定向前端测试。

## Dev Notes

### Previous Story Insights

- `1.6 / 3.1 / 4.1 / 5.1` 已完成 bounded-context 能力落地，但轻量 CC 复盘确认它们没有完全收口真实前端 route/API/i18n/UI parity。
- 本 story 负责跨 story 集成补丁，不重做 Skill / Knowledge / Agent 的领域边界。

### Scope Boundaries

- 本 story 不新增新的 Agent Platform 资源类型或 open capability 语义。
- 本 story 重点是 `web/default` 控制面收口，而不是扩展 publish / audit 新功能域。

### Testing

- 推荐至少运行：
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run i18n:sync`
  - `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx`

### References

- [Source: _bmad-output/planning-artifacts/sprint-change-proposal-2026-06-01-agent-platform-ui-integration.md]
- [Source: _bmad-output/planning-artifacts/architecture-agent-platform.md]
- [Source: _bmad-output/planning-artifacts/ux-agent-platform.md]
- [Source: _bmad-output/implementation-artifacts/ap-1-6-deliver-web-default-control-plane-shell.md]
- [Source: _bmad-output/implementation-artifacts/ap-3-1-provide-skill-management-in-control-plane.md]
- [Source: _bmad-output/implementation-artifacts/ap-4-1-manage-knowledge-resources-with-provider-metadata.md]
- [Source: _bmad-output/implementation-artifacts/ap-5-1-provide-agent-definition-management-in-control-plane.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-01: 轻量 CC 批准后创建本 story，用于统一收口 Agent Platform `web/default` 控制面 route/API/i18n/UI 问题。

### Completion Notes List

- 已确认 `/_authenticated/agent-platform/` 存在于 `web/default` 生成路由树，侧边栏入口接入同一 admin 导航与鉴权布局体系。
- Skill / Knowledge / Agent 列表优先请求专用 `/api/agent-platform/**` 控制面接口，并仅在明确 404 时 fallback 到 typed resources 兼容接口。
- Agent Platform 页面已从 narrative shell 收口为真实资源列表工作区，使用表格、空态、错误态、状态徽标和 loading skeleton。
- Review 自动修复了 React Query 抛错时页面误入空态的问题，并补充了定向回归测试。

### File List

- `_bmad-output/implementation-artifacts/ap-5-4-stabilize-agent-platform-web-default-integration.md`
- `web/default/src/features/agent-platform/api.ts`
- `web/default/src/features/agent-platform/index.tsx`
- `web/default/src/features/agent-platform/agent-platform.test.tsx`
- `web/default/src/hooks/use-sidebar-config.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`
- `web/default/src/i18n/locales/zh.json`

### Senior Developer Review (AI)

Reviewer: GPT-5 Codex  
Date: 2026-06-03

#### Outcome

Approve after auto-fix. 所有 CRITICAL/HIGH/MEDIUM 审查问题已自动修复，剩余风险为常规前端集成风险。

#### Findings Fixed

- [HIGH] Agent Platform 资源请求在 React Query 抛出非 404 错误时没有传入 `isError/error` 给卡片，页面可能显示空态而不是错误态，未满足 AC3 的错误态要求。已修复 `ResourceListCard` 的错误判断与错误消息展示，并补充 failure response 渲染测试。
- [MEDIUM] API 链路缺少非 404 错误不 fallback 的回归测试，后续可能把 500/网络错误误当兼容 404 fallback。已新增 `does not fall back for non-404 control-plane errors` 测试，并断言专用接口请求使用 `skipErrorHandler: true`。
- [MEDIUM] Story File List 只记录故事文件，遗漏实际修改的 Agent Platform 前端源码、sidebar config 和 locale 文件，git reality 与故事记录不一致。已补齐 File List。
- [LOW] Completion Notes 仍为“待实现”，与 Tasks/Subtasks 全部勾选和实际实现状态不一致。已更新为具体完成说明。

#### AC Validation

- AC1: 已实现。`web/default/src/routes/_authenticated/agent-platform/index.tsx` 提供 admin guard route，`web/default/src/routeTree.gen.ts` 包含 `/agent-platform/` 生成路由，`use-sidebar-config.ts` 将 `/agent-platform` 映射到 admin `agent_platform` 模块。
- AC2: 已实现。`api.ts` 优先请求 `/api/agent-platform/skills`、`/knowledge-bases`、`/agents`，仅在 Axios 404 时 fallback 到 `/api/agent-platform/resources?resource_type=...`。
- AC3: 已实现并修复。页面新增文案走 `t(...)`，locale sync 报告所有语言 missing/extras/untranslated 均为 0；UI 使用 admin 模块一致的表格、Badge、Empty、ErrorState、Skeleton 和 StatusBadge。

#### Validation

- `cd web/default && bun test src/features/agent-platform/agent-platform.test.tsx` - passed, 7 tests.
- `cd web/default && bun run typecheck` - passed.
- `cd web/default && bun run i18n:sync` - passed; `_sync-report.json` shows 0 missing, 0 extras, 0 untranslated for en/fr/ja/ru/vi/zh.

## Change Log

- 2026-06-01: 根据轻量 CC proposal 新建 stabilization story，进入实现中状态。
- 2026-06-03: 完成 Agent Platform `web/default` route/API/UI/i18n 收口，并进入 review。
- 2026-06-03: Senior Developer Review 自动修复错误态与测试覆盖问题，补齐故事记录并标记 done。
