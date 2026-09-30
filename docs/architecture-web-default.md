# 架构文档 — Web Default (React 19 / Rsbuild / Base UI)

> 项目部分：`web-default`
> 根路径：[web/default/](../web/default/)
> 入口：`web/default/src/main.tsx`
> 文档生成时间：2026-05-26

本文档聚焦 web-default 主题的内部架构。组件清单见 [component-inventory-web-default.md](component-inventory-web-default.md)；与后端的集成契约见 [integration-architecture.md §2-§3](integration-architecture.md)。

---

## 1. 架构风格

**Feature-sliced + 文件路由 + 服务端数据驱动**：

```
src/
├─ main.tsx              入口：QueryClient + Router + Theme + i18n
├─ routes/               TanStack Router 文件路由（自动生成 routeTree.gen.ts）
│   ├─ (auth)/           分组路由：登录/注册
│   └─ _authenticated/   守卫路由：需要登录
├─ features/             业务模块（每个自包含 api/components/hooks/types）
│   ├─ auth / channels / chat / dashboard / errors / home / keys
│   ├─ models / playground / pricing / profile / rankings / redemption-codes
│   ├─ setup / subscriptions / system-settings / usage-logs / users / wallet
├─ components/
│   ├─ ui/               shadcn 风格 UI 原语（55+ 个）
│   ├─ ai-elements/      AI 对话原子（actions/branch/conversation/...）
│   ├─ layout/           app-header / sidebar / footer / public-*
│   └─ data-table/       表格基础设施
├─ stores/               Zustand store（auth / notification / system-config）
├─ hooks/                跨 feature 共享 hook
├─ lib/                  api 客户端 / theme / utils / cache / oauth / time
├─ i18n/                 i18next（zh/en/fr/ru/ja/vi）
├─ context/              少量 React Context
├─ config/               静态配置
└─ styles/               Tailwind 入口与主题变量
```

设计意图：
- **文件路由约定**：路由由 `routes/` 目录结构决定，编译期生成 `routeTree.gen.ts`（无需手写路由表）
- **Feature 自治**：每个 feature 自带 `api.ts` / `components/` / `hooks/` / `types.ts`，跨 feature 引用通过 `lib/` 或 `stores/`
- **服务端状态与客户端状态分离**：TanStack Query 管远端数据 + Zustand 管 UI 状态
- **类型即文档**：TypeScript + Zod schema 在前端独立维护，与后端 DTO 隔离

---

## 2. 技术栈

| 层 | 技术 |
|----|-----|
| 语言 | TypeScript 5.x |
| UI 框架 | React 19 |
| 打包工具 | Rsbuild（基于 Rspack） |
| 包管理 | Bun（CLAUDE.md Rule 3） |
| 路由 | TanStack Router（文件路由 + 类型安全） |
| 数据获取 | TanStack Query（缓存 / 重试 / 乐观更新） |
| 表格 | TanStack Table + TanStack Virtual |
| 状态 | Zustand（slice 模式） |
| 表单 | React Hook Form + Zod |
| UI 原语 | Base UI（@base-ui-components/react）+ shadcn 风格自封装 |
| 样式 | Tailwind CSS + CSS Variables |
| 图标 | Lucide React |
| HTTP | axios |
| i18n | i18next + react-i18next + i18next-browser-languagedetector |
| 富文本 | marked + prismjs + katex + mermaid |
| 可视化 | @visactor/vchart |
| 通知 | sonner |
| 主题 | next-themes |

---

## 3. 路由架构（TanStack Router 文件路由）

### 3.1 路由组织约定

| 路径模式 | 含义 |
|---------|------|
| `routes/foo.tsx` | 普通路由 `/foo` |
| `routes/(group)/foo.tsx` | 分组路由（URL 不含 `group`），用于共享 layout |
| `routes/_authenticated/foo.tsx` | 守卫路由，要求 `beforeLoad` 通过 |
| `routes/__root.tsx` | 根 layout |
| `routes/index.tsx` | `/` 路由 |
| `routes/$param.tsx` | 动态参数路由 |

