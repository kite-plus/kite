import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Table, Button, Input, Tag, Card, Typography, Tooltip, Popconfirm, Switch } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconEdit, IconDelete, IconExternalOpen } from '@douyinfe/semi-icons'
import { usePageList, useDeletePage } from '@/hooks/use-pages'
import type { Page, PageStatus } from '@/types/page'
import type { ColumnProps } from '@douyinfe/semi-ui/lib/es/table'

const { Title, Text } = Typography

/** 状态标签配置 */
const statusConfig: Record<PageStatus, { label: string; color: 'blue' | 'grey' }> = {
  published: { label: '已发布', color: 'blue' },
  draft: { label: '草稿', color: 'grey' },
}

/** 格式化日期 */
function formatDate(dateStr: string | null) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

/**
 * 独立页面管理列表
 */
export function PagesPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')

  const { data: pages, isLoading } = usePageList(keyword)
  const deleteMutation = useDeletePage()

  const columns: ColumnProps[] = [
    {
      title: '页面标题', dataIndex: 'title',
      render: (_: unknown, record: Page) => (
        <div style={{ cursor: 'pointer' }} onClick={() => navigate(`/pages/${record.id}/edit`)}>
          <Text strong style={{ fontSize: 14 }}>{record.title}</Text>
          <div style={{ marginTop: 2, display: 'flex', alignItems: 'center', gap: 6 }}>
            <Text type="tertiary" size="small">/{record.slug}</Text>
            {record.showInNav && (
              <Tag size="small" color="green" type="light">导航栏</Tag>
            )}
          </div>
        </div>
      ),
    },
    {
      title: '状态', dataIndex: 'status', width: 90, align: 'center' as const,
      render: (val: PageStatus) => {
        const cfg = statusConfig[val]
        return <Tag color={cfg.color} size="small">{cfg.label}</Tag>
      },
    },
    {
      title: '排序', dataIndex: 'sortOrder', width: 80, align: 'center' as const,
      sorter: (a: Record<string, number>, b: Record<string, number>) => a.sortOrder - b.sortOrder,
      render: (val: number) => <Text type="tertiary" size="small">{val}</Text>,
    },
    {
      title: '导航栏', dataIndex: 'showInNav', width: 90, align: 'center' as const,
      render: (val: boolean) => <Switch checked={val} size="small" disabled />,
    },
    {
      title: '更新日期', dataIndex: 'updatedAt', width: 120,
      sorter: (a: Record<string, string>, b: Record<string, string>) => new Date(a.updatedAt).getTime() - new Date(b.updatedAt).getTime(),
      render: (val: string) => <Text type="tertiary" size="small">{formatDate(val)}</Text>,
    },
    {
      title: '操作', width: 120, align: 'center' as const,
      render: (_: unknown, record: Page) => (
        <div style={{ display: 'flex', gap: 4, justifyContent: 'center' }}>
          {record.status === 'published' && (
            <Tooltip content="前台预览" position="top">
              <Button
                icon={<IconExternalOpen />}
                theme="borderless"
                size="small"
                onClick={() => window.open(`/${record.slug}`, '_blank')}
              />
            </Tooltip>
          )}
          <Tooltip content="编辑" position="top">
            <Button icon={<IconEdit />} theme="borderless" size="small" onClick={() => navigate(`/pages/${record.id}/edit`)} />
          </Tooltip>
          <Popconfirm
            title="确认删除"
            content={`确定要删除页面「${record.title}」吗？此操作不可撤销。`}
            onConfirm={() => deleteMutation.mutate(record.id)}
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

  return (
    <div>
      {/* 标题区 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title heading={4} style={{ marginBottom: 4 }}>页面管理</Title>
          <Text type="tertiary" size="small">管理博客的独立页面，如关于我、归档、留言板等</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => navigate('/pages/new')}>新建页面</Button>
      </div>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 12, marginBottom: 20 }}>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">页面总数</Text>
          <Title heading={3} style={{ margin: '4px 0 0' }}>{pages?.length ?? 0}</Title>
        </Card>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">已发布</Text>
          <Title heading={3} style={{ margin: '4px 0 0' }}>{pages?.filter((p) => p.status === 'published').length ?? 0}</Title>
        </Card>
        <Card bodyStyle={{ padding: '16px 20px' }}>
          <Text type="tertiary" size="small">导航栏显示</Text>
          <Title heading={3} style={{ margin: '4px 0 0' }}>{pages?.filter((p) => p.showInNav).length ?? 0}</Title>
        </Card>
      </div>

      {/* 搜索 */}
      <div style={{ marginBottom: 16 }}>
        <Input
          prefix={<IconSearch />}
          placeholder="搜索页面标题或 Slug…"
          value={keyword}
          onChange={setKeyword}
          style={{ maxWidth: 400 }}
          showClear
        />
      </div>

      {/* 页面列表 */}
      <Card bodyStyle={{ padding: 0 }}>
        <Table
          columns={columns}
          dataSource={pages || []}
          loading={isLoading}
          pagination={false}
          rowKey="id"
          onRow={(record) => ({
            style: { cursor: 'pointer' },
            onDoubleClick: () => navigate(`/pages/${(record as Page).id}/edit`),
          })}
          empty={<Text type="tertiary" style={{ padding: 40, display: 'block', textAlign: 'center' }}>暂无独立页面</Text>}
        />
      </Card>
    </div>
  )
}
