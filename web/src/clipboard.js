import { toast } from './toast'

// navigator.clipboard needs a secure context, which a plain-http internal
// deployment is not, so fall back to the old selection trick.
export async function copy(text, message = 'Copied') {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const ta = document.createElement('textarea')
      ta.value = text
      ta.style.cssText = 'position:fixed;opacity:0'
      document.body.append(ta)
      ta.select()
      document.execCommand('copy')
      ta.remove()
    }
    toast(message)
  } catch {
    toast('Could not copy to the clipboard', 'error')
  }
}
