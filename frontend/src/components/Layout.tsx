
// src/components/Layout.tsx
import { useEffect, useRef, useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Menu, X } from 'lucide-react';
import SidePanel from './SidePanel';

const pageVariants = {
    initial: { opacity: 0, y: 30, scale: 0.98 },
    in: { opacity: 1, y: 0, scale: 1 },
    out: { opacity: 0, y: -30, scale: 1.02 },
};
const pageTransition = { type: 'spring', stiffness: 260, damping: 20 } as const;

const InteractiveGradientBackground = () => {
    const backgroundRef = useRef<HTMLDivElement>(null);
    const animationFrameId = useRef<number | null>(null);
    
    const mousePos = useRef({ targetX: window.innerWidth / 2, targetY: window.innerHeight / 2 });
    const currentPos = useRef({ x: window.innerWidth / 2, y: window.innerHeight / 2 });

    useEffect(() => {
        const backgroundEl = backgroundRef.current;
        if (!backgroundEl) return;
        
        const handleMouseMove = (event: MouseEvent) => {
            mousePos.current.targetX = event.clientX;
            mousePos.current.targetY = event.clientY;
        };

        window.addEventListener('mousemove', handleMouseMove);

        const animate = () => {
            const dx = mousePos.current.targetX - currentPos.current.x;
            const dy = mousePos.current.targetY - currentPos.current.y;

            currentPos.current.x += dx * 0.075;
            currentPos.current.y += dy * 0.075;

            backgroundEl.style.setProperty('--mouse-x', `${currentPos.current.x}px`);
            backgroundEl.style.setProperty('--mouse-y', `${currentPos.current.y}px`);

            animationFrameId.current = requestAnimationFrame(animate);
        };

        animate();
        
        return () => {
            window.removeEventListener('mousemove', handleMouseMove);
            if (animationFrameId.current) {
                cancelAnimationFrame(animationFrameId.current);
            }
        };
    }, []);

    return (
        <div 
            ref={backgroundRef} 
            className="interactive-gradient-background"
        ></div>
    );
};


const Layout = () => {
    const location = useLocation();
    const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);
    const [isHoveringSidebar, setIsHoveringSidebar] = useState(false);

    const shouldExpand = isSidebarExpanded || isHoveringSidebar;

    return (
        <div className="flex flex-col h-screen h-dvh bg-gray-100 dark:bg-gray-900 text-gray-900 dark:text-gray-100">
            
            <InteractiveGradientBackground />
            
            <button
                onClick={() => setIsSidebarExpanded(!isSidebarExpanded)}
                className="lg:hidden fixed top-4 left-4 z-30 p-2 bg-white/50 backdrop-blur-lg rounded-full text-brand-dark"
            >
                {shouldExpand ? <X size={24} /> : <Menu size={24} />}
            </button>
            
            <SidePanel 
                isExpanded={shouldExpand}
                onMouseEnter={() => setIsHoveringSidebar(true)}
                onMouseLeave={() => setIsHoveringSidebar(false)}
                onClose={() => setIsSidebarExpanded(false)}
            />
            
            {/* ✨✨✨ 核心修正：修复布局偏移问题 ✨✨✨ */}
            <div className={`
                flex-grow flex flex-col items-center p-4 
                transition-all duration-500 ease-in-out
                lg:pl-20 /* 默认在PC端为侧边栏留出 5rem 的空间 */
                ${shouldExpand ? 'lg:pl-[384px]' : 'lg:pl-20'} /* 展开时PC端内边距变大，否则恢复默认 */
            `}>
                <div className="w-full max-w-4xl z-10 flex flex-col flex-grow">
                    <header className="text-center my-8 md:my-12 flex-shrink-0">
                        <Link to="/" className="inline-block">
                            <h1 className="text-4xl md:text-5xl font-bold text-brand-dark transition-transform duration-300 hover:scale-105">
                                🪽 闪传驿站 <span className="text-brand-cyan">TempShare</span>
                            </h1>
                        </Link>
                        <p className="text-slate-500 mt-2">安全、快速、无需登录的临时文件分享</p>
                    </header>

                    <main className="relative flex-grow flex items-center justify-center">
                        <AnimatePresence mode="wait">
                            <motion.div
                                key={location.pathname}
                                initial="initial"
                                animate="in"
                                exit="out"
                                variants={pageVariants}
                                transition={pageTransition}
                                className="w-full"
                            >
                                {/* 恢复您原始的白色毛玻璃卡片样式 */}
                                <div className="bg-white/50 backdrop-blur-2xl border border-white/10 rounded-2xl shadow-soft-2xl">
                                    <Outlet />
                                </div>
                            </motion.div>
                        </AnimatePresence>
                    </main>

                    <footer className="text-center mt-8 text-slate-500 text-sm space-y-2 flex-shrink-0">
                        <p>一个纯粹、值得信赖的临时文件分享工具。我们仅记录您的IP地址用于防止滥用。</p>
                        <div>
                            <Link to="/report" className="hover:text-brand-cyan underline">
                                举报滥用内容
                            </Link>
                        </div>
                    </footer>
                </div>
            </div>
        </div>
    );
};

export default Layout;
