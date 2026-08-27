<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import {
  Database, Table2, ChevronRight, Search, RefreshCw, X,
  Eye, Columns3, Copy, SquareCode,
} from 'lucide-vue-next'
import { columnIcon, compact } from './columns'
import Skeleton from './ui/Skeleton.vue'
import State from './ui/State.vue'
import Btn from './ui/Btn.vue'

const props = defineProps({
  databases: Array,        // [name, ...]
  tables: Object,          // db -> [[name, engine, rows], ...]
  columns: Object,         // 'db.table' -> [[name, type], ...]
  loading: Boolean,
  error: String,
  selected: Object,        // { db, table } | null
})

const emit = defineEmits([
  'open-db', 'open-table', 'browse', 'structure', 'generate', 'refresh', 'copy-name',
])

const filter = ref('')
const openDbs = ref(new Set())
const openTables = ref(new Set())
const menu = ref(null)

const q = computed(() => filter.value.trim().toLowerCase())

// While filtering, only databases with a matching name or a matching table
// survive, and matches auto-expand so a hit is never hidden behind a chevron.
const shown = computed(() => {
  if (!q.value) return props.databases.map((db) => ({ db, tables: props.tables[db] }))
  return props.databases
    .map((db) => {
      const hit = db.toLowerCase().includes(q.value)
      const rows = (props.tables[db] || []).filter(([n]) => n.toLowerCase().includes(q.value))
      return hit || rows.length ? { db, tables: hit ? props.tables[db] : rows } : null
    })
    .filter(Boolean)
})

const isOpen = (db) => q.value ? true : openDbs.value.has(db)
const key = (db, t) => `${db}.${t}`

function toggleDb(db) {
  const set = new Set(openDbs.value)
  set.has(db) ? set.delete(db) : (set.add(db), emit('open-db', db))
  openDbs.value = set
}

function toggleTable(db, table) {
  const k = key(db, table)
  const set = new Set(openTables.value)
  set.has(k) ? set.delete(k) : (set.add(k), emit('open-table', db, table))
  openTables.value = set
  emit('structure', db, table)
}

function openMenu(e, db, table) {
  menu.value = { x: Math.min(e.clientX, window.innerWidth - 210), y: e.clientY, db, table }
}

const closeMenu = () => { menu.value = null }
onMounted(() => document.addEventListener('click', closeMenu))
onBeforeUnmount(() => document.removeEventListener('click', closeMenu))

const act = (name) => {
  const { db, table } = menu.value
  emit(name, db, table)
  closeMenu()
}
</script>

<template>
  <div class="tree">
    <div class="head">
      <div class="search">
        <Search :size="14" />
        <input v-model="filter" placeholder="Filter schema" aria-label="Filter schema"
               spellcheck="false" @keydown.esc="filter = ''" />
        <button v-if="filter" class="clear" aria-label="Clear filter" @click="filter = ''">
          <X :size="13" />
        </button>
      </div>
      <Btn size="sm" icon variant="ghost" data-tip="Refresh schema" @click="emit('refresh')">
        <template #icon><RefreshCw :size="14" :class="{ turning: loading }" /></template>
      </Btn>
    </div>

    <div class="body">
      <Skeleton v-if="loading && !databases.length" :rows="9" indent />

      <State v-else-if="error" kind="error" title="Schema unavailable" :detail="error">
        <template #actions>
          <Btn size="sm" @click="emit('refresh')">Try again</Btn>
        </template>
      </State>

      <State v-else-if="!databases.length" title="No databases"
             hint="Nothing here that this account is allowed to see." />

      <State v-else-if="!shown.length" title="No matches"
             :hint="`Nothing matching “${filter}”.`" />

      <ul v-else class="list">
        <li v-for="node in shown" :key="node.db">
          <button :class="['row', 'db', { open: isOpen(node.db) }]"
                  @click="toggleDb(node.db)"
                  @contextmenu.prevent.stop="openMenu($event, node.db, null)">
            <ChevronRight :size="13" class="chev" />
            <Database :size="14" class="ico" />
            <span class="name ident" :title="node.db">{{ node.db }}</span>
          </button>

          <ul v-show="isOpen(node.db)" class="sub">
            <li v-for="[name, engine, rows] in node.tables || []" :key="name">
              <button
                :class="['row', 'tbl', {
                  on: selected?.db === node.db && selected?.table === name,
                  open: openTables.has(key(node.db, name)),
                }]"
                @click="toggleTable(node.db, name)"
                @dblclick="emit('browse', node.db, name)"
                @contextmenu.prevent.stop="openMenu($event, node.db, name)"
              >
                <ChevronRight :size="13" class="chev" />
                <Table2 :size="14" class="ico" />
                <span class="name ident" :title="`${name} — ${engine}`">{{ name }}</span>
                <span v-if="compact(rows)" class="count num">{{ compact(rows) }}</span>
              </button>

              <ul v-show="openTables.has(key(node.db, name))" class="cols">
                <li v-for="[col, type] in columns[key(node.db, name)] || []" :key="col"
                    class="col" :title="`${col} ${type}`">
                  <component :is="columnIcon(type)" :size="12" class="ico" />
                  <span class="name ident">{{ col }}</span>
                  <span class="type">{{ type }}</span>
                </li>
                <li v-if="!columns[key(node.db, name)]" class="col muted">Loading columns…</li>
              </ul>
            </li>
            <li v-if="node.tables && !node.tables.length" class="col muted">No tables</li>
          </ul>
        </li>
      </ul>
    </div>

    <Teleport to="body">
      <div v-if="menu" class="menu" :style="{ left: `${menu.x}px`, top: `${menu.y}px` }"
           @click.stop>
        <p class="menu-head ident">{{ menu.table || menu.db }}</p>
        <template v-if="menu.table">
          <button @click="act('browse')"><Eye :size="13" /> Browse data</button>
          <button @click="act('structure')"><Columns3 :size="13" /> Structure</button>
          <button @click="act('generate')"><SquareCode :size="13" /> Generate SELECT</button>
        </template>
        <button @click="act('copy-name')"><Copy :size="13" /> Copy name</button>
        <button @click="act('refresh')"><RefreshCw :size="13" /> Refresh schema</button>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tree { display: flex; flex-direction: column; height: 100%; min-height: 0; }

