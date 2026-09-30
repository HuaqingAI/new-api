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
import { claudeCodeMacosTutorial } from './claude-code-macos'
import { claudeCodeWindowsTutorial } from './claude-code-windows'
import { codexCliMacosTutorial } from './codex-cli-macos'
import { codexCliWindowsTutorial } from './codex-cli-windows'
import { codexDesktopMacosTutorial } from './codex-desktop-macos'
import { codexDesktopWindowsTutorial } from './codex-desktop-windows'
import { openclawMacosTutorial } from './openclaw-macos'
import { openclawWindowsTutorial } from './openclaw-windows'
import type { TutorialDoc, TutorialSystemId, TutorialToolId } from './types'

export { resolveTutorialMarkdown } from './content'

export type {
  TutorialDoc,
  TutorialSection,
  TutorialSystemId,
  TutorialToolId,
} from './types'

export const TUTORIAL_DOCS: readonly TutorialDoc[] = [
  claudeCodeWindowsTutorial,
  claudeCodeMacosTutorial,
  codexDesktopWindowsTutorial,
  codexDesktopMacosTutorial,
  codexCliWindowsTutorial,
  codexCliMacosTutorial,
  openclawWindowsTutorial,
  openclawMacosTutorial,
]

const tutorialMap = new Map(
  TUTORIAL_DOCS.map((doc) => [getTutorialKey(doc.toolId, doc.systemId), doc])
)

export function getTutorialKey(toolId: string, systemId: string): string {
  return `${toolId}-${systemId}`
}

export function getTutorialDoc(
  toolId: string,
  systemId: string
): TutorialDoc | undefined {
  return tutorialMap.get(getTutorialKey(toolId, systemId))
}

export function getTutorialPath(
  toolId: TutorialToolId,
  systemId: TutorialSystemId
): string {
  return `/docs/tutorials/${toolId}/${systemId}`
}
