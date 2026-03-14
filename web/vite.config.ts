import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const backendUrl = process.env.KVELMO_BACKEND_URL || 'http://localhost:6337'

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  build: {
    rolldownOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) {
            if (id.includes('react-dom') || id.includes('/react/') || id.includes('zustand')) {
              return 'vendor'
            }
            if (id.includes('react-diff-viewer-continued')) {
              return 'diff'
            }
          }
        }
      }
    }
  },
  server: {
    port: 5173,
    strictPort: false, // Allow fallback to the next available port
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
