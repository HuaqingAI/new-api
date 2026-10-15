# 源码树注释 — new-api

> 文档生成时间：2026-05-26
> 范围：仓库根 + 4 个 part 关键目录
> 仓库类型：monorepo

---

## 1. 仓库根布局

```
new-api/
├─ main.go                    Go 后端入口（启动数据库、缓存、HTTP server）
├─ go.mod / go.sum            Go module 元数据（module: github.com/QuantumNous/new-api）
├─ makefile                   构建/启动便捷脚本
├─ Dockerfile                 生产镜像（多阶段：Bun build web → Go build → 运行镜像）
├─ Dockerfile.dev             开发镜像
├─ docker-compose.yml / docker-compose.dev.yml
├─ .env.example               环境变量模板
├─ new-api.service            systemd unit 模板
├─ VERSION                    版本占位（CI 注入）
├─ LICENSE / NOTICE / THIRD-PARTY-LICENSES.md
├─ README*.md                 多语言 README（en/fr/ja/zh_CN/zh_TW）
├─ .github/                   GitHub Actions workflows + 模板
│
├─ common/                    通用工具（JSON wrapper / Redis / 环境变量 / 限流 / TOTP / 邮件 / SSRF 等）
├─ constant/                  常量（Channel/API/Context Key/Cache Key/Finish Reason 等枚举）
├─ controller/                HTTP handler（约 60 文件，按业务域）
├─ dto/                       请求 / 响应数据结构（OpenAI/Claude/Gemini/Image/Audio/Video/Suno/MJ）
├─ i18n/                      go-i18n 后端国际化（en/zh，keys.go 集中键名）
├─ logger/                    日志包装
├─ middleware/                Gin 中间件（auth/distributor/rate-limit/cors/header_nav/turnstile 等）
├─ model/                     GORM 模型层（约 40 文件，DB 兼容 SQLite/MySQL/PostgreSQL）
├─ oauth/                     OAuth 抽象（generic provider + github/discord/linuxdo/oidc 子实现 + registry）
├─ pkg/
│  ├─ billingexpr/            动态计费表达式（含 expr.md，必读）
│  ├─ cachex/                 HybridCache（Redis + samber/hot 双层）
│  ├─ ionet/                  IO.Net 部署集成
│  └─ perf_metrics/           性能指标采集
├─ relay/
│  ├─ channel/                40 个上游适配器（openai/claude/gemini/aws/...）
│  └─ common/                 RelayInfo / ChannelMeta 等转发期类型
├─ router/                    路由装载（main + api + dashboard + relay + video + web）
├─ service/                   业务服务层（计费/分组/通道/Token/Codex/Channel Affinity/Tiered Settle/Webhook）
├─ setting/                   分组配置（ratio/model/system/billing/payment/perf/rate_limit/console）
├─ types/                     运行期类型（RelayFormat / FileSource / ChannelError / RWMap）
│
├─ web/                       前端容器（多主题）
│  ├─ default/                React 19 + Rsbuild + Base UI + Tailwind
│  └─ classic/                React 18 + Vite + Semi Design
│
├─ electron/                  Electron 桌面外壳（main.js / preload.js）
│
├─ docs/                      项目文档（含本工作流产出）
├─ _bmad/ / _bmad-output/     BMAD 工作流元数据与产出
├─ .agents/ / .claude/        Claude Code skills/配置
└─ bin/                       构建产物落地目录
```

---

## 2. Go 后端（part: `server`）注释源码树

### 2.1 `main.go`
- 调用 `model.InitDB()` → `model.InitLogDB()` → `common.InitRedisClient()` → 初始化 i18n / 设置 / Redis 限流 → 初始化所有 setting 默认值 → 启动后台任务（`UpdateQuotaData`、订阅重置、Codex 凭据刷新等）
- `gin.New()` → 注入 `recover/cors/request-id/logger` → `router.SetRouter()`
- `srv.ListenAndServe(:PORT)`

### 2.2 `router/`
- [main.go](../router/main.go)：聚合所有子路由；处理 `FRONTEND_BASE_URL` 重定向 vs embed
- [api-router.go](../router/api-router.go)：`/api/**` 控制台 API（200+ 端点）
- [dashboard.go](../router/dashboard.go)：`/dashboard/**`、`/v1/dashboard/**` OpenAI 兼容订阅查询
- [relay-router.go](../router/relay-router.go)：`/v1/**`、`/v1beta/**`、`/pg/**`、`/mj/**`、`/suno/**`
- [video-router.go](../router/video-router.go)：`/v1/video*`、`/kling/v1/**`、`/jimeng/`
- [web-router.go](../router/web-router.go)：嵌入式前端静态资源（`go:embed` web/dist）

