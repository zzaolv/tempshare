// src/components/PublicFilesBoard.tsx
import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { fetchPublicFiles } from '../lib/api.ts';
import type { PublicFileInfo } from '../lib/api.ts';
import { File as FileIcon, Lock, Clock, LoaderCircle } from 'lucide-react';
import Countdown from './Countdown.tsx';

const PublicFilesBoard = ({ isPanelExpanded }: { isPanelExpanded: boolean }) => {
    const [files, setFiles] = useState<PublicFileInfo[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        // ✨ 修正点: 使用 ReturnType<...> 自动推断 setInterval 的返回类型
        let intervalId: ReturnType<typeof setInterval>;

        const loadFiles = async () => {
            try {
                const fetchedFiles = await fetchPublicFiles();
                setFiles(fetchedFiles);
            } catch (error) {
                console.error(error);
                setFiles([]);
            } finally {
                setIsLoading(false);
            }
        };

        if (isPanelExpanded) {
            setIsLoading(true);
            loadFiles();
            intervalId = setInterval(loadFiles, 30000);
        } else {
            setFiles([]);
            setIsLoading(true);
        }

        return () => {
            if (intervalId) {
                clearInterval(intervalId);
            }
        };
    }, [isPanelExpanded]);

    if (!isPanelExpanded) {
        return null;
    }

    if (isLoading) {
        return <div className="text-center text-brand-light p-8 flex items-center justify-center gap-2"><LoaderCircle className="animate-spin text-brand-cyan" /> 正在加载最新文件...</div>;
    }

    if (files.length === 0) {
        return <div className="text-center text-brand-light p-8">现在还没有公开分享的文件哦，快来上传一个吧！</div>;
    }

    return (
        <div className="p-4 md:p-6 w-full">
            <h2 className="text-2xl font-bold text-center mb-4 text-brand-dark">最新公开文件</h2>
            <div className="bg-black/5 border border-black/5 rounded-xl overflow-hidden">
                <div className="grid grid-cols-2 gap-4 p-4 border-b border-black/10 text-brand-light font-bold text-sm">
                    <div className="col-span-1">文件名</div>
                    <div className="col-span-1 text-right">剩余时间</div>
                </div>
                <div className="divide-y divide-black/5 max-h-[calc(100vh-200px)] overflow-y-auto">
                    {files.map(file => (
                        <Link to={`/download/${file.accessCode}`} key={file.accessCode} className="grid grid-cols-2 gap-4 p-4 items-center hover:bg-black/10 transition-colors duration-200 text-brand-dark">
                            <div className="col-span-1 flex items-center gap-3 truncate">
                                {file.isEncrypted ? <Lock className="w-5 h-5 text-brand-yellow flex-shrink-0" /> : <FileIcon className="w-5 h-5 text-brand-cyan flex-shrink-0" />}
                                <span className="truncate">{file.filename}</span>
                            </div>
                            <div className="col-span-1 flex items-center justify-end gap-2">
                                <Clock size={16} />
                                <Countdown expiresAt={file.expiresAt} />
                            </div>
                        </Link>
                    ))}
                </div>
            </div>
        </div>
    );
};

export default PublicFilesBoard;