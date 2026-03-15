import { useState } from 'react'
import {
  Search,
  Check,
  Trash2,
  AlertTriangle,
  MessageSquare,
  Clock,
  FileText,
  Shield,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useComments, useCommentStats, useModerateComment } from '@/hooks/use-comments'
import type { CommentStatus } from '@/types/comment'

/** 状态筛选选项 */
const statusFilters: { key: CommentStatus | 'all'; label: string; icon: React.ComponentType<{ className?: string; strokeWidth?: number }> }[] = [
  { key: 'all', label: '全部', icon: MessageSquare },
  { key: 'pending', label: '待审核', icon: Clock },
  { key: 'approved', label: '已通过', icon: Check },
  { key: 'spam', label: '垃圾', icon: AlertTriangle },
]

/** 状态徽标样式 */
const statusBadge: Record<CommentStatus, { text: string; cls: string }> = {
  approved: { text: '已通过', cls: 'border-emerald-300 bg-emerald-50 text-emerald-700' },
  pending: { text: '待审核', cls: 'border-amber-300 bg-amber-50 text-amber-700' },
  spam: { text: '垃圾', cls: 'border-red-300 bg-red-50 text-red-700' },
}

/**
 * 格式化时间
 */
function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

/**
 * 评论管理页面
 * 支持状态筛选、关键词搜索、审核操作（通过/垃圾/删除）、评论展开
 */
