# 开发指南 — Web Default (React 19 / Rsbuild)

> 项目部分：`web-default`
> 路径：[web/default/](../web/default/)
> 文档生成时间：2026-05-26
> 包管理：**Bun**（CLAUDE.md Rule 3）

---

## 1. 环境准备

| 工具 | 最低版本 |
|------|---------|
| Bun | 任意稳定版（`bun --version`） |
| Node.js | 20+（仅用于个别工具的兼容性） |
| Go | 1.25.1+（仅当需要联调真后端） |

不要使用 `npm install` / `yarn` / `pnpm`，避免生成多个 lockfile。

---

## 2. 启动开发

### 2.1 完整开发环境（前后端联调）

```bash
# 终端 1：起后端依赖（PG + Redis）
docker compose -f docker-compose.dev.yml up -d

# 终端 2：起 Go 后端
go run main.go

# 终端 3：起前端
cd web/default
bun install
bun run dev   # 监听 http://localhost:5173
```

`bun run dev` 启动 Rsbuild dev server，自动代理 `/api`、`/v1`、`/mj`、`/pg`、`/v1beta`、`/dashboard` 到 `http://localhost:3000`。

### 2.2 仅前端（mock 后端）

后端不可用时无法验证业务流，但可以做纯样式调整：

```bash
cd web/default
bun install
bun run dev
```

页面会因 API 失败显示错误，但 UI 渲染不受影响。

---

## 3. 项目结构

详见 [architecture-web-default.md](architecture-web-default.md) 与 [component-inventory-web-default.md](component-inventory-web-default.md)。核心：

```
src/
├─ main.tsx               入口
├─ routes/                文件路由（自动生成 routeTree.gen.ts）
├─ features/{name}/       业务模块（自包含 api/components/hooks/types）
├─ components/ui/         UI 原语（55+ 个 shadcn 风格）
├─ stores/                Zustand store
├─ hooks/                 跨 feature 共享 hook
├─ lib/                   工具
└─ i18n/                  i18next（zh/en/fr/ru/ja/vi）
```

---

## 4. 编码规范

### 4.1 类型安全

- **TS 强制**：禁止 `any`，需要时用 `unknown` + 类型守卫
- **Zod schema 与 TS 类型同源**：`type FormValues = z.infer<typeof schema>`
- **不直接 import 后端 dto**：前端类型独立维护，字段变更通过 PR 沟通

### 4.2 路由

- 优先使用文件路由：在 `routes/` 下创建 `.tsx` 文件
- 守卫路由：放 `_authenticated/` 子目录
- 分组路由：用 `(group)/`（URL 不含 group 段）
- `<Link to="..." />` 必须写完整路径，否则 TS 报错

### 4.3 数据获取

- **必须用 TanStack Query**，禁止 `useEffect + fetch`
- queryKey 约定：`[feature, scope, ...params]`
- mutation 后必须显式 `qc.invalidateQueries({ queryKey: [...] })`
- 错误处理放 `QueryClientConfig.defaultOptions.queries.onError`

### 4.4 状态管理

- 客户端状态用 Zustand
- 服务端状态用 TanStack Query
- 不再用 React Context 做应用状态（仅 Theme 等长生命周期 props）
- Store 拆分：每个领域一个文件，导出 `useXxxStore` hook

### 4.5 UI

- 三层结构：原语（`components/ui/`）→ 业务原子（`components/{ai-elements,data-table}/`）→ Feature 组件（`features/{name}/components/`）
- Feature 间禁止 cross-import：复用提升到 `components/` 或 `lib/`
- 样式 utility-first：`className={cn("base", conditional, props.className)}`
- 复杂 variant 用 `class-variance-authority` (cva)

### 4.6 表单

- React Hook Form + Zod
- `useForm({ resolver: zodResolver(schema) })`
- 字段错误显示在控件下方
- 全局错误用 sonner toast

### 4.7 i18n

