<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  Database, Users, Shield, FileLock2, ScrollText, Search, Play, FilePlus2,
  Wand2, Eraser, RefreshCw, History as HistoryIcon, PanelLeft, LogOut, Table2,
} from 'lucide-vue-next'
import { logout, me } from './api'
import { staged } from './ledger'
import { pref, setPref } from './prefs'
import Explorer from './Explorer.vue'
import Access from './Access.vue'
import Ledger from './Ledger.vue'
import Policies from './Policies.vue'
import Log from './Log.vue'
import Palette from './Palette.vue'
import Logo from './ui/Logo.vue'
import StatusDot from './ui/StatusDot.vue'
import UserMenu from './ui/UserMenu.vue'
import ConnectionMenu from './ui/ConnectionMenu.vue'

const props = defineProps({ session: Object })
const emit = defineEmits(['signed-out'])

const SECTIONS = [
  { id: 'Explorer', icon: Database },
  { id: 'Users', icon: Users },
  { id: 'Roles', icon: Shield },
  { id: 'Policies', icon: FileLock2 },
  { id: 'Query log', icon: ScrollText },
]

const active = ref(pref('section', 'Explorer'))
const access = ref(null)
const explorer = ref(null)
const paletteOpen = ref(false)

watch(active, (v) => setPref('section', v))

const WRITES = ['Users', 'Roles']
const showLedger = computed(() => WRITES.includes(active.value) || staged.value.length > 0)
const mac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const modKey = mac ? '⌘' : 'Ctrl'

/* --- command palette ----------------------------------------------------- */
const commands = computed(() => {
  const out = SECTIONS.map((s) => ({
    id: `go:${s.id}`, group: 'Go to', label: s.id, icon: s.icon,
    act: () => { active.value = s.id },
  }))

  if (active.value === 'Explorer' && explorer.value) {
    const e = explorer.value
    out.push(
      { id: 'run', group: 'Query', label: 'Run query', icon: Play, keys: `${modKey}⏎`, act: () => e.run() },
      { id: 'new', group: 'Query', label: 'New query', icon: FilePlus2, act: () => e.newQuery() },
      { id: 'fmt', group: 'Query', label: 'Format SQL', icon: Wand2, keys: `${modKey}S`, act: () => e.formatQuery() },
      { id: 'clr', group: 'Query', label: 'Clear editor', icon: Eraser, act: () => e.clearQuery() },
      { id: 'hist', group: 'Query', label: 'Query history', icon: HistoryIcon, act: () => e.toggleHistory() },
      { id: 'refresh', group: 'Schema', label: 'Refresh schema', icon: RefreshCw, act: () => e.refreshSchema() },
      { id: 'side', group: 'View', label: 'Toggle sidebar', icon: PanelLeft, act: () => e.toggleSidebar() },
    )
    for (const { db, table } of e.tableList()) {
      out.push({
        id: `t:${db}.${table}`, group: 'Tables', label: table, hint: db, icon: Table2,
        act: () => e.browse(db, table),
      })
    }
  }

  out.push({ id: 'out', group: 'Session', label: 'Sign out', icon: LogOut, act: signOut })
  return out
})

function onKey(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    paletteOpen.value = !paletteOpen.value
  }
}

onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))

// Changing server invalidates every cached schema and result, so remount the
// screen rather than trying to reconcile state that belongs to another cluster.
const connKey = ref(0)

// The section has to be part of the key. Users and Roles are the same
// component with different props, so a shared key makes Vue patch one into the
// other in place - and the prop change is skipped, leaving Roles showing users.
const viewKey = computed(() => `${active.value}:${connKey.value}`)

async function reconnect() {
  connKey.value++
  const fresh = await me()
  if (fresh) Object.assign(props.session, fresh)
}

async function signOut() {
  await logout()
  emit('signed-out')
}
</script>

<template>
  <div class="shell" :class="{ ledger: showLedger }">
    <header class="bar">
      <div class="brand">
        <span class="mark"><Logo :size="19" /></span>
        <span class="name">ClickHouse Console</span>
      </div>

      <nav class="nav">
        <button v-for="s in SECTIONS" :key="s.id" :class="['navbtn', { on: active === s.id }]"
                @click="active = s.id">
          <component :is="s.icon" :size="14" />
          <span>{{ s.id }}</span>
        </button>
      </nav>

      <button class="finder" @click="paletteOpen = true">
        <Search :size="14" />
        <span>Search</span>
        <kbd>{{ modKey }} K</kbd>
      </button>

      <StatusDot />
      <ConnectionMenu @changed="reconnect" />
      <UserMenu :session="session" @sign-out="signOut" />
    </header>

    <main class="body">
      <Explorer v-if="active === 'Explorer'" :key="viewKey" ref="explorer" />
      <Access v-else-if="active === 'Users'" :key="viewKey" ref="access" kind="user" />
      <Access v-else-if="active === 'Roles'" :key="viewKey" ref="access" kind="role" />
      <Policies v-else-if="active === 'Policies'" :key="viewKey" />
      <Log v-else :key="viewKey" />
    </main>

    <Ledger v-if="showLedger" @applied="access?.reload()" />

    <Palette :open="paletteOpen" :commands="commands"
             @close="paletteOpen = false" @run="$event.act()" />
  </div>
</template>

<style scoped>
.shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  grid-template-rows: var(--header-h) minmax(0, 1fr);
  height: 100%;
}

.shell.ledger { grid-template-columns: minmax(0, 1fr) 328px; }

.bar {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: var(--s3);
  padding: 0 var(--s3) 0 var(--s4);
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
}

.brand { display: flex; align-items: center; gap: 9px; }

.mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  background: var(--accent-subtle);
  border: 1px solid var(--accent-border);
  border-radius: var(--r-sm);
  color: var(--accent);
}

.name { font-size: var(--fs-base); font-weight: 600; letter-spacing: -0.01em; white-space: nowrap; }

.nav {
  display: flex;
  gap: 1px;
  margin-left: var(--s4);
  padding-left: var(--s4);
  border-left: 1px solid var(--border);
}

.navbtn {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 30px;
  padding: 0 11px;
  background: none;
  border: 0;
  border-radius: var(--r-sm);
  color: var(--text-tertiary);
  font-size: var(--fs-sm);
  font-weight: 500;
  white-space: nowrap;
  transition: background var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.navbtn:hover { background: var(--bg-hover); color: var(--text-secondary); }
.navbtn.on { background: var(--bg-active); color: var(--text-primary); }
.navbtn.on svg { color: var(--accent); }

.finder {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 190px;
  height: var(--control-h);
  margin-left: auto;
  padding: 0 6px 0 10px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-tertiary);
  font-size: var(--fs-sm);
  transition: border-color var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.finder:hover { border-color: var(--border-strong); color: var(--text-secondary); }
.finder span { flex: 1; text-align: left; }

.body { min-width: 0; min-height: 0; overflow: hidden; }

@media (max-width: 1240px) {
  .navbtn span { display: none; }
  .navbtn { padding: 0 9px; }
}

@media (max-width: 1000px) {
  .name { display: none; }
  .finder { min-width: 0; }
  .finder span, .finder kbd { display: none; }
  .shell.ledger { grid-template-columns: minmax(0, 1fr) 280px; }
}

@media (max-width: 760px) {
  .shell.ledger { grid-template-columns: minmax(0, 1fr); }
  .nav { margin-left: var(--s2); padding-left: var(--s2); }
}
</style>
