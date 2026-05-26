# 数据模型 — 服务端 (Go Backend)

> 项目部分：`server`（Go / GORM v2）
> 主入口：[model/main.go](../model/main.go)
> 文档生成时间：2026-05-26

---

## 1. 数据库连接策略

`model/main.go::chooseDB` 按环境变量切换驱动：

| 环境变量 | 取值 | 驱动 |
|----------|------|------|
| `SQL_DSN` | 包含 `mysql://` 或 MySQL DSN | gorm.io/driver/mysql |
| `SQL_DSN` | 包含 `postgres://` 或 `postgresql://` | gorm.io/driver/postgres |
| 未设置或 `*.db` | SQLite | github.com/glebarez/sqlite |
| `LOG_SQL_DSN` | 同上 | 可独立指向日志数据库 |

`initCol()` 在连接建立后初始化跨库列名：

- `commonGroupCol` / `commonKeyCol`：保留字 `group` / `key` 的引号形式（PostgreSQL 用 `"…"`，MySQL/SQLite 用 `` `…` ``）
- `commonTrueVal` / `commonFalseVal`：布尔常量字面量（`true/false` vs `1/0`）
- `logGroupCol` / `logKeyCol`：日志库可能与主库不同时另算

> **跨库约束**（CLAUDE.md Rule 2）：所有 SQL 必须同时兼容 SQLite、MySQL ≥5.7.8、PostgreSQL ≥9.6。原始 SQL 中遇到 `group`、`key` 等保留字必须使用上述变量。

---

## 2. 核心实体

下面按业务域汇总，所有实体均位于 [model/](../model/) 目录。字段以 `gorm:"…"` 标注的为关键约束/索引。

### 2.1 用户与权限

#### `User` — [model/user.go](../model/user.go:24)
- 主键：`Id`
- 业务唯一：`Username`、`AccessToken`、`AffCode`
- 索引：`DisplayName`、`Email`、`GitHubId`、`DiscordId`、`OidcId`、`WeChatId`、`TelegramId`、`LinuxDOId`、`InviterId`、`StripeCustomer`、`DeletedAt`
- 关键字段：`Role`（admin/common）、`Status`（enabled/disabled）、`Quota`/`UsedQuota`/`RequestCount`、`Group`、`AffCount`/`AffQuota`/`AffHistoryQuota`、`Setting`(JSON)、`Remark`、`CreatedAt`、`LastLoginAt`
- 软删除：`DeletedAt`

#### `Token` — [model/token.go](../model/token.go:14)
- 主键：`Id`
- 唯一：`Key`（varchar(128)）
- 索引：`UserId`、`Name`、`DeletedAt`
- 关键字段：`Status`、`ExpiredTime`（-1 永久）、`RemainQuota`/`UsedQuota`、`UnlimitedQuota`、`ModelLimitsEnabled`/`ModelLimits`、`AllowIps`、`Group`、`CrossGroupRetry`

#### `PasskeyCredential` — [model/passkey.go](../model/passkey.go:23)
WebAuthn 凭证。一用户一 passkey（`UserID` 唯一索引）。`CredentialID` 唯一。

#### `TwoFA` / `TwoFABackupCode` — [model/twofa.go](../model/twofa.go:14)
2FA TOTP 配置；`Secret` 不暴露 JSON。`FailedAttempts`/`LockedUntil` 用于锁定。

#### `CustomOAuthProvider` / `UserOAuthBinding` — [model/custom_oauth_provider.go](../model/custom_oauth_provider.go), [model/user_oauth_binding.go](../model/user_oauth_binding.go:11)
- 自定义 OIDC/OAuth provider 元数据 + 访问策略（`accessPolicyPayload`，支持 `eq/ne/gt/gte/lt/lte/in/not_in/contains/not_contains/exists/not_exists` 与逻辑分组）。
- `UserOAuthBinding`：`(user_id, provider_id)` 唯一，`(provider_id, provider_user_id)` 唯一 — 一个用户每 provider 至多一条；同一 OAuth 账号在同一 provider 内仅能绑定一个用户。

#### `Setup` — [model/setup.go](../model/setup.go:3)
单例配置：版本与初始化时间。系统首次启动时如不存在 root 用户则自动创建（`username=root, password=123456`）。

### 2.2 渠道与能力

#### `Channel` — [model/channel.go](../model/channel.go:23)
AI 上游渠道配置。
- 索引：`Name`、`Tag`
- 关键字段：`Type`（渠道类型枚举，`constant.ChannelType*`）、`Key`、`Status`、`Weight`、`Priority`、`AutoBan`、`BaseURL`、`Other`、`Models`、`Group`、`UsedQuota`、`ModelMapping`(JSON)、`StatusCodeMapping`、`OtherInfo`、`Setting`、`ParamOverride`、`HeaderOverride`、`Remark`、`OpenAIOrganization`、`TestModel`、`TestTime`、`ResponseTime`、`Balance`、`BalanceUpdatedTime`
- 嵌入：`ChannelInfo`（多 Key 模式状态：`IsMultiKey`、`MultiKeySize`、`MultiKeyStatusList map[int]int`、`MultiKeyDisabledReason map[int]string`、`MultiKeyDisabledTime map[int]int64`、`MultiKeyPollingIndex`、`MultiKeyMode`），存为 `type:json` 列
- `OtherSettings` 列名 `settings`：保存 azure 版本等无需检索的辅助配置（`dto.ChannelOtherSettings`）
- 缓存：`Keys []string`（GORM 忽略，运行时填充）

