# 架构文档 — Server (Go 后端)

> 项目部分：`server`
> 根路径：仓库根 + `controller/` `service/` `model/` `relay/` `router/` `middleware/` `setting/` `common/` `dto/` `i18n/` `oauth/` `pkg/` `types/` `constant/`
> 入口：[main.go](../main.go)
> 文档生成时间：2026-05-26

本文档聚焦 **server 部分内部设计**。跨 part 的集成契约见 [integration-architecture.md](integration-architecture.md)；HTTP 端点契约见 [api-contracts-server.md](api-contracts-server.md)；数据模型见 [data-models-server.md](data-models-server.md)；源码结构见 [source-tree-analysis.md §2](source-tree-analysis.md)。

---

## 1. 架构风格

经典分层 + 适配器模式：

```
HTTP Request
    │
    ▼
┌──────────────┐   middleware/  Gin 中间件链
│  Middleware  │   (recover, request-id, cors, logger, auth,
│  Chain       │    distributor, rate-limit, stats, ...)
└──────┬───────┘
       │
       ▼
┌──────────────┐   router/  路由装载（按业务域分文件）
│   Router     │   /api  /dashboard  /v1  /pg  /mj  /suno  /v1/video  /
└──────┬───────┘
       │
       ▼
┌──────────────┐   controller/  HTTP handler（参数解析 + 响应）
│  Controller  │
└──────┬───────┘
       │
       ▼
┌──────────────┐   service/  业务逻辑（计费、选址、Token、邮件、Webhook 等）
│   Service    │
└──────┬───────┘
       │
       ├──────────────► relay/channel/  上游适配器（40 个 provider）
       │
       ▼
┌──────────────┐   model/  GORM 模型 + DB 抽象
│    Model     │
└──────┬───────┘
       │
       ▼
   DB (SQLite / MySQL / PostgreSQL) + Redis
```

设计意图：
- **关注点分离**：HTTP 框架细节（Gin）只出现在 `controller` / `middleware` / `router`，业务逻辑下沉到 `service`
- **多 DB 抽象**：跨 SQLite / MySQL / PostgreSQL 由 `model/main.go` 中的 `commonGroupCol` / `commonKeyCol` / `commonTrueVal` / `commonFalseVal` 与 `common.UsingXXX` 布尔标志屏蔽方言差异（详见 CLAUDE.md Rule 2）
- **40 个上游适配器统一接口**：`relay/channel/adapter.go::Adaptor` 接口，运行期由 `relay.GetAdaptor(channelType)` 工厂选择

---

## 2. 启动序列（[main.go](../main.go)）

`main.go` 的 `main()` 严格遵循如下顺序：

1. `common.SetupLogger()`：日志初始化
2. `common.InitEnv()`：解析环境变量到 `common.*` 全局
3. `model.InitDB()`：主库连接 + GORM 自动迁移
4. `model.InitLogDB()`：日志库连接（可独立 `LOG_SQL_DSN`）
5. `common.InitRedisClient()`：Redis 客户端 + Sentinel/Cluster 兼容
6. `i18n.Init()`：加载 go-i18n bundle（en/zh）
7. `setting.InitOptionMap()`：从 DB Option 表回填运行期配置
8. `common.InitLimiter()`：Redis 限流器初始化
9. `setting.InitAllSettings()`：初始化所有 setting 子包（ratio/model/system/billing 等）默认值
10. `service.GetTaskAdaptorFunc = relay.GetTaskAdaptor`：函数指针注入，打破 `service ↔ relay` 包循环
11. `controller.InitMemoryCache()` + `model.InitChannelCache()`：内存缓存预热
12. 静态资源注入：`InjectUmamiAnalytics()` / `InjectGoogleAnalytics()` 在 embed FS 的 `index.html` 占位符中替换
13. 启动后台 goroutine：
    - `model.SyncOptions(SyncFrequency)`：配置热加载
    - `model.UpdateQuotaData()`：聚合用量
    - `controller.AutomaticallyUpdateChannels(freq)`：周期渠道刷新
    - `controller.AutomaticallyTestChannels()`：周期模型探活
    - `service.StartCodexCredentialAutoRefreshTask()`：Codex token 续期（10 分钟检查，到期 < 1 天则刷新）
    - `service.StartSubscriptionQuotaResetTask()`：订阅配额重置
    - `controller.StartChannelUpstreamModelUpdateTask()`：上游模型列表同步
    - **主节点专属**（由 `common.IsMasterNode()` 控制）：`controller.UpdateMidjourneyTaskBulk()` + `UpdateTaskBulk()`
