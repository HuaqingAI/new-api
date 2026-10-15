# 多 Part 集成架构 — new-api

> 文档生成时间：2026-05-26
> 4 part：`server` + `web-default` + `web-classic` + `electron`

---

## 1. 集成关系总览

```
                                    ┌────────────────────────┐
                                    │      Electron Shell    │
                                    │ (electron/main.js)     │
                                    │  spawn ─── kill        │
                                    └──────────┬─────────────┘
                                               │ child process
                                               │ env: PORT=3000, SQLITE_PATH
                                               ▼
            ┌───────────────────────────────────────────────────────┐
            │                  Go Backend (server)                  │
            │  ┌─────────────────┐  ┌──────────────────────────┐    │
            │  │  router.SetRouter│ │ go:embed web/{theme}/dist │    │
            │  │  Api/Dashboard/  │ │   ↓ ThemeAssets struct    │    │
            │  │  Relay/Video/Web │ │   ↓ NewThemeAwareFS       │    │
            │  └────────┬─────────┘ └──────────────────────────┘    │
            │           │             ↑                             │
            │           │             │ ❶ build-time embed          │
            │  ┌────────┴────────┐    │                             │
            │  │ /api  /v1  /pg  │    │                             │
            │  │ /mj   /suno     │    │                             │
            │  │ /v1/video       │    │                             │
            │  │ /kling /jimeng  │    │                             │
            │  │ /                │    │                             │
            │  └────────┬────────┘    │                             │
            └──────────┼──────────────┼─────────────────────────────┘
                       │              │
                       │              │ ❷ static serve embed FS
                       │              │   (gzip + cache + theme dispatch)
                       │              │
                       ▼              │
                ┌──────────┐  ┌───────┴──────┐
                │ External │  │   Browser    │
                │ Provider │  │ (浏览器 / WebView)
                │ (40 个)  │  │              │
                └──────────┘  └──────────────┘
                                    ↑
                                    │ ❸ runtime fetch
                                    │   axios → /api  /v1
                                    │
                          ┌─────────┴──────────┐
                          │  web-default (R19) │ or  web-classic (R18)
                          │  Rsbuild bundle    │     Vite bundle
                          └────────────────────┘
```

三类集成关系：
- ❶ **构建期**：Bun 把 `web/default` 与 `web/classic` 打包成 `dist/`，Go 用 `go:embed` 编译进二进制
- ❷ **运行期（同源）**：浏览器或 Electron 直接访问后端端口，前端是后端嵌入的静态资源
- ❸ **运行期（API）**：前端 axios/fetch 调用 `/api/**`、`/v1/**`、`/dashboard/**`、`/v1/video*`、`/mj`、`/suno` 等 — 同源，无 CORS 困扰

---

## 2. 集成点 1：前端 → 后端 HTTP API

### 2.1 通信契约
- 协议：HTTP/1.1 + 部分 SSE（流式响应）+ WebSocket（仅 `/v1/realtime`）
- 同源部署：浏览器直接命中 `https://your-host/`，前后端共享 cookie / session
- 跨源开发：Vite/Rsbuild dev server 启用代理把 `/api`、`/v1` 等转发到 `localhost:3000`
- 鉴权：
  - 控制台 API：cookie session（`gin-contrib/sessions`，`HttpOnly`，`SameSite=Strict`，30 天有效期，密钥 `SESSION_SECRET`）
  - 转发 API：`Authorization: Bearer sk-xxxx`，token 落表 `tokens.key`

### 2.2 前端调用入口
- `web/default`：`src/lib/api.ts`（axios 实例）+ TanStack Query hooks（每个 feature 自带 `api.ts`）
- `web/classic`：`src/helpers/api.js`（axios + 拦截器）+ 各 hook 的封装

### 2.3 后端路由分发（详见 [api-contracts-server.md](api-contracts-server.md)）
- `router.SetRouter`：先装 `Api`、再 `Dashboard`、`Relay`、`Video`、最后 `Web`
- `web` 路由用 `static.Serve("/", themeFS)` + `NoRoute` 兜底（SPA fallback 返回对应 theme 的 `index.html`，但 `/api`、`/v1`、`/assets` 前缀的 404 走 `RelayNotFound`）

