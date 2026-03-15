/**
 * 独立页面相关类型定义
 */

/** 页面状态枚举 */
export type PageStatus = 'published' | 'draft'

/** 独立页面数据结构 */
export interface Page {
  id: string
  title: string
  slug: string
  status: PageStatus
  /** 排序优先级，数值越小越靠前 */
  sortOrder: number
  /** 是否在前台导航栏中显示 */
  showInNav: boolean
  createdAt: string
  updatedAt: string
  publishedAt: string | null
}

/** 独立页面详情（含正文） */
export interface PageDetail extends Page {
  content: string
}

/** 独立页面表单数据 */
export interface PageFormData {
  title: string
  slug: string
  content: string
  status: PageStatus
  sortOrder: number
  showInNav: boolean
}
