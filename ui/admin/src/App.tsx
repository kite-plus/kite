import { BrowserRouter, Routes, Route, Navigate } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/sonner'
import { ThemeProvider } from '@/context/theme-provider'
import { AdminLayout } from '@/layouts/AdminLayout'
import { useCurrentUser } from '@/hooks/use-auth'
import { lazy, Suspense } from 'react'
import { Loader2 } from 'lucide-react'

// 路由懒加载 — 按页面分割 chunk
const LoginPage = lazy(() => import('@/pages/LoginPage').then(m => ({ default: m.LoginPage })))
const DashboardPage = lazy(() => import('@/pages/DashboardPage').then(m => ({ default: m.DashboardPage })))
const PostsPage = lazy(() => import('@/pages/PostsPage').then(m => ({ default: m.PostsPage })))
const PostEditorPage = lazy(() => import('@/pages/PostEditorPage').then(m => ({ default: m.PostEditorPage })))
const PagesPage = lazy(() => import('@/pages/PagesPage').then(m => ({ default: m.PagesPage })))
const PageEditorPage = lazy(() => import('@/pages/PageEditorPage').then(m => ({ default: m.PageEditorPage })))
const CategoriesPage = lazy(() => import('@/pages/CategoriesPage').then(m => ({ default: m.CategoriesPage })))
const TagsPage = lazy(() => import('@/pages/TagsPage').then(m => ({ default: m.TagsPage })))
const CommentsPage = lazy(() => import('@/pages/CommentsPage').then(m => ({ default: m.CommentsPage })))
const FriendLinksPage = lazy(() => import('@/pages/FriendLinksPage').then(m => ({ default: m.FriendLinksPage })))
const SettingsPage = lazy(() => import('@/pages/SettingsPage').then(m => ({ default: m.SettingsPage })))
const ProfilePage = lazy(() => import('@/pages/ProfilePage').then(m => ({ default: m.ProfilePage })))
const NotificationsPage = lazy(() => import('@/pages/NotificationsPage').then(m => ({ default: m.NotificationsPage })))
const MenusPage = lazy(() => import('@/pages/MenusPage').then(m => ({ default: m.MenusPage })))

/** TanStack Query 客户端 */
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

/** 页面加载 Loading */
function PageLoader() {
  return (
    <div className='min-h-[60vh] flex items-center justify-center'>
      <Loader2 className='w-6 h-6 animate-spin text-muted-foreground' />
    </div>
  )
}

/**
 * 受保护路由：未登录则重定向到 /login
 */
function ProtectedRoutes() {
  const { data: currentUser, isLoading, isError } = useCurrentUser()

  if (isLoading) {
    return (
      <div className='min-h-screen flex items-center justify-center'>
        <div className='text-center'>
          <Loader2 className='w-6 h-6 animate-spin text-muted-foreground mx-auto' />
          <p className='text-sm text-muted-foreground mt-4'>正在验证身份…</p>
        </div>
      </div>
    )
  }

  if (isError || !currentUser || (currentUser.authEnabled && !currentUser.authenticated)) {
    return <Navigate to='/login' replace />
  }

  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        <Route element={<AdminLayout />}>
          <Route index element={<DashboardPage />} />
          <Route path='posts' element={<PostsPage />} />
          <Route path='posts/new' element={<PostEditorPage />} />
          <Route path='posts/:id/edit' element={<PostEditorPage />} />
          <Route path='pages' element={<PagesPage />} />
          <Route path='pages/new' element={<PageEditorPage />} />
          <Route path='pages/:id/edit' element={<PageEditorPage />} />
          <Route path='categories' element={<CategoriesPage />} />
          <Route path='tags' element={<TagsPage />} />
          <Route path='comments' element={<CommentsPage />} />
          <Route path='links' element={<FriendLinksPage />} />
          <Route path='settings' element={<SettingsPage />} />
          <Route path='menus' element={<MenusPage />} />
          <Route path='profile' element={<ProfilePage />} />
          <Route path='notifications' element={<NotificationsPage />} />
        </Route>
      </Routes>
    </Suspense>
  )
}

/**
 * 登录页路由守卫：已登录则重定向到首页
 */
function LoginGuard() {
  const { data: currentUser, isLoading } = useCurrentUser()

  if (isLoading) {
    return (
      <div className='min-h-screen flex items-center justify-center'>
        <Loader2 className='w-6 h-6 animate-spin text-muted-foreground' />
      </div>
    )
  }

  if (currentUser?.authenticated) {
    return <Navigate to='/' replace />
  }

  return (
    <Suspense fallback={<PageLoader />}>
      <LoginPage />
    </Suspense>
  )
}

/**
 * 应用根组件
 */
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <TooltipProvider>
          <BrowserRouter basename={import.meta.env.BASE_URL.replace(/\/$/, '')}>
            <Routes>
              <Route path='/login' element={<LoginGuard />} />
              <Route path='/*' element={<ProtectedRoutes />} />
            </Routes>
          </BrowserRouter>
          <Toaster position="top-center" />
        </TooltipProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
