<script setup>
import { computed, defineAsyncComponent, nextTick, ref, watch } from 'vue'
import {
  Play, Wand2, Eraser, History as HistoryIcon, ChevronLeft, ChevronRight,
  PanelLeftClose, PanelLeftOpen, TableProperties, Terminal,
} from 'lucide-vue-next'
import { query } from './api'
import { previewSQL, qualified } from './sql'
import { record } from './history'
import { pref, setPref } from './prefs'
import { copy } from './clipboard'
import { toast } from './toast'
import { startDrag, clamp } from './resizer'
import SchemaTree from './SchemaTree.vue'
const Editor = defineAsyncComponent(() => import('./Editor.vue'))
import Results from './Results.vue'
import TableInfo from './TableInfo.vue'
import History from './History.vue'
import State from './ui/State.vue'
import Btn from './ui/Btn.vue'

/* --- schema -------------------------------------------------------------- */
const databases = ref([])
const tables = ref({})
const columns = ref({})
const schemaLoading = ref(false)
const schemaError = ref('')
const selected = ref(pref('selected', null))

/* --- workspace ----------------------------------------------------------- */
const sql = ref(pref('sql', 'SELECT\n  name,\n  engine\nFROM system.databases\nORDER BY name'))
const result = ref(null)
const queryError = ref('')
const busy = ref(false)
const limit = ref(pref('limit', 100))
const tab = ref('query')
const page = ref(null)          // set only while browsing a generated SELECT
const editorRef = ref(null)

/* --- layout -------------------------------------------------------------- */
const sidebar = ref(pref('sidebar', 268))
const sidebarOpen = ref(pref('sidebarOpen', true))
const editorH = ref(pref('editorH', 240))
const historyOpen = ref(false)
const resultsFull = ref(false)

watch(sidebar, (v) => setPref('sidebar', v))
watch(sidebarOpen, (v) => setPref('sidebarOpen', v))
watch(editorH, (v) => setPref('editorH', v))
watch(limit, (v) => setPref('limit', v))
watch(sql, (v) => setPref('sql', v))
watch(selected, (v) => setPref('selected', v))

const dragSidebar = (e) => {
  const from = sidebar.value
  startDrag(e, { onMove: (dx) => { sidebar.value = clamp(from + dx, 200, 520) } })
}

const dragEditor = (e) => {
  const from = editorH.value
  startDrag(e, { cursor: 'row-resize', onMove: (_, dy) => { editorH.value = clamp(from + dy, 90, 640) } })
}

/* --- autocomplete schema: { 'db.table': ['col', ...] } ------------------- */
const schema = computed(() => {
  const out = {}
  for (const [db, rows] of Object.entries(tables.value)) {
    for (const [name] of rows) {
      const cols = columns.value[`${db}.${name}`]
      out[`${db}.${name}`] = cols ? cols.map(([c]) => c) : []
    }
  }
  return out
})

const currentMeta = computed(() =>
  selected.value ? (tables.value[selected.value.db] || []).find(([n]) => n === selected.value.table) : null)

const currentCols = computed(() =>
  selected.value ? columns.value[`${selected.value.db}.${selected.value.table}`] : null)

/* --- schema loading ------------------------------------------------------ */
loadSchema()

async function loadSchema() {
  schemaLoading.value = true
  schemaError.value = ''
  try {
    const r = await query('SELECT name FROM system.databases ORDER BY name')
    databases.value = r.data.map(([n]) => n)
    // Re-open whatever was selected last so a reload lands where you left off.
    if (selected.value) {
      await loadTables(selected.value.db)
      await loadColumns(selected.value.db, selected.value.table)
    }
  } catch (e) {
    schemaError.value = e.message
  } finally {
    schemaLoading.value = false
  }
}

async function loadTables(db) {
  if (tables.value[db]) return
  const r = await query(
    'SELECT name, engine, total_rows FROM system.tables WHERE database = {db:String} ORDER BY name',
    { db },
  )
  tables.value = { ...tables.value, [db]: r.data }
}

async function loadColumns(db, table) {
  const key = `${db}.${table}`
  if (columns.value[key]) return
  const r = await query(
    'SELECT name, type FROM system.columns' +
      ' WHERE database = {db:String} AND table = {t:String} ORDER BY position',
    { db, t: table },
  )
  columns.value = { ...columns.value, [key]: r.data }
}

async function refreshSchema() {
  tables.value = {}
  columns.value = {}
  await loadSchema()
  toast('Schema refreshed')
}

/* --- actions ------------------------------------------------------------- */
function pick(db, table) {
  selected.value = { db, table }
  loadColumns(db, table)
}

function browse(db, table) {
  pick(db, table)
  page.value = { db, table, offset: 0 }
  sql.value = previewSQL(db, table, limit.value)
  tab.value = 'query'
  run()
}

function generate(db, table) {
  pick(db, table)
  page.value = null
  sql.value = previewSQL(db, table, limit.value)
  tab.value = 'query'
  focusEditor()
}

