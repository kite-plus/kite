import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import {
  Popover, PopoverContent, PopoverTrigger,
} from '@/components/ui/popover'
import { CategoryCascader, buildCascaderTree } from '@/components/category-cascader'
import { Calendar } from '@/components/ui/calendar'
import { ArrowLeft, Save, Send, Check, X, Loader2, Sparkles, Clock, Search, Tags, Plus } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { TiptapEditor } from '@/components/TiptapEditor'
import { usePostDetail, useSavePost } from '@/hooks/use-posts'
import { useCategoryList } from '@/hooks/use-categories'
import { useTagList, useCreateTag } from '@/hooks/use-tags'
import { ImageUploader } from '@/components/ImageUploader'
import type { PostFormData } from '@/types/post'
import { toast } from 'sonner'


/**
 * 文章编辑器 — Vercel 风格
 */
export function PostEditorPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [saved, setSaved] = useState(false)
  const [tagPopoverOpen, setTagPopoverOpen] = useState(false)
  const [tagSearch, setTagSearch] = useState('')
  const [aiSummaryLoading, setAiSummaryLoading] = useState(false)
  const [aiSummary, setAiSummary] = useState('')
  const [aiKeywordsLoading, setAiKeywordsLoading] = useState(false)
  const [aiKeywords, setAiKeywords] = useState<string[]>([])

  const [form, setForm] = useState<PostFormData>({
    title: '', slug: '', summary: '', contentMarkdown: '', contentHtml: '',
    categoryId: '', tagIds: [], status: 'draft', coverImage: '', password: '',
  })

  const { data: post, isLoading } = usePostDetail(id)
  const { data: categories } = useCategoryList()
  const { data: allTags } = useTagList()
  const saveMutation = useSavePost()
  const createTagMutation = useCreateTag()

  /** 快速创建标签并自动选中 */
  function handleQuickCreateTag() {
    const name = tagSearch.trim()
    if (!name) return
    const slug = name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, '')
    createTagMutation.mutate({ name, slug }, {
      onSuccess: (newTag) => {
        setForm((prev) => ({ ...prev, tagIds: [...prev.tagIds, newTag.id] }))
        setTagSearch('')
        toast.success(`标签「${name}」已创建`)
      },
      onError: (err) => {
        toast.error('创建标签失败', { description: err.message || '请稍后重试' })
      },
    })
  }

  useEffect(() => {
    if (post) {
      setForm({
        title: post.title, slug: post.slug, summary: post.summary,
        contentMarkdown: post.contentMarkdown || '', contentHtml: post.contentHtml || '',
        categoryId: post.categoryId || '', tagIds: post.tags?.map((t) => t.id) || [],
        status: post.status, coverImage: post.coverImage || '',
        password: (post as unknown as { password?: string })?.password || '',
      })
    }
  }, [post])

  function handleTitleChange(title: string) {
    setForm((prev) => ({ ...prev, title, slug: prev.slug || title.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, '') }))
  }
  function removeTag(tagId: string) { setForm((prev) => ({ ...prev, tagIds: prev.tagIds.filter((id) => id !== tagId) }) ) }
  function handleSave(publish = false, schedule = false) {
    const data = { ...form, id }
    if (schedule && form.publishAt) {
      data.status = 'scheduled'
    } else if (publish) {
      data.status = 'published'
    }
    saveMutation.mutate(data, {
      onSuccess: () => {
        setSaved(true); setTimeout(() => setSaved(false), 2000)
        if (schedule) {
          toast.success('定时发布已设定', { description: `「${form.title}」将在指定时间自动发布` })
        } else if (publish) {
          toast.success('文章已发布', { description: `「${form.title}」已成功发布` })
        } else {
          toast.success('草稿已保存')
        }
        if (!isEdit) navigate('/posts')
      },
      onError: (err) => {
        toast.error(publish ? '发布失败' : '保存失败', { description: err.message || '请稍后重试' })
      },
    })
  }
  function handleAiSummary() {
    setAiSummaryLoading(true); setAiSummary('')
    setTimeout(() => { setAiSummary('Kite 是一个轻量级 Go 博客引擎，内置 AI 写作助手、富文本编辑器和 CSR 主题，提供极致极简的写作体验…'); setAiSummaryLoading(false) }, 800)
  }
  function handleAiKeywords() {
    setAiKeywordsLoading(true); setAiKeywords([])
    setTimeout(() => { setAiKeywords(['博客', '技术', 'Go', 'React', '静态网站', '极简风']); setAiKeywordsLoading(false) }, 800)
  }

  if (isEdit && isLoading) return <div className="flex justify-center py-16"><Loader2 className="w-4 h-4 animate-spin text-zinc-400" /></div>

  return (
    <>
      <Header fixed>
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="icon" className="w-8 h-8 rounded-md" onClick={() => navigate('/posts')}><ArrowLeft className="w-4 h-4" /></Button>
          <div className="hidden sm:block">
            <h1 className="text-sm font-semibold text-zinc-900 dark:text-zinc-50 tracking-tight">{isEdit ? '编辑文章' : '新建文章'}</h1>
            <p className="text-xs text-zinc-500">{isEdit ? `正在编辑：${post?.title}` : '撰写新的博客文章'}</p>
          </div>
        </div>
        <div className="flex gap-2 ml-auto">
          <Button variant="outline" className="shadow-sm border-zinc-200 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50" onClick={() => handleSave(false)} disabled={saveMutation.isPending}>
            {saved ? <><Check className="w-4 h-4 sm:mr-1.5" /> <span className="hidden sm:inline">已保存</span></> : <><Save className="w-4 h-4 sm:mr-1.5" /> <span className="hidden sm:inline">保存草稿</span></>}
          </Button>
          {/* 定时发布 */}
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="outline" className="shadow-sm border-zinc-200 dark:border-zinc-700 rounded-md bg-white dark:bg-zinc-900 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-50" disabled={saveMutation.isPending || !form.title.trim()}>
                <Clock className="w-4 h-4 sm:mr-1.5" /> <span className="hidden sm:inline">定时发布</span>
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-auto p-0" align="end">
              <div className="p-3 space-y-3">
                <p className="text-sm font-medium px-1">选择发布时间</p>
                <Calendar
                  mode="single"
                  selected={form.publishAt ? new Date(form.publishAt) : undefined}
                  onSelect={(date: Date | undefined) => {
                    if (!date) return
                    const prev = form.publishAt ? new Date(form.publishAt) : new Date()
                    date.setHours(prev.getHours(), prev.getMinutes())
                    setForm((p) => ({ ...p, publishAt: date.toISOString() }))
                  }}
                  disabled={(date: Date) => date < new Date(new Date().setHours(0, 0, 0, 0))}
                  initialFocus
                />
                <div className="flex items-center gap-2 px-1">
                  <Clock className="w-4 h-4 text-muted-foreground shrink-0" />
                  <Input
                    type="time"
                    value={form.publishAt ? `${String(new Date(form.publishAt).getHours()).padStart(2, '0')}:${String(new Date(form.publishAt).getMinutes()).padStart(2, '0')}` : ''}
                    onChange={(e) => {
                      const [h, m] = e.target.value.split(':').map(Number)
                      const d = form.publishAt ? new Date(form.publishAt) : new Date()
                      d.setHours(h, m)
                      setForm((p) => ({ ...p, publishAt: d.toISOString() }))
                    }}
                    className="flex-1 border-zinc-200 dark:border-zinc-700 shadow-none rounded-md text-sm h-9"
                  />
                </div>
                {form.publishAt && (
                  <p className="text-xs text-muted-foreground px-1">
                    文章将在 {new Date(form.publishAt).toLocaleString('zh-CN')} 自动发布
                  </p>
                )}
                <Button
                  className="w-full"
                  disabled={!form.publishAt || saveMutation.isPending}
                  onClick={() => handleSave(false, true)}
                >
                  <Clock className="w-4 h-4 mr-1.5" /> 设定定时发布
                </Button>
              </div>
            </PopoverContent>
          </Popover>
          <Button className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-md shadow-sm hover:bg-zinc-800 dark:hover:bg-zinc-200" onClick={() => handleSave(true)} disabled={saveMutation.isPending || !form.title.trim()}>
            <Send className="w-4 h-4 sm:mr-1.5" /> <span className="hidden sm:inline">发布文章</span>
          </Button>
        </div>
      </Header>

      <Main>
      <div className="flex gap-0 relative">
        <div className="flex-1 flex flex-col gap-4 pr-6">
          {/* 沉浸式标题 — 原生 input，零 border */}
          <div className="mb-2">
            <input
              type="text"
              value={form.title}
              onChange={(e) => handleTitleChange(e.target.value)}
              placeholder="输入文章标题..."
              className="w-full bg-white dark:bg-zinc-900 text-2xl font-bold tracking-tight text-zinc-900 dark:text-zinc-50 placeholder:text-zinc-400 dark:placeholder:text-zinc-600 border border-zinc-200 dark:border-zinc-700 rounded-lg outline-none focus:ring-2 focus:ring-zinc-900/10 dark:focus:ring-zinc-100/10 focus:border-zinc-300 dark:focus:border-zinc-600 px-4 py-3 shadow-sm transition-colors"
            />
          </div>
          <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-sm overflow-hidden">
            <TiptapEditor key={post?.id || 'new'} content={post ? (post.contentHtml || post.contentMarkdown || '') : ''} onChange={(html, markdown) => setForm((prev) => ({ ...prev, contentHtml: html, contentMarkdown: markdown }))} placeholder="开始写作…" />
          </div>
        </div>

        {/* 右侧属性面板：无外边框，纯白底色 + border-l 分割 */}
        <aside className="w-80 shrink-0 bg-white dark:bg-zinc-900 flex flex-col gap-6 p-6 border-l border-zinc-100 dark:border-zinc-800">

          {/* URL Slug */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">URL Slug</label>
            <Input value={form.slug} onChange={(e) => setForm((prev) => ({ ...prev, slug: e.target.value }))} placeholder="article-slug" className="shadow-none rounded-md border-zinc-200 dark:border-zinc-700" />
          </div>

          {/* 分类 */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">分类</label>
            <CategoryCascader
              options={buildCascaderTree(categories || [])}
              value={form.categoryId || null}
              onChange={(id) => setForm((prev) => ({ ...prev, categoryId: id || '' }))}
              allowSelectParent
            />
          </div>

          {/* 标签 */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">标签</label>
            <Popover open={tagPopoverOpen} onOpenChange={(open) => { setTagPopoverOpen(open); if (!open) setTagSearch('') }}>
              <PopoverTrigger asChild>
                <Button variant="outline" className="justify-start shadow-none rounded-md border-zinc-200 dark:border-zinc-700 text-zinc-500 font-normal hover:bg-zinc-50 dark:hover:bg-zinc-800">
                  <Tags className="w-4 h-4 mr-1.5 shrink-0" />
                  {form.tagIds.length > 0 ? `已选 ${form.tagIds.length} 个标签` : '选择标签…'}
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-64 p-0" align="start">
                <div className="p-2 border-b border-zinc-100 dark:border-zinc-800">
                  <div className="relative">
                    <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-zinc-400" />
                    <Input
                      value={tagSearch}
                      onChange={(e) => setTagSearch(e.target.value)}
                      placeholder="搜索或创建标签…"
                      className="pl-8 h-8 text-sm shadow-none border-zinc-200 dark:border-zinc-700"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') {
                          e.preventDefault()
                          const filtered = allTags?.filter((t) => t.name.toLowerCase().includes(tagSearch.toLowerCase())) || []
                          // 如果有精确匹配，勾选它；否则创建新标签
                          const exact = filtered.find((t) => t.name.toLowerCase() === tagSearch.toLowerCase())
                          if (exact) {
                            if (!form.tagIds.includes(exact.id)) setForm((prev) => ({ ...prev, tagIds: [...prev.tagIds, exact.id] }))
                            setTagSearch('')
                          } else if (tagSearch.trim()) {
                            handleQuickCreateTag()
                          }
                        }
                      }}
                    />
                  </div>
                </div>
                <div className="max-h-52 overflow-y-auto p-2">
                  <div className="flex flex-wrap gap-1.5">
                    {allTags?.filter((t) => t.name.toLowerCase().includes(tagSearch.toLowerCase())).map((tag) => {
                      const selected = form.tagIds.includes(tag.id)
                      return (
                        <button
                          key={tag.id}
                          type="button"
                          className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs cursor-pointer transition-all border ${
                            selected
                              ? 'bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 border-zinc-900 dark:border-zinc-100'
                              : 'bg-zinc-50 dark:bg-zinc-800/50 text-zinc-600 dark:text-zinc-400 border-zinc-200 dark:border-zinc-700 hover:border-zinc-400 dark:hover:border-zinc-500'
                          }`}
                          onClick={() => {
                            if (selected) {
                              removeTag(tag.id)
                            } else {
                              setForm((prev) => ({ ...prev, tagIds: [...prev.tagIds, tag.id] }))
                            }
                          }}
                        >
                          # {tag.name}
                          {selected && <X className="w-3 h-3 ml-0.5" />}
                        </button>
                      )
                    })}
                  </div>
                  {allTags?.filter((t) => t.name.toLowerCase().includes(tagSearch.toLowerCase())).length === 0 && (
                    <p className="text-xs text-zinc-400 text-center py-3">无匹配标签</p>
                  )}
                </div>
                {/* 快速创建标签 */}
                {tagSearch.trim() && !allTags?.some((t) => t.name.toLowerCase() === tagSearch.trim().toLowerCase()) && (
                  <div className="border-t border-zinc-100 dark:border-zinc-800 p-1">
                    <button
                      className="flex items-center gap-2 w-full px-2 py-1.5 rounded-md text-sm text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors cursor-pointer"
                      onClick={handleQuickCreateTag}
                      disabled={createTagMutation.isPending}
                    >
                      <Plus className="w-3.5 h-3.5 text-zinc-400" />
                      {createTagMutation.isPending ? '创建中…' : <>创建 <span className="font-medium text-zinc-900 dark:text-zinc-100">"{tagSearch.trim()}"</span></>}
                    </button>
                  </div>
                )}
              </PopoverContent>
            </Popover>
          </div>

          {/* 摘要 + AI */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <div className="flex justify-between items-center">
              <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">摘要</label>
              <button className="text-xs text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-200 flex items-center gap-1 cursor-pointer transition-colors" onClick={handleAiSummary} disabled={aiSummaryLoading}><Sparkles className="w-3 h-3" /> {aiSummaryLoading ? '…' : 'AI 生成'}</button>
            </div>
            <Textarea value={form.summary} onChange={(e) => setForm((prev) => ({ ...prev, summary: e.target.value }))} placeholder="文章摘要…" rows={3} className="shadow-none rounded-md border-zinc-200 dark:border-zinc-700 resize-none" />
            {aiSummary && (<div className="mt-1"><p className="text-xs text-zinc-500 leading-relaxed">{aiSummary}</p><button className="text-xs text-zinc-900 dark:text-zinc-100 underline underline-offset-2 cursor-pointer mt-1 hover:no-underline" onClick={() => { setForm((prev) => ({ ...prev, summary: aiSummary })); setAiSummary('') }}>应用此摘要</button></div>)}
          </div>

          {/* SEO 关键词 */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">SEO 关键词</label>
            <div className="flex gap-2"><Input value={(form as unknown as { keywords?: string }).keywords || ''} onChange={(e) => setForm((prev) => ({ ...prev, keywords: e.target.value } as unknown as PostFormData))} placeholder="博客,技术,Go" className="flex-1 shadow-none rounded-md border-zinc-200 dark:border-zinc-700" /><Button variant="outline" size="sm" className="rounded-md shadow-none border-zinc-200 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 shrink-0 gap-1 text-xs h-9 hover:bg-zinc-50 dark:hover:bg-zinc-800" onClick={handleAiKeywords} disabled={aiKeywordsLoading}><Sparkles className="w-3 h-3" /> {aiKeywordsLoading ? '…' : 'AI'}</Button></div>
            {aiKeywords.length > 0 && <p className="text-xs text-zinc-500 mt-1">建议：<span className="text-zinc-700 dark:text-zinc-300">{aiKeywords.join(', ')}</span></p>}
          </div>

          {/* 封面图 */}
          <div className="flex flex-col gap-2 pb-6 border-b border-zinc-100 dark:border-zinc-800">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">封面图</label>
            <ImageUploader value={form.coverImage} onChange={(url) => setForm((prev) => ({ ...prev, coverImage: url }))} placeholder="上传封面图片" />
          </div>

          {/* 文章密码 */}
          <div className="flex flex-col gap-2">
            <label className="text-xs font-semibold text-zinc-500 uppercase tracking-wider">🔒 文章密码</label>
            <Input type="password" value={form.password} onChange={(e) => setForm((prev) => ({ ...prev, password: e.target.value }))} placeholder="留空则不加密" className="shadow-none rounded-md border-zinc-200 dark:border-zinc-700" />
            <p className="text-xs text-zinc-500">设置后文章需密码查看</p>
          </div>

        </aside>
      </div>
      </Main>
    </>
  )
}
