import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'
import path from 'node:path'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    // Backend serves static files from ./web/dist (see src/api/routes/routes.go).
    // Output build artifacts there to avoid manual copying and stale UI.
    outDir: path.resolve(__dirname, '../../web/dist'),
    emptyOutDir: true
  },
  server: {
    host: '0.0.0.0',
    port: 3012,
    proxy: {
      '/api': {
        target: 'http://localhost:5678',
        changeOrigin: true
      }
    }
  }
})
