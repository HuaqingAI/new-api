# 架构文档 — Web Classic (React 18 / Vite / Semi Design)

> 项目部分：`web-classic`
> 根路径：[web/classic/](../web/classic/)
> 入口：`web/classic/src/main.jsx`、`web/classic/src/App.jsx`
> 文档生成时间：2026-05-26

本文档聚焦 web-classic 主题的内部架构。组件清单见 [component-inventory-web-classic.md](component-inventory-web-classic.md)；与后端的集成契约见 [integration-architecture.md §2-§3](integration-architecture.md)。

---

## 1. 架构风格

**经典 SPA + 集中式路由 + Context/Reducer**：

```
src/
├─ main.jsx               React 18 入口
├─ App.jsx                react-router-dom 集中式路由（lazy + 守卫）
├─ pages/                 顶级页面（与路由一一对应）
│   ├─ {Domain}/index.jsx
│   └─ Setting/           设置中心（多 Tab 子页面）
├─ components/            组件树（按业务域分目录）
│   ├─ auth/ common/ dashboard/ layout/headerbar/
│   ├─ model-deployments/ playground/ settings/personal/
│   ├─ setup/ table/{domain}/
├─ hooks/                 hooks（按业务域分组）
├─ helpers/               工具函数（api/auth/render/...）
├─ context/               Status / Theme / User（Context + reducer）
├─ services/              业务服务（secureVerification 等）
├─ i18n/                  i18next（zh/en/fr/ru/ja/vi）
├─ assets/                静态资源
└─ pages/Setting/*        设置子页面
```

设计意图：
- **集中式路由**：所有路由声明在 `App.jsx`，便于一目了然地审计权限
- **页面/组件二分**：`pages/` 装顶级页面（路由目标），`components/` 装可复用块
- **Context + Reducer 替代独立状态库**：简化心智模型，避免引入额外依赖
- **Semi Design 包装**：UI 完全基于字节跳动 Semi Design，组件库直接复用，少量自封装

---

## 2. 技术栈

| 层 | 技术 |
|----|-----|
| 语言 | JavaScript（少量 TS） |
| UI 框架 | React 18.2 |
| 打包工具 | Vite 5.2 |
| 包管理 | Bun / npm（CLAUDE.md Rule 3 主要约束 default 主题） |
| 路由 | react-router-dom v6（声明式 + lazy） |
| 数据获取 | axios（封装在 `helpers/api.js`） |
| 表格 | Semi Table + 自封装 `CardTable` / `CardPro` |
| 状态 | React Context + useReducer |
| 表单 | Semi Form |
| UI 库 | @douyinfe/semi-ui ^2.69 + @douyinfe/semi-icons |
| 样式 | Semi 内置 theme + 局部 CSS |
| HTTP | axios（含拦截器） |
| i18n | i18next + react-i18next 13 + i18next-browser-languagedetector |
| 富文本 | marked + mermaid + katex + prismjs |
| 可视化 | @visactor/vchart |
| 通知 | react-toastify |

---

## 3. 路由架构（[App.jsx](../web/classic/src/App.jsx)）

### 3.1 守卫组件

| 守卫 | 行为 |
|------|------|
| `AuthRedirect` | 未登录 → 跳 `/login` |
| `PrivateRoute` | 包装需登录的页面 |
| `AdminRoute` | 包装需管理员角色的页面（基于 `users.role`） |

### 3.2 路由模式

- 所有路由在 `App.jsx` 单文件声明，`Routes/Route` 嵌套
- 大页面用 `React.lazy()` + `Suspense` 分包加载
- 守卫通过 `element={<PrivateRoute><User /></PrivateRoute>}` 包裹

### 3.3 主要路由表

详见 [component-inventory-web-classic.md §2](component-inventory-web-classic.md)。覆盖：
- 公共：`/` `/about` `/pricing` `/user-agreement` `/privacy-policy`
- 鉴权：`/login` `/register` `/reset` `/passwordReset` `/oauth/:provider` `/setup`
- 用户：`/dashboard` `/token` `/topup` `/log` `/midjourney` `/task` `/chat` `/chat2link` `/playground` `/subscription`
- 管理：`/user` `/channel` `/redemption` `/model` `/model-deployment`
- 设置：`/setting`（多 Tab 容器）

---

## 4. 数据流：API 调用范式

### 4.1 axios 实例（[helpers/api.js](../web/classic/src/helpers/api.js)）

