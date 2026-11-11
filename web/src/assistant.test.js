import { test } from 'node:test'
import assert from 'node:assert/strict'
import { parts, isAccess, stepLabel } from './assistant.js'

test('parts separates prose from fenced statements', () => {
  assert.deepEqual(
    parts('Two rows.\n\n```sql\nSELECT 1\n```\nThen:\n```SQL\nSELECT 2\n```'),
    [
      { code: false, text: 'Two rows.' },
      { code: true, lang: 'sql', text: 'SELECT 1' },
      { code: false, text: 'Then:' },
      { code: true, lang: 'sql', text: 'SELECT 2' },
    ],
  )
  assert.deepEqual(parts('plain'), [{ code: false, text: 'plain' }])
  assert.deepEqual(parts(''), [])
})

test('parts keeps a statement whose fence never closed', () => {
  assert.deepEqual(parts('Try\n```sql\nSELECT 1'), [
    { code: false, text: 'Try' },
    { code: true, lang: 'sql', text: 'SELECT 1' },
  ])
})

test('isAccess picks out statements that belong in the ledger', () => {
  for (const sql of ['GRANT SELECT ON db.* TO bob', '  revoke ALL ON *.* FROM bob', 'CREATE USER bob',
    'ALTER ROLE r SETTINGS x = 1', 'DROP ROW POLICY p ON t', 'CREATE SETTINGS PROFILE p']) {
    assert.ok(isAccess(sql), sql)
  }
  for (const sql of ['SELECT 1', 'CREATE TABLE users (a Int8) ENGINE = Memory', 'DROP TABLE role', 'SHOW GRANTS']) {
    assert.ok(!isAccess(sql), sql)
  }
})

test('stepLabel survives malformed arguments', () => {
  assert.equal(stepLabel({ name: 'run_query', args: '{"sql":"SELECT 1"}' }), 'SELECT 1')
  assert.equal(stepLabel({ name: 'describe_table', args: '{"database":"d","table":"t"}' }), 'Read columns of d.t')
  assert.equal(stepLabel({ name: 'list_tables', args: 'oops' }), 'Listed tables')
})
