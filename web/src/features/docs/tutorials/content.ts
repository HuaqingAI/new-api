/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import type { TutorialDoc, TutorialSystemId, TutorialToolId } from './types'

type TutorialKind = 'claude-code' | 'codex-desktop' | 'codex-cli' | 'openclaw'

type TutorialConfig = {
  kind: TutorialKind
  systemId: TutorialSystemId
}

const API_KEY_PLACEHOLDER = 'sk-xxx'
const BASE_URL = '{{baseUrl}}'
const BASE_URL_V1 = '{{baseUrlV1}}'
const GROUP_NAME = 'hthbuddy'
const CODEX_MODEL = 'gpt-5.6-sol'

const toolNames: Record<TutorialKind, string> = {
  'claude-code': 'Claude Code',
  'codex-desktop': 'Codex 桌面端',
  'codex-cli': 'Codex CLI',
  openclaw: 'OpenClaw',
}

function systemName(systemId: TutorialSystemId): string {
  return systemId === 'windows' ? 'Windows' : 'macOS'
}

function shellName(systemId: TutorialSystemId): string {
  return systemId === 'windows' ? 'powershell' : 'bash'
}

function baseUrlForKind(kind: TutorialKind): string {
  return kind === 'claude-code' ? BASE_URL : BASE_URL_V1
}

function modelForKind(kind: TutorialKind): string {
  return kind === 'claude-code'
    ? '在 CC Switch 中留空，使用 Claude 默认模型'
    : CODEX_MODEL
}

function intro(config: TutorialConfig): string {
  const tool = toolNames[config.kind]
  const baseUrl = baseUrlForKind(config.kind)

  return `先完成一步，再继续下一步。已经完成的安装步骤可以跳过。

## 这份教程会帮你完成什么

安装并连接 HTH API，最后发送一条消息确认可以使用。所有 API Key 示例都是占位值 <code>${API_KEY_PLACEHOLDER}</code>，实际 Key 只在你自己的软件中填写。

| 要填写的内容 | 本教程使用的值 |
| --- | --- |
| 工具 | ${tool} |
| 系统 | ${config.kind === 'openclaw' && config.systemId === 'windows' ? 'Windows（OpenClaw 在 WSL2 / Ubuntu 内运行）' : systemName(config.systemId)} |
| 令牌分组 | <code>${GROUP_NAME}</code> |
| Base URL | <code>${baseUrl}</code> |
| 模型 | ${modelForKind(config.kind)} |`
}

function apiKeySection(config: TutorialConfig): string {
  const modelNote =
    config.kind === 'claude-code'
      ? '本教程使用 Claude 默认模型。如果你要改成其他模型，请先确认当前分组支持该模型。'
      : `本教程选择的主模型是 <code>${CODEX_MODEL}</code>。不要把后台审查模型设为主会话模型。`

  return `## 2. 准备 HTH API Key

先看模型广场，再创建密钥。打开 [HTH 模型广场](${BASE_URL}/pricing)，确认分组 <code>${GROUP_NAME}</code> 当前支持的模型；后面填写 model 时，直接复制页面显示的完整模型名称，避免写错模型导致调用失败。

1. 登录 [HTH 站点](${BASE_URL})，进入令牌管理页面。
2. 新建或编辑一个令牌，分组选择 <code>${GROUP_NAME}</code>。
3. 保存并复制 API Key，稍后填入本机的软件。

${modelNote}

**完成标志**：手中有一个对应分组的 API Key。`
}

function commonTroubleshooting(extraRow?: string): string {
  return `## 没成功时，先看这里

| 你看到的现象 | 先做什么 |
| --- | --- |
| 找不到命令 | 关闭并重新打开终端，再检查版本；Windows 下确认打开的是本教程指定的终端。 |
| API Key 无效 / 401 | 检查 Key 是否复制完整、是否过期，确认已启用正确的 HTH 配置。不要把完整 Key 发到聊天里。 |
| 模型不可用 | 检查令牌分组和模型名称；以本教程的配置表和模型广场为准。 |
| 403 block | ${extraRow ?? '先确认使用官方客户端和正确分组，第三方客户端另见完整手册的外接说明。'} |
| 连接失败 / 一直等待 | 换一个可用网络，检查代理；记录完整错误文字，不要同时修改多个设置。 |`
}