export function CommentsPage() {
  const [statusFilter, setStatusFilter] = useState<CommentStatus | 'all'>('all')
  const [keyword, setKeyword] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const { data: comments, isLoading } = useComments({ status: statusFilter, keyword })
  const { data: stats } = useCommentStats()
  const moderateMutation = useModerateComment()

  /** 执行审核操作 */
  function handleModerate(id: string, action: 'approve' | 'spam' | 'delete') {
    if (action === 'delete' && !window.confirm('确定删除此评论？此操作不可撤销。')) return
    moderateMutation.mutate({ id, action })
  }

  return (
    <div>
      {/* 页面标题 */}
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
          评论管理
        </h1>
        <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
          审核和管理读者的评论
        </p>
      </div>

      {/* 统计卡片 */}
      {stats && (
        <div className="mb-6 grid grid-cols-4 gap-4">
          {[
            { label: '全部评论', value: stats.total, cls: 'border-l-[var(--kite-accent)]' },
            { label: '待审核', value: stats.pending, cls: 'border-l-amber-400' },
            { label: '已通过', value: stats.approved, cls: 'border-l-emerald-400' },
            { label: '垃圾评论', value: stats.spam, cls: 'border-l-red-400' },
          ].map((card) => (
            <div
              key={card.label}
              className={cn(
                'border border-[var(--kite-border)] border-l-2 bg-[var(--kite-bg)] p-4 transition-colors duration-100 hover:border-[var(--kite-border-hover)]',
                card.cls
              )}
            >
              <p className="text-xs text-[var(--kite-text-muted)]">{card.label}</p>
              <p className="mt-1 text-2xl font-semibold text-[var(--kite-text-heading)]">{card.value}</p>
            </div>
          ))}
        </div>
      )}

      {/* 筛选栏 */}
      <div className="mb-4 flex items-center justify-between gap-4">
        {/* 状态 Tab */}
        <div className="flex border border-[var(--kite-border)]">
          {statusFilters.map((f) => {
            const Icon = f.icon
            const isActive = statusFilter === f.key
            const count = stats
              ? f.key === 'all'
                ? stats.total
                : stats[f.key as keyof typeof stats]
              : 0
            return (
              <button
                key={f.key}
                onClick={() => setStatusFilter(f.key)}
                className={cn(
                  'flex items-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors duration-100 cursor-pointer',
                  f.key !== 'all' && 'border-l border-[var(--kite-border)]',
                  isActive
                    ? 'bg-[var(--kite-accent)] text-white'
                    : 'text-[var(--kite-text-muted)] hover:bg-[var(--kite-bg-hover)] hover:text-[var(--kite-text-heading)]'
                )}
              >
                <Icon className="h-3.5 w-3.5" strokeWidth={1.5} />
                {f.label}
                <span className={cn(
                  'ml-0.5 text-xs',
                  isActive ? 'text-white/70' : 'text-[var(--kite-text-muted)]'
                )}>
                  {count}
                </span>
              </button>
            )
          })}
        </div>

        {/* 搜索框 */}
        <div className="relative max-w-xs flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
          <input
            type="text"
            placeholder="搜索评论内容、作者…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] pl-9 pr-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
          />
        </div>
      </div>

      {/* 加载态 */}
      {isLoading && (
        <div className="flex items-center justify-center py-16 text-sm text-[var(--kite-text-muted)]">
          加载中…
        </div>
      )}

      {/* 空态 */}
      {!isLoading && comments?.length === 0 && (
        <div className="flex flex-col items-center justify-center border border-dashed border-[var(--kite-border)] py-16 text-sm text-[var(--kite-text-muted)]">
          <MessageSquare className="mb-3 h-8 w-8" strokeWidth={1} />
          <p>暂无评论</p>
        </div>
      )}

      {/* 评论列表 */}
      {comments && comments.length > 0 && (
        <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)]">
          {comments.map((comment, index) => {
            const badge = statusBadge[comment.status]
            const isExpanded = expandedId === comment.id
            return (
              <div
                key={comment.id}
                className={cn(
                  'transition-colors duration-100',
                  index < comments.length - 1 && 'border-b border-[var(--kite-border)]',
                  comment.status === 'spam' && 'bg-red-50/30'
                )}
              >
                {/* 主行 */}
                <div className="flex items-start gap-4 px-5 py-4">
                  {/* 头像占位 */}
                  <div className="flex h-9 w-9 flex-shrink-0 items-center justify-center border border-[var(--kite-border)] bg-[var(--kite-bg-hover)] text-xs font-semibold text-[var(--kite-text-muted)]">
                    {comment.author.charAt(0).toUpperCase()}
                  </div>

                  {/* 内容区 */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-[var(--kite-text-heading)]">
                        {comment.author}
                      </span>
                      <span className={cn('border px-1.5 py-0.5 text-xs', badge.cls)}>
                        {badge.text}
                      </span>
                      <span className="ml-auto flex items-center gap-1 text-xs text-[var(--kite-text-muted)]">
                        <Clock className="h-3 w-3" strokeWidth={1.5} />
                        {timeAgo(comment.createdAt)}
                      </span>
                    </div>

                    {/* 所属文章 */}
                    <div className="mt-1 flex items-center gap-1 text-xs text-[var(--kite-text-muted)]">
                      <FileText className="h-3 w-3" strokeWidth={1.5} />
                      {comment.postTitle}
                    </div>

                    {/* 评论内容 */}
                    <p className={cn(
                      'mt-2 text-sm text-[var(--kite-text)] leading-relaxed',
                      !isExpanded && 'line-clamp-2'
                    )}>
                      {comment.content}
                    </p>

                    {/* 展开/收起 */}
                    {comment.content.length > 100 && (
                      <button
                        onClick={() => setExpandedId(isExpanded ? null : comment.id)}
                        className="mt-1 flex items-center gap-0.5 text-xs text-[var(--kite-text-muted)] hover:text-[var(--kite-text-heading)] cursor-pointer"
                      >
                        {isExpanded ? (
                          <>收起 <ChevronUp className="h-3 w-3" strokeWidth={1.5} /></>
                        ) : (
                          <>展开 <ChevronDown className="h-3 w-3" strokeWidth={1.5} /></>
                        )}
                      </button>
                    )}

                    {/* 展开后的详细信息 */}
                    {isExpanded && (
                      <div className="mt-3 flex gap-4 border-t border-[var(--kite-border)] pt-3 text-xs text-[var(--kite-text-muted)]">
                        <span>邮箱：{comment.email}</span>
                        <span>IP：{comment.ip}</span>
                        <span>UA：{comment.userAgent}</span>
                      </div>
                    )}
                  </div>

                  {/* 操作按钮 */}
                  <div className="flex flex-shrink-0 items-center gap-1">
                    {comment.status !== 'approved' && (
                      <button
                        onClick={() => handleModerate(comment.id, 'approve')}
                        disabled={moderateMutation.isPending}
                        className="flex h-8 items-center gap-1 border border-[var(--kite-border)] px-2 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-emerald-300 hover:text-emerald-600 cursor-pointer"
                        title="通过"
                      >
                        <Check className="h-3.5 w-3.5" strokeWidth={1.5} />
                        通过
                      </button>
                    )}
                    {comment.status !== 'spam' && (
                      <button
                        onClick={() => handleModerate(comment.id, 'spam')}
                        disabled={moderateMutation.isPending}
                        className="flex h-8 items-center gap-1 border border-[var(--kite-border)] px-2 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-amber-300 hover:text-amber-600 cursor-pointer"
                        title="标记垃圾"
                      >
                        <Shield className="h-3.5 w-3.5" strokeWidth={1.5} />
                        垃圾
                      </button>
                    )}
                    <button
                      onClick={() => handleModerate(comment.id, 'delete')}
                      disabled={moderateMutation.isPending}
                      className="flex h-8 items-center gap-1 border border-[var(--kite-border)] px-2 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-red-300 hover:text-red-600 cursor-pointer"
                      title="删除"
                    >
                      <Trash2 className="h-3.5 w-3.5" strokeWidth={1.5} />
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
