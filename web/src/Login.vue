<script setup>
import { computed, nextTick, ref } from 'vue'
import { User, Lock, ArrowRight, AlertCircle } from 'lucide-vue-next'
import { login, me } from './api'
import Btn from './ui/Btn.vue'
import Field from './ui/Field.vue'
import Logo from './ui/Logo.vue'

const emit = defineEmits(['signed-in'])

const user = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)
const userField = ref(null)

const canSubmit = computed(() => user.value.trim().length > 0 && !busy.value)

async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ''
  try {
    await login(user.value.trim(), password.value)
    emit('signed-in', await me())
  } catch (e) {
    error.value = e.message
    password.value = ''
    await nextTick()
    userField.value?.focus()
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="page">
    <div class="glow" aria-hidden="true" />
    <div class="grid" aria-hidden="true" />

    <section class="panel">
      <header class="brand">
        <span class="mark"><Logo :size="26" /></span>
        <h1>ClickHouse Console</h1>
        <p>Explore schemas, run SQL, and manage access.</p>
      </header>

      <form novalidate @submit.prevent="submit">
        <Field
          id="user"
          ref="userField"
          v-model="user"
          name="username"
          label="User"
          placeholder="default"
          autocomplete="username"
          mono
          :invalid="!!error"
        >
          <template #icon><User :size="15" /></template>
        </Field>

        <Field
          id="password"
          v-model="password"
          class="pw"
          name="password"
          type="password"
          label="Password"
          placeholder="••••••••"
          autocomplete="current-password"
          reveal
          :invalid="!!error"
        >
          <template #icon><Lock :size="15" /></template>
        </Field>

        <Transition name="err">
          <p v-if="error" class="error" role="alert">
            <AlertCircle :size="15" />
            <span>{{ error }}</span>
          </p>
        </Transition>

        <Btn type="submit" variant="primary" class="go" :loading="busy" :disabled="!canSubmit">
          {{ busy ? 'Signing in' : 'Sign in' }}
          <template #after>
            <ArrowRight v-if="!busy" :size="15" />
          </template>
        </Btn>
      </form>

      <footer>
        Statements run as you. ClickHouse decides what you may see and change.
      </footer>
    </section>

    <p class="version">chgui v1.0.0 <span>&bull;</span> ClickHouse HTTP</p>
  </main>
</template>

<style scoped>
.page {
  position: relative;
  display: grid;
  place-items: center;
  min-height: 100%;
  padding: var(--s6);
  overflow: hidden;
  background: var(--bg-sunken);
}

/* One soft light source behind the card, so the screen is not a flat void. */
.glow {
  position: absolute;
  top: -22%;
  left: 50%;
  width: min(900px, 130vw);
  aspect-ratio: 1;
  translate: -50% 0;
  background: radial-gradient(circle, rgb(240 136 62 / 0.11) 0%, transparent 62%);
  pointer-events: none;
}

.grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(var(--border-subtle) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
  background-size: 44px 44px;
  mask-image: radial-gradient(ellipse 70% 55% at 50% 40%, #000 20%, transparent 75%);
  opacity: 0.55;
  pointer-events: none;
}

.panel {
  position: relative;
  width: 100%;
  max-width: 392px;
  padding: var(--s8) var(--s8) var(--s5);
  background: var(--bg-panel);
  border: 1px solid var(--border);
  border-radius: var(--r-lg);
  box-shadow: var(--shadow-lg);
  animation: rise var(--t) var(--ease);
}

.brand { text-align: center; margin-bottom: var(--s6); }

.mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  margin-bottom: var(--s4);
  background: var(--accent-subtle);
  border: 1px solid var(--accent-border);
  border-radius: var(--r-md);
  color: var(--accent);
}

h1 { margin: 0; font-size: var(--fs-xl); font-weight: 600; letter-spacing: -0.02em; }

.brand p {
  margin: 6px 0 0;
  color: var(--text-secondary);
  font-size: var(--fs-base);
}

.pw { margin-top: var(--s4); }
.go { width: 100%; height: 38px; margin-top: var(--s5); font-size: var(--fs-md); }

.error {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: var(--s4) 0 0;
  padding: 10px 11px;
  background: var(--error-subtle);
  border: 1px solid var(--error-border);
  border-radius: var(--r-sm);
  color: var(--error);
  font: 400 var(--fs-sm)/1.5 var(--font-mono);
  word-break: break-word;
}

.error svg { flex: none; margin-top: 1px; }

.err-enter-active { transition: all var(--t) var(--ease); }
.err-enter-from { opacity: 0; transform: translateY(-4px); }

footer {
  margin-top: var(--s6);
  padding-top: var(--s4);
  border-top: 1px solid var(--border-subtle);
  color: var(--text-tertiary);
  font-size: var(--fs-sm);
  line-height: 1.6;
  text-align: center;
}

.version {
  position: absolute;
  bottom: var(--s5);
  left: 50%;
  translate: -50% 0;
  margin: 0;
  color: var(--text-disabled);
  font-size: var(--fs-xs);
  letter-spacing: 0.02em;
}

.version span { opacity: 0.6; }

@media (max-height: 620px) {
  .version { display: none; }
}
</style>