function ccSwitchSection(config: TutorialConfig): string {
  const isClaude = config.kind === 'claude-code'
  const framework = isClaude ? 'Claude Code' : 'Codex'
  const modelStep = isClaude
    ? '模型留空，使用 Claude 默认模型。'
    : `模型填写 <code>${CODEX_MODEL}</code>。`
  const restartStep =
    config.kind === 'codex-desktop'
      ? '完全退出并重新打开 Codex 桌面应用。'
      : '重新打开终端。'
  const baseUrl = baseUrlForKind(config.kind)
  const platformText = config.systemId === 'windows' ? 'Windows' : 'macOS'
  const installLine =
    config.systemId === 'macos'
      ? `从 [CC Switch 官网](https://ccswitch.io/zh/download) 下载适合 macOS 的版本并安装。已有 Homebrew 时也可使用：

~~~bash
brew tap farion1231/ccswitch && brew install --cask cc-switch
~~~`
      : `从 [CC Switch 官网](https://ccswitch.io/zh/download) 下载适合 ${platformText} 的版本并安装。`

  return `## 3. 用 CC Switch 填写配置

### 使用前看一眼：先查模型，再配置

> 第一步：打开 [HTH 模型广场](${BASE_URL}/pricing)，先查看你准备使用的分组支持哪些模型。设置 model 时请复制该分组显示的完整模型名称，避免模型名称写错或分组不支持导致调用失败。

${installLine}

**CC Switch 简称 ccs；Claude Code 简称 cc，两者是不同的软件。**

1. 打开 CC Switch，点击「添加配置」。
2. 框架选择 **${framework}**。
3. 在预设供应商中选择 **HTH**；如果没有预设，就手动填写 Base URL。
4. 把第 2 步准备的 API Key 填入对应输入框。
5. ${modelStep}
6. 保存或点击「添加」，并确认已启用这个配置。
7. ${restartStep}

确认 Base URL 是 <code>${baseUrl}</code>。${isClaude ? 'Claude Code 的地址不带 <code>/v1</code>。' : 'Codex 的地址带 <code>/v1</code>。'}

**完成标志**：CC Switch 已启用 HTH 配置，工具已重新打开。`
}

function approvalSection(config: TutorialConfig): string {
  if (config.kind === 'codex-desktop') {
    return `## 4. 检查后台审批设置

如果后台审批模型不可用，可能导致任务反复失败。建议先把 Codex 的审批策略改成不主动请求审批，确认主会话能正常工作后再按需要恢复。

1. 打开 Codex 设置，在搜索框输入「审批」。
2. 点击「从不请求审批」进入配置。
3. 在「智能体默认设置」里展开「批准策略」。
4. 选择「从不请求审批」，然后新建会话。

「从不请求审批」表示受限操作会直接失败，不会自动扩大沙盒权限。如果需要保留自动审批，请在支持模型映射的接入服务中，把审批请求使用的模型映射到当前分组可用模型；只修改主会话模型不能保证后台审批也改变。

**完成标志**：已选择合适的审批方式，使用新会话继续。`
  }

  const shell = shellName(config.systemId)
  return `## 4. 检查后台审批设置

如果你遇到后台审批反复失败，先退出 Codex，然后用下面的命令启动一个不请求审批的新会话：

~~~${shell}
codex --ask-for-approval never
~~~

正常使用且没有审批问题时，可以直接进入下一步。

「从不请求审批」表示受限操作会直接失败，不会自动扩大沙盒权限。如果需要保留自动审批，请在支持模型映射的接入服务中，把审批请求使用的模型映射到当前分组可用模型；只修改主会话模型不能保证后台审批也改变。

**完成标志**：已选择合适的审批方式，使用新会话继续。`
}

function verifySection(config: TutorialConfig): string {
  if (config.kind === 'codex-desktop') {
    return `## 5. 发一条消息验证

在 Codex 中新建会话，发送：**请只回复「配置成功」**。

**完成标志**：收到正常回复，没有 API Key 无效、模型不可用或连接失败的报错。`
  }

  const command = config.kind === 'claude-code' ? 'claude' : 'codex'
  const sectionNumber = config.kind === 'claude-code' ? 4 : 5
  const shell = shellName(config.systemId)

  return `## ${sectionNumber}. 发一条消息验证

在终端执行：

~~~${shell}
${command}
~~~

如果出现工作目录信任提示，确认目录后按界面操作，再发送：**请只回复「配置成功」**。

**完成标志**：收到正常回复，没有 API Key 无效、模型不可用或连接失败的报错。`
}

