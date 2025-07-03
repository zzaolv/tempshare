// src/components/Layout.tsx
import {  useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { motion, AnimatePresence } from 'framer-motion';
import { Menu, X } from 'lucide-react';
import SidePanel from './SidePanel';
// ✨✨✨ 核心修改点 1: 导入新的背景组件 ✨✨✨
import RotatingBackground from './RotatingBackground'; 

const pageVariants = {
    initial: { opacity: 0, y: 30, scale: 0.98 },
    in: { opacity: 1, y: 0, scale: 1 },
    out: { opacity: 0, y: -30, scale: 1.02 },
};
const pageTransition = { type: 'spring', stiffness: 260, damping: 20 } as const;

// 之前的 InteractiveGradientBackground 组件可以删除了

const Layout = () => {
    const location = useLocation();
    const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);
    const [isHoveringSidebar, setIsHoveringSidebar] = useState(false);

    const shouldExpand = isSidebarExpanded || isHoveringSidebar;

    return (
        <div className="flex flex-col min-h-screen min-h-dvh text-gray-900 dark:text-gray-100">
            
            {/* ✨✨✨ 核心修改点 2: 使用新的背景组件 ✨✨✨ */}
            <RotatingBackground />
            
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
            
            <div className={`
                flex-grow flex flex-col items-center p-4 
                transition-all duration-500 ease-in-out
                lg:pl-20 
                ${shouldExpand ? 'lg:pl-[408px]' : 'lg:pl-20'}
            `}>
                <div className="w-full max-w-4xl z-10 flex flex-col flex-grow">
                    <header className="text-center my-8 md:my-12 flex-shrink-0">
                        <Link to="/" className="inline-block">
                             {/* ✨ 建议: 为了在图片背景上更清晰，给文字添加阴影 */}
                            <h1 className="text-4xl md:text-5xl font-bold text-white transition-transform duration-300 hover:scale-105 [text-shadow:_0_2px_4px_rgb(0_0_0_/_40%)]">
                                🪽 闪传驿站 <span className="text-brand-cyan">TempShare</span>
                            </h1>
                        </Link>
                        <p className="text-slate-200 mt-2 [text-shadow:_0_1px_2px_rgb(0_0_0_/_50%)]">安全、快速、无需登录的临时文件分享</p>
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
                                <div className="bg-white/50 backdrop-blur-2xl border border-white/10 rounded-2xl shadow-soft-2xl">
                                    <Outlet />
                                </div>
                            </motion.div>
                        </AnimatePresence>
                    </main>

                    <footer className="text-center mt-8 text-slate-300 text-sm space-y-2 flex-shrink-0 [text-shadow:_0_1px_2px_rgb(0_0_0_/_50%)]">
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