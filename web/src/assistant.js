// Splits a reply into prose and fenced code, so each statement the assistant
// proposes can carry its own buttons. An unclosed fence (a reply cut short)
// still counts as code.
export function parts(text) {
  const out = []
  const fence = /```(\w*)[ \t]*\n?([\s\S]*?)(?:```|$)/g
  let at = 0
  for (const m of text.matchAll(fence)) {
    const before = text.slice(at, m.index).trim()
    if (before) out.push({ code: false, text: before })
    if (m[2].trim()) out.push({ code: true, lang: m[1].toLowerCase(), text: m[2].trim() })
    at = m.index + m[0].length
  }
  const rest = text.slice(at).trim()
  if (rest) out.push({ code: false, text: rest })
  return out
}

// Statements that change who may do what go to the ledger for review, like
// every other access change in the app, rather than into the editor.
export const isAccess = (sql) =>
  /^\s*(GRANT|REVOKE|(CREATE|ALTER|DROP)\s+(USER|ROLE|ROW\s+POLICY|POLICY|QUOTA|SETTINGS\s+PROFILE))\b/i.test(sql)

// One line describing a lookup the assistant made.
export function stepLabel({ name, args }) {
  let a = {}
  try { a = JSON.parse(args) } catch { /* show the bare name */ }
  if (name === 'list_tables') return a.database ? `Listed tables in ${a.database}` : 'Listed tables'
  if (name === 'describe_table') return `Read columns of ${a.database}.${a.table}`
  if (name === 'run_query') return a.sql ?? 'Ran a query'
  return name
}
