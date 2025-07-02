// src/hooks/useOnClickOutside.ts
import { useEffect, type RefObject } from 'react';

type Event = MouseEvent | TouchEvent;

const useOnClickOutside = <T extends HTMLElement = HTMLElement>(
  ref: RefObject<T | null>,
  handler: (event: Event) => void
) => {
  useEffect(() => {
    const listener = (event: Event) => {
      const el = ref.current;
      // 如果点击发生在 ref 元素或其子元素内部，则不执行任何操作
      if (!el || el.contains(event.target as Node)) {
        return;
      }
      handler(event);
    };

    // 添加 mousedown 和 touchstart 事件监听
    document.addEventListener('mousedown', listener);
    document.addEventListener('touchstart', listener);

    // 清理函数：在组件卸载时移除事件监听
    return () => {
      document.removeEventListener('mousedown', listener);
      document.removeEventListener('touchstart', listener);
    };
  }, [ref, handler]); // 仅在 ref 或 handler 变化时重新运行 effect
};

export default useOnClickOutside;