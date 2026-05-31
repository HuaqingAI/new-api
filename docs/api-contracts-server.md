# API 契约 — 服务端 (Go Backend)

> 项目部分：`server`（Go / Gin / GORM）
> 路由入口：[router/main.go](../router/main.go)
> 文档生成时间：2026-05-26
> 仓库类型：monorepo（4 部分）

---

## 1. 路由总览

服务端在 `router/main.go::SetRouter` 中按路由组装载，所有公共子路由会注入 `RouteTag` 中间件用于 metric / 日志区分：

| 路由族 | 注册函数 | 文件 | 说明 |
|--------|----------|------|------|
| `/api/**` | `SetApiRouter` | [router/api-router.go](../router/api-router.go) | 控制台/管理 API（用户、渠道、令牌、订阅、计费、运维） |
| `/dashboard/**`、`/v1/dashboard/**` | `SetDashboardRouter` | [router/dashboard.go](../router/dashboard.go) | OpenAI 兼容订阅查询（subscription / usage） |
| `/v1/**`、`/v1beta/**`、`/pg/**`、`/mj/**`、`/suno/**` | `SetRelayRouter` | [router/relay-router.go](../router/relay-router.go) | OpenAI / Anthropic / Gemini / Midjourney / Suno 模型转发 |
| `/v1/video*`、`/kling/v1/**`、`/jimeng/**` | `SetVideoRouter` | [router/video-router.go](../router/video-router.go) | 视频生成转发（OpenAI compat / 可灵 / 即梦） |
| `/`（前端） | `SetWebRouter` | [router/web-router.go](../router/web-router.go) | 前端静态资源（embed） |

如设置了 `FRONTEND_BASE_URL`，前端访问会被 301 重定向到外部前端站点；主节点上 `FRONTEND_BASE_URL` 会被强制忽略以避免重定向循环。

## 2. 中间件链

通用前置（全局或多组共享）：

| 中间件 | 文件 | 作用 |
|--------|------|------|
| `RouteTag` | [middleware/header_nav.go](../middleware/header_nav.go) | 路由分类标记，用于 metric/日志 |
| `gzip.Gzip` | gin-contrib/gzip | API 响应压缩 |
| `BodyStorageCleanup` | [middleware/body_cleanup.go](../middleware/body_cleanup.go) | 释放大请求体内存 |
| `GlobalAPIRateLimit` / `CriticalRateLimit` / `SearchRateLimit` / `ModelRequestRateLimit` / `EmailVerificationRateLimit` | [middleware/rate-limit.go](../middleware/rate-limit.go)、[middleware/model-rate-limit.go](../middleware/model-rate-limit.go)、[middleware/email-verification-rate-limit.go](../middleware/email-verification-rate-limit.go) | 多维度限流 |
| `CORS` | [middleware/cors.go](../middleware/cors.go) | 跨域 |
| `DecompressRequestMiddleware` | [middleware/gzip.go](../middleware/gzip.go) | 上行 gzip 解压 |
| `StatsMiddleware` | [middleware/stats.go](../middleware/stats.go) | 请求统计 |
| `SystemPerformanceCheck` | [middleware/performance.go](../middleware/performance.go) | 高负载下熔断 |
| `TurnstileCheck` | [middleware/turnstile-check.go](../middleware/turnstile-check.go) | Cloudflare Turnstile 验证码 |
| `DisableCache` | [middleware/disable-cache.go](../middleware/disable-cache.go) | 禁用响应缓存 |

鉴权中间件（[middleware/auth.go](../middleware/auth.go)）：

