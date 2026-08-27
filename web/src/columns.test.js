import { test } from 'node:test'
import assert from 'node:assert/strict'
import { compact } from './columns.js'

test('row counts compact without lying about magnitude', () => {
  assert.equal(compact(0), '0')
  assert.equal(compact(999), '999')
  assert.equal(compact(1000), '1.0K')
  assert.equal(compact(12345), '12K')
  assert.equal(compact(1234567), '1.2M')
  assert.equal(compact(3e9), '3.0B')
  assert.equal(compact(null), '')
  assert.equal(compact('1500'), '1.5K')
})