### 2.4 前后端字段映射策略
- DTO 单一来源：[dto/](../dto/) 包定义 OpenAI/Claude/Gemini/Image/Audio/Video 的请求响应；前端类型独立维护
- `playground` 路径 `/pg/chat/completions` 接受请求体里携带 `group` 字段（`PlaygroundRequest`）
- 错误体统一：`{error: {message, type}}`，参考 `dto/error.go`

---

## 3. 集成点 2：前端构建产物 → Go embed

### 3.1 构建顺序（CI / Dockerfile）
1. `cd web/default && bun install && bun run build` → `web/default/dist/`
2. `cd web/classic && bun install && bun run build` → `web/classic/dist/`
3. `go build` —— Go 二进制通过 `//go:embed` 把两个 `dist/` 嵌入

### 3.2 Embed 声明（[main.go](../main.go)）
```go
//go:embed web/default/dist
var buildFS embed.FS

//go:embed web/default/dist/index.html
var indexPage []byte

//go:embed web/classic/dist
var classicBuildFS embed.FS

//go:embed web/classic/dist/index.html
var classicIndexPage []byte
```

### 3.3 主题切换（[router/web-router.go](../router/web-router.go)）
- `ThemeAssets` 结构体打包两套资源传入 `SetWebRouter`
- `common.NewThemeAwareFS(defaultFS, classicFS)`：根据 `common.GetTheme()`（DB Option `Theme`）动态返回不同 FS
- SPA fallback 也按主题返回对应 `index.html`
- 全部走 `gzip.DefaultCompression` + `middleware.Cache()` + `GlobalWebRateLimit()`

### 3.4 Analytics 注入
启动时 `InjectUmamiAnalytics()`、`InjectGoogleAnalytics()` 在 `index.html` 的 `<!--umami-->` / `<!--Google Analytics-->` 占位符处插入脚本（两套 index 同时生效）。

### 3.5 国际化资源
- 后端 i18n：`i18n/locales/{en,zh}.json`，go-i18n bundle，`i18n.T(c, key)`
- 前端 i18n：`web/default/src/i18n/locales/{zh,en,fr,ru,ja,vi}.json` 与 `web/classic/src/i18n/locales/`，i18next 加载
- 联动：用户级语言由 `model.GetUserLanguage` 在登录时返回，前端通过 i18next-browser-languagedetector 同步

---

## 4. 集成点 3：Electron → Go 二进制 + WebView

### 4.1 进程模型
```
Electron main process (Node.js)
 ├─ spawn ─→ new-api(.exe)（PORT=3000, SQLITE_PATH=<userData>/data/new-api.db）
 ├─ checkServerAvailability(3000) ──→ Go HTTP server
 ├─ Tray + BrowserWindow
 │   └─ loadURL('http://127.0.0.1:3000')
 └─ contextBridge → renderer (window.electron)
```

### 4.2 资源传递
- 二进制：通过 electron-builder `extraResources` 打到 `process.resourcesPath/bin/`
- 数据目录：`app.getPath('userData')/data/` 通过环境变量传给 Go（`SQLITE_PATH`）
- preload 通过 `process.env.ELECTRON_DATA_DIR` 暴露给前端，便于在桌面态显示"打开数据目录"

### 4.3 桌面态判断
前端通过 `window.electron?.isElectron` 决定显隐桌面专属 UI（如版本号、备份提示）。

### 4.4 关停语义
- 用户点 X：默认隐藏到托盘
- 用户点托盘 → Quit：`app.isQuitting=true` → `before-quit` 截获 → SIGTERM 子进程 → 5 秒后 SIGKILL → `app.exit()`

---

## 5. 集成点 4：后端 → 上游 AI Provider（40 个）

### 5.1 适配器抽象
- [relay/channel/adapter.go](../relay/channel/adapter.go)：定义 `Adaptor` 接口
- 每个 provider 子目录实现 `Init / GetRequestURL / SetupRequestHeader / ConvertRequest / DoRequest / DoResponse / GetModelList / GetChannelName`
- 调度：`controller.Relay(c, types.RelayFormat*)` → `relay.GetAdaptor(channelType)` → 标准转发流水线
- 任务池适配：`service.GetTaskAdaptorFunc = func(platform constant.TaskPlatform) service.TaskPollingAdaptor` 在 `main.go` 注入，避免 service ↔ relay 包循环依赖

### 5.2 选址（[middleware/distributor.go](../middleware/distributor.go)）
按 `(group, model)` 选 Channel，候选关系通过 `model.Ability` 表（复合主键 group+model+channel_id）维护。

