import { BrowserRouter, Routes, Route } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AdminLayout } from '@/layouts/AdminLayout'
import { DashboardPage } from '@/pages/DashboardPage'
import { PostsPage } from '@/pages/PostsPage'
import { CategoriesPage } from '@/pages/CategoriesPage'
import { SettingsPage } from '@/pages/SettingsPage'
import { TagsPage } from '@/pages/TagsPage'
import { CommentsPage } from '@/pages/CommentsPage'

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
            <Route path="posts" element={<PostsPage />} />
            <Route path="categories" element={<CategoriesPage />} />
            <Route path="tags" element={<TagsPage />} />
            <Route path="comments" element={<CommentsPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  )
}

export default App

