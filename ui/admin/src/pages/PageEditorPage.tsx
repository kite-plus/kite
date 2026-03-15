import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Button, Input, Switch, Typography, InputNumber } from '@douyinfe/semi-ui'
import { IconArrowLeft, IconSave, IconSend, IconTick } from '@douyinfe/semi-icons'
import { TiptapEditor } from '@/components/TiptapEditor'
import { usePageDetail, useSavePage } from '@/hooks/use-pages'
import type { PageFormData } from '@/types/page'

const { Title, Text } = Typography

/**
 * 独立页面编辑器 — 复用 TiptapEditor，简化元数据面板
 */
export function PageEditorPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [saved, setSaved] = useState(false)

  const [form, setForm] = useState<PageFormData>({
    title: '', slug: '', content: '',
    status: 'draft', sortOrder: 10, showInNav: false,
  })

  const { data: page, isLoading } = usePageDetail(id)
  const saveMutation = useSavePage()

  useEffect(() => {
    if (page) {
      setForm({
        title: page.title, slug: page.slug, content: page.content,
        status: page.status, sortOrder: page.sortOrder, showInNav: page.showInNav,
      })
    }
  }, [page])

  function handleTitleChange(title: string) {
    setForm((prev) => ({
      ...prev, title,
      slug: prev.slug || title.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    }))
  }

  function handleSave(publish = false) {
    const data = { ...form, id }
    if (publish) data.status = 'published'
    saveMutation.mutate(data, {
      onSuccess: () => { setSaved(true); setTimeout(() => setSaved(false), 2000); if (!isEdit) navigate('/pages') },
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
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={() => navigate('/pages')} />
          <div>
            <Title heading={4}>{isEdit ? '编辑页面' : '新建页面'}</Title>
            <Text type="tertiary" size="small">{isEdit ? `正在编辑：${page?.title}` : '创建独立页面'}</Text>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button icon={saved ? <IconTick /> : <IconSave />} theme="light" onClick={() => handleSave(false)} disabled={saveMutation.isPending}>
            {saved ? '已保存' : '保存草稿'}
          </Button>
          <Button icon={<IconSend />} theme="solid" onClick={() => handleSave(true)} disabled={saveMutation.isPending || !form.title.trim()}>
            发布页面
          </Button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 24 }}>
        {/* 左侧编辑区 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Input
            value={form.title}
            onChange={(v) => handleTitleChange(v)}
            placeholder="页面标题"
            size="large"
          />
          <TiptapEditor
            content={form.content}
            onChange={(html) => setForm((prev) => ({ ...prev, content: html }))}
            placeholder="开始编写页面内容…"
          />
        </div>

        {/* 右侧元数据面板 — 比文章编辑器更简洁 */}
        <div style={{ width: 280, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Card title="URL Slug">
            <Input
              value={form.slug}
              onChange={(v) => setForm((prev) => ({ ...prev, slug: v }))}
              placeholder="page-slug"
              prefix="/"
            />
            <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
              访问地址：/{form.slug || 'page-slug'}
            </Text>
          </Card>

          <Card title="页面设置">
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <Text strong style={{ fontSize: 13 }}>显示在导航栏</Text>
                  <Text type="tertiary" size="small" style={{ display: 'block' }}>前台顶部导航显示此页面</Text>
                </div>
                <Switch
                  checked={form.showInNav}
                  onChange={(v) => setForm((prev) => ({ ...prev, showInNav: v }))}
                />
              </div>
              <div>
                <Text strong style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>排序优先级</Text>
                <InputNumber
                  value={form.sortOrder}
                  onChange={(v) => setForm((prev) => ({ ...prev, sortOrder: v as number }))}
                  min={0}
                  max={999}
                  style={{ width: '100%' }}
                />
                <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 4 }}>数值越小越靠前</Text>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
