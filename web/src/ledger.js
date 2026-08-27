import { ref } from 'vue'
import { query } from './api'

// Nothing here writes to ClickHouse until apply() runs. Staged statements are
// exactly what will be sent - the ledger shows the payload, not a summary of it.
export const staged = ref([])
export const history = ref([])

// Toggling a checkbox twice is a no-op, so staging the same statement twice
// takes it back off the queue instead of queueing it again.
export function stage(entry) {
  const at = staged.value.findIndex((e) => e.sql === entry.sql)
  if (at >= 0) staged.value.splice(at, 1)
  else staged.value.push({ ...entry, error: null })
}

export const isStaged = (sql) => staged.value.some((e) => e.sql === sql)
export const unstage = (at) => staged.value.splice(at, 1)
export const discard = () => { staged.value = [] }

// ClickHouse has no transaction around DDL, so a failure part-way through
// leaves the earlier statements applied. Stop there, keep the rest staged, and
// show the error against the statement that broke rather than pretending the
// whole batch rolled back.
export async function apply() {
  for (const entry of staged.value) entry.error = null

  while (staged.value.length) {
    const entry = staged.value[0]
    try {
      await query(entry.sql, entry.params)
      history.value.unshift({ sql: entry.sql, note: entry.note })
      staged.value.shift()
    } catch (e) {
      entry.error = e.message
      throw e
    }
  }
}
