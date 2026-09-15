# 开发指南 — Web Classic (React 18 / Vite / Semi Design)

> 项目部分：`web-classic`
> 路径：[web/classic/](../web/classic/)
> 文档生成时间：2026-05-26

---

## 1. 环境准备

| 工具 | 备注 |
|------|------|
| Bun（推荐） / npm / yarn | 任一即可，优先 Bun |
| Node.js | 20+ |
| Go | 1.25.1+（仅联调时需要） |

> CLAUDE.md Rule 3 主要约束 default 主题用 Bun，classic 主题对包管理器更宽松，但不要混用导致 lockfile 冲突。

---

## 2. 启动开发

### 2.1 完整环境

```bash
# 终端 1：起后端依赖
docker compose -f docker-compose.dev.yml up -d

# 终端 2：起 Go 后端
go run main.go

# 终端 3：起前端
cd web/classic
bun install
bun run dev   # 监听 http://localhost:5174
```

`bun run dev` 启动 Vite dev server，代理 `/api`、`/v1`、`/mj`、`/pg` 到 `http://localhost:3000`。

### 2.2 让后端使用 classic 主题

进入控制台 → 系统设置 → 主题 → 选择 `classic`。

或直接改 DB：

```sql
UPDATE options SET value = 'classic' WHERE "key" = 'Theme';   -- PG
-- 或
UPDATE options SET value = 'classic' WHERE `key` = 'Theme';   -- MySQL/SQLite
```

重启后端生效。

---

## 3. 项目结构

详见 [architecture-web-classic.md](architecture-web-classic.md) 与 [component-inventory-web-classic.md](component-inventory-web-classic.md)。核心：

```
src/
├─ main.jsx              入口
├─ App.jsx               集中式路由（react-router-dom v6）
├─ pages/{Domain}/       顶级页面
├─ pages/Setting/        设置中心多 Tab 子页
├─ components/           按业务域分目录
├─ hooks/{domain}/       业务 hooks
├─ helpers/              工具函数（api/auth/render/...）
├─ context/              Status / Theme / User（Context + reducer）
├─ services/             业务服务（secureVerification 等）
└─ i18n/                 i18next（zh/en/fr/ru/ja/vi）
```

---

## 4. 编码规范

### 4.1 路由

- 所有路由集中在 `App.jsx` 声明
- 大页面用 `React.lazy()` 包装
- 守卫通过 `<PrivateRoute>` / `<AdminRoute>` / `<AuthRedirect>` 包裹

### 4.2 数据获取

- 默认用 axios 直调 + 局部 `useState/useEffect`
- 跨组件复用的数据获取抽到 `hooks/{domain}/use{Domain}Data.js`
- 二次验证场景必须用 [helpers/secureApiCall.js](../web/classic/src/helpers/secureApiCall.js)，不要绕过 `SecureVerificationModal`

### 4.3 状态管理

- 全局态用 Context + Reducer：`Status` / `Theme` / `User`
- 局部态用 `useState`
- 跨多个组件但不跨页面 → props 提升或自定义 hook
- 不引入 Zustand / Redux / Recoil 等独立状态库

### 4.4 UI

- 主样式系统：**Semi Design（@douyinfe/semi-ui）**，不混入 Tailwind 等
- 自封装组件在 `components/common/ui/`：`CardPro` / `CardTable` / `JSONEditor` / `ScrollableContainer` 等
- 表格优先用 Semi Table；卡片列表用 `CardTable` / `CardPro`
- 紧凑模式：`useTableCompactMode` hook + `CompactModeToggle`

### 4.5 表单

- 用 Semi Form：`<Form>` + `<Form.Input>` / `<Form.Select>` 等
- 校验通过 `rules` prop
- 提交：`formApi.validate().then(values => ...)`

### 4.6 i18n

- key 风格历史上多为**中文源字符串**，与 default 主题（英文 key）不一致
- 新增 key 时遵循当前域已有约定，不要混用
- 调用：`const { t } = useTranslation(); t('保存')`

---

## 5. 常见任务

### 5.1 新增一个页面

1. 在 `pages/{Domain}/index.jsx` 创建页面组件
2. 在 `App.jsx` 注册路由：

```jsx
const Foo = lazy(() => import('./pages/Foo'))

<Route path="/foo" element={
  <PrivateRoute><Foo /></PrivateRoute>
} />
```

