<script setup>
import { Loader2 } from 'lucide-vue-next'

defineProps({
  variant: { type: String, default: 'subtle' }, // primary | subtle | ghost | danger
  size: { type: String, default: 'md' },        // sm | md
  loading: Boolean,
  disabled: Boolean,
  icon: Boolean,                                 // square, icon-only
})
</script>

<template>
  <button :class="['btn', variant, size, { icon, loading }]" :disabled="disabled || loading">
    <Loader2 v-if="loading" :size="14" class="spin" />
    <slot v-else name="icon" />
    <span v-if="!icon" class="label"><slot /></span>
    <slot name="after" />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: var(--control-h);
  padding: 0 12px;
  border: 1px solid transparent;
  border-radius: var(--r-sm);
  font-size: var(--fs-sm);
  font-weight: 500;
  white-space: nowrap;
  transition: background var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease),
              color var(--t-fast) var(--ease), opacity var(--t-fast) var(--ease);
}

.sm { height: 26px; padding: 0 9px; font-size: var(--fs-xs); }
.icon { width: var(--control-h); padding: 0; }
.icon.sm { width: 26px; }

.primary { background: var(--accent); color: #2b1400; font-weight: 600; }
.primary:hover:not(:disabled) { background: var(--accent-hover); }
.primary:active:not(:disabled) { background: var(--accent-press); }

.subtle {
  background: var(--bg-elevated);
  border-color: var(--border);
  color: var(--text-primary);
}
.subtle:hover:not(:disabled) { background: var(--bg-hover); border-color: var(--border-strong); }

.ghost { background: none; color: var(--text-secondary); }
.ghost:hover:not(:disabled) { background: var(--bg-hover); color: var(--text-primary); }

.danger { background: none; border-color: var(--error-border); color: var(--error); }
.danger:hover:not(:disabled) { background: var(--error-subtle); }

.btn:disabled { opacity: 0.5; }

.primary:disabled {
  opacity: 1;
  background: var(--bg-active);
  color: var(--text-disabled);
}
.spin { animation: spin 700ms linear infinite; }
.label { line-height: 1; }
</style>
