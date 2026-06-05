# 贡献指南 — new-api

> 文档生成时间：2026-05-26

欢迎为 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 贡献代码。本文档聚合提 PR 全流程要点。

---

## 1. 仓库与分支

- 仓库：https://github.com/QuantumNous/new-api
- 主分支：`main`（受保护，仅通过 PR 合入）
- 开发分支：`dev`（日常开发，PR 一般打到 `dev`）
- Tag：版本发布触发 `release.yml`；GitHub Release 发布后触发 `docker-build.yml`

请基于 `dev` 分支创建你的 feature 分支：

```bash
git fetch origin
git checkout -b feature/your-thing origin/dev
```

### 1.1 发布流水线

- `release.yml`：tag push 时构建 Linux / macOS / Windows 二进制并上传到 GitHub Release。
- `docker-build.yml`：GitHub Release `published` 事件或手动触发时构建 Docker Hub 多架构镜像，并在 manifest 推送成功后更新 Kubernetes Deployment 镜像。
- 双版本号规则：Docker 发布同时记录“我们的 Release tag”和“对应上游 new-api 官方版本号”。每次从上游官方分支合并版本代码时，同步更新根目录 `UPSTREAM_VERSION`：

```text
v1.0.0-rc.10
```

发布 workflow 会从 release tag 对应代码中的 `UPSTREAM_VERSION` 读取官方版本。生成的应用版本形如 `v2.0.0+new-api.v1.0.0-rc.10`，Docker/k8s 使用的组合镜像 tag 形如 `v2.0.0-newapi-v1.0.0-rc.10`。

Kubernetes 部署更新需要配置：

| 名称 | 类型 | 说明 |
|------|------|------|
| `KUBE_CONFIG` | Secret | kubeconfig YAML 或 base64 编码内容 |
| `KUBE_DEPLOY_STRATEGY` | Variable / Secret，可选 | `helm` 或 `kubectl`；配置 `HELM_RELEASE` / `HELM_CHART` 时默认 `helm`，否则默认 `kubectl` |
| `KUBE_NAMESPACE` | Variable / Secret | `kubectl` 策略目标命名空间；Helm 策略未配置 `HELM_NAMESPACE` 时也会复用 |
| `KUBE_IMAGE_REPOSITORY` | Variable / Secret，可选 | 镜像仓库，默认 `calciumion/new-api` |
| `KUBE_ROLLOUT_TIMEOUT` | Variable / Secret，可选 | `kubectl rollout status` 超时时间，默认 `5m` |

Helm 部署推荐配置：

| 名称 | 类型 | 说明 |
|------|------|------|
| `HELM_RELEASE` | Variable / Secret | Helm release 名称 |
| `HELM_CHART` | Variable / Secret | Chart 路径或 chart 引用 |
| `HELM_NAMESPACE` | Variable / Secret，可选 | Helm release 所在命名空间，未配置时使用 `KUBE_NAMESPACE` |
| `HELM_CREATE_NAMESPACE` | Variable / Secret，可选 | 设为 `true` 时给 `helm upgrade` 添加 `--create-namespace` |
| `HELM_IMAGE_REPOSITORY_KEYS` | Variable / Secret，可选 | 镜像仓库 values key，默认 `image.repository`；多个 key 用逗号分隔 |
| `HELM_IMAGE_TAG_KEYS` | Variable / Secret，可选 | 镜像 tag values key，默认 `image.tag`；master/slave 分开配置时可用逗号分隔 |
| `HELM_VALUES` | Secret，可选 | 追加的 values YAML 内容 |
| `HELM_EXTRA_SET` | Variable / Secret，可选 | 额外 `--set-string` 项，每行一个 `key=value` |
| `HELM_REPO_NAME` / `HELM_REPO_URL` | Variable / Secret，可选 | 需要添加 Helm repo 时配置 |
| `HELM_REPO_USERNAME` / `HELM_REPO_PASSWORD` | Secret，可选 | 私有 Helm repo 凭据 |

裸 Deployment 部署可配置：