支持：
- 显式 `ContextKeyTokenSpecificChannelId`（`/?channel=xxx` 或 token meta）
- Token 模型限制（`tokens.model_limits`）
- Playground 子组（请求体内 `group` 字段）
- `auto` 分组多 group 轮询
- 渠道亲和缓存（用户↔渠道粘性）
- 失败重试与跳过（`service.ShouldSkipRetryAfterChannelAffinityFailure`）

### 5.3 请求重写
- [middleware/kling_adapter.go](../middleware/kling_adapter.go)：可灵 API → 内部统一 video 请求
- [middleware/jimeng_adapter.go](../middleware/jimeng_adapter.go)：火山引擎即梦 API → 内部统一 video 请求

### 5.4 上游元数据缓存
- `model.InitChannelCache()` 启动期把 channel 列表加载到内存
- `go model.SyncChannelCache(common.SyncFrequency)` 周期同步
- `controller.AutomaticallyUpdateChannels(frequency)` 后台批量探活
- `controller.AutomaticallyTestChannels()` 后台测试模型可用性

---

## 6. 集成点 5：后端 → 数据库 / Redis / 持续 profiling

### 6.1 数据库（详见 [data-models-server.md §1](data-models-server.md)）
- 主库：SQL_DSN 切换 SQLite/MySQL/PostgreSQL
- 日志库：`LOG_SQL_DSN` 可独立指向，例如把 `Log` 表落到独立 PG，避免主库 IO 抖动
- GORM v2 自动迁移；跨库列名/布尔常量靠 `commonGroupCol`/`commonKeyCol`/`commonTrueVal`/`commonFalseVal` 抹平差异

### 6.2 Redis
- `common.InitRedisClient()` 在 `InitResources` 中启动
- 用途：Session 兜底（`SESSION_SECRET` 仍是 cookie 加密密钥，但分布式部署时多进程共享缓存）、限流（`GlobalAPIRateLimit`、`ModelRequestRateLimit`）、订阅 plan 缓存（`cachex.HybridCache` 双层）

### 6.3 Pyroscope / pprof
- `common.StartPyroScope()` 推送持续 profiling
- `ENABLE_PPROF=true` 时在 `:8005` 暴露 `net/http/pprof`
- `common.StartSystemMonitor()` 系统资源监控（系统级别，跨平台 Unix/Windows）

### 6.4 后台任务
所有任务以 goroutine 形式在 `main.go` 中启动，主节点专属任务由 `common.IsMasterNode` 控制：
- `model.SyncOptions(freq)`：热加载配置
- `model.UpdateQuotaData()`：聚合用量
- `controller.AutomaticallyUpdateChannels(freq)`：渠道刷新
- `controller.AutomaticallyTestChannels()`：渠道测试
- `service.StartCodexCredentialAutoRefreshTask()`：Codex token 续期（每 10 分钟检查，到期 < 1 天则刷新）
- `service.StartSubscriptionQuotaResetTask()`：订阅配额定时重置
- `controller.StartChannelUpstreamModelUpdateTask()`：上游模型列表同步
- 主节点：`controller.UpdateMidjourneyTaskBulk()` + `UpdateTaskBulk()`

---

## 7. 集成点 6：支付与 Webhook

| 通道 | 入站 webhook | 关键控制器 |
|------|------------|----------|
| Stripe | `POST /api/stripe/webhook` | [topup_stripe.go](../controller/topup_stripe.go)、[subscription_payment_stripe.go](../controller/subscription_payment_stripe.go) |
| Creem | `POST /api/creem/webhook` | [topup_creem.go](../controller/topup_creem.go)、[subscription_payment_creem.go](../controller/subscription_payment_creem.go) |
| Waffo | `POST /api/waffo/webhook` | [topup_waffo.go](../controller/topup_waffo.go)、`waffo-go` SDK |
| Waffo Pancake | `POST /api/waffo-pancake/webhook/:env` | [topup_waffo_pancake.go](../controller/topup_waffo_pancake.go)、`waffo-pancake-sdk-go`，`:env` 用于在 Pancake 后台为 test/prod 各登记一个 webhook |
| 易支付 (epay) | `POST/GET /api/subscription/epay/notify`、`/return` | [subscription_payment_epay.go](../controller/subscription_payment_epay.go) |