#### `Ability` — [model/ability.go](../model/ability.go:16)
渠道-模型-分组三元组（路由表）。
- 复合主键：`(Group, Model, ChannelId)`
- 索引：`ChannelId`、`Priority`、`Weight`、`Tag`
- 用于 distributor 在 `(group, model)` 下选择候选渠道。

#### `Model` — [model/model_meta.go](../model/model_meta.go:24)
模型元数据（前端展示与价格）。
- 唯一：`(ModelName, DeletedAt)`
- 关联：`VendorID`
- 关键字段：`Description`、`Icon`、`Tags`、`Endpoints`(JSON)、`Status`、`SyncOfficial`
- 运行时（GORM 忽略）：`BoundChannels []BoundChannel{Name,Type}`、`EnableGroups []string`、`QuotaTypes []int`

#### `Vendor` — [model/vendor_meta.go](../model/vendor_meta.go:15)
模型供应商。
- 唯一：`(Name, DeletedAt)`
- `Icon` 使用 `@lobehub/icons` 名称

#### `MissingModels` — [model/missing_models.go](../model/missing_models.go)
登记当前未在 model meta 中的模型。

### 2.3 计费与订阅

#### `TopUp` — [model/topup.go](../model/topup.go:14)
充值订单。
- 唯一：`TradeNo`
- 索引：`UserId`
- 关键字段：`Amount`/`Money`、`PaymentMethod`(`stripe|creem|waffo|waffo_pancake|balance`)、`PaymentProvider`（含 `epay`）、`Status`、时间戳

#### `Redemption` — [model/redemption.go](../model/redemption.go:14)
兑换码。`Key` char(32) 唯一；`UserId`/`UsedUserId`、`ExpiredTime`(0 不过期)。

#### `SubscriptionPlan` / `SubscriptionOrder` / `UserSubscription` — [model/subscription.go](../model/subscription.go:146)
订阅三件套，全部走 `cachex.HybridCache`（Redis + 内存 LRU）。
- `SubscriptionPlan`：定价（`PriceAmount`/`Currency`）、周期（`DurationUnit` ∈ year/month/day/hour/custom，`DurationValue`，`CustomSeconds`）、外部映射（`StripePriceId`、`CreemProductId`、`WaffoPancakeProductId`）、`MaxPurchasePerUser`、`UpgradeGroup`、`TotalAmount`、`QuotaResetPeriod`(never/daily/weekly/monthly/custom)
- `SubscriptionOrder`：与 `TopUp` 平行的订阅订单，含 `ProviderPayload`(text)
- `UserSubscription`：`AmountTotal`/`AmountUsed`、`StartTime`/`EndTime`、`Status`(active/expired/cancelled)、`Source`(order/admin)、`LastResetTime`/`NextResetTime`、`UpgradeGroup`/`PrevUserGroup`
- 缓存键 namespace：`new-api:subscription_plan:v1`、`new-api:subscription_plan_info:v1`
- 默认 TTL：plan 300s / info 120s（可经 `SUBSCRIPTION_PLAN_CACHE_TTL`、`SUBSCRIPTION_PLAN_INFO_CACHE_TTL` 调整）

#### `Pricing` 系列 — [model/pricing.go](../model/pricing.go), [model/pricing_default.go](../model/pricing_default.go), [model/pricing_refresh.go](../model/pricing_refresh.go)
模型计费倍率配置。配合 [setting/ratio_setting/](../setting/ratio_setting/) 与 [pkg/billingexpr/](../pkg/billingexpr/) 提供分级/动态定价（参见 [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)）。

#### `Checkin` — [model/checkin.go](../model/checkin.go:14)
签到。`(UserId, CheckinDate)` 唯一（一人一日一签）。

### 2.4 日志与统计

#### `Log` — [model/log.go](../model/log.go:19)
通用日志（消费/管理/系统/错误/退款/充值）。
- 复合索引：`(CreatedAt, Id)`、`(UserId, Id)`、`(Username, ModelName)`、`(CreatedAt, Type)`、`UpstreamRequestId`、`RequestId`
- 类型常量：`LogTypeUnknown=0, Topup=1, Consume=2, Manage=3, System=4, Error=5, Refund=6`
- 关键字段：`UserId`/`Username`、`TokenId`/`TokenName`、`ChannelId`/`ChannelName`(read-only join)、`Group`、`Ip`、`ModelName`、`Quota`、`PromptTokens`/`CompletionTokens`、`UseTime`、`IsStream`、`Other`、`RequestId`、`UpstreamRequestId`

