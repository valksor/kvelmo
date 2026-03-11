import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const backendUrl = process.env.KVELMO_BACKEND_URL || 'http://localhost:6337'

export default defineConfig({
  base: './',
  plugins: [react(), tailwindcss()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom', 'zustand'],
          diff: ['react-diff-viewer-continued'],
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
