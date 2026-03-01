import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const backendUrl = process.env.KVELMO_BACKEND_URL || 'http://localhost:6337'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    strictPort: false, // Allow fallback to next available port
    proxy: {
      '/api': {
        target: backendUrl,
        changeOrigin: true
      },
      '/ws': {
        target: backendUrl,
        changeOrigin: true,
        ws: true
      }
    }
  }
})
