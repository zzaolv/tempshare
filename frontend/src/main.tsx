// src/main.tsx

import React from 'react'
import ReactDOM from 'react-dom/client'
import { createBrowserRouter, RouterProvider } from 'react-router-dom'

// ✨ 确保这一行存在，它会加载我们所有的 CSS 样式！
import './index.css' 

import Layout from './components/Layout.tsx'
import UploaderPage from './pages/UploaderPage.tsx'
import DownloadPage from './pages/DownloadPage.tsx'
import ReportPage from './pages/ReportPage.tsx'

const router = createBrowserRouter([
  {
    path: '/',
    element: <Layout />,
    children: [
      {
        index: true,
        element: <UploaderPage />,
      },
      {
        path: 'download/:accessCode',
        element: <DownloadPage />,
      },
      {
        path: 'report',
        element: <ReportPage />,
      },
      // 你可以添加一个 404 页面
      // {
      //   path: '*',
      //   element: <NotFoundPage />,
      // }
    ],
  },
])

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <RouterProvider router={router} />
  </React.StrictMode>,
)