### 2.3 `controller/`（按业务域）
- 鉴权：`oauth.go`、`github.go`、`discord.go`、`oidc.go`、`linuxdo.go`、`wechat.go`、`telegram.go`、`custom_oauth.go`、`twofa.go`、`passkey.go`、`secure_verification.go`
- 用户/Token/兑换：`user.go`、`token.go`、`redemption.go`、`checkin.go`
- 渠道：`channel.go`、`channel-billing.go`、`channel-test.go`、`channel_affinity_cache.go`、`channel_upstream_update.go`、`codex_oauth.go`、`codex_usage.go`、`missing_models.go`
- 转发：`relay.go`、`midjourney.go`、`task.go`、`task_video.go`、`video_proxy.go`、`video_proxy_gemini.go`、`image.go`
- 模型/价格：`model.go`、`model_meta.go`、`model_sync.go`、`vendor_meta.go`、`pricing.go`、`ratio_config.go`、`ratio_sync.go`、`prefill_group.go`
- 订阅与充值：`subscription.go`、`subscription_payment_*.go`、`topup.go`、`topup_creem.go`、`topup_stripe.go`、`topup_waffo*.go`、`payment_compliance.go`、`payment_webhook_availability.go`
- 运维：`option.go`、`performance.go`、`perf_metrics.go`、`rankings.go`、`misc.go`、`uptime_kuma.go`、`return_path.go`、`setup.go`、`console_migrate.go`、`group.go`、`deployment.go`、`log.go`、`usedata.go`、`playground.go`、`swag_video.go`

### 2.4 `service/`（业务服务层）
- 转发选址：`channel_select.go`、`channel.go`、`channel_affinity.go`、`group.go`
- 计费/额度：`billing.go`、`billing_session.go`、`pre_consume_quota.go`、`quota.go`、`text_quota.go`、`tiered_settle.go`、`tool_billing.go`、`usage_helpr.go`、`task_billing.go`、`violation_fee.go`、`funding_source.go`
- 媒体处理：`audio.go`、`image.go`、`download.go`、`file_decoder.go`、`file_service.go`、`midjourney.go`、`task.go`、`task_polling.go`
- HTTP/转换：`http.go`、`http_client.go`、`convert.go`、`error.go`、`return_path.go`
- 计算 token：`token_counter.go`、`token_estimator.go`、`tokenizer.go`
- OpenAI 兼容子包：`openaicompat/`（`openai_chat_responses_*`）
- Codex：`codex_oauth.go`、`codex_credential_refresh*.go`、`codex_wham_usage.go`
- 其他：`epay.go`、`waffo_pancake.go`、`webhook.go`、`notify-limit.go`、`user_notify.go`、`subscription_reset_task.go`、`rankings.go`、`sensitive.go`、`str.go`、`passkey/`

### 2.5 `model/`
GORM 模型与 DB 接入。主要文件：`main.go`（DB 连接 + 列名常量）、`user.go`、`token.go`、`channel.go`、`ability.go`、`log.go`、`option.go`、`redemption.go`、`subscription.go`、`task.go`、`midjourney.go`、`model_meta.go`、`vendor_meta.go`、`pricing.go`、`pricing_default.go`、`pricing_refresh.go`、`topup.go`、`twofa.go`、`passkey.go`、`custom_oauth_provider.go`、`user_oauth_binding.go`、`prefill_group.go`、`usedata*.go`、`perf_metric.go`、`checkin.go`、`setup.go`、`missing_models.go`、`cache.go`。详见 [data-models-server.md](data-models-server.md)。

### 2.6 `relay/`
- [adapter.go](../relay/channel/adapter.go)：`Adaptor` 接口（Init/SetupRequestHeader/ConvertRequest/DoRequest/DoResponse 等）
- [api_request.go](../relay/channel/api_request.go)：通用 HTTP 调用骨架
- 40 个 provider 子目录：`openai`（含变种 azure/baidu/anthropic-on-bedrock）、`claude`、`gemini`、`aws`（Bedrock）、`vertex`、`ali`、`baidu` / `baidu_v2`、`tencent`、`volcengine`、`xunfei`、`zhipu` / `zhipu_4v`、`moonshot`、`deepseek`、`mistral`、`cohere`、`replicate`、`cloudflare`、`xai`、`xinference`、`siliconflow`、`lingyiwanwu`、`minimax`、`mokaai`、`ollama`、`openrouter`、`palm`、`perplexity`、`coze`、`dify`、`submodel`、`task`、`jina`、`jimeng`、`ai360`、`codex`

