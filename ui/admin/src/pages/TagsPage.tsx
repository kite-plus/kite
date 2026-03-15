import { useState } from 'react'
import { Card, Button, Input, Tag, Table, Typography, Empty } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconClose, IconDelete, IconHash, IconArticle } from '@douyinfe/semi-icons'
import { useTagList, useCreateTag, useDeleteTag } from '@/hooks/use-tags'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'

const { Title, Text } = Typography

/**
 * 标签管理页面 — Semi Card / Input / Tag / Button / Table
 */
export function TagsPage() {
  const [keyword, setKeyword] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', slug: '' })
  const [viewMode, setViewMode] = useState<'cloud' | 'list'>('cloud')

  const { data: tags, isLoading } = useTagList(keyword)
  const createMutation = useCreateTag()
  const deleteMutation = useDeleteTag()

  function handleNameChange(name: string) {
    setFormData({ name, slug: name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, '') })
  }

  function handleCreate() {
    if (!formData.name.trim()) return
    createMutation.mutate(formData, { onSuccess: () => { setFormData({ name: '', slug: '' }); setShowForm(false) } })
  }

  function handleDelete(id: string, name: string) {
    if (window.confirm(`确定删除标签「${name}」吗？`)) deleteMutation.mutate(id)
  }

  function getTagSize(postCount: number, maxCount: number): number {
    if (maxCount === 0) return 14
    const ratio = postCount / maxCount
    if (ratio > 0.7) return 20
    if (ratio > 0.4) return 16
    if (ratio > 0.15) return 14
    return 12
  }

  const maxPostCount = tags ? Math.max(...tags.map((t) => t.postCount), 1) : 1
  const totalPosts = tags ? tags.reduce((sum, t) => sum + t.postCount, 0) : 0

  const columns: ColumnProps[] = [
    {
      title: '标签名称', dataIndex: 'name',
      render: (val: string) => <Text strong><IconHash size="small" /> {val}</Text>,
    },
    { title: 'Slug', dataIndex: 'slug', width: 120, render: (val: string) => <Text type="tertiary">{val}</Text> },
    {
      title: '文章数', dataIndex: 'postCount', width: 100, align: 'center' as const,
      render: (val: number) => <Text type="tertiary"><IconArticle size="small" /> {val}</Text>,
    },
    {
      title: '操作', width: 60, align: 'center' as const,
      render: (_: unknown, record: Record<string, unknown>) => (
        <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" onClick={() => handleDelete(record.id as string, record.name as string)} />
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title heading={4}>标签管理</Title>
          <Text type="tertiary" size="small">管理文章标签，组织内容的细粒度分类</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => setShowForm(true)}>新建标签</Button>
      </div>

      {showForm && (
        <Card style={{ marginBottom: 24 }} title="新建标签" headerExtraContent={<Button icon={<IconClose />} theme="borderless" size="small" onClick={() => setShowForm(false)} />}>
          <div style={{ display: 'flex', gap: 16, alignItems: 'flex-end' }}>
            <div style={{ flex: 1 }}>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>标签名称</Text>
              <Input value={formData.name} onChange={(v) => handleNameChange(v)} placeholder="例如：Kubernetes" />
            </div>
            <div style={{ flex: 1 }}>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>Slug</Text>
              <Input value={formData.slug} onChange={(v) => setFormData((p) => ({ ...p, slug: v }))} placeholder="例如：kubernetes" />
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Button theme="light" onClick={() => setShowForm(false)}>取消</Button>
              <Button theme="solid" onClick={handleCreate} disabled={!formData.name.trim() || createMutation.isPending}>
                {createMutation.isPending ? '创建中…' : '创建'}
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* 搜索 + 视图切换 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Input prefix={<IconSearch />} placeholder="搜索标签…" value={keyword} onChange={(v) => setKeyword(v)} style={{ maxWidth: 360 }} showClear />
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Text type="tertiary" size="small">{tags?.length ?? 0} 个标签 · {totalPosts} 次引用</Text>
          <Button.Group>
            <Button theme={viewMode === 'cloud' ? 'solid' : 'light'} size="small" icon={<IconHash />} onClick={() => setViewMode('cloud')} />
            <Button theme={viewMode === 'list' ? 'solid' : 'light'} size="small" icon={<IconArticle />} onClick={() => setViewMode('list')} />
          </Button.Group>
        </div>
      </div>

      {isLoading && <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>}

      {!isLoading && tags?.length === 0 && (
        <Card><Empty image={<IconHash style={{ fontSize: 48 }} />} description="暂无标签，点击上方按钮创建" /></Card>
      )}

      {/* 标签云 */}
      {tags && tags.length > 0 && viewMode === 'cloud' && (
        <Card>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10 }}>
            {tags.map((tag) => (
              <Tag
                key={tag.id}
                closable
                onClose={() => handleDelete(tag.id, tag.name)}
                color="blue"
                size="large"
                style={{ fontSize: getTagSize(tag.postCount, maxPostCount), cursor: 'default' }}
              >
                # {tag.name} <Text type="tertiary" size="small" style={{ marginLeft: 4 }}>{tag.postCount}</Text>
              </Tag>
            ))}
          </div>
        </Card>
      )}

      {/* 列表 */}
      {tags && tags.length > 0 && viewMode === 'list' && (
        <Card bodyStyle={{ padding: 0 }}>
          <Table columns={columns} dataSource={tags} pagination={false} rowKey="id" />
        </Card>
      )}
    </div>
  )
}
