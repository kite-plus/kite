import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost } from '@/lib/api-client'

/** 管理员个人资料 */
export interface AdminProfile {
  username: string
  displayName: string
  email: string
  bio: string
  avatar: string
  website: string
  location: string
}

/** 当前用户信息 */
export interface AdminCurrentUser {
  authEnabled: boolean
  authenticated: boolean
  user: AdminProfile
  sessionExpires: string | null
}

/**
 * 获取当前登录状态 Hook
 * 调用 GET /admin/auth/me
 */
export function useCurrentUser() {
  return useQuery({
    queryKey: ['auth', 'me'],
    queryFn: () => apiGet<AdminCurrentUser>('/admin/auth/me'),
    retry: false,
    refetchOnWindowFocus: true,
  })
}

/**
 * 登录 Hook
 * 调用 POST /admin/auth/login，后端通过 Set-Cookie 设置 session
 */
export function useLogin() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { username: string; password: string }) =>
      apiPost<AdminCurrentUser>('/admin/auth/login', data),
    onSuccess: (data) => {
      queryClient.setQueryData(['auth', 'me'], data)
    },
  })
}

/**
 * 登出 Hook
 * 调用 POST /admin/auth/logout，后端清除 cookie
 */
export function useLogout() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => apiPost<void>('/admin/auth/logout'),
    onSuccess: () => {
      queryClient.setQueryData(['auth', 'me'], null)
      queryClient.clear()
    },
  })
}