function claudeInstallSection(systemId: TutorialSystemId): string {
  if (systemId === 'windows') {
    return `## 1. 安装并打开工具

右键开始按钮，打开「终端」或「Windows PowerShell」。先从 [Git for Windows](https://git-scm.com/install/windows) 安装 Git，安装后重新打开终端。

~~~powershell
git --version
~~~

执行安装命令：

~~~powershell
irm https://claude.ai/install.ps1 | iex
~~~

关闭并重新打开终端，再检查版本：

~~~powershell
claude --version
~~~

**完成标志**：版本检查能显示版本号，没有「找不到命令」的提示。`
  }

  return `## 1. 安装并打开工具

通过 Spotlight 搜索并打开「终端」。

执行安装命令：

~~~bash
curl -fsSL https://claude.ai/install.sh | bash
~~~

关闭并重新打开终端，再检查版本：

~~~bash
claude --version
~~~

**完成标志**：版本检查能显示版本号，没有「找不到命令」的提示。`
}

function claudeTutorial(systemId: TutorialSystemId): string {
  return [
    intro({ kind: 'claude-code', systemId }),
    claudeInstallSection(systemId),
    apiKeySection({ kind: 'claude-code', systemId }),
    ccSwitchSection({ kind: 'claude-code', systemId }),
    verifySection({ kind: 'claude-code', systemId }),
    commonTroubleshooting(),
    `### 以前登录过 Claude 官方账号

如果 HTH 配置没有生效，先在终端执行 <code>claude auth logout</code> 退出旧账号，再用 CC Switch 重新启用 HTH 配置。不要同时混用官方 OAuth 登录和 HTH API Key。`,
  ].join('\n\n')
}

function codexDesktopInstallSection(systemId: TutorialSystemId): string {
  return `## 1. 安装并打开工具

打开 [Codex 官方应用入口](https://developers.openai.com/codex/app/)，按页面提供的 ${systemName(systemId)} 下载和安装说明操作。已经安装的用户直接打开应用即可。

本页讲的是桌面应用；如果你使用的是终端里的 codex 命令，请回到配置手册选择 Codex CLI。

**完成标志**：可以打开 Codex 应用并进入设置。`
}

function codexCliInstallSection(systemId: TutorialSystemId): string {
  const shell = shellName(systemId)
  const gitStep =
    systemId === 'windows'
      ? `右键开始按钮，打开「终端」或「Windows PowerShell」。先从 [Git for Windows](https://git-scm.com/install/windows) 安装 Git，安装后重新打开终端。

~~~powershell
git --version
~~~

`
      : '通过 Spotlight 搜索并打开「终端」。\n\n'

  return `## 1. 安装并打开工具

${gitStep}从 [Node.js 官网](https://nodejs.org/) 安装当前 LTS 版本。重新打开终端并检查：

~~~${shell}
node -v
npm -v
~~~

然后安装 Codex CLI：

~~~${shell}
npm install -g @openai/codex
codex --version
~~~

**完成标志**：版本检查能显示版本号，没有「找不到命令」的提示。`
}

function codexTutorial(config: TutorialConfig): string {
  const install =
    config.kind === 'codex-desktop'
      ? codexDesktopInstallSection(config.systemId)
      : codexCliInstallSection(config.systemId)
  const conversationFix =
    config.kind === 'codex-desktop'
      ? `### 会话异常：先测试新会话，再创建聊天分支

先新开一个会话测试。新会话正常时，在左侧会话列表中右键原会话，选择分叉并创建聊天分支，进入新分支后重试。如果新会话也报错，再继续核对配置。`
      : ''

  return [
    intro(config),
    install,
    apiKeySection(config),
    ccSwitchSection(config),
    approvalSection(config),
    verifySection(config),
    commonTroubleshooting(),
    conversationFix,
  ]
    .filter(Boolean)
    .join('\n\n')
}

