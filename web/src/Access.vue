<script setup>
import { computed, ref, watch } from 'vue'
import { query } from './api'
import { ident } from './sql'
import {
  roleClosure, grantRoleSQL, revokeRoleSQL,
  createUserSQL, dropUserSQL, createRoleSQL, dropRoleSQL,
  ANY_HOST, anyHost, hostList, hostSQL, validAddr,
} from './rbac'
import { stage, staged, unstage } from './ledger'
import Grants from './Grants.vue'
import Btn from './ui/Btn.vue'
import Field from './ui/Field.vue'
import State from './ui/State.vue'
import Skeleton from './ui/Skeleton.vue'
import { copy } from './clipboard'
import {
  Trash2, UserPlus, ShieldPlus, Plus, Search, X, Copy, UserRound, Shield, Lock,
} from 'lucide-vue-next'

const props = defineProps({ kind: String }) // 'user' | 'role'

const principals = ref([])
const grants = ref([])
const roleGrants = ref([])
const allRoles = ref([])
const tables = ref([])
const ddl = ref('')
const ddlError = ref('')
const error = ref('')
const loading = ref(false)
const filter = ref('')
const chosen = ref(null)

const newName = ref('')
const newPassword = ref('')
const addRole = ref('')
const newHost = ref('')
const hostError = ref('')

const kindIcon = computed(() => (props.kind === 'user' ? UserRound : Shield))

const shown = computed(() => {
  const q = filter.value.trim().toLowerCase()
  return q ? principals.value.filter(([n]) => n.toLowerCase().includes(q)) : principals.value
})

const current = computed(() => principals.value.find((p) => p[0] === chosen.value))

// Users defined in users.xml appear in system.users but cannot be changed by
// SQL, so every control that would emit DDL is disabled for them.
const readonly = computed(() => current.value?.[1] === 'users_xml')

const inherited = computed(() =>
  chosen.value ? roleClosure(chosen.value, roleGrants.value) : new Set(),
)

const directRoles = computed(() =>
  roleGrants.value
    .filter(([user, role]) => (user ?? role) === chosen.value)
    .map(([, , granted]) => granted),
)

const grantableRoles = computed(() =>
  allRoles.value.filter((r) => r !== chosen.value && !directRoles.value.includes(r)),
)

watch(() => props.kind, () => { chosen.value = null; load() }, { immediate: true })
watch(chosen, showCreate)

