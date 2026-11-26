import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    proxy: {
      '/api': process.env.LATHE_API_ORIGIN ?? 'http://127.0.0.1:8080',
    },
  },
})
