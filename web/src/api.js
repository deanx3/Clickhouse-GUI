import { beginRequest, requestOk, requestFailed } from './status'

// Every call goes through the Go server, which attaches the signed-in user's
// ClickHouse credentials from the session cookie.
async function post(path, body) {
  beginRequest()
  let res
  try {
    res = await fetch(path, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body ?? {}),
    })
  } catch (e) {
    requestFailed('Cannot reach the chgui server.', true)
    throw new Error('Cannot reach the chgui server.')
  }

  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    const message = data.error || `Request failed (${res.status})`
    requestFailed(message, res.status >= 500)
    const err = new Error(message)
    err.status = res.status
    throw err
  }
  requestOk()
  return data
}

export const login = (user, password, connection) =>
  post('/api/login', { user, password, connection })

export const logout = (connection) => post('/api/logout', { connection })
export const switchTo = (connection) => post('/api/switch', { connection })
export const addConnection = (body) => post('/api/connections/add', body)
export const removeConnection = (connection) => post('/api/connections/remove', { connection })

export async function connections() {
  const res = await fetch('/api/connections')
  return res.ok ? res.json() : { connections: [], active: '' }
}
export const query = (sql, params) => post('/api/query', { sql, params })

export async function me() {
  const res = await fetch('/api/me')
  return res.ok ? res.json() : null
}
