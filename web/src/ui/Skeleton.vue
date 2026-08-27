<script setup>
defineProps({ rows: { type: Number, default: 6 }, indent: Boolean })
</script>

<template>
  <div class="skel" aria-hidden="true">
    <div v-for="i in rows" :key="i" class="bar"
         :style="{ width: `${86 - ((i * 13) % 45)}%`, marginLeft: indent && i % 3 ? '14px' : 0 }" />
  </div>
</template>

<style scoped>
.skel { display: flex; flex-direction: column; gap: 10px; padding: 12px 10px; }

.bar {
  height: 9px;
  border-radius: var(--r-xs);
  background: linear-gradient(90deg, var(--bg-elevated) 25%, var(--bg-hover) 37%, var(--bg-elevated) 63%);
  background-size: 400% 100%;
  animation: shimmer 1.3s ease infinite;
}

@keyframes shimmer { from { background-position: 100% 0 } to { background-position: 0 0 } }
</style>
