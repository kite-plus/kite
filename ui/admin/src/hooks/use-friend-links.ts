import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mockFriendLinks } from '@/mocks/friend-links'
import type { FriendLink } from '@/types/friend-link'

/** 模拟延迟 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 内存中的友链副本 */
let linksStore = [...mockFriendLinks]

/**
 * Mock 获取友链列表
 */
async function fetchLinks(keyword?: string): Promise<FriendLink[]> {
  await delay(200)
  let result = [...linksStore]
  if (keyword) {
    const kw = keyword.toLowerCase()
    result = result.filter(
      (l) => l.name.toLowerCase().includes(kw) || l.url.toLowerCase().includes(kw) || l.description.toLowerCase().includes(kw)
    )
  }
  result.sort((a, b) => a.sortOrder - b.sortOrder)
  return result
}

/**
 * Mock 创建友链
 */
async function createLink(data: Pick<FriendLink, 'name' | 'url' | 'description'>): Promise<FriendLink> {
  await delay(300)
  const newLink: FriendLink = {
    id: crypto.randomUUID(),
    name: data.name,
    url: data.url,
    logo: '',
    description: data.description,
    status: 'pending',
    sortOrder: linksStore.length + 1,
    createdAt: new Date().toISOString(),
  }
  linksStore = [...linksStore, newLink]
  return newLink
}

/**
 * Mock 删除友链
 */
async function deleteLink(id: string): Promise<void> {
  await delay(200)
  linksStore = linksStore.filter((l) => l.id !== id)
}

/**
 * Mock 切换友链状态
 */
async function toggleLinkStatus(data: { id: string; status: FriendLink['status'] }): Promise<void> {
  await delay(200)
  linksStore = linksStore.map((l) =>
    l.id === data.id ? { ...l, status: data.status } : l
  )
}

/**
 * 获取友链列表 Hook
 */
export function useFriendLinks(keyword?: string) {
  return useQuery({
    queryKey: ['friendLinks', keyword],
    queryFn: () => fetchLinks(keyword),
  })
}

/**
 * 创建友链 Hook
 */
export function useCreateFriendLink() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createLink,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friendLinks'] })
    },
  })
}

/**
 * 删除友链 Hook
 */
export function useDeleteFriendLink() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: deleteLink,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friendLinks'] })
    },
  })
}

/**
 * 切换友链状态 Hook
 */
export function useToggleLinkStatus() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: toggleLinkStatus,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['friendLinks'] })
    },
  })
}
