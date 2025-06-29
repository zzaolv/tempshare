import { useState, useEffect, useCallback } from 'react';

// 定义时间单位的中文表示
const timeUnits = {
    day: '天',
    hour: '小时',
    minute: '分钟',
    second: '秒',
};

const HumanizedCountdown = ({ expiresAt, prefix = '', suffix = '后到期' }: { expiresAt: string, prefix?: string, suffix?: string }) => {
    const calculateRemaining = useCallback(() => {
        const diff = new Date(expiresAt).getTime() - new Date().getTime();
        return Math.max(0, Math.floor(diff / 1000));
    }, [expiresAt]);

    const [remainingSeconds, setRemainingSeconds] = useState(calculateRemaining());

    useEffect(() => {
        // 每分钟更新一次就足够了，除非时间很短
        const updateInterval = remainingSeconds > 3600 ? 60000 : 1000;
        const timer = setInterval(() => {
            setRemainingSeconds(calculateRemaining());
        }, updateInterval);
        return () => clearInterval(timer);
    }, [calculateRemaining, remainingSeconds]);

    if (remainingSeconds <= 0) {
        return <span className="text-red-400">已过期</span>;
    }

    const days = Math.floor(remainingSeconds / 86400);
    const hours = Math.floor((remainingSeconds % 86400) / 3600);
    const minutes = Math.floor((remainingSeconds % 3600) / 60);
    const seconds = remainingSeconds % 60;

    const parts = [];
    if (days > 0) parts.push(`${days} ${timeUnits.day}`);
    if (hours > 0) parts.push(`${hours} ${timeUnits.hour}`);
    // 当时间大于1小时，就不显示分钟和秒
    if (days === 0 && hours === 0) {
        if (minutes > 0) parts.push(`${minutes} ${timeUnits.minute}`);
        if (minutes < 5 && seconds > 0) parts.push(`${seconds} ${timeUnits.second}`); // 只在最后几分钟显示秒
    }

    // 如果 parts 为空 (例如只剩下几秒钟), 显示一个通用消息
    if (parts.length === 0 && remainingSeconds > 0) {
        return <span className="text-sky-300">{`${prefix}即将到期${suffix}`}</span>;
    }
    
    // 取前两个最重要的时间单位
    const displayText = parts.slice(0, 2).join(' ');

    return <span className="text-slate-400">{`${prefix}${displayText}${suffix}`}</span>;
};

export default HumanizedCountdown;