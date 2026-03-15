import { Outlet } from 'react-router'
import { Sidebar } from '@/components/sidebar'
import { Header } from '@/components/header'

/**
 * Admin 全局布局组件
 * 三段式结构：左侧边栏 + 右侧（顶部 Header + 灰底主内容区）
 * 内容区使用灰色底衬，让白色卡片有层次感
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

        {/* 主内容区 - 灰底白卡片 */}
        <main className="flex-1 overflow-y-auto bg-[var(--kite-bg)] p-6">
          <div className="mx-auto max-w-[1200px]">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  )
}
