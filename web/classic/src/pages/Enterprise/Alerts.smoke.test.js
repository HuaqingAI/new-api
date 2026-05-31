import assert from 'node:assert/strict';
import fs from 'node:fs';
import { describe, test } from 'node:test';
import {
  formatClassicAlertDepartments,
  formatClassicEnterpriseUserPrimary,
  formatClassicEnterpriseUserSecondary,
} from './alertHelpers.js';

describe('Enterprise alerts classic helpers', () => {
  test('formats department snapshots and falls back to unassigned', () => {
    assert.equal(
      formatClassicAlertDepartments([
        { department_id: 11, department_name: 'Engineering' },
        { department_id: 22, department_name: 'Security' },
      ]),
      'Engineering (#11), Security (#22)',
    );

    assert.equal(formatClassicAlertDepartments([]), '未归属');
  });

  test('prefers readable names for classic enterprise user labels', () => {
    assert.equal(
      formatClassicEnterpriseUserPrimary({
        display_name: 'Alice Zhang',
        username: 'alice_ops',
        user_id: 101,
      }),
      'Alice Zhang',
    );
    assert.equal(
      formatClassicEnterpriseUserSecondary(
        {
          display_name: 'Alice Zhang',
          username: 'alice_ops',
          user_id: 101,
        },
        (value) => value,
      ),
      'alice_ops · 用户 ID #101',
    );
    assert.equal(
      formatClassicEnterpriseUserPrimary({
        username: 'alice_ops',
        user_id: 101,
      }),
      'alice_ops',
    );
  });

  test('alerts page exposes rule management entry', () => {
    const source = fs.readFileSync(
      new URL('./Alerts.js', import.meta.url),
      'utf8',
    );
    assert.match(source, /告警规则列表/);
    assert.match(source, /saveAlertRule/);
    assert.match(source, /投递结果/);
    assert.match(source, /getAlertDeliveries/);
    assert.match(source, /resendAlertDelivery/);
    assert.match(source, /人工重发/);
    assert.match(source, /部门风险概览/);
    assert.match(source, /getDepartmentRiskSummary/);
    assert.match(source, /查看风险事件/);
    assert.match(source, /enterprise\.usage\.multi_dept_disclaimer/);
    assert.match(
      source,
      /unassigned_only:\s*value\?\.unassigned_only\s*\|\|\s*undefined/,
    );
    assert.match(source, /from:\s*value\?\.from\s*\|\|\s*undefined/);
    assert.match(source, /to:\s*value\?\.to\s*\|\|\s*undefined/);
    assert.match(
      source,
      /loadDeliveries\(\{\s*event_id:\s*record\.event_id\s*\}\)/,
    );
    assert.match(source, /formatClassicEnterpriseUserPrimary/);
    assert.match(source, /历史用户名快照/);
    assert.doesNotMatch(source, /type:\s*'webhook'\s*,\s*enabled:\s*false/);
  });

  test('department page source prefers readable names in member table', () => {
    const source = fs.readFileSync(
      new URL('./Department.js', import.meta.url),
      'utf8',
    );
    assert.match(source, /formatClassicEnterpriseUserPrimary/);
    assert.match(source, /formatClassicEnterpriseUserSecondary/);
  });
});
