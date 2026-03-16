import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Button, Input, Tag, Select, Typography, TextArea } from '@douyinfe/semi-ui'
import { IconArrowLeft, IconSave, IconSend, IconTick } from '@douyinfe/semi-icons'
import { TiptapEditor } from '@/components/TiptapEditor'
import { usePostDetail, useSavePost } from '@/hooks/use-posts'
import { useCategoryList } from '@/hooks/use-categories'
import { useTagList } from '@/hooks/use-tags'
import type { PostFormData } from '@/types/post'
import { ImageUploader } from '@/components/ImageUploader'

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
    title: '', slug: '', summary: '', contentMarkdown: '', contentHtml: '',
    categoryId: '', tagIds: [], status: 'draft', coverImage: '', password: '',
  })

  const { data: post, isLoading } = usePostDetail(id)
  const { data: categories } = useCategoryList()
  const { data: allTags } = useTagList()
  const saveMutation = useSavePost()

  useEffect(() => {
    if (post) {
      setForm({
        title: post.title, slug: post.slug, summary: post.summary,
        contentMarkdown: post.contentMarkdown || '',
        contentHtml: post.contentHtml || '',
        categoryId: post.categoryId || '',
        tagIds: post.tags?.map((t) => t.id) || [],
        status: post.status, coverImage: post.coverImage || '',
        password: (post as unknown as { password?: string })?.password || '',
      })
    }
  }, [post])

  function handleTitleChange(title: string) {
    setForm((prev) => ({
      ...prev, title,
      slug: prev.slug || title.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    }))
  }

  function removeTag(tagId: string) {
    setForm((prev) => ({ ...prev, tagIds: prev.tagIds.filter((id) => id !== tagId) }))
  }

  /** 根据 tagId 获取标签名称 */
  function getTagName(tagId: string): string {
    return allTags?.find((t) => t.id === tagId)?.name || tagId
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
            key={post?.id || 'new'}
            content={post ? (post.contentHtml || post.contentMarkdown || '') : ''}
            onChange={(html, markdown) => setForm((prev) => ({ ...prev, contentHtml: html, contentMarkdown: markdown }))}
            placeholder="开始写作…"
          />
        </div>

        {/* 右侧元数据面板 */}
        <div style={{ width: 280, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Card title="URL Slug">
            <Input value={form.slug} onChange={(v) => setForm((prev) => ({ ...prev, slug: v }))} placeholder="article-slug" />
          </Card>

          <Card title="分类">
            <Select
              value={form.categoryId || undefined}
              onChange={(v) => setForm((prev) => ({ ...prev, categoryId: v as string }))}
              placeholder="选择分类"
              style={{ width: '100%' }}
            >
              {categories?.map((c) => <Select.Option key={c.id} value={c.id}>{c.name}</Select.Option>)}
            </Select>
          </Card>

          <Card title="标签">
            <div style={{ display: 'flex', gap: 8 }}>
              <Select
                filter
                value={undefined}
                onChange={(v) => {
                  if (v && !form.tagIds.includes(v as string)) {
                    setForm((prev) => ({ ...prev, tagIds: [...prev.tagIds, v as string] }))
                  }
                }}
                placeholder="选择标签…"
                style={{ flex: 1 }}
              >
                {allTags?.filter((t) => !form.tagIds.includes(t.id)).map((t) => (
                  <Select.Option key={t.id} value={t.id}>{t.name}</Select.Option>
                ))}
              </Select>
            </div>
            {form.tagIds.length > 0 && (
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 12 }}>
                {form.tagIds.map((tagId) => (
                  <Tag key={tagId} closable onClose={() => removeTag(tagId)} color="blue">{getTagName(tagId)}</Tag>
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
            <ImageUploader
              value={form.coverImage}
              onChange={(url) => setForm((prev) => ({ ...prev, coverImage: url }))}
              placeholder="上传封面图片"
            />
          </Card>

          <Card title="🔒 文章密码" style={{ marginTop: 16 }}>
            <Input
              value={form.password}
              onChange={(v) => setForm((prev) => ({ ...prev, password: v }))}
              placeholder="留空则不加密"
              type="password"
              mode="password"
            />
            <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 4 }}>
              设置后整篇文章或 :::protected 片段需要密码才能查看
            </Text>
          </Card>
        </div>
      </div>
    </div>
  )
}