async function load() {
  error.value = ''
  loading.value = true
  try {
    const [people, g, rg, roles, tbl] = await Promise.all([
      query(props.kind === 'user'
        ? 'SELECT name, storage, arrayStringConcat(auth_type, \', \') AS auth,'
        + ' host_ip, host_names, host_names_regexp, host_names_like'
        + ' FROM system.users ORDER BY name'
        : "SELECT name, storage, '' AS auth FROM system.roles ORDER BY name"),
      query('SELECT user_name, role_name, access_type, database, table, is_partial_revoke' +
            ' FROM system.grants'),
      query('SELECT user_name, role_name, granted_role_name FROM system.role_grants'),
      query('SELECT name FROM system.roles ORDER BY name'),
      query("SELECT database, name FROM system.tables" +
            " WHERE database NOT IN ('system', 'INFORMATION_SCHEMA', 'information_schema')" +
            ' ORDER BY database, name'),
    ])
    principals.value = people.data
    grants.value = g.data
    roleGrants.value = rg.data
    allRoles.value = roles.data.map(([n]) => n)
    tables.value = tbl.data
    if (chosen.value && !people.data.some(([n]) => n === chosen.value)) chosen.value = null
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function showCreate() {
  ddl.value = ''
  ddlError.value = ''
  if (!chosen.value) return
  try {
    // SHOW CREATE redacts the password, so this is safe to put on screen.
    // SHOW CREATE will not take a bound identifier, so this one is quoted.
    const kw = props.kind === 'user' ? 'USER' : 'ROLE'
    const r = await query(`SHOW CREATE ${kw} ${ident(chosen.value)}`)
    ddl.value = r.data.map((row) => row[0]).join('\n')
  } catch (e) {
    ddlError.value = e.message
  }
}

function create() {
  const name = newName.value.trim()
  if (!name) return
  if (props.kind === 'user') {
    stage({
      sql: createUserSQL(name),
      params: { password: newPassword.value },
      note: `Create user ${name}`,
    })
  } else {
    stage({ sql: createRoleSQL(name), note: `Create role ${name}` })
  }
  newName.value = ''
  newPassword.value = ''
}

const drop = () => stage({
  sql: props.kind === 'user' ? dropUserSQL(chosen.value) : dropRoleSQL(chosen.value),
  note: `Drop ${props.kind} ${chosen.value}`,
})

function attachRole() {
  if (!addRole.value) return
  stage({
    sql: grantRoleSQL(addRole.value, chosen.value),
    note: `Grant role ${addRole.value} · ${chosen.value}`,
  })
  addRole.value = ''
}

const detachRole = (role) => stage({
  sql: revokeRoleSQL(role, chosen.value),
  note: `Revoke role ${role} · ${chosen.value}`,
})

/* --- allowed hosts ------------------------------------------------------- */
const serverHosts = computed(() => {
  const [, , , ip = [], names = [], regexp = [], like = []] = current.value ?? []
  return { ip, names, regexp, like }
})

// ALTER USER ... HOST replaces the whole list, so a user has at most one host
// statement staged, carrying the full list as it will be. The chips show that.
const hostEntry = computed(() => staged.value.find((e) => e.hostsOf === chosen.value))
const hosts = computed(() => hostEntry.value?.hosts ?? serverHosts.value)

function setHosts(next) {
  const at = staged.value.indexOf(hostEntry.value)
  if (at >= 0) unstage(at)
  const sql = hostSQL(chosen.value, next)
  if (sql === hostSQL(chosen.value, serverHosts.value)) return // back where it started
  stage({ sql, note: `Allowed hosts · ${chosen.value}`, hostsOf: chosen.value, hosts: next })
}

function addHost() {
  const value = newHost.value.trim()
  if (!validAddr(value)) {
    hostError.value = 'Enter an IP address or subnet, like 192.168.1.5 or 10.0.0.0/8.'
    return
  }
  hostError.value = ''
  newHost.value = ''
  const h = hosts.value
  // The first address replaces "any host": that is the point of adding one.
  if (anyHost(h)) setHosts({ ip: [value], names: [], regexp: [], like: [] })
  else if (!h.ip.includes(value)) setHosts({ ...h, ip: [...h.ip, value] })
}

const removeHost = ({ kind, value }) =>
  setHosts({ ...hosts.value, [kind]: hosts.value[kind].filter((v) => v !== value) })

watch(chosen, () => { newHost.value = ''; hostError.value = '' })

defineExpose({ reload: () => { load(); showCreate() } })
</script>

<template>
  <div class="access">
    <aside class="list">
      <div class="lhead">
        <span class="eyebrow">{{ kind }}s</span>
        <span class="count num">{{ principals.length }}</span>
      </div>

      <div class="searchbox">
        <Search :size="14" />
        <input v-model="filter" :placeholder="`Filter ${kind}s`" :aria-label="`Filter ${kind}s`"
               spellcheck="false" @keydown.esc="filter = ''" />
        <button v-if="filter" aria-label="Clear filter" @click="filter = ''"><X :size="13" /></button>
      </div>

      <div class="rows">
        <Skeleton v-if="loading && !principals.length" :rows="7" />
        <p v-else-if="!shown.length" class="none">
          {{ filter ? `Nothing matching “${filter}”.` : `No ${kind}s visible to this account.` }}
        </p>
        <button v-for="[name, storage] in shown" :key="name" type="button"
                :class="['row', { on: name === chosen }]" @click="chosen = name">
          <component :is="kindIcon" :size="14" class="ico" />
          <span class="ident nm" :title="name">{{ name }}</span>
          <span v-if="storage === 'users_xml'" class="tag"
                data-tip-left data-tip="Defined in users.xml, not editable by SQL">xml</span>
        </button>
      </div>

      <form class="new" @submit.prevent="create">
        <span class="eyebrow">New {{ kind }}</span>
        <Field v-model="newName" :placeholder="`${kind} name`" :aria-label="`New ${kind} name`" mono />
        <Field v-if="kind === 'user'" v-model="newPassword" type="password"
               placeholder="password" aria-label="Password" autocomplete="new-password" reveal />
        <Btn type="submit" variant="subtle" :disabled="!newName.trim()">
          <template #icon><component :is="kind === 'user' ? UserPlus : ShieldPlus" :size="14" /></template>
          Stage new {{ kind }}
        </Btn>
      </form>
    </aside>

    <section class="detail">
      <State v-if="error" kind="error" :title="`Could not load ${kind}s`" :detail="error">
        <template #actions><Btn size="sm" variant="primary" @click="load">Try again</Btn></template>
      </State>

      <State v-else-if="loading && !principals.length" kind="loading" :title="`Loading ${kind}s`" />

      <State v-else-if="!chosen" :title="`No ${kind} selected`"
             :hint="`Pick a ${kind} on the left to see what it can do.`">
        <template #icon><component :is="kindIcon" :size="20" /></template>
      </State>

      <div v-else class="sheet">
        <header>
          <span class="tile"><component :is="kindIcon" :size="20" /></span>
          <div class="titles">
            <h2 class="ident">{{ chosen }}</h2>
            <p class="sub">
              {{ kind === 'user' ? 'User' : 'Role' }}
              <template v-if="current?.[2]"> · <span class="ident">{{ current[2] }}</span></template>
            </p>
          </div>
          <span class="spacer" />
          <Btn variant="danger" size="sm" :disabled="readonly" @click="drop">
            <template #icon><Trash2 :size="13" /></template>Drop {{ kind }}
          </Btn>
        </header>

        <p v-if="readonly" class="warn">
          <Lock :size="14" />
          <span>
            Defined in users.xml. ClickHouse can show this account but cannot alter it
            by SQL, so it has to be edited in the server config.
          </span>
        </p>

        <section v-if="kind === 'user'" class="block">
          <h3 class="eyebrow">Roles</h3>
          <div class="chips">
            <span v-for="r in directRoles" :key="r" class="chip ident">
              {{ r }}
              <button v-if="!readonly" :aria-label="`Revoke role ${r}`" @click="detachRole(r)">
                <X :size="12" />
              </button>
            </span>
            <span v-if="!directRoles.length" class="muted">No roles granted.</span>
          </div>
          <div v-if="!readonly" class="add">
            <select v-model="addRole" class="select" aria-label="Role to grant">
              <option value="">Grant a role&hellip;</option>
              <option v-for="r in grantableRoles" :key="r" :value="r">{{ r }}</option>
            </select>
            <Btn :disabled="!addRole" @click="attachRole">
              <template #icon><Plus :size="13" /></template>Grant
            </Btn>
          </div>
        </section>

        <section v-if="kind === 'user'" class="block">
          <h3 class="eyebrow">Allowed hosts <span v-if="hostEntry" class="pending">staged</span></h3>
          <div class="chips">
            <span v-if="anyHost(hosts)" class="chip ident">Any host</span>
            <template v-else>
              <span v-for="h in hostList(hosts)" :key="h.kind + h.value" class="chip ident">
                <small v-if="h.kind !== 'ip'">{{ h.kind }}</small>{{ h.value }}
                <button v-if="!readonly" :aria-label="`Remove ${h.value}`" @click="removeHost(h)">
                  <X :size="12" />
                </button>
              </span>
              <span v-if="!hostList(hosts).length" class="lockout">
                No host allowed. This user cannot connect from anywhere.
              </span>
            </template>
          </div>
          <template v-if="!readonly">
            <form class="add" @submit.prevent="addHost">
              <Field v-model="newHost" placeholder="192.168.1.5 or 10.0.0.0/8" mono
                     :invalid="!!hostError" aria-label="IP address or subnet to allow" />
              <Btn type="submit" :disabled="!newHost.trim()">
                <template #icon><Plus :size="13" /></template>Allow
              </Btn>
              <Btn v-if="!anyHost(hosts)" variant="ghost" type="button"
                   @click="setHosts(ANY_HOST)">Allow any host</Btn>
            </form>
            <p v-if="hostError" class="hint bad" role="alert">{{ hostError }}</p>
            <p v-else class="hint">
              {{ anyHost(hosts)
                ? 'Allowing an address limits this user to the addresses listed.'
                : 'Only these addresses can sign in as this user.' }}
              ClickHouse sees the address of the machine that connects to it,
              which for this console is the machine running it.
            </p>
          </template>
        </section>

        <section class="block">
          <h3 class="eyebrow">Grants</h3>
          <Grants :grantee="chosen" :grants="grants" :roles="inherited"
                  :tables="tables" :readonly="readonly" />
        </section>

        <section class="block">
          <h3 class="eyebrow">
            Definition
            <Btn v-if="ddl" size="sm" icon variant="ghost" data-tip="Copy definition"
                 aria-label="Copy definition" @click="copy(ddl, 'Definition copied')">
              <template #icon><Copy :size="13" /></template>
            </Btn>
          </h3>
          <p v-if="ddlError" class="ddl bad" role="alert">
            <span>Could not load the definition: {{ ddlError }}</span>
            <Btn size="sm" @click="showCreate">Try again</Btn>
          </p>
          <pre v-else class="ddl">{{ ddl || 'Loading…' }}</pre>
        </section>
      </div>
    </section>
  </div>
</template>

<style scoped>
.access { display: grid; grid-template-columns: 248px minmax(0, 1fr); height: 100%; min-height: 0; }

.list {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-right: 1px solid var(--border);
  background: var(--bg-panel);
}

.lhead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--s3) var(--s3) var(--s2);
}

