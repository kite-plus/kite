import { useState } from 'react'
import { Card, Button, Input, Tag, Tabs, TabPane, Divider, Typography, Empty, Avatar } from '@douyinfe/semi-ui'
import { IconSearch, IconTick, IconDelete, IconClose, IconComment, IconClock, IconArticle, IconShield } from '@douyinfe/semi-icons'
import { useComments, useCommentStats, useModerateComment } from '@/hooks/use-comments'
import type { CommentStatus } from '@/types/comment'

const { Title, Text, Paragraph } = Typography

/** 状态徽标 */
const statusBadge: Record<CommentStatus, { text: string; color: string }> = {
  approved: { text: '已通过', color: 'blue' },
  pending: { text: '待审核', color: 'orange' },
  spam: { text: '垃圾', color: 'red' },
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `${minutes} 分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时前`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days} 天前`
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

/**
 * 评论管理页面 — Semi Tabs / Card / Tag / Button / Input
 */
export function CommentsPage() {
  const [statusFilter, setStatusFilter] = useState<CommentStatus | 'all'>('all')
  const [keyword, setKeyword] = useState('')
  const [expandedId, setExpandedId] = useState<string | null>(null)

  const { data: comments, isLoading } = useComments({ status: statusFilter, keyword })
  const { data: stats } = useCommentStats()
  const moderateMutation = useModerateComment()

  function handleModerate(id: string, action: 'approve' | 'spam' | 'delete') {
    if (action === 'delete' && !window.confirm('确定删除此评论？此操作不可撤销。')) return
    moderateMutation.mutate({ id, action })
  }

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title heading={4}>评论管理</Title>
        <Text type="tertiary" size="small">审核和管理读者的评论</Text>
      </div>

      {/* 统计卡片 */}
      {stats && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 24 }}>
          {[
            { label: '全部评论', value: stats.total },
            { label: '待审核', value: stats.pending },
            { label: '已通过', value: stats.approved },
            { label: '垃圾评论', value: stats.spam },
          ].map((card) => (
            <Card key={card.label}>
              <Text type="tertiary" size="small">{card.label}</Text>
              <Title heading={3} style={{ margin: '4px 0 0' }}>{card.value}</Title>
            </Card>
          ))}
        </div>
      )}

      {/* 筛选栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Tabs type="button" activeKey={statusFilter} onChange={(key) => setStatusFilter(key as CommentStatus | 'all')}>
          <TabPane tab={<><IconComment size="small" /> 全部 {stats?.total ?? 0}</>} itemKey="all" />
          <TabPane tab={<><IconClock size="small" /> 待审核 {stats?.pending ?? 0}</>} itemKey="pending" />
          <TabPane tab={<><IconTick size="small" /> 已通过 {stats?.approved ?? 0}</>} itemKey="approved" />
          <TabPane tab={<><IconClose size="small" /> 垃圾 {stats?.spam ?? 0}</>} itemKey="spam" />
        </Tabs>
        <Input prefix={<IconSearch />} placeholder="搜索评论内容、作者…" value={keyword} onChange={(v) => setKeyword(v)} style={{ maxWidth: 280 }} showClear />
      </div>

      {isLoading && <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>}

      {!isLoading && comments?.length === 0 && (
        <Card><Empty image={<IconComment style={{ fontSize: 48 }} />} description="暂无评论" /></Card>
      )}

      {/* 评论列表 */}
      {comments && comments.length > 0 && (
        <Card bodyStyle={{ padding: 0 }}>
          {comments.map((comment, index) => {
            const badge = statusBadge[comment.status]
            const isExpanded = expandedId === comment.id
            return (
              <div key={comment.id}>
                {index > 0 && <Divider margin={0} />}
                <div style={{ display: 'flex', gap: 12, padding: '16px 20px', background: comment.status === 'spam' ? 'rgba(255,0,0,0.02)' : undefined }}>
                  <Avatar size="small" alt={comment.author}>{comment.author.charAt(0).toUpperCase()}</Avatar>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <Text strong>{comment.author}</Text>
                      <Tag color={badge.color} size="small">{badge.text}</Tag>
                      <Text type="tertiary" size="small" style={{ marginLeft: 'auto' }}>
                        <IconClock size="extra-small" /> {timeAgo(comment.createdAt)}
                      </Text>
                    </div>
                    <Text type="tertiary" size="small" style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 4 }}>
                      <IconArticle size="extra-small" /> {comment.postTitle}
                    </Text>
                    <Paragraph
                      ellipsis={!isExpanded ? { rows: 2, expandable: true, onExpand: () => setExpandedId(comment.id) } : undefined}
                      style={{ marginTop: 8, fontSize: 14 }}
                    >
                      {comment.content}
                    </Paragraph>
                    {isExpanded && (
                      <>
                        <Button theme="borderless" size="small" onClick={() => setExpandedId(null)}>收起</Button>
                        <Divider margin={8} />
                        <Text type="tertiary" size="small">
                          邮箱：{comment.email} · IP：{comment.ip} · UA：{comment.userAgent}
                        </Text>
                      </>
                    )}
                  </div>
                  <div style={{ display: 'flex', gap: 4, alignItems: 'flex-start' }}>
                    {comment.status !== 'approved' && (
                      <Button icon={<IconTick />} theme="light" size="small" disabled={moderateMutation.isPending} onClick={() => handleModerate(comment.id, 'approve')}>通过</Button>
                    )}
                    {comment.status !== 'spam' && (
                      <Button icon={<IconShield />} theme="light" size="small" disabled={moderateMutation.isPending} onClick={() => handleModerate(comment.id, 'spam')}>垃圾</Button>
                    )}
                    <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" disabled={moderateMutation.isPending} onClick={() => handleModerate(comment.id, 'delete')} />
                  </div>
                </div>
              </div>
            )
          })}
        </Card>
      )}
    </div>
  )
}
