import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mockComments } from '@/mocks/comments'
import type { Comment, CommentStatus } from '@/types/comment'

/** 模拟延迟 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/** 内存中的评论副本 */
let commentsStore = [...mockComments]

/** 查询参数 */
interface CommentQueryParams {
  status?: CommentStatus | 'all'
  keyword?: string
}

/**
 * Mock 获取评论列表
 */
async function fetchComments(params: CommentQueryParams): Promise<Comment[]> {
  await delay(200)
  let result = [...commentsStore]

  if (params.status && params.status !== 'all') {
    result = result.filter((c) => c.status === params.status)
  }

  if (params.keyword) {
    const kw = params.keyword.toLowerCase()
    result = result.filter(
      (c) =>
        c.content.toLowerCase().includes(kw) ||
        c.author.toLowerCase().includes(kw) ||
        c.postTitle.toLowerCase().includes(kw)
    )
  }

  // 按时间倒序
  result.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
  return result
}

/**
 * Mock 审核评论（通过/标记垃圾/删除）
 */
async function moderateComment(data: { id: string; action: 'approve' | 'spam' | 'delete' }): Promise<void> {
  await delay(200)
  if (data.action === 'delete') {
    commentsStore = commentsStore.filter((c) => c.id !== data.id)
  } else {
    commentsStore = commentsStore.map((c) =>
      c.id === data.id ? { ...c, status: data.action === 'approve' ? 'approved' : 'spam' } : c
    )
  }
}

/**
 * 获取评论列表 Hook
 */
export function useComments(params: CommentQueryParams) {
  return useQuery({
    queryKey: ['comments', params],
    queryFn: () => fetchComments(params),
  })
}

/**
 * 评论审核状态统计
 */
export function useCommentStats() {
  return useQuery({
    queryKey: ['commentStats'],
    queryFn: async () => {
      await delay(100)
      return {
        total: commentsStore.length,
        approved: commentsStore.filter((c) => c.status === 'approved').length,
        pending: commentsStore.filter((c) => c.status === 'pending').length,
        spam: commentsStore.filter((c) => c.status === 'spam').length,
      }
    },
  })
}

/**
 * 审核评论 Hook
 */
export function useModerateComment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: moderateComment,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments'] })
      queryClient.invalidateQueries({ queryKey: ['commentStats'] })
    },
  })
}
