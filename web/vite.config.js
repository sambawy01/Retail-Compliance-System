import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://localhost:8080',
        changeOrigin: true,
      },
      '/ws': {
        target: (process.env.VITE_WS_URL || process.env.VITE_API_URL || 'http://localhost:8080').replace(/^http/, 'ws'),
        ws: true,
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      output: {
        // Split large vendor libraries into their own cacheable chunks so the
        // initial (Login) bundle stays small and route chunks stay under 500 kB.
        manualChunks(id) {
          if (id.includes('node_modules')) {
            // React core + router: stable, shared by every authenticated route.
            if (id.includes('/react/') || id.includes('/react-dom/') || id.includes('/react-router') || id.includes('/scheduler/')) {
              return 'react-vendor'
            }
            // Recharts pulls in victory-vendor (d3) + lodash, which together
            // push the chart chunk over 500 kB. Split them into a sibling chunk
            // so neither exceeds the limit; both load lazily with Dashboard.
            if (id.includes('/victory-vendor/') || id.includes('/d3-') || id.includes('/lodash/') || id.includes('/react-smooth/') || id.includes('/recharts-scale/') || id.includes('/eventemitter3/') || id.includes('/react-is/')) {
              return 'recharts-vendor'
            }
            if (id.includes('/recharts/')) {
              return 'recharts'
            }
            if (id.includes('/lucide-react/')) {
              return 'lucide-icons'
            }
            if (id.includes('/axios/')) {
              return 'axios'
            }
          }
        },
      },
    },
    // 450 kB keeps the warning quiet for legitimate page chunks that bundle
    // app code + small libs, while still flagging genuinely oversized bundles.
    chunkSizeWarningLimit: 450,
  },
})