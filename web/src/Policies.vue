<script setup>
import { ref } from 'vue'
import { query } from './api'
import Results from './Results.vue'
import State from './ui/State.vue'
import Btn from './ui/Btn.vue'

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
  error.value = ''
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
    <State v-if="error" kind="error" title="Could not load policies" :detail="error">
      <template #actions><Btn size="sm" variant="primary" @click="load">Try again</Btn></template>
    </State>

    <State v-else-if="!Object.keys(results).length" kind="loading" title="Loading policies" />

    <template v-else>
      <section v-for="s in SECTIONS" :key="s.title">
        <h3>{{ s.title }} <span class="count num">{{ results[s.title].rows }}</span></h3>
        <p class="blurb">{{ s.blurb }}</p>
        <div class="card">
          <p v-if="results[s.title].rows === 0" class="none">None defined.</p>
          <Results v-else :result="results[s.title]" />
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.policies { height: 100%; overflow: auto; padding: var(--s6) var(--s8) var(--s10); }
.policies > .state { height: 100%; }

section { margin-bottom: var(--s8); }

h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: var(--fs-md);
  font-weight: 600;
}

.count {
  padding: 0 7px;
  background: var(--bg-active);
  border-radius: var(--r-full);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
  font-weight: 500;
}

.blurb { margin: 3px 0 var(--s3); color: var(--text-tertiary); font-size: var(--fs-sm); }

.card { border: 1px solid var(--border-subtle); border-radius: var(--r-md); overflow: hidden; }
.none { margin: 0; padding: var(--s4); color: var(--text-tertiary); font-size: var(--fs-sm); }

/* Results stretches to fill by default; here each table should only be as
   tall as its own rows. */
section :deep(.grid) { flex: none; max-height: 340px; }

@media (max-width: 900px) { .policies { padding: var(--s4); } }
</style>
