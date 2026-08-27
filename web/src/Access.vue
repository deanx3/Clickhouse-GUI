<script setup>
import { computed, ref, watch } from 'vue'
import { query } from './api'
import { ident } from './sql'
import {
  roleClosure, grantRoleSQL, revokeRoleSQL,
  createUserSQL, dropUserSQL, createRoleSQL, dropRoleSQL,
} from './rbac'
import { stage } from './ledger'
import Grants from './Grants.vue'
import Btn from './ui/Btn.vue'
import Field from './ui/Field.vue'
import { Trash2, UserPlus, ShieldPlus, Plus } from 'lucide-vue-next'

const props = defineProps({ kind: String }) // 'user' | 'role'

const principals = ref([])
const grants = ref([])
const roleGrants = ref([])
const allRoles = ref([])
const tables = ref([])
const ddl = ref('')
const error = ref('')
const chosen = ref(null)

const newName = ref('')
const newPassword = ref('')
const addRole = ref('')

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
  try {
    const [people, g, rg, roles, tbl] = await Promise.all([
      query(props.kind === 'user'
        ? 'SELECT name, storage, arrayStringConcat(auth_type, \', \') AS auth'
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
  }
}

async function showCreate() {
  ddl.value = ''
  if (!chosen.value) return
  try {
    // SHOW CREATE redacts the password, so this is safe to put on screen.
    // SHOW CREATE will not take a bound identifier, so this one is quoted.
    const kw = props.kind === 'user' ? 'USER' : 'ROLE'
    const r = await query(`SHOW CREATE ${kw} ${ident(chosen.value)}`)
    ddl.value = r.data.map((row) => row[0]).join('\n')
  } catch (e) {
    ddl.value = e.message
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

defineExpose({ reload: () => { load(); showCreate() } })
</script>

<template>
  <div class="access">
    <aside class="list">
      <div v-for="[name, storage] in principals" :key="name"
           :class="['row', { on: name === chosen }]" @click="chosen = name">
        <span class="ident">{{ name }}</span>
        <span v-if="storage === 'users_xml'" class="tag" title="Defined in users.xml, not editable by SQL">xml</span>
      </div>

      <form class="new" @submit.prevent="create">
        <Field v-model="newName" :placeholder="`new ${kind} name`" mono />
        <Field v-if="kind === 'user'" v-model="newPassword" type="password"
               placeholder="password" reveal />
        <Btn type="submit" variant="subtle" :disabled="!newName.trim()">
          <template #icon>
            <component :is="kind === 'user' ? UserPlus : ShieldPlus" :size="14" />
          </template>
          Create {{ kind }}
        </Btn>
      </form>
    </aside>

    <section class="detail">
      <p v-if="error" class="error">{{ error }}</p>

      <p v-else-if="!chosen" class="empty">Pick a {{ kind }} to see what it can do.</p>

      <template v-else>
        <header>
          <h2 class="ident">{{ chosen }}</h2>
          <span v-if="current?.[2]" class="auth ident">{{ current[2] }}</span>
          <span class="spacer" />
          <Btn variant="danger" size="sm" :disabled="readonly" @click="drop">
            <template #icon><Trash2 :size="13" /></template>Drop {{ kind }}
          </Btn>
        </header>

        <p v-if="readonly" class="warn">
          Defined in users.xml. ClickHouse can show this account but cannot alter it
          by SQL, so it has to be edited in the server config.
        </p>

        <section v-if="kind === 'user'" class="block">
          <h3>Roles</h3>
          <div class="chips">
            <span v-for="r in directRoles" :key="r" class="chip ident">
              {{ r }}
              <button :disabled="readonly" @click="detachRole(r)">&times;</button>
            </span>
            <span v-if="!directRoles.length" class="empty">No roles granted.</span>
          </div>
          <div v-if="!readonly" class="add">
            <select v-model="addRole">
              <option value="">Grant a role&hellip;</option>
              <option v-for="r in grantableRoles" :key="r" :value="r">{{ r }}</option>
            </select>
            <Btn size="sm" :disabled="!addRole" @click="attachRole">
              <template #icon><Plus :size="13" /></template>Grant
            </Btn>
          </div>
        </section>

        <section class="block">
          <h3>Grants</h3>
          <Grants :grantee="chosen" :grants="grants" :roles="inherited"
                  :tables="tables" :readonly="readonly" />
        </section>

        <section class="block">
          <h3>Definition</h3>
          <pre class="ddl">{{ ddl }}</pre>
        </section>
      </template>
    </section>
  </div>
</template>

<style scoped>
.access { display: grid; grid-template-columns: 220px 1fr; height: 100%; min-height: 0; }

.list {
  display: flex;
  flex-direction: column;
  overflow: auto;
  padding: var(--s2);
  border-right: 1px solid var(--border);
  background: var(--bg-panel);
}

.row {
  display: flex;
  align-items: center;
  gap: 6px;
  height: var(--row-h);
  padding: 0 8px;
  border-radius: var(--md-corner-xs);
  cursor: pointer;
}

.row {
  color: var(--text-secondary);
  transition: background var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.row:hover { background: var(--bg-hover); color: var(--text-primary); }

.row.on {
  background: var(--accent-subtle);
  color: var(--text-primary);
  box-shadow: inset 2px 0 0 var(--accent);
}

.tag {
  margin-left: auto;
  padding: 1px 5px;
  border: 1px solid var(--md-outline);
  border-radius: var(--md-corner-xs);
  color: var(--md-on-surface-variant);
  font-size: 10px;
}

.new {
  display: flex;
  flex-direction: column;
  gap: var(--s2);
  margin-top: auto;
  padding-top: var(--s4);
  border-top: 1px solid var(--border-subtle);
}

.detail { overflow: auto; padding: var(--s6) var(--s8); }

header { display: flex; align-items: center; gap: 12px; margin-bottom: 6px; }
h2 { margin: 0; font-size: var(--fs-lg); font-weight: 600; }
.auth { color: var(--md-on-surface-variant); font-size: 11px; }
.spacer { flex: 1; }

.warn {
  margin: 10px 0;
  padding: 9px 11px;
  background: var(--md-surface-container);
  border-left: 2px solid var(--md-outline);
  color: var(--md-on-surface-variant);
  font-size: 12px;
  line-height: 1.55;
}

.block { margin-top: 26px; }

h3 {
  margin: 0 0 10px;
  color: var(--md-on-surface-variant);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.chips { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }

.chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 4px 0 9px;
  background: var(--md-tertiary-container);
  border-radius: var(--md-corner-xs);
  font-size: 12px;
}

.chip button {
  background: none;
  border: 0;
  color: inherit;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}

.add { display: flex; gap: 8px; margin-top: 10px; }

.add select {
  height: 28px;
  padding: 0 8px;
  background: var(--md-surface-container-high);
  border: 1px solid var(--md-outline);
  border-radius: var(--md-corner-xs);
  font-size: 12px;
}

.ddl {
  margin: 0;
  padding: 11px 13px;
  background: var(--md-surface-container-low);
  border: 1px solid var(--md-outline-variant);
  border-radius: var(--md-corner-xs);
  font: 400 12px/1.6 var(--font-mono);
  white-space: pre-wrap;
}

.empty { color: var(--md-on-surface-variant); font-size: 13px; }

.error {
  padding: 10px 12px;
  background: var(--md-error-container);
  border-radius: var(--md-corner-xs);
  font: 400 12px/1.5 var(--font-mono);
}

button:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
