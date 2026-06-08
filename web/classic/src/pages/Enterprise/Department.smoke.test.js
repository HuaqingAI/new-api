import assert from 'node:assert/strict';
import fs from 'node:fs';
import { describe, test } from 'node:test';
import {
  formatClassicEnterpriseUserPrimary,
  formatClassicEnterpriseUserSecondary,
} from './alertHelpers.js';

describe('Enterprise department classic smoke', () => {
  test('formats department members with readable names before username and user id', () => {
    const t = (value) => value;

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
        t,
      ),
      'alice_ops · 用户 ID #101',
    );
    assert.equal(
      formatClassicEnterpriseUserPrimary({
        display_name: '',
        username: 'alice_ops',
        user_id: 101,
      }),
      'alice_ops',
    );
    assert.equal(
      formatClassicEnterpriseUserPrimary({
        display_name: '',
        username: '',
        user_id: 101,
      }),
      '#101',
    );
    assert.equal(
      formatClassicEnterpriseUserSecondary(
        {
          display_name: 'Alice Zhang',
          username: 'alice_ops',
        },
        t,
      ),
      'alice_ops',
    );
    assert.equal(formatClassicEnterpriseUserPrimary({}), '-');
    assert.equal(formatClassicEnterpriseUserSecondary({}, t), '-');
    assert.notEqual(formatClassicEnterpriseUserSecondary({}, t), '用户 ID #-');
  });

  test('department page member table consumes the shared enterprise user formatter', () => {
    const source = fs.readFileSync(
      new URL('./Department.js', import.meta.url),
      'utf8',
    );

    assert.match(source, /title:\s*t\('用户'\)/);
    assert.match(source, /formatClassicEnterpriseUserPrimary\(record\)/);
    assert.match(source, /formatClassicEnterpriseUserSecondary\(record,\s*t\)/);
  });
});
