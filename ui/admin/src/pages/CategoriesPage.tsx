import { useState } from 'react'
import { Card, Button, Input, Typography, Divider, Empty } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconEdit, IconDelete, IconClose, IconArticle, IconFolder } from '@douyinfe/semi-icons'
import { useCategoryList, useCreateCategory, useDeleteCategory } from '@/hooks/use-categories'

const { Title, Text } = Typography

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

/**
 * 分类管理页面 — Semi Card / Input / Button
 */
export function CategoriesPage() {
  const [keyword, setKeyword] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', slug: '', description: '' })

  const { data: categories, isLoading } = useCategoryList(keyword)
  const createMutation = useCreateCategory()
  const deleteMutation = useDeleteCategory()

  function handleNameChange(name: string) {
    setFormData((prev) => ({ ...prev, name, slug: name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, '') }))
  }

  function handleCreate() {
    if (!formData.name.trim()) return
    createMutation.mutate(formData, { onSuccess: () => { setFormData({ name: '', slug: '', description: '' }); setShowForm(false) } })
  }

  function handleDelete(id: string, name: string) {
    if (window.confirm(`确定删除分类「${name}」吗？此操作不可撤销。`)) deleteMutation.mutate(id)
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title heading={4}>分类管理</Title>
          <Text type="tertiary" size="small">管理博客文章的分类体系</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => setShowForm(true)}>新建分类</Button>
      </div>

      {/* 新建表单 */}
      {showForm && (
        <Card style={{ marginBottom: 24 }} title="新建分类" headerExtraContent={<Button icon={<IconClose />} theme="borderless" size="small" onClick={() => setShowForm(false)} />}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <div>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>分类名称</Text>
              <Input value={formData.name} onChange={(v) => handleNameChange(v)} placeholder="例如：前端开发" />
            </div>
            <div>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>Slug</Text>
              <Input value={formData.slug} onChange={(v) => setFormData((prev) => ({ ...prev, slug: v }))} placeholder="例如：frontend" />
            </div>
          </div>
          <div style={{ marginTop: 16 }}>
            <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>描述</Text>
            <Input value={formData.description} onChange={(v) => setFormData((prev) => ({ ...prev, description: v }))} placeholder="简要描述此分类的内容范围…" />
          </div>
          <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button theme="light" onClick={() => setShowForm(false)}>取消</Button>
            <Button theme="solid" onClick={handleCreate} disabled={!formData.name.trim() || createMutation.isPending}>
              {createMutation.isPending ? '创建中…' : '创建'}
            </Button>
          </div>
        </Card>
      )}

      {/* 搜索栏 */}
      <div style={{ marginBottom: 16, maxWidth: 360 }}>
        <Input prefix={<IconSearch />} placeholder="搜索分类名称或 Slug…" value={keyword} onChange={(v) => setKeyword(v)} showClear />
      </div>

      {isLoading && <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>}

      {!isLoading && categories?.length === 0 && (
        <Card>
          <Empty image={<IconFolder style={{ fontSize: 48 }} />} description="暂无分类，点击上方按钮创建第一个分类" />
        </Card>
      )}

      {/* 分类卡片网格 */}
      {categories && categories.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
          {categories.map((cat) => (
            <Card key={cat.id}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div style={{ minWidth: 0, flex: 1 }}>
                  <Text strong>{cat.name}</Text>
                  <br />
                  <Text type="tertiary" size="small">/{cat.slug}</Text>
                </div>
                <div style={{ display: 'flex', gap: 4 }}>
                  <Button icon={<IconEdit />} theme="borderless" size="small" />
                  <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" onClick={() => handleDelete(cat.id, cat.name)} />
                </div>
              </div>
              <Text type="tertiary" style={{ display: 'block', marginTop: 12, fontSize: 14 }}>{cat.description}</Text>
              <Divider margin={12} />
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text type="tertiary" size="small"><IconArticle size="small" /> {cat.postCount} 篇文章</Text>
                <Text type="tertiary" size="small">更新于 {formatDate(cat.updatedAt)}</Text>
              </div>
            </Card>
          ))}
        </div>
      )}

      {categories && categories.length > 0 && (
        <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 16 }}>共 {categories.length} 个分类</Text>
      )}
    </div>
  )
}
