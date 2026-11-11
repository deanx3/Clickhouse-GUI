<script setup>
import { computed, ref } from 'vue'
import {
  PRIVILEGES, resolve, scopeId, scopeLabel, sourceLabel, grantSQL, revokeSQL,
} from './rbac'
import { stage, isStaged } from './ledger'
import { Plus } from 'lucide-vue-next'
import Btn from './ui/Btn.vue'

const props = defineProps({
  grantee: String,
  grants: Array,     // system.grants rows
  roles: Object,     // Set of inherited role names
  tables: Array,     // [database, name] pairs, for the scope picker
  readonly: Boolean,
})

const extra = ref([])
const pickDb = ref('')
const pickTable = ref('*')

const databases = computed(() => [...new Set(props.tables.map(([db]) => db))].sort())
const tablesIn = computed(() => props.tables.filter(([db]) => db === pickDb.value).map(([, t]) => t))

// Rows are every scope this grantee already touches - directly or through a
// role - plus whatever the user adds. *.* is always present because it is the
// scope people most often need to check and most often forget to look at.
const scopes = computed(() => {
  const found = new Map([[scopeId(null, null), [null, null]]])
  for (const [user, role, , db, table] of props.grants) {
    const holder = user ?? role
    if (holder !== props.grantee && !props.roles.has(holder)) continue
    found.set(scopeId(db, table), [db, table])
  }
  for (const scope of extra.value) found.set(scopeId(...scope), scope)
  return [...found.values()].sort(
    (a, b) => scopeLabel(...a).localeCompare(scopeLabel(...b)),
  )
})

// Grants recorded as partial revokes are negative, and this matrix does not
// model them. Flag the scope rather than quietly showing a wrong answer.
// ponytail: display-only; edit partial revokes in the query editor.
const partial = computed(() => {
  const out = new Set()
  for (const [user, role, , db, table, isPartial] of props.grants) {
    const holder = user ?? role
    if (!isPartial) continue
    if (holder !== props.grantee && !props.roles.has(holder)) continue
    out.add(scopeId(db, table))
  }
  return out
})

const cell = (priv, [db, table]) =>
  resolve(props.grantee, priv, db, table, props.grants, props.roles)

// Everything the template needs to draw one checkbox, worked out once.
function view(priv, scope) {
  const state = cell(priv, scope)
  const direct = !!state?.direct
  const sql = (direct ? revokeSQL : grantSQL)(priv, scope[0], scope[1], props.grantee)
  return {
    direct,
    cls: { direct, inherited: !!state && !direct, pending: isStaged(sql) },
    tip: sourceLabel(state) || (state ? 'granted directly' : 'not granted'),
  }
}

function toggle(priv, [db, table]) {
  if (props.readonly) return
  const state = cell(priv, [db, table])
  const sql = state?.direct
    ? revokeSQL(priv, db, table, props.grantee)
    : grantSQL(priv, db, table, props.grantee)
  stage({ sql, note: `${state?.direct ? 'Revoke' : 'Grant'} ${priv} · ${props.grantee}` })
}

function addScope() {
  if (!pickDb.value) return
  extra.value.push([pickDb.value, pickTable.value === '*' ? null : pickTable.value])
}
</script>

<template>
  <div class="grants">
    <div class="scroller">
      <table>
        <thead>
          <tr>
            <th class="scope">Scope</th>
            <th v-for="p in PRIVILEGES" :key="p">{{ p }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="scope in scopes" :key="scopeId(...scope)">
            <th class="scope ident">
              {{ scopeLabel(...scope) }}
              <span v-if="partial.has(scopeId(...scope))" class="flag"
                    title="This scope has partial revokes, which this matrix does not show">
                partial revoke
              </span>
            </th>
            <td v-for="p in PRIVILEGES" :key="p">
              <!-- One-item v-for: a local name for view(), so it runs once per cell. -->
              <button v-for="v in [view(p, scope)]" :key="p" class="cell"
                      :disabled="readonly" :aria-pressed="v.direct" :title="v.tip"
                      :aria-label="`${p} on ${scopeLabel(...scope)}: ${v.tip}`"
                      @click="toggle(p, scope)">
                <span :class="['mark', v.cls]" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="foot">
      <div v-if="!readonly" class="add">
        <select v-model="pickDb" class="select sm" aria-label="Database">
          <option value="">Add scope&hellip;</option>
          <option v-for="db in databases" :key="db" :value="db">{{ db }}</option>
        </select>
        <select v-model="pickTable" class="select sm" :disabled="!pickDb" aria-label="Table">
          <option value="*">* (whole database)</option>
          <option v-for="t in tablesIn" :key="t" :value="t">{{ t }}</option>
        </select>
        <Btn size="sm" :disabled="!pickDb" @click="addScope">
          <template #icon><Plus :size="13" /></template>Add
        </Btn>
      </div>

      <p class="key">
        <span class="mark direct" /> direct
        <span class="mark inherited" /> inherited
        <span class="mark pending" /> staged
        <span class="mark" /> none
      </p>
    </div>
  </div>
</template>

<style scoped>
.grants { border: 1px solid var(--border-subtle); border-radius: var(--r-md); overflow: hidden; }
.scroller { overflow: auto; }

table { border-collapse: collapse; min-width: 100%; }

thead th {
  height: 34px;
  padding: 0 6px;
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
  color: var(--text-tertiary);
  font: 500 10px/1 var(--font-sans);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  white-space: nowrap;
}

/* The scope stays put while a wide privilege list scrolls under it. */
th.scope {
  position: sticky;
  left: 0;
  z-index: 1;
  padding: 0 var(--s5) 0 var(--s4);
  text-align: left;
  white-space: nowrap;
}

tbody th.scope {
  height: var(--row-h);
  background: var(--bg-base);
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
  font: 400 var(--fs-sm)/1 var(--font-mono);
}

td { padding: 0; border-bottom: 1px solid var(--border-subtle); text-align: center; }
tbody tr:last-child > * { border-bottom: 0; }
tbody tr:hover > * { background: var(--bg-panel); }

.flag {
  margin-left: 8px;
  padding: 1px 5px;
  border: 1px solid var(--error-border);
  border-radius: var(--r-xs);
  color: var(--error);
  font: 400 10px/1.4 var(--font-sans);
}

/* The whole cell is the target; the 15px box inside is only the picture. */
.cell {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-width: 44px;
  height: var(--row-h);
  padding: 0;
  background: none;
  border: 0;
}

.cell:hover:not(:disabled) { background: var(--bg-hover); }
.cell:hover:not(:disabled) .mark { border-color: var(--text-secondary); }
.cell:disabled { cursor: not-allowed; opacity: 0.6; }

/* Fill means granted right here. A ring means it arrives from somewhere else,
   which is the question this screen exists to answer. */
.mark {
  display: inline-block;
  width: 15px;
  height: 15px;
  border: 1px solid var(--border-strong);
  border-radius: 3px;
  transition: border-color var(--t-fast) var(--ease), background var(--t-fast) var(--ease);
}

.mark.direct { background: var(--text-primary); border-color: var(--text-primary); }
.mark.inherited { border-color: var(--info); box-shadow: inset 0 0 0 3px var(--info); }
.mark.pending { background: var(--accent); border-color: var(--accent); box-shadow: none; }

.foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--s3);
  padding: var(--s2) var(--s3);
  background: var(--bg-panel);
  border-top: 1px solid var(--border-subtle);
}

.add { display: flex; gap: 6px; }

.key {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 0 auto;
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
}

.key .mark { margin-left: 10px; }
.key .mark:first-child { margin-left: 0; }
</style>
