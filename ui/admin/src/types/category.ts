/**
 * 分类相关类型定义
 */

/** 分类数据结构 */
export interface Category {
  id: string
  name: string
  slug: string
  description: string
  icon: string
  parentId: string | null
  postCount: number
  children?: Category[]
  createdAt: string
  updatedAt: string
}
