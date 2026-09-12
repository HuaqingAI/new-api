# 组件清单 — Classic Web (React 18 / Vite / Semi Design)

> 项目部分：`web-classic`
> 根路径：[web/classic/](../web/classic/)
> 文档生成时间：2026-05-26
> 入口：`web/classic/src/main.jsx`、`web/classic/src/App.jsx`

---

## 1. 技术栈摘要

- React 18.2 + JavaScript（少量 TS）
- 打包：Vite 5.2（dev/build/preview，开发态代理 `/api`、`/v1`、`/mj`、`/pg` 到 `http://localhost:3000`）
- 路由：react-router-dom v6（声明式 `Routes/Route` + 自定义 `PrivateRoute/AdminRoute/AuthRedirect`）
- UI：@douyinfe/semi-ui 2.69+（Semi Design 组件库），@douyinfe/semi-icons
- 国际化：i18next + react-i18next 13 + i18next-browser-languagedetector（同 default 主题共享 zh/en/fr/ru/ja/vi）
- 数据请求：axios（封装在 `helpers/api.js`）
- 表格：Semi Table（默认）+ 自封装 `CardTable/CardPro`
- 富文本/可视化：marked、mermaid、katex、prismjs、@visactor/vchart
- 通知：react-toastify
- 状态：React Context + useReducer（无 Redux/Zustand）
- 包管理：Bun（i18n 工具）+ npm/yarn（兼容 Vite）

## 2. 路由结构（[App.jsx](../web/classic/src/App.jsx)）

主路由由 `react-router-dom` 的 `Routes/Route` 渲染，懒加载（`lazy()`）应用于较重页面。

### 路由守卫
- `AuthRedirect`：未登录用户重定向至 `/login`
- `PrivateRoute`：包装需要用户登录的页面
- `AdminRoute`：包装管理员权限页面

### 主要路由表

| 路由 | 组件 | 鉴权 | 说明 |
|------|------|------|------|
| `/` | Home | 公共 | 首页 |
| `/about` | About | 公共 | 关于 |
| `/user-agreement` | UserAgreement | 公共 | 用户协议 |
| `/privacy-policy` | PrivacyPolicy | 公共 | 隐私政策 |
| `/pricing` | Pricing | 公共 | 模型价格 |
| `/login` / `/register` / `/reset` / `/passwordReset` | LoginForm 等 | 未登录 | 鉴权页 |
| `/oauth/:provider` | OAuth2Callback | 公共 | OAuth 回调 |
| `/setup` | Setup | 首次安装 | 安装向导 |
| `/dashboard` | Dashboard | UserAuth | 用户控制台首页 |
| `/user` | User | AdminAuth | 用户管理 |
| `/channel` | Channel | AdminAuth | 渠道管理 |
| `/token` | Token | UserAuth | API 令牌管理 |
| `/redemption` | Redemption | AdminAuth | 兑换码管理 |
| `/topup` | TopUp | UserAuth | 充值 |
| `/log` | Log | UserAuth | 用量/管理日志 |
| `/midjourney` | Midjourney | UserAuth | Midjourney 任务 |
| `/task` | Task | UserAuth | 视频/Suno 任务 |
| `/setting` | Setting | UserAuth/AdminAuth/RootAuth | 综合设置（多 Tab） |
| `/chat` 、 `/chat2link` | Chat / Chat2Link | UserAuth | 内嵌 Chat |
| `/playground` | Playground | UserAuth | 模型试玩 |
| `/pricing/:modelName` | ModelPricing | 公共 | 模型详情 |
| `/model` | Model | AdminAuth | 模型元数据 |
| `/model-deployment` | ModelDeployment | AdminAuth | IO.Net 部署 |
| `/subscription` | Subscription | UserAuth | 订阅 |
| `/403` / `/404` | Forbidden / NotFound | — | 错误页 |

---

## 3. 页面目录（[pages/](../web/classic/src/pages/))

