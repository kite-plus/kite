import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mockTags } from '@/mocks/tags'
import type { Tag } from '@/types/tag'

/** 模拟延迟 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 内存中的标签副本 */
let tagsStore = [...mockTags]

/**
 * Mock 获取标签列表
 */
async function fetchTags(keyword?: string): Promise<Tag[]> {
  await delay(200)
  let result = [...tagsStore]
  if (keyword) {
    const kw = keyword.toLowerCase()
    result = result.filter(
      (t) => t.name.toLowerCase().includes(kw) || t.slug.toLowerCase().includes(kw)
    )
  }
  // 按文章数倒序
  result.sort((a, b) => b.postCount - a.postCount)
  return result
}

/**
 * Mock 创建标签
 */
async function createTag(data: Pick<Tag, 'name' | 'slug'>): Promise<Tag> {
  await delay(300)
  const newTag: Tag = {
    id: crypto.randomUUID(),
    name: data.name,
    slug: data.slug,
    postCount: 0,
    createdAt: new Date().toISOString(),
  }
  tagsStore = [newTag, ...tagsStore]
  return newTag
}

/**
 * Mock 删除标签
 */
async function deleteTag(id: string): Promise<void> {
  await delay(200)
  tagsStore = tagsStore.filter((t) => t.id !== id)
}

/**
 * 获取标签列表 Hook
 */
export function useTagList(keyword?: string) {
  return useQuery({
    queryKey: ['tagList', keyword],
    queryFn: () => fetchTags(keyword),
  })
}

/**
 * 创建标签 Hook
 */
export function useCreateTag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createTag,
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
    mutationFn: deleteTag,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tagList'] })
    },
  })
}