- 同源部署：baseURL 空，打 `/api/**` `/v1/**`
- 跨源开发：Vite dev server 代理 `/api` `/v1` `/mj` `/pg` 到 `http://localhost:3000`
- 拦截器：
  - 请求：附加 cookie / Authorization
  - 响应：401 → 调用 `User` context 清空用户态，跳 `/login`
  - 全局错误：`react-toastify` 弹 toast

### 4.2 业务调用模式

无独立 hooks 抽象层（与 default 主题不同），页面/组件中直接调用 axios：

```jsx
const fetchData = async () => {
  setLoading(true)
  try {
    const res = await api.get('/api/foo')
    setData(res.data.data)
  } catch (e) {
    showError(e.message)
  } finally {
    setLoading(false)
  }
}

useEffect(() => { fetchData() }, [])
```

部分领域已抽到 `hooks/{domain}/use*Data.js` 中（如 `useChannelsData`、`useDashboardData`），但未全面铺开。

### 4.3 二次验证

敏感操作通过 [helpers/secureApiCall.js](../web/classic/src/helpers/secureApiCall.js) 包装：触发 `SecureVerificationModal` → 验证通过后才执行真实请求。

---

## 5. 状态管理（Context + Reducer）

### 5.1 三个全局 Context

| Context | 文件 | 关注点 |
|---------|------|-------|
| `Status` | [context/Status/](../web/classic/src/context/Status/) | 系统配置、公告、登录态注释（reducer 模式） |
| `Theme` | [context/Theme/](../web/classic/src/context/Theme/) | 亮/暗 + Semi 主题切换 |
| `User` | [context/User/](../web/classic/src/context/User/) | 当前用户信息与角色（reducer 模式） |

### 5.2 模式

```jsx
// reducer.js
export const reducer = (state, action) => {
  switch (action.type) {
    case 'login': return { ...state, user: action.user }
    case 'logout': return { ...state, user: null }
    default: return state
  }
}

// index.jsx
export const UserProvider = ({ children }) => {
  const [state, dispatch] = useReducer(reducer, initialState)
  return <UserContext.Provider value={[state, dispatch]}>{children}</UserContext.Provider>
}
```

### 5.3 局部状态

页面级状态用 `useState`；跨多个组件但不跨页面的状态用 props 提升或自定义 hooks。

---

## 6. UI 组件体系

### 6.1 组件分类

| 子目录 | 用途 |
|-------|------|
| `components/auth/` | 登录/注册/OAuth 回调/密码重置/2FA |
| `components/common/` | DocumentRenderer / ErrorBoundary / markdown / modals / ui（CardPro/CardTable/JSONEditor/Loading 等） |
| `components/dashboard/` | 仪表盘 panel（公告/API 信息/图表/统计/Uptime） |
| `components/layout/` | Footer / PageLayout / SetupCheck / SiderBar / HeaderBar |
| `components/layout/headerbar/` | 头部子模块（语言切换/通知/主题切换/用户菜单/移动菜单） |
| `components/playground/` | 试玩组件（聊天区/代码查看/参数控制/SSE Viewer 等） |
| `components/settings/` | 设置中心组件（按业务域） |
| `components/settings/personal/` | 个人设置（账户管理/2FA/通知设置/绑定 modal） |
| `components/setup/` | 安装向导（StepNavigation + steps） |
| `components/table/{domain}/` | 各业务域表格（channels/mj-logs/models/redemptions/subscriptions/task-logs/model-deployments/model-pricing），每个含 Actions/ColumnDefs/Filters/Table/Tabs/modals |

### 6.2 表格基础设施

- 默认：Semi Table（声明式列定义）
- 自封装：`CardTable`（卡片式列表）、`CardPro`（增强卡片）
- 紧凑模式：`CompactModeToggle` + `useTableCompactMode` hook

### 6.3 文档/Markdown 渲染

- `components/common/DocumentRenderer/`：通用文档渲染
- `components/common/markdown/MarkdownRenderer.jsx`：marked + prismjs + katex + mermaid 集成

---

## 7. Hooks 组织

按业务域分组（详见 [component-inventory-web-classic.md §5](component-inventory-web-classic.md)）：

