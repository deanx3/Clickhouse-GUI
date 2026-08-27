<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import {
  ArrowDown, ArrowUp, ChevronsUpDown, Copy, Download, Maximize2, Minimize2, Rows3,
} from 'lucide-vue-next'
import { copy } from './clipboard'
import { toast } from './toast'
import Btn from './ui/Btn.vue'

const props = defineProps({ result: Object, sql: String })
const emit = defineEmits(['fullscreen'])

const full = ref(false)
const sort = ref(null)          // { col, dir }
const widths = ref({})          // column index -> px
const menu = ref(null)

const meta = computed(() => props.result?.meta ?? [])
const numeric = (t) => /^(Nullable\()?(U?Int|Float|Decimal)/.test(t)

// Sorting is over the rows already fetched, and says so in the tooltip: it is
// not a re-query, so it orders this page only.
const rows = computed(() => {
  const data = props.result?.data ?? []
  if (!sort.value) return data
  const { col, dir } = sort.value
  const num = numeric(meta.value[col]?.type)
  return [...data].sort((a, b) => {
    const x = a[col], y = b[col]
    if (x === null) return 1
    if (y === null) return -1
    const r = num ? Number(x) - Number(y) : String(x).localeCompare(String(y))
    return dir === 'asc' ? r : -r
  })
})

function toggleSort(i) {
  const s = sort.value
  if (!s || s.col !== i) sort.value = { col: i, dir: 'asc' }
  else if (s.dir === 'asc') sort.value = { col: i, dir: 'desc' }
  else sort.value = null
}

/* --- column resizing ---------------------------------------------------- */
let drag = null

function startResize(e, i) {
  const th = e.target.closest('th')
  drag = { i, x: e.clientX, w: th.getBoundingClientRect().width }
  document.addEventListener('mousemove', onResize)
  document.addEventListener('mouseup', endResize)
  document.body.style.cursor = 'col-resize'
}

function onResize(e) {
  if (!drag) return
  widths.value = { ...widths.value, [drag.i]: Math.max(56, drag.w + e.clientX - drag.x) }
}

function endResize() {
  drag = null
  document.removeEventListener('mousemove', onResize)
  document.removeEventListener('mouseup', endResize)
  document.body.style.cursor = ''
}

onBeforeUnmount(endResize)

/* --- cell menu ----------------------------------------------------------- */
const openMenu = (e, row, value) => {
  menu.value = { x: Math.min(e.clientX, innerWidth - 200), y: e.clientY, row, value }
}

const cellText = (v) => (v === null ? 'NULL' : String(v))

function copyRow() {
  copy(menu.value.row.map(cellText).join('\t'), 'Row copied')
  menu.value = null
}

function copyCell() {
  copy(cellText(menu.value.value), 'Cell copied')
  menu.value = null
}

/* --- export -------------------------------------------------------------- */
const csvCell = (v) => {
  if (v === null) return ''
  const s = String(v)
  return /["\n,]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

function download(kind) {
  const names = meta.value.map((c) => c.name)
  const text = kind === 'csv'
    ? [names.map(csvCell).join(','), ...rows.value.map((r) => r.map(csvCell).join(','))].join('\n')
    : JSON.stringify(rows.value.map((r) => Object.fromEntries(r.map((v, i) => [names[i], v]))), null, 2)

  const url = URL.createObjectURL(new Blob([text], { type: kind === 'csv' ? 'text/csv' : 'application/json' }))
  const a = document.createElement('a')
  a.href = url
  a.download = `chgui-result.${kind}`
  a.click()
  URL.revokeObjectURL(url)
  toast(`Exported ${rows.value.length} rows as ${kind.toUpperCase()}`)
}

function toggleFull() {
  full.value = !full.value
  emit('fullscreen', full.value)
}

/* --- footer statistics --------------------------------------------------- */
const ms = (s) => (s < 1 ? `${Math.round(s * 1000)} ms` : `${s.toFixed(2)} s`)
const int = (n) => Number(n).toLocaleString()

function bytes(n) {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = Number(n)
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${i === 0 ? v : v.toFixed(1)} ${units[i]}`
}
</script>

<template>
  <div class="results" :class="{ full }">
    <div v-if="!meta.length" class="ok">
      <Rows3 :size="17" />
      <span>Statement ran. No rows returned.</span>
    </div>

    <template v-else>
      <div class="toolbar">
        <span class="lead">Result</span>
        <span v-if="sort" class="chip">
          sorted by {{ meta[sort.col].name }} {{ sort.dir }}
          <button aria-label="Clear sort" @click="sort = null">&times;</button>
        </span>
        <span class="spacer" />
        <Btn size="sm" variant="ghost" data-tip="Copy as TSV"
             @click="copy([meta.map(c => c.name).join('\t'), ...rows.map(r => r.map(cellText).join('\t'))].join('\n'), 'Result copied')">
          <template #icon><Copy :size="13" /></template>
        </Btn>
        <Btn size="sm" variant="ghost" @click="download('csv')">
          <template #icon><Download :size="13" /></template>CSV
        </Btn>
        <Btn size="sm" variant="ghost" @click="download('json')">JSON</Btn>
        <Btn size="sm" icon variant="ghost" :data-tip="full ? 'Exit fullscreen' : 'Fullscreen'"
             @click="toggleFull">
          <template #icon><component :is="full ? Minimize2 : Maximize2" :size="13" /></template>
        </Btn>
      </div>

      <div class="grid">
        <table>
          <thead>
            <tr>
              <th v-for="(c, i) in meta" :key="c.name"
                  :style="widths[i] ? { width: `${widths[i]}px` } : null">
                <button class="sorter" :data-tip="`Sort the ${rows.length} fetched rows`"
                        @click="toggleSort(i)">
                  <span class="cname ident">{{ c.name }}</span>
                  <component
                    :is="sort?.col === i ? (sort.dir === 'asc' ? ArrowUp : ArrowDown) : ChevronsUpDown"
                    :size="11" :class="['sort-ico', { active: sort?.col === i }]" />
                </button>
                <span class="ctype">{{ c.type }}</span>
                <span class="handle" @mousedown.prevent="startResize($event, i)" />
              </th>
              <th class="filler" />
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in rows" :key="i">
              <td v-for="(v, j) in row" :key="j"
                  :title="v === null ? '' : String(v)"
                  :style="widths[j] ? { width: `${widths[j]}px`, maxWidth: `${widths[j]}px` } : null"
                  :class="{ num: numeric(meta[j].type), nil: v === null }"
                  @contextmenu.prevent="openMenu($event, row, v)">
                {{ v === null ? 'NULL' : v }}
              </td>
              <td class="filler" />
            </tr>
          </tbody>
        </table>
      </div>

      <footer class="stats">
        <span><b class="num">{{ int(result.rows) }}</b> rows</span>
        <template v-if="result.statistics">
          <i />
          <span><b class="num">{{ ms(result.statistics.elapsed) }}</b></span>
          <i />
          <span><b class="num">{{ int(result.statistics.rows_read) }}</b> read</span>
          <i />
          <span><b class="num">{{ bytes(result.statistics.bytes_read) }}</b></span>
        </template>
      </footer>
    </template>

    <Teleport to="body">
      <div v-if="menu" class="cellmenu" :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
           @click.stop>
        <button @click="copyCell"><Copy :size="13" /> Copy cell</button>
        <button @click="copyRow"><Copy :size="13" /> Copy row</button>
      </div>
      <div v-if="menu" class="scrim" @click="menu = null" @contextmenu.prevent="menu = null" />
    </Teleport>
  </div>
</template>

<style scoped>
.results { display: flex; flex-direction: column; min-height: 0; height: 100%; }

.results.full {
  position: fixed;
  inset: 0;
  z-index: 50;
  background: var(--bg-base);
  animation: fade-in var(--t-fast) var(--ease);
}

.ok {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 22px;
  color: var(--text-secondary);
  font-size: var(--fs-base);
}

.ok svg { color: var(--success); }

.toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 5px var(--s2) 5px var(--s3);
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border-subtle);
}

.lead {
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: 8px;
  padding: 2px 4px 2px 8px;
  background: var(--accent-subtle);
  border-radius: var(--r-full);
  color: var(--accent-text);
  font-size: var(--fs-xs);
}

.chip button { background: none; border: 0; color: inherit; font-size: 13px; line-height: 1; padding: 0 3px; }
.spacer { flex: 1; }

.grid { flex: 1; overflow: auto; }
table { border-collapse: separate; border-spacing: 0; width: max-content; min-width: 100%; }

th {
  position: sticky;
  top: 0;
  z-index: 2;
  padding: 5px 0 6px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
  text-align: left;
  vertical-align: top;
  white-space: nowrap;
}

.sorter {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 0 12px;
  background: none;
  border: 0;
  color: var(--text-primary);
}

.cname { font-size: var(--fs-sm); font-weight: 500; }
.sort-ico { color: var(--text-disabled); transition: color var(--t-fast) var(--ease); }
.sorter:hover .sort-ico { color: var(--text-secondary); }
.sort-ico.active { color: var(--accent); }

.ctype {
  display: block;
  padding: 1px 12px 0;
  color: var(--text-disabled);
  font: 400 10px/1.3 var(--font-mono);
}

.handle {
  position: absolute;
  top: 0;
  right: 0;
  width: 7px;
  height: 100%;
  cursor: col-resize;
  border-right: 1px solid var(--border-subtle);
}

.handle:hover { border-right-color: var(--accent); }

td {
  height: var(--row-h);
  max-width: 420px;
  padding: 0 12px;
  border-bottom: 1px solid var(--border-subtle);
  font: 400 var(--fs-sm)/var(--row-h) var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Native offscreen skipping keeps thousands of rows fluid without a
   virtual scroller. */
tbody tr { content-visibility: auto; contain-intrinsic-size: auto var(--row-h); }
tbody tr:nth-child(even) { background: rgb(255 255 255 / 0.012); }
tbody tr:hover { background: var(--bg-hover); }

.num { text-align: right; font-variant-numeric: tabular-nums; }
.nil { color: var(--text-disabled); font-style: italic; }

/* Without a filler the browser spreads leftover width across the real
   columns, stretching the first one absurdly. */
.filler { width: 100%; padding: 0; border-bottom: 1px solid var(--border-subtle); }
thead .filler { background: var(--bg-panel); border-bottom-color: var(--border); }

.stats {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px var(--s4);
  background: var(--bg-panel);
  border-top: 1px solid var(--border-subtle);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
}

.stats b { color: var(--text-secondary); font-weight: 500; }
.stats i { width: 3px; height: 3px; border-radius: var(--r-full); background: var(--border-strong); }

.cellmenu {
  position: fixed;
  z-index: 80;
  padding: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-lg);
}

.cellmenu button {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 12px 7px 9px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  text-align: left;
}

.cellmenu button:hover { background: var(--bg-hover); color: var(--text-primary); }
.scrim { position: fixed; inset: 0; z-index: 79; }
</style>