| 名称 | 类型 | 说明 |
|------|------|------|
| `KUBE_DEPLOYMENTS` | Variable / Secret | 目标 Deployment 名称；多个用逗号分隔，例如 `new-api-master,new-api-slave` |
| `KUBE_CONTAINER` | Variable / Secret | Deployment 内需要更新的容器名 |

---

## 2. 开发流程

### 2.1 选择 part 并阅读对应开发指南

- Go 后端：[development-guide-server.md](development-guide-server.md)
- Web Default：[development-guide-web-default.md](development-guide-web-default.md)
- Web Classic：[development-guide-web-classic.md](development-guide-web-classic.md)
- Electron：[development-guide-electron.md](development-guide-electron.md)

### 2.2 必读约定

- [CLAUDE.md](../CLAUDE.md)：7 条项目级强约束规则
  - **Rule 5 受保护标识**：`new-api` / `QuantumNous` 严禁修改/删除/替换
- [project-overview.md](project-overview.md)：项目总览
- 修改计费相关：[pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)（Rule 7）

### 2.3 本地验证

提交前确保：

| 检查项 | 命令 |
|--------|------|
| Go 编译 | `go build ./...` |
| Go 测试 | `go test ./...`（受影响包） |
| Go 静态检查 | `go vet ./...` |
| 前端构建（default） | `cd web/default && bun install && bun run build` |
| 前端类型检查（default） | `cd web/default && bun run type-check` |
| 前端构建（classic） | `cd web/classic && bun install && bun run build` |
| Electron 启动 | `cd electron && npm install && npm start` |

---

## 3. PR 模板与 Anti-Slop 检查

### 3.1 PR 模板

`.github/pull_request_template.md` 强制要求填写：
- 变更摘要
- 测试方式
- 关联 issue
- Breaking changes（如有）

### 3.2 Anti-Slop 自动化检查

`.github/workflows/pr-check.yml` 使用 `peakoss/anti-slop@v0.2.1` 在 PR 打开 / 重开时自动运行。规则：

- **屏蔽 AI 生成痕迹**：PR 描述与提交信息中**不允许**出现以下内容：
  - `🤖 Generated with Claude Code`
  - 类似的"由 AI 助手生成"署名
- **强制 PR 模板**：必须填写完整
- **最低账号年龄**：30 天
- **其他启发式规则**：参见 anti-slop action 文档

> 这意味着：即使你用 Claude / Cursor / Copilot 等 AI 辅助开发，**不要**在提交信息或 PR 描述中保留 AI 署名行。

### 3.3 提交信息约定

无强制 commit convention，但建议：
- 简洁陈述变更（中英皆可，仓库内两种都常见）
- 关联 issue：`fix #123` / `close #456`
- 多行：第一行 < 72 字符，空行后展开

参考最近提交：

```
🐛 fix(system-settings): resolve save detection and number input NaN issues
🎨 fix(logs): tune usage table typography
fix: use actual user id for channel tests (#5109)
```

支持 emoji 前缀风格（gitmoji），但不强制。

---

## 4. 代码审查关注点

PR 评审会重点关注：

### 4.1 通用
- [ ] 是否破坏 [CLAUDE.md](../CLAUDE.md) 的 7 条规则
- [ ] 是否动到了受保护标识（Rule 5）
- [ ] 是否有 secret / token / credentials 误提交
- [ ] 文档是否同步更新（api-contracts / data-models / 架构图等）

### 4.2 后端（Go）
- [ ] JSON 操作走 `common.Marshal` / `Unmarshal`（Rule 1）
- [ ] DB 代码跨 SQLite/MySQL/PostgreSQL 兼容（Rule 2）
- [ ] 新 channel 的 StreamOptions 注册（Rule 4）
- [ ] DTO 可选字段使用指针 + `omitempty`（Rule 6）
- [ ] 计费改动遵循 `pkg/billingexpr/expr.md`（Rule 7）
- [ ] 测试覆盖关键分支
- [ ] 跨包导入未引入循环依赖（注意 service ↔ relay）

