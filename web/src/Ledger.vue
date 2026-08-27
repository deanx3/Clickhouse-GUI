<script setup>
import { ref } from 'vue'
import { staged, history, unstage, discard, apply } from './ledger'

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
      <span>SQL ledger</span>
      <span v-if="staged.length" class="count">{{ staged.length }} staged</span>
    </header>

    <div class="scroll">
      <p v-if="!staged.length && !history.length" class="empty">
        Changes queue up here as SQL. Nothing runs until you apply it.
      </p>

      <div v-for="(entry, i) in staged" :key="entry.sql" class="entry staged">
        <div class="note">
          {{ entry.note }}
          <button class="x" title="Remove" @click="unstage(i)">&times;</button>
        </div>
        <pre>{{ entry.sql }}</pre>
        <p v-if="entry.error" class="err">{{ entry.error }}</p>
      </div>

      <div v-for="(entry, i) in history" :key="`h${i}`" class="entry done">
        <div class="note">{{ entry.note }}</div>
        <pre>{{ entry.sql }}</pre>
      </div>
    </div>

    <footer v-if="staged.length">
      <button class="ghost" :disabled="busy" @click="discard">Discard</button>
      <button class="apply" :disabled="busy" @click="run">
        {{ busy ? 'Applying' : `Apply ${staged.length}` }}
      </button>
    </footer>
  </aside>
</template>

<style scoped>
.ledger {
  display: flex;
  flex-direction: column;
  min-height: 0;
  border-left: 1px solid var(--md-outline-variant);
  background: var(--md-surface-container-low);
}

header {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 34px;
  padding: 0 14px;
  border-bottom: 1px solid var(--md-outline-variant);
  font-size: 12px;
  font-weight: 500;
}

.count { margin-left: auto; color: var(--md-primary); font-size: 11px; }

.scroll { flex: 1; overflow: auto; padding: 10px; }

.empty {
  margin: 6px 4px;
  color: var(--md-on-surface-variant);
  font-size: 12px;
  line-height: 1.6;
}

.entry {
  margin-bottom: 8px;
  border: 1px solid var(--md-outline-variant);
  border-radius: var(--md-corner-xs);
  overflow: hidden;
}

/* Yellow means staged: about to happen, not yet true. */
.staged { border-color: var(--md-primary-container); }
.done { opacity: 0.5; }

.note {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  background: var(--md-surface-container);
  color: var(--md-on-surface-variant);
  font-size: 11px;
}

.staged .note { color: var(--md-primary); }

.x {
  margin-left: auto;
  padding: 0 2px;
  background: none;
  border: 0;
  color: inherit;
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}

pre {
  margin: 0;
  padding: 7px 8px;
  font: 400 11.5px/1.5 var(--font-mono);
  white-space: pre-wrap;
  word-break: break-word;
}

.err {
  margin: 0;
  padding: 6px 8px;
  background: var(--md-error-container);
  font: 400 11px/1.4 var(--font-mono);
}

footer {
  display: flex;
  gap: 8px;
  padding: 10px;
  border-top: 1px solid var(--md-outline-variant);
}

footer button {
  flex: 1;
  height: 30px;
  border-radius: var(--md-corner-xs);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.ghost {
  background: none;
  border: 1px solid var(--md-outline);
  color: var(--md-on-surface-variant);
}

.apply { background: var(--md-primary); border: 0; color: var(--md-on-primary); }
button:disabled { opacity: 0.45; cursor: not-allowed; }
</style>
