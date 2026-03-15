import { useLocation, useNavigate } from 'react-router'
import { Layout, Breadcrumb, Button } from '@douyinfe/semi-ui'
import { IconSetting, IconMenu } from '@douyinfe/semi-icons'
import { useSidebarStore } from '@/stores/use-sidebar-store'

const { Header: SemiHeader } = Layout

/** 面包屑路径映射 */
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
 * 使用 Semi Breadcrumb + Button
 */
export function Header() {
  const { toggle } = useSidebarStore()
  const location = useLocation()
  const navigate = useNavigate()
  const currentLabel = breadcrumbMap[location.pathname] || '未知页面'

  return (
    <SemiHeader style={{ background: '#fff', borderBottom: '1px solid var(--semi-color-border)', padding: '0 16px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
        <Button icon={<IconMenu />} theme="borderless" onClick={toggle} />
        <Breadcrumb>
          <Breadcrumb.Item>Kite</Breadcrumb.Item>
          <Breadcrumb.Item>{currentLabel}</Breadcrumb.Item>
        </Breadcrumb>
      </div>
      <div>
        <Button icon={<IconSetting />} theme="borderless" onClick={() => navigate('/settings')} />
      </div>
    </SemiHeader>
  )
}
