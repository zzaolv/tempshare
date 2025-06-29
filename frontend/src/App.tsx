// src/App.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout.tsx'; // 添加 .tsx
import UploaderPage from './pages/UploaderPage.tsx'; // 添加 .tsx
import DownloadPage from './pages/DownloadPage.tsx'; // 添加 .tsx
import ReportPage from './pages/ReportPage.tsx'; // 添加 .tsx

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<UploaderPage />} />
          <Route path="download/:accessCode" element={<DownloadPage />} />
          <Route path="report" element={<ReportPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;