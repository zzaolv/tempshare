// frontend/src/store/uploaderStore.ts
import { create } from 'zustand';

// 定义 state 和 actions 的类型
interface UploaderState {
  isInputMode: boolean;
  code: string;
  setIsInputMode: (isInput: boolean) => void;
  setCode: (newCode: string) => void;
  resetCode: () => void;
}

// 创建 store
export const useUploaderStore = create<UploaderState>((set) => ({
  // 初始状态
  isInputMode: false,
  code: '',

  // Actions
  setIsInputMode: (isInput) => set({ isInputMode: isInput }),
  setCode: (newCode) => set({ code: newCode }),
  resetCode: () => set({ code: '' }),
}));