<script setup>
defineProps({ size: { type: Number, default: 28 } })
</script>

<template>
  <!-- Column bars over a baseline: a columnar store, drawn rather than borrowed. -->
  <svg :width="size" :height="size" viewBox="0 0 32 32" fill="none" aria-hidden="true">
    <rect x="3" y="5" width="4.5" height="16" rx="1.2" fill="currentColor" opacity=".45" />
    <rect x="10.2" y="5" width="4.5" height="22" rx="1.2" fill="currentColor" />
    <rect x="17.4" y="5" width="4.5" height="12" rx="1.2" fill="currentColor" opacity=".7" />
    <rect x="24.6" y="5" width="4.5" height="19" rx="1.2" fill="currentColor" opacity=".3" />
  </svg>
</template>
