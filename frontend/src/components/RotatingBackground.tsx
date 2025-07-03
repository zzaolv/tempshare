// frontend/src/components/RotatingBackground.tsx
import { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';

// ✨ 在这里定义您的背景图片列表
// 图片路径是相对于 /public 目录的
const backgroundImages = [
    '/backgrounds/bg1.jpg',
    '/backgrounds/bg2.jpg',
    '/backgrounds/bg3.jpg',
    // 您可以根据需要添加更多图片
];

// 图片轮播间隔（毫秒）
const ROTATION_INTERVAL = 10000; // 10 秒

const RotatingBackground = () => {
    const [index, setIndex] = useState(0);

    useEffect(() => {
        const timer = setInterval(() => {
            setIndex((prevIndex) => (prevIndex + 1) % backgroundImages.length);
        }, ROTATION_INTERVAL);

        return () => clearInterval(timer); // 组件卸载时清除定时器
    }, []);

    return (
        <div className="fixed inset-0 z-[-1] overflow-hidden bg-black">
            <AnimatePresence>
                <motion.div
                    key={index} // ✨ 使用 key 来触发 AnimatePresence 的动画
                    className="absolute inset-0 bg-cover bg-center"
                    style={{ backgroundImage: `url(${backgroundImages[index]})` }}
                    initial={{ opacity: 0, scale: 1.05 }}
                    animate={{ opacity: 1, scale: 1, transition: { duration: 1.5, ease: 'easeOut' } }}
                    exit={{ opacity: 0, scale: 1.05, transition: { duration: 1.5, ease: 'easeIn' } }}
                />
            </AnimatePresence>
            {/* 添加一个半透明的黑色遮罩层，以确保前景文字的可读性 */}
            <div className="absolute inset-0 bg-black/30"></div>
        </div>
    );
};

export default RotatingBackground;