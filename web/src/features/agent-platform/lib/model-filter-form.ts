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
import { z } from 'zod'

export const clientModelFilterFormSchema = z.object({
  excluded_models: z.array(z.string()).superRefine((models, context) => {
    if (models.length > 1000) {
      context.addIssue({
        code: 'custom',
        message: 'Select at most 1000 models',
      })
      return
    }
    const normalized = new Set<string>()
    for (const model of models) {
      const id = model.trim()
      const characters = [...id]
      if (characters.length > 255) {
        context.addIssue({
          code: 'custom',
          message: 'Model IDs must be at most 255 characters',
        })
        return
      }
      if (
        [...model].some((character) => {
          const point = character.codePointAt(0) ?? 0
          return (
            point < 0x20 ||
            point === 0x7f ||
            (point >= 0xd800 && point <= 0xdfff)
          )
        })
      ) {
        context.addIssue({
          code: 'custom',
          message:
            'Model IDs must not contain control characters or invalid Unicode',
        })
        return
      }
      if (id) normalized.add(id)
    }
    if (
      new TextEncoder().encode(JSON.stringify([...normalized])).byteLength >
      60 * 1024
    ) {
      context.addIssue({
        code: 'custom',
        message: 'Model filter must not exceed 60 KiB',
      })
    }
  }),
})

export type ClientModelFilterFormValues = z.infer<
  typeof clientModelFilterFormSchema
>
