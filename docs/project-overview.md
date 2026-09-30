# 项目总览 — new-api

> 文档生成时间：2026-05-26
> 项目类型：monorepo，4 part
> 主分支：`main` / 开发分支：`dev`

---

## 1. 项目定位

**new-api** 是一个 **AI API 网关 / 代理**，由 [QuantumNous](https://github.com/QuantumNous/new-api) 维护：

- 把 **40+ 上游 AI 提供商**（OpenAI / Claude / Gemini / Azure / AWS Bedrock / Vertex / 阿里 / 百度 / 腾讯 / 火山 / 讯飞 / 智谱 / 月之暗面 / DeepSeek / Mistral / Cohere / Replicate / Cloudflare / xAI / Ollama / OpenRouter / Perplexity / Coze / Dify / 即梦 / 可灵 / Codex / 等）聚合在一组统一的 OpenAI 兼容 API 之后
- 内置 **用户管理 / 权限 / 计费 / 限流 / 渠道管理 / 模型可见性 / 订阅 / 充值 / 兑换码 / OAuth / Passkey / 2FA / 安装向导 / 控制台**
- 支持 **流式 / 多模态 / 图像 / 音频 / 视频 / 推理 / 工具调用 / Realtime WebSocket** 全套接口形态
- 一份单二进制即可启动（前端通过 `go:embed` 嵌入），亦支持 Docker / systemd / Electron 桌面端
- 4 个 part 共存：`server`（Go）+ `web-default`（React 19）+ `web-classic`（React 18）+ `electron`

---

## 2. 4-Part 架构概览

```
┌────────────────────┐
│  electron 桌面外壳  │ ← 包装 server，端口 3000，托盘 + 数据目录隔离
└──────────┬─────────┘
           │ spawn
           ▼
┌────────────────────────────────────────────────┐
│              server (Go 后端)                   │
│  Router → Controller → Service → Model         │
│  + relay/channel (40 个 provider 适配器)        │
│  + go:embed 两套前端 dist                       │
└──────────┬──────────────────────┬──────────────┘
           │ static               │ API
           ▼                      ▲
   ┌──────────────┐       ┌──────────────┐
   │ web-default  │       │ web-classic  │
   │ React 19     │       │ React 18     │
   │ Rsbuild      │       │ Vite         │
   │ Base UI      │       │ Semi Design  │
   └──────────────┘       └──────────────┘
   （二选一，由 DB Option `Theme` 决定运行期返回哪套）
```

| Part | 路径 | 入口 | 主要技术 |
|------|------|------|---------|
| `server` | 仓库根（`controller/`、`service/`、`relay/`、`router/`、`middleware/`、`model/` 等） | [main.go](../main.go) | Go 1.25.1 / Gin / GORM v2 |
| `web-default` | [web/default/](../web/default/) | `src/main.tsx` | React 19 / TypeScript / Rsbuild / Base UI / Tailwind |
| `web-classic` | [web/classic/](../web/classic/) | `src/main.jsx` | React 18 / Vite / Semi Design |
| `electron` | [electron/](../electron/) | `main.js` | Electron 39 / electron-builder |

---

## 3. 数据库支持矩阵（CLAUDE.md Rule 2）

| DB | 最低版本 | DSN 形式 | 备注 |
|----|---------|---------|------|
| SQLite | — | 未设置 `SQL_DSN` 或 `.db` 后缀 | glebarez/sqlite，无 CGO |
| MySQL | 5.7.8+ | `user:pwd@tcp(host:3306)/db?parseTime=true` | UTF8MB4 |
| PostgreSQL | 9.6+ | `postgresql://user:pwd@host:5432/db` | 跨 schema 注意列名引号 |

**所有数据库代码必须同时兼容三者**。详细约束见 [CLAUDE.md](../CLAUDE.md) Rule 2 与 [architecture-server.md §7](architecture-server.md)。

---

## 4. 关键依赖（节选）

### Go 后端
- `gin-gonic/gin@v1.9.1`
- `gorm.io/gorm@v1.25.2`（含 mysql / postgres / glebarez/sqlite 驱动）
- `go-redis/redis/v8`
- `go-webauthn/webauthn`
- `golang-jwt/jwt/v5`
- `pquerna/otp`（TOTP）
- `aws/aws-sdk-go-v2/service/bedrockruntime`
- `pkoukk/tiktoken-go`（token 计数）
- `expr-lang/expr`（计费表达式，详见 [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)）
- `stripe/stripe-go/v81`、`go-epay`、Creem / Waffo / Waffo Pancake SDK

### Web Default
- `react@19`、`@tanstack/react-router`、`@tanstack/react-query`、`@tanstack/react-table`、`@tanstack/react-virtual`
- `@base-ui-components/react`、`tailwindcss`、`zustand`、`react-hook-form`、`zod`
- `i18next` / `react-i18next` / `i18next-browser-languagedetector`
- `marked`、`prismjs`、`katex`、`mermaid`
- `@visactor/vchart`

### Web Classic
- `react@18.2`、`react-router-dom@v6`
- `@douyinfe/semi-ui@^2.69`、`@douyinfe/semi-icons`
- `i18next` / `react-i18next@13`
- `marked`、`mermaid`、`katex`、`prismjs`
- `@visactor/vchart`、`react-toastify`

### Electron
- `electron@39.8.5`
- `electron-builder@^26.7.0`

---

## 5. 关键设计原则（CLAUDE.md 摘要）

1. **Rule 1**：所有 JSON 操作必须使用 `common/json.go` 包装，禁止直接 import `encoding/json`
2. **Rule 2**：跨 SQLite / MySQL / PostgreSQL 全兼容，使用 `commonGroupCol` / `commonKeyCol` / `commonTrueVal` / `commonFalseVal` 抹平差异
3. **Rule 3**：`web/default/` 用 Bun（`bun install` / `bun run *`）
4. **Rule 4**：新增 channel 时确认 `StreamOptions` 支持，加入 `streamSupportedChannels`
5. **Rule 5**：`new-api` 项目名与 `QuantumNous` 组织名是受保护标识，禁止修改/删除/替换
6. **Rule 6**：转发请求 DTO 的可选标量字段必须用指针 + `omitempty`，保留显式零值语义
7. **Rule 7**：动态计费修改前必读 [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)

---

## 6. 文档地图

| 类别 | 文档 |
|------|------|
| **总览** | 本文档 [project-overview.md](project-overview.md) |
| **架构** | [architecture-server.md](architecture-server.md)、[architecture-web-default.md](architecture-web-default.md)、[architecture-web-classic.md](architecture-web-classic.md)、[architecture-electron.md](architecture-electron.md) |
| **集成** | [integration-architecture.md](integration-architecture.md) |
| **API 契约** | [api-contracts-server.md](api-contracts-server.md) |
| **数据模型** | [data-models-server.md](data-models-server.md) |
| **组件清单** | [component-inventory-web-default.md](component-inventory-web-default.md)、[component-inventory-web-classic.md](component-inventory-web-classic.md)、[component-inventory-electron.md](component-inventory-electron.md) |
| **源码树** | [source-tree-analysis.md](source-tree-analysis.md) |
| **运维** | [deployment-guide.md](deployment-guide.md) |
| **开发** | [development-guide-server.md](development-guide-server.md)、[development-guide-web-default.md](development-guide-web-default.md)、[development-guide-web-classic.md](development-guide-web-classic.md)、[development-guide-electron.md](development-guide-electron.md) |
| **协作** | [contribution-guide.md](contribution-guide.md) |
| **机器可读** | [project-parts.json](project-parts.json) |
| **索引** | [index.md](index.md) |

---

## 7. 仓库快速导航

| 想做什么 | 看哪里 |
|---------|-------|
| 新增一个上游 AI provider | [architecture-server.md §5.4](architecture-server.md) + [relay/channel/adapter.go](../relay/channel/adapter.go) + 任一现有 provider 子目录 |
| 新增一个 HTTP 端点 | [api-contracts-server.md](api-contracts-server.md) + `controller/` + `router/api-router.go` |
| 修改计费规则 | **必读** [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md) + `service/billing.go` + `setting/ratio_setting/` |
| 新增数据表 | [data-models-server.md](data-models-server.md) + `model/` + 跨 DB 兼容性测试 |
| 修改前端（默认主题） | [development-guide-web-default.md](development-guide-web-default.md) + `web/default/src/features/` |
| 修改前端（经典主题） | [development-guide-web-classic.md](development-guide-web-classic.md) + `web/classic/src/pages/` |
| 调整 Electron 行为 | [development-guide-electron.md](development-guide-electron.md) + `electron/main.js` |
| 部署到生产 | [deployment-guide.md](deployment-guide.md) |
| 提 PR | [contribution-guide.md](contribution-guide.md) |
