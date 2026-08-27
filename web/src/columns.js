import {
  Hash, Type, CalendarClock, ToggleLeft, Braces, Binary, MapPin, KeyRound,
} from 'lucide-vue-next'

// A column's icon comes from its ClickHouse type, so the shape of a table is
// readable before you read a single name.
export function columnIcon(type = '') {
  const t = type.replace(/^(Nullable|LowCardinality)\(/, '')
  if (/^(U?Int|Float|Decimal)/.test(t)) return Hash
  if (/^(Date|DateTime)/.test(t)) return CalendarClock
  if (/^Bool/.test(t)) return ToggleLeft
  if (/^(Array|Map|Tuple|Nested|JSON|Object)/.test(t)) return Braces
  if (/^(UUID|IPv[46])/.test(t)) return KeyRound
  if (/^(FixedString|String|Enum)/.test(t)) return Type
  if (/^Point|Ring|Polygon/.test(t)) return MapPin
  return Binary
}

// 1000 -> 1K, 1234567 -> 1.2M. Row counts are context, not a measurement.
export function compact(n) {
  // system.tables reports null for views and system tables, and Number(null)
  // is 0 - which would render a confident, wrong "0 rows".
  if (n === null || n === undefined || n === '') return ''
  const v = Number(n)
  if (!Number.isFinite(v)) return ''
  if (v < 1000) return String(v)
  if (v < 1e6) return `${(v / 1e3).toFixed(v < 1e4 ? 1 : 0)}K`
  if (v < 1e9) return `${(v / 1e6).toFixed(1)}M`
  return `${(v / 1e9).toFixed(1)}B`
}
