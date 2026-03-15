import type { Page } from '@/types/page'

/**
 * 独立页面 Mock 数据
 */
export const mockPages: Page[] = [
  {
    id: 'page-1',
    title: '关于我',
    slug: 'about',
    status: 'published',
    sortOrder: 1,
    showInNav: true,
    createdAt: '2025-12-01T10:00:00Z',
    updatedAt: '2026-02-20T14:30:00Z',
    publishedAt: '2025-12-01T10:00:00Z',
  },
  {
    id: 'page-2',
    title: '归档',
    slug: 'archives',
    status: 'published',
    sortOrder: 2,
    showInNav: true,
    createdAt: '2025-12-05T08:00:00Z',
    updatedAt: '2026-03-01T09:00:00Z',
    publishedAt: '2025-12-05T08:00:00Z',
  },
  {
    id: 'page-3',
    title: '留言板',
    slug: 'guestbook',
    status: 'published',
    sortOrder: 3,
    showInNav: true,
    createdAt: '2026-01-10T12:00:00Z',
    updatedAt: '2026-03-05T16:00:00Z',
    publishedAt: '2026-01-10T12:00:00Z',
  },
  {
    id: 'page-4',
    title: '隐私政策',
    slug: 'privacy',
    status: 'published',
    sortOrder: 10,
    showInNav: false,
    createdAt: '2026-01-15T09:00:00Z',
    updatedAt: '2026-01-15T09:00:00Z',
    publishedAt: '2026-01-15T09:00:00Z',
  },
  {
    id: 'page-5',
    title: '简历',
    slug: 'resume',
    status: 'draft',
    sortOrder: 20,
    showInNav: false,
    createdAt: '2026-03-10T11:00:00Z',
    updatedAt: '2026-03-10T11:00:00Z',
    publishedAt: null,
  },
]