.count {
  padding: 0 6px;
  background: var(--bg-active);
  border-radius: var(--r-full);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
}

.searchbox { flex: none; margin: 0 var(--s2) var(--s2); }

.rows { flex: 1; min-height: 0; overflow: auto; padding: 0 var(--s1) var(--s2); }

.row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: var(--row-h);
  padding: 0 8px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  text-align: left;
  transition: background var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.row:hover { background: var(--bg-hover); color: var(--text-primary); }

.row.on {
  background: var(--accent-subtle);
  color: var(--text-primary);
  box-shadow: inset 2px 0 0 var(--accent);
}

.row .ico { flex: none; color: var(--text-tertiary); }
.row.on .ico { color: var(--accent); }
.nm { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.tag {
  flex: none;
  padding: 0 5px;
  border: 1px solid var(--border-strong);
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
  font-size: 10px;
  line-height: 1.5;
}

.none { margin: var(--s3) var(--s2); color: var(--text-tertiary); font-size: var(--fs-sm); }

.new {
  display: flex;
  flex: none;
  flex-direction: column;
  gap: var(--s2);
  padding: var(--s3);
  border-top: 1px solid var(--border);
}

.detail { min-width: 0; overflow: auto; }
.detail > .state { height: 100%; }
.sheet { max-width: 1120px; padding: var(--s6) var(--s8) var(--s10); animation: fade-in var(--t) var(--ease); }

header { display: flex; align-items: center; gap: var(--s3); }

.tile {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  background: var(--accent-subtle);
  border: 1px solid var(--accent-border);
  border-radius: var(--r-md);
  color: var(--accent);
}

.titles { min-width: 0; }
h2 { margin: 0; overflow: hidden; font-size: var(--fs-lg); font-weight: 600; text-overflow: ellipsis; }
.sub { margin: 1px 0 0; color: var(--text-tertiary); font-size: var(--fs-sm); }
.spacer { flex: 1; }

.warn {
  display: flex;
  gap: 9px;
  max-width: 72ch;
  margin: var(--s4) 0 0;
  padding: 9px 11px;
  background: var(--warning-subtle);
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  line-height: 1.55;
}

.warn svg { flex: none; margin-top: 2px; color: var(--warning); }

.block { margin-top: var(--s6); }
.block h3 { display: flex; align-items: center; gap: 6px; min-height: 26px; margin-bottom: 6px; }

.chips { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 26px;
  padding: 0 9px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-primary);
}

