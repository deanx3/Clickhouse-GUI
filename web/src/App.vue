<script setup>
import { onMounted, ref } from 'vue'
import { me } from './api'
import Login from './Login.vue'
import Shell from './Shell.vue'
import Toaster from './ui/Toaster.vue'

const session = ref(null)
const ready = ref(false)

async function reload() {
  // Re-ask rather than assume signed out: the desktop app signs straight back in.
  session.value = await me()
  ready.value = true
}

onMounted(reload)
</script>

<template>
  <Shell v-if="ready && session" :session="session" @signed-out="reload" />
  <Login v-else-if="ready" @signed-in="session = $event" />
  <Toaster />
</template>
