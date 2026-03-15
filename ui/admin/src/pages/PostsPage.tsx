import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Table, Button, Input, Tag, Select, Card, Typography, Pagination, Tooltip, Popconfirm } from '@douyinfe/semi-ui'
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

/** 状态筛选标签 */
const statusTabs = [
  { key: 'all' as const, label: '全部' },
  { key: 'published' as const, label: '已发布' },
  { key: 'draft' as const, label: '草稿' },
  { key: 'archived' as const, label: '已归档' },
]

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
  const [selectedKeys, setSelectedKeys] = useState<string[]>([])
  const pageSize = 10

  const { data, isLoading } = usePosts({ page, pageSize, keyword, status, category })
  const { data: categories } = useCategories()

  const columns: ColumnProps[] = [
    {
      title: '标题', dataIndex: 'title',
      render: (_: unknown, record: Record<string, unknown>) => (
        <div
          style={{ cursor: 'pointer' }}
          onClick={() => navigate(`/posts/${record.id}/edit`)}
        >
          <Text strong style={{ fontSize: 14 }}>{record.title as string}</Text>
          <div style={{ marginTop: 2 }}>
            <Text type="tertiary" size="small" ellipsis={{ showTooltip: true }} style={{ maxWidth: 400 }}>
              {record.summary as string}
            </Text>
          </div>
        </div>
      ),
    },
    {
      title: '分类', dataIndex: 'category', width: 100,
      render: (val: string) => <Tag size="small" color="light-blue">{val}</Tag>,
    },
    {
      title: '状态', dataIndex: 'status', width: 90, align: 'center' as const,
      render: (val: PostStatus) => {
        const cfg = statusConfig[val]
        return <Tag color={cfg.color} size="small">{cfg.label}</Tag>
      },
    },
    {
      title: '浏览', dataIndex: 'viewCount', width: 90, align: 'center' as const,
      sorter: (a: Record<string, number>, b: Record<string, number>) => a.viewCount - b.viewCount,
      render: (val: number) => (
        <Text type="tertiary" size="small" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4 }}>
          <IconEyeOpened size="small" /> {val.toLocaleString()}
        </Text>
      ),
    },
    {
      title: '评论', dataIndex: 'commentCount', width: 80, align: 'center' as const,
      sorter: (a: Record<string, number>, b: Record<string, number>) => a.commentCount - b.commentCount,
      render: (val: number) => (
        <Text type="tertiary" size="small" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4 }}>
          <IconComment size="small" /> {val}
        </Text>
      ),
    },
    {
      title: '更新日期', dataIndex: 'updatedAt', width: 120,
      sorter: (a: Record<string, string>, b: Record<string, string>) => new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime(),
      render: (val: string) => <Text type="tertiary" size="small">{formatDate(val)}</Text>,
    },
    {
      title: '操作', width: 100, align: 'center' as const,
      render: (_: unknown, record: Record<string, unknown>) => (
        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
          <Tooltip content="编辑" position="top">
            <Button icon={<IconEdit />} theme="borderless" size="small" onClick={() => navigate(`/posts/${record.id}/edit`)} />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            content={`确定要删除「${record.title}」吗？此操作不可撤销。`}
            onConfirm={() => { /* TODO: 调用删除 mutation */ }}
            position="topRight"
          >
            <Tooltip content="删除" position="top">
              <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" />
            </Tooltip>
          </Popconfirm>
        </div>
      ),
    },
  ]

  const rowSelection = {
    selectedRowKeys: selectedKeys,
    onChange: (_: unknown, selectedRows: Record<string, unknown>[]) => {
      setSelectedKeys(selectedRows?.map((r) => r.id as string) || [])
    },
  }

  return (
    <div>
      {/* 标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title heading={4} style={{ marginBottom: 4 }}>文章管理</Title>
          <Text type="tertiary" size="small">管理你的所有博客文章</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => navigate('/posts/new')}>新建文章</Button>
      </div>

      {/* 状态标签快速筛选 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div style={{ display: 'flex', gap: 6 }}>
          {statusTabs.map((tab) => (
            <Tag
              key={tab.key}
              color={status === tab.key ? 'blue' : undefined}
              type={status === tab.key ? 'light' : 'ghost'}
              size="large"
              style={{ cursor: 'pointer', padding: '4px 14px' }}
              onClick={() => { setStatus(tab.key); setPage(1) }}
            >
              {tab.label}
              {tab.key !== 'all' && data && (
                <Text type="tertiary" size="small" style={{ marginLeft: 4 }}>
                  {tab.key === 'published' ? data.list.filter((p) => p.status === 'published').length : ''}
                  {tab.key === 'draft' ? data.list.filter((p) => p.status === 'draft').length : ''}
                  {tab.key === 'archived' ? data.list.filter((p) => p.status === 'archived').length : ''}
                </Text>
              )}
            </Tag>
          ))}
        </div>

        {/* 批量操作 */}
        {selectedKeys.length > 0 && (
          <Popconfirm title="批量删除" content={`确定要删除选中的 ${selectedKeys.length} 篇文章吗？`}>
            <Button icon={<IconDelete />} type="danger" size="small">
              删除 {selectedKeys.length} 篇
            </Button>
          </Popconfirm>
        )}
      </div>

      {/* 搜索与筛选 */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索文章标题、摘要或标签…"
          value={keyword}
          onChange={(v) => { setKeyword(v); setPage(1) }}
          style={{ flex: 1 }}
          showClear
        />
        <Select
          value={category || 'all'}
          onChange={(v) => { setCategory(v === 'all' ? '' : v as string); setPage(1) }}
          style={{ width: 140 }}
          prefix="分类"
        >
          <Select.Option value="all">全部</Select.Option>
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
          rowSelection={rowSelection}
          onRow={(record) => ({
            style: { cursor: 'pointer' },
            onDoubleClick: () => navigate(`/posts/${(record as Record<string, unknown>).id}/edit`),
          })}
          empty={<Text type="tertiary" style={{ padding: 40, display: 'block', textAlign: 'center' }}>没有找到匹配的文章</Text>}
        />
      </Card>

      {/* 分页 */}
      {data && data.total > 0 && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 16 }}>
          <Text type="tertiary" size="small">共 {data.total} 篇文章</Text>
          <Pagination total={data.total} pageSize={pageSize} currentPage={page} onPageChange={setPage} showSizeChanger />
        </div>
      )}
    </div>
  )
}