### 3.1 顶级页面（默认导出 `index.jsx`）

| 页面 | 路径 | 说明 |
|------|------|------|
| About | [pages/About/index.jsx](../web/classic/src/pages/About/index.jsx) | 关于 |
| Channel | [pages/Channel/index.jsx](../web/classic/src/pages/Channel/index.jsx) | 渠道管理 |
| Chat、Chat2Link | [pages/Chat/](../web/classic/src/pages/Chat/), [pages/Chat2Link/](../web/classic/src/pages/Chat2Link/) | 内嵌聊天 |
| Dashboard | [pages/Dashboard/index.jsx](../web/classic/src/pages/Dashboard/index.jsx) | 控制台首页 |
| Forbidden、NotFound | [pages/Forbidden/](../web/classic/src/pages/Forbidden/), [pages/NotFound/](../web/classic/src/pages/NotFound/) | 错误页 |
| Home | [pages/Home/index.jsx](../web/classic/src/pages/Home/index.jsx) | 首页 |
| Log | [pages/Log/index.jsx](../web/classic/src/pages/Log/index.jsx) | 日志 |
| Midjourney | [pages/Midjourney/index.jsx](../web/classic/src/pages/Midjourney/index.jsx) | MJ 任务 |
| Model | [pages/Model/index.jsx](../web/classic/src/pages/Model/index.jsx) | 模型元数据 |
| ModelDeployment | [pages/ModelDeployment/index.jsx](../web/classic/src/pages/ModelDeployment/index.jsx) | 部署管理 |
| Playground | [pages/Playground/index.jsx](../web/classic/src/pages/Playground/index.jsx) | 试玩 |
| Pricing | [pages/Pricing/index.jsx](../web/classic/src/pages/Pricing/index.jsx) | 价格 |
| PrivacyPolicy、UserAgreement | [pages/PrivacyPolicy/](../web/classic/src/pages/PrivacyPolicy/), [pages/UserAgreement/](../web/classic/src/pages/UserAgreement/) | 法律 |
| Redemption | [pages/Redemption/index.jsx](../web/classic/src/pages/Redemption/index.jsx) | 兑换码 |
| Setup | [pages/Setup/index.jsx](../web/classic/src/pages/Setup/index.jsx) | 安装向导 |
| Subscription | [pages/Subscription/index.jsx](../web/classic/src/pages/Subscription/index.jsx) | 订阅 |
| Task | [pages/Task/index.jsx](../web/classic/src/pages/Task/index.jsx) | 视频/Suno 任务 |
| Token | [pages/Token/index.jsx](../web/classic/src/pages/Token/index.jsx) | 令牌管理 |
| TopUp | [pages/TopUp/index.js](../web/classic/src/pages/TopUp/index.js) | 充值 |
| User | [pages/User/index.jsx](../web/classic/src/pages/User/index.jsx) | 用户管理 |

### 3.2 设置子页面 [pages/Setting/](../web/classic/src/pages/Setting/)

设置中心采用 Tab 切换，子文件按业务域组织：

- **Chat**：[Setting/Chat/SettingsChats.jsx](../web/classic/src/pages/Setting/Chat/SettingsChats.jsx)
- **Dashboard**：Announcements、APIInfo、DataDashboard、FAQ、UptimeKuma
- **Drawing**：SettingsDrawing（绘图）
- **Model**：SettingClaudeModel、SettingGeminiModel、SettingGrokModel、SettingGlobalModel、SettingModelDeployment
- **Operation**：ChannelAffinity、Checkin、CreditLimit、General、HeaderNavModules、Log、Monitoring、SensitiveWords、SidebarModulesAdmin
- **Payment**：SettingsGeneralPayment、SettingsPaymentGateway、SettingsPaymentGatewayCreem、SettingsPaymentGatewayStripe、SettingsPaymentGatewayWaffo、SettingsPaymentGatewayWaffoPancake
- **Performance**：SettingsPerformance
- **Personal**：SettingsSidebarModulesUser
- **RateLimit**：SettingsRequestRateLimit
- **Ratio**：GroupRatioSettings、ModelPricingCombined、ModelRatioSettings、ModelSettingsVisualEditor、ModelRationNotSetEditor、ToolPriceSettings、UpstreamRatioSync
  - `components/`：AutoGroupList、GroupGroupRatioRules、GroupSpecialUsableRules、GroupTable、ModelPricingEditor、TieredPricingEditor、requestRuleExpr
  - `hooks/useModelPricingEditorState.js`

