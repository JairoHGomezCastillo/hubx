import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // El MFE nunca llama directo a la API: /api se reenvía al gateway,
      // evitando CORS en desarrollo y respetando la arquitectura de tres piezas.
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
