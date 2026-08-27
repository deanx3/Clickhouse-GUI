import { EditorView } from '@codemirror/view'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags as t } from '@lezer/highlight'

// Everything reads from the design tokens, so the editor cannot drift away
// from the rest of the UI when the palette changes.
export const chguiTheme = EditorView.theme({
  '&': {
    height: '100%',
    fontSize: 'var(--fs-base)',
    backgroundColor: 'var(--bg-base)',
    color: 'var(--text-primary)',
  },
  '.cm-scroller': { fontFamily: 'var(--font-mono)', lineHeight: '1.65' },
  '.cm-content': { padding: '12px 0', caretColor: 'var(--accent)' },
  '&.cm-focused': { outline: 'none' },
  '.cm-line': { padding: '0 16px' },

  '.cm-gutters': {
    backgroundColor: 'var(--bg-base)',
    borderRight: '1px solid var(--border-subtle)',
    color: 'var(--text-disabled)',
    paddingRight: '2px',
  },
  '.cm-lineNumbers .cm-gutterElement': { padding: '0 8px 0 14px', minWidth: '36px' },
  '.cm-activeLineGutter': { backgroundColor: 'transparent', color: 'var(--text-secondary)' },
  '.cm-activeLine': { backgroundColor: 'rgb(255 255 255 / 0.025)' },
  '.cm-foldGutter .cm-gutterElement': { padding: '0 4px', color: 'var(--text-disabled)' },

  '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--accent)', borderLeftWidth: '2px' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, ::selection': {
    backgroundColor: 'rgb(88 166 255 / 0.22)',
  },
  '.cm-matchingBracket, &.cm-focused .cm-matchingBracket': {
    backgroundColor: 'var(--accent-subtle)',
    outline: '1px solid var(--accent-border)',
    color: 'inherit',
  },

  '.cm-tooltip': {
    backgroundColor: 'var(--bg-elevated)',
    border: '1px solid var(--border-strong)',
    borderRadius: 'var(--r-md)',
    boxShadow: 'var(--shadow-lg)',
    overflow: 'hidden',
  },
  '.cm-tooltip-autocomplete > ul': { fontFamily: 'var(--font-mono)', maxHeight: '16em' },
  '.cm-tooltip-autocomplete > ul > li': { padding: '4px 10px', color: 'var(--text-secondary)' },
  '.cm-tooltip-autocomplete > ul > li[aria-selected]': {
    backgroundColor: 'var(--bg-active)',
    color: 'var(--text-primary)',
  },
  '.cm-completionLabel': { color: 'inherit' },
  '.cm-completionDetail': { color: 'var(--text-tertiary)', fontStyle: 'normal', marginLeft: '10px' },
  '.cm-completionIcon': { color: 'var(--accent)', opacity: 0.8, paddingRight: '14px' },
}, { dark: true })

export const chguiHighlight = syntaxHighlighting(HighlightStyle.define([
  { tag: t.keyword, color: 'var(--accent-text)', fontWeight: '500' },
  { tag: [t.string, t.special(t.string)], color: '#7ee787' },
  { tag: [t.number, t.bool, t.null], color: '#79c0ff' },
  { tag: [t.function(t.variableName), t.standard(t.variableName)], color: '#d2a8ff' },
  { tag: [t.typeName, t.className], color: '#ffa657' },
  { tag: [t.operator, t.punctuation], color: 'var(--text-secondary)' },
  { tag: t.comment, color: 'var(--text-disabled)', fontStyle: 'italic' },
  { tag: t.propertyName, color: 'var(--text-primary)' },
  { tag: t.invalid, color: 'var(--error)' },
]))