| 中间件 | 鉴权来源 | 作用域 |
|--------|----------|--------|
| `UserAuth` | session 或 Authorization access-token（最低 RoleCommonUser） | 普通登录用户 |
| `AdminAuth` | session 或 Authorization access-token（最低 RoleAdminUser） | 管理员 |
| `RootAuth` | session 或 Authorization access-token（RoleRootUser） | 超级管理员 |
| `TokenAuth` | API Key（`Authorization: Bearer …`）+ token 到渠道映射 | 模型转发 |
| `TokenAuthReadOnly` | API Key（只读用法 token-usage 等） | 仅查询额度/用量 |
| `TokenOrUserAuth` | 同时接受 API Key 或 session | 视频代理读取 |
| `HeaderNavModuleAuth("…")` | 顶部导航模块开关 + 用户态 | 公共 / 用户 |
| `SecureVerificationRequired` | 二次安全验证 | 高敏感操作（如读取渠道 Key） |

转发链路核心：`Distribute()`（[middleware/distributor.go](../middleware/distributor.go)）。该中间件根据 `(group, model)` 选渠道，支持：
- 显式 `ContextKeyTokenSpecificChannelId` 直连指定渠道
- 令牌模型限制（`ContextKeyTokenModelLimit`，命中 `ratio_setting.FormatMatchingModelName`）
- Playground 子组校验（`/pg/chat/completions`）
- 渠道亲和缓存（`service.GetPreferredChannelByAffinity`）
- `auto` 分组下的多组轮询（`service.GetUserAutoGroup`）
- 失败时按重试策略回退（参见 `service.ShouldSkipRetryAfterChannelAffinityFailure`）

特殊请求重写：`KlingRequestConvert`、`JimengRequestConvert` 在转发前把上游官方协议（kling.ai / jimeng / 火山）规范成内部统一表征。

## 3. 控制台 API（`/api/**`）

> 共 60+ 控制器文件位于 [controller/](../controller/)。下表按业务域汇总核心端点；具体参数、响应字段以处理器源代码与 [dto/](../dto/) 为准。

### 3.1 公开端点（无需鉴权或仅注册期）
| Method | Path | Handler | 描述 |
|--------|------|---------|------|
| GET | `/api/setup` | `controller.GetSetup` | 系统是否已初始化 |
| POST | `/api/setup` | `controller.PostSetup` | 首次初始化，创建 root |
| GET | `/api/status` | `controller.GetStatus` | 系统配置/能力 |
| GET | `/api/uptime/status` | `controller.GetUptimeKumaStatus` | Uptime Kuma 探活反代 |
| GET | `/api/notice` / `/about` / `/home_page_content` / `/user-agreement` / `/privacy-policy` | misc.go | 站点静态文案 |
| GET | `/api/pricing` | `controller.GetPricing` | 价格列表（受 HeaderNavModule 开关） |
| GET | `/api/perf-metrics` / `/perf-metrics/summary` | `controller.GetPerfMetrics(Summary)` | 模型性能指标 |
| GET | `/api/rankings` | `controller.GetRankings` | 模型排行 |
| GET | `/api/ratio_config` | `controller.GetRatioConfig` | 倍率配置（前端启动加载） |

