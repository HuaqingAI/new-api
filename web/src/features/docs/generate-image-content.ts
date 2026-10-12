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
export type GenerateImageRelease = {
  skillName: 'generate-image'
  version: string
  downloadUrl: string
  pythonRequirement: string
}

export const GENERATE_IMAGE_RELEASE: GenerateImageRelease = {
  skillName: 'generate-image',
  version: '1.0',
  downloadUrl:
    'https://hqdtc.oss-us-west-1.aliyuncs.com/hth-skills/generate-image/generate-image.zip',
  pythonRequirement: 'Python 3.11+',
}

export function buildGenerateImageInstallPrompt(
  release: GenerateImageRelease
): string {
  return `请帮我在这台电脑上安装 generate-image（HTH 生图 Skill），安装到当前 Codex 能识别的用户级技能目录，以便在不同项目中使用。

安装包下载地址：
${release.downloadUrl}

发布版本：${release.version}

请直接执行能够完成的检查、下载、安装和离线校验，并遵循以下要求：

1. 先确认当前运行 Codex 的系统为 Windows 或 macOS，并检查可用的 Python 3.11+ 命令。当前官方推荐的用户级技能目录是用户主目录下的 .agents/skills。Windows 通常对应 %USERPROFILE%\\.agents\\skills；macOS 通常对应 ~/.agents/skills。

2. 检查是否已经安装同名 generate-image Skill。如果当前 Codex 已明确识别另一个兼容的用户级技能位置，先确认该位置再处理，不要把同名 Skill 同时安装到多个目录。相同版本且内容一致时保留已有安装；需要更新时先备份已有目录，备份必须放在 Codex 不扫描的目录中，不能放在 .agents/skills 等活动技能目录内，以免备份被识别为同名 Skill。保留已有 .env 等本地配置，并在结果中给出备份位置。

3. 将上面的完整 HTTPS 地址作为一个 URL 下载到临时目录。检查 HTTP 下载是否成功、文件是否是真正的 ZIP。不要把错误页面当成 ZIP 解压；下载检查失败时停止安装并说明实际错误，不要自行猜测其他下载地址。

4. 检查压缩包中的 SKILL.md、scripts/generate_image.py 和 references/models.md。完整安装 generate-image 目录，并确保最终是 <用户级技能目录>/generate-image/SKILL.md，不能变成 generate-image/generate-image/SKILL.md。解压时检查路径，不允许压缩包中的文件写到目标目录之外。

5. 先阅读安装后的 SKILL.md，再按照包内说明使用实际可用的 Python 3.11+ 命令执行脚本的 --help、--list-models 和一个不带远程输入的 --dry-run。当前发布包使用 Python 标准库，不需要额外安装 pip 依赖。这些检查只在本机执行，不调用生图接口，不生成图片。

6. 最后报告：实际安装目录、Skill 版本、下载检查结果、Python 版本、离线检查结果、是否有旧版本备份，以及还有什么步骤尚未完成。Codex 通常会自动发现新增 Skill；CLI 或 IDE 可通过 /skills 或 $generate-image 检查，桌面客户端可在 Skills 侧栏查看。如果当前会话未显示，请按当前客户端说明检查，必要时重启 Codex后再试。

7. 给我一条安装完成后可以发送的示例，例如：使用 $generate-image 生成一张白色陶瓷杯的产品照片，背景简洁，图片保存到当前工作目录的 output/generate-image 中。

如果当前会话不能访问网络、不能操作本机文件或不能写入用户级技能目录，请明确说明受限的步骤，并提供对应的手动操作方法，不要声称已经完成未执行的操作。`
}

export const GENERATE_IMAGE_EXAMPLES = [
  {
    id: 'text-to-image',
    title: '文生图',
    description: '用自然语言描述主体、光照与用途，生成产品照片。',
    prompt:
      '使用 $generate-image 的 gpt-image-2.5-flare 帮我生成一张白色陶瓷杯的产品照片。杯子放在浅色桌面上，柔和自然光，背景简洁，适合电商展示。',
  },
  {
    id: 'edit',
    title: '精细改图',
    description: '先把参考图附加到 Codex，或提供本机文件路径。',
    prompt:
      '使用 $generate-image 的 gpt-image-2.5-sunburst  编辑我提供的图片：把杯子的颜色改成深蓝色，保留杯子的形状、光照和背景。',
  },
  {
    id: 'composite',
    title: '多参考图合成',
    description: '先提供产品图和背景图，再说明需要保留与调整的内容。',
    prompt:
      '使用 $generate-image 的 doubao-seedream-5.0-pro 将我提供的产品图和背景图合成一张宣传图片。保持产品外观和比例，统一光照与透视，让产品自然融入背景。',
  },
  {
    id: 'gemini',
    title: 'Gemini 场景图',
    description: '描述场景、氛围和画面比例，生成横向风景图。',
    prompt:
      '使用 $generate-image 的 gemini-3.1-flash-image  生成一张清晨山谷的横向风景图，有薄雾和柔和阳光，画面比例 16:9。',
  },
]

export const GENERATE_IMAGE_MODELS = [
  {
    name: 'OpenAI',
    models: [
      'gpt-image-2',
      'gpt-image-2.5',
      'gpt-image-2.5-flare',
      'gpt-image-2.5-sunburst',
    ],
    description: '支持文生图与图片编辑，默认模型为 gpt-image-2.5。',
  },
  {
    name: 'Google Gemini',
    models: [
      'gemini-2.5-flash-image',
      'gemini-3-pro-image',
      'gemini-3.1-flash-image',
    ],
    description: '支持文生图、参考图编辑，以及比例与分辨率需求。',
  },
  {
    name: 'Doubao Seedream',
    models: ['doubao-seedream-5.0-pro'],
    description: '支持文生图、图片编辑与多参考图合成。',
  },
]
