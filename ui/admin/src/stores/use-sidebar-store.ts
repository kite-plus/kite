import { create } from 'zustand'

/**
 * 侧边栏状态管理
 */
interface SidebarState {
  /** 是否折叠 */
  isCollapsed: boolean
  /** 切换折叠状态 */
  toggle: () => void
  /** 设置折叠状态 */
  setCollapsed: (collapsed: boolean) => void
}

export const useSidebarStore = create<SidebarState>((set) => ({
  isCollapsed: false,
  toggle: () => set((state) => ({ isCollapsed: !state.isCollapsed })),
  setCollapsed: (collapsed) => set({ isCollapsed: collapsed }),
}))
