import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useNavigate } from 'react-router'
import {
  IconHome,
  IconArticle,
  IconCopy,
  IconGridView,
  IconPriceTag,
  IconComment,
  IconLink,
  IconSetting,
  IconSearch,
} from '@douyinfe/semi-icons'
import { mockPosts } from '@/mocks/posts'
import { mockPages } from '@/mocks/pages'
import '@/styles/command-palette.css'

/** 命令项类型 */
interface CommandItem {
  id: string
  title: string
  subtitle?: string
  icon: React.ReactNode
  group: string
  action: () => void
}

/**
 * 全局命令面板（⌘K）
 * 类似 Notion / VS Code 的全局搜索与快速导航
 */
export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [activeIndex, setActiveIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const navigate = useNavigate()

  /** 导航类命令 */
  const navCommands: CommandItem[] = useMemo(() => [
    { id: 'nav-dashboard', title: '仪表盘', icon: <IconHome />, group: '导航', action: () => navigate('/') },
    { id: 'nav-posts', title: '文章管理', icon: <IconArticle />, group: '导航', action: () => navigate('/posts') },
    { id: 'nav-pages', title: '页面管理', icon: <IconCopy />, group: '导航', action: () => navigate('/pages') },
    { id: 'nav-categories', title: '分类管理', icon: <IconGridView />, group: '导航', action: () => navigate('/categories') },
    { id: 'nav-tags', title: '标签管理', icon: <IconPriceTag />, group: '导航', action: () => navigate('/tags') },
    { id: 'nav-comments', title: '评论管理', icon: <IconComment />, group: '导航', action: () => navigate('/comments') },
    { id: 'nav-links', title: '友情链接', icon: <IconLink />, group: '导航', action: () => navigate('/links') },
    { id: 'nav-settings', title: '系统设置', icon: <IconSetting />, group: '导航', action: () => navigate('/settings') },
    { id: 'nav-new-post', title: '撰写新文章', subtitle: '创建一篇新的博客文章', icon: <IconArticle />, group: '操作', action: () => navigate('/posts/new') },
    { id: 'nav-new-page', title: '创建新页面', subtitle: '创建一个新的独立页面', icon: <IconCopy />, group: '操作', action: () => navigate('/pages/new') },
  ], [navigate])

  /** 文章搜索命令 */
  const postCommands: CommandItem[] = useMemo(() =>
    mockPosts.map(post => ({
      id: `post-${post.id}`,
      title: post.title,
      subtitle: `${post.category} · ${post.status === 'published' ? '已发布' : post.status === 'draft' ? '草稿' : '已归档'}`,
      icon: <IconArticle />,
      group: '文章',
      action: () => navigate(`/posts/${post.id}/edit`),
    })),
    [navigate]
  )

  /** 页面搜索命令 */
  const pageCommands: CommandItem[] = useMemo(() =>
    mockPages.map(page => ({
      id: `page-${page.id}`,
      title: page.title,
      subtitle: `/${page.slug}`,
      icon: <IconCopy />,
      group: '页面',
      action: () => navigate(`/pages/${page.id}/edit`),
    })),
    [navigate]
  )

  /** 所有命令 */
  const allCommands = useMemo(
    () => [...navCommands, ...postCommands, ...pageCommands],
    [navCommands, postCommands, pageCommands]
  )

  /** 搜索过滤 */
  const filteredCommands = useMemo(() => {
    if (!query.trim()) {
      // 无搜索词时只显示导航和操作
      return navCommands
    }
    const q = query.toLowerCase()
    return allCommands.filter(
      cmd => cmd.title.toLowerCase().includes(q) ||
        cmd.subtitle?.toLowerCase().includes(q) ||
        cmd.group.toLowerCase().includes(q)
    )
  }, [query, allCommands, navCommands])

  /** 按分组归类 */
  const groupedCommands = useMemo(() => {
    const groups: Record<string, CommandItem[]> = {}
    for (const cmd of filteredCommands) {
      if (!groups[cmd.group]) groups[cmd.group] = []
      groups[cmd.group].push(cmd)
    }
    return groups
  }, [filteredCommands])

  /** 扁平化列表（用于键盘导航索引） */
  const flatList = filteredCommands

  /** 监听 ⌘K / Ctrl+K */
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen(prev => !prev)
      }
      if (e.key === 'Escape') {
        setOpen(false)
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  /** 打开时聚焦输入框 */
  useEffect(() => {
    if (open) {
      setQuery('')
      setActiveIndex(0)
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [open])

  /** activeIndex 变化时滚动到可见区 */
  useEffect(() => {
    const item = listRef.current?.querySelector(`[data-index="${activeIndex}"]`)
    item?.scrollIntoView({ block: 'nearest' })
  }, [activeIndex])

  /** 键盘导航 */
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActiveIndex(i => (i + 1) % flatList.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActiveIndex(i => (i - 1 + flatList.length) % flatList.length)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (flatList[activeIndex]) {
        flatList[activeIndex].action()
        setOpen(false)
      }
    }
  }, [flatList, activeIndex])

  /** 执行命令 */
  const executeCommand = useCallback((cmd: CommandItem) => {
    cmd.action()
    setOpen(false)
  }, [])

  if (!open) return null

  // 计算全局 index
  let globalIndex = 0

  return (
    <div className="cmd-overlay" onClick={() => setOpen(false)}>
      <div className="cmd-panel" onClick={e => e.stopPropagation()} onKeyDown={handleKeyDown}>
        {/* 搜索输入 */}
        <div className="cmd-input-wrapper">
          <IconSearch className="cmd-input-icon" />
          <input
            ref={inputRef}
            className="cmd-input"
            value={query}
            onChange={e => { setQuery(e.target.value); setActiveIndex(0) }}
            placeholder="搜索文章、页面或快速跳转…"
            autoComplete="off"
            spellCheck={false}
          />
          <kbd className="cmd-kbd">ESC</kbd>
        </div>

        {/* 命令列表 */}
        <div className="cmd-list" ref={listRef}>
          {flatList.length === 0 && (
            <div className="cmd-empty">没有找到匹配的结果</div>
          )}
          {Object.entries(groupedCommands).map(([group, items]) => (
            <div key={group} className="cmd-group">
              <div className="cmd-group-label">{group}</div>
              {items.map(item => {
                const idx = globalIndex++
                return (
                  <div
                    key={item.id}
                    className={`cmd-item${idx === activeIndex ? ' active' : ''}`}
                    data-index={idx}
                    onClick={() => executeCommand(item)}
                    onMouseEnter={() => setActiveIndex(idx)}
                  >
                    <span className="cmd-item-icon">{item.icon}</span>
                    <div className="cmd-item-text">
                      <span className="cmd-item-title">{item.title}</span>
                      {item.subtitle && <span className="cmd-item-subtitle">{item.subtitle}</span>}
                    </div>
                  </div>
                )
              })}
            </div>
          ))}
        </div>

        {/* 底部提示 */}
        <div className="cmd-footer">
          <span><kbd>↑</kbd><kbd>↓</kbd> 移动</span>
          <span><kbd>↵</kbd> 选择</span>
          <span><kbd>ESC</kbd> 关闭</span>
        </div>
      </div>
    </div>
  )
}
