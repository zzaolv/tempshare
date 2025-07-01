// src/components/CodeInput.tsx
import React, { useRef, useImperativeHandle, forwardRef, useEffect } from 'react';

// 定义组件接收的 props 类型
interface CodeInputProps {
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
}

// 定义通过 ref 暴露给父组件的句柄类型
export interface CodeInputHandle {
  focus: () => void;
}

const CODE_LENGTH = 6;

// 使用 forwardRef 来让父组件可以获取到内部的 input 的 ref
const CodeInput = forwardRef<CodeInputHandle, CodeInputProps>(({ value, onChange, onComplete }, ref) => {
  const inputRef = useRef<HTMLInputElement>(null);

  // useImperativeHandle 可以让我们自定义暴露给父组件的 ref 值
  useImperativeHandle(ref, () => ({
    focus: () => {
      inputRef.current?.focus();
    },
  }));

  // 当输入值变化时，检查是否已满6位
  useEffect(() => {
    if (value.length === CODE_LENGTH && onComplete) {
      onComplete(value);
    }
  }, [value, onComplete]);

  // 点击容器时，自动聚焦到隐藏的输入框
  const handleContainerClick = () => {
    inputRef.current?.focus();
  };

  // 处理输入框内容变化
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    // 只允许字母和数字，自动转为大写，并限制长度
    const newValue = e.target.value.replace(/[^a-zA-Z0-9]/g, '').toUpperCase().slice(0, CODE_LENGTH);
    onChange(newValue);
  };

  // 将输入的字符串分割成字符数组，用于渲染
  const characters = value.split('');

  return (
    <div
      className="relative w-full h-16 flex items-center justify-center gap-2 md:gap-3 cursor-text"
      onClick={handleContainerClick}
    >
      {/* 渲染 6 个视觉上的方块 */}
      {Array.from({ length: CODE_LENGTH }).map((_, index) => {
        const hasCharacter = index < characters.length;
        const isCurrent = index === characters.length; // 当前光标所在的位置

        return (
          <div
            key={index}
            className={`
              w-10 h-14 md:w-12 md:h-16 flex items-center justify-center
              text-2xl md:text-3xl font-mono font-bold text-brand-dark
              bg-black/5 border-2 rounded-lg
              transition-all duration-200 ease-in-out
              ${isCurrent ? 'border-brand-cyan shadow-lg shadow-brand-cyan/20' : 'border-transparent'}
              ${hasCharacter ? 'border-black/10' : ''}
            `}
          >
            {characters[index] || ''}
            {/* 仅在当前输入位置显示一个闪烁的光标 */}
            {isCurrent && (
              <span className="absolute w-0.5 h-7 bg-brand-cyan animate-pulse rounded-full"></span>
            )}
          </div>
        );
      })}

      {/* 隐藏的、真正接收输入的 input 元素 */}
      <input
        ref={inputRef}
        type="text"
        value={value}
        onChange={handleInputChange}
        maxLength={CODE_LENGTH}
        // 使用 TailwindCSS 的 sr-only 类或自定义样式将其完全隐藏，但保留其功能
        className="absolute w-full h-full opacity-0 p-0 m-0 border-none"
        style={{ top: 0, left: 0, caretColor: 'transparent' }}
      />
    </div>
  );
});

// 添加 displayName，便于 React DevTools 调试
CodeInput.displayName = 'CodeInput';

export default CodeInput;