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
import { isRedirect } from '@tanstack/react-router'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import { describe, test } from 'node:test'
import { Route as EnterpriseAlertsRoute } from '@/routes/_authenticated/enterprise-alerts/index'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'
import {
  buildDeliveryFiltersAfterResend,
  buildAlertRulePayload,
  describeAlertRuleSecretStatus,
  enterpriseAlertsSearchSchema,
  formatDeliveryStatus,
  formatTriggerSource,
  formatDeliveryTraceSummary,
  formatDepartmentSnapshot,
  mapAlertFilterFormToSearch,
} from './index'
import {
  alertDeliveriesListQueryKey,
  alertRulesListQueryKey,
  enterpriseAlertDeliveriesQueryKey,
  enterpriseAlertRulesQueryKey,
} from './api'

describe('Enterprise alerts feature', () => {
  test('maps form filters into API search params', () => {
    assert.deepEqual(
      mapAlertFilterFormToSearch({
        tenant_id: '7',
        department_id: '11',
        user_id: '99',
        username: 'alice',
        model_name: 'gpt-4o-mini',
        risk_type: 'sensitive_words',
        from: '1717117200',
        to: '1717203600',
        page_size: '50',
      }),
      {
        tenant_id: 7,
        department_id: 11,
        user_id: 99,
        username: 'alice',
        model_name: 'gpt-4o-mini',
        risk_type: 'sensitive_words',
        from: 1717117200,
        to: 1717203600,
        page: 1,
        page_size: 50,
      }
    )
  })

  test('search schema preserves page and page size defaults', () => {
    const parsed = enterpriseAlertsSearchSchema.parse({
      department_id: '11',
      page: '2',
      page_size: '50',
    })
    assert.equal(parsed.department_id, 11)
    assert.equal(parsed.page, 2)
    assert.equal(parsed.page_size, 50)
  })

  test('formats department snapshot for multi-department events and falls back to unassigned', () => {
    assert.equal(
      formatDepartmentSnapshot(
        {
          department_snapshot: [
            {
              department_id: 11,
              department_name: 'Engineering',
              external_source: 'manual',
              status: 1,
            },
            {
              department_id: 22,
              department_name: 'Security',
              external_source: 'manual',
              status: 1,
            },
          ],
        },
        'Unassigned'
      ),
      'Engineering (#11), Security (#22)'
    )

    assert.equal(
      formatDepartmentSnapshot({ department_snapshot: [] }, 'Unassigned'),
      'Unassigned'
    )
  })

  test('builds feature-scoped query keys for rules mutations and queries', () => {
    assert.deepEqual(enterpriseAlertRulesQueryKey, [
      'enterprise',
      'alerts',
      'rules',
    ])
    assert.deepEqual(alertRulesListQueryKey(7), [
      'enterprise',
      'alerts',
      'rules',
      7,
    ])
    assert.deepEqual(enterpriseAlertDeliveriesQueryKey, [
      'enterprise',
      'alerts',
      'deliveries',
    ])
    assert.deepEqual(alertDeliveriesListQueryKey({ tenant_id: 7, page: 1 }), [
      'enterprise',
      'alerts',
      'deliveries',
      { tenant_id: 7, page: 1 },
    ])
  })

  test('formats delivery status and trace summary for read-only delivery views', () => {
    const t = (value: string) => value
    assert.equal(formatDeliveryStatus('final_failed', t), 'Final failed')
    assert.equal(formatTriggerSource('manual_resend', t), 'Manual resend')
    assert.equal(
      formatDeliveryTraceSummary(
        {
          trace: {
            event_id: 1,
            request_id: 'req-1',
            tenant_id: 7,
            username: 'alice',
            model_name: 'gpt-4o-mini',
            risk_type: 'abuse',
            action_result: 'blocked',
            event_created_at: 1717117200,
            department_snapshot: [],
            department_summary: 'Engineering (#11)',
            event_summary: 'policy only',
            rule_id: 1,
            rule_name: 'Critical abuse',
            detail_route: '/enterprise-alerts?event_id=1',
            detail_api_path: '/api/enterprise/alerts/events?tenant_id=7',
          },
        },
        'No trace details yet'
      ),
      'Engineering (#11) · req-1 · /enterprise-alerts?event_id=1'
    )
    assert.equal(
      formatDeliveryTraceSummary({ trace: undefined }, 'No trace details yet'),
      'No trace details yet'
    )
  })

  test('deliveries query key preserves resend filters', () => {
    assert.deepEqual(
      alertDeliveriesListQueryKey({
        tenant_id: 7,
        status: 'final_failed',
        trigger_source: 'manual_resend',
        page: 1,
      }),
      [
        'enterprise',
        'alerts',
        'deliveries',
        {
          tenant_id: 7,
          status: 'final_failed',
          trigger_source: 'manual_resend',
          page: 1,
        },
      ]
    )
  })

  test('resend clears blocking status filters so old and new deliveries can be reviewed together', () => {
    assert.deepEqual(
      buildDeliveryFiltersAfterResend({
        status: 'final_failed',
        channel_type: 'webhook',
        trigger_source: 'manual_resend',
      }),
      {
        status: '',
        channel_type: 'webhook',
        trigger_source: '',
      }
    )
  })

  test('builds sanitized rule payloads while preserving optional false values', () => {
    assert.deepEqual(
      buildAlertRulePayload(
        {
          id: 9,
          name: ' Abusive output ',
          enabled: false,
          riskTypes: 'abuse\nsensitive_words\nabuse',
          departmentIds: '11, 22, invalid',
          dedupeWindowSeconds: '300',
          emailEnabled: true,
          emailReceivers: 'ops@example.com\nowner@example.com',
          webhookEnabled: false,
          webhookUrl: 'https://hooks.example.com/alerts?token=secret',
          webhookSecret: '',
          webhookSecretConfigured: true,
          webhookSecretMasked: '******cret',
          dingtalkEnabled: true,
          dingtalkRobotUrl: 'https://oapi.dingtalk.com/robot/send?access_token=token',
          dingtalkRobotSecret: 'robot-secret',
          dingtalkSecretConfigured: false,
          dingtalkSecretMasked: '',
        },
        7
      ),
      {
        id: 9,
        tenant_id: 7,
        name: 'Abusive output',
        enabled: false,
        risk_types: ['abuse', 'sensitive_words', 'abuse'],
        department_ids: [11, 22],
        dedupe_window_seconds: 300,
        channel_configs: [
          {
            type: 'email',
            enabled: true,
            receivers: ['ops@example.com', 'owner@example.com'],
          },
          {
            type: 'webhook',
            enabled: false,
            webhook_url: 'https://hooks.example.com/alerts?token=secret',
          },
          {
            type: 'dingtalk_robot',
            enabled: true,
            dingtalk_robot_url: 'https://oapi.dingtalk.com/robot/send?access_token=token',
            dingtalk_robot_secret: 'robot-secret',
          },
        ],
      }
    )
  })

  test('describes configured secret state without revealing plaintext', () => {
    assert.equal(
      describeAlertRuleSecretStatus(
        true,
        '******cret',
        'Secret already configured',
        'No secret configured'
      ),
      'Secret already configured (******cret)'
    )
    assert.equal(
      describeAlertRuleSecretStatus(
        false,
        '',
        'Secret already configured',
        'No secret configured'
      ),
      'No secret configured'
    )
  })

  test('build payload keeps zero dedupe values and drops negative ones', () => {
    assert.equal(
      buildAlertRulePayload(
        {
          id: undefined,
          name: 'Zero dedupe',
          enabled: true,
          riskTypes: 'abuse',
          departmentIds: '',
          dedupeWindowSeconds: '0',
          emailEnabled: true,
          emailReceivers: 'ops@example.com',
          webhookEnabled: false,
          webhookUrl: '',
          webhookSecret: '',
          webhookSecretConfigured: false,
          webhookSecretMasked: '',
          dingtalkEnabled: false,
          dingtalkRobotUrl: '',
          dingtalkRobotSecret: '',
          dingtalkSecretConfigured: false,
          dingtalkSecretMasked: '',
        }
      ).dedupe_window_seconds,
      0
    )

    assert.equal(
      buildAlertRulePayload(
        {
          id: undefined,
          name: 'Negative dedupe',
          enabled: true,
          riskTypes: 'abuse',
          departmentIds: '',
          dedupeWindowSeconds: '-5',
          emailEnabled: true,
          emailReceivers: 'ops@example.com',
          webhookEnabled: false,
          webhookUrl: '',
          webhookSecret: '',
          webhookSecretConfigured: false,
          webhookSecretMasked: '',
          dingtalkEnabled: false,
          dingtalkRobotUrl: '',
          dingtalkRobotSecret: '',
          dingtalkSecretConfigured: false,
          dingtalkSecretMasked: '',
        }
      ).dedupe_window_seconds,
      0
    )
  })

  test('route guard redirects non-admin users and allows admins', () => {
    const { auth } = useAuthStore.getState()
    const previousUser = auth.user

    try {
      auth.setUser({
        id: 1001,
        username: 'member',
        role: ROLE.USER,
      })

      let redirected: unknown = null
      try {
        EnterpriseAlertsRoute.options.beforeLoad?.({} as never)
      } catch (error) {
        redirected = error
      }

      assert.ok(isRedirect(redirected))
      if (isRedirect(redirected)) {
        assert.equal(redirected.options.to, '/403')
      }

      auth.setUser({
        id: 1002,
        username: 'admin',
        role: ROLE.ADMIN,
      })

      assert.doesNotThrow(() =>
        EnterpriseAlertsRoute.options.beforeLoad?.({} as never)
      )
    } finally {
      auth.setUser(previousUser)
    }
  })

  test('mapped filters never include sensitive raw content fields', () => {
    const mapped = mapAlertFilterFormToSearch({
      tenant_id: '',
      department_id: '',
      user_id: '',
      username: '',
      model_name: '',
      risk_type: '',
      from: '',
      to: '',
      page_size: '20',
    })

    assert.ok(!('prompt' in mapped))
    assert.ok(!('messages' in mapped))
  })

  test('enterprise alerts page source keeps V1 workflow guidance and secret-safe messaging', () => {
    const source = fs.readFileSync(
      new URL('./index.tsx', import.meta.url),
      'utf8'
    )

    for (const expected of [
      'Deliveries',
      'Review notification delivery results, retry timing, and trace lookup hints without exposing secrets or raw prompts.',
      'Show final failed only',
      'Retry source',
      'Parent delivery',
      'Resend created',
      'Existing manual resend is still pending',
      'Manual resend',
      'No deliveries yet',
      'No final failed deliveries',
      'Alert deliveries will appear here after the background dispatcher runs.',
      'No trace details yet',
      'No error',
      'Alert Rules',
      'Configure department-aware risk notifications. Email is required for the V1 alert loop.',
      'Optional channels can stay disabled. They must not block email-based alert recording.',
      'No alert rules yet',
      'Create the first rule to notify owners when risky content is detected.',
      'Email Channel Enabled',
      'Webhook Channel',
      'DingTalk Robot Channel',
      'Secret already configured',
      'No secret configured',
    ]) {
      assert.match(source, new RegExp(escapeRegExp(expected)))
    }

    assert.match(source, /describeAlertRuleSecretStatus\(/)
    assert.match(source, /buildAlertRulePayload\(draft, normalizedSearch\.tenant_id\)/)
    assert.match(source, /formatDeliveryTraceSummary\(/)
    assert.doesNotMatch(source, /plain-secret/i)
    assert.doesNotMatch(source, /token=secret/i)
  })
})

function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
