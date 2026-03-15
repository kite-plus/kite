import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Table, Button, Input, Tag, Select, Card, Typography, Pagination } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconEdit, IconDelete, IconEyeOpened, IconComment } from '@douyinfe/semi-icons'
import { usePosts, useCategories } from '@/hooks/use-posts'
import type { PostStatus } from '@/types/post'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'

const { Title, Text } = Typography

/** 状态标签配置 */
const statusConfig: Record<PostStatus, { label: string; color: string }> = {
  published: { label: '已发布', color: 'blue' },
  draft: { label: '草稿', color: 'grey' },
  archived: { label: '已归档', color: 'yellow' },
}

/** 格式化日期 */
function formatDate(dateStr: string | null) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

/**
 * 文章管理页面 — Semi Table / Input / Select / Tag / Button
 */
export function PostsPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<PostStatus | 'all'>('all')
  const [category, setCategory] = useState('')
  const [page, setPage] = useState(1)
  const pageSize = 5

  const { data, isLoading } = usePosts({ page, pageSize, keyword, status, category })
  const { data: categories } = useCategories()

  const columns: ColumnProps[] = [
    {
      title: '标题', dataIndex: 'title', render: (_: unknown, record: Record<string, unknown>) => (
        <div>
          <Text strong ellipsis={{ showTooltip: true }} style={{ width: 300 }}>{record.title as string}</Text>
          <br />
          <Text type="tertiary" size="small" ellipsis={{ showTooltip: true }} style={{ width: 300 }}>{record.summary as string}</Text>
        </div>
      ),
    },
    { title: '分类', dataIndex: 'category', width: 100 },
    {
      title: '状态', dataIndex: 'status', width: 80, render: (val: PostStatus) => {
        const cfg = statusConfig[val]
        return <Tag color={cfg.color}>{cfg.label}</Tag>
      },
    },
    {
      title: '浏览', dataIndex: 'viewCount', width: 80, align: 'center' as const,
      render: (val: number) => <Text type="tertiary"><IconEyeOpened size="small" /> {val}</Text>,
    },
    {
      title: '评论', dataIndex: 'commentCount', width: 80, align: 'center' as const,
      render: (val: number) => <Text type="tertiary"><IconComment size="small" /> {val}</Text>,
    },
    {
      title: '更新日期', dataIndex: 'updatedAt', width: 110,
      render: (val: string) => <Text type="tertiary" size="small">{formatDate(val)}</Text>,
    },
    {
      title: '操作', width: 90, align: 'center' as const,
      render: (_: unknown, record: Record<string, unknown>) => (
        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
          <Button icon={<IconEdit />} theme="borderless" size="small" onClick={() => navigate(`/posts/${record.id}/edit`)} />
          <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" />
        </div>
      ),
    },
  ]

  return (
    <div>
      {/* 标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title heading={4}>文章管理</Title>
          <Text type="tertiary" size="small">管理你的所有博客文章</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => navigate('/posts/new')}>新建文章</Button>
      </div>

      {/* 搜索与筛选 */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索文章标题、摘要或标签…"
          value={keyword}
          onChange={(v) => { setKeyword(v); setPage(1) }}
          style={{ flex: 1, minWidth: 240 }}
          showClear
        />
        <Select value={status} onChange={(v) => { setStatus(v as PostStatus | 'all'); setPage(1) }} style={{ width: 130 }}>
          <Select.Option value="all">全部状态</Select.Option>
          <Select.Option value="published">已发布</Select.Option>
          <Select.Option value="draft">草稿</Select.Option>
          <Select.Option value="archived">已归档</Select.Option>
        </Select>
        <Select value={category || 'all'} onChange={(v) => { setCategory(v === 'all' ? '' : v as string); setPage(1) }} style={{ width: 130 }}>
          <Select.Option value="all">全部分类</Select.Option>
          {categories?.map((c) => (
            <Select.Option key={c} value={c}>{c}</Select.Option>
          ))}
        </Select>
      </div>

      {/* 文章列表 */}
      <Card bodyStyle={{ padding: 0 }}>
        <Table
          columns={columns}
          dataSource={data?.list || []}
          loading={isLoading}
          pagination={false}
          rowKey="id"
          empty={<Text type="tertiary" style={{ padding: 40 }}>没有找到匹配的文章</Text>}
        />
      </Card>

      {/* 分页 */}
      {data && data.total > 0 && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 16 }}>
          <Text type="tertiary" size="small">共 {data.total} 篇文章</Text>
          <Pagination total={data.total} pageSize={pageSize} currentPage={page} onPageChange={setPage} />
        </div>
      )}
    </div>
  )
}
