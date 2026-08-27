// Splits whatever someone types for a server into scheme / host / port, so the
// form can keep them as separate fields. Omitting the port is the single most
// common mistake: ClickHouse serves HTTP on 8123, not 80.
export const DEFAULT_PORT = '8123'

export function parseTarget(input, fallbackPort = DEFAULT_PORT) {
  const raw = String(input ?? '').trim()
  if (!raw) return null

  // [::1]:8123 keeps its brackets so an IPv6 address is not split on its colons.
  const m = raw.match(/^(?:(https?):\/\/)?(\[[^\]]+\]|[^:/?#\s]+)(?::(\d+))?/i)
  if (!m) return null

  const [, scheme, host, port] = m
  return {
    scheme: (scheme || 'http').toLowerCase(),
    host,
    port: port || fallbackPort,
  }
}

export const buildURL = ({ scheme, host, port }) =>
  `${scheme || 'http'}://${host}:${port || DEFAULT_PORT}`
