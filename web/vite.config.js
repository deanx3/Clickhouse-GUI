import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Keep the dev proxy pointed at whatever port the Go server is on.
const api = process.env.CHGUI_API || 'http://localhost:8090'

export default defineConfig({
  plugins: [vue()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: { proxy: { '/api': api } },
})
