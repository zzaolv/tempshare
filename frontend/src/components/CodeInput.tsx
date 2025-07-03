// src/components/CodeInput.tsx
import React, { useRef, useImperativeHandle, forwardRef, useEffect } from 'react';
// ✨ 导入 LoaderCircle 图标
import { LoaderCircle } from 'lucide-react';

// ✨ 增加 isSubmitting prop
interface CodeInputProps {
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  isSubmitting?: boolean;
}

export interface CodeInputHandle {
  focus: () => void;
}

const CODE_LENGTH = 6;

const CodeInput = forwardRef<CodeInputHandle, CodeInputProps>(({ value, onChange, onComplete, isSubmitting = false }, ref) => {
  const inputRef = useRef<HTMLInputElement>(null);

  useImperativeHandle(ref, () => ({
    focus: () => {
      inputRef.current?.focus();
    },
  }));

  useEffect(() => {
    // ✨ 当输入完成且未在提交状态时，才调用 onComplete
    if (value.length === CODE_LENGTH && onComplete && !isSubmitting) {
      onComplete(value);
    }
  }, [value, onComplete, isSubmitting]);

  const handleContainerClick = () => {
    // ✨ 提交时禁止聚焦，防止用户再次输入
    if (!isSubmitting) {
        inputRef.current?.focus();
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (isSubmitting) return;
    const newValue = e.target.value.replace(/[^a-zA-Z0-9]/g, '').toUpperCase().slice(0, CODE_LENGTH);
    onChange(newValue);
  };

  const characters = value.split('');

  return (
    <div
      className={`relative w-full h-16 flex items-center justify-center gap-2 md:gap-3 ${isSubmitting ? 'cursor-wait' : 'cursor-text'}`}
      onClick={handleContainerClick}
    >
      {Array.from({ length: CODE_LENGTH }).map((_, index) => {
        const hasCharacter = index < characters.length;
        const isCurrent = index === characters.length;

        return (
          <div
            key={index}
            className={`
              w-10 h-14 md:w-12 md:h-16 flex items-center justify-center
              text-2xl md:text-3xl font-mono font-bold text-brand-dark
              bg-black/5 border-2 rounded-lg
              transition-all duration-200 ease-in-out
              ${isCurrent && !isSubmitting ? 'border-brand-cyan shadow-lg shadow-brand-cyan/20' : 'border-transparent'}
              ${hasCharacter ? 'border-black/10' : ''}
              ${isSubmitting ? 'opacity-70' : ''}
            `}
          >
            {characters[index] || ''}
            {/* ✨ 核心修改点: 根据状态显示光标或加载图标 ✨ */}
            {isCurrent && !isSubmitting && (
              <span className="absolute w-0.5 h-7 bg-brand-cyan animate-pulse rounded-full"></span>
            )}
            {isSubmitting && index === CODE_LENGTH - 1 && (
                <div className="absolute flex items-center justify-center">
                    <LoaderCircle className="w-6 h-6 text-brand-cyan animate-spin" />
                </div>
            )}
          </div>
        );
      })}

      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={handleInputChange}
        maxLength={CODE_LENGTH}
        disabled={isSubmitting} // ✨ 提交时禁用输入
        className="absolute w-full h-full opacity-0 p-0 m-0 border-none"
        style={{ top: 0, left: 0, caretColor: 'transparent' }}
      />
    </div>
  );
});

CodeInput.displayName = 'CodeInput';

export default CodeInput;