3. 如需在侧边栏显示，去 `components/layout/SiderBar.jsx` 加菜单项

### 5.2 新增一个表格

1. 在 `components/table/{domain}/` 创建子目录，包含：
   - `Table.jsx`（主体）
   - `ColumnDefs.jsx`（列定义）
   - `Filters.jsx`（筛选器）
   - `Actions.jsx`（行操作）
   - `Tabs.jsx`（多 Tab，可选）
   - `modals/`（弹窗）
2. 在 `hooks/{domain}/use{Domain}Data.js` 写数据加载 hook
3. 在页面中调用：

```jsx
import {{Domain}}Table from '@/components/table/{domain}/Table'
const { data, loading, refresh } = use{Domain}Data()
return <{Domain}Table data={data} loading={loading} onRefresh={refresh} />
```

### 5.3 新增一个 Setting 子页

1. 在 `pages/Setting/{Domain}/{Section}.jsx` 创建组件
2. 在 `pages/Setting/index.jsx` 加 Tab：

```jsx
<TabPane tab="新设置" itemKey="new-settings">
  <NewSettings />
</TabPane>
```

### 5.4 接入二次验证

```jsx
import { secureApiCall } from '@/helpers/secureApiCall'

const handleSensitive = async () => {
  await secureApiCall(
    () => api.delete(`/api/dangerous/${id}`),
    { actionLabel: '删除危险资源' }
  )
}
```

`secureApiCall` 会触发 `SecureVerificationModal`，验证通过后才执行真实请求。

---

## 6. 命令清单

| 命令 | 用途 |
|------|------|
| `bun install` | 安装依赖 |
| `bun run dev` | 启动 dev server（5174） |
| `bun run build` | 生产构建 → `dist/` |
| `bun run preview` | 预览生产构建 |

---

## 7. 调试

### 7.1 React DevTools

浏览器扩展，常规用法。

### 7.2 Semi Design 调试

Semi 组件通过 `data-component` 属性暴露内部结构，开发者工具中可定位。

### 7.3 网络代理

代理配置在 `vite.config.js`。改完需重启 dev server。

---

## 8. 构建与部署

### 8.1 本地构建

```bash
cd web/classic
bun install
bun run build   # → dist/
```

### 8.2 部署形态

不独立部署。前端通过 Go 二进制 `//go:embed web/classic/dist` 嵌入。

### 8.3 主题切换

由后端 DB Option `Theme` 决定运行期返回 default 还是 classic 资源。前端不参与决策。

---

## 9. 与 Default 主题的协作

两套主题共用同一份后端 API。修改后端字段时建议同时更新两套前端：

| 文件 | Default | Classic |
|------|---------|---------|
| API 调用 | `web/default/src/features/{name}/api.ts` | `web/classic/src/helpers/api.js` |
| 类型 | `web/default/src/features/{name}/types.ts` | （JS，无类型） |
| Hook | `web/default/src/features/{name}/hooks/` | `web/classic/src/hooks/{domain}/` |
| 组件 | `web/default/src/features/{name}/components/` | `web/classic/src/components/{domain}/` |
| 页面 | `web/default/src/routes/_authenticated/...` | `web/classic/src/pages/{Domain}/` |

---

## 10. 提交前自检

- [ ] `bun run build` 成功
- [ ] 用户可见字符串都走 `t()`
- [ ] 二次验证场景使用 `secureApiCall`
- [ ] 没有混入 Tailwind 等其他样式系统
- [ ] 守卫包装正确（`PrivateRoute` / `AdminRoute`）
- [ ] 大页面用 `lazy()` 加载
- [ ] 跨主题字段变更已同步到 default 主题

---

## 11. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 架构详解 | [architecture-web-classic.md](architecture-web-classic.md) |
| 组件清单 | [component-inventory-web-classic.md](component-inventory-web-classic.md) |
| Default 主题对照 | [architecture-web-default.md](architecture-web-default.md) §11、[development-guide-web-default.md](development-guide-web-default.md) |
| 跨 part 集成 | [integration-architecture.md](integration-architecture.md) |
| 后端 API 契约 | [api-contracts-server.md](api-contracts-server.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
| 提 PR | [contribution-guide.md](contribution-guide.md) |
