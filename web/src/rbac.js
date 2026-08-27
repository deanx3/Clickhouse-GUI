import { ident } from './sql.js'

// The columns of the grants matrix. ClickHouse has 206 access types, but it
// records the parent when you grant one - GRANT ALTER stores ALTER, not its
// forty children - so these eight cover ordinary table work.
export const PRIVILEGES = [
  'SELECT', 'INSERT', 'ALTER', 'CREATE', 'DROP', 'TRUNCATE', 'OPTIMIZE', 'SHOW',
]

// A scope is [database, table] where null means "any". system.grants stores it
// as two nullable columns; GRANT spells it *.*, db.* or db.table.
export const scopeId = (db, table) => JSON.stringify([db, table])
export const scopeLabel = (db, table) => `${db ?? '*'}.${table ?? '*'}`
export const scopeSQL = (db, table) =>
  `${db === null ? '*' : ident(db)}.${table === null ? '*' : ident(table)}`

// Every role reachable from a grantee, following role-to-role grants.
// edges are system.role_grants rows: [user_name, role_name, granted_role_name].
export function roleClosure(name, edges) {
  const seen = new Set()
  const queue = [name]
  while (queue.length) {
    const from = queue.shift()
    for (const [user, role, granted] of edges) {
      if ((user ?? role) !== from || seen.has(granted)) continue
      seen.add(granted) // also breaks cycles
      queue.push(granted)
    }
  }
  return seen
}

// Why does this grantee have this privilege here? Privileges arrive along two
// independent axes - through a role, and through a broader scope - so the
// answer names both. Returns null when they simply do not have it.
//
// grants are system.grants rows:
//   [user_name, role_name, access_type, database, table, is_partial_revoke]
export function resolve(grantee, priv, db, table, grants, roles) {
  let inherited = null

  for (const [user, role, access, gdb, gtable, partialRevoke] of grants) {
    if (partialRevoke || access !== priv) continue

    const holder = user ?? role
    if (holder !== grantee && !roles.has(holder)) continue

    // A null in the grant means "any", so it covers what we are asking about.
    // A value must match exactly: a grant on demo.events does not cover demo.*.
    if (gdb !== null && gdb !== db) continue
    if (gtable !== null && gtable !== table) continue

    const here = gdb === db && gtable === table
    if (here && holder === grantee) return { direct: true }

    inherited ??= {
      direct: false,
      role: holder === grantee ? null : holder,
      scope: here ? null : scopeLabel(gdb, gtable),
    }
  }
  return inherited
}

export function sourceLabel(state) {
  if (!state || state.direct) return ''
  if (state.role && state.scope) return `via ${state.role} on ${state.scope}`
  return `via ${state.role || state.scope}`
}

export const grantSQL = (priv, db, table, grantee) =>
  `GRANT ${priv} ON ${scopeSQL(db, table)} TO ${ident(grantee)}`

export const revokeSQL = (priv, db, table, grantee) =>
  `REVOKE ${priv} ON ${scopeSQL(db, table)} FROM ${ident(grantee)}`

export const grantRoleSQL = (role, grantee) => `GRANT ${ident(role)} TO ${ident(grantee)}`
export const revokeRoleSQL = (role, grantee) => `REVOKE ${ident(role)} FROM ${ident(grantee)}`

export const createRoleSQL = (name) => `CREATE ROLE ${ident(name)}`
export const dropRoleSQL = (name) => `DROP ROLE ${ident(name)}`
export const dropUserSQL = (name) => `DROP USER ${ident(name)}`

// The password binds as a parameter, so it never reaches the ledger, the
// screen, or query_log.
export const createUserSQL = (name) =>
  `CREATE USER ${ident(name)} IDENTIFIED WITH sha256_password BY {password:String}`
