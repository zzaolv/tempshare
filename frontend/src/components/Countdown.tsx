import { useState, useEffect, useCallback } from 'react';

const Countdown = ({ expiresAt, className }: { expiresAt: string, className?: string }) => {
    const calculateRemaining = useCallback(() => {
        const diff = new Date(expiresAt).getTime() - new Date().getTime();
        return Math.max(0, Math.floor(diff / 1000));
    }, [expiresAt]);

    const [remainingSeconds, setRemainingSeconds] = useState(calculateRemaining());

    useEffect(() => {
        const timer = setInterval(() => setRemainingSeconds(calculateRemaining()), 1000);
        return () => clearInterval(timer);
    }, [calculateRemaining]);

    if (remainingSeconds <= 0) return <span className="text-red-500">已过期</span>;

    const h = String(Math.floor(remainingSeconds / 3600)).padStart(2, '0');
    const m = String(Math.floor((remainingSeconds % 3600) / 60)).padStart(2, '0');
    const s = String(remainingSeconds % 60).padStart(2, '0');
    
    // ✨ 美化改动: 调整倒计时颜色
    return <span className={`font-mono text-brand-light ${className || ''}`}>{h}:{m}:{s}</span>;
};

export default Countdown;