14. `common.StartPyroScope()` / `common.StartSystemMonitor()`：可观测性
15. `gin.New()` → `router.SetRouter(server, ...)` → `srv.ListenAndServe(:PORT)`

---

## 3. Router 装载顺序（[router/main.go](../router/main.go)）

`router.SetRouter(server, buildFS, indexPage, classicBuildFS, classicIndexPage)` 严格按以下顺序装载，**顺序对 NoRoute 兜底至关重要**：

| # | 子路由 | 文件 | 前缀 |
|---|--------|------|------|
| 1 | API | [api-router.go](../router/api-router.go) | `/api/**`（控制台 200+ 端点） |
| 2 | Dashboard | [dashboard.go](../router/dashboard.go) | `/dashboard/**` `/v1/dashboard/**` |
| 3 | Relay | [relay-router.go](../router/relay-router.go) | `/v1/**` `/v1beta/**` `/pg/**` `/mj/**` `/suno/**` |
| 4 | Video | [video-router.go](../router/video-router.go) | `/v1/video*` `/kling/v1/**` `/jimeng/**` |
| 5 | Web | [web-router.go](../router/web-router.go) | `/`（SPA + embed FS） |

`NoRoute` 兜底逻辑：
- 命中 `/api`、`/v1`、`/dashboard`、`/assets` 等 API 前缀的 404 → `RelayNotFound`（返回 OpenAI 风格错误）
- 其他 404 → SPA fallback，返回当前 theme 的 `index.html`（让前端路由接管）

---

## 4. 中间件链路

### 4.1 全局中间件（`router.SetRouter` 顶层）
- `recover.Recover`：panic 恢复（自定义打印 + Sentry）
- `middleware.RequestId()`：每请求生成 X-Request-Id
- `middleware.Logger()`：访问日志
- `middleware.CORS()`：跨域（默认允许所有，由 `setting` 控制）
- `middleware.PoweredBy()`：响应头注入 `X-Powered-By: new-api`
- `middleware.I18n()`：根据 `Accept-Language` 切换 go-i18n locale
- `middleware.DecompressRequest()`：支持 gzip 请求体
- `middleware.Stats()`：QPS / 延迟统计
- `middleware.SystemPerformanceCheck()`：根据 `performance_setting` 拒绝过载请求

### 4.2 鉴权中间件（按路由组装）
- `UserAuth`：cookie session（控制台）
- `AdminAuth` / `RootAuth`：基于 `users.role`
- `TokenAuth`：`Authorization: Bearer sk-xxx`，解析 `tokens.key` → 注入 `User` / `Channel` / `Group` 到 context
- `TokenAuthReadOnly`：只读访问
- `TokenOrUserAuth`：兼容两种凭证
- `HeaderNavModuleAuth`：控制台模块级别可见性
- `SecureVerificationRequired`：渠道密钥读取等敏感操作要求二次验证（默认开启）
- `TurnstileCheck`：Cloudflare Turnstile 人机校验

### 4.3 转发专属中间件
- `middleware.Distribute()`：核心选址。流程：
  1. 检查 `ContextKeyTokenSpecificChannelId`（`?channel=xxx` 或 token meta）→ 指定渠道直通
  2. 检查 `tokens.model_limits`（模型白名单）
  3. 检查 Playground 子组（请求体 `group` 字段）
  4. 处理 `auto` 分组多 group 轮询
  5. 查 `model.Ability`（复合主键 `group + model + channel_id`）得候选渠道
  6. 应用渠道亲和缓存（用户↔渠道粘性）
  7. 选定 `Channel` 写入 context
- `middleware.KlingAdapter` / `middleware.JimengAdapter`：可灵 / 火山即梦 → 内部统一 video 请求重写
- `GlobalAPIRateLimit` / `ModelRequestRateLimit`：全局 + 按模型限流（Redis 计数器）

---

## 5. 适配器模式（`relay/channel/`）

### 5.1 `Adaptor` 接口

每个上游 provider 子目录必须实现 [relay/channel/adapter.go](../relay/channel/adapter.go) 中定义的接口：

