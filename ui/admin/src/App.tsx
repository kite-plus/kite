import { BrowserRouter, Routes, Route, Navigate } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AdminLayout } from '@/layouts/AdminLayout'
import { LoginPage } from '@/pages/LoginPage'
import { DashboardPage } from '@/pages/DashboardPage'
import { PostsPage } from '@/pages/PostsPage'
import { CategoriesPage } from '@/pages/CategoriesPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { TagsPage } from '@/pages/TagsPage'
import { CommentsPage } from '@/pages/CommentsPage'
import { FriendLinksPage } from '@/pages/FriendLinksPage'
import { PostEditorPage } from '@/pages/PostEditorPage'
import { PagesPage } from '@/pages/PagesPage'
import { PageEditorPage } from '@/pages/PageEditorPage'
import { useCurrentUser } from '@/hooks/use-auth'
import { Typography, Spin } from '@douyinfe/semi-ui'

const { Text } = Typography

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
 * 受保护路由：未登录则重定向到 /login
 */
function ProtectedRoutes() {
  const { data: currentUser, isLoading, isError } = useCurrentUser()

  // 加载中
  if (isLoading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{ textAlign: 'center' }}>
          <Spin size="large" />
          <Text type="tertiary" style={{ display: 'block', marginTop: 16 }}>正在验证身份…</Text>
        </div>
      </div>
    )
  }

  // 未登录或鉴权失败
  if (isError || !currentUser || (currentUser.authEnabled && !currentUser.authenticated)) {
    return <Navigate to="/login" replace />
  }

  // 已登录，渲染管理后台
  return (
    <Routes>
      <Route element={<AdminLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="posts" element={<PostsPage />} />
        <Route path="posts/new" element={<PostEditorPage />} />
        <Route path="posts/:id/edit" element={<PostEditorPage />} />
        <Route path="pages" element={<PagesPage />} />
        <Route path="pages/new" element={<PageEditorPage />} />
        <Route path="pages/:id/edit" element={<PageEditorPage />} />
        <Route path="categories" element={<CategoriesPage />} />
        <Route path="tags" element={<TagsPage />} />
        <Route path="comments" element={<CommentsPage />} />
        <Route path="links" element={<FriendLinksPage />} />
        <Route path="settings" element={<SettingsPage />} />
      </Route>
    </Routes>
  )
}

/**
 * 登录页路由守卫：已登录则重定向到首页
 */
function LoginGuard() {
  const { data: currentUser, isLoading } = useCurrentUser()

  if (isLoading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Spin size="large" />
      </div>
    )
  }

  // 已登录
  if (currentUser?.authenticated) {
    return <Navigate to="/" replace />
  }

  return <LoginPage />
}

/**
 * 应用根组件
 */
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter basename={import.meta.env.BASE_URL.replace(/\/$/, '')}>
        <Routes>
          <Route path="/login" element={<LoginGuard />} />
          <Route path="/*" element={<ProtectedRoutes />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App
