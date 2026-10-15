# 开发指南 — Electron 桌面外壳

> 项目部分：`electron`
> 路径：[electron/](../electron/)
> 文档生成时间：2026-05-26

---

## 1. 环境准备

| 工具 | 最低版本 |
|------|---------|
| Node.js | 20+ |
| npm | 10+（electron-builder 偏好 npm） |
| Go | 1.25.1+（构建 Go 二进制时需要） |
| Bun | 任意稳定版（前端构建） |

平台特殊依赖：
- **Windows**：无（NSIS 由 electron-builder 自带）
- **macOS**：可选 Apple Developer ID 用于签名（默认未签名 `identity: null`）
- **Linux**：`fpm`、`rpm-build`（如需 deb / rpm）

---

## 2. 启动开发

### 2.1 推荐工作流（手动起后端 + 前端）

Electron 开发模式会跳过 spawn Go 二进制（`NODE_ENV=development`），假设开发者已手动启动后端与前端。

```bash
# 终端 1：起 Go 后端（端口 3000）
go run main.go

# 终端 2：起前端 dev server（端口 5173）
cd web/default
bun install
bun run dev

# 终端 3：起 Electron
cd electron
npm install
npm run dev
```

Electron 主窗口会 loadURL `http://127.0.0.1:5173`，前端 dev server 代理 API 到 3000。

### 2.2 完整生产模拟

要测试 spawn 二进制 + IPv4 IPC + 健康检查的完整流程：

```bash
# 1. 构建前端
cd web/default && bun install && bun run build && cd ../..
cd web/classic && bun install && bun run build && cd ../..

# 2. 构建 Go 二进制到仓库根
go build -o new-api    # macOS/Linux
go build -o new-api.exe   # Windows

# 3. Electron 直接运行（不打包）
cd electron
npm install
NODE_ENV=production npm start
```

此时 Electron 会 spawn `../new-api(.exe)`、监听端口 3000、loadURL 同源。

---

## 3. 项目结构

```
electron/
├─ main.js                    主进程：窗口 / 托盘 / 后端进程托管 / 错误诊断
├─ preload.js                 contextBridge 暴露 window.electron
├─ create-tray-icon.js        离线工具：用 canvas 生成 22×22 mac 模板托盘图标
├─ package.json               electron-builder + 平台 extraResources 配置
├─ icon.png                   应用图标
├─ tray-icon-windows.png      Windows / Linux 托盘（彩色）
├─ tray-iconTemplate.png      macOS 托盘（模板图，自适应）
├─ tray-iconTemplate@2x.png   macOS Retina 托盘
└─ entitlements.mac.plist     macOS 沙盒权限（仅打包时使用）
```

详见 [architecture-electron.md](architecture-electron.md) 与 [component-inventory-electron.md](component-inventory-electron.md)。

---

## 4. 编码规范

### 4.1 主进程安全实践（必须保留）

```js
new BrowserWindow({
  webPreferences: {
    preload: path.join(__dirname, 'preload.js'),
    nodeIntegration: false,        // 禁止渲染端使用 require
    contextIsolation: true,        // 隔离主进程 / 渲染端 context
  }
})
```

不要为方便而开启 `nodeIntegration: true` —— 这会让任何 XSS 直接获得 Node.js 能力。

### 4.2 IPC

当前架构无双向 IPC，所有桌面态信息单向通过 `process.env.ELECTRON_DATA_DIR` → preload → `window.electron` 注入。

如需新增双向通信：
1. 主进程：`ipcMain.handle('foo', async (event, args) => { ... })`
2. preload：`contextBridge.exposeInMainWorld('electron', { foo: (args) => ipcRenderer.invoke('foo', args) })`
3. 渲染端：`await window.electron.foo(args)`

**IPC 通道是攻击面**，每个新增 channel 都要严格校验输入。

### 4.3 子进程管理

- spawn Go 二进制时必须传完整 env：`{ ...process.env, PORT, SQLITE_PATH }`
- stderr 必须捕获并限长（`serverErrorLogs` 保留最近 100 条）
- 退出处理必须分支：code===0 / code!==0 / code===null（信号）
- 优雅退出：`SIGTERM` → 5s 超时 → `SIGKILL`

### 4.4 平台分支

```js
if (process.platform === 'darwin') {
  // macOS 专属：dock 显隐、模板图标
} else if (process.platform === 'win32') {
  // Windows 专属：彩色托盘、可执行后缀 .exe
} else {
  // Linux
}
```

不要假设平台行为相同（如托盘图标格式、路径分隔符）。

### 4.5 错误诊断

新增错误模式时在 `analyzeError(logs)` 中追加正则：

```js
if (logsText.match(/your-pattern/i)) {
  return { type: 'YOUR_TYPE', message: '友好提示', suggestion: '修复建议' }
}
```

保持 fallback 分支以兜底未识别错误。

---

## 5. 常见任务

### 5.1 修改窗口尺寸 / 行为

编辑 `createWindow()` 中 `BrowserWindow` 的构造参数。注意保留 `webPreferences` 安全选项。

### 5.2 新增托盘菜单项

```js
const contextMenu = Menu.buildFromTemplate([
  { label: 'Show New API', click: () => mainWindow.show() },
  { label: 'New Action', click: () => doSomething() },   // 新增
  { type: 'separator' },
  { label: 'Quit', click: () => { app.isQuitting = true; app.quit() } }
])
```

