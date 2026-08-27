<script setup>
import { onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import { EditorState, Compartment } from '@codemirror/state'
import {
  EditorView, keymap, lineNumbers, highlightActiveLine, highlightActiveLineGutter,
  drawSelection, rectangularSelection, highlightSpecialChars,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
import {
  autocompletion, completionKeymap, closeBrackets, closeBracketsKeymap,
} from '@codemirror/autocomplete'
import {
  bracketMatching, indentOnInput, foldGutter, foldKeymap, indentUnit,
} from '@codemirror/language'
import { sql, StandardSQL } from '@codemirror/lang-sql'
import { chguiTheme, chguiHighlight } from './editor-theme'

const props = defineProps({
  modelValue: { type: String, default: '' },
  schema: { type: Object, default: () => ({}) }, // { 'db.table': ['col', ...] }
})
const emit = defineEmits(['update:modelValue', 'run', 'save'])

const host = ref(null)
const view = shallowRef(null)
const schemaSlot = new Compartment()

// Rebuilding the whole editor on every schema load would lose the cursor, so
// the SQL dialect lives in a compartment that can be swapped in place.
const sqlExtension = () =>
  sql({ dialect: StandardSQL, schema: props.schema, upperCaseKeywords: true })

onMounted(() => {
  view.value = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        lineNumbers(),
        foldGutter(),
        highlightSpecialChars(),
        history(),
        drawSelection(),
        rectangularSelection(),
        indentOnInput(),
        indentUnit.of('  '),
        bracketMatching(),
        closeBrackets(),
        autocompletion({ activateOnTyping: true, icons: true }),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        schemaSlot.of(sqlExtension()),
        chguiTheme,
        chguiHighlight,
        keymap.of([
          { key: 'Mod-Enter', preventDefault: true, run: () => (emit('run'), true) },
          { key: 'Shift-Enter', preventDefault: true, run: () => (emit('run'), true) },
          { key: 'Mod-s', preventDefault: true, run: () => (emit('save'), true) },
          ...closeBracketsKeymap,
          ...completionKeymap,
          ...historyKeymap,
          ...foldKeymap,
          indentWithTab,
          ...defaultKeymap,
        ]),
        EditorView.lineWrapping,
        EditorView.updateListener.of((u) => {
          if (u.docChanged) emit('update:modelValue', u.state.doc.toString())
        }),
      ],
    }),
  })
})

onBeforeUnmount(() => view.value?.destroy())

watch(() => props.schema, () => {
  view.value?.dispatch({ effects: schemaSlot.reconfigure(sqlExtension()) })
})

// Only push in changes that came from elsewhere, or typing fights the update.
watch(() => props.modelValue, (next) => {
  const v = view.value
  if (!v || next === v.state.doc.toString()) return
  v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: next } })
})

defineExpose({ focus: () => view.value?.focus() })
</script>

<template>
  <div ref="host" class="editor" />
</template>

<style scoped>
.editor { height: 100%; overflow: hidden; }
.editor :deep(.cm-editor) { height: 100%; }
</style>
