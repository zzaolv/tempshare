// frontend/vite.config.ts
import { defineConfig, loadEnv } from 'vite';
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
      // ✨✨✨ 核心修复点 ✨✨✨
      // 将代理目标从 http 修改为 https，以匹配我们刚刚修改的 Go 后端。
      proxy: { 
        '/api': {
          target: 'https://localhost:8080',
          changeOrigin: true,
          secure: false, // `secure: false` 允许代理到自签名的证书，这在本地开发中很关键
        },
        '/data': {
          target: 'https://localhost:8080',
          changeOrigin: true,
          secure: false, 
        }
      }
    }
  };

  // 保持本地开发服务器的 HTTPS 配置不变，这是正确的
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
      console.warn('找不到用于开发服务器的SSL证书。将以 HTTP 模式运行。');
      console.warn('要启用 HTTPS 开发, 请在 backend 文件夹下运行 `mkcert -install && mkcert localhost`。');
    }
  }

  return config;
});