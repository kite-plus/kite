import { Link, useLocation } from 'react-router'
import {
  LayoutDashboard,
  FileText,
  FolderTree,
  Tag,
  MessageSquare,
  Link2,
  Settings,
  Wind,
  type LucideIcon,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useSidebarStore } from '@/stores/use-sidebar-store'

/**
 * 导航项定义
 */
interface NavItem {
  label: string
  icon: LucideIcon
  path: string
}

/** 侧边栏导航配置 */
const navItems: NavItem[] = [
  { label: '仪表盘', icon: LayoutDashboard, path: '/' },
  { label: '文章', icon: FileText, path: '/posts' },
  { label: '分类', icon: FolderTree, path: '/categories' },
  { label: '标签', icon: Tag, path: '/tags' },
  { label: '评论', icon: MessageSquare, path: '/comments' },
  { label: '友链', icon: Link2, path: '/links' },
  { label: '设置', icon: Settings, path: '/settings' },
]

/**
 * 侧边栏组件
 * 特性：左侧 2px 高亮硬边框选中态、折叠模式、绝对直角
 */
export function Sidebar() {
  const location = useLocation()
  const { isCollapsed } = useSidebarStore()

  return (
    <aside
      className={cn(
        'h-screen border-r border-[var(--kite-border)] bg-[var(--kite-bg-sidebar)]',
        'flex flex-col transition-[width] duration-200 ease-in-out',
        isCollapsed ? 'w-16' : 'w-56'
      )}
    >
      {/* Logo 区域 */}
      <div className="flex h-14 items-center border-b border-[var(--kite-border)] px-4">
        <Wind className="h-5 w-5 text-[var(--kite-text-heading)] flex-shrink-0" strokeWidth={1.5} />
        {!isCollapsed && (
          <span className="ml-2.5 text-base font-semibold tracking-tight text-[var(--kite-text-heading)]">
            Kite
          </span>
        )}
      </div>

      {/* 导航列表 */}
      <nav className="flex-1 px-2 py-4">
        <ul className="space-y-0.5">
          {navItems.map((item) => {
            const isActive = location.pathname === item.path
            const Icon = item.icon

            return (
              <li key={item.path}>
                <Link
                  to={item.path}
                  className={cn(
                    'group flex items-center h-9 px-3 text-sm transition-colors duration-100',
                    'border-l-2 no-underline',
                    isActive
                      ? 'border-l-[var(--kite-accent)] bg-[var(--kite-bg-hover)] text-[var(--kite-text-heading)] font-medium'
                      : 'border-l-transparent text-[var(--kite-text-muted)] hover:bg-[var(--kite-bg-hover)] hover:text-[var(--kite-text-heading)]',
                    isCollapsed && 'justify-center px-0'
                  )}
                >
                  <Icon
                    className={cn(
                      'h-4 w-4 flex-shrink-0',
                      isActive
                        ? 'text-[var(--kite-text-heading)]'
                        : 'text-[var(--kite-text-muted)] group-hover:text-[var(--kite-text-heading)]'
                    )}
                    strokeWidth={1.5}
                  />
                  {!isCollapsed && (
                    <span className="ml-3 truncate">{item.label}</span>
                  )}
                </Link>
              </li>
            )
          })}
        </ul>
      </nav>

      {/* 底部版本信息 */}
      {!isCollapsed && (
        <div className="border-t border-[var(--kite-border)] px-4 py-3">
          <p className="text-xs text-[var(--kite-text-muted)]">Kite v0.1.0</p>
        </div>
      )}
    </aside>
  )
}
