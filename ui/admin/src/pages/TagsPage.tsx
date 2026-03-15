import { useState } from 'react'
import {
  Search,
  Plus,
  X,
  Tag as TagIcon,
  FileText,
  Trash2,
  Hash,
} from 'lucide-react'
import { cn } from '@/lib/cn'
import { useTagList, useCreateTag, useDeleteTag } from '@/hooks/use-tags'

/**
 * 标签管理页面
 * 标签云 + 列表双视图、内联创建、搜索
 */
export function TagsPage() {
  const [keyword, setKeyword] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', slug: '' })
  const [viewMode, setViewMode] = useState<'cloud' | 'list'>('cloud')

  const { data: tags, isLoading } = useTagList(keyword)
  const createMutation = useCreateTag()
  const deleteMutation = useDeleteTag()

  /** 自动生成 slug */
  function handleNameChange(name: string) {
    setFormData({
      name,
      slug: name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    })
  }

  /** 提交新建 */
  function handleCreate() {
    if (!formData.name.trim()) return
    createMutation.mutate(formData, {
      onSuccess: () => {
        setFormData({ name: '', slug: '' })
        setShowForm(false)
      },
    })
  }

  /** 确认删除 */
  function handleDelete(id: string, name: string) {
    if (window.confirm(`确定删除标签「${name}」吗？`)) {
      deleteMutation.mutate(id)
    }
  }

  /** 计算标签云字号 */
  function getTagSize(postCount: number, maxCount: number): string {
    if (maxCount === 0) return 'text-sm'
    const ratio = postCount / maxCount
    if (ratio > 0.7) return 'text-xl font-semibold'
    if (ratio > 0.4) return 'text-base font-medium'
    if (ratio > 0.15) return 'text-sm'
    return 'text-xs'
  }

  const maxPostCount = tags ? Math.max(...tags.map((t) => t.postCount), 1) : 1
  const totalPosts = tags ? tags.reduce((sum, t) => sum + t.postCount, 0) : 0

  return (
    <div>
      {/* 页面标题区 */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
            标签管理
          </h1>
          <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
            管理文章标签，组织内容的细粒度分类
          </p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="flex h-9 items-center gap-2 border border-[var(--kite-accent)] bg-[var(--kite-accent)] px-4 text-sm font-medium text-white transition-colors duration-100 hover:bg-[#333] cursor-pointer"
        >
          <Plus className="h-4 w-4" strokeWidth={1.5} />
          新建标签
        </button>
      </div>

      {/* 新建标签表单 */}
      {showForm && (
        <div className="mb-6 border border-[var(--kite-border)] bg-[var(--kite-bg)] p-5">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">新建标签</h2>
            <button
              onClick={() => setShowForm(false)}
              className="flex h-7 w-7 items-center justify-center text-[var(--kite-text-muted)] hover:text-[var(--kite-text-heading)] cursor-pointer"
            >
              <X className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">
                标签名称
              </label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="例如：Kubernetes"
                className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
              />
            </div>
            <div className="flex-1">
              <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">
                Slug
              </label>
              <input
                type="text"
                value={formData.slug}
                onChange={(e) => setFormData((prev) => ({ ...prev, slug: e.target.value }))}
                placeholder="例如：kubernetes"
                className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
              />
            </div>
            <div className="flex items-end gap-2">
              <button
                onClick={() => setShowForm(false)}
                className="h-9 border border-[var(--kite-border)] bg-transparent px-4 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] cursor-pointer"
              >
                取消
              </button>
              <button
                onClick={handleCreate}
                disabled={!formData.name.trim() || createMutation.isPending}
                className="h-9 border border-[var(--kite-accent)] bg-[var(--kite-accent)] px-4 text-sm font-medium text-white transition-colors duration-100 hover:bg-[#333] disabled:opacity-50 cursor-pointer"
              >
                {createMutation.isPending ? '创建中…' : '创建'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 搜索栏 + 视图切换 + 统计 */}
      <div className="mb-4 flex items-center justify-between gap-4">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
          <input
            type="text"
            placeholder="搜索标签…"
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] pl-9 pr-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
          />
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-[var(--kite-text-muted)]">
            {tags?.length ?? 0} 个标签 · {totalPosts} 次引用
          </span>
          <div className="flex border border-[var(--kite-border)]">
            <button
              onClick={() => setViewMode('cloud')}
              className={cn(
                'flex h-8 w-8 items-center justify-center text-sm transition-colors duration-100 cursor-pointer',
                viewMode === 'cloud'
                  ? 'bg-[var(--kite-accent)] text-white'
                  : 'text-[var(--kite-text-muted)] hover:bg-[var(--kite-bg-hover)]'
              )}
              title="标签云"
            >
              <Hash className="h-3.5 w-3.5" strokeWidth={1.5} />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={cn(
                'flex h-8 w-8 items-center justify-center border-l border-[var(--kite-border)] text-sm transition-colors duration-100 cursor-pointer',
                viewMode === 'list'
                  ? 'bg-[var(--kite-accent)] text-white'
                  : 'text-[var(--kite-text-muted)] hover:bg-[var(--kite-bg-hover)]'
              )}
              title="列表"
            >
              <FileText className="h-3.5 w-3.5" strokeWidth={1.5} />
            </button>
          </div>
        </div>
      </div>

      {/* 加载态 */}
      {isLoading && (
        <div className="flex items-center justify-center py-16 text-sm text-[var(--kite-text-muted)]">
          加载中…
        </div>
      )}

      {/* 空态 */}
      {!isLoading && tags?.length === 0 && (
        <div className="flex flex-col items-center justify-center border border-dashed border-[var(--kite-border)] py-16 text-sm text-[var(--kite-text-muted)]">
          <TagIcon className="mb-3 h-8 w-8" strokeWidth={1} />
          <p>暂无标签，点击上方按钮创建</p>
        </div>
      )}

      {/* 标签云视图 */}
      {tags && tags.length > 0 && viewMode === 'cloud' && (
        <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)] p-6">
          <div className="flex flex-wrap gap-2.5">
            {tags.map((tag) => (
              <div
                key={tag.id}
                className="group relative flex items-center"
              >
                <span
                  className={cn(
                    'inline-flex items-center gap-1.5 border border-[var(--kite-border)] px-3 py-1.5 text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:bg-[var(--kite-bg-hover)]',
                    getTagSize(tag.postCount, maxPostCount)
                  )}
                >
                  <Hash className="h-3 w-3 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                  {tag.name}
                  <span className="text-xs font-normal text-[var(--kite-text-muted)]">
                    {tag.postCount}
                  </span>
                </span>
                {/* hover 删除按钮 */}
                <button
                  onClick={() => handleDelete(tag.id, tag.name)}
                  className="absolute -right-1 -top-1 flex h-4 w-4 items-center justify-center border border-red-300 bg-white text-red-500 opacity-0 transition-opacity duration-100 group-hover:opacity-100 hover:bg-red-50 cursor-pointer"
                  title="删除"
                >
                  <X className="h-2.5 w-2.5" strokeWidth={2} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 列表视图 */}
      {tags && tags.length > 0 && viewMode === 'list' && (
        <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)]">
          {/* 表头 */}
          <div className="grid grid-cols-[1fr_120px_100px_60px] gap-4 border-b border-[var(--kite-border)] px-5 py-3 text-xs font-medium uppercase tracking-wider text-[var(--kite-text-muted)]">
            <span>标签名称</span>
            <span>Slug</span>
            <span className="text-center">文章数</span>
            <span className="text-center">操作</span>
          </div>
          {/* 数据行 */}
          {tags.map((tag, index) => (
            <div
              key={tag.id}
              className={cn(
                'group grid grid-cols-[1fr_120px_100px_60px] gap-4 px-5 py-3 text-sm transition-colors duration-100 hover:bg-[var(--kite-bg-hover)]',
                index < tags.length - 1 && 'border-b border-[var(--kite-border)]'
              )}
            >
              <div className="flex items-center gap-2">
                <Hash className="h-3.5 w-3.5 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                <span className="font-medium text-[var(--kite-text-heading)]">{tag.name}</span>
              </div>
              <div className="flex items-center text-[var(--kite-text-muted)]">
                {tag.slug}
              </div>
              <div className="flex items-center justify-center">
                <span className="inline-flex items-center gap-1 text-[var(--kite-text-muted)]">
                  <FileText className="h-3 w-3" strokeWidth={1.5} />
                  {tag.postCount}
                </span>
              </div>
              <div className="flex items-center justify-center">
                <button
                  onClick={() => handleDelete(tag.id, tag.name)}
                  className="flex h-7 w-7 items-center justify-center border border-transparent text-[var(--kite-text-muted)] transition-colors duration-100 hover:border-red-300 hover:text-red-600 cursor-pointer"
                  title="删除"
                >
                  <Trash2 className="h-3.5 w-3.5" strokeWidth={1.5} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