function openClawInstallSection(systemId: TutorialSystemId): string {
  const shell = shellName(systemId)
  const prefix =
    systemId === 'windows'
      ? `OpenClaw 在本教程中使用 WSL2 / Ubuntu。如果还没有 WSL，以管理员身份打开 PowerShell，执行下面的命令，按提示完成安装或重启：

~~~powershell
wsl --install
~~~

之后从开始菜单打开 Ubuntu，完成首次用户名和密码设置。后续 OpenClaw 命令都在 Ubuntu 终端执行。`
      : '通过 Spotlight 搜索并打开「终端」。'

  return `## 1. 安装并打开工具

${prefix}

从 [Node.js 官网](https://nodejs.org/) 安装当前 LTS 版本。${systemId === 'windows' ? '请选择 Linux / Ubuntu 安装方式，在 WSL 内安装。' : ''}重新打开终端并检查：

~~~bash
node -v
npm -v
~~~

然后安装 OpenClaw：

~~~${shell}
npm install -g openclaw@latest
openclaw --version
~~~

**完成标志**：版本检查能显示版本号，没有「找不到命令」的提示。`
}

function openClawInitSection(systemId: TutorialSystemId): string {
  const terminal = systemId === 'windows' ? 'Ubuntu 终端' : '终端'

  return `## 3. 初始化 OpenClaw

在${terminal}中执行：

~~~bash
openclaw onboard --install-daemon
~~~

1. 选择 QuickStart，按引导完成本机网关初始化。
2. 供应商尚未配置时先跳过，下一步接入 HTH。
3. 先完成本地网关配置，再根据需要添加飞书或其他通知渠道。
4. 如果引导展示了本地控制台地址，使用它打开页面；默认端口是 18789。

**完成标志**：初始化完成，本机已生成 OpenClaw 配置文件。`
}

function openClawConfigSection(): string {
  return `## 4. 填入 HTH 连接配置

打开 <code>~/.openclaw/openclaw.json</code>。先备份文件，再把下面的 <code>agents.defaults.model</code> 和 <code>models.providers</code> 字段合并到现有 JSON 中。保留已有 gateway、channels 和认证配置。

~~~json
{
  "agents": {
    "defaults": {
      "model": {
        "primary": "hth-codex/${CODEX_MODEL}"
      }
    }
  },
  "models": {
    "mode": "merge",
    "providers": {
      "hth-codex": {
        "api": "openai-responses",
        "apiKey": "${API_KEY_PLACEHOLDER}",
        "baseUrl": "${BASE_URL_V1}",
        "headers": {
          "Authorization": "Bearer ${API_KEY_PLACEHOLDER}",
          "User-Agent": "codex_cli_rs/0.77.0 (Windows 10.0.26100; x86_64) WindowsTerminal"
        },
        "models": [
          {
            "id": "${CODEX_MODEL}",
            "name": "${CODEX_MODEL}"
          }
        ]
      }
    }
  }
}
~~~

把两处 <code>${API_KEY_PLACEHOLDER}</code> 都替换成自己的同一个 HTH API Key。

保存后执行：

~~~bash
openclaw gateway restart
~~~

**完成标志**：网关能够重新启动，模型指向 HTH provider。`
}

function openClawVerifySection(): string {
  return `## 5. 发一条消息验证

执行下面的命令打开控制台，再在聊天页面发送：**请只回复「配置成功」**。

~~~bash
openclaw dashboard
~~~

**完成标志**：收到正常回复，没有 API Key 无效、模型不可用或连接失败的报错。`
}

function openClawTutorial(systemId: TutorialSystemId): string {
  return [
    intro({ kind: 'openclaw', systemId }),
    openClawInstallSection(systemId),
    apiKeySection({ kind: 'openclaw', systemId }),
    openClawInitSection(systemId),
    openClawConfigSection(),
    openClawVerifySection(),
    commonTroubleshooting('确认保留了本教程的 User-Agent 请求头。'),
  ].join('\n\n')
}

export function createTutorialDoc(config: TutorialConfig): TutorialDoc {
  const toolId = config.kind as TutorialToolId
  const title = `${toolNames[config.kind]} · ${systemName(config.systemId)} 配置教程`
  let markdown: string

  if (config.kind === 'claude-code') {
    markdown = claudeTutorial(config.systemId)
  } else if (config.kind === 'openclaw') {
    markdown = openClawTutorial(config.systemId)
  } else {
    markdown = codexTutorial(config)
  }

  return {
    markdown,
    systemId: config.systemId,
    title,
    toolId,
  }
}

export function resolveTutorialMarkdown(
  markdown: string,
  origin: string
): string {
  const baseUrl = origin || BASE_URL

  return markdown
    .replaceAll(BASE_URL_V1, `${baseUrl}/v1`)
    .replaceAll(BASE_URL, baseUrl)
}