function turnPage(step) {
  if (!page.value) return
  const offset = Math.max(0, page.value.offset + step * limit.value)
  page.value = { ...page.value, offset }
  sql.value = previewSQL(page.value.db, page.value.table, limit.value, offset)
  run()
}

function relimit() {
  if (!page.value) return
  page.value = { ...page.value, offset: 0 }
  sql.value = previewSQL(page.value.db, page.value.table, limit.value)
  run()
}

// Editing the statement by hand ends the browse session, because Prev/Next
// would otherwise rewrite whatever the user just typed.
watch(sql, (next) => {
  if (!page.value) return
  const { db, table, offset } = page.value
  if (next !== previewSQL(db, table, limit.value, offset)) page.value = null
})

async function run() {
  if (busy.value || !sql.value.trim()) return
  busy.value = true
  queryError.value = ''
  tab.value = 'query'
  const started = performance.now()
  try {
    const r = await query(sql.value)
    result.value = r
    record({
      sql: sql.value,
      at: Date.now(),
      ok: true,
      rows: r.rows ?? null,
      ms: Math.round((r.statistics?.elapsed ?? (performance.now() - started) / 1000) * 1000),
    })
  } catch (e) {
    queryError.value = e.message
    result.value = null
    record({ sql: sql.value, at: Date.now(), ok: false, error: e.message })
  } finally {
    busy.value = false
  }
}

async function formatQuery() {
  try {
    // Pull in one dialect rather than the package's default export, which
    // drags in every dialect it supports.
    const { formatDialect, postgresql } = await import('sql-formatter')
    sql.value = formatDialect(sql.value, { dialect: postgresql, keywordCase: 'upper', tabWidth: 2 })
    toast('Formatted')
  } catch {
    toast('Could not format this statement', 'error')
  }
}

function clearQuery() {
  sql.value = ''
  result.value = null
  queryError.value = ''
  focusEditor()
}

function newQuery() {
  clearQuery()
  page.value = null
}

const focusEditor = () => nextTick(() => editorRef.value?.focus())

function useHistory(text, immediate = false) {
  sql.value = text
  page.value = null
  if (immediate) run()
  else focusEditor()
}

