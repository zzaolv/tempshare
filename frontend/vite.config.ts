// frontend/vite.config.ts
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import fs from 'fs';
import path from 'path';

export default defineConfig(({ mode }) => {
  // 加载特定模式下的 .env 文件
  const env = loadEnv(mode, process.cwd(), '');
  
  return {
    plugins: [react()],
    // 定义全局常量替换
    define: {
      // 在构建时，将代码中的 'import.meta.env.VITE_DIRECT_API_BASE_URL' 替换为环境变量的值
      // 如果环境变量未设置，则回退到空字符串 ''
      'import.meta.env.VITE_DIRECT_API_BASE_URL': JSON.stringify(env.VITE_DIRECT_API_BASE_URL || '')
    },
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
         '/data': {
          target: 'https://localhost:8080',
          changeOrigin: true,
          secure: false, 
        }
      }
    }
  }
})