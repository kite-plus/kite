/**
 * 文章相关类型定义
 */

/** 文章状态枚举 */
export type PostStatus = 'published' | 'draft' | 'archived'

/** 文章数据结构（与后端对齐） */
export interface Post {
  id: string
  title: string
  slug: string
  summary: string
  category: { id: string; name: string; slug: string } | null
  categoryId: string | null
  tags: { id: string; name: string; slug: string }[]
  status: PostStatus
  coverImage: string
  showComments: boolean
  hasPassword: boolean
  hasProtected: boolean
  createdAt: string
  updatedAt: string
  publishedAt: string | null
}

/** 文章列表查询参数 */
export interface PostQueryParams {
  page: number
  pageSize: number
  keyword?: string
  status?: PostStatus | 'all'
  categoryId?: string
  tagId?: string
}

/** 分页响应 */
export interface PaginatedData<T> {
  items: T[]
  pagination: {
    page: number
    pageSize: number
    total: number
  }
}

/** 文章详情（含正文内容） */
export interface PostDetail extends Post {
  contentMarkdown: string
  contentHtml: string
}

/** 文章表单数据 */
export interface PostFormData {
  title: string
  slug: string
  summary: string
  contentMarkdown: string
  categoryId: string
  tagIds: string[]
  status: PostStatus
  coverImage: string
  password: string
}
