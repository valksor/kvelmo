import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/',
  build: {
    outDir: '../internal/server/static/app',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks: {
          // Core React - very stable, changes rarely
          'vendor-react': ['react', 'react-dom'],
          // Routing - stable, changes with major updates
          'vendor-router': ['react-router-dom'],
          // Data layer - moderately stable
          'vendor-data': [
            '@tanstack/react-query',
            'zustand',
            'zod',
            '@hookform/resolvers',
            'react-hook-form',
          ],
          // UI utilities - updated more frequently
          'vendor-ui': ['lucide-react', 'date-fns'],
        },
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
