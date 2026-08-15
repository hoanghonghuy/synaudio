import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    proxy: {
      '/health': 'http://localhost:8080',
      '/ready': 'http://localhost:8080',
      '/api': 'http://localhost:8080',
    },
  },
})
