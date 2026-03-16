import { useQuery } from '@tanstack/react-query'
import { apiGet } from '@/lib/api-client'

/** 仪表盘统计数据 */
export interface DashboardStats {
  postCount: number
  categoryCount: number
  tagCount: number
  commentPending: number
}

/**
 * 获取仪表盘统计数据 Hook
 * 聚合多个 API 调用获取统计概览
 */
export function useDashboardStats() {
  return useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: async (): Promise<DashboardStats> => {
      // 并行请求所有统计数据
      const [posts, categories, tags, commentStats] = await Promise.all([
        apiGet<{ pagination: { total: number } }>('/admin/posts', { pageSize: 1 }),
        apiGet<{ pagination: { total: number } }>('/admin/categories', { pageSize: 1 }),
        apiGet<{ pagination: { total: number } }>('/admin/tags', { pageSize: 1 }),
        apiGet<{ pending: number }>('/admin/comments/stats'),
      ])
      return {
        postCount: posts.pagination.total,
        categoryCount: categories.pagination.total,
        tagCount: tags.pagination.total,
        commentPending: commentStats.pending,
      }
    },
  })
}

/** 最近文章类型 */
interface RecentPost {
  id: string
  title: string
  status: string
  createdAt: string
}

/**
 * 获取最近文章 Hook
 */
export function useRecentPosts(limit = 5) {
  return useQuery({
    queryKey: ['dashboard', 'recentPosts', limit],
    queryFn: async () => {
      const result = await apiGet<{ items: RecentPost[] }>('/admin/posts', { pageSize: limit })
      return result.items
    },
  })
}
