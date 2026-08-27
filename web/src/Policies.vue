<script setup>
import { ref } from 'vue'
import { query } from './api'
import Results from './Results.vue'

// Read-only for now: these three are worth seeing next to users and roles, but
// editing them is a separate shape of form and nobody has asked yet.
// ponytail: display only; add DDL here when someone needs to author a policy.
const SECTIONS = [
  {
    title: 'Row policies',
    blurb: 'A permissive policy limits what a user sees; a restrictive one narrows it further.',
    sql: `SELECT name, database, table, select_filter AS filter,
       if(is_restrictive, 'restrictive', 'permissive') AS mode,
       if(apply_to_all, 'everyone', arrayStringConcat(apply_to_list, ', ')) AS applies_to,
       storage
FROM system.row_policies
ORDER BY database, table, name`,
  },
  {
    title: 'Quotas',
    blurb: 'Caps on how much a grantee may consume over each interval, in seconds.',
    sql: `SELECT name,
       arrayStringConcat(arrayMap(k -> toString(k), keys), ', ') AS counted_per,
       arrayStringConcat(arrayMap(d -> toString(d), durations), ', ') AS intervals_s,
       if(apply_to_all, 'everyone', arrayStringConcat(apply_to_list, ', ')) AS applies_to,
       storage
FROM system.quotas
ORDER BY name`,
  },
  {
    title: 'Settings profiles',
    blurb: 'Named bundles of server settings attached to users and roles.',
    sql: `SELECT name, num_elements AS settings,
       if(apply_to_all, 'everyone', arrayStringConcat(apply_to_list, ', ')) AS applies_to,
       storage
FROM system.settings_profiles
ORDER BY name`,
  },
]

const results = ref({})
const error = ref('')

load()
async function load() {
  try {
    const rows = await Promise.all(SECTIONS.map((s) => query(s.sql)))
    results.value = Object.fromEntries(SECTIONS.map((s, i) => [s.title, rows[i]]))
  } catch (e) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="policies">
    <p v-if="error" class="error">{{ error }}</p>

    <section v-for="s in SECTIONS" :key="s.title">
      <h3>{{ s.title }}</h3>
      <p class="blurb">{{ s.blurb }}</p>
      <div v-if="results[s.title]?.rows === 0" class="empty">None defined.</div>
      <Results v-else-if="results[s.title]" :result="results[s.title]" />
    </section>
  </div>
</template>

<style scoped>
.policies { overflow: auto; padding: 20px 24px; height: 100%; }

section { margin-bottom: 34px; }

h3 {
  margin: 0 0 4px;
  color: var(--md-on-surface-variant);
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.blurb { margin: 0 0 12px; color: var(--md-on-surface-variant); font-size: 12.5px; }

.empty { color: var(--md-on-surface-variant); font-size: 13px; }

/* Results stretches to fill by default; here each table should only be as
   tall as its own rows. */
section :deep(.grid) { flex: none; max-height: 340px; }

.error {
  padding: 10px 12px;
  background: var(--md-error-container);
  border-radius: var(--md-corner-xs);
  font: 400 12px/1.5 var(--font-mono);
}
</style>