### 4.3 前端（Default 主题）
- [ ] TypeScript 类型完整，无 `any` 滥用
- [ ] 数据获取用 TanStack Query
- [ ] mutation 后已 invalidate
- [ ] i18n key 已 `bun run i18n:sync`
- [ ] 没有 cross-feature import

### 4.4 前端（Classic 主题）
- [ ] 路由集中在 `App.jsx`
- [ ] 守卫包装正确（`PrivateRoute` / `AdminRoute`）
- [ ] 二次验证场景使用 `secureApiCall`
- [ ] 未混入其他样式系统

### 4.5 Electron
- [ ] `nodeIntegration: false` / `contextIsolation: true` 保留
- [ ] 新增 IPC 有输入校验
- [ ] 跨平台分支完整（darwin / win32 / linux）

---

## 5. 测试要求

### 5.1 必须

- 修改触及 `pkg/billingexpr/`：必须有单元测试覆盖新表达式 case
- 修改触及 `model/`：在 SQLite 上至少跑过 `go test ./model/...`
- 新增 channel 适配器：参考 `relay/channel/api_request_test.go` 风格

### 5.2 建议

- UI 改动：截图或录屏附在 PR 描述中
- 性能敏感改动：附 benchmark / pprof 数据
- 跨 DB 改动：MySQL 或 PG 至少一个手测验证

### 5.3 现状说明

项目当前测试覆盖率不高：
- Go 后端：单测散落各包
- Web Default：少量 `*.test.tsx`，主要靠 TS 类型 + 人工
- Web Classic：无独立单测
- Electron：无单测

新增测试一律欢迎，但不会因缺测试而拒绝你的 PR（除非属于 5.1 必须场景）。

---

## 6. 文档更新

代码变更涉及以下场景必须同步更新文档：

| 变更场景 | 需要更新 |
|---------|---------|
| 新增 / 修改 / 删除 HTTP 端点 | [api-contracts-server.md](api-contracts-server.md) |
| 新增 / 修改数据表 | [data-models-server.md](data-models-server.md) |
| 新增 / 修改环境变量 | [deployment-guide.md §2](deployment-guide.md) + `.env.example` |
| 新增上游 provider | [architecture-server.md §5](architecture-server.md) + [project-parts.json](project-parts.json) |
| 新增前端页面 | 对应 [component-inventory-*.md](.) |
| 修改架构 | 对应 [architecture-*.md](.) + [integration-architecture.md](integration-architecture.md) |

---

## 7. Issue / Bug 提交

### 7.1 Bug

提供：
- 复现步骤
- 期望行为 vs 实际行为
- 环境（OS / 部署形态 / DB 类型 / new-api 版本）
- 相关日志（敏感信息脱敏）

### 7.2 Feature Request

说明：
- 用户场景与动机
- 期望接口 / 行为
- 是否愿意自己实现

---

## 8. 不会被接受的 PR 类型

- 修改受保护标识（Rule 5）
- 重大架构重构未经事前讨论
- 引入大量未使用依赖
- 提交信息或 PR 描述含 AI 署名（被 anti-slop 拦截）
- 删除已有 i18n 语言（zh/en/fr/ru/ja/vi 全保留）
- 强制要求修改默认部署方式（如 PR：从 SQLite 默认改为 MySQL 默认）

---

## 9. 联系与帮助

- Issue Tracker：https://github.com/QuantumNous/new-api/issues
- Discussions：https://github.com/QuantumNous/new-api/discussions
- 文档主索引：[index.md](index.md)

---

## 10. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 项目总览 | [project-overview.md](project-overview.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
| 各 part 开发指南 | [development-guide-server.md](development-guide-server.md) / [development-guide-web-default.md](development-guide-web-default.md) / [development-guide-web-classic.md](development-guide-web-classic.md) / [development-guide-electron.md](development-guide-electron.md) |
| 部署运维 | [deployment-guide.md](deployment-guide.md) |
| 计费表达式 | [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md) |
