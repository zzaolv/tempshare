// src/components/SidePanel.tsx
import { useEffect, useRef, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { animate, JSAnimation } from 'animejs';
// ✨ 修正：添加缺失的 import
import PublicFilesBoard from './PublicFilesBoard';
import { ChevronRight } from 'lucide-react';

interface SidePanelProps {
    isExpanded: boolean;
    onMouseEnter: () => void;
    onMouseLeave: () => void;
    onClose: () => void;
}

const SidePanel = ({ isExpanded, onMouseEnter, onMouseLeave, onClose }: SidePanelProps) => {
    const [isMobile, setIsMobile] = useState(false);
    const panelRef = useRef<HTMLDivElement>(null);
    const animationRef = useRef<JSAnimation | null>(null);

    useEffect(() => {
        const checkMobile = () => {
            const mobile = window.innerWidth < 1024;
            setIsMobile(mobile);
            if (panelRef.current) {
                if (mobile) {
                    panelRef.current.style.width = '320px';
                    panelRef.current.style.transform = isExpanded ? 'translateX(0%)' : 'translateX(-100%)';
                } else {
                    panelRef.current.style.transform = '';
                    panelRef.current.style.width = isExpanded ? '384px' : '64px';
                }
            }
        };
        checkMobile();
        window.addEventListener('resize', checkMobile);
        return () => window.removeEventListener('resize', checkMobile);
    }, [isExpanded]);

    useEffect(() => {
        if (panelRef.current) {
            if (animationRef.current) {
                animationRef.current.pause();
            }

            animationRef.current = animate(
                panelRef.current, // 第一个参数: target
                {                 // 第二个参数: parameters (一个对象)
                    ...(isMobile
                        ? { translateX: isExpanded ? '0%' : '-100%' }
                        : { width: isExpanded ? '384px' : '64px' }),
                    duration: 500,
                    easing: 'easeOutQuint',
                }
            );
        }
    }, [isExpanded, isMobile]);

    const desktopHoverHandlers = isMobile ? {} : { onMouseEnter, onMouseLeave };

    return (
        <>
            <AnimatePresence>
                {isExpanded && isMobile && (
                    <motion.div
                        className="fixed inset-0 bg-black/50 z-10 lg:hidden"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        exit={{ opacity: 0 }}
                        onClick={onClose}
                    />
                )}
            </AnimatePresence>
            
            <div
                ref={panelRef}
                className="fixed top-0 left-0 h-full z-20"
                style={{
                    width: isMobile ? '320px' : '64px', 
                    transform: isMobile ? 'translateX(-100%)' : 'none' 
                }}
                {...desktopHoverHandlers}
            >
                <div className="h-full w-full bg-white/50 backdrop-blur-2xl border-r border-white/20 shadow-soft-xl overflow-hidden">
                    <motion.div
                        className="w-[384px] max-w-full"
                        initial={{ opacity: 0 }}
                        animate={{ opacity: isExpanded ? 1 : 0 }}
                        transition={{ delay: isExpanded ? 0.2 : 0, duration: 0.2 }}
                    >
                        <PublicFilesBoard isPanelExpanded={isExpanded} />
                    </motion.div>

                    <motion.div
                        className="absolute top-1/2 -translate-y-1/2 left-0 w-16 flex flex-col items-center justify-center gap-4 text-brand-dark"
                        animate={{ opacity: isExpanded ? 0 : 1 }}
                        transition={{ delay: isExpanded ? 0 : 0.2, duration: 0.2 }}
                    >
                        <ChevronRight className="w-6 h-6" />
                        <p className="[writing-mode:vertical-rl] tracking-widest text-sm font-semibold">
                            最新文件
                        </p>
                    </motion.div>
                </div>
            </div>
        </>
    );
};

export default SidePanel;