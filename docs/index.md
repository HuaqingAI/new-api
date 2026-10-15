# 文档主索引 — new-api

> 文档生成时间：2026-05-26
> 项目：[QuantumNous/new-api](https://github.com/QuantumNous/new-api)
> 仓库形态：Monorepo（4 个 part）

---

## 0. 快速导航

| 我是… | 我想… | 看这里 |
|------|------|--------|
| 新人 | 了解项目是什么 | [project-overview.md](project-overview.md) |
| 新人 | 知道项目有哪些规则 | [../CLAUDE.md](../CLAUDE.md) |
| 后端开发 | 改 Go 代码 | [development-guide-server.md](development-guide-server.md) |
| Default 主题前端开发 | 改 React 19 代码 | [development-guide-web-default.md](development-guide-web-default.md) |
| Classic 主题前端开发 | 改 React 18 代码 | [development-guide-web-classic.md](development-guide-web-classic.md) |
| 桌面端开发 | 改 Electron 代码 | [development-guide-electron.md](development-guide-electron.md) |
| 运维 | 部署 / 升级 | [deployment-guide.md](deployment-guide.md) |
| 贡献者 | 提 PR | [contribution-guide.md](contribution-guide.md) |
| 集成方 | 调用 API | [api-contracts-server.md](api-contracts-server.md) |
| 工具链 / AI | 机器读取项目结构 | [project-parts.json](project-parts.json) |

---

## 1. 入门必读

| 文档 | 一句话描述 |
|------|----------|
| [project-overview.md](project-overview.md) | 项目定位、4-part 架构总览、关键依赖、文档地图 |
| [../CLAUDE.md](../CLAUDE.md) | 7 条项目级强约束规则（JSON / 跨 DB / Bun / StreamOptions / 受保护标识 / DTO 指针 / 计费表达式） |
| [contribution-guide.md](contribution-guide.md) | 分支策略、PR 模板、Anti-Slop 检查、代码评审清单 |

---

## 2. 架构理解（按 part 划分）

| Part | 架构详解 | 组件清单 |
|------|---------|---------|
| **server**（Go 后端） | [architecture-server.md](architecture-server.md) | — |
| **web-default**（React 19 / Rsbuild） | [architecture-web-default.md](architecture-web-default.md) | [component-inventory-web-default.md](component-inventory-web-default.md) |
| **web-classic**（React 18 / Vite / Semi Design） | [architecture-web-classic.md](architecture-web-classic.md) | [component-inventory-web-classic.md](component-inventory-web-classic.md) |
| **electron**（桌面外壳） | [architecture-electron.md](architecture-electron.md) | [component-inventory-electron.md](component-inventory-electron.md) |

跨 part 集成关系详见 [integration-architecture.md](integration-architecture.md)。

---

## 3. 开发参考（按 part 划分）

| Part | 开发指南 |
|------|---------|
| server | [development-guide-server.md](development-guide-server.md) |
| web-default | [development-guide-web-default.md](development-guide-web-default.md) |
| web-classic | [development-guide-web-classic.md](development-guide-web-classic.md) |
| electron | [development-guide-electron.md](development-guide-electron.md) |

每份指南覆盖：环境准备、启动开发、项目结构、编码规范、常见任务、命令清单、调试、构建、提交前自检、关键文档交叉引用。

---

## 4. 后端契约与数据

| 文档 | 一句话描述 |
|------|----------|
| [api-contracts-server.md](api-contracts-server.md) | Go 后端 HTTP API 契约：路由分组、鉴权、请求/响应、错误码、relay/中转端点 |
| [data-models-server.md](data-models-server.md) | GORM 数据模型：表结构、索引、关系、跨 DB 兼容性注意 |
| [enterprise/usage-aggregation.md](enterprise/usage-aggregation.md) | 企业部门用量聚合、导出与定期报告实现说明 |
| [enterprise/alerting.md](enterprise/alerting.md) | 企业风险事件、告警规则、投递状态、风险概览与 scheduler 说明 |
| [openapi/](openapi/) | OpenAPI / Swagger 资源 |
| [../pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md) | 计费表达式系统（Rule 7 必读） |

---

## 5. 运维与部署

| 文档 | 一句话描述 |
|------|----------|
| [deployment-guide.md](deployment-guide.md) | Docker / 二进制 / 多节点 / 数据库迁移 / 配置注入 |
| [installation/](installation/) | 各部署形态的详细安装步骤 |
| [source-tree-analysis.md](source-tree-analysis.md) | 仓库目录树解析 + 关键路径标注 |

---

## 6. Channel / Provider 适配

| 文档 | 一句话描述 |
|------|----------|
| [channel/](channel/) | 各上游 AI provider 适配文档（OpenAI / Claude / Gemini / Azure / AWS Bedrock 等 40+） |
| [ionet-client.md](ionet-client.md) | io.net Provider 客户端实现说明 |

新增 channel 步骤参见 [development-guide-server.md §5.1](development-guide-server.md)。

---

## 7. i18n 与翻译

| 文档 | 一句话描述 |
|------|----------|
| [translation-glossary.md](translation-glossary.md) | 中英术语对照（基础） |
| [translation-glossary.fr.md](translation-glossary.fr.md) | 法语术语对照 |
| [translation-glossary.ru.md](translation-glossary.ru.md) | 俄语术语对照 |

后端 i18n（`nicksnyder/go-i18n/v2`，en / zh）与前端 i18n（`i18next`，zh / en / fr / ru / ja / vi）的位置见 [../CLAUDE.md](../CLAUDE.md) 中 i18n 章节。

---

## 8. 跨 part 集成

| 文档 | 一句话描述 |
|------|----------|
| [integration-architecture.md](integration-architecture.md) | server ↔ web-default / server ↔ web-classic / electron ↔ server 的契约、go:embed、Theme Option 切换、IPv4 端口契约 |
| [project-parts.json](project-parts.json) | 机器可读的 4-part 清单（依赖关系 / 入口 / 构建命令 / 集成点） |

---

## 9. 协作流程

| 文档 | 一句话描述 |
|------|----------|
| [contribution-guide.md](contribution-guide.md) | PR 全流程：分支 / 模板 / 校验 / 评审 / 测试 / 文档同步 |
| [../CLAUDE.md](../CLAUDE.md) | Rule 5 受保护标识 `new-api` / `QuantumNous` 不可改 |
| [../.github/pull_request_template.md](../.github/pull_request_template.md) | PR 模板 |

---

## 10. 机器可读资源

| 文件 | 用途 |
|------|------|
| [project-parts.json](project-parts.json) | 工具链 / Agent 读取仓库结构 |
| [project-scan-report.json](project-scan-report.json) | BMAD 文档项目工作流的扫描报告 |
| [openapi/](openapi/) | API 自动化客户端生成 |

---

## 11. 快速命令速查

| 操作 | 命令 |
|------|------|
| 起后端 | `go run main.go` |
| 起 default 前端 | `cd web/default && bun run dev` |
| 起 classic 前端 | `cd web/classic && bun run dev` |
| 起 Electron | `cd electron && npm run dev` |
| Go 编译 | `go build ./...` |
| Go 测试 | `go test ./...` |
| 前端构建（default） | `cd web/default && bun run build` |
| 前端构建（classic） | `cd web/classic && bun run build` |
| 类型检查（default） | `cd web/default && bun run type-check` |
| i18n 同步（default） | `cd web/default && bun run i18n:sync` |
| Electron 打包 | `cd electron && npm run build:win` / `build:mac` / `build:linux` |
| Docker 起依赖 | `docker compose -f docker-compose.dev.yml up -d` |

---

## 12. 文档维护说明

- 文档生成时间：**2026-05-26**
- 当代码变更涉及以下场景，必须同步更新对应文档（详见 [contribution-guide.md §6](contribution-guide.md)）：
  - 新增 / 修改 / 删除 HTTP 端点 → [api-contracts-server.md](api-contracts-server.md)
  - 新增 / 修改数据表 → [data-models-server.md](data-models-server.md)
  - 新增 / 修改环境变量 → [deployment-guide.md](deployment-guide.md)
  - 新增上游 provider → [architecture-server.md](architecture-server.md) + [project-parts.json](project-parts.json) + [channel/](channel/)
  - 新增前端页面 → 对应 [component-inventory-*.md](.)
  - 修改架构 → 对应 [architecture-*.md](.) + [integration-architecture.md](integration-architecture.md)

---

## 13. 项目主页

- 仓库：[QuantumNous/new-api](https://github.com/QuantumNous/new-api)
- Issues：[QuantumNous/new-api/issues](https://github.com/QuantumNous/new-api/issues)
- Discussions：[QuantumNous/new-api/discussions](https://github.com/QuantumNous/new-api/discussions)
