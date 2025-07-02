// frontend/vite.config.ts
import { defineConfig, loadEnv } from 'vite';
import type { UserConfig } from 'vite'; 
import react from '@vitejs/plugin-react';
import fs from 'fs';
import path from 'path';

export default defineConfig(({ mode }) => {
  // 加载环境变量
  const env = loadEnv(mode, process.cwd(), '');
  
  const config: UserConfig = {
    plugins: [react()],
    // define 用于在客户端代码中替换环境变量，确保生产构建时 API URL 正确
    define: {
      'import.meta.env.VITE_DIRECT_API_BASE_URL': JSON.stringify(env.VITE_DIRECT_API_BASE_URL || '')
    },
    server: {
      // HTTPS is still needed for local dev to match the backend protocol
      // ✨✨✨ 修复点: 添加 Vite proxy 配置 ✨✨✨
      proxy: {
        // 将所有 /api/v1 的请求代理到后端服务器
        '/api/v1': {
          target: 'https://localhost:8080', // 后端服务器地址
          changeOrigin: true, // 需要更改源，以避免 CORS 问题
          secure: false, // 因为我们使用的是自签名证书，所以需要这个
        },
      }
    }
  };

  // 仅在开发模式下为 Vite 开发服务器启用 HTTPS
  if (mode === 'development') {
    if (!config.server) {
      config.server = {};
    }
    try {
      // 尝试从后端目录读取证书文件
      const keyPath = path.resolve(__dirname, '../backend/key.pem');
      const certPath = path.resolve(__dirname, '../backend/cert.pem');
      
      if (fs.existsSync(keyPath) && fs.existsSync(certPath)) {
        config.server.https = {
          key: fs.readFileSync(keyPath),
          cert: fs.readFileSync(certPath),
        };
        console.log('Vite dev server is running in HTTPS mode.');
      } else {
          throw new Error('SSL certificates not found.');
      }
    } catch (e) {
      console.warn('\nCould not find SSL certificates for Vite dev server. It will run in HTTP mode.');
      console.warn('This will likely cause issues when connecting to the HTTPS backend.');
      console.warn('➡️ To fix this, run `mkcert -install && mkcert localhost` in the `backend` directory.\n');
      // 清除 https 配置，使其回退到 http
      if (config.server) {
        delete config.server.https;
      }
    }
  }

  return config;
});