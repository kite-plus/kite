/**
 * 系统设置相关类型定义
 */

/** 站点基础设置 */
export interface SiteSettings {
  siteName: string
  siteUrl: string
  description: string
  keywords: string
  favicon: string
  logo: string
  icp: string
  footer: string
}

/** 文章相关设置 */
export interface PostSettings {
  postsPerPage: number
  enableComment: boolean
  enableToc: boolean
  summaryLength: number
  defaultCoverUrl: string
}

/** 渲染模式设置 */
export interface RenderSettings {
  renderMode: 'classic' | 'headless'
  apiPrefix: string
  enableCors: boolean
}

/** AI 集成设置 */
export interface AiSettings {
  enabled: boolean
  provider: 'deepseek' | 'openai' | ''
  apiKey: string
  model: string
  autoSummary: boolean
  autoTag: boolean
}

/** 全部设置聚合 */
export interface AllSettings {
  site: SiteSettings
  post: PostSettings
  render: RenderSettings
  ai: AiSettings
}
