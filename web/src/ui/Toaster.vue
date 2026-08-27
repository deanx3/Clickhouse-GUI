<script setup>
import { CheckCircle2, XCircle, Info, X } from 'lucide-vue-next'
import { toasts, dismiss } from '../toast'

const ICON = { success: CheckCircle2, error: XCircle, info: Info }
</script>

<template>
  <div class="toaster" role="status" aria-live="polite">
    <TransitionGroup name="toast">
      <div v-for="t in toasts" :key="t.id" :class="['toast', t.kind]">
        <component :is="ICON[t.kind] || Info" :size="15" />
        <span>{{ t.message }}</span>
        <button aria-label="Dismiss" @click="dismiss(t.id)"><X :size="13" /></button>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toaster {
  position: fixed;
  right: var(--s5);
  bottom: var(--s5);
  z-index: 90;
  display: flex;
  flex-direction: column;
  gap: 8px;
  pointer-events: none;
}

.toast {
  display: flex;
  align-items: center;
  gap: 9px;
  max-width: 380px;
  padding: 10px 10px 10px 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-lg);
  font-size: var(--fs-sm);
  pointer-events: auto;
}

.success { color: var(--success); }
.error { color: var(--error); }
.info { color: var(--info); }
.toast span { color: var(--text-primary); }

.toast button {
  display: flex;
  margin-left: auto;
  padding: 3px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
}

.toast button:hover { background: var(--bg-hover); color: var(--text-primary); }

.toast-enter-active, .toast-leave-active { transition: all var(--t) var(--ease); }
.toast-enter-from { opacity: 0; transform: translateX(16px); }
.toast-leave-to { opacity: 0; transform: scale(0.96); }
.toast-move { transition: transform var(--t) var(--ease); }
</style>
