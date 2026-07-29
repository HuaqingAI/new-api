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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  createQuotaSettingsSchema,
  enterpriseBudgetThresholdValidationMessage,
} from './quota-settings-section'

describe('Quota settings schema', () => {
  test('rejects invalid enterprise budget thresholds', () => {
    const schema = createQuotaSettingsSchema((key) => key)
    const result = schema.safeParse({
      QuotaForNewUser: 0,
      PreConsumedQuota: 0,
      QuotaForInviter: 0,
      QuotaForInvitee: 0,
      TopUpLink: '',
      general_setting: {
        docs_link: '',
      },
      quota_setting: {
        enable_free_model_pre_consume: true,
        enterprise_budget_warning_threshold: 95,
        enterprise_budget_critical_threshold: 80,
      },
    })

    assert.equal(result.success, false)
    if (result.success) return
    const issues = JSON.stringify(result.error.flatten().fieldErrors)
    assert.match(issues, new RegExp(enterpriseBudgetThresholdValidationMessage))
  })

  test('accepts valid enterprise budget thresholds', () => {
    const schema = createQuotaSettingsSchema((key) => key)
    const result = schema.safeParse({
      QuotaForNewUser: 0,
      PreConsumedQuota: 0,
      QuotaForInviter: 0,
      QuotaForInvitee: 0,
      TopUpLink: '',
      general_setting: {
        docs_link: '',
      },
      quota_setting: {
        enable_free_model_pre_consume: true,
        enterprise_budget_warning_threshold: 80,
        enterprise_budget_critical_threshold: 95,
      },
    })

    assert.equal(result.success, true)
  })
})