.chip button {
  display: flex;
  margin-right: -5px;
  padding: 3px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
}

.chip button:hover { background: var(--error-subtle); color: var(--error); }
.chip small { margin-right: 2px; color: var(--text-tertiary); font-size: 10px; }

.add { display: flex; align-items: center; gap: 8px; margin-top: 10px; }
.add .field { width: 260px; }
.add :deep(.box) { height: var(--control-h); }
.add .select { min-width: 180px; }

.muted { color: var(--text-tertiary); font-size: var(--fs-sm); }
.pending { color: var(--accent-text); letter-spacing: 0; text-transform: none; }
.lockout { color: var(--warning); font-size: var(--fs-sm); }
.hint { max-width: 62ch; margin: 8px 0 0; color: var(--text-tertiary); font-size: var(--fs-sm); line-height: 1.55; }
.hint.bad { color: var(--error); }

.ddl {
  margin: 0;
  padding: 11px 13px;
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-md);
  color: var(--text-secondary);
  font: 400 var(--fs-sm)/1.6 var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
}

.ddl.bad {
  display: flex;
  align-items: center;
  gap: var(--s3);
  border-color: var(--error-border);
  background: var(--error-subtle);
  color: var(--error);
}
.ddl.bad span { flex: 1; min-width: 0; }

@media (max-width: 900px) {
  .access { grid-template-columns: 200px minmax(0, 1fr); }
  .sheet { padding: var(--s4); }
  header { flex-wrap: wrap; }
}
</style>
