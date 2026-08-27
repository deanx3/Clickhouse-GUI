<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronDown, Server, Check, LogIn, Plus, Trash2, X } from 'lucide-vue-next'
import { connections, switchTo, login, addConnection, removeConnection } from '../api'
import { toast } from '../toast'
import Field from './Field.vue'
import { parseTarget, buildURL, DEFAULT_PORT } from '../target'
import Btn from './Btn.vue'

const emit = defineEmits(['changed'])

const open = ref(false)
const rows = ref([])
const activeId = ref('')
const canAdd = ref(false)
const root = ref(null)

const mode = ref(null)      // null | 'add' | connection id being signed into
const blank = () => ({ name: '', scheme: 'http', host: '', port: DEFAULT_PORT, user: 'default', password: '' })
const form = ref(blank())
const busy = ref(false)
const error = ref('')

const activeRow = () => rows.value.find((c) => c.id === activeId.value)

async function refresh() {
  const data = await connections()
  rows.value = data.connections
  activeId.value = data.active
  canAdd.value = data.canAdd
}

refresh()

function reset(next) {
  mode.value = mode.value === next ? null : next
  error.value = ''
  form.value = blank()
}

async function choose(row) {
  if (row.id === activeId.value) { open.value = false; return }
  if (!row.signedIn) { reset(row.id); return }

  await switchTo(row.id)
  open.value = false
  mode.value = null
  await refresh()
  toast(`Switched to ${row.name}`)
  emit('changed')
}

async function submit(row) {
  busy.value = true
  error.value = ''
  try {
    if (row) await login(form.value.user.trim(), form.value.password, row.id)
    else await addConnection({
      name: form.value.name,
      url: buildURL(form.value),
      user: form.value.user.trim(),
      password: form.value.password,
    })

    open.value = false
    mode.value = null
    await refresh()
    toast(row ? `Connected to ${row.name}` : 'Connection added')
    emit('changed')
  } catch (e) {
    error.value = e.message
  } finally {
    busy.value = false
  }
}

// Pasting "https://host:9440" into Host fills the other fields instead of
// producing a host that is silently wrong.
function splitHost() {
  const parsed = parseTarget(form.value.host, form.value.port)
  if (!parsed) return
  form.value.scheme = parsed.scheme
  form.value.host = parsed.host
  form.value.port = parsed.port
}

async function remove(row) {
  await removeConnection(row.id)
  await refresh()
  toast(`Removed ${row.name}`)
  emit('changed')
}

const away = (e) => {
  if (!root.value?.contains(e.target)) { open.value = false; mode.value = null }
}

onMounted(() => document.addEventListener('mousedown', away))
onBeforeUnmount(() => document.removeEventListener('mousedown', away))

defineExpose({ refresh })
</script>

<template>
  <div ref="root" class="cm">
    <button class="trigger" :aria-expanded="open" @click="open = !open">
      <Server :size="14" />
      <span class="label">{{ activeRow()?.name ?? 'Connection' }}</span>
      <ChevronDown :size="13" :class="['chev', { up: open }]" />
    </button>

    <Transition name="pop">
      <div v-if="open" class="pane" role="menu">
        <p class="head">Connections</p>

        <template v-for="row in rows" :key="row.id">
          <div class="rowwrap">
            <button class="row" :class="{ on: row.id === activeId, broken: row.broken }"
                    @click="choose(row)">
              <span class="dot" :class="{ live: row.signedIn }" />
              <span class="names">
                <span class="nm">{{ row.name }}</span>
                <span class="sub ident">
                  {{ row.signedIn ? `${row.user} · ${row.url}` : 'not signed in' }}
                </span>
                <span v-if="row.broken" class="badbadge">
                  unreadable — CHGUI_SECRET changed, remove and re-add
                </span>
                <span v-if="row.saved" class="shared">
                  shared · added by {{ row.createdBy }}
                </span>
              </span>
              <Check v-if="row.id === activeId" :size="14" class="tick" />
              <LogIn v-else-if="!row.signedIn" :size="13" class="tick dim" />
            </button>
            <button v-if="row.saved" class="kill" :data-tip="`Remove ${row.name}`"
                    :aria-label="`Remove ${row.name}`" @click.stop="remove(row)">
              <Trash2 :size="12" />
            </button>
          </div>

          <form v-if="mode === row.id" class="inline" @submit.prevent="submit(row)">
            <Field v-model="form.user" placeholder="user" mono :invalid="!!error" />
            <Field v-model="form.password" type="password" placeholder="password" reveal />
            <p v-if="error" class="err">{{ error }}</p>
            <Btn type="submit" variant="primary" size="sm" :loading="busy"
                 :disabled="!form.user.trim()">Sign in to {{ row.name }}</Btn>
          </form>
        </template>

        <template v-if="canAdd">
          <button class="add" @click="reset('add')">
            <component :is="mode === 'add' ? X : Plus" :size="14" />
            {{ mode === 'add' ? 'Cancel' : 'Add a connection' }}
          </button>

          <form v-if="mode === 'add'" class="inline" @submit.prevent="submit(null)">
            <Field v-model="form.name" label="Name" placeholder="Production EU" />

            <div class="hostrow">
              <label class="scheme">
                <span class="lbl">Scheme</span>
                <select v-model="form.scheme">
                  <option value="http">http</option>
                  <option value="https">https</option>
                </select>
              </label>
              <Field v-model="form.host" class="host" label="Host"
                     placeholder="clickhouse.internal" mono :invalid="!!error"
                     @blur="splitHost" />
              <Field v-model="form.port" class="port" label="Port"
                     :placeholder="DEFAULT_PORT" mono inputmode="numeric" />
            </div>

            <p class="preview ident">{{ buildURL(form) }}</p>

            <Field v-model="form.user" label="User" placeholder="default" mono />
            <Field v-model="form.password" label="Password" type="password" reveal />
            <p v-if="error" class="err">{{ error }}</p>
            <Btn type="submit" variant="primary" size="sm" :loading="busy"
                 :disabled="!form.host.trim() || !form.user.trim()">
              Test and add
            </Btn>
            <p class="note warn">
              Saved on the server and <b>shared with everyone using chgui</b>.
              The password is encrypted at rest, but anyone here can query as
              this user. The credentials are checked before saving.
            </p>
          </form>
        </template>

        <p v-else class="note edge">Connections are configured on the server.</p>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.cm { position: relative; }

