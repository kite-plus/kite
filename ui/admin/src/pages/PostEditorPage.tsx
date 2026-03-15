import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Button, Input, Tag, Select, Typography, TextArea } from '@douyinfe/semi-ui'
import { IconArrowLeft, IconSave, IconSend, IconTick } from '@douyinfe/semi-icons'
import { TiptapEditor } from '@/components/TiptapEditor'
import { usePostDetail, useSavePost, useCategories } from '@/hooks/use-posts'
import type { PostFormData } from '@/types/post'

const { Title, Text } = Typography

/**
 * 文章编辑页面 — Semi Design
 */
export function PostEditorPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [saved, setSaved] = useState(false)

  const [form, setForm] = useState<PostFormData>({
    title: '', slug: '', summary: '', content: '',
    category: '', tags: [], status: 'draft', coverUrl: '',
  })
  const [tagInput, setTagInput] = useState('')

  const { data: post, isLoading } = usePostDetail(id)
  const { data: categories } = useCategories()
  const saveMutation = useSavePost()

  useEffect(() => {
    if (post) {
      setForm({
        title: post.title, slug: post.slug, summary: post.summary,
        content: post.content, category: post.category, tags: [...post.tags],
        status: post.status, coverUrl: post.coverUrl,
      })
    }
  }, [post])

  function handleTitleChange(title: string) {
    setForm((prev) => ({
      ...prev, title,
      slug: prev.slug || title.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    }))
  }

  function addTag() {
    const tag = tagInput.trim()
    if (tag && !form.tags.includes(tag)) setForm((prev) => ({ ...prev, tags: [...prev.tags, tag] }))
    setTagInput('')
  }

  function removeTag(tag: string) {
    setForm((prev) => ({ ...prev, tags: prev.tags.filter((t) => t !== tag) }))
  }

  function handleSave(publish = false) {
    const data = { ...form, id }
    if (publish) data.status = 'published'
    saveMutation.mutate(data, {
      onSuccess: () => { setSaved(true); setTimeout(() => setSaved(false), 2000); if (!isEdit) navigate('/posts') },
    })
  }

  if (isEdit && isLoading) {
    return <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>
  }

  return (
    <div>
      {/* 顶部操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={() => navigate('/posts')} />
          <div>
            <Title heading={4}>{isEdit ? '编辑文章' : '新建文章'}</Title>
            <Text type="tertiary" size="small">{isEdit ? `正在编辑：${post?.title}` : '撰写新的博客文章'}</Text>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button icon={saved ? <IconTick /> : <IconSave />} theme="light" onClick={() => handleSave(false)} disabled={saveMutation.isPending}>
            {saved ? '已保存' : '保存草稿'}
          </Button>
          <Button icon={<IconSend />} theme="solid" onClick={() => handleSave(true)} disabled={saveMutation.isPending || !form.title.trim()}>
            发布文章
          </Button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 24 }}>
        {/* 左侧编辑区 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Input
            value={form.title}
            onChange={(v) => handleTitleChange(v)}
            placeholder="文章标题"
            size="large"
          />
          <TiptapEditor
            content={form.content}
            onChange={(html) => setForm((prev) => ({ ...prev, content: html }))}
            placeholder="开始写作…"
          />
        </div>

        {/* 右侧元数据面板 */}
        <div style={{ width: 280, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Card title="URL Slug">
            <Input value={form.slug} onChange={(v) => setForm((prev) => ({ ...prev, slug: v }))} placeholder="article-slug" />
          </Card>

          <Card title="分类">
            <Select value={form.category} onChange={(v) => setForm((prev) => ({ ...prev, category: v as string }))} placeholder="选择分类" style={{ width: '100%' }}>
              {categories?.map((c) => <Select.Option key={c} value={c}>{c}</Select.Option>)}
            </Select>
          </Card>

          <Card title="标签">
            <div style={{ display: 'flex', gap: 8 }}>
              <Input
                value={tagInput}
                onChange={(v) => setTagInput(v)}
                onEnterPress={() => addTag()}
                placeholder="输入标签…"
                style={{ flex: 1 }}
              />
              <Button theme="light" size="small" onClick={addTag} disabled={!tagInput.trim()}>添加</Button>
            </div>
            {form.tags.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 12 }}>
                {form.tags.map((tag) => (
                  <Tag key={tag} closable onClose={() => removeTag(tag)} color="blue">{tag}</Tag>
                ))}
              </div>
            )}
          </Card>

          <Card title="摘要">
            <TextArea
              value={form.summary}
              onChange={(v) => setForm((prev) => ({ ...prev, summary: v }))}
              placeholder="文章摘要…"
              rows={3}
            />
          </Card>

          <Card title="封面图">
            <Input
              value={form.coverUrl}
              onChange={(v) => setForm((prev) => ({ ...prev, coverUrl: v }))}
              placeholder="https://example.com/cover.jpg"
            />
            {form.coverUrl && (
              <img src={form.coverUrl} alt="封面预览" style={{ marginTop: 8, width: '100%', borderRadius: 4, border: '1px solid var(--semi-color-border)' }} />
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
