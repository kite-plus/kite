import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import { Search, Plus, Hash, Loader2, X, ExternalLink } from 'lucide-react'
import { useTagList, useCreateTag, useUpdateTag, useDeleteTag } from '@/hooks/use-tags'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search as SearchBtn } from '@/components/search'
import { useConfirm } from '@/hooks/use-confirm'

/**
 * 标签管理页面 — Badge 标签墙
 */
export function TagsPage() {
  const [keyword, setKeyword] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingTag, setEditingTag] = useState<{ id: string; name: string; slug: string } | null>(null)
  const [formData, setFormData] = useState({ name: '', slug: '' })

  const { data: tags, isLoading } = useTagList(keyword)
  const createMutation = useCreateTag()
  const updateMutation = useUpdateTag()
  const deleteMutation = useDeleteTag()
  const { confirm, ConfirmDialog } = useConfirm()

  /** 标签名称变更时自动生成 slug */
  function handleNameChange(name: string) {
    setFormData({ name, slug: name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, '') })
  }

  /** 打开新建弹窗 */
  function openCreateDialog() {
    setEditingTag(null)
    setFormData({ name: '', slug: '' })
    setDialogOpen(true)
  }

  /** 点击标签打开编辑弹窗 */
  function openEditDialog(tag: { id: string; name: string; slug: string }) {
    setEditingTag(tag)
    setFormData({ name: tag.name, slug: tag.slug })
    setDialogOpen(true)
  }

  /** 创建或更新标签 */
  function handleSubmit() {
    if (!formData.name.trim()) return
    if (editingTag) {
      updateMutation.mutate({ id: editingTag.id, ...formData }, {
        onSuccess: () => { setDialogOpen(false); setEditingTag(null) },
      })
    } else {
      createMutation.mutate(formData, {
        onSuccess: () => { setFormData({ name: '', slug: '' }); setDialogOpen(false) },
      })
    }
  }

  /** 删除标签 */
  async function handleDelete(e: React.MouseEvent, id: string) {
    e.stopPropagation()
    if (await confirm({ title: '删除标签', description: '确定删除此标签吗？此操作不可撤销。', confirmText: '删除', variant: 'destructive' })) {
      deleteMutation.mutate(id)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <>
      <ConfirmDialog />
      <Header fixed>
        <SearchBtn />
        <div className='ml-auto' />
      </Header>
      <Main>
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-lg font-semibold text-zinc-950 dark:text-zinc-50">标签管理</h1>
            <p className="text-sm text-zinc-500 mt-1">管理文章标签，组织内容的细粒度分类</p>
          </div>
          <Button className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none rounded-md hover:bg-zinc-800 dark:hover:bg-zinc-200" onClick={openCreateDialog}>
            <Plus className="w-4 h-4 mr-1.5" /> 新建标签
          </Button>
        </div>

        {/* 搜索 */}
        <div className="flex justify-between items-center mb-4">
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400" />
            <Input placeholder="搜索标签…" value={keyword} onChange={(e) => setKeyword(e.target.value)} className="pl-9 border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none" />
          </div>
          <span className="text-xs text-zinc-500">{tags?.length ?? 0} 个标签</span>
        </div>

        {/* 加载中 */}
        {isLoading && <div className="text-center py-16"><Loader2 className="w-5 h-5 animate-spin text-zinc-400 mx-auto" /></div>}

        {/* 空状态 */}
        {!isLoading && tags?.length === 0 && (
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900 py-16">
            <div className="flex flex-col items-center text-center">
              <Hash className="w-10 h-10 text-zinc-300 dark:text-zinc-700 mb-3" />
              <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">暂无标签</p>
              <p className="text-sm text-zinc-500 mt-1">点击上方按钮创建第一个标签</p>
              <Button variant="outline" size="sm" className="mt-3 shadow-none border-zinc-200 dark:border-zinc-800" onClick={openCreateDialog}>新建标签</Button>
            </div>
          </div>
        )}

        {/* 标签墙 */}
        {tags && tags.length > 0 && (
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900 p-5">
            <div className="flex flex-wrap gap-2">
              {tags.map((tag) => (
                <button
                  key={tag.id}
                  className="group relative inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm cursor-pointer transition-all bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700 hover:border-zinc-400 dark:hover:border-zinc-500 hover:shadow-sm"
                  onClick={() => openEditDialog(tag)}
                >
                  <Hash className="w-3 h-3 text-zinc-400" />
                  <span>{tag.name}</span>
                  {tag.postCount > 0 && (
                    <span className="text-[10px] text-zinc-400 ml-0.5">{tag.postCount}</span>
                  )}
                  {/* hover 操作按钮 */}
                  <span className="absolute -top-1.5 -right-1.5 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
                    <a
                      href={`/tags/${tag.slug}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="w-4 h-4 rounded-full bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 flex items-center justify-center hover:bg-blue-500 dark:hover:bg-blue-500 dark:hover:text-white"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <ExternalLink className="w-2.5 h-2.5" />
                    </a>
                    <span
                      className="w-4 h-4 rounded-full bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 flex items-center justify-center cursor-pointer hover:bg-red-500 dark:hover:bg-red-500 dark:hover:text-white"
                      onClick={(e) => handleDelete(e, tag.id)}
                    >
                      <X className="w-2.5 h-2.5" />
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* 新建/编辑标签弹窗 */}
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>{editingTag ? '编辑标签' : '新建标签'}</DialogTitle>
            </DialogHeader>
            <div className="flex flex-col gap-4 py-2">
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-medium text-zinc-500">标签名称 *</label>
                <div className="relative">
                  <Hash className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400" />
                  <Input
                    value={formData.name}
                    onChange={(e) => handleNameChange(e.target.value)}
                    placeholder="例如：Kubernetes"
                    className="pl-9 border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none"
                    autoFocus
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleSubmit() } }}
                  />
                </div>
              </div>
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-medium text-zinc-500">Slug</label>
                <Input
                  value={formData.slug}
                  onChange={(e) => setFormData((p) => ({ ...p, slug: e.target.value }))}
                  placeholder="自动生成"
                  className="border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none"
                />
              </div>
              {formData.name && (
                <div className="flex items-center gap-2">
                  <span className="text-xs text-zinc-500">预览：</span>
                  <span className="text-sm font-medium text-zinc-900 dark:text-zinc-100"># {formData.name}</span>
                </div>
              )}
            </div>
            <DialogFooter>
              <Button variant="outline" className="shadow-none border-zinc-200 dark:border-zinc-800" onClick={() => setDialogOpen(false)}>取消</Button>
              <Button
                className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none hover:bg-zinc-800 dark:hover:bg-zinc-200"
                onClick={handleSubmit}
                disabled={!formData.name.trim() || isPending}
              >
                {isPending ? '处理中…' : editingTag ? '保存修改' : '创建标签'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </Main>
    </>
  )
}
