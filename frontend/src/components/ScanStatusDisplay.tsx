// src/components/ScanStatusDisplay.tsx
import { LoaderCircle, ShieldCheck, XCircle, SkipForward, ShieldQuestion } from 'lucide-react';
import type { FileMetadata } from '../lib/api.ts';

const ScanStatusDisplay = ({ status, result }: { status: FileMetadata['scanStatus'], result: string }) => {
    // ✨ 美化改动: 调整颜色以适应新主题
    switch (status) {
        case 'clean':
            return <div className="flex items-center gap-2 text-green-600"><ShieldCheck size={20} /><span>{result}</span></div>;
        case 'infected':
            return <div className="flex items-center gap-2 text-red-600"><XCircle size={20} /><span>威胁: {result}</span></div>;
        case 'skipped':
             return <div className="flex items-center gap-2 text-slate-500"><SkipForward size={20} /><span>{result}</span></div>;
        case 'error':
            return <div className="flex items-center gap-2 text-amber-600"><ShieldQuestion size={20} /><span>扫描状态: {result}</span></div>;
        default:
            return <div className="flex items-center gap-2 text-slate-500"><LoaderCircle className="animate-spin" size={20} /><span>获取扫描状态...</span></div>;
    }
};

export default ScanStatusDisplay;