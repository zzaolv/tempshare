// src/components/PublicFilesBoard.tsx
import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { fetchPublicFiles } from '../lib/api.ts';
import type { PublicFileInfo } from '../lib/api.ts';
import { File as FileIcon, Clock } from 'lucide-react';
import Countdown from './Countdown.tsx';

// ✨✨✨ 核心修改点 1: 创建一个内部的骨架屏组件 ✨✨✨
const FileItemSkeleton = () => (
    <div className="grid grid-cols-2 gap-4 p-4 items-center">
        <div className="col-span-1 flex items-center gap-3">
            <div className="w-5 h-5 bg-slate-200 rounded animate-pulse"></div>
            <div className="w-3/4 h-4 bg-slate-200 rounded animate-pulse"></div>
        </div>
        <div className="col-span-1 flex items-center justify-end gap-2">
            <div className="w-12 h-4 bg-slate-200 rounded animate-pulse"></div>
        </div>
    </div>
);

const PublicFilesBoard = ({ isPanelExpanded }: { isPanelExpanded: boolean }) => {
    const [files, setFiles] = useState<PublicFileInfo[]>([]);
    const [isLoading, setIsLoading] = useState(true);

    useEffect(() => {
        let intervalId: ReturnType<typeof setInterval> | undefined;

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
            setIsLoading(true); // 重置为 loading，以便下次展开时显示骨架屏
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
    
    return (
        <div className="p-4 md:p-6 w-full">
            <h2 className="text-2xl font-bold text-center mb-4 text-brand-dark">最新公开文件</h2>
            <div className="bg-black/5 border border-black/5 rounded-xl overflow-hidden">
                <div className="grid grid-cols-2 gap-4 p-4 border-b border-black/10 text-brand-light font-bold text-sm">
                    <div className="col-span-1">文件名</div>
                    <div className="col-span-1 text-right">剩余时间</div>
                </div>
                <div className="divide-y divide-black/5 max-h-[calc(100vh-200px)] overflow-y-auto">
                    {/* ✨✨✨ 核心修改点 2: 根据状态渲染内容或骨架屏 ✨✨✨ */}
                    {isLoading ? (
                        // 渲染骨架屏
                        Array.from({ length: 8 }).map((_, index) => <FileItemSkeleton key={index} />)
                    ) : files.length > 0 ? (
                        // 渲染真实文件列表
                        files.map(file => (
                            <Link to={`/download/${file.accessCode}`} key={file.accessCode} className="grid grid-cols-2 gap-4 p-4 items-center hover:bg-black/10 transition-colors duration-200 text-brand-dark">
                                <div className="col-span-1 flex items-center gap-3 truncate">
                                    <FileIcon className="w-5 h-5 text-brand-cyan flex-shrink-0" />
                                    <span className="truncate">{file.filename}</span>
                                </div>
                                <div className="col-span-1 flex items-center justify-end gap-2">
                                    <Clock size={16} />
                                    <Countdown expiresAt={file.expiresAt} />
                                </div>
                            </Link>
                        ))
                    ) : (
                        // 渲染空状态提示
                        <div className="text-center text-brand-light p-8">现在还没有公开分享的文件哦，快来上传一个吧！</div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default PublicFilesBoard;