---

## 4. 组件目录（[components/](../web/classic/src/components/))

### 4.1 鉴权 `components/auth/`
LoginForm、OAuth2Callback、PasswordResetConfirm、PasswordResetForm、RegisterForm、TwoFAVerification

### 4.2 公共 `components/common/`
- `DocumentRenderer/`：通用文档渲染器
- `ErrorBoundary.jsx`
- `examples/ChannelKeyViewExample.jsx`
- `logo/`：LinuxDoIcon、OIDCIcon、WeChatIcon
- `markdown/MarkdownRenderer.jsx`
- `modals/`：RiskAcknowledgementModal、SecureVerificationModal、TwoFactorAuthModal
- `ui/`：CardPro、CardTable、ChannelKeyDisplay、CompactModeToggle、JSONEditor、Loading、RenderUtils、ScrollableContainer、SelectableButtonGroup

### 4.3 仪表盘 `components/dashboard/`
AnnouncementsPanel、ApiInfoPanel、ChartsPanel、DashboardHeader、FaqPanel、StatsCards、UptimePanel、`modals/SearchModal`、`index.jsx`

### 4.4 布局 `components/layout/`
- Footer、PageLayout、SetupCheck、SiderBar、NoticeModal
- `components/SkeletonWrapper.jsx`
- **HeaderBar 子模块** `headerbar/`：ActionButtons、HeaderLogo、LanguageSelector、MobileMenuButton、Navigation、NewYearButton、NotificationButton、ThemeToggle、UserArea、`index.jsx` 聚合

### 4.5 模型部署 `components/model-deployments/`
DeploymentAccessGuard

### 4.6 Playground `components/playground/`
ChatArea、CodeViewer、ConfigManager、CustomInputRender、CustomRequestEditor、DebugPanel、FloatingButtons、ImageUrlInput、MessageActions、MessageContent、OptimizedComponents、ParameterControl、SettingsPanel、SSEViewer、ThinkingContent、`configStorage.js`、`index.js`

### 4.7 设置组件 `components/settings/`
- 顶级：ChannelSelectorModal、ChatsSetting、CustomOAuthSetting、DashboardSetting、DrawingSetting、HttpStatusCodeRulesInput、ModelDeploymentSetting、ModelSetting、OperationSetting、OtherSetting、PaymentSetting、PerformanceSetting、PersonalSetting、RateLimitSetting、RatioSetting、SystemSetting
- **个人设置子组件** `personal/`：
  - `cards/`：AccountManagement、CheckinCalendar、ModelsList、NotificationSettings、PreferencesSettings
  - `components/`：TwoFASetting、UserInfoHeader
  - `modals/`：AccountDeleteModal、ChangePasswordModal、EmailBindModal、WeChatBindModal

### 4.8 安装向导 `components/setup/`
SetupWizard、`components/StepNavigation`、`components/steps/(AdminStep, CompleteStep, DatabaseStep, ...)`

### 4.9 表格 `components/table/`
按业务域聚合，每个子目录通常包含 `Actions`（行操作）/`ColumnDefs`（列定义）/`Filters`（筛选器）/`Table`（主体）/`Tabs`（多 Tab）/`modals`（弹窗）：

