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
import { usageLogsSearchSchema } from '@/routes/_authenticated/usage-logs/$section'
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import {
  getUsageLogsDepartmentContextLabel,
  getUsageLogsPageTitleKey,
} from './index'

describe('Usage logs route and section helpers', () => {
  test('coerces drill-down search params from the URL query string', () => {
    const parsed = usageLogsSearchSchema.parse({
      page: '2',
      pageSize: '50',
      startTime: '1748476800000',
      endTime: '1748563200000',
      departmentId: '7',
      departmentName: 'Engineering',
      username: 'alice',
    })

    assert.equal(parsed.page, 2)
    assert.equal(parsed.pageSize, 50)
    assert.equal(parsed.startTime, 1748476800000)
    assert.equal(parsed.endTime, 1748563200000)
    assert.equal(parsed.departmentId, 7)
    assert.equal(parsed.departmentName, 'Engineering')
    assert.equal(parsed.username, 'alice')
  })

  test('returns the correct page title for each usage logs section', () => {
    assert.equal(getUsageLogsPageTitleKey('common'), 'Common Logs')
    assert.equal(getUsageLogsPageTitleKey('drawing'), 'Drawing Logs')
    assert.equal(getUsageLogsPageTitleKey('task'), 'Task Logs')
  })

  test('formats department drill-down context without duplicating the label', () => {
    assert.equal(
      getUsageLogsDepartmentContextLabel({
        departmentId: 7,
        departmentName: 'Engineering',
      }),
      'Engineering'
    )
    assert.equal(
      getUsageLogsDepartmentContextLabel({
        departmentId: 7,
      }),
      '#7'
    )
    assert.equal(getUsageLogsDepartmentContextLabel({}), '')
  })
})