### 3.2 守卫机制

`_authenticated` 路径下的路由统一在父路由 `beforeLoad` 中检查 `useAuthStore.getState().user`，未登录跳 `/login`。

`(auth)` 分组的登录/注册页则在 `beforeLoad` 中检查"如果已登录则跳 `/dashboard`"。

### 3.3 类型安全

`@tanstack/router-vite-plugin` 自动生成 `src/routeTree.gen.ts`，导出强类型的路由树。调用 `<Link to="/dashboard" />` 时 TS 会校验路径合法性、参数类型。

---

## 4. 数据流：API 调用范式

### 4.1 axios 实例（[src/lib/api.ts](../web/default/src/lib/api.ts)）

- 同源部署：baseURL 为空，直接打 `/api/**` `/v1/**`
- 跨源开发：Rsbuild dev server 代理 `/api` `/v1` `/mj` `/pg` 到 `http://localhost:3000`
- 拦截器：
  - 请求：附加 `Authorization` header（Token 模式）
  - 响应：401 → 跳登录；统一错误提示（sonner）；自动重试（受 TanStack Query 控制）

### 4.2 Feature API 层

每个 feature 自带 `api.ts`，定义：
```ts
export const fooApi = {
  list: (params) => api.get('/api/foo', { params }),
  get: (id) => api.get(`/api/foo/${id}`),
  create: (body) => api.post('/api/foo', body),
  update: (id, body) => api.put(`/api/foo/${id}`, body),
  delete: (id) => api.delete(`/api/foo/${id}`),
}
```

### 4.3 TanStack Query Hooks

每个 feature 的 `hooks/` 封装：
```ts
export const useFooList = (params) =>
  useQuery({
    queryKey: ['foo', 'list', params],
    queryFn: () => fooApi.list(params),
  })

export const useFooCreate = () => {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: fooApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['foo'] }),
  })
}
```

queryKey 约定：`[feature, scope, ...params]`，便于精准 invalidate。

---

## 5. 状态管理

### 5.1 Zustand（客户端状态）

| Store | 关注点 |
|-------|-------|
| `authStore` | 当前登录用户 + role + token，登录/登出 action |
| `notificationStore` | 通知队列与已读状态 |
| `systemConfigStore` | 系统配置（从 `/api/status` + `/api/options` 拉取） |

约定：
- 每个 store 一个文件，导出 `useXxxStore` hook
- 副作用（持久化）通过 `persist` middleware 写 `localStorage`
- 跨 store 协调用 `subscribe` 监听

### 5.2 TanStack Query（服务端状态）

- Cache 默认 `staleTime: 30s`、`gcTime: 5min`
- 全局 `QueryClient` 在 `main.tsx` 创建并注入
- 错误处理统一在 `QueryClientConfig.defaultOptions.queries.onError`

### 5.3 Context（仅极少数）

仅用于不需要订阅的"长生命周期 props"，如 Theme 上下文（由 `next-themes` 提供）。

---

## 6. UI 组件体系

### 6.1 三层组件结构

| 层 | 目录 | 职责 |
|----|------|------|
| 原语 | `components/ui/` | Base UI 包装 + shadcn 风格，无业务（Button/Input/Dialog/Toast/Table 等 55+ 个） |
| 业务原子 | `components/ai-elements/`、`components/data-table/` | 跨 feature 复用（AI 对话块、表格基础设施） |
| Layout | `components/layout/` | app-header / sidebar / footer / public-* |
| Feature | `features/{name}/components/` | 业务专属（不跨 feature 复用） |

### 6.2 样式约定

- Tailwind utility-first，禁止写 inline style
- CSS 变量在 `styles/global.css` 中定义主题色（light / dark）
- shadcn 风格命名：`className={cn("base classes", conditionalClass, props.className)}`
- 复杂状态用 `class-variance-authority` (cva) 抽象 variant

### 6.3 表单约定