所有 webhook 公网无鉴权，必须在 handler 内自行验签（Stripe 用 `Stripe-Signature` header；Creem 用自有 token；Waffo 用 SDK 校验）。

`payment_webhook_availability_test.go` 覆盖支付路由可达性的回归测试。

---

## 8. Theme 与 FRONTEND_BASE_URL 联动

| 部署 | 配置 | 行为 |
|------|------|------|
| 全在线（同源前端） | `FRONTEND_BASE_URL=` | embed FS 内部 SPA 路由 + `Web` router 静态服务 |
| 边缘 + 中心（外部前端） | `FRONTEND_BASE_URL=https://app.example.com` 且非主节点 | 前端静态资源 301 跳转到外部站点（API 仍由本节点处理） |
| 主节点 | `NODE_TYPE=master` | 即使设置了 `FRONTEND_BASE_URL` 也强制忽略，避免循环重定向 |

主题选择：DB `Option.Key='Theme'` 决定 `default` / `classic`，运行时通过 `common.GetTheme()` 动态切换 `NewThemeAwareFS` 行为。

---

## 9. 数据流时序示例：前端发起 chat completion

1. 用户在 `web/default/playground` 输入消息
2. 前端 `axios.post('/v1/chat/completions', body, {headers: {Authorization: 'Bearer sk-xxx'}})`
3. Gin 进入 `middleware`：`PoweredBy → I18n → Logger → CORS → DecompressRequest → Stats → SystemPerformanceCheck`
4. `relay-router` 装 `TokenAuth` → 从 `tokens.key` 查询解析用户 + group + 模型限制 + 渠道指定
5. `Distribute()` 中间件：根据 `(group, model)` 找到 `Ability` 候选 → 应用渠道亲和缓存 → 选出 `Channel`
6. `controller.Relay(c, types.RelayFormatOpenAI)`：拿到 `Adaptor`
7. Adaptor 流水线：
   - `Init(meta)` → `GetRequestURL` → `SetupRequestHeader` → `ConvertRequest`（DTO 透传或转换）→ `DoRequest`（http_client 发送）→ `DoResponse`（流式则 SSE 透传，非流式则解码）
8. `service.PostConsumeQuotaPostBilling`：扣费（含表达式计费 [pkg/billingexpr/](../pkg/billingexpr/) 规则）
9. 写 `Log`（消费类型）+ 异步更新 `QuotaData`
10. 响应回前端，前端 streaming UI 渲染

---

## 10. 测试策略

| Part | 测试 |
|------|------|
| `server` | Go 单测散落各包：`controller/*_test.go`、`service/*_test.go`、`model/*_test.go`、`common/json_test.go`、`pkg/billingexpr/*_test.go`、`relay/channel/api_request_test.go` |
| `web-default` | 见 `components/ui/*_test.tsx`（少量），主要靠 TS 类型 + Rsbuild 校验 |
| `web-classic` | 无独立单测，依赖手测 |
| `electron` | 无单测，依赖人工验证打包 + 启动流程 |

CI 工作流（详见 [deployment-guide.md §8](deployment-guide.md)）：
- `pr-check.yml`：PR 模板与 anti-AI-slop 检查
- `release.yml` / `docker-build.yml`：构建期含 `go build` 自动跑通用 type/编译检查；前端 build 跑 ESLint（默认前端默认禁用，由 `DISABLE_ESLINT_PLUGIN='true'` 控制）

---

## 11. 已知耦合点（重构需注意）

1. **`service` ↔ `relay` 包循环**：通过 `service.GetTaskAdaptorFunc` 函数指针 + `main.go` 注入解耦；新增 task 平台时需同时在 `relay.GetTaskAdaptor` 注册
2. **DTO 重用**：`dto.OpenAI*` 同时承担前端入参 / 上游出参，修改字段时务必检查 Rule 6（指针类型保留 zero value）
3. **跨 theme HTML 注入**：Umami / GA 注入在启动时一次性替换两套 `index.html`，主题切换不会重读，需重启
4. **`go:embed` 强约束**：`web/{theme}/dist/` 必须存在（即使空），否则 `go build` 失败；`Dockerfile.dev` 用占位 HTML 满足约束
5. **多节点共享状态**：必须共享 Redis + DB；单点缓存（`common.OptionMap`、`channelCache`）通过 `SyncFrequency` 收敛
6. **`PORT` 默认 3000 写死**：Electron 也假设 3000，修改 `PORT` 后 Electron 探活将失败
