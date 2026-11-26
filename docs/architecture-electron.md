# 架构文档 — Electron (桌面外壳)

> 项目部分：`electron`
> 根路径：[electron/](../electron/)
> 入口：[electron/main.js](../electron/main.js)、[electron/preload.js](../electron/preload.js)
> 文档生成时间：2026-05-26

本文档聚焦 Electron 桌面外壳的内部架构。组件清单见 [component-inventory-electron.md](component-inventory-electron.md)；与后端/前端的集成关系见 [integration-architecture.md §4](integration-architecture.md)。

---

## 1. 架构定位

Electron 是一个 **"瘦壳"**：

```
┌──────────────────────────────────────────────────┐
│              Electron Main Process               │
│                  (electron/main.js)              │
│                                                  │
│  ┌─────────────┐   ┌─────────────────────────┐  │
│  │   Tray      │   │   BrowserWindow         │  │
│  │  + Menu     │   │  loadURL('http://       │  │
│  └─────────────┘   │   127.0.0.1:3000')      │  │
│                    └─────────────────────────┘  │
│                                                  │
│  ┌──────────────────────────────────────────┐   │
│  │     Spawned Go Binary (子进程)            │   │
│  │     bin/new-api(.exe)                    │   │
│  │     env: PORT=3000, SQLITE_PATH=...      │   │
│  └──────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
                       │
                       │ contextBridge
                       ▼
┌──────────────────────────────────────────────────┐
│       Renderer Process (React 前端 SPA)          │
│        加载自 http://127.0.0.1:3000              │
│        window.electron.{isElectron, dataDir, …} │
└──────────────────────────────────────────────────┘
```

设计意图：
- **复用现有前后端**：Electron 不重写任何 UI，只是把 Web SPA 套上窗口
- **后端进程托管**：Electron 负责拉起 Go 二进制，崩溃监控、优雅退出
- **桌面化体验**：托盘常驻、单实例、数据目录隔离、原生错误诊断
- **零代码侵入**：Web 端通过 `window.electron?.isElectron` 探测，按需启用桌面专属 UI

---

## 2. 技术栈

| 层 | 技术 |
|----|-----|
| 框架 | Electron 39.8.5（Chromium + Node.js） |
| 打包 | electron-builder ^26.7.0 |
| 平台目标 | macOS（dmg/zip）、Windows（nsis/portable）、Linux（AppImage/deb） |
| 包管理 | npm |
| 渲染层 | 直接加载 `http://127.0.0.1:3000`（不引入额外框架） |
| 子进程 | Node.js `child_process.spawn`（管理 Go 二进制） |
| 平台抽象 | Electron 内置 `app.getPath()` + 平台 if 分支 |

`electron/package.json` 关键元数据：
- `name`：`new-api-electron`
- `productName`：`New-API-App`
- `appId`：`com.newapi.desktop`
- `author`：`QuantumNous`
- `repository`：`https://github.com/QuantumNous/new-api`

---

## 3. 主进程生命周期（[main.js](../electron/main.js)）

```
app.whenReady
   ├─► startServer()           ── 拉起 Go 二进制 + 等待端口可用
   ├─► createTray()            ── 创建托盘 + 菜单
   ├─► createWindow()          ── 创建主窗口 + loadURL
   └─► registerAppEvents()     ── activate / before-quit / second-instance
```

### 3.1 模式判断

| 模式 | 判断 | 行为 |
|------|------|------|
| 开发 | `NODE_ENV=development` | 跳过 spawn；假设开发者已手动 `go run main.go` (3000) + `bun dev` (5173) |
| 生产 | 默认 | spawn `process.resourcesPath/bin/new-api(.exe)`，等待 3000 端口可用 |

### 3.2 端口契约

- `PORT = 3000`：Go 后端，与 [main.go](../main.go) 默认值绑定
- `DEV_FRONTEND_PORT = 5173`：前端 Vite/Rsbuild dev server
- 主窗口 loadURL：开发=`http://127.0.0.1:5173`，生产=`http://127.0.0.1:3000`
- **显式 IPv4**：避免 macOS / Windows 上 `localhost` 解析到 IPv6 (::1) 时 Go 服务不监听导致连接失败

