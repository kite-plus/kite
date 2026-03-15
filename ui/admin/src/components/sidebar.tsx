import { useLocation, useNavigate } from 'react-router'
import { Layout, Nav, Input, Avatar } from '@douyinfe/semi-ui'
import {
  IconHome,
  IconArticle,
  IconGridView,
  IconPriceTag,
  IconComment,
  IconLink,
  IconSetting,
  IconSearch,
  IconExit,
} from '@douyinfe/semi-icons'
import { useSidebarStore } from '@/stores/use-sidebar-store'

const { Sider } = Layout

/** 导航项配置 */
const navItems = [
  { itemKey: '/', text: '仪表盘', icon: <IconHome /> },
  { itemKey: '/posts', text: '文章', icon: <IconArticle /> },
  { itemKey: '/categories', text: '分类', icon: <IconGridView /> },
  { itemKey: '/tags', text: '标签', icon: <IconPriceTag /> },
  { itemKey: '/comments', text: '评论', icon: <IconComment /> },
  { itemKey: '/links', text: '友链', icon: <IconLink /> },
  { itemKey: '/settings', text: '设置', icon: <IconSetting /> },
]

/**
 * 侧边栏组件
 * 使用 Semi Nav 组件替代 shadcn Button/Tooltip
 */
export function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const { isCollapsed } = useSidebarStore()

  return (
    <Sider style={{ background: 'var(--semi-color-bg-0)', borderRight: '1px solid var(--semi-color-border)' }}>
      {/* 内层 flex 容器 — Sider 自身样式会被 Semi 覆盖，需要额外包一层 */}
      <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
        {/* Logo 区域 */}
        <div style={{ height: 56, display: 'flex', alignItems: 'center', padding: '0 20px', gap: 10, flexShrink: 0 }}>
          <span style={{ fontSize: 20, fontWeight: 700, letterSpacing: -0.5 }}>
            {isCollapsed ? 'K' : 'Kite'}
          </span>
        </div>

        {/* 搜索栏 */}
        {!isCollapsed && (
          <div style={{ padding: '0 12px 8px', flexShrink: 0 }}>
            <Input
              prefix={<IconSearch />}
              placeholder="搜索"
              suffix={<kbd style={{ fontSize: 10, color: 'var(--semi-color-text-2)', background: 'var(--semi-color-fill-0)', padding: '2px 6px', borderRadius: 4 }}>⌘K</kbd>}
              showClear={false}
            />
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

        {/* 底部用户信息 — 固定在底部 */}
        <div style={{ padding: '12px 16px', borderTop: '1px solid var(--semi-color-border)', display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
          <Avatar size="small" alt="A">A</Avatar>
          {!isCollapsed && (
            <>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 13, fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>Admin</div>
                <div style={{ fontSize: 11, color: 'var(--semi-color-text-2)' }}>超级管理员</div>
              </div>
              <IconExit style={{ cursor: 'pointer', color: 'var(--semi-color-text-2)' }} />
            </>
          )}
        </div>
      </div>
    </Sider>
  )
}
