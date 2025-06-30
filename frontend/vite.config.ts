// frontend/vite.config.ts
import { defineConfig, loadEnv } from 'vite';
// ✨✨✨ 修复点: 使用 'import type' 单独导入类型 ✨✨✨
import type { UserConfig } from 'vite'; 
import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  
  const config: UserConfig = {
    plugins: [react()],
    define: {
      'import.meta.env.VITE_DIRECT_API_BASE_URL': JSON.stringify(env.VITE_DIRECT_API_BASE_URL || '')
    },
    server: {
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
  };

  if (mode === 'development') {
    if (!config.server) {
      config.server = {};
    }
    try {
      config.server.https = {
        key: fs.readFileSync(path.resolve(__dirname, '../backend/key.pem')),
        cert: fs.readFileSync(path.resolve(__dirname, '../backend/cert.pem')),
      };
    } catch (e) {
      console.warn('Could not find SSL certificates for dev server. Running in HTTP mode.');
      console.warn('To enable HTTPS for dev, run `mkcert -install && mkcert localhost` in the backend folder.');
    }
  }

  return config;
});