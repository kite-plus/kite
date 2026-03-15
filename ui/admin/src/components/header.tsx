import { PanelLeftClose, PanelLeft, User } from 'lucide-react'
import { useLocation } from 'react-router'
import { useSidebarStore } from '@/stores/use-sidebar-store'

/**
 * 面包屑路径映射
 */
const breadcrumbMap: Record<string, string> = {
  '/': '仪表盘',
  '/posts': '文章管理',
  '/categories': '分类管理',
  '/tags': '标签管理',
  '/comments': '评论管理',
  '/links': '友链管理',
  '/settings': '系统设置',
}

/**
 * 顶部 Header 组件
 * 特性：折叠按钮、面包屑、用户头像占位，底部 1px 硬线条
 */
export function Header() {
  const { isCollapsed, toggle } = useSidebarStore()
  const location = useLocation()
  const currentLabel = breadcrumbMap[location.pathname] || '未知页面'

  return (
    <header className="flex h-14 items-center justify-between border-b border-[var(--kite-border)] bg-[var(--kite-bg)] px-6">
      {/* 左侧：折叠按钮 + 面包屑 */}
      <div className="flex items-center gap-4">
        <button
          onClick={toggle}
          className="flex h-8 w-8 items-center justify-center border border-[var(--kite-border)] bg-transparent text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:text-[var(--kite-text-heading)] cursor-pointer"
          aria-label={isCollapsed ? '展开侧边栏' : '折叠侧边栏'}
        >
          {isCollapsed ? (
            <PanelLeft className="h-4 w-4" strokeWidth={1.5} />
          ) : (
            <PanelLeftClose className="h-4 w-4" strokeWidth={1.5} />
          )}
        </button>

        {/* 面包屑 */}
        <nav className="flex items-center text-sm">
          <span className="text-[var(--kite-text-muted)]">Kite</span>
          <span className="mx-2 text-[var(--kite-text-muted)]">/</span>
          <span className="font-medium text-[var(--kite-text-heading)]">
            {currentLabel}
          </span>
        </nav>
      </div>

      {/* 右侧：用户头像占位 */}
      <div className="flex items-center">
        <div className="flex h-8 w-8 items-center justify-center border border-[var(--kite-border)] text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-[var(--kite-border-hover)]">
          <User className="h-4 w-4" strokeWidth={1.5} />
        </div>
      </div>
    </header>
  )
}
