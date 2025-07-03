// src/pages/DownloadPage.tsx
import { useState, useEffect, useRef } from 'react';
import { useParams } from 'react-router-dom';
// ✨ 修复点: 重新导入了 LoaderCircle 用于解密按钮
import { LoaderCircle, ServerCrash, Lock, KeyRound, ShieldAlert, FileText, Download, Eye, Image as ImageIcon, Film, Music, FileQuestion } from 'lucide-react';
import streamSaver from 'streamsaver';
import { createTimeline } from 'animejs'; 
import { E2EE } from '../lib/crypto';
import { fetchFileMetadata, DIRECT_API_BASE_URL } from '../lib/api';
import type { FileMetadata } from '../lib/api';
import HumanizedCountdown from '../components/HumanizedCountdown';
import ScanStatusDisplay from '../components/ScanStatusDisplay';
import PreviewModal, { previewableExtensions } from '../components/PreviewModal';
import DownloadPageSkeleton from '../components/DownloadPageSkeleton'; 

const formatBytes = (bytes: number, decimals = 2) => {
    if (!+bytes) return '0 Bytes'
    const k = 1024
    const dm = decimals < 0 ? 0 : decimals
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`
}

const getFileTypeIcon = (filename: string) => {
    const extension = filename.split('.').pop()?.toLowerCase() || '';
    const iconProps = { className: "w-16 h-16 text-brand-cyan flex-shrink-0", strokeWidth: 1.5 };

    if (previewableExtensions.image.includes(extension)) {
        return <ImageIcon {...iconProps} />;
    }
    if (previewableExtensions.video.includes(extension)) {
        return <Film {...iconProps} />;
    }
    if (previewableExtensions.audio.includes(extension)) {
        return <Music {...iconProps} />;
    }
    if ([...previewableExtensions.pdf, ...previewableExtensions.text].includes(extension)) {
        return <FileText {...iconProps} />;
    }
    return <FileQuestion {...iconProps} />;
};


const DownloadPage = () => {
    const { accessCode } = useParams<{ accessCode: string }>();
    const [meta, setMeta] = useState<FileMetadata | null>(null);
    const [error, setError] = useState<string | null>(null);
    const [isLoading, setIsLoading] = useState(true);
    const [password, setPassword] = useState('');
    const [isDecrypting, setIsDecrypting] = useState(false);
    const [decryptionError, setDecryptionError] = useState<string | null>(null);
    const [decryptionProgress, setDecryptionProgress] = useState(0);
    const [isPreviewModalOpen, setIsPreviewModalOpen] = useState(false);
    
    const leftCardRef = useRef(null);
    const rightCardRef = useRef(null);

    const isPreviewable = (() => {
        if (!meta || meta.isEncrypted || meta.scanStatus === 'infected') {
            return false;
        }
        const fileExtension = meta.filename.split('.').pop()?.toLowerCase() || '';
        const allSupportedExtensions = [
            ...previewableExtensions.image,
            ...previewableExtensions.video,
            ...previewableExtensions.audio,
            ...previewableExtensions.pdf,
            ...previewableExtensions.office,
            ...previewableExtensions.text
        ];
        return allSupportedExtensions.includes(fileExtension);
    })();


    useEffect(() => {
        if (!accessCode) {
            setError("未提供文件码");
            setIsLoading(false);
            return;
        }
        const loadMetadata = async () => {
            setIsLoading(true);
            try {
                const metadata = await fetchFileMetadata(accessCode);
                setMeta(metadata);
            } catch (err: any) {
                setError(err.message);
            } finally {
                setIsLoading(false);
            }
        };
        loadMetadata();
    }, [accessCode]);

    useEffect(() => {
        if (!isLoading && meta && leftCardRef.current && rightCardRef.current) {
            const tl = createTimeline();
            tl.add(
                leftCardRef.current,
                { opacity: [0, 1], translateX: [-50, 0], rotateZ: [-5, 0], duration: 800, easing: 'easeOutExpo' }
            ).add(
                rightCardRef.current,
                { opacity: [0, 1], translateX: [50, 0], rotateZ: [5, 0], duration: 800, easing: 'easeOutExpo' },
                '-=600'
            );
        }
    }, [isLoading, meta]);


    const handleStreamDecryptAndDownload = async () => {
        if (!meta || !meta.isEncrypted || !password) return;
        setIsDecrypting(true);
        setDecryptionError(null);
        setDecryptionProgress(0);

        try {
            const salt = E2EE.base64ToBuffer(meta.encryptionSalt);
            const [key, verificationHash] = await Promise.all([
                E2EE.deriveKeyFromPassword(password, new Uint8Array(salt)),
                E2EE.createVerificationHash(password, new Uint8Array(salt))
            ]);

            const response = await fetch(`${DIRECT_API_BASE_URL}/data/${accessCode}`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ verificationHash }),
            });

            if (!response.ok) {
                if (response.status === 401 || response.status === 403) {
                    throw new Error("密码错误，请重试。");
                }
                const errorData = await response.json().catch(() => null);
                throw new Error(errorData?.message || `服务器错误: ${response.statusText}`);
            }

            if (!response.body) {
                throw new Error("服务器未返回文件流。");
            }
            
            const fileStream = streamSaver.createWriteStream(meta.filename, { size: meta.originalSizeBytes });
            const decryptionStream = E2EE.createDecryptionStream(key);
            
            let bytesProcessed = 0;
            const progressStream = new TransformStream({
                transform(chunk, controller) {
                    bytesProcessed += chunk.byteLength;
                    const progress = meta.originalSizeBytes ? Math.round((bytesProcessed / meta.originalSizeBytes) * 100) : 0;
                    setDecryptionProgress(progress);
                    controller.enqueue(chunk);
                }
            });
            
            await response.body
                .pipeThrough(decryptionStream)
                .pipeThrough(progressStream)
                .pipeTo(fileStream);

        } catch (err: any) {
            console.error("流式解密失败:", err);
            setDecryptionError(err.message || "解密失败。密码错误或文件已损坏。");
        } finally {
            setIsDecrypting(false);
        }
    };
    
    const renderContent = () => {
        if (isLoading) { 
            return <DownloadPageSkeleton />;
        }
        if (error) { return <div className="text-center p-8"><ServerCrash className="w-16 h-16 text-red-500 mx-auto mb-4" /><h2 className="text-2xl font-bold text-red-600">出错了</h2><p className="text-brand-light mt-1">{error}</p></div>; }
        if (!meta) return null;

        // ✨✨✨ 核心修复点: 将这段完整的 JSX 从文件末尾移回到了这里 ✨✨✨
        const downloadOnceNotice = meta.downloadOnce && (
            <div className="bg-brand-yellow/30 border border-brand-yellow/50 text-yellow-800 px-4 py-2 rounded-lg text-sm flex items-center gap-3 my-4">
                <ShieldAlert size={20} />
                <span>这是一个“阅后即焚”文件，下载一次后将永久销毁。</span>
            </div>
        );
        const infectedWarning = meta.scanStatus === 'infected' && (
             <div className="bg-red-500/10 border-2 border-red-500/50 text-red-700 px-4 py-3 rounded-xl flex flex-col items-center gap-3 my-4 shadow-lg shadow-red-500/10">
                <div className="flex items-center gap-3"><ShieldAlert size={24} /><span className="font-bold text-lg">警告：检测到潜在威胁！</span></div>
                <p className="text-sm">我们的扫描器在此文件中检测到威胁: <strong className="font-mono">{meta.scanResult}</strong>。强烈建议您不要下载或打开此文件。</p>
                <p className="text-xs text-red-600">下载风险由您自行承担。</p>
            </div>
        );
            
        const downloadButtonText = meta.isEncrypted ? '解密并下载' : '下载';
        const downloadAction = meta.isEncrypted ? handleStreamDecryptAndDownload : () => window.location.href = `${DIRECT_API_BASE_URL}/data/${accessCode}`;

        return (
            <div className="w-full max-w-6xl mx-auto p-4 md:p-8 grid grid-cols-1 lg:grid-cols-2 gap-8">
                <div
                    ref={leftCardRef}
                    style={{ opacity: 0 }}
                    className="w-full bg-white/70 backdrop-blur-xl border border-white/30 rounded-2xl shadow-soft-2xl p-8 flex flex-col"
                >
                    <div className="text-center">
                        <h2 className="text-3xl font-bold break-all text-brand-dark">{meta.filename}</h2>
                        <div className="mt-2 text-brand-light">
                           <HumanizedCountdown expiresAt={meta.expiresAt} />
                        </div>
                    </div>
                    
                    {downloadOnceNotice}
                    {infectedWarning && !meta.isEncrypted && <div className="mt-4">{infectedWarning}</div>}
                    
                    <div className="w-full my-6 p-4 bg-black/5 rounded-lg space-y-3">
                        <div className="flex items-center justify-between text-md">
                            <span className="flex items-center gap-2 text-brand-light"><FileText size={18}/>文件大小</span>
                            <span className="font-medium">{formatBytes(meta.isEncrypted ? meta.originalSizeBytes : meta.sizeBytes)}</span>
                        </div>
                        <div className="w-full h-px bg-black/10"></div>
                        <div className="flex items-center justify-between">
                             <ScanStatusDisplay status={meta.scanStatus} result={meta.scanResult} />
                        </div>
                    </div>

                    {meta.isEncrypted ? (
                         <div className="w-full space-y-4 mt-auto">
                            <p className="text-brand-light text-center">这是一个加密文件，请输入密码下载。</p>
                            <div className="relative">
                                <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
                                <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="请输入密码" className="w-full bg-black/5 p-3 pl-10 rounded-md border-2 border-transparent focus:border-brand-yellow focus:ring-brand-yellow transition"/>
                            </div>
                            <button onClick={downloadAction} disabled={!password || isDecrypting} className="w-full text-lg inline-flex items-center justify-center bg-brand-yellow disabled:bg-slate-300 disabled:cursor-not-allowed text-brand-dark font-bold py-3 rounded-lg transition-all hover:scale-105 hover:brightness-110 active:scale-95">
                                {isDecrypting ? <LoaderCircle className="animate-spin mr-2" /> : <Download className="w-5 h-5 mr-3" />}
                                <span>{isDecrypting ? `验证并解密... ${decryptionProgress}%` : downloadButtonText}</span>
                            </button>
                            {isDecrypting && <div className="w-full bg-slate-200 rounded-full h-2.5 mt-2 overflow-hidden"><div className="bg-brand-yellow h-2.5 rounded-full transition-all duration-300" style={{width: `${decryptionProgress}%`}}></div></div>}
                            {decryptionError && <p className="text-red-500 mt-2">{decryptionError}</p>}
                        </div>
                    ) : (
                         <button onClick={downloadAction} disabled={isDecrypting} className={`w-full text-lg inline-flex items-center justify-center font-bold py-3 rounded-lg transition-all mt-auto hover:scale-105 hover:brightness-110 active:scale-95 ${meta.scanStatus === 'infected' ? 'bg-red-500 hover:bg-red-600 text-white' : 'bg-brand-cyan hover:bg-brand-cyan text-white'}`}>
                            <Download className="w-5 h-5 mr-3" />
                            <span>{downloadButtonText}</span>
                        </button>
                    )}
                    <p className="text-xs text-slate-400 mt-6 text-center">上传于: {new Date(meta.createdAt).toLocaleString()}</p>
                </div>

                <div
                    ref={rightCardRef}
                    style={{ opacity: 0 }}
                    className="w-full bg-white/60 backdrop-blur-xl border border-white/20 rounded-2xl shadow-soft-xl p-4 flex flex-col min-h-[300px] lg:min-h-0"
                >
                    <h3 className="text-xl font-bold mb-4 text-brand-dark">文件预览</h3>
                    <div className="flex-grow flex flex-col items-center justify-center text-center p-8 bg-black/5 rounded-lg">
                        <div className="mb-4">
                           {meta.isEncrypted ? 
                                <Lock className="w-16 h-16 text-brand-yellow flex-shrink-0" strokeWidth={1.5} /> : 
                                getFileTypeIcon(meta.filename)
                           }
                        </div>
                        <h4 className="text-xl font-bold text-brand-dark break-all">{meta.filename}</h4>

                        {isPreviewable ? (
                             <button 
                                onClick={() => setIsPreviewModalOpen(true)} 
                                className="mt-8 inline-flex items-center justify-center gap-2 bg-slate-600 hover:bg-slate-700 text-white font-bold py-2 px-6 rounded-lg transition-colors hover:scale-105 active:scale-95"
                             >
                                <Eye size={18}/>
                                <span>预览</span>
                            </button>
                        ) : (
                            <p className="mt-8 text-brand-light text-sm">
                                {meta.isEncrypted ? "加密文件无法预览" : "此文件类型不支持预览"}
                            </p>
                        )}
                    </div>
                </div>
            </div>
        );
    };
    
    return (
        <>
            {isPreviewModalOpen && meta && (
                <PreviewModal
                    file={meta}
                    onClose={() => setIsPreviewModalOpen(false)}
                />
            )}
            <div className="min-h-[60vh] flex items-center justify-center">
              {renderContent()}
            </div>
        </>
    );
};

export default DownloadPage;