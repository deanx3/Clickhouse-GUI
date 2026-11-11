<script setup>
import { ref } from 'vue'
import { staged, history, unstage, discard, apply } from './ledger'
import { X, Check, ListChecks } from 'lucide-vue-next'
import Btn from './ui/Btn.vue'
import State from './ui/State.vue'

const emit = defineEmits(['applied'])
const busy = ref(false)

async function run() {
  busy.value = true
  try {
    await apply()
  } catch {
    // The failing statement carries its own error; nothing to add here.
  } finally {
    busy.value = false
    emit('applied')
  }
}
</script>

<template>
  <aside class="ledger">
    <header>
      <ListChecks :size="14" class="mark" />
      <span class="ttl">SQL ledger</span>
      <span v-if="staged.length" class="count num">{{ staged.length }} staged</span>
    </header>

    <div class="scroll">
      <State v-if="!staged.length && !history.length" title="Nothing staged"
             hint="Changes queue up here as SQL. Nothing runs until you apply it.">
        <template #icon><ListChecks :size="20" /></template>
      </State>

      <div v-for="(entry, i) in staged" :key="entry.sql" :class="['entry', 'staged', { failed: entry.error }]">
        <div class="note">
          <span>{{ entry.note }}</span>
          <button :aria-label="`Remove: ${entry.note}`" data-tip-left data-tip="Remove" @click="unstage(i)">
            <X :size="12" />
          </button>
        </div>
        <pre>{{ entry.sql }}</pre>
        <p v-if="entry.error" class="err" role="alert">{{ entry.error }}</p>
      </div>

      <template v-if="history.length">
        <h4 class="eyebrow">Applied this session</h4>
        <div v-for="(entry, i) in history" :key="`h${i}`" class="entry done">
          <div class="note"><Check :size="12" /><span>{{ entry.note }}</span></div>
          <pre>{{ entry.sql }}</pre>
        </div>
      </template>
    </div>

    <footer v-if="staged.length">
      <Btn :disabled="busy" @click="discard">Discard</Btn>
      <Btn variant="primary" :loading="busy" @click="run">
        {{ busy ? 'Applying' : `Apply ${staged.length}` }}
      </Btn>
    </footer>
  </aside>
</template>

<style scoped>
.ledger {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-panel);
  border-left: 1px solid var(--border);
}

header {
  display: flex;
  flex: none;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 var(--s3);
  border-bottom: 1px solid var(--border-subtle);
}

.mark { color: var(--accent); }
.ttl { flex: 1; font-size: var(--fs-sm); font-weight: 600; }

.count {
  padding: 1px 8px;
  background: var(--accent-subtle);
  border-radius: var(--r-full);
  color: var(--accent-text);
  font-size: var(--fs-xs);
}

.scroll { flex: 1; overflow: auto; padding: var(--s2); }
.scroll > .state { height: 100%; }

h4 { margin: var(--s4) 4px var(--s2); }

.entry {
  margin-bottom: 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-sm);
  overflow: hidden;
  animation: rise var(--t) var(--ease);
}

/* Orange means staged: about to happen, not yet true. */
.staged { border-color: var(--accent-border); }
.failed { border-color: var(--error-border); }
.done { opacity: 0.6; animation: none; }

.note {
  display: flex;
  align-items: center;
  gap: 6px;
  min-height: 28px;
  padding: 0 4px 0 9px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
  font-weight: 500;
}

.note span { flex: 1; min-width: 0; }
.staged .note { color: var(--accent-text); }
.done .note svg { color: var(--success); }

.note button {
  display: flex;
  padding: 4px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
}

.note button:hover { background: var(--bg-active); color: var(--text-primary); }

pre {
  margin: 0;
  padding: 7px 9px;
  color: var(--text-primary);
  font: 400 var(--fs-xs)/1.55 var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
}

.err {
  margin: 0;
  padding: 6px 9px;
  background: var(--error-subtle);
  color: var(--error);
  font: 400 var(--fs-xs)/1.45 var(--font-mono);
  word-break: break-word;
}

footer {
  display: flex;
  flex: none;
  gap: 8px;
  padding: var(--s2);
  border-top: 1px solid var(--border);
}

footer > * { flex: 1; }
</style>
