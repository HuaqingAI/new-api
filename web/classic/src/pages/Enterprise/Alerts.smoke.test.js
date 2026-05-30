import assert from 'node:assert/strict';
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
});