| 方法 | 职责 |
|------|------|
| `Init(meta *relaycommon.RelayInfo)` | 注入 RelayInfo（包含 Channel / User / Model 元数据） |
| `GetRequestURL(info)` | 拼上游 URL（含 path 替换、stream 后缀等） |
| `SetupRequestHeader(c, req, info)` | 注入鉴权 header（Bearer / x-api-key / 签名） |
| `ConvertRequest(c, info, request)` | DTO 转换（OpenAI → Claude/Gemini/etc.） |
| `DoRequest(c, info, requestBody)` | 用 `service.http_client` 发送，返回 `*http.Response` |
| `DoResponse(c, resp, info)` | 流式则 SSE 透传，非流式解码 + 用量统计 |
| `GetModelList()` | 静态模型列表 |
| `GetChannelName()` | 渠道展示名 |

### 5.2 40 个 provider 适配器列表

`openai`（含 azure/baidu/anthropic-on-bedrock 变种）、`claude`、`gemini`、`aws`（Bedrock）、`vertex`、`ali`、`baidu` / `baidu_v2`、`tencent`、`volcengine`、`xunfei`、`zhipu` / `zhipu_4v`、`moonshot`、`deepseek`、`mistral`、`cohere`、`replicate`、`cloudflare`、`xai`、`xinference`、`siliconflow`、`lingyiwanwu`、`minimax`、`mokaai`、`ollama`、`openrouter`、`palm`、`perplexity`、`coze`、`dify`、`submodel`、`task`、`jina`、`jimeng`、`ai360`、`codex`

### 5.3 任务池适配器（异步任务平台）

视频 / Suno / Midjourney 等异步平台需要轮询任务状态，通过另一组接口：
- `service.GetTaskAdaptorFunc = func(platform constant.TaskPlatform) service.TaskPollingAdaptor` 在 `main.go` 中注入为 `relay.GetTaskAdaptor`
- 这避免了 `service` import `relay` 导致的包循环依赖

### 5.4 新增 provider 的标准步骤

1. 在 `constant/channel.go` 添加 `ChannelTypeXxx` 枚举
2. 在 `relay/channel/xxx/` 创建子目录，实现 `Adaptor` 接口（一般至少 `adaptor.go` + `request.go` + `response.go` + `models.go`）
3. 在 `relay.GetAdaptor` 工厂注册
4. **CLAUDE.md Rule 4**：确认是否支持 `StreamOptions`，若支持需加入 `streamSupportedChannels`
5. **CLAUDE.md Rule 6**：DTO 中可选标量必须用 `*int` / `*float64` / `*bool` + `omitempty`，避免显式零值被丢弃
6. 在 `setting/model_setting/` 或对应配置子包注册默认参数
7. 前端 `model_meta` 同步上游模型列表

---

## 6. Service 层职责划分

按业务域分类：

| 子域 | 关键文件 |
|------|---------|
| 转发选址 | `channel_select.go`、`channel.go`、`channel_affinity.go`、`group.go` |
| 计费/额度 | `billing.go`、`billing_session.go`、`pre_consume_quota.go`、`quota.go`、`text_quota.go`、`tiered_settle.go`、`tool_billing.go`、`task_billing.go`、`violation_fee.go` |
| 媒体处理 | `audio.go`、`image.go`、`download.go`、`file_decoder.go`、`file_service.go`、`midjourney.go`、`task.go`、`task_polling.go` |
| HTTP 客户端 | `http.go`、`http_client.go`、`convert.go`、`error.go` |
| Token 计数 | `token_counter.go`、`token_estimator.go`、`tokenizer.go`（tiktoken-go） |
| OpenAI 兼容 | `openaicompat/`（`openai_chat_responses_*`） |
| Codex（GitHub Copilot） | `codex_oauth.go`、`codex_credential_refresh*.go`、`codex_wham_usage.go` |
| 支付 | `epay.go`、`waffo_pancake.go` |
| 通知 | `webhook.go`、`notify-limit.go`、`user_notify.go` |
| 调度任务 | `subscription_reset_task.go` |
| Passkey | `passkey/` 子包 |

**计费链路（参考 [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)，CLAUDE.md Rule 7）：**
1. `service.PreConsumeQuota`：请求进入时按估算预扣
2. Adaptor `DoResponse`：解析上游 usage
3. `service.PostConsumeQuotaPostBilling`：实际扣费（含表达式规则）
4. 写 `Log`（消费类型）+ 异步更新 `QuotaData`

