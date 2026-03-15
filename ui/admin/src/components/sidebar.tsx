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
  Search,
  LogOut,
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

/** 导航分组定义 */
interface NavGroup {
  title: string
  items: NavItem[]
}

/** 侧边栏导航分组配置 */
const navGroups: NavGroup[] = [
  {
    title: '',
    items: [
      { label: '仪表盘', icon: LayoutDashboard, path: '/' },
    ],
  },
  {
    title: '内容',
    items: [
      { label: '文章', icon: FileText, path: '/posts' },
      { label: '分类', icon: FolderTree, path: '/categories' },
      { label: '标签', icon: Tag, path: '/tags' },
      { label: '评论', icon: MessageSquare, path: '/comments' },
    ],
  },
  {
    title: '管理',
    items: [
      { label: '友链', icon: Link2, path: '/links' },
      { label: '设置', icon: Settings, path: '/settings' },
    ],
  },
]

/**
 * 侧边栏组件
 * 分组导航、搜索框、底部用户信息
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
      <div className="flex h-14 items-center px-4">
        <Wind className="h-5 w-5 text-[var(--kite-accent)] flex-shrink-0" strokeWidth={2} />
        {!isCollapsed && (
          <span className="ml-2.5 text-base font-bold tracking-tight text-[var(--kite-text-heading)]">
            Kite
          </span>
        )}
      </div>

      {/* 搜索框 */}
      {!isCollapsed && (
        <div className="px-3 pb-2">
          <div className="flex h-8 items-center gap-2 border border-[var(--kite-border)] bg-[var(--kite-bg-hover)] px-2.5 text-xs text-[var(--kite-text-muted)]">
            <Search className="h-3.5 w-3.5 flex-shrink-0" strokeWidth={1.5} />
            <span>搜索…</span>
            <kbd className="ml-auto border border-[var(--kite-border)] bg-white px-1.5 py-0.5 text-[10px] font-medium text-[var(--kite-text-muted)]">
              ⌘K
            </kbd>
          </div>
        </div>
      )}

      {/* 导航列表 - 分组 */}
      <nav className="flex-1 overflow-y-auto px-3 py-2">
        {navGroups.map((group, gi) => (
          <div key={gi} className={cn(gi > 0 && 'mt-4')}>
            {/* 分组标题 */}
            {group.title && !isCollapsed && (
              <p className="mb-1.5 px-2 text-[11px] font-semibold uppercase tracking-widest text-[var(--kite-text-muted)]">
                {group.title}
              </p>
            )}
            {group.title && isCollapsed && gi > 0 && (
              <div className="mx-2 mb-2 border-t border-[var(--kite-border)]" />
            )}
            <ul className="space-y-0.5">
              {group.items.map((item) => {
                const isActive = location.pathname === item.path
                const Icon = item.icon
                return (
                  <li key={item.path}>
                    <Link
                      to={item.path}
                      className={cn(
                        'group flex items-center h-9 px-2.5 text-[13px] transition-colors duration-100 no-underline',
                        isActive
                          ? 'bg-[var(--kite-bg-hover)] text-[var(--kite-text-heading)] font-medium border border-[var(--kite-border)]'
                          : 'border border-transparent text-[var(--kite-text)] hover:bg-[var(--kite-bg-hover)] hover:text-[var(--kite-text-heading)]',
                        isCollapsed && 'justify-center px-0'
                      )}
                    >
                      <Icon
                        className={cn(
                          'h-[18px] w-[18px] flex-shrink-0',
                          isActive
                            ? 'text-[var(--kite-text-heading)]'
                            : 'text-[var(--kite-text-muted)] group-hover:text-[var(--kite-text-heading)]'
                        )}
                        strokeWidth={1.5}
                      />
                      {!isCollapsed && (
                        <span className="ml-2.5 truncate">{item.label}</span>
                      )}
                    </Link>
                  </li>
                )
              })}
            </ul>
          </div>
        ))}
      </nav>

      {/* 底部用户信息 */}
      <div className="border-t border-[var(--kite-border)] p-3">
        {isCollapsed ? (
          <div className="flex h-9 w-full items-center justify-center border border-[var(--kite-border)] bg-[var(--kite-bg-hover)] text-xs font-semibold text-[var(--kite-text-heading)]">
            A
          </div>
        ) : (
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center border border-[var(--kite-border)] bg-[var(--kite-bg-hover)] text-xs font-semibold text-[var(--kite-text-heading)]">
              A
            </div>
            <div className="min-w-0 flex-1">
              <p className="truncate text-[13px] font-medium text-[var(--kite-text-heading)]">Admin</p>
              <p className="truncate text-[11px] text-[var(--kite-text-muted)]">超级管理员</p>
            </div>
            <button className="flex h-7 w-7 items-center justify-center text-[var(--kite-text-muted)] hover:text-[var(--kite-text-heading)] cursor-pointer" title="退出登录">
              <LogOut className="h-3.5 w-3.5" strokeWidth={1.5} />
            </button>
          </div>
        )}
      </div>
    </aside>
  )
}
