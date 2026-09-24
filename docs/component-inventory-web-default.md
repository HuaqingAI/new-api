# 组件清单 — Default Web (React 19 / Rsbuild)

> 项目部分：`web-default`
> 根路径：[web/default/](../web/default/)
> 文档生成时间：2026-05-26
> 入口：`web/default/src/main.tsx`、`web/default/src/routes/__root.tsx`

---

## 1. 技术栈摘要

- React 19 + TypeScript
- 打包：Rsbuild（dev / build / preview）
- 路由：TanStack Router（文件夹路由 + `(group)`/`_layout`/`$param` 约定）
- 数据：TanStack Query + axios
- 表格：TanStack Table + TanStack Virtual
- 状态：Zustand（少量 store）
- 表单：React Hook Form + Zod
- 国际化：i18next + react-i18next（zh/en/fr/ru/ja/vi）
- UI：Base UI 头像/原始组件，本地 `@/components/ui/*` shadcn 风格封装，Tailwind CSS 4
- 主题：自定义 `ThemeCustomizationProvider`
- 图表：@visactor/vchart
- 包管理：Bun

## 2. 路由结构（TanStack file-router）

> 路由文件位于 [web/default/src/routes/](../web/default/src/routes/)，由 router 插件自动生成 `routeTree.gen.ts`。

| 路由 | 组件文件 | 鉴权 | 说明 |
|------|----------|------|------|
| `/` | [routes/index.tsx](../web/default/src/routes/index.tsx) | 公共 | 首页（重定向到登录或仪表盘） |
| `/about` | [routes/about/index.tsx](../web/default/src/routes/about/index.tsx) | 公共 | 关于 |
| `/pricing` | [routes/pricing/index.tsx](../web/default/src/routes/pricing/index.tsx) | 公共 | 价格列表 |
| `/pricing/$modelId` | [routes/pricing/$modelId/index.tsx](../web/default/src/routes/pricing/$modelId/index.tsx) | 公共 | 模型详情 |
| `/rankings` | [routes/rankings/index.tsx](../web/default/src/routes/rankings/index.tsx) | 公共 | 模型排行 |
| `/setup` | [routes/setup/index.tsx](../web/default/src/routes/setup/index.tsx) | 首次安装 | 初始化向导 |
| `/privacy-policy`、`/user-agreement` | [routes/privacy-policy.tsx](../web/default/src/routes/privacy-policy.tsx)、[routes/user-agreement.tsx](../web/default/src/routes/user-agreement.tsx) | 公共 | 法律条款 |
| `/oauth/$provider` | [routes/oauth/$provider.tsx](../web/default/src/routes/oauth/$provider.tsx) | 公共 | OAuth 回调 |
| `(auth)` 路由组 | [routes/(auth)/route.tsx](../web/default/src/routes/(auth)/route.tsx) | 未登录 | 登录/注册/找回/OTP/重置 |
| `(errors)` 路由组 | [routes/(errors)/{401,403,404,500,503}.tsx](../web/default/src/routes/(errors)/) | — | 错误页 |
| `_authenticated` 布局 | [routes/_authenticated/route.tsx](../web/default/src/routes/_authenticated/route.tsx) | UserAuth | 控制台外壳 |
| `/dashboard`、`/dashboard/$section` | [routes/_authenticated/dashboard/](../web/default/src/routes/_authenticated/dashboard/) | UserAuth | 多 tab 仪表盘 |
| `/channels` | [routes/_authenticated/channels/index.tsx](../web/default/src/routes/_authenticated/channels/index.tsx) | AdminAuth | 渠道管理 |
| `/chat/$chatId`、`/chat2link` | [routes/_authenticated/chat/](../web/default/src/routes/_authenticated/chat/) | UserAuth | 内嵌聊天 |
| `/keys` | [routes/_authenticated/keys/index.tsx](../web/default/src/routes/_authenticated/keys/index.tsx) | UserAuth | API Token 管理 |
| `/models`、`/models/$section` | [routes/_authenticated/models/](../web/default/src/routes/_authenticated/models/) | AdminAuth | 模型元数据 |
| `/playground` | [routes/_authenticated/playground/index.tsx](../web/default/src/routes/_authenticated/playground/index.tsx) | UserAuth | 模型试玩 |
| `/profile` | [routes/_authenticated/profile/index.tsx](../web/default/src/routes/_authenticated/profile/index.tsx) | UserAuth | 个人资料/2FA/Passkey |
| `/redemption-codes` | [routes/_authenticated/redemption-codes/index.tsx](../web/default/src/routes/_authenticated/redemption-codes/index.tsx) | AdminAuth | 兑换码 |
| `/subscriptions` | [routes/_authenticated/subscriptions/index.tsx](../web/default/src/routes/_authenticated/subscriptions/index.tsx) | UserAuth | 订阅 |
| `/usage-logs`、`/usage-logs/$section` | [routes/_authenticated/usage-logs/](../web/default/src/routes/_authenticated/usage-logs/) | UserAuth/AdminAuth | 用量日志 |
| `/users` | [routes/_authenticated/users/index.tsx](../web/default/src/routes/_authenticated/users/index.tsx) | AdminAuth | 用户管理 |
| `/wallet` | [routes/_authenticated/wallet/index.tsx](../web/default/src/routes/_authenticated/wallet/index.tsx) | UserAuth | 钱包/充值 |
| `/system-settings/**` | [routes/_authenticated/system-settings/](../web/default/src/routes/_authenticated/system-settings/) | RootAuth | 系统设置（6 大组：auth/billing/content/models/operations/security/site） |
| `/errors/$error` | [routes/_authenticated/errors/$error.tsx](../web/default/src/routes/_authenticated/errors/$error.tsx) | — | 业务错误兜底 |
| `/console/log`、`/console/topup` | [routes/console/](../web/default/src/routes/console/) | UserAuth | 老控制台兼容跳转 |

