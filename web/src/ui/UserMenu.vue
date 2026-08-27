<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ChevronDown, LogOut, Server, UserRound } from 'lucide-vue-next'

defineProps({ session: Object })
const emit = defineEmits(['sign-out'])

const open = ref(false)
const root = ref(null)

const away = (e) => { if (!root.value?.contains(e.target)) open.value = false }
onMounted(() => document.addEventListener('mousedown', away))
onBeforeUnmount(() => document.removeEventListener('mousedown', away))
</script>

<template>
  <div ref="root" class="menu">
    <button class="trigger" :aria-expanded="open" @click="open = !open" @keydown.esc="open = false">
      <span class="avatar"><UserRound :size="14" /></span>
      <span class="who ident">{{ session.user }}</span>
      <ChevronDown :size="14" :class="['chev', { up: open }]" />
    </button>

    <Transition name="pop">
      <div v-if="open" class="pane" role="menu">
        <div class="head">
          <p class="ident name">{{ session.user }}</p>
          <p class="sub">Signed in to ClickHouse</p>
        </div>
        <p class="row">
          <Server :size="14" />
          <span class="ident url">{{ session.clickhouse }}</span>
        </p>
        <button class="item" role="menuitem" @click="open = false; emit('sign-out')">
          <LogOut :size="14" /> Sign out
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.menu { position: relative; }

.trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  height: var(--control-h);
  padding: 0 8px 0 5px;
  background: none;
  border: 1px solid transparent;
  border-radius: var(--r-sm);
  color: var(--text-primary);
  transition: background var(--t-fast) var(--ease), border-color var(--t-fast) var(--ease);
}

.trigger:hover, .trigger[aria-expanded="true"] {
  background: var(--bg-hover);
  border-color: var(--border);
}

.avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  background: var(--accent-subtle);
  border-radius: var(--r-full);
  color: var(--accent);
}

.who { font-size: var(--fs-sm); }
.chev { color: var(--text-tertiary); transition: transform var(--t-fast) var(--ease); }
.chev.up { transform: rotate(180deg); }

.pane {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 70;
  min-width: 238px;
  padding: 5px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-strong);
  border-radius: var(--r-md);
  box-shadow: var(--shadow-lg);
}

.head { padding: 8px 9px 9px; border-bottom: 1px solid var(--border-subtle); }
.name { margin: 0; font-size: var(--fs-base); }
.sub { margin: 2px 0 0; color: var(--text-tertiary); font-size: var(--fs-xs); }

.row {
  display: flex;
  align-items: center;
  gap: 9px;
  margin: 0;
  padding: 9px;
  color: var(--text-tertiary);
}

.url { color: var(--text-secondary); font-size: var(--fs-xs); word-break: break-all; }

.item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 8px 9px;
  background: none;
  border: 0;
  border-top: 1px solid var(--border-subtle);
  border-radius: var(--r-xs);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  text-align: left;
}

.item:hover { background: var(--error-subtle); color: var(--error); }

.pop-enter-active, .pop-leave-active { transition: all var(--t-fast) var(--ease); }
.pop-enter-from, .pop-leave-to { opacity: 0; transform: translateY(-5px) scale(0.98); }
</style>
