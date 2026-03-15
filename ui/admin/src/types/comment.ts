/**
 * 评论相关类型定义
 */

/** 评论状态 */
export type CommentStatus = 'approved' | 'pending' | 'spam'

/** 评论数据结构 */
export interface Comment {
  id: string
  postId: string
  postTitle: string
  author: string
  email: string
  content: string
  status: CommentStatus
  ip: string
  userAgent: string
  createdAt: string
}