const copyName = (db, table) =>
  copy(table ? qualified(db, table).replace(/`/g, '') : db, 'Name copied')

defineExpose({
  run, newQuery, formatQuery, clearQuery, refreshSchema, focusEditor, browse,
  toggleHistory: () => { historyOpen.value = !historyOpen.value },
  toggleSidebar: () => { sidebarOpen.value = !sidebarOpen.value },
  tableList: () => Object.entries(tables.value).flatMap(([db, rows]) => rows.map(([t]) => ({ db, table: t }))),
})
</script>

<template>
  <div class="workspace" :class="{ 'no-side': !sidebarOpen }"
       :style="{ '--side': `${sidebar}px`, '--hist': historyOpen ? '320px' : '0px' }">
    <aside v-if="sidebarOpen" class="side">
      <SchemaTree
        :databases="databases" :tables="tables" :columns="columns"
        :loading="schemaLoading" :error="schemaError" :selected="selected"
        @open-db="loadTables" @open-table="pick" @browse="browse"
        @structure="(db, t) => { pick(db, t); tab = 'structure' }"
        @generate="generate" @copy-name="copyName" @refresh="refreshSchema"
      />
      <span class="grip" @mousedown="dragSidebar" />
    </aside>

    <section class="main">
      <nav class="tabs">
        <Btn size="sm" icon variant="ghost"
             :data-tip="sidebarOpen ? 'Hide sidebar' : 'Show sidebar'"
             @click="sidebarOpen = !sidebarOpen">
          <template #icon>
            <component :is="sidebarOpen ? PanelLeftClose : PanelLeftOpen" :size="15" />
          </template>
        </Btn>

        <button :class="['tab', { on: tab === 'query' }]" @click="tab = 'query'">
          <Terminal :size="14" /> Query
        </button>
        <button :class="['tab', { on: tab === 'structure' }]" @click="tab = 'structure'">
          <TableProperties :size="14" /> Structure
          <span v-if="selected" class="ident crumb">{{ selected.table }}</span>
        </button>

        <span class="spacer" />

        <Btn size="sm" :variant="historyOpen ? 'subtle' : 'ghost'"
             @click="historyOpen = !historyOpen">
          <template #icon><HistoryIcon :size="14" /></template>History
        </Btn>
      </nav>

      <template v-if="tab === 'structure'">
        <TableInfo :db="selected?.db" :table="selected?.table"
                   :meta="currentMeta" :columns="currentCols"
                   @browse="browse" @generate="generate" />
      </template>

      <template v-else>
        <div class="editor-pane" :style="{ height: `${editorH}px` }">
          <Editor ref="editorRef" v-model="sql" :schema="schema" @run="run" @save="formatQuery" />
        </div>

        <div class="bar">
          <Btn variant="primary" size="sm" :loading="busy" @click="run">
            <template #icon><Play :size="13" /></template>
            Run
            <template #after><kbd class="on-primary">&#8984;&#9166;</kbd></template>
          </Btn>
          <Btn size="sm" variant="ghost" data-tip="Format SQL" @click="formatQuery">
            <template #icon><Wand2 :size="13" /></template>Format
          </Btn>
          <Btn size="sm" variant="ghost" data-tip="Clear editor" @click="clearQuery">
            <template #icon><Eraser :size="13" /></template>Clear
          </Btn>

          <span class="spacer" />

          <div v-if="page" class="pager">
            <Btn size="sm" icon variant="ghost" :disabled="!page.offset || busy"
                 data-tip="Previous page" @click="turnPage(-1)">
              <template #icon><ChevronLeft :size="14" /></template>
            </Btn>
            <span class="range num">{{ page.offset.toLocaleString() }}–{{ (page.offset + limit).toLocaleString() }}</span>
            <Btn size="sm" icon variant="ghost" :disabled="busy" data-tip="Next page"
                 @click="turnPage(1)">
              <template #icon><ChevronRight :size="14" /></template>
            </Btn>
          </div>
          <span v-else class="pager-off"
                data-tip-left data-tip="Paging applies to generated SELECTs, not hand-written SQL">
            no paging
          </span>

          <label class="limit">
            Rows
            <select v-model.number="limit" @change="relimit">
              <option :value="100">100</option>
              <option :value="1000">1000</option>
              <option :value="10000">10000</option>
            </select>
          </label>
        </div>

        <span class="hgrip" @mousedown="dragEditor" />

        <div class="out">
          <State v-if="queryError" kind="error" title="Query failed" :detail="queryError">
            <template #actions>
              <Btn size="sm" variant="primary" @click="run">Run again</Btn>
            </template>
          </State>
          <Results v-else-if="result" :result="result" :sql="sql"
                   @fullscreen="resultsFull = $event" />
          <State v-else-if="busy" kind="loading" title="Running" />
          <State v-else title="No results yet"
                 hint="Write a statement and press ⌘↵, or pick a table on the left." />
        </div>
      </template>
    </section>

    <History v-if="historyOpen" @use="useHistory" @close="historyOpen = false" />
  </div>
</template>

<style scoped>
.workspace {
  display: grid;
  grid-template-columns: var(--side) minmax(0, 1fr) var(--hist);
  height: 100%;
  min-height: 0;
}

.no-side { grid-template-columns: 0 minmax(0, 1fr) var(--hist); }

.side {
  position: relative;
  min-width: 0;
  background: var(--bg-panel);
  border-right: 1px solid var(--border);
}

.grip, .hgrip {
  position: absolute;
  z-index: 5;
  transition: background var(--t-fast) var(--ease);
}

.grip { top: 0; right: -3px; width: 6px; height: 100%; cursor: col-resize; }
.grip:hover { background: var(--accent); }

.main { display: flex; flex-direction: column; min-width: 0; min-height: 0; position: relative; }

.tabs {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 0 var(--s2);
  height: 40px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
}

.tab {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 28px;
  padding: 0 11px;
  background: none;
  border: 0;
  border-radius: var(--r-sm);
  color: var(--text-tertiary);
  font-size: var(--fs-sm);
  font-weight: 500;
  transition: background var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.tab:hover { background: var(--bg-hover); color: var(--text-secondary); }
.tab.on { background: var(--bg-active); color: var(--text-primary); }
.crumb { color: var(--text-disabled); font-size: var(--fs-xs); }
.spacer { flex: 1; }

.editor-pane { flex: none; min-height: 90px; border-bottom: 1px solid var(--border-subtle); }

.bar {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px var(--s3);
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
}

.hgrip { position: relative; height: 5px; margin-top: -3px; cursor: row-resize; }
.hgrip:hover { background: var(--accent); }

.pager { display: flex; align-items: center; gap: 2px; }
.range { min-width: 92px; color: var(--text-secondary); font-size: var(--fs-xs); text-align: center; }

.pager-off {
  color: var(--text-disabled);
  font-size: var(--fs-xs);
  padding: 0 8px;
  border-bottom: 1px dotted var(--border-strong);
  cursor: help;
}

.limit {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-left: var(--s3);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
}

.limit select {
  height: 26px;
  padding: 0 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-primary);
  font-size: var(--fs-xs);
}

.on-primary { background: rgb(0 0 0 / 0.16); border-color: rgb(0 0 0 / 0.22); color: inherit; opacity: 0.75; }

.out { flex: 1; min-height: 0; overflow: hidden; display: flex; flex-direction: column; }
.out > * { flex: 1; min-height: 0; }

@media (max-width: 860px) {
  .workspace { grid-template-columns: minmax(0, 1fr); }
  .side { display: none; }
  .limit, .pager-off { display: none; }
}
</style>
