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

import { MODEL_FAMILIES, REASONING_EFFORTS } from '../model-reasoning-api'

export const modelReasoningFormSchema = z
  .object({
    rules: z.array(
      z.object({
        match_type: z.enum(['family', 'model']),
        match_value: z.string(),
        efforts: z
          .array(z.enum(REASONING_EFFORTS))
          .min(1, 'Select at least one reasoning effort')
          .max(6, 'Invalid model reasoning settings'),
      })
    ),
  })
  .superRefine((values, context) => {
    if (values.rules.length > 1000) {
      context.addIssue({
        code: 'custom',
        path: ['rules'],
        message: 'Add at most 1000 reasoning rules',
      })
      return
    }
    const targets = new Set<string>()
    for (const [index, rule] of values.rules.entries()) {
      const value = rule.match_value.trim()
      let message: string | undefined
      if (!value) message = 'Enter a model ID or select a model family'
      if ([...value].length > 255) {
        message = 'Model IDs must be at most 255 characters'
      }
      if (
        [...rule.match_value].some((character) => {
          const code = character.codePointAt(0) ?? 0
          return (
            code < 0x20 || code === 0x7f || (code >= 0xd800 && code <= 0xdfff)
          )
        })
      ) {
        message =
          'Model IDs must not contain control characters or invalid Unicode'
      }
      const target = rule.match_type === 'family' ? value.toLowerCase() : value
      if (
        rule.match_type === 'family' &&
        !MODEL_FAMILIES.some((family) => family === target)
      ) {
        message = 'Select a supported model family'
      }
      const key = `${rule.match_type}:${target}`
      if (targets.has(key)) {
        message = 'Each model or family can have only one rule'
      }
      targets.add(key)
      if (message) {
        context.addIssue({
          code: 'custom',
          path: ['rules', index, 'match_value'],
          message,
        })
      }
    }
    const normalized = values.rules.map((rule) => ({
      ...rule,
      match_value:
        rule.match_type === 'family'
          ? rule.match_value.trim().toLowerCase()
          : rule.match_value.trim(),
      efforts: REASONING_EFFORTS.filter((effort) =>
        rule.efforts.includes(effort)
      ),
    }))
    if (
      new TextEncoder().encode(
        JSON.stringify({ schema_version: 1, rules: normalized })
      ).byteLength >
      60 * 1024
    ) {
      context.addIssue({
        code: 'custom',
        path: ['rules'],
        message: 'Model reasoning settings are too large',
      })
    }
  })

export type ModelReasoningFormValues = z.infer<typeof modelReasoningFormSchema>
