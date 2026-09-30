# 开发指南 — Server (Go 后端)

> 项目部分：`server`
> 文档生成时间：2026-05-26
> 适用环境：Go 1.25.1+

---

## 1. 环境准备

| 工具 | 最低版本 | 备注 |
|------|---------|------|
| Go | 1.25.1（推荐 1.26.x，CI 使用 1.26.1-alpine） | `go version` |
| Bun | 任意稳定版 | 仅在前端构建时需要（CLAUDE.md Rule 3） |
| Docker | 任意 | 本地起 PostgreSQL/Redis |
| MySQL / PostgreSQL | 5.7.8+ / 9.6+ | 可选，默认 SQLite |
| Redis | 任意 | 多节点部署必需 |

可选：
- `golangci-lint`：本地 lint
- `delve`：断点调试
- `air`：热重载（自行配置 `.air.toml`）

---

## 2. 初次启动

### 2.1 仅后端 + SQLite（最简）

```bash
git clone https://github.com/QuantumNous/new-api.git
cd new-api

# 占位前端（避免 //go:embed 找不到 dist 报错）
mkdir -p web/default/dist web/classic/dist
echo '<html></html>' > web/default/dist/index.html
echo '<html></html>' > web/classic/dist/index.html

go mod download
go run main.go
```

访问 `http://localhost:3000` → 跳安装向导 `/setup` → 自动创建 `root / 123456`。

### 2.2 完整前后端

```bash
# 终端 1：起后端依赖（Postgres + Redis）
docker compose -f docker-compose.dev.yml up -d

# 终端 2：起后端
go run main.go

# 终端 3：起前端（默认主题）
cd web/default
bun install
bun run dev   # 监听 5173，代理到 3000
```

或一行：

```bash
make dev   # = make dev-api + make dev-web
```

详见 [deployment-guide.md §5](deployment-guide.md)。

---

## 3. 项目结构速览

```
controller/   HTTP handler（按业务域）
service/      业务逻辑（计费/选址/邮件/Webhook 等）
model/        GORM 模型 + DB 抽象
relay/        转发逻辑 + 40 个 provider 适配器
router/       路由装载
middleware/   Gin 中间件
setting/      配置子包（ratio/model/system/...）
common/       共享工具（含 json.go 必用）
dto/          请求/响应结构体
constant/     枚举常量
types/        运行期类型
i18n/         go-i18n 国际化
oauth/        OAuth 抽象
pkg/          内部包（billingexpr/cachex/ionet）
```

详见 [source-tree-analysis.md §2](source-tree-analysis.md) 与 [architecture-server.md](architecture-server.md)。

---

## 4. 编码规范（强约束）

> 这些是 [CLAUDE.md](../CLAUDE.md) 中的**项目级规则**，不是建议。

### 4.1 Rule 1 — JSON 必须走 `common/json.go`

```go
// ❌ 禁止
import "encoding/json"
data, _ := json.Marshal(v)

// ✅ 正确
import "github.com/QuantumNous/new-api/common"
data, _ := common.Marshal(v)
```

可用：`common.Marshal` / `common.Unmarshal` / `common.UnmarshalJsonStr` / `common.DecodeJson` / `common.GetJsonType`。

`json.RawMessage` / `json.Number` 作为类型仍可引用，但 marshal/unmarshal 调用必须走 wrapper。

### 4.2 Rule 2 — 跨 DB 兼容

- 优先 GORM 方法（`Create` / `Find` / `Where` / `Updates`）
- 主键不要写 `AUTO_INCREMENT` / `SERIAL`，让 GORM 处理
- 引用 `group` / `key` 列时使用 `commonGroupCol` / `commonKeyCol`
- 布尔字面量用 `commonTrueVal` / `commonFalseVal`
- 分支逻辑用 `common.UsingPostgreSQL` / `UsingMySQL` / `UsingSQLite`
- 禁止：MySQL `GROUP_CONCAT`、PG `JSONB ?`、SQLite `ALTER COLUMN`

### 4.3 Rule 4 — 新增 channel 检查 StreamOptions

如果上游支持 `stream_options.include_usage` 字段，必须在 `relay/channel/openai/relay-openai.go`（或等价位置）的 `streamSupportedChannels` 列表中注册新 channel type。

### 4.4 Rule 5 — 受保护标识

`new-api` / `QuantumNous` 在所有文件中受保护，**严禁**修改/删除/替换。包括 README、license、Go module 路径、Docker image 名、HTML title、注释等。

### 4.5 Rule 6 — 可选标量用指针

转发请求 DTO 中的可选字段：

```go
// ❌ 错误（false / 0 会被 omitempty 静默丢弃）
type Request struct {
    Stream bool    `json:"stream,omitempty"`
    Temp   float64 `json:"temperature,omitempty"`
}

// ✅ 正确
type Request struct {
    Stream *bool    `json:"stream,omitempty"`
    Temp   *float64 `json:"temperature,omitempty"`
}
```

### 4.6 Rule 7 — 计费表达式必读 expr.md

修改 `pkg/billingexpr/` 或 `service/billing*.go` 前，必须读 [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md)，了解表达式语言、token 归一化（`p`/`c` 自动剔除）、配额换算、表达式版本化等规则。

---

## 5. 常见任务步骤

### 5.1 新增一个上游 AI Provider

1. 在 `constant/channel.go` 添加枚举 `ChannelTypeXxx = NN`
2. 在 `relay/channel/xxx/` 创建目录，至少包含：
   - `adaptor.go`（实现 `relay.Adaptor` 接口）
   - `request.go` / `response.go`（DTO 定义，遵守 Rule 6 指针规则）
   - `models.go`（静态模型列表）