#### `QuotaData` — [model/usedata.go](../model/usedata.go:13)
看板柱状图数据，按 `(UserID, ModelName, CreatedAt)` 聚合。带 `(ModelName, Username)` 复合索引。后台 `UpdateQuotaData()` 周期把内存缓存 flush 到表。

#### `RankingsData` 等 — [model/usedata_rankings.go](../model/usedata_rankings.go), [model/perf_metric.go](../model/perf_metric.go)
排行榜与性能指标聚合表。

### 2.5 多媒体任务

#### `Task` — [model/task.go](../model/task.go:44)
统一异步任务（视频/Suno 等）。
- 主键：`ID`（auto-increment）
- 索引：`CreatedAt`、`TaskID`、`Platform`、`UserId`、`ChannelId`、`Action`、`Status`、`Progress`、`SubmitTime`/`StartTime`/`FinishTime`
- 状态枚举：`NOT_START / SUBMITTED / QUEUED / IN_PROGRESS / FAILURE / SUCCESS / UNKNOWN`，到 `dto.VideoStatus*` 的映射

#### `Midjourney` — [model/midjourney.go](../model/midjourney.go:3)
Midjourney 专用任务表（与 `Task` 并存，历史原因）。`mj_id`/`status`/`progress`/`submit_time`/`start_time`/`finish_time` 索引。

### 2.6 其他配置

#### `Option` — [model/option.go](../model/option.go:18)
全局 KV 配置（前端 / 渠道 / 邮件 / 计费等）。`Key` 主键。`InitOptionMap` 初始化时从 [setting/](../setting/) 各子包同步默认值，统一通过 `common.OptionMap`（带读写锁）暴露。

#### `PrefillGroup` — [model/prefill_group.go](../model/prefill_group.go:20)
可复用的命名集合（模型组/标签组/端点组）。`Items` 字段使用 `JSONValue`（自定义 `json.RawMessage` 包装，实现 `driver.Valuer` / `sql.Scanner`，兼容 []byte/string 不同驱动返回）。

#### `Deployment` 系列 — [controller/deployment.go](../controller/deployment.go)、IO.Net 集成
模型部署元数据（硬件类型、副本、价格估算）。配合 [pkg/ionet/](../pkg/ionet/) 调用上游。

---

## 3. 跨库注意事项（实战）

### 3.1 列名引号
- 使用 `commonGroupCol` / `commonKeyCol` 处理 `group` / `key`，例如：
  ```go
  DB.Where(commonGroupCol+" = ? and enabled = ?", group, true)
  ```
- 日志相关 SQL 使用 `logGroupCol` / `logKeyCol`（日志库可能不同）。

### 3.2 布尔/NULL
- 用 `commonTrueVal` / `commonFalseVal` 拼字面量（`true/false` vs `1/0`）。

### 3.3 主键
- 默认依赖 GORM 的自增（不要直接写 `AUTO_INCREMENT` / `SERIAL`）。
- 复合主键示例：`Ability` 用 `gorm:"primaryKey;autoIncrement:false"` 拼装。

### 3.4 JSON 列
- 文本型 JSON 用 `gorm:"type:text"` + 应用层自管 marshal/unmarshal（必须走 `common/json.go`，CLAUDE.md Rule 1）。
- `Channel.ChannelInfo` 用 `gorm:"type:json"`，依赖驱动支持。`PrefillGroup.Items` 自管 `JSONValue`。

### 3.5 软删除
- 通用 `gorm.DeletedAt` 索引；与唯一索引组合时（如 `Model.ModelName + DeletedAt`、`Vendor.Name + DeletedAt`）以保证软删后名称可复用。

### 3.6 自定义表名
- `Checkin` → `checkins`
- `UserOAuthBinding` → `user_oauth_bindings`

---

## 4. 关键索引概览（性能相关）

| 表 | 索引 | 用途 |
|----|------|------|
| logs | `(created_at, id)`、`(user_id, id)`、`(created_at, type)`、`(username, model_name)` | 时间倒序分页、用户/类型/模型筛选 |
| logs | `idx_logs_request_id`、`idx_logs_upstream_request_id` | 排障关联 |
| user_subscriptions | `(user_id, status, end_time)` (`idx_user_sub_active`) | 当前活跃订阅查询 |
| quota_data | `(model_name, username)`、`(created_at)` | 看板/排名查询 |
| tokens | `key`(uniq)、`name`、`user_id` | 鉴权与列表 |
| channels | `name`、`tag` | 列表/标签批操作 |
| abilities | `(group, model, channel_id)` PK + `priority`、`weight`、`tag` | 路由 |
| tasks | `task_id`、`(status, progress)`、`platform` | 任务查询 |

---

## 5. 启动顺序

`main.go` → `model.InitDB()` → 自动迁移所有结构体 → `createRootAccountIfNeed` → `CheckSetup` → 初始化各类缓存（`subscriptionPlanCache`、`token cache`、`channel cache` 等）。

> 详细控制器与 DTO 字段映射参见 [api-contracts-server.md](api-contracts-server.md)；上游适配器使用的 RelayInfo / ChannelMeta 等运行时类型见 [relay/common/relay_info.go](../relay/common/relay_info.go)。
