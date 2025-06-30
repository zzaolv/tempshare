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
      port: 5173, 
      strictPort: true,
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
      const keyPath = path.resolve(__dirname, '../backend/key.pem');
      const certPath = path.resolve(__dirname, '../backend/cert.pem');
      
      if (fs.existsSync(keyPath) && fs.existsSync(certPath)) {
        config.server.https = {
          key: fs.readFileSync(keyPath),
          cert: fs.readFileSync(certPath),
        };
        console.log("成功加载 SSL 证书，Vite 开发服务器将以 HTTPS 模式运行。");
      } else {
        throw new Error("证书文件未找到。");
      }
    } catch (e) {
      // ✨✨✨ 核心修复点 ✨✨✨
      // 我们在这里对 'e' 的类型进行检查，确保它是一个 Error 对象。
      let errorMessage = "一个未知的错误发生了。";
      if (e instanceof Error) {
        errorMessage = e.message;
      }
      
      console.warn('警告：无法加载 SSL 证书。Vite 将以 HTTP 模式运行，这可能导致功能异常。');
      console.warn(`错误信息: ${errorMessage}`);
      console.warn('要解决此问题，请进入项目的 `backend` 目录，然后运行以下命令：');
      console.warn('`mkcert -install && mkcert localhost`');
    }
  }

  return config;
});