import { ref } from 'vue'

// Query history lives in the browser, not on the server: it is a personal
// convenience, and putting it server-side would mean inventing per-user
// storage that this app deliberately does not have.
const KEY = 'chgui.history'
const LIMIT = 200

const load = () => {
  try { return JSON.parse(localStorage.getItem(KEY)) ?? [] } catch { return [] }
}

export const history = ref(load())

const save = () => {
  try { localStorage.setItem(KEY, JSON.stringify(history.value)) } catch { /* not fatal */ }
}

export function record(entry) {
  const sql = entry.sql.trim()
  if (!sql) return
  // Re-running the same statement moves it up rather than stacking duplicates.
  const at = history.value.findIndex((e) => e.sql === sql)
  if (at >= 0) history.value.splice(at, 1)
  history.value.unshift({ id: `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`, ...entry, sql })
  if (history.value.length > LIMIT) history.value.length = LIMIT
  save()
}

export function forget(id) {
  const at = history.value.findIndex((e) => e.id === id)
  if (at >= 0) history.value.splice(at, 1)
  save()
}

export function clearHistory() {
  history.value = []
  save()
}

// Buckets for the panel's date headings.
export function bucketOf(at) {
  const day = new Date(at); day.setHours(0, 0, 0, 0)
  const today = new Date(); today.setHours(0, 0, 0, 0)
  const days = Math.round((today - day) / 86400000)
  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return 'This week'
  return 'Earlier'
}

export const clockOf = (at) =>
  new Date(at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
