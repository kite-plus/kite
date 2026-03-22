import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api-client'
import type { Tag } from '@/types/tag'

export interface TagListResponse {
  total: number
  items: Tag[]
}

/**
 * 获取标签列表 Hook
 */
export function useTagList(keyword?: string) {
  return useQuery({
    queryKey: ['tagList', keyword],
    queryFn: () => {
      const params = keyword ? `?keyword=${encodeURIComponent(keyword)}` : ''
      return apiGet<TagListResponse>(`/admin/tags${params}`).then((res) => res.items || [])
    },
  })
}

/**
 * 创建标签 Hook
 */
export function useCreateTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Pick<Tag, 'name' | 'slug'>) =>
      apiPost<Tag>('/admin/tags', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tagList'] })
    },
  })
}

/**
 * 更新标签 Hook
 */
export function useUpdateTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: Pick<Tag, 'id' | 'name' | 'slug'>) =>
      apiPut<Tag>(`/admin/tags/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tagList'] })
    },
  })
}

/**
 * 删除标签 Hook
 */
export function useDeleteTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/admin/tags/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tagList'] })
    },
  })
}
