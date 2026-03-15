import type { AllSettings } from '@/types/settings'

/**
 * Mock 系统设置数据
 */
export const mockSettings: AllSettings = {
  site: {
    siteName: 'Kite Blog',
    siteUrl: 'https://blog.example.com',
    description: '一个由 Go 驱动的极简 AI 原生博客引擎',
    keywords: '博客,Go,React,AI,技术',
    favicon: '/favicon.svg',
    logo: '',
    icp: '',
    footer: '© 2026 Kite Blog. Powered by Kite Engine.',
  },
  post: {
    postsPerPage: 10,
    enableComment: true,
    enableToc: true,
    summaryLength: 200,
    defaultCoverUrl: '',
  },
  render: {
    renderMode: 'classic',
    apiPrefix: '/api/v1',
    enableCors: false,
  },
  ai: {
    enabled: false,
    provider: 'deepseek',
    apiKey: '',
    model: 'deepseek-chat',
    autoSummary: true,
    autoTag: false,
  },
}
