<script>
export default { inheritAttrs: false }
</script>

<script setup>
import { ref } from 'vue'
import { Eye, EyeOff } from 'lucide-vue-next'

const props = defineProps({
  type: { type: String, default: 'text' },
  label: String,
  placeholder: String,
  autocomplete: String,
  mono: Boolean,
  invalid: Boolean,
  reveal: Boolean, // show a password visibility toggle
})

const model = defineModel({ type: String, default: '' })
const shown = ref(false)
const el = ref(null)

defineExpose({ focus: () => el.value?.focus() })
</script>

<template>
  <label class="field">
    <span v-if="label" class="lbl">{{ label }}</span>
    <span :class="['box', { invalid }]">
      <span class="lead"><slot name="icon" /></span>
      <input
        ref="el"
        v-bind="$attrs"
        v-model="model"
        :type="reveal && shown ? 'text' : type"
        :placeholder="placeholder"
        :autocomplete="autocomplete"
        :class="{ mono }"
        autocapitalize="off"
        spellcheck="false"
      />
      <button
        v-if="reveal"
        type="button"
        class="peek"
        :data-tip="shown ? 'Hide password' : 'Show password'"
        :aria-label="shown ? 'Hide password' : 'Show password'"
        @click="shown = !shown"
      >
        <component :is="shown ? EyeOff : Eye" :size="15" />
      </button>
    </span>
  </label>
</template>

<style scoped>
.field { display: block; }

.lbl {
  display: block;
  margin-bottom: 6px;
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: 500;
}

.box {
  display: flex;
  align-items: center;
  height: 38px;
  padding: 0 4px 0 10px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  transition: border-color var(--t-fast) var(--ease), box-shadow var(--t-fast) var(--ease);
}

.box:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-subtle);
}

.box.invalid { border-color: var(--error); }
.box.invalid:focus-within { box-shadow: 0 0 0 3px var(--error-subtle); }

.lead { display: flex; color: var(--text-tertiary); }
.lead:empty { display: none; }

input {
  flex: 1;
  min-width: 0;
  height: 100%;
  padding: 0 8px;
  background: none;
  border: 0;
  outline: none;
  font-size: var(--fs-md);
}

input.mono { font-family: var(--font-mono); font-size: var(--fs-base); }
input::placeholder { color: var(--text-disabled); }

.peek {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  background: none;
  border: 0;
  border-radius: var(--r-xs);
  color: var(--text-tertiary);
  transition: color var(--t-fast) var(--ease), background var(--t-fast) var(--ease);
}

.peek:hover { background: var(--bg-hover); color: var(--text-primary); }
</style>
