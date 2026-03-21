import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPost, apiPut, apiDelete } from '@/lib/api-client'
import type { Category } from '@/types/category'

/** 后端分类列表响应 */
interface CategoryListResponse {
  items: Category[]
  pagination: { page: number; pageSize: number; total: number }
}

/** 分类创建/更新的参数 */
interface CategoryFormData {
  name: string
  slug: string
  description?: string
  icon?: string
  parent_id?: string
}

/**
 * 获取分类列表 Hook
 */
export function useCategoryList(keyword?: string) {
  return useQuery({
    queryKey: ['categoryList', keyword],
    queryFn: async () => {
      const result = await apiGet<CategoryListResponse>('/admin/categories', {
        pageSize: 100,
        keyword,
      })
      return result.items
    },
  })
}

/**
 * 创建分类 Hook
 */
export function useCreateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CategoryFormData) =>
      apiPost<Category>('/admin/categories', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categoryList'] })
    },
  })
}

/**
 * 更新分类 Hook
 */
export function useUpdateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: CategoryFormData & { id: string }) =>
      apiPut<Category>(`/admin/categories/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categoryList'] })
    },
  })
}

/**
 * 删除分类 Hook
 */
export function useDeleteCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/admin/categories/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['categoryList'] })
    },
  })
}

