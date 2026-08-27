<script setup>
import { computed, ref } from 'vue'
import {
  PRIVILEGES, resolve, scopeId, scopeLabel, sourceLabel, grantSQL, revokeSQL,
} from './rbac'
import { stage, isStaged } from './ledger'

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
            <button
              :class="[
                'mark',
                { direct: cell(p, scope)?.direct,
                  inherited: cell(p, scope) && !cell(p, scope).direct,
                  pending: isStaged(cell(p, scope)?.direct
                    ? revokeSQL(p, scope[0], scope[1], grantee)
                    : grantSQL(p, scope[0], scope[1], grantee)) }
              ]"
              :disabled="readonly"
              :title="sourceLabel(cell(p, scope)) || (cell(p, scope) ? 'granted directly' : 'not granted')"
              @click="toggle(p, scope)"
            />
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="!readonly" class="add">
      <select v-model="pickDb">
        <option value="">Add scope&hellip;</option>
        <option v-for="db in databases" :key="db" :value="db">{{ db }}</option>
      </select>
      <select v-model="pickTable" :disabled="!pickDb">
        <option value="*">* (whole database)</option>
        <option v-for="t in tablesIn" :key="t" :value="t">{{ t }}</option>
      </select>
      <button :disabled="!pickDb" @click="addScope">Add</button>
    </div>

    <p class="key">
      <span class="mark direct" /> direct
      <span class="mark inherited" /> inherited
      <span class="mark pending" /> staged
      <span class="mark" /> none
    </p>
  </div>
</template>

<style scoped>
.grants { overflow: auto; }

table { border-collapse: collapse; }

thead th {
  height: 30px;
  padding: 0 6px;
  border-bottom: 1px solid var(--md-outline);
  color: var(--md-on-surface-variant);
  font: 500 10px/1 var(--font-sans);
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

th.scope {
  padding-right: 20px;
  text-align: left;
  text-transform: none;
  letter-spacing: 0;
  white-space: nowrap;
}

tbody th.scope {
  height: var(--row-h);
  border-bottom: 1px solid var(--md-outline-variant);
  color: var(--md-on-surface);
  font: 400 12.5px/1 var(--font-mono);
}

.flag {
  margin-left: 8px;
  padding: 1px 5px;
  border: 1px solid var(--md-error);
  border-radius: var(--md-corner-xs);
  color: var(--md-error);
  font: 400 10px/1.4 var(--font-sans);
}

td {
  border-bottom: 1px solid var(--md-outline-variant);
  text-align: center;
}

/* Fill means granted right here. A ring means it arrives from somewhere else,
   which is the question this screen exists to answer. */
.mark {
  width: 15px;
  height: 15px;
  padding: 0;
  margin: 0 9px;
  background: none;
  border: 1px solid var(--md-outline);
  border-radius: 3px;
  vertical-align: middle;
  cursor: pointer;
}

.mark:hover:not(:disabled) { border-color: var(--md-on-surface-variant); }
.mark:disabled { cursor: not-allowed; }

.direct { background: var(--md-on-surface); border-color: var(--md-on-surface); }

.inherited {
  background: none;
  border-color: var(--md-tertiary);
  box-shadow: inset 0 0 0 3px var(--md-tertiary);
}

.pending {
  border-color: var(--md-primary);
  background: var(--md-primary);
  box-shadow: none;
}

.add {
  display: flex;
  gap: 8px;
  padding: 14px 0 4px;
}

select, .add button {
  height: 26px;
  padding: 0 8px;
  background: var(--md-surface-container-high);
  border: 1px solid var(--md-outline);
  border-radius: var(--md-corner-xs);
  font-size: 12px;
  cursor: pointer;
}

.key {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 14px 0 0;
  color: var(--md-on-surface-variant);
  font-size: 11px;
}

.key .mark { margin: 0 2px 0 10px; cursor: default; }
.key .mark:first-child { margin-left: 0; }
</style>
