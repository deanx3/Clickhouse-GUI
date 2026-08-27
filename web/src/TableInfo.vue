<script setup>
import { computed } from 'vue'
import { Table2, Hash, Columns3, Play, Copy, HardDrive } from 'lucide-vue-next'
import { columnIcon, compact } from './columns'
import { copy } from './clipboard'
import Btn from './ui/Btn.vue'
import State from './ui/State.vue'

const props = defineProps({
  db: String,
  table: String,
  meta: Array,      // [name, engine, rows] from system.tables
  columns: Array,   // [[name, type], ...]
})

const emit = defineEmits(['browse', 'generate'])

const engine = computed(() => props.meta?.[1] ?? '')
const rows = computed(() => props.meta?.[2])
const qualified = computed(() => `${props.db}.${props.table}`)
</script>

<template>
  <State v-if="!table" title="No table selected"
         hint="Pick a table in the sidebar to see its structure." />

  <div v-else class="info">
    <header>
      <span class="ico"><Table2 :size="20" /></span>
      <div class="titles">
        <h2 class="ident">{{ table }}</h2>
        <p class="ident path">{{ db }}</p>
      </div>
      <div class="acts">
        <Btn size="sm" variant="primary" @click="emit('browse', db, table)">
          <template #icon><Play :size="13" /></template>Browse data
        </Btn>
        <Btn size="sm" @click="emit('generate', db, table)">Generate SELECT</Btn>
        <Btn size="sm" icon data-tip="Copy qualified name" @click="copy(qualified, 'Name copied')">
          <template #icon><Copy :size="13" /></template>
        </Btn>
      </div>
    </header>

    <dl class="facts">
      <div>
        <dt><Hash :size="13" /> Rows</dt>
        <dd class="num">{{ compact(rows) || 'unknown' }}</dd>
      </div>
      <div>
        <dt><Columns3 :size="13" /> Columns</dt>
        <dd class="num">{{ columns?.length ?? '—' }}</dd>
      </div>
      <div>
        <dt><HardDrive :size="13" /> Engine</dt>
        <dd class="ident">{{ engine || '—' }}</dd>
      </div>
    </dl>

    <section>
      <h3>Columns</h3>
      <ul v-if="columns?.length" class="cols">
        <li v-for="([name, type], i) in columns" :key="name">
          <span class="pos num">{{ i + 1 }}</span>
          <component :is="columnIcon(type)" :size="14" class="cico" />
          <span class="cname ident">{{ name }}</span>
          <span class="ctype ident">{{ type }}</span>
        </li>
      </ul>
      <p v-else class="loading">Loading columns…</p>
    </section>
  </div>
</template>

<style scoped>
.info { height: 100%; overflow: auto; padding: var(--s6) var(--s8); }

header { display: flex; align-items: center; gap: var(--s3); }

.ico {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  background: rgb(88 166 255 / 0.1);
  border: 1px solid rgb(88 166 255 / 0.25);
  border-radius: var(--r-md);
  color: var(--info);
}

.titles { min-width: 0; }
h2 { margin: 0; font-size: var(--fs-lg); font-weight: 600; }
.path { margin: 1px 0 0; color: var(--text-tertiary); font-size: var(--fs-sm); }
.acts { display: flex; gap: 6px; margin-left: auto; }

.facts {
  display: flex;
  gap: var(--s2);
  margin: var(--s5) 0 var(--s6);
}

.facts > div {
  flex: 1;
  min-width: 0;
  padding: var(--s3) var(--s4);
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-md);
}

dt {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

dd { margin: 6px 0 0; color: var(--text-primary); font-size: var(--fs-lg); font-weight: 500; }
dd.ident { font-size: var(--fs-md); }

h3 {
  margin: 0 0 var(--s3);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.cols {
  margin: 0;
  padding: 0;
  list-style: none;
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-md);
  overflow: hidden;
}

.cols li {
  display: flex;
  align-items: center;
  gap: var(--s3);
  height: 38px;
  padding: 0 var(--s4);
  border-bottom: 1px solid var(--border-subtle);
  transition: background var(--t-fast) var(--ease);
}

.cols li:last-child { border-bottom: 0; }
.cols li:hover { background: var(--bg-panel); }

.pos { width: 20px; color: var(--text-disabled); font-size: var(--fs-xs); text-align: right; }
.cico { color: var(--text-tertiary); flex: none; }
.cname { flex: 1; min-width: 0; font-size: var(--fs-base); overflow: hidden; text-overflow: ellipsis; }
.ctype { color: var(--text-secondary); font-size: var(--fs-sm); }
.loading { color: var(--text-tertiary); font-size: var(--fs-sm); }

@media (max-width: 900px) {
  .info { padding: var(--s4); }
  header { flex-wrap: wrap; }
  .acts { margin-left: 0; width: 100%; }
  .facts { flex-direction: column; }
}
</style>
