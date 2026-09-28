<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { User, Lock, ArrowRight, AlertCircle, Server, RotateCw, ChevronRight } from 'lucide-vue-next'
import { connections, login, me, ping } from './api'
import Btn from './ui/Btn.vue'
import Field from './ui/Field.vue'
import Logo from './ui/Logo.vue'
import { parseTarget, buildURL, DEFAULT_PORT } from './target'

const emit = defineEmits(['signed-in'])

const user = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)
const userField = ref(null)

// null while checking, then { name, url?, ok, error? } from /api/ping. Only
// the desktop app sends url, and only then can the server be edited.
const server = ref(null)
const checking = ref(false)
const editable = ref(false)
const target = ref({ scheme: 'http', host: '', port: DEFAULT_PORT })
const LAST = 'chgui.server'
// Servers the desktop app connected to before; one click reconnects.
const recent = ref([])
const opening = ref('')

const url = computed(() => (editable.value && target.value.host.trim() ? buildURL(target.value) : undefined))
const canSubmit = computed(() => user.value.trim().length > 0 && !busy.value)

async function check() {
  checking.value = true
  server.value = await ping(url.value)
  checking.value = false
}

// Pasting "https://host:9440" into Host fills the other fields instead of
// producing a host that is silently wrong.
function splitHost() {
  const parsed = parseTarget(target.value.host, target.value.port)
  if (parsed) target.value = parsed
  check()
}

onMounted(async () => {
  await check()
  if (!server.value.url) return
  let last = ''
  try { last = localStorage.getItem(LAST) || '' } catch {}
  // A saved URL without a port means the scheme's default, not ClickHouse's.
  const saved = last || server.value.url
  target.value = parseTarget(saved, saved.startsWith('https:') ? '443' : '80')
  editable.value = true
  recent.value = (await connections()).connections.filter((c) => c.saved && !c.broken)
  if (last && last !== server.value.url) await check()
})

async function reopen(row) {
  if (opening.value) return
  opening.value = row.id
  error.value = ''
  try {
    await login('', '', row.id)
    emit('signed-in', await me())
  } catch (e) {
    error.value = e.message
  } finally {
    opening.value = ''
  }
}

async function submit() {
  if (!canSubmit.value) return
  busy.value = true
  error.value = ''
  try {
    await login(user.value.trim(), password.value, undefined, url.value)
    if (url.value) try { localStorage.setItem(LAST, url.value) } catch {}
    emit('signed-in', await me())
  } catch (e) {
    error.value = e.message
    password.value = ''
    // A failure may mean the server went away; refresh the banner to match.
    check()
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
        <h1>Connect to ClickHouse</h1>
        <p>Use a ClickHouse user. There is no separate app account.</p>
      </header>

      <section v-if="recent.length" class="recent" aria-label="Saved connections">
        <span class="lbl">Saved connections</span>
        <button
          v-for="row in recent"
          :key="row.id"
          type="button"
          class="conn"
          :disabled="!!opening"
          @click="reopen(row)"
        >
          <Server :size="14" class="ico" />
          <span class="who">
            <span class="name">{{ row.name }}</span>
            <span class="meta">{{ row.user }} · {{ row.url }}</span>
          </span>
          <RotateCw v-if="opening === row.id" :size="14" class="spin" />
          <ChevronRight v-else :size="14" class="go-ico" />
        </button>
        <span class="or">or connect to a server</span>
      </section>

      <div v-if="editable" class="hostrow">
        <label class="scheme">
          <span class="lbl">Scheme</span>
          <select v-model="target.scheme" @change="check">
            <option value="http">http</option>
            <option value="https">https</option>
          </select>
        </label>
        <!-- Field puts class on its input, so size the wrappers instead. -->
        <div class="host">
          <Field v-model="target.host" label="Host" placeholder="clickhouse.internal"
                 mono :invalid="server && !server.ok" @blur="splitHost" @keydown.enter.prevent="splitHost">
            <template #icon><Server :size="15" /></template>
          </Field>
        </div>
        <div class="port">
          <Field v-model="target.port" label="Port" :placeholder="DEFAULT_PORT"
                 mono inputmode="numeric" @change="check" />
        </div>
      </div>

      <div v-else :class="['server', { down: server && !server.ok }]">
        <Server :size="15" class="ico" />
        <span class="where">
          <span class="lbl">Server</span>
          <span class="addr">{{ server ? server.name : 'Checking…' }}</span>
        </span>
      </div>

      <p v-if="server" :class="['state', { down: !server.ok }]">
        <span class="dot" />{{ checking ? 'Checking…' : server.ok ? 'Reachable' : 'Unreachable' }}
      </p>

      <Transition name="err">
        <div v-if="server && !server.ok && !error" class="error down-msg" role="alert">
          <AlertCircle :size="15" />
          <span>{{ server.error }}</span>
          <Btn type="button" size="sm" class="retry" :loading="checking" @click="check">
            <template #icon><RotateCw :size="13" /></template>
            Retry
          </Btn>
        </div>
      </Transition>

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
          {{ busy ? 'Connecting' : 'Connect' }}
          <template #after>
            <ArrowRight v-if="!busy" :size="15" />
          </template>
        </Btn>
      </form>

      <footer>
        Statements run as this user. ClickHouse decides what you may see and change.
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

.server {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--bg-sunken);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
}

.server .ico { flex: none; color: var(--text-tertiary); }
.where { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.where .lbl { color: var(--text-tertiary); font-size: var(--fs-xs); }
.addr {
  overflow: hidden;
  font: 500 var(--fs-sm)/1.4 var(--font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.recent { display: flex; flex-direction: column; gap: 6px; margin-bottom: var(--s5); }
.recent .lbl { color: var(--text-secondary); font-size: var(--fs-sm); font-weight: 500; }
.conn {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 11px;
  background: var(--bg-sunken);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-primary);
  text-align: left;
  cursor: pointer;
}
.conn:hover:not(:disabled) { border-color: var(--accent-border); }
.conn:focus-visible { outline: 2px solid var(--accent); outline-offset: 1px; }
.conn:disabled { cursor: default; opacity: 0.7; }
.conn .ico, .conn .go-ico { flex: none; color: var(--text-tertiary); }
.who { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.who .name { font-size: var(--fs-sm); font-weight: 500; }
.who .meta {
  overflow: hidden;
  color: var(--text-tertiary);
  font: 400 var(--fs-xs)/1.4 var(--font-mono);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.spin { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.or {
  margin-top: var(--s2);
  color: var(--text-tertiary);
  font-size: var(--fs-xs);
  text-align: center;
}

.hostrow { display: flex; gap: 6px; align-items: flex-end; }
.hostrow .host { flex: 1; min-width: 0; }
.hostrow .port { width: 78px; flex: none; }
.scheme { width: 78px; flex: none; }
.scheme .lbl {
  display: block;
  margin-bottom: 6px;
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: 500;
}
.scheme select {
  width: 100%;
  height: 38px;
  padding: 0 6px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-primary);
  font-size: var(--fs-sm);
}

.state {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 8px 0 var(--s4);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
}

.dot { width: 7px; height: 7px; border-radius: var(--r-full); background: var(--success); }
.down .dot { background: var(--error); }
.down { border-color: var(--error-border); }


.error.down-msg { align-items: center; margin: 0 0 var(--s5); }
.down-msg span { flex: 1; }
.retry { flex: none; }

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
