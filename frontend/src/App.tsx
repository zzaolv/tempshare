// src/App.tsx
import { useState, useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { LoaderCircle } from 'lucide-react';

import Layout from './components/Layout';
import UploaderPage from './pages/UploaderPage';
import DownloadPage from './pages/DownloadPage';
import ReportPage from './pages/ReportPage';
import SetupPage from './pages/SetupPage.tsx'; // 新增
import { checkInitStatus } from './lib/api';

function App() {
  const [needsInit, setNeedsInit] = useState<boolean | null>(null);

  useEffect(() => {
    // 组件挂载时检查初始化状态
    const checkStatus = async () => {
      const status = await checkInitStatus();
      setNeedsInit(status.needsInit);
    };
    checkStatus();
  }, []);

  // 正在检查状态时，显示加载中...
  if (needsInit === null) {
    return (
      <div className="w-screen h-screen flex flex-col items-center justify-center bg-gray-100 text-slate-600 gap-4">
        <LoaderCircle className="w-12 h-12 animate-spin text-brand-cyan" />
        <p>正在连接到服务...</p>
      </div>
    );
  }

  return (
    <BrowserRouter>
      <Routes>
        {needsInit ? (
          // 如果需要初始化，所有路由都指向 SetupPage
          <>
            <Route path="/setup" element={<SetupPage />} />
            <Route path="*" element={<Navigate to="/setup" replace />} />
          </>
        ) : (
          // 正常模式下的路由
          <Route path="/" element={<Layout />}>
            <Route index element={<UploaderPage />} />
            <Route path="download/:accessCode" element={<DownloadPage />} />
            <Route path="report" element={<ReportPage />} />
            {/* 可以添加一个 404 页面 */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        )}
      </Routes>
    </BrowserRouter>
  );
}

export default App;