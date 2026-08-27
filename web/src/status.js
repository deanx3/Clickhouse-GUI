import { ref } from 'vue'

// Connection state is inferred from real traffic rather than a heartbeat: the
// only thing worth reporting is whether the queries this app makes are working.
export const status = ref('connected') // connected | connecting | error
export const lastError = ref('')

export const beginRequest = () => { if (status.value !== 'error') status.value = 'connecting' }
export const requestOk = () => { status.value = 'connected'; lastError.value = '' }
export const requestFailed = (message, transport) => {
  // A rejected SQL statement means the server is fine; only a transport
  // failure means the connection is down.
  if (transport) { status.value = 'error'; lastError.value = message }
  else status.value = 'connected'
}
