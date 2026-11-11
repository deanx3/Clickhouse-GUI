<script setup>
import { computed, ref, watch } from 'vue'
import { query } from './api'
import Results from './Results.vue'
import State from './ui/State.vue'
import Btn from './ui/Btn.vue'
import { RefreshCw } from 'lucide-vue-next'

// ClickHouse tags GRANT and REVOKE with their own query_kind, but lumps user
// DDL in with table DDL under Create/Drop/Alter, so those need the statement
// itself. This is a substring match anywhere in the query, not a prefix, so a
// statement that merely mentions one of these phrases is kept too. That is the
// right way round for an audit view: showing one query too many costs a glance,
// hiding a real access change costs more.
const ACCESS_DDL = [
  'CREATE USER', 'ALTER USER', 'DROP USER',
  'CREATE ROLE', 'ALTER ROLE', 'DROP ROLE',
  'CREATE ROW POLICY', 'ALTER ROW POLICY', 'DROP ROW POLICY',
  'CREATE QUOTA', 'ALTER QUOTA', 'DROP QUOTA',
  'CREATE SETTINGS PROFILE', 'ALTER SETTINGS PROFILE', 'DROP SETTINGS PROFILE',
]

const ACCESS_FILTER =
  `query_kind IN ('Grant', 'Revoke')` +
  ` OR (query_kind IN ('Create', 'Drop', 'Alter')` +
  ` AND multiSearchAnyCaseInsensitive(query, [` +
  ACCESS_DDL.map((s) => `'${s}'`).join(', ') + ']))'

const mode = ref('access')
const who = ref('')
const users = ref([])
const result = ref(null)
const error = ref('')
const busy = ref(false)

const sql = computed(() => [
  'SELECT event_time, user, query_kind AS kind, query, query_duration_ms AS ms,',
  '       read_rows, formatReadableSize(read_bytes) AS bytes_read, exception',
  'FROM system.query_log',
  "WHERE type != 'QueryStart' AND is_initial_query",
  mode.value === 'access' ? `  AND (${ACCESS_FILTER})` : '',
  who.value ? '  AND user = {who:String}' : '',
  'ORDER BY event_time DESC',
  'LIMIT 200',
].filter(Boolean).join('\n'))

load()
watch([mode, who], load)

async function load() {
  busy.value = true
  error.value = ''
  try {
    const [rows, people] = await Promise.all([
      query(sql.value, who.value ? { who: who.value } : undefined),
      users.value.length ? null : query('SELECT name FROM system.users ORDER BY name'),
    ])
    result.value = rows
    if (people) users.value = people.data.map(([n]) => n)
  } catch (e) {
    error.value = e.message
    result.value = null
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="log">
    <div class="bar">
      <div class="seg" role="tablist" aria-label="Which queries">
        <button role="tab" :aria-selected="mode === 'access'" :class="{ on: mode === 'access' }"
                @click="mode = 'access'">Access changes</button>
        <button role="tab" :aria-selected="mode === 'all'" :class="{ on: mode === 'all' }"
                @click="mode = 'all'">All queries</button>
      </div>

      <label class="who">
        User
        <select v-model="who" class="select sm">
          <option value="">Anyone</option>
          <option v-for="u in users" :key="u" :value="u">{{ u }}</option>
        </select>
      </label>

      <span class="hint">Latest 200, read from ClickHouse's own system.query_log.</span>

      <Btn size="sm" :disabled="busy" @click="load">
        <template #icon><RefreshCw :size="13" :class="{ turning: busy }" /></template>Refresh
      </Btn>
    </div>

    <div class="out">
      <State v-if="error" kind="error" title="Could not read the query log" :detail="error">
        <template #actions><Btn size="sm" variant="primary" @click="load">Try again</Btn></template>
      </State>
      <State v-else-if="!result" kind="loading" title="Loading" />
      <State v-else-if="!result.rows"
             :title="mode === 'access' ? 'No access changes logged' : 'No queries logged'"
             :hint="who ? `Nothing from ${who}. Try Anyone.` : 'Nothing matching in system.query_log yet.'" />
      <Results v-else :result="result" :class="{ stale: busy }" />
    </div>
  </div>
</template>

<style scoped>
.log { display: flex; flex-direction: column; height: 100%; min-height: 0; }

.bar {
  display: flex;
  flex: none;
  align-items: center;
  gap: var(--s3);
  height: 40px;
  padding: 0 var(--s3);
  background: var(--bg-panel);
  border-bottom: 1px solid var(--border);
}

.seg { display: flex; gap: 2px; }

.seg button {
  height: 28px;
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

.seg button:hover { background: var(--bg-hover); color: var(--text-secondary); }
.seg button.on { background: var(--bg-active); color: var(--text-primary); }

.who { display: flex; align-items: center; gap: 7px; color: var(--text-tertiary); font-size: var(--fs-xs); }

.hint {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.turning { animation: spin 900ms linear infinite; }

.out { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.out > * { flex: 1; min-height: 0; }
.stale { opacity: 0.55; transition: opacity var(--t) var(--ease); }

@media (max-width: 760px) { .hint { display: none; } }
</style>