### 3.2 注册 / 登录 / 邮箱 / 2FA / Passkey
| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/user/register` | `Register` |
| POST | `/api/user/login` | `Login` |
| POST | `/api/user/login/2fa` | `Verify2FALogin` |
| POST | `/api/user/passkey/login/begin` / `…/finish` | `PasskeyLoginBegin` / `Finish` |
| GET | `/api/user/logout` | `Logout` |
| GET | `/api/verification` | `SendEmailVerification` |
| GET | `/api/reset_password` / POST `/api/user/reset` | `SendPasswordResetEmail` / `ResetPassword` |
| POST | `/api/verify` | `UniversalVerify`（统一二次验证） |

### 3.3 OAuth
- 标准 OAuth：`GET /api/oauth/state`、`GET /api/oauth/:provider`（GitHub / Discord / OIDC / LinuxDO）
- 非标 OAuth：`GET /api/oauth/wechat`、`POST /api/oauth/wechat/bind`、`GET /api/oauth/telegram/login`、`GET /api/oauth/telegram/bind`
- 自定义 OAuth Provider 管理（root 权限）：`/api/custom-oauth-provider/**`（discovery / CRUD）
- 用户态绑定：`/api/user/self/oauth/bindings`、`DELETE /:provider_id`

### 3.4 用户自管理 `/api/user/self/**`（UserAuth）
- 获取/更新/删除自身 `GET/PUT/DELETE /self`
- 模型/分组/Aff：`/models`、`/self/groups`、`/aff`、`/aff_transfer`
- 充值入口：`/topup`、`/pay`、`/stripe/pay`、`/creem/pay`、`/waffo/pay`、`/waffo-pancake/pay` 及对应 `*/amount`
- 设置：`/setting`
- 2FA：`/2fa/status`、`/2fa/setup`、`/2fa/enable`、`/2fa/disable`、`/2fa/backup_codes`
- Passkey：`/passkey`、`/passkey/register/(begin|finish)`、`/passkey/verify/(begin|finish)`、`DELETE /passkey`
- 签到：`GET /checkin`、`POST /checkin`

### 3.5 用户管理（AdminAuth）`/api/user/**`
- `GET /`、`GET /search`、`GET /:id`、`POST /`、`POST /manage`、`PUT /`、`DELETE /:id`
- Admin 端 OAuth 解绑：`GET /:id/oauth/bindings`、`DELETE /:id/oauth/bindings/:provider_id`、`DELETE /:id/bindings/:binding_type`、`DELETE /:id/reset_passkey`
- Admin 端 2FA：`GET /2fa/stats`、`DELETE /:id/2fa`
- TopUp 列表与人工补单：`GET /topup`、`POST /topup/complete`

### 3.6 渠道 `/api/channel/**`（AdminAuth；部分需 RootAuth）
关键端点：
- 列表/搜索/详情/CRUD：`GET /`、`/search`、`/:id`、`POST /`、`PUT /`、`DELETE /:id`
- 渠道密钥（敏感）：`POST /:id/key`（RootAuth + CriticalRateLimit + DisableCache + SecureVerificationRequired）
- 测试与平衡查询：`/test`、`/test/:id`、`/update_balance`、`/update_balance/:id`
- 模型列表：`/models`、`/models_enabled`、`/fetch_models/:id`、`POST /fetch_models`(root)、`/tag/models`
- 标签批量：`/tag/disabled`、`/tag/enabled`、`PUT /tag`、`POST /batch/tag`
- 多 Key：`POST /multi_key/manage`
- Codex OAuth：`/codex/oauth/start|complete`、`/:id/codex/oauth/start|complete`、`/:id/codex/refresh`、`GET /:id/codex/usage`
- Ollama：`/ollama/pull`、`/ollama/pull/stream`、`/ollama/delete`、`/ollama/version/:id`
- 上游模型同步：`/upstream_updates/(detect|detect_all|apply|apply_all)`
- 复制/批量删除：`/copy/:id`、`POST /batch`、`DELETE /disabled`、`/fix`

### 3.7 令牌 `/api/token/**`（UserAuth）
`GET /`、`/search`、`/:id`、`POST /:id/key`（CriticalRateLimit + DisableCache）、`POST /`、`PUT /`、`DELETE /:id`、`POST /batch`、`POST /batch/keys`

### 3.8 用量 `/api/usage/token/`
基于 token 只读鉴权，返回 token 用量。

### 3.9 兑换码 `/api/redemption/**`（AdminAuth）
`GET /`、`/search`、`/:id`、`POST /`、`PUT /`、`DELETE /invalid`、`DELETE /:id`

### 3.10 日志/统计 `/api/log/**`、`/api/data/**`
- 全量日志（admin）：`GET /`、`DELETE /`、`GET /stat`、`GET /search`、`GET /channel_affinity_usage_cache`
- 用户日志：`GET /self`、`GET /self/stat`、`GET /self/search`
- 令牌日志只读：`GET /log/token`（TokenAuthReadOnly）
- 配额统计：`GET /api/data/`、`/users`、`/self`

### 3.11 订阅 `/api/subscription/**`
- 用户：`GET /plans`、`GET /self`、`PUT /self/preference`、`POST /balance/pay`、`POST /epay/pay`、`POST /stripe/pay`、`POST /creem/pay`、`POST /waffo-pancake/pay`
- 管理：`/admin/plans` CRUD、`PATCH /:id`（status）、`POST /admin/bind`、`/admin/users/:id/subscriptions`（list/create）、`/admin/user_subscriptions/:id/(invalidate|delete)`
- 易支付回调（无鉴权）：`POST/GET /api/subscription/epay/notify`、`/return`

### 3.12 系统选项 `/api/option/**`（RootAuth）
- `GET / PUT /`、`POST /payment_compliance`
- 渠道亲和缓存：`GET/DELETE /channel_affinity_cache`
- 倍率/迁移：`POST /rest_model_ratio`、`POST /migrate_console_setting`
- Waffo Pancake：`/waffo-pancake/(catalog|pair|save|subscription-product|subscription-product-options)`

### 3.13 性能 `/api/performance/**`（RootAuth）
`GET /stats`、`DELETE /disk_cache`、`POST /reset_stats`、`POST /gc`、`GET /logs`、`DELETE /logs`

### 3.14 倍率同步 `/api/ratio_sync/**`（RootAuth）
`GET /channels`、`POST /fetch`

### 3.15 模型元数据 `/api/models/**`（AdminAuth）
`GET /sync_upstream/preview`、`POST /sync_upstream`、`GET /missing`、`GET / /:id /search`、`POST /`、`PUT /`、`DELETE /:id`

### 3.16 部署 `/api/deployments/**`（AdminAuth, IO.Net 集成）
- 设置：`GET /settings`、`POST /settings/test-connection`
- CRUD：`GET / /:id /search`、`POST /`、`PUT /:id`、`DELETE /:id`
- 元信息：`/hardware-types`、`/locations`、`/available-replicas`、`/check-name`
- 估价/扩容：`POST /price-estimation`、`POST /:id/extend`
- 容器：`GET /:id/(logs|containers|containers/:container_id)`
- 名称：`PUT /:id/name`

### 3.17 任务 `/api/task/**`、Midjourney `/api/mj/**`、分组 `/api/group/`、预填分组 `/api/prefill_group/**`、供应商 `/api/vendors/**`
均为列表/CRUD，权限按管理员/用户区分。

### 3.18 支付 Webhook（公网回调，无鉴权）
- `POST /api/stripe/webhook`、`/api/creem/webhook`、`/api/waffo/webhook`、`/api/waffo-pancake/webhook/:env`
- `:env` 用于运营方在 Pancake 后台为 `test` / `prod` 各登记一个 webhook，处理器内会校验。

### 3.19 企业管理 `/api/enterprise/**`

企业管理路由由 `RegisterEnterpriseRouter` 挂载到 `/api/enterprise`，路由组先经过 `UserAuth`，再按资源叠加 `AdminAuth`、`RootAuth`、`EnterpriseAdmin` 或 `EnterpriseDepartmentAdmin(...)`。

#### 组织与预算
- `GET /api/enterprise/departments/tree`
- `GET/PUT /api/enterprise/users/:id/departments`
- `GET /api/enterprise/departments/:id/members`
- `POST /api/enterprise/departments/:id/members`
- `DELETE /api/enterprise/departments/:id/members/:user_id`
- `POST /api/enterprise/departments/:id/members/:user_id/restore`
- `GET /api/enterprise/departments/:id/budget`
- `GET /api/enterprise/departments/:id/budgets`
- `GET /api/enterprise/departments/:id/budgets/:budget_id`
- `POST /api/enterprise/departments/:id/budget`
- `GET/POST /api/enterprise/quota-allocations`
- `POST /api/enterprise/quota-allocations/:id/revoke`
- `POST /api/enterprise/departments/:id/admins`
- `DELETE /api/enterprise/departments/:id/admins/:user_id`
- `GET /api/enterprise/admin-actions`
- `GET /api/enterprise/admin-actions/:id`

#### 内容风险事件与告警
- `GET /api/enterprise/alerts/events`
  - 过滤参数支持 `tenant_id`、`department_id`、`unassigned_only`、`user_id`、`username`、`model_name`、`risk_type`、`from`、`to`、`page`、`page_size`
  - 查询语义基于 `enterprise_alert_events` 中的历史部门快照，不回查当前成员关系
- `GET /api/enterprise/alerts/department-summary`
  - 必填参数：`from`、`to`
  - 可选参数：`tenant_id`、`summary_sort`、`summary_order`
  - 时间语义为 `[from, to)`；风险分子来自 `enterprise_alert_events`，风险分母复用 `enterprise_usage_snapshots`
- `GET /api/enterprise/alerts/deliveries`
  - 过滤参数支持 `tenant_id`、`rule_id`、`event_id`、`manual_parent_id`、`channel_type`、`status`、`trigger_source`、`page`、`page_size`
- `POST /api/enterprise/alerts/deliveries/:id/resend`
  - 支持按既有投递记录触发手动重发，并写入新的投递状态与审计记录
- `GET /api/enterprise/alerts/rules`
- `GET /api/enterprise/alerts/rules/:id`
- `PUT /api/enterprise/alerts/rules`
  - 同一接口承载创建与更新；通道配置当前支持 `email`、`webhook`、`dingtalk_robot`
- `DELETE /api/enterprise/alerts/rules/:id`

#### 用量聚合与报告
- `GET /api/enterprise/usage/department-summary`
- `GET /api/enterprise/usage/department-detail`
- `GET /api/enterprise/usage/export`
- `GET/PUT /api/enterprise/usage/reports`

#### 钉钉企业集成
- `GET/PUT /api/enterprise/dingtalk/config`
- `POST /api/enterprise/dingtalk/connectivity-test`
- `POST /api/enterprise/dingtalk/sync/full`
- `GET /api/enterprise/dingtalk/sync/tasks/:id`
- `GET /api/enterprise/dingtalk/sync/logs`
- `GET /api/enterprise/dingtalk/sync/conflicts`
- `POST /api/enterprise/dingtalk/sync/conflicts/:id/bind-candidate`

## 4. 模型转发 API（`SetRelayRouter`）

转发路径统一遵循 `controller.Relay(c, types.RelayFormat*)` 的入口分发，最终落到 `relay/channel/<provider>/` 适配器实现。可见 [relay/channel/](../relay/channel/)（40 个 provider）：`openai`, `claude`, `gemini`, `aws`(Bedrock), `vertex`, `ali`, `baidu`, `baidu_v2`, `tencent`, `volcengine`, `xunfei`, `zhipu`, `zhipu_4v`, `moonshot`, `deepseek`, `mistral`, `cohere`, `replicate`, `cloudflare`, `xai`, `xinference`, `siliconflow`, `lingyiwanwu`, `minimax`, `mokaai`, `ollama`, `openrouter`, `palm`, `perplexity`, `coze`, `dify`, `submodel`, `task`, `jina`, `jimeng`, `ai360`, `codex`。

### 4.1 OpenAI 兼容 `/v1/**`
| Method | Path | RelayFormat |
|--------|------|-------------|
| GET | `/v1/models` | List/Retrieve（按 header `x-api-key` + `anthropic-version` 或 `x-goog-api-key` 自适应到 Anthropic / Gemini，否则 OpenAI） |
| GET | `/v1/models/:model` | Retrieve（同上自适应） |
| GET | `/v1/realtime` | `RelayFormatOpenAIRealtime`（WebSocket） |
| POST | `/v1/messages` | `RelayFormatClaude` |
| POST | `/v1/completions`、`/v1/chat/completions` | `RelayFormatOpenAI` |
| POST | `/v1/responses`、`/v1/responses/compact` | `RelayFormatOpenAIResponses(Compaction)` |
| POST | `/v1/edits`、`/v1/images/generations`、`/v1/images/edits` | `RelayFormatOpenAIImage` |
| POST | `/v1/embeddings` | `RelayFormatEmbedding` |
| POST | `/v1/audio/(transcriptions\|translations\|speech)` | `RelayFormatOpenAIAudio` |
| POST | `/v1/rerank` | `RelayFormatRerank` |
| POST | `/v1/engines/:model/embeddings`、`/v1/models/*path` | `RelayFormatGemini`（OpenAI compat 下的 Gemini path） |
| POST | `/v1/moderations` | `RelayFormatOpenAI` |
| 多个 | `/v1/files/*`、`/v1/fine-tunes/*`、`/v1/images/variations` | `controller.RelayNotImplemented` |

### 4.2 Gemini 原生 `/v1beta/**`
- `GET /v1beta/models`、`GET /v1beta/openai/models`：Gemini / OpenAI compat 列表
- `POST /v1beta/models/*path`：转发 `RelayFormatGemini`

### 4.3 Playground `/pg/chat/completions`
受 `UserAuth` + `Distribute`，允许在请求体里携带 `group` 字段切换分组（需在 user usable groups 内）。

### 4.4 Midjourney `/mj` 与 `/:mode/mj`
覆盖 `submit/(action|shorten|modal|imagine|change|simple-change|describe|blend|edits|video)`、`task/:id/fetch`、`task/:id/image-seed`、`task/list-by-condition`、`insight-face/swap`、`submit/upload-discord-images`、`image/:id`（无鉴权图床）。

### 4.5 Suno `/suno`
`POST /submit/:action`、`POST /fetch`、`GET /fetch/:id`。

### 4.6 视频 `/v1/video*`、`/v1/videos*`、`/kling/v1/**`、`/jimeng`
- 通用：`POST /v1/video/generations`、`GET /v1/video/generations/:task_id`、`POST /v1/videos/:video_id/remix`
- OpenAI 兼容：`POST /v1/videos`、`GET /v1/videos/:task_id`、`GET /v1/videos/:task_id/content`（双鉴权）
- 可灵：`POST /kling/v1/videos/(text2video|image2video)`、`GET /kling/v1/videos/(text2video|image2video)/:task_id`
- 即梦：`POST /jimeng/`（火山官方 `Action=CVSync2AsyncSubmitTask|GetResult`）

## 5. 公共请求/响应规范

- 内容协商：默认 `application/json`；流式接口（`?stream=true` 或 client request 体含 `stream:true`）走 SSE，遵循上游格式。
- 鉴权头：API Key 走 `Authorization: Bearer <token>`；Anthropic 兼容支持 `x-api-key`；Gemini 兼容支持 `x-goog-api-key` 或 query `?key=`。
- 限流：分两层 — 全局（IP/路由组）+ 模型粒度（`ModelRequestRateLimit`）。
- 错误体：以 OpenAI Error 格式返回（`abortWithOpenAiMessage`），具体见 [types/](../types/) 与 [dto/error.go](../dto/error.go)。
- i18n：所有 distributor 错误消息走 `i18n.T(c, …)`，参见 [i18n/](../i18n/)。

## 6. 端点统计

- 控制台 `/api/**`：约 200+ 端点（用户/订阅/渠道/令牌/兑换/日志/配置/部署/订阅/Vendor/Model meta/Task/MJ）
- 其中 `/api/enterprise/**` 已覆盖组织树、部门预算、预算分配、管理员审计、钉钉同步、风险事件、告警规则、告警投递、风险概览和用量报告配置
- 转发 `/v1`、`/v1beta`、`/pg`、`/mj`、`/suno`：约 40 端点（含 8 个 NotImplemented 占位）
- 视频 `/v1/video*`、`/kling`、`/jimeng`：8 端点
- Dashboard 兼容 `/dashboard`、`/v1/dashboard`：4 端点

> 详细字段定义见 [data-models-server.md](data-models-server.md) 与 [dto/](../dto/) 源码。
