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

export const clientNotificationFormSchema = z
  .object({
    title: z
      .string()
      .trim()
      .min(1, 'Notification title is required')
      .refine(
        (value) => [...value].length <= 120,
        'Notification title must be at most 120 characters'
      ),
    summary: z
      .string()
      .trim()
      .refine(
        (value) => [...value].length <= 240,
        'Notification summary must be at most 240 characters'
      ),
    content: z
      .string()
      .transform((value) => value.replaceAll('\r\n', '\n'))
      .refine(
        (value) => value.trim().length > 0,
        'Notification content is required'
      )
      .refine(
        (value) =>
          [...value].length <= 10000 &&
          new TextEncoder().encode(value).byteLength <= 40 * 1024,
        'Notification content must be at most 10000 characters and 40 KiB'
      ),
    enabled: z.boolean(),
  })
  .superRefine((values, context) => {
    for (const name of ['title', 'summary', 'content'] as const) {
      const invalid = [...values[name]].some((character) => {
        const point = character.codePointAt(0) ?? 0
        if (point >= 0xd800 && point <= 0xdfff) return true
        if (name !== 'title' && ['\n', '\t'].includes(character)) return false
        if (name === 'title' && (point === 0x2028 || point === 0x2029)) {
          return true
        }
        return point < 0x20 || (point >= 0x7f && point <= 0x9f)
      })
      if (invalid) {
        context.addIssue({
          code: 'custom',
          path: [name],
          message: 'Notification text contains invalid characters',
        })
      }
    }
  })

export type ClientNotificationFormValues = z.infer<
  typeof clientNotificationFormSchema
>
