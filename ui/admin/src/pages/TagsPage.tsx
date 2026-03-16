import { useState } from 'react'
import { Card, Button, ButtonGroup, Input, Tag, Table, Typography, Empty, Tooltip, Popconfirm, Divider } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconClose, IconDelete, IconHash, IconArticle, IconList } from '@douyinfe/semi-icons'
import { useTagList, useCreateTag, useDeleteTag } from '@/hooks/use-tags'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'
import type { TagColor } from '@douyinfe/semi-ui/lib/es/tag/interface'

const { Title, Text } = Typography

/** 根据引用数获取标签颜色 */
function getTagColor(postCount: number, maxCount: number): TagColor {
  if (maxCount === 0) return 'blue'
  const ratio = postCount / maxCount
  if (ratio > 0.7) return 'blue'
  if (ratio > 0.4) return 'cyan'
  if (ratio > 0.15) return 'light-blue'
  return 'grey'
}

/** 根据引用数获取字号 */
function getTagFontSize(postCount: number, maxCount: number): number {
  if (maxCount === 0) return 14
  const ratio = postCount / maxCount
  if (ratio > 0.7) return 22
  if (ratio > 0.4) return 18
  if (ratio > 0.15) return 15
  return 13
}

/**
 * 标签管理页面 — 标签云 + 列表视图
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
    deleteMutation.mutate(id)
    void name // 消除 unused 警告
  }

  const maxPostCount = tags ? Math.max(...tags.map((t) => t.postCount), 1) : 1
  const totalPosts = tags ? tags.reduce((sum, t) => sum + t.postCount, 0) : 0
  const hotTags = tags ? tags.filter((t) => t.postCount >= 3).length : 0
  const emptyTags = tags ? tags.filter((t) => t.postCount === 0).length : 0

  const columns: ColumnProps[] = [
    {
      title: '标签名称', dataIndex: 'name',
      render: (val: string) => (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Tag color="blue" size="small"># {val}</Tag>
        </div>
      ),
    },
    {
      title: 'Slug', dataIndex: 'slug', width: 160,
      render: (val: string) => <Text type="tertiary" size="small" copyable>{val}</Text>,
    },
    {
      title: '文章数', dataIndex: 'postCount', width: 100, align: 'center' as const,
      sorter: (a: Record<string, number>, b: Record<string, number>) => a.postCount - b.postCount,
      render: (val: number) => (
        <Text type={val > 0 ? 'primary' : 'tertiary'} size="small" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4 }}>
          <IconArticle size="small" /> {val}
        </Text>
      ),
    },
    {
      title: '热度', dataIndex: 'postCount', width: 120,
      render: (val: number) => {
        const pct = maxPostCount > 0 ? Math.round((val / maxPostCount) * 100) : 0
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div style={{ flex: 1, height: 6, background: 'var(--semi-color-fill-1)', borderRadius: 3, overflow: 'hidden' }}>
              <div style={{ width: `${pct}%`, height: '100%', background: pct > 70 ? 'var(--semi-color-primary)' : pct > 30 ? 'var(--semi-color-info)' : 'var(--semi-color-fill-2)', borderRadius: 3, transition: 'width 0.3s' }} />
            </div>
            <Text type="tertiary" size="small">{pct}%</Text>
          </div>
        )
      },
    },
    {
      title: '操作', width: 80, align: 'center' as const,
      render: (_: unknown, record: Record<string, unknown>) => (
        <Popconfirm
          title="确认删除"
          content={`确定要删除标签「${record.name}」吗？`}
          onConfirm={() => handleDelete(record.id as string, record.name as string)}
          position="topRight"
        >
          <Tooltip content="删除标签" position="top">
            <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" />
          </Tooltip>
        </Popconfirm>
      ),
    },
  ]

  return (
    <div>
      {/* 标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title heading={4} style={{ marginBottom: 4 }}>标签管理</Title>
          <Text type="tertiary" size="small">管理文章标签，组织内容的细粒度分类</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => setShowForm(true)}>新建标签</Button>
      </div>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 12, marginBottom: 20 }}>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">标签总数</Text>
          <Title heading={3} style={{ margin: '4px 0 0' }}>{tags?.length ?? 0}</Title>
        </Card>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">总引用次数</Text>
          <Title heading={3} style={{ margin: '4px 0 0' }}>{totalPosts}</Title>
        </Card>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">热门标签</Text>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
            <Title heading={3} style={{ margin: '4px 0 0' }}>{hotTags}</Title>
            <Text type="tertiary" size="small">≥3 篇文章</Text>
          </div>
        </Card>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">空标签</Text>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6 }}>
            <Title heading={3} style={{ margin: '4px 0 0', color: emptyTags > 0 ? 'var(--semi-color-warning)' : undefined }}>{emptyTags}</Title>
            {emptyTags > 0 && <Text type="warning" size="small">建议清理</Text>}
          </div>
        </Card>
      </div>

      {/* 新建标签表单 */}
      {showForm && (
        <Card
          style={{ marginBottom: 20 }}
          title="新建标签"
          headerExtraContent={<Button icon={<IconClose />} theme="borderless" size="small" onClick={() => setShowForm(false)} />}
        >
          <div style={{ display: 'flex', gap: 16, alignItems: 'flex-end' }}>
            <div style={{ flex: 1 }}>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>标签名称 *</Text>
              <Input
                value={formData.name}
                onChange={(v) => handleNameChange(v)}
                placeholder="例如：Kubernetes"
                prefix={<IconHash />}
                autoFocus
              />
            </div>
            <div style={{ flex: 1 }}>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>Slug（URL 标识）</Text>
              <Input
                value={formData.slug}
                onChange={(v) => setFormData((p) => ({ ...p, slug: v }))}
                placeholder="自动生成，可手动修改"
              />
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Button theme="light" onClick={() => setShowForm(false)}>取消</Button>
              <Button theme="solid" onClick={handleCreate} disabled={!formData.name.trim() || createMutation.isPending}>
                {createMutation.isPending ? '创建中…' : '创建标签'}
              </Button>
            </div>
          </div>
          {formData.name && (
            <div style={{ marginTop: 12 }}>
              <Text type="tertiary" size="small">预览：</Text>{' '}
              <Tag color="blue" size="large"># {formData.name}</Tag>
            </div>
          )}
        </Card>
      )}

      {/* 搜索 + 视图切换 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索标签…"
          value={keyword}
          onChange={(v) => setKeyword(v)}
          style={{ maxWidth: 360 }}
          showClear
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Text type="tertiary" size="small">{tags?.length ?? 0} 个标签 · {totalPosts} 次引用</Text>
          <ButtonGroup>
            <Tooltip content="标签云" position="top">
              <Button theme={viewMode === 'cloud' ? 'solid' : 'light'} size="small" icon={<IconHash />} onClick={() => setViewMode('cloud')} />
            </Tooltip>
            <Tooltip content="列表视图" position="top">
              <Button theme={viewMode === 'list' ? 'solid' : 'light'} size="small" icon={<IconList />} onClick={() => setViewMode('list')} />
            </Tooltip>
          </ButtonGroup>
        </div>
      </div>

      {isLoading && <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>}

      {!isLoading && tags?.length === 0 && (
        <Card>
          <Empty
            image={<IconHash style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            description="暂无标签，点击上方按钮创建第一个标签"
          />
        </Card>
      )}

      {/* 标签云视图 */}
      {tags && tags.length > 0 && viewMode === 'cloud' && (
        <Card>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 10, alignItems: 'center', justifyContent: 'center', padding: '16px 0' }}>
            {tags.map((tag) => (
              <Tooltip key={tag.id} content={`${tag.name} · ${tag.postCount} 篇文章 · slug: ${tag.slug}`} position="top">
                <Tag
                  color={getTagColor(tag.postCount, maxPostCount)}
                  size="large"
                  style={{
                    fontSize: getTagFontSize(tag.postCount, maxPostCount),
                    cursor: 'pointer',
                    padding: '6px 14px',
                    transition: 'transform 0.2s, box-shadow 0.2s',
                    opacity: tag.postCount === 0 ? 0.5 : 1,
                  }}>
                
                  # {tag.name}
                  <Text type="tertiary" size="small" style={{ marginLeft: 6 }}>{tag.postCount}</Text>
                </Tag>
              </Tooltip>
            ))}
          </div>
          <Divider margin={16} />
          <div style={{ display: 'flex', justifyContent: 'center', gap: 24 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Tag color="blue" size="small">热门</Tag>
              <Text type="tertiary" size="small">≥ 70% 引用</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Tag color="cyan" size="small">活跃</Tag>
              <Text type="tertiary" size="small">40-70%</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Tag color="light-blue" size="small">普通</Tag>
              <Text type="tertiary" size="small">15-40%</Text>
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Tag color="grey" size="small">冷门</Tag>
              <Text type="tertiary" size="small">&lt; 15%</Text>
            </div>
          </div>
        </Card>
      )}

      {/* 列表视图 */}
      {tags && tags.length > 0 && viewMode === 'list' && (
        <Card bodyStyle={{ padding: 0 }}>
          <Table
            columns={columns}
            dataSource={tags}
            pagination={false}
            rowKey="id"
          />
        </Card>
      )}
    </div>
  )
}
