import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { mockPages } from '@/mocks/pages'
import type { Page, PageDetail, PageFormData } from '@/types/page'

/**
 * 模拟 API 请求延迟
 */
function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

/**
 * 获取独立页面列表 Hook
 */
export function usePageList(keyword?: string) {
  return useQuery<Page[]>({
    queryKey: ['pages', keyword],
    queryFn: async () => {
      await delay(200)
      let pages = [...mockPages]
      if (keyword) {
        const kw = keyword.toLowerCase()
        pages = pages.filter((p) => p.title.toLowerCase().includes(kw) || p.slug.toLowerCase().includes(kw))
      }
      // 按 sortOrder 升序
      pages.sort((a, b) => a.sortOrder - b.sortOrder)
      return pages
    },
  })
}

/** Mock 页面正文内容 */
const mockPageContents: Record<string, string> = {
  'page-1': `<h2>关于我</h2>
<p>嗨，欢迎来到我的博客！我是一名全栈开发工程师，热爱 <strong>Go</strong> 和 <strong>React</strong>。</p>
<h3>技术栈</h3>
<ul>
  <li>后端：Go, Gin, GORM, PostgreSQL</li>
  <li>前端：React, TypeScript, Vite</li>
  <li>DevOps：Docker, Kubernetes, GitHub Actions</li>
</ul>
<h3>联系方式</h3>
<p>📧 Email: hello@example.com</p>
<p>🐙 GitHub: <a href="https://github.com">github.com/username</a></p>`,
  'page-2': `<h2>归档</h2>
<p>这里按时间线列出我的所有文章。</p>
<p>（此页面内容由系统自动生成）</p>`,
  'page-3': `<h2>留言板</h2>
<p>欢迎在这里留下你的想法、建议或问候 👋</p>
<p>请文明留言，共同维护良好的交流环境。</p>`,
  'page-4': `<h2>隐私政策</h2>
<p>本站非常重视用户隐私，以下是我们的隐私保护声明：</p>
<ul>
  <li>本站不会收集任何个人隐私信息</li>
  <li>本站使用匿名统计分析访问数据</li>
  <li>评论功能需提供昵称和邮箱，仅用于通知回复</li>
</ul>`,
  'page-5': `<h2>简历</h2>
<p>（草稿中，尚未发布）</p>`,
}

/**
 * 获取独立页面详情 Hook
 */
export function usePageDetail(id: string | undefined) {
  return useQuery<PageDetail>({
    queryKey: ['page', id],
    queryFn: async () => {
      await delay(300)
      const page = mockPages.find((p) => p.id === id)
      if (!page) throw new Error('页面不存在')
      return { ...page, content: mockPageContents[page.id] || '<p>暂无内容</p>' }
    },
    enabled: !!id,
  })
}

/**
 * 保存独立页面 Hook（新建 / 更新）
 */
export function useSavePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: PageFormData & { id?: string }) => {
      await delay(500)
      return { id: data.id || crypto.randomUUID(), ...data }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pages'] })
    },
  })
}

/**
 * 删除独立页面 Hook
 */
export function useDeletePage() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (id: string) => {
      await delay(300)
      return id
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pages'] })
    },
  })
}
