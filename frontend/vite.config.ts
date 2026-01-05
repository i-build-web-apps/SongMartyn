import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { execSync } from 'child_process'

// Get git info for build ID
function getBuildInfo() {
  try {
    const commitHash = execSync('git rev-parse --short HEAD').toString().trim()
    const isDirty = execSync('git status --porcelain').toString().trim() !== ''
    const buildTime = new Date().toISOString()
    return {
      hash: commitHash + (isDirty ? '-dirty' : ''),
      time: buildTime,
    }
  } catch {
    return {
      hash: 'unknown',
      time: new Date().toISOString(),
    }
  }
}

const buildInfo = getBuildInfo()

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  define: {
    __BUILD_HASH__: JSON.stringify(buildInfo.hash),
    __BUILD_TIME__: JSON.stringify(buildInfo.time),
  },
  server: {
    host: true, // Allow external connections for mobile testing
    port: 5173,
  },
})
