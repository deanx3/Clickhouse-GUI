import { ref } from 'vue'

export const toasts = ref([])
let next = 0

export function toast(message, kind = 'success', ms = 3200) {
  const id = ++next
  toasts.value.push({ id, message, kind })
  setTimeout(() => dismiss(id), ms)
  return id
}

export const dismiss = (id) => {
  const at = toasts.value.findIndex((t) => t.id === id)
  if (at >= 0) toasts.value.splice(at, 1)
}
