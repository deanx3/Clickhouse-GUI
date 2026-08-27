<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { Search, CornerDownLeft } from 'lucide-vue-next'

const props = defineProps({ open: Boolean, commands: Array })
const emit = defineEmits(['close', 'run'])

const q = ref('')
const cursor = ref(0)
const input = ref(null)

const matches = computed(() => {
  const needle = q.value.trim().toLowerCase()
  const list = props.commands.filter((c) => !c.hidden)
  if (!needle) return list
  return list
    .map((c) => {
      const hay = `${c.group} ${c.label} ${c.hint ?? ''}`.toLowerCase()
      const at = hay.indexOf(needle)
      return at < 0 ? null : { ...c, rank: at }
    })
    .filter(Boolean)
    .sort((a, b) => a.rank - b.rank)
})

watch(matches, () => { cursor.value = 0 })

watch(() => props.open, async (on) => {
  if (!on) return
  q.value = ''
  cursor.value = 0
  await nextTick()
  input.value?.focus()
})

function move(step) {
  const n = matches.value.length
  if (n) cursor.value = (cursor.value + step + n) % n
}

function choose(cmd = matches.value[cursor.value]) {
  if (!cmd) return
  emit('close')
  emit('run', cmd)
}
</script>

<template>
  <Transition name="pal">
    <div v-if="open" class="scrim" @click.self="emit('close')" @keydown.esc="emit('close')">
      <div class="pal" role="dialog" aria-modal="true" aria-label="Command palette">
        <div class="bar">
          <Search :size="16" />
          <input
            ref="input"
            v-model="q"
            placeholder="Search commands, tables, actions…"
            spellcheck="false"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="choose()"
            @keydown.esc.prevent="emit('close')"
          />
          <kbd>Esc</kbd>
        </div>

        <ul v-if="matches.length" class="list">
          <template v-for="(cmd, i) in matches" :key="cmd.id">
            <li v-if="i === 0 || matches[i - 1].group !== cmd.group" class="group">{{ cmd.group }}</li>
            <li
              :class="['item', { on: i === cursor }]"
              @mousemove="cursor = i"
              @click="choose(cmd)"
            >
              <component :is="cmd.icon" :size="15" />
              <span class="label">{{ cmd.label }}</span>
              <span v-if="cmd.hint" class="hint ident">{{ cmd.hint }}</span>
              <kbd v-if="cmd.keys">{{ cmd.keys }}</kbd>
              <CornerDownLeft v-else-if="i === cursor" :size="13" class="enter" />
            </li>
          </template>
        </ul>

        <p v-else class="none">Nothing matches “{{ q }}”.</p>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.scrim {
  position: fixed;
  inset: 0;
  z-index: 100;
  display: flex;
  justify-content: center;
  padding-top: 14vh;
  background: rgb(3 5 9 / 0.62);
  backdrop-filter: blur(3px);
}

.pal {
  width: min(600px, 92vw);
  max-height: 62vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

.bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 var(--s3) 0 var(--s4);
  height: 50px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-tertiary);
}

.bar input {
  flex: 1;
  min-width: 0;
  background: none;
  border: 0;
  outline: none;
  color: var(--text-primary);
  font-size: var(--fs-md);
}

.bar input::placeholder { color: var(--text-disabled); }

.list { flex: 1; margin: 0; padding: 5px; overflow: auto; list-style: none; }

.group {
  padding: 9px 9px 5px;
  color: var(--text-disabled);
  font-size: var(--fs-xs);
  font-weight: 500;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.item {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 8px 10px;
  border-radius: var(--r-sm);
  color: var(--text-secondary);
  font-size: var(--fs-base);
  cursor: pointer;
}

.item.on { background: var(--bg-active); color: var(--text-primary); }
.item.on svg:first-child { color: var(--accent); }
.label { flex: none; }

.hint {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text-disabled);
  font-size: var(--fs-xs);
}

.enter { margin-left: auto; color: var(--text-tertiary); }
.item kbd { margin-left: auto; }
.none { padding: 26px; color: var(--text-tertiary); font-size: var(--fs-base); text-align: center; }

.pal-enter-active, .pal-leave-active { transition: opacity var(--t-fast) var(--ease); }
.pal-enter-active .pal, .pal-leave-active .pal { transition: transform var(--t) var(--ease); }
.pal-enter-from, .pal-leave-to { opacity: 0; }
.pal-enter-from .pal, .pal-leave-to .pal { transform: translateY(-10px) scale(0.98); }
</style>
