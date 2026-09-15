# 组件清单 — Electron 桌面外壳

> 项目部分：`electron`
> 根路径：[electron/](../electron/)
> 文档生成时间：2026-05-26
> 入口：`electron/main.js`、`electron/preload.js`

---

## 1. 技术栈摘要

- Electron 39.8.5（Chromium + Node.js）
- 构建：electron-builder ^26.7.0（NSIS / portable / dmg / zip / AppImage / deb）
- 开发热重载：cross-env（仅切换 `NODE_ENV`）
- 包管理：npm（无 lock 之外的额外锁文件）
- 不引入额外渲染层框架（直接加载内嵌前端 `web/dist` 或开发期 Vite 5173）

`electron/package.json` 关键元数据：
- `name`：`new-api-electron`
- `productName`：`New-API-App`
- `appId`：`com.newapi.desktop`
- `author`：`QuantumNous`
- `repository`：`https://github.com/QuantumNous/new-api`

## 2. 文件清单

| 文件 | 作用 |
|------|------|
| [electron/main.js](../electron/main.js) | 主进程入口：托盘、窗口、二进制 Go 服务进程托管、健康检查、错误诊断 |
| [electron/preload.js](../electron/preload.js) | 通过 `contextBridge` 暴露 `window.electron`（isElectron/version/platform/versions/dataDir） |
| [electron/create-tray-icon.js](../electron/create-tray-icon.js) | 离线小工具：使用 `canvas` 生成 22×22 mac 模板托盘图标，运行 `node create-tray-icon.js` |
| [electron/package.json](../electron/package.json) | electron-builder 配置（mac/win/linux 三平台 extraResources） |
| `electron/icon.png`、`tray-icon-windows.png`、`tray-iconTemplate.png`、`tray-iconTemplate@2x.png` | 应用与托盘图标 |

## 3. 主进程（main.js）核心流程

`app.whenReady` → `startServer()` → `createTray()` → `createWindow()`

### 3.1 模式判断
- `NODE_ENV=development`：跳过启动 Go 二进制，假设开发者已手动 `go run main.go`（端口 3000）+ `cd web && bun dev`（端口 5173），仅 `checkServerAvailability(5173)` 即可
- 生产：spawn 内置 `bin/new-api(.exe)`，传入 `PORT=3000` 与 `SQLITE_PATH=<userData>/data/new-api.db`，工作目录 `process.resourcesPath`

### 3.2 端口
- `PORT = 3000`：Go 后端端口
- `DEV_FRONTEND_PORT = 5173`：前端 Vite dev server 端口
- 主窗口加载 URL：`http://127.0.0.1:${loadPort}`（开发=5173 / 生产=3000，二者均显式使用 IPv4 避免 IPv6 解析问题）

### 3.3 健康检查 `checkServerAvailability(port)`
- 默认 30 次重试，1 秒间隔
- IPv4 直连 `127.0.0.1`
- 10 秒超时
- 任意 HTTP 响应即视为成功（不验证 status）

### 3.4 数据目录
- `app.getPath('userData')/data` 自动创建，`SQLITE_PATH` 指向 `new-api.db`
- 进程环境变量 `ELECTRON_DATA_DIR` 同步给 preload，供前端展示备份提示

### 3.5 子进程管理
- `serverProcess = spawn(binaryPath, [])`
- 捕获 stdout/stderr，stderr 写入 `serverErrorLogs`（仅保留最近 100 条）
- 异常退出（exit code ≠ 0 且 ≠ null）触发：
  1. `analyzeError(serverErrorLogs)` 模式匹配：端口占用 / 数据库锁 / 权限不足 / 网络不可达 / 配置错误 / 内存不足 / 文件缺失
  2. 显示 `dialog.showMessageBox`，提供"退出 / 查看完整日志"双选项
  3. 查看完整日志走 `saveAndOpenErrorLog()` → 写入 `app.getPath('logs')/new-api-crash-<timestamp>.log` → `shell.openPath` 打开
- 退出（`before-quit`）：`SIGTERM` → 5 秒后 `SIGKILL` 兜底

### 3.6 托盘 `createTray()`
- macOS：`tray-iconTemplate.png`（模板图，自适应深色 / 浅色菜单栏）
- Windows / Linux：`tray-icon-windows.png`（彩色）
- 上下文菜单：Show New API / Quit
- 单击托盘切换窗口显隐；macOS 上同步 `app.dock.show()/hide()`

### 3.7 窗口 `createWindow()`
- 大小 1080×720
- `webPreferences`：`preload=preload.js`、`nodeIntegration=false`、`contextIsolation=true`（安全实践）
- `close` 事件被截获，默认隐藏到托盘；只有 `app.isQuitting=true` 时才真正退出（macOS 同步 `app.dock.hide()`）

## 4. preload.js 暴露 API

通过 `contextBridge.exposeInMainWorld('electron', …)` 注入到渲染端：

| 字段 | 类型 | 说明 |
|------|------|------|
| `isElectron` | bool | 渲染端用于探测桌面环境 |
| `version` | string | Electron 版本 |
| `platform` | string | `process.platform`（darwin/win32/linux） |
| `versions` | object | `process.versions`（含 chrome/node/v8 等） |
| `dataDir` | string | 后端 SQLite 数据目录绝对路径，方便"打开数据目录"按钮 |

前端通过 `window.electron.isElectron` 即可判断当前运行在 Electron 中并启用桌面专属 UI（例如显示数据目录路径、原生备份提示）。

## 5. electron-builder 配置摘要

`package.json` 中 `build.extraResources` 在三平台分别附带：
- 上一级目录构建产物 `../new-api`（macOS/Linux）或 `../new-api.exe`（Windows） → `bin/new-api(.exe)`
- macOS 额外打入 `../web/dist` → `web/dist`（其他平台依赖二进制内部 embed）
- License / NOTICE / THIRD-PARTY-LICENSES.md / Electron Chromium License 复制到 `licenses/`

平台目标：
- macOS：dmg + zip，`category=public.app-category.developer-tools`，无签名（`identity:null, gatekeeperAssess:false, hardenedRuntime:false`）
- Windows：nsis（非一键、可改路径）+ portable
- Linux：AppImage + deb，`category=Development`

## 6. 与其他 part 的集成

- 与 `server`：通过 `spawn(./new-api[.exe])` 启动 Go HTTP 服务，端口 3000；通过 `SQLITE_PATH` 环境变量隔离数据目录到 `userData/data/`
- 与 `web-default` / `web-classic`：仅作为外壳加载 `http://127.0.0.1:3000`，二进制内置 embed 的前端（参见 [router/web-router.go](../router/web-router.go)）
- 开发期：手动 `go run main.go` + `cd web/<theme> && bun dev` + `bun run dev-app`（在 electron/ 目录）

## 7. 已知约束

- `create-tray-icon.js` 依赖 npm 包 `canvas`，未安装会回退到 1×1 透明 PNG 占位（仅用于一次性生成图标，正式打包不再调用）
- 主进程未使用 `IPC` 双向通信，所有桌面态信息单向通过 `process.env.ELECTRON_DATA_DIR` 注入 preload
- 生产模式假设 Go 二进制位于 `process.resourcesPath/bin/`，开发模式假设位于上级目录（仓库根）
- 退出代码为 `null`（被信号杀死）时不会弹错误对话框，直接关闭窗口
