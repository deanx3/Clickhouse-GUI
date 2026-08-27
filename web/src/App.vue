<script setup>
import { onMounted, ref } from 'vue'
import { me } from './api'
import Login from './Login.vue'
import Shell from './Shell.vue'
import Toaster from './ui/Toaster.vue'

const session = ref(null)
const ready = ref(false)

onMounted(async () => {
  session.value = await me()
  ready.value = true
})
</script>

<template>
  <Shell v-if="ready && session" :session="session" @signed-out="session = null" />
  <Login v-else-if="ready" @signed-in="session = $event" />
  <Toaster />
</template>
