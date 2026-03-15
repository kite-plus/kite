import { Outlet } from 'react-router'
import { Layout } from '@douyinfe/semi-ui'
import { Sidebar } from '@/components/sidebar'
import { Header } from '@/components/header'
import { CommandPalette } from '@/components/CommandPalette'

const { Content } = Layout

/**
 * Admin 全局布局组件
 * 三段式结构：左侧边栏 + 右侧（顶部 Header + 灰底主内容区）
 */
export function AdminLayout() {
  return (
    <Layout style={{ height: '100vh' }}>
      <Sidebar />
      <Layout>
        <Header />
        <Content style={{ overflow: 'auto', background: 'var(--semi-color-bg-1)', padding: 24 }}>
          <div style={{ maxWidth: 1200, margin: '0 auto' }}>
            <Outlet />
          </div>
        </Content>
      </Layout>
      {/* 全局命令面板 */}
      <CommandPalette />
    </Layout>
  )
}