- 所有用户可见字符串走 `t()` 调用
- key 用 **英文源字符串**（与 default 主题约定一致）
- 同步：`bun run i18n:sync`
- 资源文件：`src/i18n/locales/{zh,en,fr,ru,ja,vi}.json`

---

## 5. 常见任务

### 5.1 新增一个 Feature

```
features/billing/
├─ api.ts          axios 调用封装
├─ components/     业务组件
├─ hooks/          TanStack Query hooks
├─ types.ts        TS 类型 + Zod schema
└─ index.ts        统一导出
```

然后在 `routes/_authenticated/billing.tsx` 注册路由。

### 5.2 新增一个路由

在 `routes/` 下创建文件，TanStack Router 会自动检测并写入 `routeTree.gen.ts`：

```tsx
// routes/_authenticated/foo/$id.tsx
import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_authenticated/foo/$id')({
  component: FooDetail,
  loader: ({ params }) => fooApi.get(params.id),
})

function FooDetail() {
  const { id } = Route.useParams()
  return <div>Foo {id}</div>
}
```

### 5.3 新增一个 API hook

```ts
// features/foo/hooks/useFooList.ts
import { useQuery } from '@tanstack/react-query'
import { fooApi } from '../api'

export const useFooList = (params: ListParams) =>
  useQuery({
    queryKey: ['foo', 'list', params],
    queryFn: () => fooApi.list(params),
  })
```

### 5.4 新增一个 UI 原语

参考 `components/ui/` 内现有组件（如 `button.tsx`），用 Base UI 包装 + cva variants。

---

## 6. 命令清单

| 命令 | 用途 |
|------|------|
| `bun install` | 安装依赖 |
| `bun run dev` | 启动 dev server（5173） |
| `bun run build` | 生产构建 → `dist/` |
| `bun run preview` | 预览生产构建 |
| `bun run i18n:sync` | i18n 资源同步 |
| `bun run lint` | ESLint（默认禁用） |
| `bun run type-check` | TypeScript 类型检查 |

---

## 7. 调试

### 7.1 React DevTools

浏览器扩展，常规用法。

### 7.2 TanStack Router DevTools

dev 环境下默认启用（在 `main.tsx` 中根据 `process.env.NODE_ENV` 条件渲染）。可视化路由树、loaders、search params。

### 7.3 TanStack Query DevTools

dev 环境下默认启用，悬浮按钮在右下。可查看 query 缓存、状态、refetch。

### 7.4 网络代理调试

dev server 代理在 `rsbuild.config.ts` 中。改完代理需要重启 dev server。

---

## 8. 构建与部署

### 8.1 本地构建

```bash
cd web/default
bun install
bun run build   # → dist/
```

构建产物 `dist/` 会被 Go `//go:embed web/default/dist` 嵌入二进制。

### 8.2 部署形态

不独立部署。前端必须通过 Go 二进制提供（同源 `/`）。详见 [integration-architecture.md §3](integration-architecture.md)。

### 8.3 主题切换

DB Option `Theme=default` → 后端返回 default 主题；`Theme=classic` → 返回 classic 主题。前端不参与切换决策。

---

## 9. 提交前自检

- [ ] `bun run build` 成功
- [ ] `bun run type-check` 无错误
- [ ] 新增 i18n key 已 `bun run i18n:sync`
- [ ] 用户可见字符串都走 `t()`
- [ ] 数据获取用 TanStack Query 而非 `useEffect + fetch`
- [ ] mutation 后已 invalidate queries
- [ ] 没有 cross-feature import
- [ ] 没有 `any` 滥用
- [ ] 桌面态（Electron）相关代码用 `window.electron?.isElectron` 探测

---

## 10. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 架构详解 | [architecture-web-default.md](architecture-web-default.md) |
| 组件清单 | [component-inventory-web-default.md](component-inventory-web-default.md) |
| 跨 part 集成 | [integration-architecture.md](integration-architecture.md) |
| 后端 API 契约 | [api-contracts-server.md](api-contracts-server.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
| 提 PR | [contribution-guide.md](contribution-guide.md) |
