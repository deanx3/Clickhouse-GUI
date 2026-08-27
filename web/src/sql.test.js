import { test } from 'node:test'
import assert from 'node:assert/strict'
import { ident, previewSQL } from './sql.js'

test('identifiers are backquoted, and backquotes inside them are escaped', () => {
  assert.equal(ident('events'), '`events`')
  assert.equal(ident('a table'), '`a table`')
  assert.equal(ident('a`b'), '`a\\`b`')
  assert.equal(ident('a\\b'), '`a\\\\b`')
})

test('a name cannot close its own quote and append a statement', () => {
  assert.equal(
    previewSQL('demo', '`; DROP TABLE demo.events; --', 10),
    'SELECT *\nFROM `demo`.`\\`; DROP TABLE demo.events; --`\nLIMIT 10',
  )
})

test('paging appends OFFSET only past the first page, and coerces it too', () => {
  assert.equal(previewSQL('demo', 'events', 100, 0), 'SELECT *\nFROM `demo`.`events`\nLIMIT 100')
  assert.equal(previewSQL('demo', 'events', 100, 200), 'SELECT *\nFROM `demo`.`events`\nLIMIT 100 OFFSET 200')
  assert.match(previewSQL('demo', 'events', 100, '5; DROP TABLE x'), /LIMIT 100$/)
})

test('the row limit is coerced to a number', () => {
  assert.match(previewSQL('demo', 'events', '100; DROP TABLE demo.events'), /LIMIT 100$/)
  assert.match(previewSQL('demo', 'events', undefined), /LIMIT 100$/)
})
