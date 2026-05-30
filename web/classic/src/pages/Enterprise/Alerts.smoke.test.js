import assert from 'node:assert/strict';
import fs from 'node:fs';
import { describe, test } from 'node:test';
import { formatClassicAlertDepartments } from './alertHelpers.js';

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

  test('alerts page exposes rule management entry', () => {
    const source = fs.readFileSync(new URL('./Alerts.js', import.meta.url), 'utf8');
    assert.match(source, /告警规则列表/);
    assert.match(source, /saveAlertRule/);
    assert.match(source, /投递结果/);
    assert.match(source, /getAlertDeliveries/);
    assert.match(source, /resendAlertDelivery/);
    assert.match(source, /人工重发/);
  });
});
