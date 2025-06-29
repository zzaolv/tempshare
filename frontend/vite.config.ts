// frontend/vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import fs from 'fs';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  // ✨ 新增 optimizeDeps 配置
  optimizeDeps: {
    include: ['animejs'],
  },
  server: {
    https: {
      key: fs.readFileSync(path.resolve(__dirname, '../backend/key.pem')),
      cert: fs.readFileSync(path.resolve(__dirname, '../backend/cert.pem')),
    },
    proxy: { 
      '/api': {
        target: 'https://localhost:8080',
        changeOrigin: true,
        secure: false, 
      },
    }
  }
})