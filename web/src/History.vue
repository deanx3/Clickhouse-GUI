<script setup>
import { computed, ref } from 'vue'
import { Search, Play, Copy, Trash2, X, CheckCircle2, XCircle } from 'lucide-vue-next'
import { history, forget, clearHistory, bucketOf, clockOf } from './history'
import { copy } from './clipboard'
import Btn from './ui/Btn.vue'
import State from './ui/State.vue'

const emit = defineEmits(['use', 'close'])
const filter = ref('')

const groups = computed(() => {
  const q = filter.value.trim().toLowerCase()
  const rows = q ? history.value.filter((e) => e.sql.toLowerCase().includes(q)) : history.value
  const out = []
  for (const entry of rows) {
    const bucket = bucketOf(entry.at)
    if (out.at(-1)?.bucket !== bucket) out.push({ bucket, entries: [] })
    out.at(-1).entries.push(entry)
  }
  return out
})
</script>

<template>
  <aside class="history">
    <header>
      <span class="ttl">Query history</span>
      <Btn v-if="history.length" size="sm" variant="ghost" data-tip="Clear all"
           @click="clearHistory()">
        <template #icon><Trash2 :size="13" /></template>
      </Btn>
      <Btn size="sm" icon variant="ghost" data-tip="Close" @click="emit('close')">
        <template #icon><X :size="14" /></template>
      </Btn>
    </header>

    <div class="search">
      <Search :size="13" />
      <input v-model="filter" placeholder="Search history" aria-label="Search history"
             spellcheck="false" @keydown.esc="filter = ''" />
    </div>

    <div class="body">
      <State v-if="!history.length" title="Nothing yet"
             hint="Statements you run are kept here, in this browser only." />
      <State v-else-if="!groups.length" title="No matches" :hint="`Nothing matching “${filter}”.`" />

      <template v-else>
        <section v-for="g in groups" :key="g.bucket">
          <h4>{{ g.bucket }}</h4>
          <article v-for="e in g.entries" :key="e.id" :class="{ failed: !e.ok }"
                   @dblclick="emit('use', e.sql)">
            <pre class="sql">{{ e.sql }}</pre>
            <footer>
              <component :is="e.ok ? CheckCircle2 : XCircle" :size="11"
                         :class="e.ok ? 'ok' : 'bad'" />
              <span v-if="e.ok" class="num">{{ e.ms }} ms</span>
              <span v-else class="bad">failed</span>
              <span v-if="e.ok && e.rows != null" class="num">· {{ e.rows }} rows</span>
              <span class="spacer" />
              <span class="clock num">{{ clockOf(e.at) }}</span>
              <div class="acts">
                <button data-tip="Run" @click="emit('use', e.sql, true)"><Play :size="12" /></button>
                <button data-tip="Copy" @click="copy(e.sql, 'Query copied')"><Copy :size="12" /></button>
                <button data-tip="Remove" @click="forget(e.id)"><Trash2 :size="12" /></button>
              </div>
            </footer>
          </article>
        </section>
      </template>
    </div>
  </aside>
</template>

<style scoped>
.history {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--bg-panel);
  border-left: 1px solid var(--border);
}

header {
  display: flex;
  align-items: center;
  gap: 2px;
  height: var(--header-h);
  padding: 0 var(--s2) 0 var(--s4);
  border-bottom: 1px solid var(--border-subtle);
}

.ttl { flex: 1; font-size: var(--fs-sm); font-weight: 600; }

.search {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: var(--s2);
  padding: 0 9px;
  height: 28px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-tertiary);
}

.search:focus-within { border-color: var(--accent); }
.search input {
  flex: 1; min-width: 0; background: none; border: 0; outline: none;
  color: var(--text-primary); font-size: var(--fs-sm);
}

.body { flex: 1; overflow: auto; padding: 0 var(--s2) var(--s5); }

h4 {
  position: sticky;
  top: 0;
  z-index: 1;
  margin: 0;
  padding: var(--s3) 6px 6px;
  background: var(--bg-panel);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

article {
  margin-bottom: 6px;
  padding: 8px 9px 6px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-sm);
  cursor: default;
  transition: border-color var(--t-fast) var(--ease);
}

article:hover { border-color: var(--border-strong); }
article.failed { border-left: 2px solid var(--error); }

.sql {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin: 0;
  color: var(--text-secondary);
  font: 400 var(--fs-xs)/1.55 var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
}

footer {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 7px;
  color: var(--text-tertiary);
  font-size: 10px;
}

.ok { color: var(--success); }
.bad { color: var(--error); }
.spacer { flex: 1; }
.clock { color: var(--text-disabled); }

.acts { display: flex; gap: 1px; margin-left: 4px; opacity: 0; transition: opacity var(--t-fast) var(--ease); }
article:hover .acts, article:focus-within .acts { opacity: 1; }

.acts button {
  display: flex;
  padding: 4px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
}

.acts button:hover { background: var(--bg-active); color: var(--text-primary); }
</style>