.head {
  display: flex;
  gap: 6px;
  padding: var(--s2);
  border-bottom: 1px solid var(--border-subtle);
}

.search {
  display: flex;
  align-items: center;
  gap: 7px;
  flex: 1;
  min-width: 0;
  height: var(--control-h);
  padding: 0 6px 0 9px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-tertiary);
  transition: border-color var(--t-fast) var(--ease);
}

.search:focus-within { border-color: var(--accent); }

.search input {
  flex: 1;
  min-width: 0;
  background: none;
  border: 0;
  outline: none;
  color: var(--text-primary);
  font-size: var(--fs-sm);
}

.search input::placeholder { color: var(--text-disabled); }

.clear { display: flex; padding: 3px; background: none; border: 0; border-radius: var(--r-xs); color: inherit; }
.clear:hover { background: var(--bg-hover); color: var(--text-primary); }

.body { flex: 1; overflow: auto; padding: var(--s2) var(--s1) var(--s5); }

ul { margin: 0; padding: 0; list-style: none; }
.sub { margin-left: 11px; border-left: 1px solid var(--border-subtle); }
.cols { margin-left: 11px; border-left: 1px solid var(--border-subtle); }

.row {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 27px;
  padding: 0 6px 0 3px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  text-align: left;
  transition: background var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.row:hover { background: var(--bg-hover); color: var(--text-primary); }
.row.on { background: var(--accent-subtle); color: var(--text-primary); box-shadow: inset 2px 0 0 var(--accent); }

.chev {
  flex: none;
  color: var(--text-disabled);
  transition: transform var(--t-fast) var(--ease);
}

.row.open .chev { transform: rotate(90deg); }
.ico { flex: none; color: var(--text-tertiary); }
.db .ico { color: var(--accent); opacity: 0.85; }
.tbl .ico { color: var(--info); opacity: 0.8; }
.row.on .ico { opacity: 1; }

.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.db .name { font-weight: 500; color: var(--text-primary); }

.count {
  flex: none;
  color: var(--text-disabled);
  font-size: var(--fs-xs);
  letter-spacing: 0.02em;
}

.col {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 6px 0 22px;
  color: var(--text-tertiary);
  overflow: hidden;
}

.col .name { font-size: var(--fs-xs); color: var(--text-secondary); }

.col .type {
  flex: none;
  max-width: 42%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-disabled);
  font: 400 10px/1 var(--font-mono);
}

.muted { padding-left: 24px; font-size: var(--fs-xs); }

.turning { animation: spin 900ms linear infinite; }

/* Context menu is teleported to <body> so tree overflow cannot clip it. */
.menu {
  position: fixed;
  z-index: 80;
  min-width: 186px;
  padding: 4px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-lg);
  animation: fade-in var(--t-fast) var(--ease);
}

.menu-head {
  margin: 0;
  padding: 6px 9px 7px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
}

.menu button {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 7px 9px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  text-align: left;
}

.menu button:hover { background: var(--bg-hover); color: var(--text-primary); }
</style>
