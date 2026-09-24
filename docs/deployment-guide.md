# 部署与运维指南 — new-api

> 文档生成时间：2026-05-26
> 适用版本：基于 `main.go` / `go.mod`（Go 1.25.1）/ `Dockerfile` / `electron/package.json` 当前状态

---

## 1. 运行形态

| 形态 | 入口 | 说明 |
|------|------|------|
| 单二进制 | `./new-api` 或 `new-api.exe` | embed 了两个前端主题（`web/default/dist` 与 `web/classic/dist`），单文件即可启动 |
| Docker | `calciumion/new-api:latest` | 官方镜像（多架构 amd64/arm64） |
| systemd | [new-api.service](../new-api.service) 模板 | Linux 长期运行 |
| Electron 桌面 | `New-API-App` | 内部 spawn 二进制 + 加载本地 HTTP |

监听端口默认 `3000`（环境变量 `PORT` 覆盖）。

---

## 2. 环境变量（[.env.example](../.env.example)）

### 2.1 基础
| 变量 | 默认 | 说明 |
|------|------|------|
| `PORT` | 3000 | HTTP 监听端口 |
| `FRONTEND_BASE_URL` | — | 设置后非主节点会 301 跳到外部前端；主节点强制忽略 |
| `NODE_TYPE` | — | 设为 `master` 表示主节点（影响 redirect/sync 行为） |
| `SESSION_SECRET` | 随机 | 多节点部署必须设为固定字符串保证会话一致 |
| `TZ` | 系统 | 推荐设为 `Asia/Shanghai` |
| `TRUSTED_REDIRECT_DOMAINS` | — | 逗号分隔白名单，验证支付回调 URL |

### 2.2 数据库（详见 [data-models-server.md §1](data-models-server.md)）
| 变量 | 形式 | 适配驱动 |
|------|------|---------|
| `SQL_DSN` | `user:pwd@tcp(host:3306)/db?parseTime=true` | MySQL |
| `SQL_DSN` | `postgresql://user:pwd@host:5432/db` | PostgreSQL |
| `SQL_DSN` | 未设置 / `.db` | SQLite（glebarez/sqlite） |
| `SQLITE_PATH` | 文件路径 | SQLite 数据库位置 |
| `LOG_SQL_DSN` | 同 `SQL_DSN` | 独立日志库（可选） |
| `SQL_MAX_IDLE_CONNS` | 100 | 空闲连接池 |
| `SQL_MAX_OPEN_CONNS` | 1000 | 最大连接 |
| `SQL_MAX_LIFETIME` | 60 | 单位秒 |

### 2.3 缓存与同步
| 变量 | 说明 |
|------|------|
| `REDIS_CONN_STRING` | `redis://user:pwd@host:6379/0` |
| `SYNC_FREQUENCY` | DB 同步频率（秒） |
| `MEMORY_CACHE_ENABLED` | 启用内存缓存 |
| `CHANNEL_UPDATE_FREQUENCY` | 渠道刷新频率（秒） |
| `BATCH_UPDATE_ENABLED` / `BATCH_UPDATE_INTERVAL` | 批量持久化 |

### 2.4 业务/转发
| 变量 | 说明 |
|------|------|
| `RELAY_TIMEOUT` | 总请求超时秒数，0=不限 |
| `STREAMING_TIMEOUT` | SSE 无响应超时（建议生产 300） |
| `TLS_INSECURE_SKIP_VERIFY` | 跳过上游 TLS 校验（不推荐） |
| `GEMINI_VISION_MAX_IMAGE_NUM` | Gemini 多图限制 |
| `GENERATE_DEFAULT_TOKEN` | 是否给注册用户自动生成 token |
| `COHERE_SAFETY_SETTING` | Cohere 安全档位 |
| `GET_MEDIA_TOKEN` / `GET_MEDIA_TOKEN_NOT_STREAM` | 图像 token 统计开关 |
| `DIFY_DEBUG` | Dify 渠道输出工作流节点信息 |
| `ERROR_LOG_ENABLED` | 启用错误日志表 |
| `UPDATE_TASK` | 启用任务轮询 |
| `STREAMING_TIMEOUT` | 流式超时 |

### 2.5 监控/性能
| 变量 | 说明 |
|------|------|
| `ENABLE_PPROF` | 启用 pprof HTTP 接口 |
| `DEBUG` | 调试日志 |
| `PYROSCOPE_URL` / `PYROSCOPE_APP_NAME` / `PYROSCOPE_BASIC_AUTH_USER` / `PYROSCOPE_BASIC_AUTH_PASSWORD` | Pyroscope 持续 profiling |
| `PYROSCOPE_MUTEX_RATE` / `PYROSCOPE_BLOCK_RATE` | 采样率 |
| `HOSTNAME` | 节点标识 |
| `NODE_NAME` | 审计日志节点名（多实例时建议设置） |

