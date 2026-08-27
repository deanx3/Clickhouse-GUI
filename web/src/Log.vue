<script setup>
import { computed, ref, watch } from 'vue'
import { query } from './api'
import Results from './Results.vue'

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
      <div class="tabs">
        <button :class="{ on: mode === 'access' }" @click="mode = 'access'">Access changes</button>
        <button :class="{ on: mode === 'all' }" @click="mode = 'all'">All queries</button>
      </div>

      <select v-model="who">
        <option value="">Anyone</option>
        <option v-for="u in users" :key="u" :value="u">{{ u }}</option>
      </select>

      <span class="hint">
        ClickHouse already keeps this. Nothing here is a log chgui writes.
      </span>

      <button class="ghost" :disabled="busy" @click="load">
        {{ busy ? 'Loading' : 'Refresh' }}
      </button>
    </div>

    <p v-if="error" class="error">{{ error }}</p>
    <Results v-else-if="result" :result="result" />
  </div>
</template>

<style scoped>
.log { display: flex; flex-direction: column; height: 100%; min-height: 0; }

.bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--md-outline-variant);
}

.tabs { display: flex; gap: 1px; }

.tabs button {
  height: 26px;
  padding: 0 12px;
  background: var(--md-surface-container-low);
  border: 1px solid var(--md-outline-variant);
  color: var(--md-on-surface-variant);
  font-size: 12px;
  cursor: pointer;
}

.tabs button:first-child { border-radius: var(--md-corner-xs) 0 0 var(--md-corner-xs); }
.tabs button:last-child { border-radius: 0 var(--md-corner-xs) var(--md-corner-xs) 0; }

.tabs button.on {
  background: var(--md-surface-container-high);
  border-color: var(--md-outline);
  color: var(--md-on-surface);
  font-weight: 500;
}

select {
  height: 26px;
  padding: 0 8px;
  background: var(--md-surface-container-high);
  border: 1px solid var(--md-outline);
  border-radius: var(--md-corner-xs);
  font-size: 12px;
}

.hint { margin-left: auto; color: var(--md-on-surface-variant); font-size: 11.5px; }

.ghost {
  height: 26px;
  padding: 0 12px;
  background: none;
  border: 1px solid var(--md-outline);
  border-radius: var(--md-corner-xs);
  color: var(--md-on-surface-variant);
  font-size: 12px;
  cursor: pointer;
}

.error {
  margin: 14px;
  padding: 10px 12px;
  background: var(--md-error-container);
  border-radius: var(--md-corner-xs);
  font: 400 12px/1.5 var(--font-mono);
}
</style>
