import { test } from 'node:test'
import assert from 'node:assert/strict'
import { roleClosure, resolve, sourceLabel, grantSQL, revokeSQL, scopeSQL } from './rbac.js'

// [user_name, role_name, granted_role_name]
const EDGES = [
  ['analyst', null, 'reader'],
  [null, 'reader', 'base'],
  [null, 'base', 'reader'], // a cycle, which must not hang
]

// [user_name, role_name, access_type, database, table, is_partial_revoke]
const GRANTS = [
  [null, 'reader', 'SELECT', 'demo', null, 0],
  ['analyst', null, 'INSERT', 'demo', 'events', 0],
  ['root', null, 'SELECT', null, null, 0],
  ['ghost', null, 'SELECT', 'demo', 'events', 1], // a partial revoke, not a grant
]

test('role closure follows role-to-role grants and survives cycles', () => {
  assert.deepEqual([...roleClosure('analyst', EDGES)].sort(), ['base', 'reader'])
  assert.deepEqual([...roleClosure('nobody', EDGES)], [])
})

test('a privilege granted straight to the grantee at this scope reads as direct', () => {
  const roles = roleClosure('analyst', EDGES)
  assert.deepEqual(resolve('analyst', 'INSERT', 'demo', 'events', GRANTS, roles), { direct: true })
})

test('a privilege reaching a table through a role on the database names both', () => {
  const roles = roleClosure('analyst', EDGES)
  const state = resolve('analyst', 'SELECT', 'demo', 'events', GRANTS, roles)
  assert.deepEqual(state, { direct: false, role: 'reader', scope: 'demo.*' })
  assert.equal(sourceLabel(state), 'via reader on demo.*')
})

test('a grant on *.* covers everything below it', () => {
  const state = resolve('root', 'SELECT', 'demo', 'events', GRANTS, new Set())
  assert.equal(sourceLabel(state), 'via *.*')
  assert.equal(state.role, null)
})

test('a narrow grant does not widen: demo.events is not demo.*', () => {
  assert.equal(resolve('analyst', 'INSERT', 'demo', null, GRANTS, new Set()), null)
})

test('a partial revoke is never read as a grant', () => {
  assert.equal(resolve('ghost', 'SELECT', 'demo', 'events', GRANTS, new Set()), null)
})

test('missing privileges resolve to nothing', () => {
  assert.equal(resolve('analyst', 'DROP', 'demo', 'events', GRANTS, roleClosure('analyst', EDGES)), null)
})

test('scopes and grantees are quoted in generated statements', () => {
  assert.equal(scopeSQL(null, null), '*.*')
  assert.equal(scopeSQL('demo', null), '`demo`.*')
  assert.equal(grantSQL('SELECT', 'demo', 'events', 'an`alyst'),
    'GRANT SELECT ON `demo`.`events` TO `an\\`alyst`')
  assert.equal(revokeSQL('SELECT', null, null, 'bob'), 'REVOKE SELECT ON *.* FROM `bob`')
})
