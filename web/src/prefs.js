// Small preferences that should survive a reload. Storage can throw in a
// locked-down browser, so every read and write is guarded and falls back to
// the default rather than taking the app down with it.
const KEY = 'chgui.prefs'

const read = () => {
  try { return JSON.parse(localStorage.getItem(KEY)) ?? {} } catch { return {} }
}

let cache = read()

export const pref = (name, fallback) => cache[name] ?? fallback

export function setPref(name, value) {
  cache = { ...cache, [name]: value }
  try { localStorage.setItem(KEY, JSON.stringify(cache)) } catch { /* not fatal */ }
}

// The modifier the platform prints on its keys, for shortcut hints.
const mac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
export const modKey = mac ? '⌘' : 'Ctrl'