### 3.3 健康检查 `checkServerAvailability(port)`

| 参数 | 值 |
|------|----|
| 重试次数 | 30 |
| 间隔 | 1 秒 |
| 单次超时 | 10 秒 |
| 成功条件 | 任意 HTTP 响应（不校验 status code） |

> 若 30 次重试后仍失败，弹出 dialog 提示用户检查日志并退出。

### 3.4 数据目录隔离

- 自动创建：`app.getPath('userData')/data/`
- 通过环境变量传给 Go：`SQLITE_PATH=<userData>/data/new-api.db`
- 通过环境变量传给 preload：`ELECTRON_DATA_DIR=<userData>/data`
- 前端通过 `window.electron.dataDir` 读取，展示"打开数据目录"按钮

### 3.5 子进程管理

```js
serverProcess = spawn(binaryPath, [], {
  cwd: process.resourcesPath,
  env: { ...process.env, PORT: '3000', SQLITE_PATH: '...' }
})

serverProcess.stdout.on('data', ...)
serverProcess.stderr.on('data', chunk => {
  serverErrorLogs.push(chunk.toString())
  if (serverErrorLogs.length > 100) serverErrorLogs.shift()
})

serverProcess.on('exit', (code, signal) => {
  if (code !== 0 && code !== null) {
    const diagnosis = analyzeError(serverErrorLogs)
    dialog.showMessageBox({...})  // 退出 / 查看完整日志
  }
})
```

### 3.6 错误诊断（`analyzeError`）

对 stderr 模式匹配，给用户友好提示：

| 错误模式 | 诊断 |
|---------|------|
| `address already in use` | 端口 3000 被占用 |
| `database is locked` | SQLite 数据库被其他进程锁定 |
| `permission denied` | 文件权限不足 |
| `connection refused` / `no route to host` | 网络不可达 |
| `unmarshal` / `invalid config` | 配置文件解析失败 |
| `out of memory` / `cannot allocate` | 内存不足 |
| `no such file` | 文件缺失 |

完整日志写入 `app.getPath('logs')/new-api-crash-<ISO>.log`，通过 `shell.openPath()` 打开。

### 3.7 优雅退出

```
用户点托盘 Quit
   └─► app.isQuitting = true
       └─► before-quit 事件
           └─► serverProcess.kill('SIGTERM')
               └─► 5 秒超时
                   └─► serverProcess.kill('SIGKILL')
                       └─► app.exit()
```

> 退出 code 为 `null`（被信号杀死）时不弹错误对话框，视为正常关闭。

---

## 4. 窗口与托盘

### 4.1 主窗口 `createWindow()`

| 属性 | 值 |
|------|----|
| 大小 | 1080 × 720 |
| `webPreferences.preload` | `preload.js` |
| `webPreferences.nodeIntegration` | `false`（安全实践） |
| `webPreferences.contextIsolation` | `true`（安全实践） |
| `close` 事件 | 默认隐藏到托盘，仅 `app.isQuitting=true` 时真正关闭 |

> macOS 上隐藏时同步 `app.dock.hide()`，显示时 `app.dock.show()`。

### 4.2 托盘 `createTray()`

| 平台 | 图标 |
|------|------|
| macOS | `tray-iconTemplate.png` + `@2x.png`（模板图，菜单栏自适应深浅色） |
| Windows / Linux | `tray-icon-windows.png`（彩色） |

托盘菜单：
- **Show New API**：显示/隐藏窗口
- **Quit**：触发 `app.isQuitting=true` + `app.quit()`

托盘单击：切换窗口显隐（与 macOS dock 同步）。

---

## 5. 进程间通信（IPC）

**当前未使用双向 IPC**。所有桌面态信息单向通过 `process.env.ELECTRON_DATA_DIR` 注入到 preload，再由 [preload.js](../electron/preload.js) 通过 `contextBridge.exposeInMainWorld('electron', ...)` 暴露给渲染端。

### preload.js 暴露 API

```js
contextBridge.exposeInMainWorld('electron', {
  isElectron: true,
  version: process.versions.electron,
  platform: process.platform,            // darwin/win32/linux
  versions: process.versions,            // {electron, chrome, node, v8, ...}
  dataDir: process.env.ELECTRON_DATA_DIR // SQLite 数据目录
})
```

