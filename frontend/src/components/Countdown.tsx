// frontend/src/components/Countdown.tsx
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

    if (remainingSeconds <= 0) return <span className="text-red-500 font-semibold">已过期</span>;

    const h = String(Math.floor(remainingSeconds / 3600)).padStart(2, '0');
    const m = String(Math.floor((remainingSeconds % 3600) / 60)).padStart(2, '0');
    const s = String(remainingSeconds % 60).padStart(2, '0');
    
    // ✨✨✨ 核心修改点: 根据剩余时间动态改变样式 ✨✨✨
    const isUrgent = remainingSeconds < 60;
    const timeColor = isUrgent ? 'text-red-500 animate-pulse' : 'text-brand-light';

    return (
        <span className={`font-mono transition-colors duration-300 ${timeColor} ${className || ''}`}>
            {h}:{m}:{s}
        </span>
    );
};

export default Countdown;