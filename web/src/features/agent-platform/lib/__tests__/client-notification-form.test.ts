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
import { describe, expect, it } from 'vitest'

import { clientNotificationFormSchema } from '../client-notification-form'

const values = {
  title: 'Update',
  summary: '',
  content: 'Details',
  enabled: false,
}

describe('notification form contract', () => {
  it('trims short fields, normalizes CRLF, and preserves explicit false and body indentation', () => {
    expect(
      clientNotificationFormSchema.parse({
        ...values,
        title: '  Update  ',
        summary: '  Summary  ',
        content: '  A\r\nB  ',
      })
    ).toEqual({
      title: 'Update',
      summary: 'Summary',
      content: '  A\nB  ',
      enabled: false,
    })
  })
  it.each([
    { title: '🧠'.repeat(120) },
    { summary: '🧠'.repeat(240) },
    { content: '🧠'.repeat(10000) },
  ])('accepts Unicode code point limits: %#', (change) => {
    expect(
      clientNotificationFormSchema.safeParse({ ...values, ...change }).success
    ).toBe(true)
  })
  it.each([
    { title: ' ' },
    { title: '🧠'.repeat(121) },
    { title: 'A\nB' },
    { title: 'A\u2028B' },
    { summary: 'x'.repeat(241) },
    { content: '  \n  ' },
    { content: 'x'.repeat(10001) },
    { content: 'A\u0000B' },
    { content: 'A\ud800B' },
    { content: 'A\u0085B' },
    { enabled: null },
  ])(
    'rejects missing, oversized or invalid notification fields: %#',
    (change) => {
      expect(
        clientNotificationFormSchema.safeParse({ ...values, ...change }).success
      ).toBe(false)
    }
  )
})
