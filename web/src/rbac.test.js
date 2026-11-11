import { test } from 'node:test'
import assert from 'node:assert/strict'
import {
  roleClosure, resolve, sourceLabel, grantSQL, revokeSQL, scopeSQL, hostSQL, validAddr, ANY_HOST,
} from './rbac.js'

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

test('host statements spell out the whole list, in a stable order', () => {
  const none = { ip: [], names: [], regexp: [], like: [] }
  assert.equal(hostSQL('etl', ANY_HOST), 'ALTER USER `etl` HOST ANY')
  assert.equal(hostSQL('etl', none), 'ALTER USER `etl` HOST NONE')
  assert.equal(
    hostSQL('etl', { ...none, ip: ['192.168.1.5', '10.0.0.0/8'], names: ['box'], like: ["%.l'an"] }),
    "ALTER USER `etl` HOST IP '10.0.0.0/8', IP '192.168.1.5', LIKE '%.l\\'an', NAME 'box'",
  )
  assert.equal(
    hostSQL('etl', { ...none, ip: ['b', 'a'] }),
    hostSQL('etl', { ...none, ip: ['a', 'b'] }),
  )
})

test('validAddr takes addresses and subnets, nothing else', () => {
  for (const ok of ['10.0.0.1', '10.0.0.0/8', '0.0.0.0/0', '::1', '2001:db8::/32', '::ffff:10.0.0.1', '::/0']) {
    assert.ok(validAddr(ok), ok)
  }
  for (const bad of ['', 'box.example', '10.0.0', '10.0.0.256', '10.0.0.0/33', '10.0.0.0/', '1.2.3.4/8/9',
    "1.1.1.1' OR", '2001:db8::/129', 'abc']) {
    assert.ok(!validAddr(bad), bad)
  }
})
