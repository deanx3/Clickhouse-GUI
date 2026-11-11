<script setup>
import { nextTick, ref } from 'vue'
import { Sparkles, X, Trash2, ArrowUp, Square, Search, Copy, FileInput, ListPlus, Check } from 'lucide-vue-next'
import { ask } from './api'
import { parts, isAccess, stepLabel } from './assistant'
import { stage, isStaged } from './ledger'
import { copy } from './clipboard'
import Btn from './ui/Btn.vue'
import State from './ui/State.vue'

// context: what the user is looking at - { sql, database, table, error }.
const props = defineProps({ context: Object })
const emit = defineEmits(['use', 'close'])

const STARTERS = [
  'What tables are here, and what is in them?',
  'Explain the statement in the editor.',
  'Summarise the table I have open.',
]

// { role, content, steps?, error? } - steps and error are display only.
const messages = ref([])
const draft = ref('')
const busy = ref(false)
const scroller = ref(null)
const input = ref(null)
let abort = null

const toBottom = () => nextTick(() => {
  if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
})

async function send(text = draft.value) {
  text = text.trim()
  if (!text || busy.value) return
  draft.value = ''
  messages.value.push({ role: 'user', content: text })

  // Failed turns stay on screen but are not replayed to the model.
  const history = messages.value.filter((m) => m.content).map(({ role, content }) => ({ role, content }))
  messages.value.push({ role: 'assistant', content: '', steps: [], error: '' })
  const reply = messages.value[messages.value.length - 1]

  busy.value = true
  abort = new AbortController()
  toBottom()
  try {
    await ask({ messages: history, context: props.context }, (ev) => {
      if (ev.type === 'tool') reply.steps.push(ev)
      else if (ev.type === 'text') reply.content = ev.text
      else if (ev.type === 'error') reply.error = ev.text
      toBottom()
    }, abort.signal)
    if (!reply.content && !reply.error) reply.error = 'The assistant sent an empty reply.'
  } catch (e) {
    reply.error = e.name === 'AbortError' ? 'Stopped.' : e.message
  } finally {
    busy.value = false
    toBottom()
  }
}

function clear() {
  abort?.abort()
  messages.value = []
}

const focus = () => nextTick(() => input.value?.focus())

defineExpose({ send, focus })
</script>

<template>
  <aside class="assistant">
    <header>
      <Sparkles :size="14" class="mark" />
      <span class="ttl">Assistant</span>
      <Btn v-if="messages.length" size="sm" icon variant="ghost" data-tip="Clear chat" @click="clear">
        <template #icon><Trash2 :size="13" /></template>
      </Btn>
      <Btn size="sm" icon variant="ghost" data-tip="Close" @click="emit('close')">
        <template #icon><X :size="14" /></template>
      </Btn>
    </header>

    <div ref="scroller" class="body">
      <State v-if="!messages.length" title="Ask about your data"
             hint="It can read your schema and run read-only queries as you. It never changes anything: you run what it writes.">
        <template #icon><Sparkles :size="20" /></template>
        <template #actions>
          <div class="starters">
            <button v-for="s in STARTERS" :key="s" @click="send(s)">{{ s }}</button>
          </div>
        </template>
      </State>

      <template v-for="(m, i) in messages" :key="i">
        <p v-if="m.role === 'user'" class="you">{{ m.content }}</p>

        <div v-else class="reply">
          <p v-for="(s, j) in m.steps" :key="j" class="step" :title="stepLabel(s)">
            <Search :size="11" /><span>{{ stepLabel(s) }}</span>
          </p>

          <template v-for="(p, j) in parts(m.content)" :key="`p${j}`">
            <div v-if="p.code" class="code">
              <pre>{{ p.text }}</pre>
              <footer>
                <button v-if="isAccess(p.text)" :class="{ on: isStaged(p.text) }"
                        @click="stage({ sql: p.text, note: 'From the assistant' })">
                  <component :is="isStaged(p.text) ? Check : ListPlus" :size="12" />
                  {{ isStaged(p.text) ? 'Staged' : 'Stage in ledger' }}
                </button>
                <button v-else @click="emit('use', p.text)">
                  <FileInput :size="12" /> Insert in editor
                </button>
                <button @click="copy(p.text, 'Copied')"><Copy :size="12" /> Copy</button>
              </footer>
            </div>
            <p v-else class="prose">{{ p.text }}</p>
          </template>

          <p v-if="m.error" class="err">{{ m.error }}</p>
          <p v-else-if="busy && i === messages.length - 1" class="thinking">
            <span /><span /><span />
          </p>
        </div>
      </template>
    </div>

    <form class="ask" @submit.prevent="send()">
      <textarea ref="input" v-model="draft" rows="2" placeholder="Ask about your data"
                aria-label="Message the assistant" spellcheck="false"
                @keydown.enter.exact.prevent="send()" />
      <Btn v-if="busy" size="sm" icon data-tip="Stop" type="button" @click="abort?.abort()">
        <template #icon><Square :size="12" /></template>
      </Btn>
      <Btn v-else size="sm" icon variant="primary" data-tip="Send" :disabled="!draft.trim()">
        <template #icon><ArrowUp :size="14" /></template>
      </Btn>
    </form>
  </aside>