前端使用模式：

```js
if (window.electron?.isElectron) {
  console.log('Running in Electron', window.electron.platform)
  console.log('Data dir:', window.electron.dataDir)
}
```

> 如未来需要主进程主动通知渲染端（例如更新提醒），需要引入 `ipcMain` / `ipcRenderer` 双向通道。

---

## 6. electron-builder 配置

### 6.1 平台目标

| 平台 | 格式 | 备注 |
|------|------|------|
| macOS | `dmg` + `zip` | `category=public.app-category.developer-tools`，无签名（identity:null, gatekeeperAssess:false, hardenedRuntime:false） |
| Windows | `nsis`（非一键，可改路径）+ `portable` | NSIS 自定义安装界面 |
| Linux | `AppImage` + `deb` | `category=Development` |

### 6.2 extraResources（关键）

```jsonc
"extraResources": [
  // 把 Go 二进制打入安装包
  { "from": "../new-api{.exe}", "to": "bin/new-api{.exe}" },
  // macOS 额外打入前端（其他平台依赖 Go 二进制 embed）
  { "from": "../web/dist", "to": "web/dist" },
  // License
  { "from": "../LICENSE", "to": "licenses/LICENSE" },
  { "from": "../NOTICE", "to": "licenses/NOTICE" },
  { "from": "../THIRD-PARTY-LICENSES.md", "to": "licenses/THIRD-PARTY-LICENSES.md" }
]
```

> 运行时通过 `process.resourcesPath/bin/new-api(.exe)` 定位二进制。

### 6.3 打包流程

1. CI 中先构建前端：`cd web/default && bun install && bun run build`
2. 构建 Go 二进制：`go build -o new-api(.exe)`
3. `cd electron && npm install`
4. `npm run build:win` / `build:mac` / `build:linux`
5. 输出 `electron/dist/`

详见 [deployment-guide.md §10](deployment-guide.md)。

---

## 7. 与其他 part 的集成

| 对端 | 集成方式 |
|------|---------|
| `server` | `spawn(./bin/new-api[.exe])` + `PORT=3000` + `SQLITE_PATH=<userData>/data/new-api.db` |
| `web-default` / `web-classic` | 不直接打包前端，仅 `loadURL('http://127.0.0.1:3000')`，前端由 Go 二进制 embed 提供 |
| 开发期 | `NODE_ENV=development` 跳过 spawn，手动起 `go run main.go` + `bun dev` |

详见 [integration-architecture.md §4](integration-architecture.md)。

---

## 8. 已知约束与设计权衡

1. **PORT 3000 隐式契约**：与 Go 后端默认端口绑定。若用户修改 Go `PORT` 环境变量，Electron 探活将失败（无重定向逻辑）
2. **单向 IPC**：当前架构无法从主进程主动推消息到渲染端。未来引入更新提醒等需求时需要扩展 `ipcMain`
3. **macOS 未签名**：`identity: null`，用户首次运行需手动允许（隔离来源警告）。商用分发需自行配置 Apple Developer ID 签名
4. **退出 code 为 null 不弹窗**：被外部信号杀死时（如系统强制关机）静默关闭，避免误报
5. **Windows-only CI 矩阵**：`electron-build.yml` 当前仅构建 Windows 安装包，macOS / Linux 用户需本地构建（见 [deployment-guide.md §10](deployment-guide.md)）
6. **`create-tray-icon.js` 离线工具**：依赖 `canvas` npm 包，未安装会退回 1×1 透明 PNG。仅在首次生成图标时手动运行，正式打包不再调用
7. **二进制路径假设**：生产模式假设 `process.resourcesPath/bin/`，开发模式假设上级仓库根。修改这两个路径需同步更新 `build.extraResources`

---

## 9. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 组件清单 | [component-inventory-electron.md](component-inventory-electron.md) |
| 跨 part 集成 | [integration-architecture.md §4](integration-architecture.md) |
| 打包流程 | [deployment-guide.md §10](deployment-guide.md) |
| 源码树 | [source-tree-analysis.md §5](source-tree-analysis.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