| 子目录 | 用途 |
|--------|------|
| `channels/` | 渠道列表 |
| `mj-logs/` | Midjourney 日志 |
| `model-deployments/` | 模型部署列表 |
| `model-pricing/` | 模型价格表 |
| `models/` | 模型元数据 |
| `redemptions/` | 兑换码 |
| `subscriptions/` | 订阅 |
| `task-logs/` | 异步任务日志（视频/Suno） |

---

## 5. Hooks（[hooks/](../web/classic/src/hooks/))

按业务领域分组，每组通常包含 `use*Data`（数据加载与状态）与场景专用 hook：

| 域 | 文件 |
|----|------|
| `channels/` | useChannelsData、useChannelUpstreamUpdates、upstreamUpdateUtils |
| `chat/` | useTokenKeys |
| `common/` | useContainerWidth、useHeaderBar、useIsMobile、useMinimumLoadingTime、useNavigation、useNotifications、useSecureVerification、useSidebar、useSidebarCollapsed、useTableCompactMode、useUserPermissions |
| `dashboard/` | useDashboardCharts、useDashboardData、useDashboardStats |
| `mj-logs/` | useMjLogsData |
| `model-deployments/` | useDeploymentResources、useDeploymentsData、useEnhancedDeploymentActions、useModelDeploymentSettings |
| `model-pricing/` | useModelPricingData、usePricingFilterCounts |
| `models/` | useModelsData |
| `playground/` | useApiRequest、useDataLoader、useMessageActions、useMessageEdit、usePlaygroundState、useSyncMessageAndCustomBody |
| `redemptions/` | useRedemptionsData |
| `subscriptions/` | useSubscriptionsData |
| `task-logs/` | useTaskLogsData |
| `tokens/` | useTokensData |
| `usage-logs/` | useUsageLogsData |
| `users/` | useUsersData |

---

## 6. Helpers（[helpers/](../web/classic/src/helpers/))

- `api.js`：axios 实例 + 拦截器
- `auth.jsx`：鉴权辅助（RBAC、Redirect）
- `base64.js`、`boolean.js`、`data.js`、`history.js`、`utils.jsx`
- `dashboard.jsx`、`log.js`、`subscriptionFormat.js`：业务格式化
- `passkey.js`：WebAuthn 客户端
- `quota.js`：额度计算
- `render.jsx`：通用渲染函数
- `secureApiCall.js`：二次安全验证 API 包装
- `statusCodeRules.js`：HTTP 状态码规则
- `token.js`：本地 token 存取
- `index.js`：barrel re-export

---

## 7. Context（[context/](../web/classic/src/context/))

经典的 React Context + useReducer 模式：

- `Status/index.jsx` + `Status/reducer.js`：系统配置与公告状态
- `Theme/index.jsx`：主题（亮/暗 + Semi 主题切换）
- `User/index.jsx` + `User/reducer.js`：当前登录用户信息与角色

---

## 8. Services & 其他

- `services/secureVerification.js`：二次验证服务
- 国际化资源同 default 主题位于 `web/classic/src/i18n/locales/`（i18next 加载）
- 静态资源：`web/classic/src/assets/`、`web/classic/public/`

---

## 9. 与 Default 主题对照

| 维度 | Default | Classic |
|------|---------|---------|
| React | 19 | 18.2 |
| 打包 | Rsbuild | Vite 5 |
| 路由 | TanStack Router（文件路由） | react-router-dom v6（声明式） |
| 状态 | Zustand + TanStack Query | Context + useReducer |
| UI 库 | Base UI + Tailwind + shadcn 风格 | Semi Design |
| 表单 | React Hook Form + Zod | Semi Form |
| 表格 | TanStack Table + Virtual | Semi Table + CardTable/CardPro |

两个主题共用后端 `/api/**`、`/v1/**`、`/dashboard/**` 接口，由 [router/main.go](../router/main.go) 通过 `FRONTEND_BASE_URL`/主题切换决定 embed 的前端目录。