</template>

<style scoped>
.assistant {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  min-height: 0;
  background: var(--bg-panel);
  border-left: 1px solid var(--border);
}

header {
  display: flex;
  align-items: center;
  gap: 2px;
  height: 40px;
  flex: none;
  padding: 0 var(--s2) 0 var(--s3);
  border-bottom: 1px solid var(--border-subtle);
}

.mark { color: var(--accent); margin-right: 6px; }
.ttl { flex: 1; font-size: var(--fs-sm); font-weight: 600; }

.body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  display: flex;
  flex-direction: column;
  gap: var(--s3);
  padding: var(--s3);
}

.body > .state { flex: 1; }

.starters { display: flex; flex-direction: column; gap: 6px; width: 100%; }
.starters button {
  padding: 7px 10px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  text-align: left;
  transition: border-color var(--t-fast) var(--ease), color var(--t-fast) var(--ease);
}
.starters button:hover { border-color: var(--border-strong); color: var(--text-primary); }

p { margin: 0; }

.you {
  align-self: flex-end;
  max-width: 88%;
  padding: 7px 10px;
  background: var(--bg-active);
  border-radius: var(--r-md) var(--r-md) var(--r-xs) var(--r-md);
  color: var(--text-primary);
  font-size: var(--fs-base);
  line-height: 1.5;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.reply { display: flex; flex-direction: column; gap: var(--s2); min-width: 0; }

.step {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-tertiary);
  font: 400 var(--fs-xs)/1.5 var(--font-mono);
}
.step svg { flex: none; }
.step span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.prose {
  color: var(--text-primary);
  font-size: var(--fs-base);
  line-height: 1.6;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.code {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  overflow: hidden;
}

.code pre {
  margin: 0;
  padding: 9px 10px;
  max-height: 260px;
  overflow: auto;
  color: var(--text-primary);
  font: 400 var(--fs-sm)/1.55 var(--font-mono);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.code footer {
  display: flex;
  gap: 2px;
  padding: 3px;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.code button {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  font-size: var(--fs-xs);
}
.code button:hover { background: var(--bg-active); color: var(--text-primary); }
.code button.on { color: var(--accent-text); }

.err {
  padding: 7px 10px;
  background: var(--error-subtle);
  border: 1px solid var(--error-border);
  border-radius: var(--r-sm);
  color: var(--error);
  font-size: var(--fs-sm);
  line-height: 1.5;
  overflow-wrap: anywhere;
}

.thinking { display: flex; gap: 4px; padding: 4px 0; }
.thinking span {
  width: 5px; height: 5px;
  border-radius: var(--r-full);
  background: var(--text-tertiary);
  animation: pulse 1s var(--ease) infinite;
}
.thinking span:nth-child(2) { animation-delay: 150ms; }
.thinking span:nth-child(3) { animation-delay: 300ms; }
@keyframes pulse { 0%, 100% { opacity: 0.25; } 50% { opacity: 1; } }

.ask {
  display: flex;
  align-items: flex-end;
  gap: 6px;
  flex: none;
  margin: var(--s2);
  padding: 6px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-md);
}
.ask:focus-within { border-color: var(--accent); }

.ask textarea {
  flex: 1;
  min-width: 0;
  padding: 3px 4px;
  background: none;
  border: 0;
  outline: none;
  resize: none;
  color: var(--text-primary);
  font: 400 var(--fs-base)/1.5 var(--font-sans);
}

@media (prefers-reduced-motion: reduce) {
  .thinking span { animation: none; opacity: 0.6; }
}
</style>