`__root.tsx` 在每次加载时：
- 通过 `useSystemConfig` 拉取系统配置（Logo、系统名、可见模块开关等）
- 从 query string 抓 `aff` 并写入 `localStorage`（推广码）
- 缓存 setup 状态（`localStorage` 键 `setup_status_checked`）避免重复请求

---

## 3. Feature 目录

每个 feature 自包含 `index.tsx`（页面入口）、`api.ts`（HTTP 客户端）、`components/`（局部组件）、`hooks/`、`lib/`、`types.ts`、`constants.ts`，遵循 [feature-first](https://www.patterns.dev/posts/feature-folders/) 结构。

| Feature | 入口 | 关键子目录 |
|---------|------|-----------|
| `about` | [features/about/index.tsx](../web/default/src/features/about/index.tsx) | api / types |
| `auth` | [features/auth/index.ts](../web/default/src/features/auth/index.ts) | sign-in / sign-up / otp / forgot-password / reset-password-confirm / passkey / secure-verification / oauth / lib / hooks |
| `channels` | [features/channels/index.tsx](../web/default/src/features/channels/index.tsx) | components/{drawers, dialogs}、constants、lib、hooks |
| `chat` | （无 index，纯 hook/lib） | hooks/use-active-chat-key、use-chat-presets、lib/chat-links、send-to-fluent |
| `dashboard` | [features/dashboard/index.tsx](../web/default/src/features/dashboard/index.tsx) | components/{overview, models, users, ui}、section-registry、hooks/use-dashboard-config |
| `errors` | (各 `*-error.tsx`) | general-error / not-found-error |
| `home` | [features/home/index.tsx](../web/default/src/features/home/index.tsx) | — |
| `keys` | [features/keys/index.tsx](../web/default/src/features/keys/index.tsx) | components / api |
| `legal` | [features/legal/index.ts](../web/default/src/features/legal/index.ts) | privacy-policy / user-agreement |
| `models` | [features/models/index.tsx](../web/default/src/features/models/index.tsx) | components / api / types |
| `playground` | [features/playground/index.tsx](../web/default/src/features/playground/index.tsx) | components/(playground-chat, playground-input, message-actions, message-action-button, message-error) |
| `pricing` | [features/pricing/index.tsx](../web/default/src/features/pricing/index.tsx) | components / api |
| `profile` | [features/profile/index.tsx](../web/default/src/features/profile/index.tsx) | 2FA、Passkey 管理 |
| `rankings` | [features/rankings/index.tsx](../web/default/src/features/rankings/index.tsx) | — |
| `redemption-codes` | [features/redemption-codes/index.tsx](../web/default/src/features/redemption-codes/index.tsx) | components / api |
| `setup` | [features/setup/index.ts](../web/default/src/features/setup/index.ts) | api（首次安装） |
| `subscriptions` | [features/subscriptions/index.tsx](../web/default/src/features/subscriptions/index.tsx) | components |
| `system-settings` | [features/system-settings/index.tsx](../web/default/src/features/system-settings/index.tsx) | 多分组：auth, billing, content, general, integrations, maintenance, models, operations, request-limits, security, site；每组带 `section-registry.tsx` 与 `components/` |
| `usage-logs` | [features/usage-logs/index.tsx](../web/default/src/features/usage-logs/index.tsx) | components |
| `users` | [features/users/index.tsx](../web/default/src/features/users/index.tsx) | components / api |
| `wallet` | [features/wallet/index.tsx](../web/default/src/features/wallet/index.tsx) | 充值/订单 |

### 3.1 重点：`features/channels`
大型 feature，含：
- `components/channels-{table,columns,dialogs,primary-buttons,provider}.tsx`
- 多 Key 管理对话框：`dialogs/multi-key-{manage,statistics-card,table-row-actions}-dialog.tsx`
- Codex OAuth 弹窗：`dialogs/codex-oauth-dialog.tsx`、`codex-usage-dialog.tsx`
- Ollama 模型对话框：`dialogs/ollama-models-dialog.tsx`
- 模型映射编辑：`model-mapping-editor.tsx`
- Drawer 分段：`drawers/sections/channel-{advanced,api-access,auth,basic,models,editor-loading-state}-section.tsx`
- 工具函数：`lib/(channel-actions, channel-form, channel-type-config, channel-utils, model-mapping-validation, multi-key-utils, ollama-utils, status-code-risk-guard, upstream-update-utils).ts`

### 3.2 重点：`features/system-settings`
按业务子域分组，每个子域有自己的 `section-registry.tsx`（注册到设置外壳）：
- `auth/`：基本鉴权、机器人保护、OAuth（含 custom-oauth 子模块）、Passkey
- `billing/`：订阅、Plan 管理（registry-only，section 由其他位置 import）
- `content/`：公告、API 信息、Chat 设置（含可视化编辑器）、Dashboard、FAQ、Drawing、Uptime Kuma、JSON Toggle
- `general/`：渠道亲和（含 cache-stats-dialog 与 rule-editor-dialog）、签到、价格、配额、系统行为、系统信息
- `integrations/`：金额折扣（amount-discount）、金额选项、Creem 产品、邮件、IoNet 部署、监控、支付（payment-method/methods、payment-settings、Waffo、Waffo Pancake、Worker）
- `maintenance/`：顶部导航、日志设置、Notice、性能、Sidebar、版本检查
- `models/`：Claude/Gemini/Grok/Global 模型卡、Channel 选择对话框、冲突确认、Group Ratio（表单 + 可视化）、Group special usable、Model Ratio（表单 + 可视化）、Model Pricing Sheet、Ratio Settings Card、Tiered Pricing Editor、Tool Price、Upstream Ratio Sync（含 columns / table）
- `operations/`：聚合页 + registry
- `request-limits/`：速率限制（dialog + visual editor + section）、敏感词、SSRF
- `security/`、`site/`：聚合页 + registry
- 公共 components：`form-dirty-indicator`、`form-navigation-guard`、`settings-accordion/card/form-layout/page/section/page-context`

### 3.3 重点：`features/dashboard`
- `overview/`：announcements-panel、api-info-panel/item、faq-panel、overview-dashboard、performance-health-panel、summary-cards、uptime-panel
- `models/`：consumption-distribution-chart、log-stat-cards、model-charts、models-chart-preferences、models-filter-dialog、performance-overview
- `users/`：user-charts
- `ui/`：panel-wrapper、stat-card
- `hooks/use-dashboard-config.tsx`、`section-registry.tsx`

### 3.4 重点：`features/playground`
极简：`playground-chat`、`playground-input`、`message-actions`、`message-action-button`、`message-error`。底层 AI 元素来自 `components/ai-elements/`（见下）。

---

## 4. 通用组件

### 4.1 UI 原语 [components/ui/](../web/default/src/components/ui/)
shadcn 风格封装（基于 Base UI 与 Radix 思路），共约 55 个：
`accordion, alert-dialog, alert, aspect-ratio, avatar, badge, breadcrumb, button-group, button, calendar, card, carousel, chart, checkbox, collapsible, combobox(-input), command, context-menu, dialog, direction, drawer, dropdown-menu(+test), empty, field, form, hover-card, input-group, input-otp, input, item, kbd, label, markdown, menubar, native-select, navigation-menu, pagination, popover, progress, radio-group, resizable, scroll-area, select, separator, sheet, sidebar, skeleton, slider, sonner, spinner, switch, table, tabs, textarea, titled-card, toggle-group, toggle, tooltip`。

### 4.2 AI 对话元素 [components/ai-elements/](../web/default/src/components/ai-elements/)
聊天/工作流 UI 原子：
`actions, artifact, branch, canvas, chain-of-thought, code-block, confirmation, connection, context, controls, conversation, edge, image, inline-citation, loader, message, node, open-in-chat, panel, plan, prompt-input, queue, reasoning, response, shimmer, sources, suggestion, task, tool, toolbar, web-preview`。

### 4.3 业务通用组件 [components/](../web/default/src/components/)
- 布局：`layout/components/{app-header, app-sidebar, authenticated-layout, chat-presets-item, footer, glow, header(-logo), logo, main, mobile-drawer, mockup, nav-group, nav-link-item, navbar, page-footer, public-header, public-layout, public-navigation, section(-page-layout), sidebar-view-header, system-brand, top-nav}`
- 数据表：`data-table/{bulk-actions, column-header, data-table-page, faceted-filter, mobile-card-list, pagination, table-empty, table-skeleton, toolbar, view-options}`
- 表单 / 弹层：`command-menu, config-drawer, confirm-dialog, json-editor, multi-select, model-group-selector, password-input, profile-dropdown, risk-acknowledgement-dialog, search, sign-out-dialog, tag-input`
- 反馈/状态：`auto-skeleton, coming-soon, empty-state, error-state, loading-state, learn-more, navigation-progress, page-transition, status-badge, table-id, truncated-text, long-text`
- 个性化：`language-switcher, theme-quick-switcher, theme-switch`
- 用户态：`notification-popover, group-badge, masked-value-display, copy-button, date-picker, datetime-picker, animate-in-view, skip-to-main`

---

## 5. Hooks（[hooks/](../web/default/src/hooks/)）

`use-admin, use-copy-to-clipboard, use-countdown, use-debounce, use-dialog, use-hidden-click-unlock, use-media-query, use-minimum-loading-time, use-mobile, use-notifications, use-sidebar-config, use-sidebar-data, use-sidebar-view, use-status, use-system-config, use-table-compact-mode, use-table-url-state, use-top-nav-links, use-user-display`

`use-system-config` 在 `__root.tsx` 自动调用，拉取后端 `/api/status` + `/api/notice` 等，把站点配置写入 `system-config-store`。

---

## 6. 全局状态（[stores/](../web/default/src/stores/)）

Zustand store：`auth-store`、`notification-store`、`system-config-store`。其余视图状态依赖 TanStack Query。

---

## 7. 工具库（[lib/](../web/default/src/lib/)）

`api`（axios 客户端 + interceptor）、`avatar`、`build-metadata`、`colors`、`constants`、`cookies`、`copy-to-clipboard`、`currency`、`dayjs`、`dom-utils`、`format`、`frontend-cache`、`handle-server-error`、`http-status-code-rules`、`motion`、`nav-modules`、`oauth`、`passkey`、`roles`、`secure-verification`、`theme-customization`、`theme-radius`、`time`、`use-chart-theme`、`use-controllable-state`、`utils`、`vchart`

> 国际化、Provider、context 等其余目录见 [web/default/src/i18n/](../web/default/src/i18n/) 与 [web/default/src/context/](../web/default/src/context/)。
