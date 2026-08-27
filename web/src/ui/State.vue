<script setup>
import { AlertTriangle, Inbox, Loader2 } from 'lucide-vue-next'
import Btn from './Btn.vue'
import { copy } from '../clipboard'

const props = defineProps({
  kind: { type: String, default: 'empty' }, // loading | empty | error
  title: String,
  detail: String,
  hint: String,
})
</script>

<template>
  <div :class="['state', kind]">
    <Loader2 v-if="kind === 'loading'" :size="20" class="spin" />
    <AlertTriangle v-else-if="kind === 'error'" :size="20" />
    <span v-else-if="$slots.icon" class="custom"><slot name="icon" /></span>
    <Inbox v-else :size="20" />

    <p class="title">{{ title }}</p>
    <p v-if="detail" class="detail">{{ detail }}</p>
    <p v-if="hint" class="hint">{{ hint }}</p>

    <div class="acts">
      <Btn v-if="kind === 'error' && detail" size="sm" @click="copy(detail, 'Error copied')">
        Copy error
      </Btn>
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 48px 24px;
  text-align: center;
  color: var(--text-tertiary);
  animation: fade-in var(--t) var(--ease);
}

.error { color: var(--error); }
.custom { display: flex; }
.spin { animation: spin 700ms linear infinite; }

.title {
  margin: 8px 0 0;
  color: var(--text-primary);
  font-size: var(--fs-md);
  font-weight: 500;
}

.error .title { color: var(--error); }

.detail {
  max-width: 620px;
  margin: 2px 0 0;
  color: var(--text-secondary);
  font: 400 var(--fs-sm)/1.6 var(--font-mono);
  text-align: left;
  white-space: pre-wrap;
  word-break: break-word;
}

.hint { margin: 6px 0 0; color: var(--text-tertiary); font-size: var(--fs-sm); }
.acts { display: flex; gap: 8px; margin-top: 14px; }
.acts:empty { display: none; }
</style>
