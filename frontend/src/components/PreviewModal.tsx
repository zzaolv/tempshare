// frontend/src/components/PreviewModal.tsx
import { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, FileText, Download, LoaderCircle } from 'lucide-react';
import hljs from 'highlight.js/lib/core';
import 'highlight.js/styles/atom-one-dark.css';

// 按需加载高亮语言
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import xml from 'highlight.js/lib/languages/xml';
import css from 'highlight.js/lib/languages/css';
import json from 'highlight.js/lib/languages/json';
import markdown from 'highlight.js/lib/languages/markdown';
import plaintext from 'highlight.js/lib/languages/plaintext';

import { DIRECT_API_BASE_URL } from '../lib/api.ts';
import type { FileMetadata } from '../lib/api.ts';

// 注册语言
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('xml', xml);
hljs.registerLanguage('css', css);
hljs.registerLanguage('json', json);
hljs.registerLanguage('markdown', markdown);
hljs.registerLanguage('plaintext', plaintext);

export const previewableExtensions = {
    image: ['png', 'jpg', 'jpeg', 'gif', 'webp', 'svg'],
    video: ['mp4', 'webm', 'ogg'],
    audio: ['mp3', 'wav'],
    pdf: ['pdf'],
    office: ['ppt', 'pptx', 'doc', 'docx', 'xls', 'xlsx'],
    text: ['txt', 'md', 'json', 'js', 'ts', 'css', 'html', 'xml', 'log', 'go', 'py', 'java', 'c', 'cpp', 'cs', 'sh', 'bat']
};

const backdropVariants = {
    hidden: { opacity: 0 },
    visible: { opacity: 1, transition: { duration: 0.2 } },
};

const modalVariants = {
    hidden: { scale: 0.9, opacity: 0 },
    visible: {
        scale: 1,
        opacity: 1,
        transition: { type: 'spring', stiffness: 400, damping: 35 } as const
    },
    exit: {
        scale: 0.9,
        opacity: 0,
        transition: { duration: 0.2 } as const
    }
};

