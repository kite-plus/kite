import { BrowserRouter, Routes, Route } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AdminLayout } from '@/layouts/AdminLayout'
import { DashboardPage } from '@/pages/DashboardPage'

/** TanStack Query 客户端 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

/**
 * 应用根组件
 * 集成 QueryClientProvider + BrowserRouter + 路由配置
 */
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route element={<AdminLayout />}>
            <Route index element={<DashboardPage />} />
            {/* 后续页面路由在此扩展 */}
            <Route path="posts" element={<PlaceholderPage title="文章管理" />} />
            <Route path="categories" element={<PlaceholderPage title="分类管理" />} />
            <Route path="settings" element={<PlaceholderPage title="系统设置" />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

/** 通用占位页面 */
function PlaceholderPage({ title }: { title: string }) {
  return (
    <div>
      <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
        {title}
      </h1>
      <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
        此页面尚未实现
      </p>
    </div>
  )
}

export default App
