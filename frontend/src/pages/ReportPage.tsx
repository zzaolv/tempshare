// src/pages/ReportPage.tsx
import { useState } from 'react';
import type { FormEvent } from 'react';
import { motion } from 'framer-motion';
import { LoaderCircle, ShieldAlert } from 'lucide-react';
import { submitReport } from '../lib/api';

const ReportPage = () => {
    const [accessCode, setAccessCode] = useState('');
    const [reason, setReason] = useState('');
    const [message, setMessage] = useState('');
    const [isError, setIsError] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);

    const handleSubmit = async (e: FormEvent) => {
        e.preventDefault();
        setIsSubmitting(true);
        setMessage('');
        setIsError(false);
        try {
            const data = await submitReport(accessCode, reason);
            setMessage(data.message);
            const hasError = data.message.includes('失败') || data.message.includes('无效');
            setIsError(hasError);
            if (!hasError) {
              setAccessCode('');
              setReason('');
            }
        } catch (error) {
            setMessage('提交失败，请检查网络连接。');
            setIsError(true);
        } finally {
            setIsSubmitting(false);
        }
    };

    return (
        // ✨ 美化改动: 调整内边距和最大宽度
        <div className="p-6 md:p-8 w-full max-w-lg mx-auto">
            <h2 className="text-2xl font-bold text-center mb-6 text-brand-dark">举报非法或滥用内容</h2>
            <form onSubmit={handleSubmit} className="space-y-6">
                <div>
                    {/* ✨ 美化改动: 调整标签颜色 */}
                    <label htmlFor="report-code" className="block text-sm font-medium text-brand-light mb-2">6位文件便捷码</label>
                    {/* ✨ 美化改动: 调整输入框样式 */}
                    <input id="report-code" type="text" value={accessCode} onChange={e => setAccessCode(e.target.value.toUpperCase())} maxLength={6} required className="w-full bg-black/5 p-3 rounded-md border-2 border-transparent focus:border-brand-cyan focus:ring-brand-cyan transition font-mono tracking-widest text-center" />
                </div>
                <div>
                    <label htmlFor="report-reason" className="block text-sm font-medium text-brand-light mb-2">举报原因 (选填)</label>
                    <textarea id="report-reason" value={reason} onChange={e => setReason(e.target.value)} rows={4} className="w-full bg-black/5 p-3 rounded-md border-2 border-transparent focus:border-brand-cyan focus:ring-brand-cyan transition"></textarea>
                </div>
                {/* ✨ 美化改动: 调整按钮样式和动画 */}
                <motion.button type="submit" disabled={isSubmitting || accessCode.length !== 6} whileHover={{scale: 1.05}} whileTap={{scale:0.95}} className="w-full inline-flex items-center justify-center bg-red-500 hover:bg-red-600 disabled:bg-slate-300 text-white font-bold py-3 rounded-lg transition-colors">
                    {isSubmitting ? <LoaderCircle className="animate-spin mr-2" /> : <ShieldAlert className="mr-2" />} 提交举报
                </motion.button>
                {message && (
                  // ✨ 美化改动: 调整消息提示框样式
                  <p className={`text-center mt-4 p-3 rounded-lg ${isError ? 'bg-red-500/10 text-red-700' : 'bg-green-500/10 text-green-700'}`}>
                      {message}
                  </p>
                )}
            </form>
        </div>
    );
};

export default ReportPage;