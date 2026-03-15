import { useState } from 'react'
import {
  Search,
  Plus,
  X,
  ExternalLink,
  Trash2,
  Link as LinkIcon,
  CheckCircle,
  Clock,
  AlertTriangle,
  Globe,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useFriendLinks, useCreateFriendLink, useDeleteFriendLink, useToggleLinkStatus } from '@/hooks/use-friend-links'
import type { LinkStatus } from '@/types/friend-link'

/** 状态徽标配置 */
const statusConfig: Record<LinkStatus, { text: string; cls: string; icon: React.ComponentType<{ className?: string; strokeWidth?: number }> }> = {
  active: { text: '正常', cls: 'border-emerald-300 bg-emerald-50 text-emerald-700', icon: CheckCircle },
  pending: { text: '待审核', cls: 'border-amber-300 bg-amber-50 text-amber-700', icon: Clock },
  down: { text: '已下线', cls: 'border-red-300 bg-red-50 text-red-700', icon: AlertTriangle },
}

/**
 * 友链管理页面
 * 卡片列表布局，支持创建、删除、状态切换
 */
export function FriendLinksPage() {
  const [keyword, setKeyword] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', url: '', description: '' })

  const { data: links, isLoading } = useFriendLinks(keyword)
  const createMutation = useCreateFriendLink()
  const deleteMutation = useDeleteFriendLink()
  const toggleMutation = useToggleLinkStatus()

  /** 提交新建 */
  function handleCreate() {
    if (!formData.name.trim() || !formData.url.trim()) return
    createMutation.mutate(formData, {
      onSuccess: () => {
        setFormData({ name: '', url: '', description: '' })
        setShowForm(false)
      },
    })
  }

  /** 删除友链 */
  function handleDelete(id: string, name: string) {
    if (window.confirm(`确定删除友链「${name}」吗？`)) {
      deleteMutation.mutate(id)
    }
  }

  /** 切换状态 */
  function handleToggle(id: string, currentStatus: LinkStatus) {
    const nextStatus: LinkStatus = currentStatus === 'active' ? 'down' : 'active'
    toggleMutation.mutate({ id, status: nextStatus })
  }

  /** 提取域名显示 */
  function extractDomain(url: string): string {
    try {
      return new URL(url).hostname
    } catch {
      return url
    }
  }

  const activeCount = links?.filter((l) => l.status === 'active').length ?? 0
  const totalCount = links?.length ?? 0

  return (
    <div>
      {/* 页面标题区 */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
            友链管理
          </h1>
          <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
            管理博客友情链接，互换链接、共建生态
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex h-9 items-center gap-2 border border-[var(--kite-accent)] bg-[var(--kite-accent)] px-4 text-sm font-medium text-white transition-colors duration-100 hover:bg-[#333] cursor-pointer"
        >
          <Plus className="h-4 w-4" strokeWidth={1.5} />
          新增友链
        </button>
      </div>

      {/* 新增友链表单 */}
      {showForm && (
        <div className="mb-6 border border-[var(--kite-border)] bg-[var(--kite-bg)] p-5">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">新增友链</h2>
            <button
              onClick={() => setShowForm(false)}
              className="flex h-7 w-7 items-center justify-center text-[var(--kite-text-muted)] hover:text-[var(--kite-text-heading)] cursor-pointer"
            >
              <X className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">名称 *</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData((p) => ({ ...p, name: e.target.value }))}
                placeholder="例如：阮一峰的网络日志"
                className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">链接 *</label>
              <input
                type="url"
                value={formData.url}
                onChange={(e) => setFormData((p) => ({ ...p, url: e.target.value }))}
                placeholder="https://example.com"
                className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
              />
            </div>
          </div>
          <div className="mt-4">
            <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">描述</label>
            <input
              type="text"
              value={formData.description}
              onChange={(e) => setFormData((p) => ({ ...p, description: e.target.value }))}
              placeholder="一句话介绍这个博客…"
              className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
            />
          </div>
          <div className="mt-4 flex justify-end gap-2">
            <button
              onClick={() => setShowForm(false)}
              className="h-9 border border-[var(--kite-border)] bg-transparent px-4 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] cursor-pointer"
            >
              取消
            </button>
            <button
              onClick={handleCreate}
              disabled={!formData.name.trim() || !formData.url.trim() || createMutation.isPending}
              className="h-9 border border-[var(--kite-accent)] bg-[var(--kite-accent)] px-4 text-sm font-medium text-white transition-colors duration-100 hover:bg-[#333] disabled:opacity-50 cursor-pointer"
            >
              {createMutation.isPending ? '创建中…' : '创建'}
            </button>
          </div>
        </div>
      )}

      {/* 搜索栏 + 统计 */}
      <div className="mb-4 flex items-center justify-between gap-4">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
          <input
            type="text"
            placeholder="搜索友链…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] pl-9 pr-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
          />
        </div>
        <span className="text-xs text-[var(--kite-text-muted)]">
          共 {totalCount} 条友链 · {activeCount} 条在线
        </span>
      </div>

      {/* 加载态 */}
      {isLoading && (
        <div className="flex items-center justify-center py-16 text-sm text-[var(--kite-text-muted)]">
          加载中…
        </div>
      )}

      {/* 空态 */}
      {!isLoading && links?.length === 0 && (
        <div className="flex flex-col items-center justify-center border border-dashed border-[var(--kite-border)] py-16 text-sm text-[var(--kite-text-muted)]">
          <LinkIcon className="mb-3 h-8 w-8" strokeWidth={1} />
          <p>暂无友链，点击上方按钮添加</p>
        </div>
      )}

      {/* 友链卡片列表 */}
      {links && links.length > 0 && (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {links.map((link) => {
            const st = statusConfig[link.status]
            const StIcon = st.icon
            return (
              <div
                key={link.id}
                className={cn(
                  'group border border-[var(--kite-border)] bg-[var(--kite-bg)] transition-colors duration-100 hover:border-[var(--kite-border-hover)]',
                  link.status === 'down' && 'opacity-60'
                )}
              >
                <div className="p-5">
                  {/* 头部：名称 + 状态 */}
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      {/* Logo 占位 */}
                      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center border border-[var(--kite-border)] bg-[var(--kite-bg-hover)] text-sm font-semibold text-[var(--kite-text-muted)]">
                        {link.name.charAt(0)}
                      </div>
                      <div>
                        <h3 className="text-sm font-semibold text-[var(--kite-text-heading)]">
                          {link.name}
                        </h3>
                        <div className="mt-0.5 flex items-center gap-1 text-xs text-[var(--kite-text-muted)]">
                          <Globe className="h-3 w-3" strokeWidth={1.5} />
                          {extractDomain(link.url)}
                        </div>
                      </div>
                    </div>
                    <span className={cn('flex items-center gap-1 border px-1.5 py-0.5 text-xs', st.cls)}>
                      <StIcon className="h-3 w-3" strokeWidth={1.5} />
                      {st.text}
                    </span>
                  </div>

                  {/* 描述 */}
                  {link.description && (
                    <p className="mt-3 line-clamp-2 text-sm text-[var(--kite-text-muted)] leading-relaxed">
                      {link.description}
                    </p>
                  )}
                </div>

                {/* 底部操作栏 */}
                <div className="flex items-center justify-between border-t border-[var(--kite-border)] px-5 py-2.5">
                  <span className="text-xs text-[var(--kite-text-muted)]">
                    #{link.sortOrder}
                  </span>
                  <div className="flex items-center gap-1 opacity-0 transition-opacity duration-100 group-hover:opacity-100">
                    <a
                      href={link.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="flex h-7 items-center gap-1 border border-[var(--kite-border)] px-2 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:text-[var(--kite-text-heading)] no-underline"
                      title="访问"
                    >
                      <ExternalLink className="h-3 w-3" strokeWidth={1.5} />
                      访问
                    </a>
                    <button
                      onClick={() => handleToggle(link.id, link.status)}
                      disabled={toggleMutation.isPending}
                      className={cn(
                        'flex h-7 items-center gap-1 border px-2 text-xs transition-colors duration-100 cursor-pointer',
                        link.status === 'active'
                          ? 'border-[var(--kite-border)] text-[var(--kite-text-muted)] hover:border-amber-300 hover:text-amber-600'
                          : 'border-[var(--kite-border)] text-[var(--kite-text-muted)] hover:border-emerald-300 hover:text-emerald-600'
                      )}
                      title={link.status === 'active' ? '下线' : '上线'}
                    >
                      {link.status === 'active' ? (
                        <><AlertTriangle className="h-3 w-3" strokeWidth={1.5} />下线</>
                      ) : (
                        <><CheckCircle className="h-3 w-3" strokeWidth={1.5} />上线</>
                      )}
                    </button>
                    <button
                      onClick={() => handleDelete(link.id, link.name)}
                      disabled={deleteMutation.isPending}
                      className="flex h-7 items-center justify-center border border-[var(--kite-border)] px-2 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-red-300 hover:text-red-600 cursor-pointer"
                      title="删除"
                    >
                      <Trash2 className="h-3 w-3" strokeWidth={1.5} />
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