| 域 | 文件示例 |
|----|---------|
| `channels/` | `useChannelsData`、`useChannelUpstreamUpdates`、`upstreamUpdateUtils` |
| `dashboard/` | `useDashboardCharts`、`useDashboardData`、`useDashboardStats` |
| `playground/` | `useApiRequest`、`useDataLoader`、`useMessageActions`、`useMessageEdit`、`usePlaygroundState` |
| `common/` | `useContainerWidth`、`useHeaderBar`、`useIsMobile`、`useNotifications`、`useSecureVerification`、`useSidebar`、`useUserPermissions` |
| 其余 | `mj-logs/`、`model-deployments/`、`model-pricing/`、`models/`、`redemptions/`、`subscriptions/`、`task-logs/`、`tokens/`、`usage-logs/`、`users/`、`chat/` |

每域通常提供 `use{Domain}Data`（数据加载 + 过滤 + 分页 + 删改）+ 场景专用 hook。

---

## 8. 国际化

### 8.1 资源组织

- 6 语言：与 default 主题对齐（zh/en/fr/ru/ja/vi）
- 路径：`web/classic/src/i18n/locales/{lang}.json`
- 扁平 JSON 格式

### 8.2 调用模式

```jsx
import { useTranslation } from 'react-i18next'
const { t } = useTranslation()
return <Button>{t('保存')}</Button>
```

注意：classic 主题历史上 key 多为中文源字符串，与 default 主题（英文 key）不一致；新增 key 时建议遵循当前域已有约定。

---

## 9. 构建链路

### 9.1 Vite 配置（[vite.config.js](../web/classic/vite.config.js)）

- 入口：`src/main.jsx`
- 输出：`dist/`（被 Go `//go:embed` 嵌入）
- Dev server 端口默认 `5174`（避免与 default 5173 冲突）
- 代理：`/api` `/v1` `/mj` `/pg` → `http://localhost:3000`
- 别名：`@/` → `src/`

### 9.2 脚本

| 命令 | 用途 |
|------|------|
| `bun install` / `npm install` | 安装依赖 |
| `bun run dev` / `npm run dev` | 启动 dev server |
| `bun run build` / `npm run build` | 生产构建 |
| `bun run preview` | 本地预览 |

### 9.3 部署形态

- 构建产物 → Go `//go:embed web/classic/dist` → 嵌入单二进制
- 主题选择：DB Option `Theme=classic` 通过 `NewThemeAwareFS` 切换

---

## 10. 桌面态支持

与 default 主题相同，通过 `window.electron?.isElectron` 探测 Electron 环境，启用桌面专属 UI（数据目录路径、备份提示等）。

---

## 11. 与 Default 主题的关键差异

| 维度 | Classic | Default |
|------|---------|---------|
| React | 18.2 | 19 |
| 打包 | Vite 5 | Rsbuild |
| 路由 | react-router-dom v6（声明式） | TanStack Router（文件路由） |
| 状态 | Context + useReducer | Zustand + TanStack Query |
| 类型 | JavaScript（少量 TS） | TypeScript 全量 |
| UI 库 | Semi Design | Base UI + Tailwind + shadcn 风格 |
| 表单 | Semi Form | React Hook Form + Zod |
| 表格 | Semi Table + CardTable/CardPro | TanStack Table + Virtual |
| 数据层 | axios 直调 + 局部 hooks | TanStack Query 全量 + feature API 抽象 |
| i18n key | 中文源字符串 | 英文源字符串 |

**两套主题共用同一套后端 API**，差异仅在前端实现栈与设计语言。

---

## 12. 关键设计约束

1. **保留 Semi Design 作为主样式系统**：不要混入 Tailwind 等其他样式方案
2. **页面/组件二分**：新增页面放 `pages/`，可复用块放 `components/`
3. **守卫通过 `App.jsx` 集中声明**：不要在页面内做权限判断
4. **二次验证强制**：敏感操作必须走 `secureApiCall.js`，不要直接绕过 `SecureVerificationModal`
5. **lazy 加载优先**：新页面默认 `React.lazy()`，避免首屏体积膨胀
6. **i18n key 风格保持一致**：在某个域内不要混用中英文 key

---

## 13. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 组件清单 | [component-inventory-web-classic.md](component-inventory-web-classic.md) |
| Default 主题对照 | [architecture-web-default.md](architecture-web-default.md) |
| 跨 part 集成 | [integration-architecture.md](integration-architecture.md) |
| 后端 API 契约 | [api-contracts-server.md](api-contracts-server.md) |
| 部署与运维 | [deployment-guide.md](deployment-guide.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