### 2.6 第三方分析
| 变量 | 说明 |
|------|------|
| `GOOGLE_ANALYTICS_ID` | GA Measurement ID |
| `UMAMI_WEBSITE_ID` / `UMAMI_SCRIPT_URL` | Umami 集成 |

### 2.7 OAuth（部分）
| 变量 | 说明 |
|------|------|
| `LINUX_DO_TOKEN_ENDPOINT` / `LINUX_DO_USER_ENDPOINT` | LinuxDo OAuth 端点 |

订阅缓存：
- `SUBSCRIPTION_PLAN_CACHE_TTL`（默认 300s）
- `SUBSCRIPTION_PLAN_INFO_CACHE_TTL`（默认 120s）

---

## 3. 单容器快速启动

```bash
docker run -d \
  -p 3000:3000 \
  -v $PWD/data:/data \
  -e SQL_DSN= \
  -e TZ=Asia/Shanghai \
  --name new-api \
  calciumion/new-api:latest
```

不设置 `SQL_DSN` 时自动使用 SQLite，文件位于 `/data/new-api.db`。

首次访问 `http://localhost:3000` 会跳到安装向导（`/setup`）；如无 root 用户则自动创建 `root / 123456`，请立即修改。

---

## 4. docker-compose（生产）

参见 [docker-compose.yml](../docker-compose.yml)：

- `new-api`：主服务，`--log-dir /app/logs`，端口 3000
  - 默认 SQL_DSN 指向同栈 `postgres:5432`
  - `REDIS_CONN_STRING=redis://:123456@redis:6379`
  - 健康检查：`wget -q -O - http://localhost:3000/api/status | grep -o '"success":\\s*true'`，每 30s 检测
- `redis`：`redis:latest` + `--requirepass 123456`
- `postgres`：`postgres:15`，库名 `new-api`，账户 `root/123456`
- 可选 `mysql:8.2`（按注释切换 SQL_DSN）

数据卷：
- `./data` → `/data`：SQLite/上传/缓存（仅 SQLite 模式使用）
- `./logs` → `/app/logs`：应用日志
- `pg_data`：PostgreSQL 持久化
- `mysql_data`（注释中）

**部署前必改：** 三处默认密码（postgres、redis、SESSION_SECRET）。

---

## 5. docker-compose.dev（前端开发）

参见 [docker-compose.dev.yml](../docker-compose.dev.yml)：

- `new-api-dev`：用 `Dockerfile.dev` 构建（跳过前端构建，使用占位 HTML）
- 健康前置：`postgres` `healthcheck=pg_isready`
- 用法：
  ```bash
  docker compose -f docker-compose.dev.yml up -d
  cd web/default && bun install && bun run dev   # 或 web/classic
  ```
  前端 dev server 自动代理 `/api`、`/v1`、`/mj`、`/pg` 到 `localhost:3000`。
- 重建：`docker compose -f docker-compose.dev.yml up -d --build new-api`
- 重置（清数据库卷）：`docker compose -f docker-compose.dev.yml down -v`

---

## 6. systemd（裸机部署）

参见 [new-api.service](../new-api.service)：

