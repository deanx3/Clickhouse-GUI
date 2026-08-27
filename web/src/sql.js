// ClickHouse cannot bind an identifier as a query parameter, so every database,
// table, user and role name spliced into a statement goes through ident().
// Values never do - they ride along as {name:Type} parameters instead.
//
// SQL is composed here on the client, not on the server, so that the statement
// shown to the user before it runs is literally the statement that gets sent.
export function ident(name) {
  return '`' + String(name).replace(/\\/g, '\\\\').replace(/`/g, '\\`') + '`'
}

export const qualified = (db, table) => `${ident(db)}.${ident(table)}`

export const previewSQL = (db, table, limit, offset = 0) => {
  const rows = Number(limit) || 100
  const skip = Number(offset) || 0
  return `SELECT *\nFROM ${qualified(db, table)}\nLIMIT ${rows}${skip ? ` OFFSET ${skip}` : ''}`
}

export const literal = (s) =>
  `'${String(s).replace(/\\/g, '\\\\').replace(/'/g, "\\'")}'`
