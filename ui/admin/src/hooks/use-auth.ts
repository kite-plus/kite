import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost, apiPut } from '@/lib/api-client'
import { toast } from 'sonner'

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
      toast.success('登录成功', { description: '欢迎回来！' })
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

/** 个人资料更新输入 */
export interface ProfileInput {
  display_name: string
  email: string
  bio: string
  avatar: string
  website: string
  location: string
}

/** 个人资料响应 */
export interface ProfileOutput {
  username: string
  display_name: string
  email: string
  bio: string
  avatar: string
  website: string
  location: string
}

/**
 * 更新个人资料 Hook
 * 调用 PUT /admin/profile
 */
export function useUpdateProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: ProfileInput) =>
      apiPut<ProfileOutput>('/admin/profile', data),
    onSuccess: () => {
      // 刷新当前用户信息以同步侧边栏等展示
      queryClient.invalidateQueries({ queryKey: ['auth', 'me'] })
    },
  })
}

/** 修改密码输入 */
export interface ChangePasswordInput {
  old_password: string
  new_password: string
}

/**
 * 修改密码 Hook
 * 调用 PUT /admin/profile/password
 */
export function useChangePassword() {
  return useMutation({
    mutationFn: (data: ChangePasswordInput) =>
      apiPut<{ changed: boolean }>('/admin/profile/password', data),
  })
}
