import path from 'path'
import fs from 'fs'
import { execSync } from 'child_process'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

const packageJson = JSON.parse(fs.readFileSync(new URL('./package.json', import.meta.url), 'utf-8'))

function getVersion() {
  if (process.env.VITE_APP_VERSION) return process.env.VITE_APP_VERSION
  if (packageJson?.version) return packageJson.version
  try {
    return execSync('git describe --tags --abbrev=0', { encoding: 'utf-8' }).trim()
  } catch {
    return 'dev'
  }
}

export default defineConfig({
  plugins: [react(), tailwindcss()],
  base: '/admin/',
  define: {
    __APP_VERSION__: JSON.stringify(getVersion()),
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1000
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080'
    }
  }
})
