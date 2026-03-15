import { Outlet } from 'react-router'
import { Sidebar } from '@/components/sidebar'
import { Header } from '@/components/header'

/**
 * Admin 全局布局组件
 * 三段式结构：左侧边栏 + 右侧（顶部 Header + 主内容区）
 * 特性：无圆角、硬线条分隔、高留白内容区
 */
export function AdminLayout() {
  return (
    <div className="flex h-screen overflow-hidden bg-[var(--kite-bg)]">
      {/* 左侧边栏 - 固定 */}
      <Sidebar />

      {/* 右侧主区域 */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* 顶部 Header */}
        <Header />

        {/* 主内容区 - 高留白 */}
        <main className="flex-1 overflow-y-auto p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
