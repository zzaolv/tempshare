// src/pages/UploaderPage.tsx
import { useState, useCallback, useRef, useMemo, useEffect } from 'react';
import type { FormEvent, ChangeEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useDropzone } from 'react-dropzone';
import { motion, AnimatePresence } from 'framer-motion';
import { FileText, X, Copy, KeyRound, Plus, Trash2, Repeat, Check, QrCode, UploadCloud, Clock, Feather, FolderPlus } from 'lucide-react';
import JSZip from 'jszip';
import { QRCodeSVG } from 'qrcode.react';
import { E2EE } from '../lib/crypto.ts';
import type { ShareDetails } from '../lib/api';
import { DIRECT_API_BASE_URL } from '../lib/api.ts';
import UploadProgressCircle from '../components/UploadProgressCircle.tsx';
import HumanizedCountdown from '../components/HumanizedCountdown.tsx';
import CodeInput from '../components/CodeInput';
import type { CodeInputHandle } from '../components/CodeInput';
import { useUploaderStore } from '../store/uploaderStore.ts';
import useMediaQuery from '../hooks/useMediaQuery';
import useOnClickOutside from '../hooks/useOnClickOutside';

// (辅助函数 createProgressStream 和 formatBytes 保持不变)
function createProgressStream(totalSize: number, onProgress: (progress: number) => void, onSpeedUpdate: (speed: string) => void): TransformStream<Uint8Array, Uint8Array> {
    let bytesSent = 0;
    let lastTime = Date.now();
    let lastBytesSent = 0;
    const intervalId = setInterval(() => {
        const now = Date.now();
        const timeDiff = (now - lastTime) / 1000;
        if (timeDiff > 0) {
            const bytesDiff = bytesSent - lastBytesSent;
            const speed = bytesDiff / timeDiff;
            onSpeedUpdate(`${formatBytes(speed, 1)}/s`);
        }
        lastTime = now;
        lastBytesSent = bytesSent;
    }, 1000);
    return new TransformStream({
        transform(chunk, controller) {
            bytesSent += chunk.length;
            const progress = totalSize > 0 ? Math.round((bytesSent / totalSize) * 100) : 0;
            onProgress(progress);
            controller.enqueue(chunk);
        },
        flush() {
            clearInterval(intervalId);
            onSpeedUpdate("完成");
        }
    });
}
const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const dm = decimals < 0 ? 0 : decimals;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
};

