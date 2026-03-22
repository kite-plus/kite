import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiGet, apiPut } from '@/lib/api-client'

/** 导航菜单项（最多二级） */
export interface NavMenuItem {
  title: string
  url: string
  icon?: string
  openInNewTab: boolean
  children?: NavMenuItem[]
}

/** 获取导航菜单列表 */
export function useMenuList() {
  return useQuery<NavMenuItem[]>({
    queryKey: ['navMenus'],
    queryFn: () => apiGet<NavMenuItem[]>('/admin/settings/nav-menus'),
  })
}

/** 保存导航菜单列表 */
export function useSaveMenus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (menus: NavMenuItem[]) =>
      apiPut<NavMenuItem[]>('/admin/settings/nav-menus', menus),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['navMenus'] })
    },
  })
}
