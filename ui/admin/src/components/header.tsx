import { useRef } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { Layout, Breadcrumb, Button, Avatar, Dropdown, Badge, Tooltip, Typography } from '@douyinfe/semi-ui'
import { IconSetting, IconMenu, IconBell, IconExternalOpen, IconMoon, IconSun, IconLanguage, IconExit } from '@douyinfe/semi-icons'
import { useSidebarStore } from '@/stores/use-sidebar-store'
import { useThemeStore } from '@/stores/use-theme-store'
import { useCurrentUser, useLogout } from '@/hooks/use-auth'

const { Header: SemiHeader } = Layout
const { Text } = Typography

/** 面包屑路径映射 */
const breadcrumbMap: Record<string, string> = {
  '/': '仪表盘',
  '/posts': '文章管理',
  '/posts/new': '新建文章',
  '/categories': '分类管理',
  '/tags': '标签管理',
  '/comments': '评论管理',
  '/links': '友链管理',
  '/settings': '系统设置',
}

/**
 * 顶部 Header 组件
 * 面包屑导航 + 右侧快捷操作区
 */
export function Header() {
  const { toggle } = useSidebarStore()
  const location = useLocation()
  const navigate = useNavigate()
  const headerRef = useRef<HTMLDivElement>(null)
  const { isDark, toggle: toggleTheme } = useThemeStore()
  const { data: currentUser } = useCurrentUser()
  const logoutMutation = useLogout()

  const displayName = currentUser?.user?.displayName || currentUser?.user?.username || 'Admin'

  /* 解析面包屑 */
  const pathSegments = location.pathname.split('/').filter(Boolean)
  const currentLabel = breadcrumbMap[location.pathname]
  const isEditPost = pathSegments[0] === 'posts' && pathSegments.length >= 3

  /** 让 Tooltip / Dropdown 弹出层相对 Header 定位，避免飘到左上角 */
  const getContainer = () => headerRef.current || document.body

  return (
    <SemiHeader
      style={{
        background: 'var(--semi-color-bg-0)',
        borderBottom: '1px solid var(--semi-color-border)',
        padding: '0 20px',
        height: 56,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        position: 'relative',
      }}
    >
      {/* 用于弹出层定位的容器 ref */}
      <div ref={headerRef} style={{ position: 'absolute', inset: 0, pointerEvents: 'none' }} />

      {/* 左侧：菜单 + 面包屑 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <Tooltip content="收起/展开侧边栏" position="bottom" getPopupContainer={getContainer}>
          <Button
            icon={<IconMenu />}
            theme="borderless"
            type="tertiary"
            onClick={toggle}
            style={{ width: 36, height: 36 }}
          />
        </Tooltip>

        <Breadcrumb style={{ marginLeft: 4 }}>
          <Breadcrumb.Item onClick={() => navigate('/')}>Kite</Breadcrumb.Item>
          {isEditPost ? (
            <>
              <Breadcrumb.Item onClick={() => navigate('/posts')}>文章管理</Breadcrumb.Item>
              <Breadcrumb.Item>编辑文章</Breadcrumb.Item>
            </>
          ) : (
            <Breadcrumb.Item>{currentLabel || '页面'}</Breadcrumb.Item>
          )}
        </Breadcrumb>
      </div>

      {/* 右侧：操作区 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
        {/* 预览站点 */}
        <Tooltip content="预览站点" position="bottom" getPopupContainer={getContainer}>
          <Button
            icon={<IconExternalOpen />}
            theme="borderless"
            type="tertiary"
            style={{ width: 36, height: 36 }}
            onClick={() => window.open('/', '_blank')}
          />
        </Tooltip>

        {/* 主题切换 */}
        <Tooltip content={isDark ? '切换到亮色模式' : '切换到暗色模式'} position="bottom" getPopupContainer={getContainer}>
          <Button
            icon={isDark ? <IconSun /> : <IconMoon />}
            theme="borderless"
            type="tertiary"
            style={{ width: 36, height: 36 }}
            onClick={toggleTheme}
          />
        </Tooltip>

        {/* 通知 */}
        <Tooltip content="通知" position="bottom" getPopupContainer={getContainer}>
          <Badge count={3} type="danger" overflowCount={9}>
            <Button
              icon={<IconBell />}
              theme="borderless"
              type="tertiary"
              style={{ width: 36, height: 36 }}
            />
          </Badge>
        </Tooltip>

        {/* 分隔 */}
        <div style={{ width: 1, height: 24, background: 'var(--semi-color-border)', margin: '0 8px' }} />

        {/* 用户菜单 */}
        <Dropdown
          position="bottomRight"
          getPopupContainer={getContainer}
          render={
            <Dropdown.Menu>
              <Dropdown.Item icon={<IconSetting />} onClick={() => navigate('/settings')}>
                系统设置
              </Dropdown.Item>
              <Dropdown.Item icon={<IconLanguage />}>
                语言设置
              </Dropdown.Item>
              <Dropdown.Divider />
              <Dropdown.Item icon={<IconExit />} type="danger" onClick={() => logoutMutation.mutate()}>
                退出登录
              </Dropdown.Item>
            </Dropdown.Menu>
          }
        >
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', padding: '4px 8px', borderRadius: 6 }}>
            <Avatar size="small" color="blue" alt={displayName}>{displayName.charAt(0).toUpperCase()}</Avatar>
            <Text style={{ fontSize: 13, fontWeight: 500 }}>{displayName}</Text>
          </div>
        </Dropdown>
      </div>
    </SemiHeader>
  )
}