// ✨ InteractiveUploader 现在接收 isSubmittingCode prop
const InteractiveUploader = ({ onAddFileClick, onAddFolderClick, onCodeSubmit, onCodeComplete, isSubmittingCode }: {
    onAddFileClick: () => void;
    onAddFolderClick: () => void;
    onCodeSubmit: (e: FormEvent) => void;
    onCodeComplete: (value: string) => void;
    isSubmittingCode: boolean;
}) => {
    const { isInputMode, code, setIsInputMode, setCode } = useUploaderStore();
    const codeInputRef = useRef<CodeInputHandle>(null);
    const uploaderRef = useRef<HTMLDivElement>(null);
    const [hoverTarget, setHoverTarget] = useState<'none' | 'code' | 'upload'>('none');
    const [mobileUploadExpanded, setMobileUploadExpanded] = useState(false);
    const isMobile = useMediaQuery('(max-width: 768px)');

    useOnClickOutside(uploaderRef, () => {
        if (isMobile && mobileUploadExpanded) {
            setMobileUploadExpanded(false);
        }
    });

    const handleSwitchToInput = () => {
        setIsInputMode(true);
        setHoverTarget('none');
        setTimeout(() => codeInputRef.current?.focus(), 100);
    };

    const handleUploadClick = () => {
        if (isMobile) {
            setMobileUploadExpanded(!mobileUploadExpanded);
        } else {
            onAddFileClick();
        }
    };

    const handleSwitchToUpload = () => setIsInputMode(false);

    const spring = { type: "spring", stiffness: 400, damping: 30 } as const;
    const cardBgClass = "bg-white/70 backdrop-blur-xl border border-white/30";

    return (
        <div ref={uploaderRef} className="relative w-full max-w-[450px] h-20 flex justify-center items-center">
            <AnimatePresence mode="wait">
                {isInputMode ? (
                    <motion.form
                        key="input-form"
                        layoutId="uploader-container"
                        onSubmit={onCodeSubmit}
                        className={`w-full max-w-[420px] h-16 ${cardBgClass} rounded-full shadow-soft-xl flex items-center justify-between px-4`}
                        initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
                        transition={{ ...spring, duration: 0.3 }}
                    >
                        <motion.div className="flex-grow" initial={{ opacity: 0 }} animate={{ opacity: 1, transition: { delay: 0.15 } }} exit={{ opacity: 0, transition: { duration: 0.1 } }}>
                            <CodeInput 
                                ref={codeInputRef} 
                                value={code} 
                                onChange={setCode} 
                                onComplete={onCodeComplete} 
                                isSubmitting={isSubmittingCode}
                            />
                        </motion.div>
                        <motion.button type="button" whileHover={{ scale: 1.15, rotate: 90 }} whileTap={{ scale: 0.9 }} onClick={handleSwitchToUpload} className="p-2 rounded-full text-slate-500 hover:bg-black/10 flex-shrink-0" aria-label="关闭" initial={{ opacity: 0 }} animate={{ opacity: 1, transition: { delay: 0.15 } }} exit={{ opacity: 0, transition: { duration: 0.1 } }}>
                            <X size={24} />
                        </motion.button>
                    </motion.form>
                ) : (
                    <motion.div
                        key="button-group"
                        layoutId="uploader-container"
                        initial={{ opacity: 0, width: 280 }}
                        animate={{ width: isMobile ? (mobileUploadExpanded ? 360 : 280) : (hoverTarget === 'code' ? 360 : 280), opacity: 1 }}
                        transition={spring}
                        className={`relative h-16 ${cardBgClass} rounded-full shadow-soft-xl flex items-center`}
                        onMouseLeave={() => !isMobile && setHoverTarget('none')}
                    >
                        <div
                            className="w-[120px] h-full flex-shrink-0 cursor-pointer flex items-center justify-center"
                            onMouseEnter={() => !isMobile && setHoverTarget('code')}
                            onClick={handleSwitchToInput}
                        >
                            <div className="relative w-full h-full flex items-center justify-center">
                                <motion.span className="absolute flex items-center justify-center gap-1 whitespace-nowrap" animate={{ opacity: hoverTarget === 'code' ? 0 : 1 }}>
                                    <KeyRound className="text-brand-cyan" size={20} />
                                    <span className="text-lg font-semibold text-brand-cyan">取件</span>
                                </motion.span>
                                <motion.span className="absolute w-full h-full flex items-center justify-center gap-2 whitespace-nowrap pl-4" animate={{ opacity: hoverTarget === 'code' ? 1 : 0 }}>
                                    <KeyRound className="text-brand-cyan" size={22} />
                                    <span className="text-xl font-semibold text-brand-cyan">取件口令</span>
                                </motion.span>
                            </div>
                        </div>

                        <motion.div
                            className="w-px h-8 bg-slate-300 pointer-events-none"
                            animate={{ opacity: hoverTarget !== 'none' ? 0 : 1 }}
                        />
                        
                        <div
                            className="relative flex-1 h-full cursor-pointer flex items-center justify-center"
                            onClick={handleUploadClick}
                            onMouseEnter={() => !isMobile && setHoverTarget('upload')}
                        >
                            <motion.div
                                className="flex items-center justify-center gap-2 p-3 rounded-full"
                                animate={{ opacity: (isMobile && mobileUploadExpanded) || (!isMobile && hoverTarget === 'upload') ? 0 : 1 }}
                            >
                                <Plus className="text-brand-dark" size={24} />
                                <span className="text-xl font-semibold text-brand-dark">上传</span>
                            </motion.div>
                        </div>

                        <AnimatePresence>
                            {((!isMobile && hoverTarget === 'upload') || (isMobile && mobileUploadExpanded)) && (
                                <motion.div
                                    className="absolute inset-0 bg-brand-cyan/95 rounded-full flex z-10"
                                    initial={{ clipPath: 'inset(0 100% 0 0)' }}
                                    animate={{ clipPath: 'inset(0 0% 0 0)' }}
                                    exit={{ clipPath: 'inset(0 100% 0 0)' }}
                                    transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
                                >
                                    <button onClick={onAddFileClick} className="flex-1 h-full flex flex-col items-center justify-center text-white hover:bg-black/10 transition-colors rounded-l-full">
                                        <Plus size={20} /><span className="font-medium text-xs mt-1">上传文件</span>
                                    </button>
                                    <div className="w-px h-6 bg-white/30 self-center pointer-events-none"></div>
                                    <button onClick={onAddFolderClick} className="flex-1 h-full flex flex-col items-center justify-center text-white hover:bg-black/10 transition-colors rounded-r-full">
                                        <FolderPlus size={20} /><span className="font-medium text-xs mt-1">上传文件夹</span>
                                    </button>
                                </motion.div>
                            )}
                        </AnimatePresence>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
};


const UploadSettingsPanel = ({
    files, onAddFiles, onRemoveFile, onStartUpload, onCancel,
    usePassword, setUsePassword, password, setPassword, expiry, setExpiry, downloadOnce, setDownloadOnce, isProcessing
}: { 
    files: File[], onAddFiles: () => void, onRemoveFile: (file: File) => void, onStartUpload: () => void, onCancel: () => void,
    usePassword: boolean, setUsePassword: (val: boolean) => void, password: string, setPassword: (val: string) => void,
    expiry: number, setExpiry: (val: number) => void,
    downloadOnce: boolean, setDownloadOnce: (val: boolean) => void,
    isProcessing: boolean
}) => {
    const totalSize = useMemo(() => files.reduce((acc, file) => acc + file.size, 0), [files]);
    const [tosAccepted, setTosAccepted] = useState(false);

    const ToggleSwitch = ({ enabled, setEnabled, accentClass = 'bg-brand-cyan' }: { enabled: boolean, setEnabled: (enabled: boolean) => void, accentClass?: string }) => (
        <button type="button" role="switch" aria-checked={enabled} onClick={() => setEnabled(!enabled)} className={`${enabled ? accentClass : 'bg-slate-200'} relative inline-flex h-6 w-11 items-center rounded-full transition-colors`}>
            <motion.span layout className={`${enabled ? 'translate-x-6' : 'translate-x-1'} inline-block h-4 w-4 transform rounded-full bg-white shadow-sm transition-transform`} />
        </button>
    );

    return (
        <motion.div initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }} exit={{ opacity: 0, scale: 0.95 }} transition={{ duration: 0.2 }}
            className="w-full max-w-md mx-auto bg-white/70 backdrop-blur-xl border border-white/30 rounded-2xl shadow-soft-2xl text-brand-dark p-6 flex flex-col gap-4"
        >
            <div className="flex justify-between items-center">
                <h2 className="text-2xl font-bold">文件传输</h2>
                <button onClick={onAddFiles} className="w-10 h-10 bg-brand-yellow rounded-full flex items-center justify-center text-brand-dark hover:brightness-105 transition-all shadow-sm">
                    <Plus size={24} />
                </button>
            </div>
            <div className="space-y-2 max-h-40 overflow-y-auto pr-2">
                {files.map((file, index) => (
                    <div key={`${file.name}-${index}`} className="flex items-center gap-3 p-2 rounded-lg bg-black/5">
                        {file.name.toLowerCase().endsWith('.zip') ? 
                          <FolderPlus className="text-brand-light flex-shrink-0" /> :
                          <FileText className="text-brand-light flex-shrink-0" />
                        }
                        <div className="flex-grow truncate">
                            <p className="font-medium truncate">{file.name}</p>
                            <p className="text-sm text-brand-light">{formatBytes(file.size)}</p>
                        </div>
                        <button onClick={() => onRemoveFile(file)} className="p-1 text-slate-400 hover:text-red-500"><Trash2 size={16}/></button>
                    </div>
                ))}
            </div>
            <p className="text-right font-medium text-sm text-brand-light">共 {files.length} 个文件 / {formatBytes(totalSize)}</p>
            <div className="border-t border-black/10 my-2"></div>
            
            <div className="space-y-4">
                 <div className="flex justify-between items-center">
                    <label className="font-semibold flex items-center gap-2">加密传输</label>
                    <ToggleSwitch enabled={usePassword} setEnabled={setUsePassword} accentClass="bg-brand-yellow" />
                </div>
                {usePassword && (
                    <motion.div initial={{opacity:0, height: 0}} animate={{opacity: 1, height: 'auto'}} className="overflow-hidden">
                        <div className="relative">
                            <KeyRound className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" size={20} />
                            <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="请输入密码" className="w-full bg-black/5 p-2 pl-10 rounded-md border-slate-200 focus:border-brand-yellow focus:ring-brand-yellow transition"/>
                        </div>
                    </motion.div>
                )}
                 <div className="flex justify-between items-center">
                    <label className="font-semibold flex items-center gap-2">阅后即焚</label>
                    <ToggleSwitch enabled={downloadOnce} setEnabled={setDownloadOnce} />
                </div>
                <div className="flex justify-between items-center">
                    <label className="font-semibold">有效期</label>
                    <select value={expiry} onChange={e => setExpiry(Number(e.target.value))} className="bg-black/5 border-none rounded-md font-semibold focus:ring-2 focus:ring-brand-cyan">
                        <option value={180}>3分钟</option>
                        <option value={600}>10分钟</option>
                        <option value={3600}>1小时</option>
                        <option value={86400}>24小时</option>
                        <option value={604800}>7天</option>
                    </select>
                </div>
            </div>
            <div className="flex items-start gap-3 mt-4">
                <div 
                    onClick={() => setTosAccepted(!tosAccepted)}
                    className={`h-4 w-4 mt-1 rounded border-2 cursor-pointer flex items-center justify-center transition-all duration-200 flex-shrink-0 ${
                        tosAccepted 
                            ? 'bg-brand-cyan border-brand-cyan' 
                            : 'bg-white border-slate-300 hover:border-slate-400'
                    }`}
                >
                    {tosAccepted && <Check size={10} className="text-white" />}
                </div>
                <label className="text-xs text-brand-light cursor-pointer select-none" onClick={() => setTosAccepted(!tosAccepted)}>
                    我已阅读且同意 用户协议 和 隐私政策 并对我分享的文件的合法合规性负责
                </label>
            </div>
            <div className="flex gap-4 mt-2">
                 <button onClick={onCancel} className="flex-1 bg-slate-200 text-brand-light font-bold py-3 rounded-lg hover:bg-slate-300 transition-colors">取消</button>
                 <button onClick={onStartUpload} disabled={!tosAccepted || files.length === 0 || isProcessing} className="flex-1 bg-brand-cyan text-white font-bold py-3 rounded-lg hover:brightness-110 transition-all disabled:bg-slate-300 disabled:cursor-not-allowed flex items-center justify-center">
                    {isProcessing ? <X className="animate-spin"/> : '开始上传'}
                 </button>
            </div>
        </motion.div>
    );
};

const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
        opacity: 1,
        transition: {
            staggerChildren: 0.15,
            delayChildren: 0.2,
        },
    },
};