### 5.3 调整 spawn 后端行为

`startServer()` 函数中：

- 修改环境变量：调整 `env: { ... PORT, SQLITE_PATH }`
- 修改重试策略：`checkServerAvailability(port)` 中的 30 次 / 1s 间隔
- 添加 stderr 模式匹配：`analyzeError` 函数

### 5.4 修改 electron-builder 配置

在 `electron/package.json` 的 `build` 字段中：

- `appId` / `productName`：**Rule 5 受保护，不要改**
- `extraResources`：调整打包时复制的文件
- `mac` / `win` / `linux`：平台专属选项
- `nsis` / `dmg` / `appImage`：安装格式选项

### 5.5 生成新的托盘图标

```bash
cd electron
npm install canvas   # create-tray-icon.js 依赖
node create-tray-icon.js
# 输出 tray-iconTemplate.png + @2x.png
```

只在首次或图标设计变更时运行，正式打包不调用此脚本。

---

## 6. 命令清单

| 命令 | 用途 |
|------|------|
| `npm install` | 安装依赖 |
| `npm start` / `npm run dev` | 启动 Electron（开发模式） |
| `npm run build:win` | 打包 Windows（NSIS + portable） |
| `npm run build:mac` | 打包 macOS（dmg + zip） |
| `npm run build:linux` | 打包 Linux（AppImage + deb） |

> 具体脚本以 `electron/package.json` 中实际定义为准。

---

## 7. 调试

### 7.1 主进程

主进程用 Node.js 调试器：

```bash
electron --inspect=5858 main.js
```

VS Code 中 `Attach to Process` 调试。

### 7.2 渲染进程

主窗口默认未开 DevTools。可临时在 `createWindow()` 中加：

```js
mainWindow.webContents.openDevTools()
```

或运行时按 `F12` / `Ctrl+Shift+I`（如已注册快捷键）。

### 7.3 后端 stderr

实时查看 spawn 子进程的 stderr：

```js
// main.js 中临时调整
serverProcess.stderr.on('data', chunk => {
  process.stdout.write(chunk)   // 转发到 Electron 主进程 stdout
})
```

### 7.4 崩溃日志

正式打包后崩溃日志位置：
- Windows：`%APPDATA%/new-api-electron/logs/`
- macOS：`~/Library/Logs/new-api-electron/`
- Linux：`~/.config/new-api-electron/logs/`

日志文件名：`new-api-crash-<ISO>.log`

---

## 8. 打包发布

### 8.1 标准流程

```bash
# 1. 前端构建
cd web/default && bun install && bun run build && cd ../..
cd web/classic && bun install && bun run build && cd ../..

# 2. Go 二进制（输出到仓库根）
go build -o new-api.exe   # 或 new-api（mac/linux）

# 3. Electron 打包
cd electron
npm install
npm run build:win   # / build:mac / build:linux
```

### 8.2 CI 工作流

`.github/workflows/electron-build.yml` 当前矩阵仅 Windows。macOS / Linux 安装包需本地构建或扩展矩阵（详见 [deployment-guide.md §8](deployment-guide.md)）。

### 8.3 macOS 签名

默认未签名（`identity: null` / `gatekeeperAssess: false` / `hardenedRuntime: false`），用户首次运行需手动允许。

商用分发需配置：
1. Apple Developer 账户（$99/yr）
2. 在 `electron/package.json` `build.mac` 中设置 `identity: 'Developer ID Application: ...'`
3. 启用 `hardenedRuntime: true`、`entitlements: 'entitlements.mac.plist'`
4. 公证（notarization）：`afterSign` hook 调用 `electron-notarize`

---

## 9. 已知约束

详见 [architecture-electron.md §8](architecture-electron.md)：
1. PORT 3000 隐式契约（与 Go 后端默认端口绑定）
2. 单向 IPC（无主进程主动推送）
3. macOS 默认未签名
4. Windows-only CI 矩阵
5. `create-tray-icon.js` 依赖 canvas npm 包
6. 二进制路径假设（`process.resourcesPath/bin/`）

---

## 10. 提交前自检

- [ ] `npm start` 开发模式可运行
- [ ] `webPreferences.nodeIntegration` 仍为 `false`
- [ ] `webPreferences.contextIsolation` 仍为 `true`
- [ ] 新增 IPC 通道有输入校验
- [ ] 平台分支覆盖 darwin / win32 / linux 三者
- [ ] 受保护标识 `new-api` / `QuantumNous` / `New-API-App` / `com.newapi.desktop` 未被改动
- [ ] 至少在一个平台上跑过 `npm run build:xxx` 验证 extraResources 正确

---

## 11. 关键文档交叉引用

| 主题 | 文档 |
|------|------|
| 架构详解 | [architecture-electron.md](architecture-electron.md) |
| 组件清单 | [component-inventory-electron.md](component-inventory-electron.md) |
| 跨 part 集成 | [integration-architecture.md §4](integration-architecture.md) |
| 打包流程 | [deployment-guide.md §10](deployment-guide.md) |
| 项目约定 | [CLAUDE.md](../CLAUDE.md) |
| 提 PR | [contribution-guide.md](contribution-guide.md) |