```ini
[Unit]
Description=One API Service
After=network.target

[Service]
User=ubuntu                            # 修改为部署用户
WorkingDirectory=/path/to/new-api      # 修改为实际目录
ExecStart=/path/to/new-api/new-api --port 3000 --log-dir /path/to/new-api/logs
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启用：
```bash
sudo cp new-api.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now new-api
sudo systemctl status new-api
```

---

## 7. Makefile 任务

参见 [makefile](../makefile)：

| 任务 | 说明 |
|------|------|
| `make build-frontend` | Bun 构建 default 前端 |
| `make build-frontend-classic` | Bun 构建 classic 前端 |
| `make build-all-frontends` | 两者顺序构建 |
| `make start-backend` | `go run main.go &` 后台启动后端 |
| `make all` | 构建前端 + 启动后端 |
| `make dev-api` | `docker compose -f docker-compose.dev.yml up -d` |
| `make dev-api-rebuild` | 重建 backend 镜像 |
| `make dev-web` | 启动 default 前端 dev server |
| `make dev-web-classic` | 启动 classic 前端 dev server |
| `make dev` | 同时 `dev-api` + `dev-web` |
| `make reset-setup` | 重置安装向导（检测到 docker dev postgres 则跑 SQL 清表；否则尝试本地 SQLite） |

---

## 8. 构建流水线（[.github/workflows/](../.github/workflows/)）

| Workflow | 触发 | 用途 |
|----------|------|------|
| `docker-build.yml` | tag 推送（非 nightly）/ 手动 | 多架构 Docker 镜像（amd64 + arm64 matrix） |
| `docker-image-alpha.yml` | alpha tag | 灰度镜像 |
| `docker-image-nightly.yml` | nightly tag | 每夜镜像 |
| `release.yml` | tag 推送（非 alpha）/ 手动 | Linux/macOS/Windows 二进制发布；checkout → setup-bun → 构建 default+classic 前端 → Go 1.25.1 → 静态链接 amd64 + arm64 |
| `electron-build.yml` | release tag | Electron 安装包（当前矩阵仅 Windows） |
| `pr-check.yml` | PR 打开/重开 | `peakoss/anti-slop@v0.2.1` 防 AI Slop（屏蔽 "🤖 Generated with Claude Code"、强制 PR 模板、最低账号年龄 30 天等） |
| `sync-to-gitee.yml` | push | 镜像同步至 Gitee |

---

## 9. 构建链（[Dockerfile](../Dockerfile)）

三阶段多架构构建：

1. **`builder`**：`oven/bun:1` 镜像
   - `bun install` + `bun run build` 默认前端
   - 注入 `VITE_REACT_APP_VERSION=$(cat VERSION)`
   - 输出 `/build/dist`
2. **`builder-classic`**：同上构建 classic 前端
3. **`builder2`**：`golang:1.26.1-alpine`
   - `go mod download`
   - 拷贝 builder 两个产物到 `web/default/dist` 与 `web/classic/dist`
   - `go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api`
   - `GOEXPERIMENT=greenteagc` 开启 Go 1.25 Green Tea GC 实验
4. **运行镜像**：`debian:bookworm-slim`
   - 安装 `ca-certificates tzdata libasan8 wget`
   - 拷贝 `new-api` 与三份 license
   - `EXPOSE 3000`，`WORKDIR /data`，`ENTRYPOINT ["/new-api"]`

`Dockerfile.dev` 区别：跳过前端，写占位 `index.html` 满足 `go:embed` 约束，便于后端独立构建。

---

## 10. Electron 打包

详见 [component-inventory-electron.md §5](component-inventory-electron.md)。

简要流程：
1. CI 中先 `bun install + bun run build` 前端（写到 `web/dist`）
2. `go build` Go 二进制
3. `cd electron && bun install` 安装 electron-builder
4. `bun run build:win` / `build:mac` / `build:linux`

`extraResources` 自动从仓库根目录复制 `new-api(.exe)`、`web/dist`（仅 macOS 需要）、License 文件到 `Resources/bin/` 与 `Resources/licenses/`。

---

## 11. 监控与故障排查

### 11.1 内置健康检查
- `GET /api/status` 返回 `{"success":true, ...}` 表示后端 OK；compose 健康检查依赖此接口
- `GET /api/perf-metrics` 与 `/api/perf-metrics/summary`：模型层性能指标
- `GET /api/uptime/status`：Uptime Kuma 探活反代

### 11.2 性能控制台
RootAuth 路由：
- `GET /api/performance/stats`
- `DELETE /api/performance/disk_cache`
- `POST /api/performance/reset_stats`
- `POST /api/performance/gc`
- `GET /api/performance/logs` / `DELETE /api/performance/logs`

### 11.3 Pyroscope / pprof
设置 `ENABLE_PPROF=true` 启用 `net/http/pprof`；设置 Pyroscope 环境变量推送持续性能采样到 `PYROSCOPE_URL`。

### 11.4 常见错误（Electron `analyzeError`）
桌面端针对二进制崩溃自动诊断：端口占用、SQLite 锁、权限不足、网络不可达、配置解析失败、内存不足、文件缺失。日志保存至 `app.getPath('logs')/new-api-crash-<ISO>.log`。

---

## 12. 升级与备份

- **数据备份**：复制 `/data` 目录（含 SQLite 与上传文件） + 数据库 dump；Electron 桌面版位于 `app.getPath('userData')/data/`
- **就地升级**：替换二进制 / 重新 `docker pull` 后重启即可，GORM 自动执行 schema 迁移
- **重要：** 跨大版本升级前查阅 README 中的 Breaking Changes 与 `CLAUDE.md` Rule 2（跨库迁移约束）
- **保留版本**：`VERSION` 文件在 CI 中由 tag 注入 `common.Version` ldflag，便于审计当前部署版本

---

## 13. 安全清单

- [ ] 修改默认 `root/123456` 凭据
- [ ] 修改 `docker-compose.yml` 中 postgres/redis 默认密码
- [ ] 多节点部署设置固定 `SESSION_SECRET`
- [ ] 配置 `TRUSTED_REDIRECT_DOMAINS` 防 open redirect
- [ ] 生产关闭 `DEBUG`、`ENABLE_PPROF`、`TLS_INSECURE_SKIP_VERIFY`
- [ ] 渠道密钥读取启用 `SecureVerificationRequired`（默认已开）
- [ ] Cloudflare Turnstile 配置（如启用 `TurnstileCheck`）
- [ ] 部署反代时正确转发 `X-Forwarded-For` / `X-Real-IP`
- [ ] 定期备份数据库与 `/data`