3. 在 `relay/channel/get_adaptor.go`（或类似工厂文件）注册
4. 在 `relay/channel/api_request.go` 中检查是否需要特化 HTTP 调用骨架
5. 检查 Rule 4：是否支持 `StreamOptions`
6. 在 `setting/model_setting/` 注册默认参数
7. 前端 `web/{theme}/src/...` 同步模型列表展示
8. 写测试：`relay/channel/api_request_test.go` 风格

参考：`relay/channel/openai/`、`relay/channel/claude/`、`relay/channel/gemini/`。

### 5.2 新增一个 HTTP 端点

1. 在 `controller/` 对应业务域文件加 handler 函数
2. 在 `dto/` 加请求/响应结构体（遵守 Rule 6）
3. 在 `router/api-router.go`（或合适的子路由文件）注册路由 + 中间件
4. 写测试：与 controller 文件同目录，`*_test.go`
5. 更新 [api-contracts-server.md](api-contracts-server.md)

### 5.3 新增一个数据表

1. 在 `model/` 创建文件，定义 struct + GORM tags
2. 在 `model/main.go` 的 `InitDB` / `migrateDB` 中注册 `db.AutoMigrate(&YourModel{})`
3. 跨 DB 兼容性自检（Rule 2）
4. SQLite 不支持 `ALTER COLUMN` — 用 `ADD COLUMN` 兜底
5. 测试：`model/your_model_test.go`
6. 更新 [data-models-server.md](data-models-server.md)

### 5.4 新增一个配置项

1. 选择合适的 `setting/` 子包（如 `system_setting/`）
2. 加字段到 struct + 默认值到 `Init` 函数
3. 暴露 `GetXxx()` / `SetXxx()` 函数
4. 在 controller `option.go` 中加字段名到允许列表（如有白名单）
5. 前端控制台加表单字段（两个主题各一处）
6. 持久化：写入 `options` 表，由 `model.SyncOptions(SyncFrequency)` 周期同步

---

## 6. 测试

### 6.1 现状

- 单元测试散落在各包：`controller/*_test.go`、`service/*_test.go`、`model/*_test.go`、`common/json_test.go`、`pkg/billingexpr/*_test.go`、`relay/channel/api_request_test.go` 等
- 无独立 e2e 框架，依赖 `payment_webhook_availability_test.go` 风格的回归测试

### 6.2 跑测试

```bash
go test ./...                            # 全量
go test ./controller/...                 # 单包
go test -run TestXxx ./service/          # 单测试
go test -race ./...                      # 含竞态检测
go test -cover ./service/                # 覆盖率
```

### 6.3 跨 DB 测试

测试默认用 SQLite。如需验证 MySQL/PostgreSQL，临时设置：

```bash
SQL_DSN='postgresql://...' go test ./model/...
SQL_DSN='user:pwd@tcp(localhost:3306)/test_db?parseTime=true' go test ./model/...
```

---

## 7. 调试

### 7.1 启用 pprof

```bash
ENABLE_PPROF=true go run main.go
# 访问 http://localhost:8005/debug/pprof/
```

### 7.2 详细日志

```bash
DEBUG=true go run main.go
```

### 7.3 Pyroscope（持续 profiling）

```bash
PYROSCOPE_URL=http://localhost:4040 \
PYROSCOPE_APP_NAME=new-api-dev \
go run main.go
```

### 7.4 流式请求调试

```bash
DIFY_DEBUG=true   # Dify 渠道输出工作流节点信息
```

---

## 8. 构建

### 8.1 本地构建

```bash
# 前端先构建（必须，因为 //go:embed）
cd web/default && bun install && bun run build && cd ../..
cd web/classic && bun install && bun run build && cd ../..

# Go 二进制
go build -o new-api -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'"
```

### 8.2 多平台

```bash
GOOS=linux GOARCH=amd64 go build -o new-api-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o new-api-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o new-api.exe
```

### 8.3 启用 Go 1.25 实验性 GC

```bash
GOEXPERIMENT=greenteagc go build ...   # 与 Dockerfile 一致
```

---

## 9. 性能优化提示

- 所有热路径必须避免 panic（影响 recover 性能）
- 大量 JSON 处理时考虑流式（`common.DecodeJson`），避免一次性 `Marshal/Unmarshal`
- DB 访问优先批量（`Where(...).Find(&items)` 一次拉），避免 N+1
- 缓存优先 `pkg/cachex.HybridCache`（Redis + samber/hot 双层）
- HTTP 调用必须用 `service/http_client.go` 共享池，禁止裸 `http.Get`
- 后台任务用 `common/gopool.go`，避免无限 goroutine

---

## 10. 提交前自检

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过（受影响模块）
- [ ] `go vet ./...` 无新警告
- [ ] JSON 操作走 `common.Marshal` / `Unmarshal`
- [ ] DB 代码跨 SQLite/MySQL/PostgreSQL 验证
- [ ] 受保护标识 `new-api` / `QuantumNous` 未被改动
- [ ] DTO 可选字段使用指针 + `omitempty`
- [ ] 计费改动已对照 `pkg/billingexpr/expr.md`
- [ ] 关联文档（api-contracts / data-models / architecture）已更新

---

## 11. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 架构详解 | [architecture-server.md](architecture-server.md) |
| API 契约 | [api-contracts-server.md](api-contracts-server.md) |
| 数据模型 | [data-models-server.md](data-models-server.md) |
| 部署运维 | [deployment-guide.md](deployment-guide.md) |
| 计费表达式 | [pkg/billingexpr/expr.md](../pkg/billingexpr/expr.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
| 提 PR | [contribution-guide.md](contribution-guide.md) |