- React Hook Form `useForm({ resolver: zodResolver(schema) })`
- Zod schema 与 TS 类型同源（`type FormValues = z.infer<typeof schema>`）
- 提交错误显示在字段级别 + 全局 toast

---

## 7. 国际化

### 7.1 资源组织

- 6 语言：zh / en / fr / ru / ja / vi
- 路径：`src/i18n/locales/{lang}.json`
- 格式：**扁平 JSON，key 为英文源字符串**（无嵌套命名空间）

### 7.2 调用模式

```tsx
const { t } = useTranslation()
return <Button>{t('Save Changes')}</Button>
```

### 7.3 同步工具

`bun run i18n:sync` 扫描 `t('...')` 调用更新到 `locales/*.json`（CLAUDE.md Rule 3）。

### 7.4 语言检测

`i18next-browser-languagedetector` 顺序：`localStorage` → `navigator.language` → `en`。用户登录后由 `model.GetUserLanguage` 返回的语言覆盖。

---

## 8. 构建链路

### 8.1 Rsbuild 配置（[rsbuild.config.ts](../web/default/rsbuild.config.ts)）

- 入口：`src/main.tsx`
- 输出：`dist/`（被 Go `//go:embed` 嵌入）
- Dev server 端口默认 `5173`
- 代理：`/api` `/v1` `/mj` `/pg` `/dashboard` `/v1beta` → `http://localhost:3000`
- 别名：`@/` → `src/`
- 环境变量：`process.env.VITE_*` 通过 `define` 注入
- 构建期注入 `VITE_REACT_APP_VERSION`（CI 从 `VERSION` 文件读取）

### 8.2 Bun 脚本

| 命令 | 用途 |
|------|------|
| `bun install` | 安装依赖 |
| `bun run dev` | 启动 dev server |
| `bun run build` | 生产构建 |
| `bun run preview` | 本地预览生产构建 |
| `bun run i18n:sync` | i18n 资源同步 |
| `bun run lint` | ESLint（默认 `DISABLE_ESLINT_PLUGIN=true` 关闭） |

### 8.3 部署形态

- 构建产物 `dist/` → Go `//go:embed web/default/dist` → 单二进制内置
- 主题切换通过 DB Option `Theme` + `common.NewThemeAwareFS` 动态返回 default 或 classic FS

---

## 9. 桌面态支持

通过 `window.electron?.isElectron` 探测：

```tsx
const isElectron = typeof window !== 'undefined' && (window as any).electron?.isElectron
if (isElectron) {
  const dataDir = (window as any).electron.dataDir
  // 显示"打开数据目录"按钮、版本号、备份提示
}
```

Electron preload 注入字段：`isElectron` / `version` / `platform` / `versions` / `dataDir`（详见 [component-inventory-electron.md §4](component-inventory-electron.md)）。

---

## 10. 关键设计约束

1. **类型安全优先**：所有新模块强制 TS，禁止 `any`。前端类型独立维护，不直接 import 后端 dto（后端字段变更通过 PR 沟通同步）
2. **Feature 边界**：feature 间禁止 import `features/foo/components/Bar`，需要复用的提升到 `components/` 或 `lib/`
3. **路由约定优先于配置**：新增路由优先用文件路由，仅在动态场景下用 `createRoute()`
4. **TanStack Query 是默认数据层**：避免 `useEffect + fetch`；mutation 后必须显式 invalidate
5. **Bun 锁定**：CLAUDE.md Rule 3 要求 web/default 用 Bun，禁止误用 `npm install` 生成 `package-lock.json`
6. **ESLint 默认禁用**：构建期 `DISABLE_ESLINT_PLUGIN=true`，依赖 TS 类型检查 + 人工 review

---

## 11. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 组件清单 | [component-inventory-web-default.md](component-inventory-web-default.md) |
| 跨 part 集成 | [integration-architecture.md](integration-architecture.md) |
| 后端 API 契约 | [api-contracts-server.md](api-contracts-server.md) |
| 部署与运维 | [deployment-guide.md](deployment-guide.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
