// Shared mousedown -> drag -> release plumbing for the panel splitters.
// Returns a handler; the caller captures whatever start value it needs.
export function startDrag(e, { onMove, cursor = 'col-resize' }) {
  e.preventDefault()
  const from = { x: e.clientX, y: e.clientY }

  const move = (ev) => onMove(ev.clientX - from.x, ev.clientY - from.y)
  const up = () => {
    document.removeEventListener('mousemove', move)
    document.removeEventListener('mouseup', up)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }

  document.body.style.cursor = cursor
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', move)
  document.addEventListener('mouseup', up)
}

export const clamp = (v, lo, hi) => Math.min(hi, Math.max(lo, v))