---

## 7. 数据访问层（`model/`）

### 7.1 跨 DB 抽象（CLAUDE.md Rule 2）

[model/main.go](../model/main.go) 暴露以下符号屏蔽方言差异：

| 符号 | 用途 |
|------|------|
| `commonGroupCol` | `group` 列的引号形式（PG `"group"`，MySQL/SQLite `` `group` ``） |
| `commonKeyCol` | `key` 列的引号形式 |
| `commonTrueVal` / `commonFalseVal` | 布尔字面量（PG `true/false`，MySQL/SQLite `1/0`） |
| `common.UsingPostgreSQL` / `UsingSQLite` / `UsingMySQL` | 运行期判断 DB 类型 |

**自动迁移约束：**
- SQLite 不支持 `ALTER COLUMN`，迁移须用 `ADD COLUMN` 兜底
- JSON 类字段统一存 `TEXT`，不用 PG 的 `JSONB`
- 索引名跨 DB 全局唯一，避免 PG 默认 schema 冲突

### 7.2 GORM 模型概览

详见 [data-models-server.md](data-models-server.md)。核心表：
- `users` / `tokens` / `channels` / `abilities` / `logs` / `options` / `redemptions`
- `subscriptions` / `subscription_plans` / `topups`
- `tasks` / `midjourney_tasks`
- `model_metas` / `vendor_metas` / `pricings`
- `twofas` / `passkeys` / `custom_oauth_providers` / `user_oauth_bindings`
- `prefill_groups` / `usedata_*` / `perf_metrics` / `checkins`

### 7.3 缓存层

- `model.InitChannelCache()`：启动期一次性加载 channel 列表到内存
- `go model.SyncChannelCache(common.SyncFrequency)`：周期同步
- `pkg/cachex.HybridCache`：Redis + samber/hot 双层（订阅 plan 缓存使用）
- `cache.go`：通用 GORM 缓存包装

---

## 8. 横切关注点

### 8.1 JSON 操作（CLAUDE.md Rule 1）

业务代码必须使用 [common/json.go](../common/json.go) 包装，禁止直接 import `encoding/json`：

| 函数 | 用途 |
|------|------|
| `common.Marshal(v any) ([]byte, error)` | 序列化 |
| `common.Unmarshal(data []byte, v any) error` | 反序列化 |
| `common.UnmarshalJsonStr(data string, v any) error` | 字符串反序列化 |
| `common.DecodeJson(reader io.Reader, v any) error` | 流式反序列化 |
| `common.GetJsonType(data json.RawMessage) string` | 类型探测 |

类型层 `json.RawMessage` / `json.Number` 仍可引用。

### 8.2 错误处理

- HTTP 错误统一 [dto/error.go](../dto/error.go) `{error: {message, type, code, param}}`
- 上游错误适配 [types/channel_error.go](../types/channel_error.go)（含可重试判断）
- 渠道失败重试：`service.ShouldSkipRetryAfterChannelAffinityFailure` 决定是否跳过亲和缓存

### 8.3 国际化（[i18n/](../i18n/)）

- `i18n.T(c *gin.Context, key string, args ...)` 工具
- 键名常量集中在 [i18n/keys.go](../i18n/keys.go)
- 资源文件 [i18n/locales/{en,zh}.json](../i18n/locales/)
- locale 由 `middleware.I18n` 中间件根据 `Accept-Language` 注入 context

### 8.4 OAuth（[oauth/](../oauth/)）

- `provider.go` 定义抽象接口
- `generic.go` 通用 OIDC 实现
- 内置 provider：`github.go` / `discord.go` / `linuxdo.go` / `oidc.go`
- 自定义 provider 通过 `registry.go` 注册（DB 持久化 `custom_oauth_providers` 表）

---

## 9. 配置体系（[setting/](../setting/)）

按业务域拆子包，每个子包暴露 `GetXxxConfig()` 读取入口、`SetXxxConfig()` 写入入口（持久化到 `options` 表）：

