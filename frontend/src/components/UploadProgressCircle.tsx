// frontend/src/components/UploadProgressCircle.tsx

import { motion } from 'framer-motion';

interface UploadProgressCircleProps {
  status: 'transferring' | 'success' | 'initial';
  progress: number;
  speed?: string;
}

const circleVariants = {
  hidden: {
    strokeDashoffset: 314,
  },
  visible: (progress: number) => ({
    strokeDashoffset: 314 - (progress / 100) * 314,
    transition: { type: 'spring' as const, stiffness: 50, damping: 15 },
  }),
};

const checkmarkVariants = {
  hidden: { pathLength: 0 },
  visible: {
    pathLength: 1,
    transition: { duration: 0.5, ease: 'easeOut' as const, delay: 0.2 },
  },
};

const UploadProgressCircle = ({ progress, status, speed }: UploadProgressCircleProps) => {
  return (
    <div className="relative w-48 h-48 flex items-center justify-center">
      <svg className="absolute w-full h-full" viewBox="0 0 120 120">
        {/* ✨ 美化改动: 背景环颜色 */}
        <circle
          cx="60"
          cy="60"
          r="50"
          strokeWidth="8"
          className="stroke-black/10"
          fill="none"
        />
        {/* Progress Circle */}
        {status === 'transferring' && (
          <motion.circle
            cx="60"
            cy="60"
            r="50"
            strokeWidth="8"
            // ✨ 美化改动: 进度条使用品牌青色
            className="stroke-brand-cyan"
            fill="none"
            transform="rotate(-90 60 60)"
            strokeLinecap="round"
            strokeDasharray="314"
            variants={circleVariants}
            initial="hidden"
            animate="visible"
            custom={progress}
          />
        )}
        {/* Success Circle */}
        {status === 'success' && (
           <motion.circle
            cx="60"
            cy="60"
            r="50"
            strokeWidth="8"
             // ✨ 美化改动: 成功状态使用品牌绿色
            className="stroke-brand-mint"
            fill="none"
            initial={{ scale: 0.8, opacity: 0 }}
            animate={{ scale: 1, opacity: 1, transition: { duration: 0.5, ease: 'easeOut' as const } }}
          />
        )}
      </svg>
      {/* ✨ 美化改动: 调整文字颜色 */}
      <div className="z-10 text-center text-brand-dark">
        {status === 'transferring' && (
          <>
            <div className="text-4xl font-bold">{Math.round(progress)}<span className="text-2xl">%</span></div>
            <div className="text-sm text-brand-light mt-1 h-5">{speed}</div>
          </>
        )}
        {status === 'success' && (
          <motion.svg
            // ✨ 美化改动: 对勾使用品牌绿色
            className="w-20 h-20 text-brand-mint"
            viewBox="0 0 52 52"
          >
            <motion.path
              fill="none"
              strokeWidth="4"
              strokeLinecap="round"
              strokeLinejoin="round"
              stroke="currentColor"
              d="M14 27l6 6 18-18"
              variants={checkmarkVariants}
              initial="hidden"
              animate="visible"
            />
          </motion.svg>
        )}
      </div>
    </div>
  );
};

export default UploadProgressCircle;