### 2.7 `middleware/`
- 鉴权：`auth.go`（UserAuth/AdminAuth/RootAuth/TokenAuth/TokenAuthReadOnly/TokenOrUserAuth/HeaderNavModuleAuth/SecureVerificationRequired）
- 转发：`distributor.go`（核心选址）+ `kling_adapter.go`、`jimeng_adapter.go`（请求重写）
- 限流：`rate-limit.go`、`model-rate-limit.go`、`email-verification-rate-limit.go`
- 缓存/CORS/日志：`cache.go`、`cors.go`、`logger.go`、`request-id.go`、`recover.go`
- 性能/统计：`performance.go`、`stats.go`
- 其他：`gzip.go`、`body_cleanup.go`、`disable-cache.go`、`turnstile-check.go`、`header_nav.go`、`secure_verification.go`、`i18n.go`

### 2.8 `setting/`
配置子包，每个使用 `setting.GetXXXConfig()` 暴露读取入口：
- `ratio_setting/`：模型/分组倍率
- `model_setting/`：模型可见性、Claude/Gemini/Grok/Global
- `system_setting/`：系统行为
- `operation_setting/`：日常运营
- `billing_setting/`：计费策略
- `console_setting/`：控制台开关
- `payment_creem/stripe/waffo/waffo_pancake.go`、`payment_waffo_pancake.go`
- `perf_metrics_setting/`、`performance_setting/`
- `rate_limit.go`、`reasoning/`、`sensitive.go`、`auto_group.go`、`user_usable_group.go`、`midjourney.go`、`chat.go`
- `config/`：基础配置加载

### 2.9 `common/`
- JSON 包装：`json.go`（必用，CLAUDE.md Rule 1）
- 数据库/Redis：`database.go`、`redis.go`
- 环境/配置：`env.go`、`init.go`、`constants.go`
- 安全：`crypto.go`、`hash.go`、`totp.go`、`url_validator.go`、`ssrf_protection.go`、`validate.go`、`verification.go`
- 邮件：`email.go`、`email-outlook-auth.go`
- 限流：`limiter/`、`rate-limit.go`
- 性能：`disk_cache.go`、`disk_cache_config.go`、`performance_config.go`、`pyro.go`、`pprof.go`、`system_monitor*.go`、`gopool.go`
- 其他：`audio.go`、`body_storage.go`、`copy.go`、`custom-event.go`、`embed-file-system.go`、`endpoint_*.go`、`go-channel.go`、`gin.go`、`ip.go`、`model.go`、`page_info.go`、`quota.go`、`str.go`、`sys_log.go`、`topup-ratio.go`、`utils.go`

### 2.10 `dto/`
请求/响应结构。文件名与端点一一对应：
`audio.go`、`claude.go`、`embedding.go`、`error.go`、`gemini.go`、`midjourney.go`、`notify.go`、`openai_compaction.go`、`openai_image.go`、`openai_request.go`、`openai_response.go`、`openai_responses_compaction_request.go`、`openai_video.go`、`playground.go`、`pricing.go`、`ratio_sync.go`、`realtime.go`、`request_common.go`、`rerank.go`、`sensitive.go`、`suno.go`、`task.go`、`user_settings.go`、`values.go`、`video.go`、`channel_settings.go`

### 2.11 `i18n/`
- `i18n.go`：go-i18n bundle 与 `T(c, key, args...)` 工具
- `keys.go`：键名常量
- `locales/`：JSON 资源（en、zh）

### 2.12 `oauth/`
- `provider.go`：Provider 抽象接口
- `registry.go`：自定义 OAuth provider 注册中心
- `types.go`：OAuth 通用类型
- `generic.go`：通用 OIDC 实现
- `github.go`、`discord.go`、`linuxdo.go`、`oidc.go`：内置 provider 子实现

### 2.13 `pkg/`
- `billingexpr/`：表达式计费（**修改前必读 `expr.md`**）
- `cachex/`：HybridCache 双层缓存
- `ionet/`：IO.Net SaaS 集成
- `perf_metrics/`：模型性能指标采集

### 2.14 `types/`
- `relay_format.go`：`RelayFormatOpenAI/Claude/Gemini/Embedding/Audio/Image/Realtime/Rerank/Responses/Compaction/Video/MJ/Suno`
- `error.go`、`channel_error.go`：错误抽象
- `file_source.go`、`file_data.go`：上传文件来源
- `price_data.go`：价格快照
- `request_meta.go`：请求元信息
- `rw_map.go`、`set.go`：并发安全集合

