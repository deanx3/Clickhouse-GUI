<script setup>
import { computed } from 'vue'
import { status, lastError } from '../status'

const LABEL = { connected: 'Connected', connecting: 'Connecting', error: 'Disconnected' }
const label = computed(() => LABEL[status.value])
</script>

<template>
  <span class="status" :class="status" :data-tip="lastError || `${label} to ClickHouse`">
    <span class="dot" />
    <span class="txt">{{ label }}</span>
  </span>
</template>

<style scoped>
.status {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  padding: 0 4px;
  color: var(--text-secondary);
  font-size: var(--fs-sm);
}

.dot {
  width: 7px;
  height: 7px;
  border-radius: var(--r-full);
  background: var(--text-tertiary);
  transition: background var(--t) var(--ease), box-shadow var(--t) var(--ease);
}

.connected .dot { background: var(--success); box-shadow: 0 0 0 3px var(--success-subtle); }
.connecting .dot { background: var(--warning); animation: pulse 1.1s ease-in-out infinite; }
.error .dot { background: var(--error); box-shadow: 0 0 0 3px var(--error-subtle); }
.error .txt { color: var(--error); }

@keyframes pulse { 50% { opacity: 0.35 } }

@media (max-width: 1100px) { .txt { display: none; } }
</style>