.trigger {
  display: flex;
  align-items: center;
  gap: 7px;
  height: var(--control-h);
  padding: 0 8px;
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  transition: border-color var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}

.trigger:hover, .trigger[aria-expanded="true"] {
  border-color: var(--border-strong);
  color: var(--text-primary);
}

.label { max-width: 150px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.chev { color: var(--text-tertiary); transition: transform var(--t-fast) var(--ease); }
.chev.up { transform: rotate(180deg); }

.pane {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 70;
  width: 312px;
  max-height: 76vh;
  overflow: auto;
  padding: 5px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-lg);
}

.head {
  margin: 0;
  padding: 7px 9px;
  color: var(--text-disabled);
  font-size: var(--fs-xs);
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.rowwrap { display: flex; align-items: center; }

.row {
  display: flex;
  align-items: center;
  gap: 9px;
  flex: 1;
  min-width: 0;
  padding: 8px 9px;
  background: none;
  border: 0;
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  text-align: left;
}

.row:hover { background: var(--bg-hover); }
.row.on { background: var(--accent-subtle); }

.dot {
  flex: none;
  width: 7px;
  height: 7px;
  border-radius: var(--r-full);
  background: var(--border-strong);
}

.dot.live { background: var(--success); box-shadow: 0 0 0 3px var(--success-subtle); }

.names { flex: 1; min-width: 0; }
.nm { display: block; color: var(--text-primary); font-size: var(--fs-sm); }

.sub {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-tertiary);
  font-size: 10px;
}

.tick { flex: none; color: var(--accent); }
.tick.dim { color: var(--text-disabled); }

.kill {
  display: flex;
  flex: none;
  padding: 6px;
  margin-right: 3px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-disabled);
}

.kill:hover { background: var(--error-subtle); color: var(--error); }

.add {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  margin-top: 3px;
  padding: 9px;
  background: none;
  border: 0;
  border-top: 1px solid var(--border-subtle);
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  text-align: left;
}

.add:hover { background: var(--bg-hover); color: var(--text-primary); }

.inline {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 4px;
  padding: 11px;
  background: var(--bg-panel);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-sm);
  animation: fade-in var(--t-fast) var(--ease);
}

.hostrow { display: flex; gap: 6px; align-items: flex-end; }
.hostrow .host { flex: 1; min-width: 0; }
.hostrow .port { width: 74px; flex: none; }

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

.preview {
  margin: 0;
  padding: 6px 8px;
  background: var(--bg-sunken);
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
  font-size: 10px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.err { margin: 0; color: var(--error); font: 400 10px/1.45 var(--font-mono); word-break: break-word; }

.note { margin: 2px 0 0; color: var(--text-tertiary); font-size: 10px; line-height: 1.5; }

.note.warn {
  padding: 7px 8px;
  background: var(--warning-subtle);
  border-radius: var(--r-xs);
  color: var(--warning);
}

.note.warn b { color: var(--warning); }

.broken .nm { color: var(--text-tertiary); text-decoration: line-through; }
.broken .dot { background: var(--error); box-shadow: 0 0 0 3px var(--error-subtle); }

.badbadge {
  display: block;
  margin-top: 2px;
  color: var(--error);
  font-size: 9px;
  line-height: 1.4;
}

.shared {
  display: block;
  color: var(--text-disabled);
  font-size: 9px;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}
.note.edge { padding: 9px; border-top: 1px solid var(--border-subtle); }
</style>