const itemVariants = {
    hidden: { y: 20, opacity: 0 },
    visible: {
        y: 0,
        opacity: 1,
        transition: { type: 'spring', stiffness: 100 } as const,
    },
};


const UploaderPage = () => {
    const { code: accessCodeInput, resetCode, setIsInputMode } = useUploaderStore();
    const [view, setView] = useState<'initial' | 'settings' | 'transferring' | 'success'>('initial');
    const [files, setFiles] = useState<File[]>([]);
    const folderInputRef = useRef<HTMLInputElement>(null);
    const navigate = useNavigate();
    const [uploadProgress, setUploadProgress] = useState(0);
    const [uploadSpeed, setUploadSpeed] = useState("0 B/s");
    const [error, setError] = useState<string | null>(null);
    const [shareDetails, setShareDetails] = useState<ShareDetails | null>(null);
    const [usePassword, setUsePassword] = useState(false);
    const [password, setPassword] = useState('');
    const [copiedLink, setCopiedLink] = useState(false);
    const [copiedCode, setCopiedCode] = useState(false);
    const [showQRCode, setShowQRCode] = useState(false);
    const uploadController = useRef<AbortController | null>(null);
    const [expiry, setExpiry] = useState(180); 
    const [downloadOnce, setDownloadOnce] = useState(false);
    const [processedFilename, setProcessedFilename] = useState('');
    const [isProcessing, setIsProcessing] = useState(false);
    const [isSubmittingCode, setIsSubmittingCode] = useState(false); // ✨ 新增状态
    const totalUploadSize = useMemo(() => files.reduce((acc, file) => acc + file.size, 0), [files]);

    const resetState = useCallback(() => {
        if(folderInputRef.current) {
            folderInputRef.current.value = "";
        }
        setView('initial');
        if (uploadController.current) {
            uploadController.current.abort("用户取消上传");
            uploadController.current = null;
        }
        setFiles([]);
        setUploadProgress(0);
        setUploadSpeed("0 B/s");
        setError(null);
        setShareDetails(null);
        setUsePassword(false);
        setPassword('');
        setCopiedLink(false);
        setCopiedCode(false);
        setShowQRCode(false);
        setExpiry(180);
        setDownloadOnce(false);
        resetCode();
        setIsInputMode(false);
        setProcessedFilename('');
        setIsProcessing(false);
        setIsSubmittingCode(false); // ✨ 重置提交状态
    }, [resetCode, setIsInputMode]);

    useEffect(() => {
        return () => {
            resetState();
        }
    }, [resetState]);

    const initiateStreamUpload = async (file: File) => {
        if (usePassword && password.length < 6) {
             setError("加密密码必须至少为6位。");
             setView('settings');
             setIsProcessing(false);
             return;
        }
        uploadController.current = new AbortController();
        try {
            let uploadableStream: ReadableStream<Uint8Array>;
            let salt: Uint8Array | null = null;
            let verificationHash: string | null = null;
            
            const progressStream = createProgressStream(file.size, setUploadProgress, setUploadSpeed);
            let sourceStream = file.stream().pipeThrough(progressStream);

            if (usePassword && password) {
                salt = E2EE.generateSalt();
                const [key, hash] = await Promise.all([
                    E2EE.deriveKeyFromPassword(password, salt),
                    E2EE.createVerificationHash(password, salt)
                ]);
                verificationHash = hash;
                const chunkingStream = E2EE.createChunkingStream();
                const encryptionStream = E2EE.createEncryptionStream(key);
                uploadableStream = sourceStream.pipeThrough(chunkingStream).pipeThrough(encryptionStream) as ReadableStream<Uint8Array>;
            } else {
                uploadableStream = sourceStream;
            }
            
            const headers: Record<string, string> = {
                'Content-Type': 'application/octet-stream',
                'X-File-Name': encodeURIComponent(file.name),
                'X-File-Original-Size': file.size.toString(),
                'X-File-Encrypted': usePassword.toString(),
                'X-File-Expires-In': expiry.toString(),
                'X-File-Download-Once': downloadOnce.toString()
            };

            if (salt) headers['X-File-Salt'] = E2EE.bufferToBase64(salt);
            if (verificationHash) headers['X-File-Verification-Hash'] = verificationHash;

            const uploadResponse = await fetch(`${DIRECT_API_BASE_URL}/api/v1/uploads/stream-complete`, {
                method: 'POST', headers, body: uploadableStream, signal: uploadController.current.signal, 
                //@ts-ignore
                duplex: 'half',
            });

            if (!uploadResponse.ok) {
                const errorText = await uploadResponse.text();
                try { throw new Error(JSON.parse(errorText).message || errorText); } catch { throw new Error(errorText); }
            }
            
            const details = await uploadResponse.json();
            setShareDetails(details);
            setUploadProgress(100);
            setView('success');

        } catch (err: any) {
            if (err.name !== 'AbortError') {
                setError(err.message || "上传时发生未知错误");
                setView('settings');
            } else {
                resetState();
            }
        } finally {
            setIsProcessing(false);
        }
    };
    
    const handleStartUpload = async () => {
        setError(null);
        setIsProcessing(true);
        
        if (files.length > 1) {
            setView('transferring');
            setUploadSpeed("正在压缩...");
            try {
                const zip = new JSZip();
                files.forEach(file => zip.file(file.name, file));
                const zipBlob = await zip.generateAsync({ type: 'blob', compression: 'DEFLATE' });
                const archiveName = `archive-${Date.now()}.zip`;
                const archiveFile = new File([zipBlob], archiveName, { type: 'application/zip' });
                
                setFiles([archiveFile]); 
                setProcessedFilename(archiveName);
                await initiateStreamUpload(archiveFile);
            } catch (err) {
                setError("压缩文件失败");
                setView('settings');
                setIsProcessing(false);
            }
        } else if (files.length === 1) {
            setView('transferring');
            setUploadSpeed("计算中...");
            setProcessedFilename(files[0].name);
            await initiateStreamUpload(files[0]);
        } else {
            setIsProcessing(false);
        }
    };
    
    const onDrop = useCallback((acceptedFiles: File[]) => {
        if (acceptedFiles.length > 0) {
            setFiles(prev => [...prev, ...acceptedFiles].filter((f, i, self) => i === self.findIndex(t => t.name === f.name && t.size === f.size)));
            setView('settings');
        }
    }, []);

    const handleFolderSelect = async (e: ChangeEvent<HTMLInputElement>) => {
        const selectedFiles = e.target.files;
        if (!selectedFiles || selectedFiles.length === 0) {
            return;
        }
        
        const filesArray = Array.from(selectedFiles);
        
        // @ts-ignore: webkitRelativePath is not in standard File type but exists
        const rootFolderName = filesArray[0].webkitRelativePath.split('/')[0];
        const zipFileName = `${rootFolderName}.zip`;

        setError(null);
        setIsProcessing(true);
        setView('settings');
        setFiles([]);

        setFiles([{ name: `正在压缩 ${rootFolderName}...`, size: 0, type: 'application/zip' } as File]);

        try {
            const zip = new JSZip();
            for (const file of filesArray) {
                // @ts-ignore
                zip.file(file.webkitRelativePath, file);
            }
            
            const zipBlob = await zip.generateAsync({ type: 'blob', compression: 'DEFLATE' });
            const finalZipFile = new File([zipBlob], zipFileName, { type: 'application/zip' });
            
            setFiles([finalZipFile]);
            
        } catch (err) {
            setError("压缩文件夹失败");
            setFiles([]);
            setView('initial');
        } finally {
            setIsProcessing(false);
        }
    };
    
    const { getRootProps, getInputProps, open: openFileDialog, isDragActive } = useDropzone({ 
        onDrop, 
        noClick: true, 
        noKeyboard: true 
    });
    
    const handleRemoveFile = (fileToRemove: File) => {
        const updatedFiles = files.filter(file => file !== fileToRemove);
        if (updatedFiles.length === 0) setView('initial');
        else setFiles(updatedFiles);
    };

    const copyToClipboard = (text: string, type: 'link' | 'code') => {
        navigator.clipboard.writeText(text).then(() => { 
            if (type === 'link') setCopiedLink(true); else setCopiedCode(true);
            setTimeout(() => { if (type === 'link') setCopiedLink(false); else setCopiedCode(false); }, 2000);
        });
    };

    // ✨ 更新提交逻辑以包含加载状态
    const handleAccessCodeSubmit = (e: FormEvent) => {
        e.preventDefault();
        if (accessCodeInput.trim().length === 6 && !isSubmittingCode) {
            setIsSubmittingCode(true);
            setTimeout(() => {
                navigate(`/download/${accessCodeInput.trim().toUpperCase()}`);
                // 组件即将卸载，无需重置状态，但以防万一
                setIsSubmittingCode(false);
            }, 500);
        }
    };

    const handleCodeComplete = (completedCode: string) => {
        if (!isSubmittingCode) {
            setIsSubmittingCode(true);
            setTimeout(() => {
                navigate(`/download/${completedCode.trim().toUpperCase()}`);
                setIsSubmittingCode(false);
            }, 500);
        }
    };

    const renderContent = () => {
        switch(view) {
            case 'transferring':
            case 'success':
                const isSuccess = view === 'success';
                const shareUrl = shareDetails ? `${window.location.origin}${shareDetails.urlPath}` : '';
                const expiryDate = new Date(Date.now() + expiry * 1000).toISOString();
                
                return (
                     <motion.div 
                        key="result" 
                        variants={containerVariants}
                        initial="hidden"
                        animate="visible"
                        className="w-full max-w-5xl mx-auto p-4 md:p-8 grid grid-cols-1 lg:grid-cols-5 gap-8 items-center"
                     >
                        <motion.div
                            variants={itemVariants}
                            className="lg:col-span-2 flex flex-col items-center justify-center text-center"
                        >
                            <UploadProgressCircle progress={uploadProgress} status={view} speed={uploadSpeed} />
                            <h2 className="text-3xl font-bold mt-4 text-brand-dark">{isSuccess ? '已完成' : '传输中'}</h2>
                            <p className="text-brand-light -mt-1 mb-6">
                                {isSuccess ? '文件已准备就绪' : '请保持页面开启'}
                            </p>
                            
                            {isSuccess && (
                                <div className="flex gap-4">
                                    <motion.button 
                                        onClick={resetState} 
                                        whileHover={{scale: 1.05, filter: 'brightness(1.1)'}} 
                                        whileTap={{scale:0.95, filter: 'brightness(0.95)'}} 
                                        className="bg-brand-cyan text-white font-bold py-3 px-6 rounded-lg transition-all flex items-center justify-center gap-2 shadow-soft-lg"
                                    >
                                        <Repeat size={18} /> 再传一次
                                    </motion.button>
                                </div>
                            )}
                        </motion.div>

                        <motion.div 
                            variants={itemVariants}
                            className="lg:col-span-3 bg-white/60 backdrop-blur-xl border border-white/20 rounded-2xl shadow-soft-xl p-6 text-brand-dark"
                        >
                             <h3 className="text-xl font-bold mb-1">传输预览</h3>
                             <div className="flex items-center gap-2 text-sm text-brand-light mb-4">
                                <Clock size={14}/>
                                <HumanizedCountdown expiresAt={expiryDate} />
                             </div>
                             <div className="bg-black/5 border border-black/5 rounded-xl p-4 mb-6">
                                <div className="flex items-center gap-3">
                                    <div className="p-3 bg-white/80 rounded-lg">
                                       {processedFilename.toLowerCase().endsWith('.zip') ? 
                                          <FolderPlus className="text-brand-cyan"/> :
                                          <FileText className="text-brand-cyan"/>
                                       }
                                    </div>
                                    <div className="flex-grow truncate">
                                        <p className="font-bold truncate">{processedFilename || "..."}</p>
                                        <p className="text-sm text-brand-light">
                                            {processedFilename.toLowerCase().endsWith('.zip') ? "文件夹" : `${files.length} 个文件`} / {formatBytes(totalUploadSize)}
                                        </p>
                                    </div>
                                </div>
                             </div>

                            {isSuccess && shareDetails && (
                                <motion.div 
                                    className="w-full space-y-4"
                                    variants={containerVariants}
                                    initial="hidden"
                                    animate="visible"
                                >
                                    <motion.h3 variants={itemVariants} className="text-xl font-bold">链接或口令分享</motion.h3>
                                    
                                    <motion.div variants={itemVariants} className="relative">
                                        <input type="text" readOnly value={shareUrl} className="w-full bg-black/5 p-3 pr-24 rounded-lg border-2 border-transparent text-brand-dark truncate focus:outline-none focus:border-brand-cyan"/>
                                        <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                                            <button 
                                                onClick={() => setShowQRCode(!showQRCode)}
                                                className={`p-2 rounded-md transition-colors ${showQRCode ? 'bg-brand-cyan text-white' : 'text-slate-400 hover:text-brand-cyan'}`}
                                            >
                                                <QrCode size={20} />
                                            </button>
                                            <button onClick={() => copyToClipboard(shareUrl, 'link')} className="p-2 w-9 h-9 flex items-center justify-center rounded-md bg-slate-200 hover:bg-brand-mint transition-colors text-brand-dark">
                                                <AnimatePresence mode="wait" initial={false}>
                                                    {copiedLink ? (
                                                        <motion.div key="check" initial={{ scale: 0.5, opacity: 0, rotate: -90 }} animate={{ scale: 1, opacity: 1, rotate: 0 }} exit={{ scale: 0.5, opacity: 0, rotate: 90 }} transition={{ type: 'spring', stiffness: 400, damping: 20 }}>
                                                            <Check size={20}/>
                                                        </motion.div>
                                                    ) : (
                                                        <motion.div key="copy" initial={{ scale: 0.5, opacity: 0, rotate: -90 }} animate={{ scale: 1, opacity: 1, rotate: 0 }} exit={{ scale: 0.5, opacity: 0, rotate: 90 }} transition={{ type: 'spring', stiffness: 400, damping: 20 }}>
                                                            <Copy size={20}/>
                                                        </motion.div>
                                                    )}
                                                </AnimatePresence>
                                            </button>
                                        </div>
                                    </motion.div>

                                    <motion.div variants={itemVariants} className="relative">
                                        <input type="text" readOnly value={shareDetails.accessCode} className="w-full bg-black/5 p-3 pr-14 rounded-lg border-2 border-transparent text-brand-dark font-mono tracking-widest text-center text-lg focus:outline-none focus:border-brand-cyan"/>
                                        <button onClick={() => copyToClipboard(shareDetails.accessCode, 'code')} className="absolute right-2 top-1/2 -translate-y-1/2 p-2 w-9 h-9 flex items-center justify-center rounded-md bg-slate-200 hover:bg-brand-mint transition-colors text-brand-dark">
                                            <AnimatePresence mode="wait" initial={false}>
                                                {copiedCode ? (
                                                    <motion.div key="check-code" initial={{ scale: 0.5, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} exit={{ scale: 0.5, opacity: 0 }}>
                                                        <Check size={20}/>
                                                    </motion.div>
                                                ) : (
                                                    <motion.div key="copy-code" initial={{ scale: 0.5, opacity: 0 }} animate={{ scale: 1, opacity: 1 }} exit={{ scale: 0.5, opacity: 0 }}>
                                                        <Copy size={20}/>
                                                    </motion.div>
                                                )}
                                            </AnimatePresence>
                                        </button>
                                    </motion.div>

                                    <AnimatePresence>
                                        {showQRCode && (
                                            <motion.div 
                                                initial={{ opacity: 0, height: 0, y: 20 }}
                                                animate={{ opacity: 1, height: 'auto', y: 0 }}
                                                exit={{ opacity: 0, height: 0, y: 20 }}
                                                className="bg-white p-4 rounded-lg mt-3"
                                            >
                                                <div className="relative flex justify-center items-center">
                                                    <QRCodeSVG
                                                        value={shareUrl}
                                                        size={160}
                                                        fgColor="#2BD8FB"
                                                        bgColor="transparent"
                                                        level="H"
                                                    />
                                                    <div className="absolute w-10 h-10 bg-white rounded-lg flex items-center justify-center shadow-md">
                                                        <Feather className="text-brand-cyan" size={24} />
                                                    </div>
                                                </div>
                                            </motion.div>
                                        )}
                                    </AnimatePresence>
                                </motion.div>
                            )}

                             {!isSuccess && <p className="text-center text-brand-light">完成传输后，将在此处显示分享详情。</p>}
                        </motion.div>
                     </motion.div>
                );

            case 'settings':
                return (
                     <UploadSettingsPanel
                        files={files}
                        onAddFiles={openFileDialog}
                        onRemoveFile={handleRemoveFile}
                        onStartUpload={handleStartUpload}
                        onCancel={resetState}
                        usePassword={usePassword}
                        setUsePassword={setUsePassword}
                        password={password}
                        setPassword={setPassword}
                        expiry={expiry}
                        setExpiry={setExpiry}
                        downloadOnce={downloadOnce}
                        setDownloadOnce={setDownloadOnce}
                        isProcessing={isProcessing}
                    />
                );
            case 'initial':
            default:
                return (
                    <motion.div key="initial" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }} className="flex flex-col items-center justify-center w-full gap-12">
                        <InteractiveUploader
                            onAddFileClick={openFileDialog}
                            onAddFolderClick={() => folderInputRef.current?.click()}
                            onCodeSubmit={handleAccessCodeSubmit}
                            onCodeComplete={handleCodeComplete}
                            isSubmittingCode={isSubmittingCode}
                        />
                    </motion.div>
                );
        }
    };

    return (
        <div {...getRootProps({ className: "relative w-full outline-none p-4 md:p-8 flex flex-col items-center justify-center min-h-[60vh] md:min-h-[500px] flex-grow" })}>
            <input {...getInputProps()} />
            <input
                type="file"
                ref={folderInputRef}
                onChange={handleFolderSelect}
                style={{ display: 'none' }}
                // @ts-ignore
                webkitdirectory=""
                directory=""
            />
            
            <AnimatePresence mode="wait">
              {renderContent()}
            </AnimatePresence>
            
            {error && <p className="text-center text-red-500 mt-4 p-3 rounded-lg bg-red-100">{error}</p>}

            <AnimatePresence>
                {isDragActive && (
                    <motion.div
                        className="absolute inset-0 bg-brand-cyan/20 backdrop-blur-sm flex flex-col items-center justify-center z-50 pointer-events-none border-4 border-dashed border-brand-cyan rounded-3xl"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        transition={{ duration: 0.2 }}
                    >
                        <motion.div
                           initial={{ y: 20, scale: 0.9 }}
                           animate={{ y: 0, scale: 1 }}
                           transition={{ type: 'spring', stiffness: 300, damping: 20 } as const}
                        >
                            <UploadCloud className="w-24 h-24 text-brand-cyan" />
                        </motion.div>
                        <p className="mt-4 text-2xl font-bold text-brand-dark">松手即可添加文件</p>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
};

export default UploaderPage;