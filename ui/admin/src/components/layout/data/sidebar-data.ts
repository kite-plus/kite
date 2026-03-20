import {
  LayoutDashboard,
  FileText,
  FilePlus,
  FolderTree,
  Tags,
  MessageSquare,
  Link2,
  Settings,
} from 'lucide-react'
import { KiteIcon } from '@/components/KiteIcon'
import { type SidebarData } from '../types'

/**
 * Kite 博客管理后台侧边栏导航数据
 */
export const sidebarData: SidebarData = {
  user: {
    name: 'Admin',
    email: 'admin@kite.blog',
    avatar: '',
  },
  teams: [
    {
      name: 'Kite',
      logo: KiteIcon,
      plan: '轻量级博客引擎',
    },
  ],
  navGroups: [
    {
      title: '概览',
      items: [
        {
          title: '仪表盘',
          url: '/',
          icon: LayoutDashboard,
        },
      ],
    },
    {
      title: '内容管理',
      items: [
        {
          title: '文章',
          url: '/posts',
          icon: FileText,
        },
        {
          title: '页面',
          url: '/pages',
          icon: FilePlus,
        },
        {
          title: '分类',
          url: '/categories',
          icon: FolderTree,
        },
        {
          title: '标签',
          url: '/tags',
          icon: Tags,
        },
      ],
    },
    {
      title: '互动',
      items: [
        {
          title: '评论',
          url: '/comments',
          icon: MessageSquare,
        },
        {
          title: '友链',
          url: '/links',
          icon: Link2,
        },
      ],
    },
    {
      title: '系统',
      items: [
        {
          title: '设置',
          url: '/settings',
          icon: Settings,
        },
      ],
    },
  ],
}
