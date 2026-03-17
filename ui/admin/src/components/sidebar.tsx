import { useLocation, useNavigate } from 'react-router'
import { Layout, Nav, Avatar } from '@douyinfe/semi-ui'
import {
  IconHome,
  IconArticle,
  IconCopy,
  IconGridView,
  IconPriceTag,
  IconComment,
  IconLink,
  IconSetting,
  IconSearch,
  IconExit,
} from '@douyinfe/semi-icons'
import { useSidebarStore } from '@/stores/use-sidebar-store'
import { useCurrentUser, useLogout } from '@/hooks/use-auth'

const { Sider } = Layout

/** 导航项配置 */
const navItems = [
  { itemKey: '/', text: '仪表盘', icon: <IconHome /> },
  { itemKey: '/posts', text: '文章', icon: <IconArticle /> },
  { itemKey: '/pages', text: '页面', icon: <IconCopy /> },
  { itemKey: '/categories', text: '分类', icon: <IconGridView /> },
  { itemKey: '/tags', text: '标签', icon: <IconPriceTag /> },
  { itemKey: '/comments', text: '评论', icon: <IconComment /> },
  { itemKey: '/links', text: '友链', icon: <IconLink /> },
  { itemKey: '/settings', text: '设置', icon: <IconSetting /> },
]

/**
 * 侧边栏组件
 * 使用 Semi Nav 组件，通过 CSS 类名适配暗色模式
 */
export function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const { isCollapsed } = useSidebarStore()
  const { data: currentUser } = useCurrentUser()
  const logoutMutation = useLogout()

  // 取实际登录用户信息
  const displayName = currentUser?.user?.displayName || currentUser?.user?.username || 'Admin'
  const username = currentUser?.user?.username || ''

  return (
    <Sider className="kite-sider">
      {/* 内层 flex 容器 */}
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Logo 区域 */}
        <div className="sidebar-logo">
          <span className="sidebar-logo-text">
            {isCollapsed ? 'K' : 'Kite'}
          </span>
        </div>

        {/* 搜索栏 — 点击打开命令面板 */}
        {!isCollapsed && (
          <div style={{ padding: '0 12px 8px', flexShrink: 0 }}>
            <div
              className="sidebar-search"
              onClick={() => window.dispatchEvent(new KeyboardEvent('keydown', { key: 'k', metaKey: true }))}
            >
              <IconSearch style={{ fontSize: 14 }} />
              <span style={{ flex: 1 }}>搜索</span>
              <kbd>⌘K</kbd>
            </div>
          </div>
        )}

        {/* 导航 — flex:1 填充剩余空间 */}
        <Nav
          items={navItems}
          selectedKeys={[location.pathname]}
          onSelect={({ itemKey }) => navigate(itemKey as string)}
          isCollapsed={isCollapsed}
          style={{ flex: 1, overflow: 'auto' }}
        />

        {/* 底部用户信息 — 使用实际登录用户数据 */}
        <div className="sidebar-footer">
          <Avatar size="small" color="blue" alt={displayName}>
            {displayName.charAt(0).toUpperCase()}
          </Avatar>
          {!isCollapsed && (
            <>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="sidebar-footer-name">{displayName}</div>
                <div className="sidebar-footer-role">{username}</div>
              </div>
              <IconExit
                className="sidebar-footer-exit"
                onClick={() => logoutMutation.mutate()}
              />
            </>
          )}
        </div>
      </div>
    </Sider>
  )
}
