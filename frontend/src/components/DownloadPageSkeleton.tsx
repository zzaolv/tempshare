// frontend/src/components/DownloadPageSkeleton.tsx
import { FileQuestion, ShieldQuestion } from 'lucide-react';

const SkeletonBox = ({ className = '' }: { className?: string }) => (
  <div className={`bg-slate-200/80 animate-pulse rounded-md ${className}`} />
);

const DownloadPageSkeleton = () => {
  return (
    <div className="w-full max-w-6xl mx-auto p-4 md:p-8 grid grid-cols-1 lg:grid-cols-2 gap-8 pointer-events-none">
      {/* Left Card Skeleton */}
      <div className="w-full bg-white/50 backdrop-blur-xl border border-white/10 rounded-2xl shadow-soft-2xl p-8 flex flex-col">
        <div className="text-center">
          <SkeletonBox className="h-9 w-3/4 mx-auto" />
          <SkeletonBox className="h-5 w-1/2 mx-auto mt-3" />
        </div>
        
        <div className="w-full my-6 p-4 bg-black/5 rounded-lg space-y-3">
            <div className="flex items-center justify-between">
              <SkeletonBox className="h-5 w-1/4" />
              <SkeletonBox className="h-5 w-1/4" />
            </div>
            <div className="w-full h-px bg-black/10"></div>
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-2 text-slate-400">
                    <ShieldQuestion size={20} /><span>扫描状态</span>
                </div>
                <SkeletonBox className="h-5 w-1/3" />
            </div>
        </div>

        <div className="mt-auto">
            <SkeletonBox className="h-14 w-full" />
        </div>
         <SkeletonBox className="h-4 w-1/3 mx-auto mt-6" />
      </div>

      {/* Right Card Skeleton */}
      <div className="w-full bg-white/50 backdrop-blur-xl border border-white/10 rounded-2xl shadow-soft-xl p-4 flex flex-col min-h-[300px] lg:min-h-0">
         <SkeletonBox className="h-7 w-1/3 mb-4" />
          <div className="flex-grow flex flex-col items-center justify-center text-center p-8 bg-black/5 rounded-lg">
              <div className="mb-4">
                  <FileQuestion className="w-16 h-16 text-slate-300 flex-shrink-0" strokeWidth={1.5} />
              </div>
              <SkeletonBox className="h-7 w-4/5" />
              <SkeletonBox className="h-10 w-28 mt-8" />
          </div>
      </div>
    </div>
  );
};

export default DownloadPageSkeleton;