const PreviewModal = ({ file, onClose }: { file: FileMetadata, onClose: () => void }) => {
    const [previewContent, setPreviewContent] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const directDownloadUrl = `${DIRECT_API_BASE_URL}/data/${file.accessCode}`;
    const proxiedPreviewUrl = `/api/v1/preview/${file.accessCode}`;
    const dataUriPreviewUrl = `/api/v1/preview/data-uri/${file.accessCode}`;

    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            if (e.key === 'Escape') onClose();
        };
        window.addEventListener('keydown', handleKeyDown);

        const fileExtension = file.filename.split('.').pop()?.toLowerCase() || '';

        const fetchContentForPreview = async () => {
            setIsLoading(true);
            setError(null);
            
            // 检查是否是需要特殊处理（通过 data-uri API）的类型
            const needsDataUri = previewableExtensions.pdf.includes(fileExtension) || previewableExtensions.text.includes(fileExtension);
            
            // 如果不需要特殊处理，则直接返回，不发起 fetch 请求
            if (!needsDataUri) {
                setIsLoading(false);
                return;
            }

            try {
                const response = await fetch(dataUriPreviewUrl);
                if (!response.ok) {
                    const errData = await response.json().catch(() => ({ message: '无法获取预览内容' }));
                    throw new Error(errData.message);
                }

                const data = await response.json();

                // 如果是文本，进行解码和高亮
                if (previewableExtensions.text.includes(fileExtension)) {
                    const base64Content = data.dataUri.split(',')[1];
                    const textContent = atob(base64Content);
                    const highlighted = hljs.highlightAuto(textContent).value;
                    setPreviewContent(highlighted);
                } else {
                    // PDF 直接使用后端返回的完整 dataUri
                    setPreviewContent(data.dataUri);
                }

            } catch (err: any) {
                console.error("预览加载失败:", err);
                setError(err.message || '加载预览时发生未知错误。');
            } finally {
                setIsLoading(false);
            }
        };
        
        fetchContentForPreview();

        return () => window.removeEventListener('keydown', handleKeyDown);
    }, [file.accessCode, file.filename, dataUriPreviewUrl, onClose]);

    const fileExtension = file.filename.split('.').pop()?.toLowerCase() || '';

    const renderContent = () => {
        if (isLoading) {
            return <div className="flex items-center justify-center h-full text-brand-dark"><LoaderCircle className="animate-spin mr-2" /> 正在加载预览...</div>
        }
        if (error) {
            return <div className="flex flex-col items-center justify-center h-full text-red-600 p-4 text-center"><X className="w-12 h-12 mb-4"/>预览加载失败: <br/>{error}</div>
        }

        if (previewableExtensions.image.includes(fileExtension)) {
            return <div className="w-full h-full flex items-center justify-center"><img src={proxiedPreviewUrl} alt={file.filename} className="max-w-full max-h-full object-contain" /></div>;
        }
        if (previewableExtensions.video.includes(fileExtension)) {
            return <video src={proxiedPreviewUrl} controls autoPlay className="w-full h-full bg-black" />;
        }
        if (previewableExtensions.audio.includes(fileExtension)) {
            return (
                <div className="w-full h-full flex flex-col items-center justify-center p-8 bg-gray-50">
                     <h3 className="text-xl font-bold mb-4 break-all text-brand-dark">{file.filename}</h3>
                     <audio src={proxiedPreviewUrl} controls autoPlay className="w-full max-w-md" />
                </div>
            );
        }
        if (previewableExtensions.office.includes(fileExtension)) {
            const officeViewerUrl = `https://view.officeapps.live.com/op/embed.aspx?src=${encodeURIComponent(directDownloadUrl)}`;
            return <iframe src={officeViewerUrl} title={file.filename} className="w-full h-full border-0" />;
        }
        
        if (previewableExtensions.pdf.includes(fileExtension)) {
            return previewContent ? <iframe src={previewContent} title={file.filename} className="w-full h-full border-0 bg-white" /> : null;
        }
        if (previewableExtensions.text.includes(fileExtension)) {
            return <pre className="w-full h-full overflow-auto p-4 bg-[#282c34]"><code className="hljs" dangerouslySetInnerHTML={{ __html: previewContent }}></code></pre>;
        }

        return (
            <div className="w-full h-full flex flex-col items-center justify-center p-8 bg-gray-50 text-center">
                <FileText className="w-16 h-16 mb-4 text-brand-cyan"/>
                <h3 className="text-2xl font-bold mb-2 break-all text-brand-dark">{file.filename}</h3>
                <p className="text-brand-light">此文件类型不支持在线预览</p>
                <a 
                    href={directDownloadUrl} 
                    download 
                    className="mt-6 inline-flex items-center gap-2 bg-brand-cyan hover:brightness-110 text-white font-bold py-2 px-4 rounded-lg transition-all"
                >
                    <Download size={18} />
                    下载文件
                </a>
            </div>
        );
    }

    return (
        <AnimatePresence>
            <motion.div
                className="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50 p-4 sm:p-6 md:p-8"
                variants={backdropVariants}
                initial="hidden"
                animate="visible"
                exit="hidden"
                onClick={onClose}
            >
                <motion.div
                    className="relative w-full h-full max-w-5xl max-h-[90vh] bg-white rounded-xl shadow-2xl flex flex-col overflow-hidden"
                    variants={modalVariants}
                    initial="hidden"
                    animate="visible"
                    exit="exit"
                    onClick={e => e.stopPropagation()}
                >
                    {renderContent()}

                    <motion.button 
                        onClick={onClose} 
                        whileHover={{ scale: 1.1, rotate: 90 }}
                        whileTap={{ scale: 0.9 }}
                        className="absolute top-3 right-3 p-2 rounded-full bg-black/10 hover:bg-black/20 text-brand-dark transition-all duration-200"
                        aria-label="关闭预览"
                    >
                        <X size={20} />
                    </motion.button>
                </motion.div>
            </motion.div>
        </AnimatePresence>
    );
};

export default PreviewModal;