| 子包 | 关注点 |
|------|-------|
| `ratio_setting/` | 模型/分组倍率 |
| `model_setting/` | Claude / Gemini / Grok / Global 模型可见性 |
| `system_setting/` | 系统级行为 |
| `operation_setting/` | 日常运营 |
| `billing_setting/` | 计费策略 |
| `console_setting/` | 控制台开关 |
| `payment_creem/stripe/waffo/waffo_pancake.go` | 支付通道 |
| `perf_metrics_setting/` `performance_setting/` | 性能指标 |
| `reasoning/` | 推理参数 |
| `sensitive.go` | 敏感词 |
| `auto_group.go` `user_usable_group.go` | 分组规则 |
| `midjourney.go` `chat.go` | 业务专属 |
| `rate_limit.go` | 限流参数 |
| `config/` | 基础配置加载 |

配置热加载：`model.SyncOptions(SyncFrequency)` 周期从 DB 拉 `options` 表写回 setting 子包。

---

## 10. 多节点协同

多节点部署关键约束：

| 维度 | 机制 |
|------|------|
| Session 一致性 | `SESSION_SECRET` 必须固定一致；Redis 作 session 共享（可选） |
| 配置同步 | `model.SyncOptions(SyncFrequency)` 周期拉 DB |
| 限流共享 | Redis 计数器（`GlobalAPIRateLimit` / `ModelRequestRateLimit`） |
| 渠道缓存收敛 | `SyncChannelCache(SyncFrequency)` |
| 主节点专属任务 | `common.IsMasterNode()` 守门（由 `NODE_TYPE=master` 设置） |
| `FRONTEND_BASE_URL` 行为 | 非主节点 301 跳外部前端，主节点强制忽略避免循环 |
| 节点标识 | `HOSTNAME` / `NODE_NAME` 用于审计日志 |

---

## 11. 可观测性

| 工具 | 入口 | 用途 |
|------|------|------|
| `/api/status` | [controller/misc.go](../controller/misc.go) | 健康检查（docker-compose 健康探针使用） |
| `/api/perf-metrics` + `/api/perf-metrics/summary` | [controller/perf_metrics.go](../controller/perf_metrics.go) | 模型层性能指标 |
| `/api/performance/*` | [controller/performance.go](../controller/performance.go) | RootAuth 性能控制台（stats/disk_cache/gc/logs） |
| `/api/uptime/status` | [controller/uptime_kuma.go](../controller/uptime_kuma.go) | Uptime Kuma 探活反代 |
| `net/http/pprof` | 监听 `:8005` | `ENABLE_PPROF=true` 启用 |
| Pyroscope | `common.StartPyroScope()` | 持续 profiling（`PYROSCOPE_URL` 等环境变量） |
| 系统监控 | `common.StartSystemMonitor()` | CPU / 内存 / 磁盘（跨平台 Unix/Windows） |

---

## 12. 设计上的关键耦合点

1. **`service` ↔ `relay` 循环**：通过 `service.GetTaskAdaptorFunc` 函数指针 + `main.go` 注入解耦。新增异步任务平台时必须同时在 `relay.GetTaskAdaptor` 注册
2. **DTO 双向复用**：`dto/openai_*.go` 同时承担前端入参解析与上游出参编码。修改字段时严格遵循 CLAUDE.md Rule 6（指针 + `omitempty`）保留显式零值语义
3. **`go:embed` 强约束**：`web/default/dist/` 与 `web/classic/dist/` 必须存在（即使空），否则 `go build` 失败。`Dockerfile.dev` 用占位 HTML 满足约束
4. **HTML 注入一次性**：`InjectUmamiAnalytics()` / `InjectGoogleAnalytics()` 在启动时一次性替换 embed FS 中的 `index.html`，主题切换不会重读，配置变更需重启
5. **跨 DB 列名**：所有触及 `group` 或 `key` 列的 raw SQL 必须用 `commonGroupCol` / `commonKeyCol`，否则在 PG 下会因引号风格不同而失败
6. **PORT 隐式契约**：`PORT=3000` 默认值与 Electron `checkServerAvailability(3000)` 形成隐式契约，修改后 Electron 探活将失败

---

## 13. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| HTTP 端点契约 | [api-contracts-server.md](api-contracts-server.md) |
| 数据模型 | [data-models-server.md](data-models-server.md) |
| 跨 part 集成 | [integration-architecture.md](integration-architecture.md) |
| 源码树详解 | [source-tree-analysis.md §2](source-tree-analysis.md) |
| 部署与运维 | [deployment-guide.md](deployment-guide.md) |
| 计费表达式语言 | [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
