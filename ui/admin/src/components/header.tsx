import { PanelLeftClose, PanelLeft, Settings } from 'lucide-react'
import { useLocation, useNavigate } from 'react-router'
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
 * 页面图标前缀 emoji
 */
const pageEmoji: Record<string, string> = {
  '/': '📊',
  '/posts': '📝',
  '/categories': '📂',
  '/tags': '🏷️',
  '/comments': '💬',
  '/links': '🔗',
  '/settings': '⚙️',
}

/**
 * 顶部 Header 组件
 * 扁平化硬线条，干净留白
 */
export function Header() {
  const { isCollapsed, toggle } = useSidebarStore()
  const location = useLocation()
  const navigate = useNavigate()
  const currentLabel = breadcrumbMap[location.pathname] || '未知页面'
  const emoji = pageEmoji[location.pathname] || '📄'

  return (
    <header className="flex h-14 items-center justify-between border-b border-[var(--kite-border)] bg-[var(--kite-bg)] px-6">
      {/* 左侧：折叠按钮 + 面包屑 */}
      <div className="flex items-center gap-3">
        <button
          onClick={toggle}
          className="flex h-8 w-8 items-center justify-center text-[var(--kite-text-muted)] transition-colors duration-100 hover:text-[var(--kite-text-heading)] cursor-pointer"
          aria-label={isCollapsed ? '展开侧边栏' : '折叠侧边栏'}
        >
          {isCollapsed ? (
            <PanelLeft className="h-4 w-4" strokeWidth={1.5} />
          ) : (
            <PanelLeftClose className="h-4 w-4" strokeWidth={1.5} />
          )}
        </button>

        {/* 分隔线 */}
        <div className="h-4 w-px bg-[var(--kite-border)]" />

        {/* 面包屑 */}
        <nav className="flex items-center gap-2 text-sm">
          <span className="text-base">{emoji}</span>
          <span className="font-medium text-[var(--kite-text-heading)]">
            {currentLabel}
          </span>
        </nav>
      </div>

      {/* 右侧：设置快捷入口 */}
      <div className="flex items-center gap-2">
        <button
          onClick={() => navigate('/settings')}
          className="flex h-8 w-8 items-center justify-center text-[var(--kite-text-muted)] transition-colors duration-100 hover:text-[var(--kite-text-heading)] cursor-pointer"
          title="设置"
        >
          <Settings className="h-4 w-4" strokeWidth={1.5} />
        </button>
      </div>
    </header>
  )
}