### 2.15 `constant/`
枚举常量：`api_type.go`、`channel.go`、`context_key.go`、`cache_key.go`、`endpoint_type.go`、`env.go`、`finish_reason.go`、`midjourney.go`、`multi_key_mode.go`、`setup.go`、`task.go`、`waffo_pay_method.go`、`azure.go`

---

## 3. Default 前端（part: `web-default`）注释源码树

```
web/default/
├─ src/
│  ├─ main.tsx                React 19 入口，加载 TanStack Router、QueryClient、Theme
│  ├─ routeTree.gen.ts        TanStack Router 自动生成
│  ├─ env.d.ts                环境类型
│  ├─ tanstack-table.d.ts     扩展 TanStack Table 类型
│  ├─ assets/                 图标/插画
│  ├─ components/             通用组件
│  │  ├─ ui/                  shadcn 风格 UI 原语（55+ 个）
│  │  ├─ ai-elements/         AI 对话原子（actions/branch/conversation/...）
│  │  ├─ layout/              app-header/sidebar/footer/public-* 等
│  │  └─ data-table/          表格基础设施
│  ├─ config/                 静态配置
│  ├─ context/                React context（少量）
│  ├─ features/               业务模块（每个自包含 index/api/components/hooks/types）
│  │  ├─ about / auth / channels / chat / dashboard / errors / home / keys
│  │  ├─ legal / models / playground / pricing / profile / rankings
│  │  ├─ redemption-codes / setup / subscriptions / system-settings
│  │  ├─ usage-logs / users / wallet
│  ├─ hooks/                  全局 hook（use-system-config 等）
│  ├─ i18n/                   i18next（locales/zh.json 等 6 语言）
│  ├─ lib/                    api/客户端、theme、utils、cache、oauth、time…
│  ├─ routes/                 TanStack 文件路由（含 (auth)/_authenticated/system-settings 等）
│  ├─ stores/                 Zustand store（auth/notification/system-config）
│  └─ styles/                 Tailwind 入口与主题变量
├─ rsbuild.config.ts          Rsbuild 打包配置
├─ tailwind.config.ts
├─ package.json               Bun 依赖
└─ tsconfig.json
```

详见 [component-inventory-web-default.md](component-inventory-web-default.md)。

---

## 4. Classic 前端（part: `web-classic`）注释源码树

```
web/classic/
├─ src/
│  ├─ main.jsx                React 18 入口
│  ├─ App.jsx                 react-router-dom 路由树（lazy + 守卫）
│  ├─ assets/                 静态资源
│  ├─ components/             auth / common / dashboard / layout / settings / setup / table / playground
│  ├─ context/                Status / Theme / User（Context + reducer）
│  ├─ helpers/                api/auth/passkey/render/secureApiCall/quota/...
│  ├─ hooks/                  按业务域分组（channels/dashboard/playground/...）
│  ├─ i18n/                   i18next（与 default 共享 6 语言）
│  ├─ pages/                  顶级页面 + Setting/* 子页面
│  └─ services/               secureVerification 等服务
├─ vite.config.js
└─ package.json
```

详见 [component-inventory-web-classic.md](component-inventory-web-classic.md)。

---

## 5. Electron 外壳（part: `electron`）注释源码树

```
electron/
├─ main.js                    主进程（窗口 + 托盘 + Go 二进制托管 + 错误诊断）
├─ preload.js                 contextBridge 暴露 window.electron
├─ create-tray-icon.js        一次性托盘图标生成工具
├─ package.json               electron-builder + 平台 extraResources
├─ icon.png / tray-icon-windows.png / tray-iconTemplate*.png
└─ entitlements.mac.plist     macOS 沙盒权限（出现在 build.mac.entitlements）
```

详见 [component-inventory-electron.md](component-inventory-electron.md)。

---

## 6. 工作流与运维相关目录

- `.github/`：CI workflows、issue/PR 模板、CODEOWNERS
- `docs/`：项目文档（含本次扫描产出）
- `_bmad/` / `_bmad-output/`：BMAD 工作流配置与产出
- `.agents/` / `.claude/`：本地 IDE/Claude Code 配置
- `bin/`：构建产物缓存
- `Dockerfile*` / `docker-compose*.yml` / `new-api.service` / `makefile` / `.env.example`：运行/部署相关，详见 `deployment-guide.md`
