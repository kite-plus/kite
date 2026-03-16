import { useNavigate } from 'react-router'
import { Card, Button, Tag, Divider, Empty, Typography, Spin } from '@douyinfe/semi-ui'
import { IconEdit, IconEyeOpened, IconSetting, IconGridView, IconArticle, IconComment, IconTick } from '@douyinfe/semi-icons'
import { useDashboardStats, useRecentPosts } from '@/hooks/use-dashboard'

const { Title, Text, Paragraph } = Typography

/** 格式化日期 */
function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

/**
 * 仪表盘页面 — Semi Design（真实 API 数据）
 */
export function DashboardPage() {
  const navigate = useNavigate()
  const { data: stats, isLoading: statsLoading } = useDashboardStats()
  const { data: recentPosts, isLoading: postsLoading } = useRecentPosts(5)

  /** 统计卡片配置 */
  const statsCards = [
    { label: '文章', value: stats?.postCount ?? '—', icon: <IconArticle style={{ fontSize: 20 }} /> },
    { label: '分类', value: stats?.categoryCount ?? '—', icon: <IconGridView style={{ fontSize: 20 }} /> },
    { label: '标签', value: stats?.tagCount ?? '—', icon: <IconEyeOpened style={{ fontSize: 20 }} /> },
    { label: '待审评论', value: stats?.commentPending ?? '—', icon: <IconComment style={{ fontSize: 20 }} /> },
  ]

  return (
    <div>
      {/* 页面标题 */}
      <div style={{ marginBottom: 24 }}>
        <Title heading={4}>仪表盘</Title>
        <Text type="tertiary" style={{ fontSize: 14 }}>
          欢迎回到 Kite 后台管理 — 以下是你的站点概览
        </Text>
      </div>

      {/* 统计卡片 */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 24 }}>
        {statsCards.map((card) => (
          <Card key={card.label}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
              <div style={{ width: 44, height: 44, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--semi-color-fill-0)', borderRadius: 8 }}>
                {card.icon}
              </div>
              <div>
                <Text type="tertiary" size="small">{card.label}</Text>
                {statsLoading ? (
                  <Spin size="small" />
                ) : (
                  <Title heading={3} style={{ margin: 0 }}>{card.value}</Title>
                )}
              </div>
            </div>
          </Card>
        ))}
      </div>

      {/* 第二行：快捷操作 + 最近文章 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 16, marginBottom: 16 }}>
        {/* 快捷操作 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Card title="快捷操作">
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <Button icon={<IconEdit />} theme="light" block onClick={() => navigate('/posts/new')}>写文章</Button>
              <Button icon={<IconGridView />} theme="light" block onClick={() => navigate('/categories')}>管理分类</Button>
              <Button icon={<IconEyeOpened />} theme="light" block onClick={() => window.open('/', '_blank')}>预览站点</Button>
              <Button icon={<IconSetting />} theme="light" block onClick={() => navigate('/settings')}>系统设置</Button>
            </div>
          </Card>
          <Card title="通知">
            <Empty
              image={<IconTick style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
              description="当前没有未读的消息"
            />
          </Card>
        </div>

        {/* 最近文章 */}
        <Card title="最近文章" headerExtraContent={<Button theme="borderless" size="small" onClick={() => navigate('/posts')}>查看全部 →</Button>}>
          {postsLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
          ) : recentPosts && recentPosts.length > 0 ? (
            recentPosts.map((post, i) => (
              <div key={post.id}>
                {i > 0 && <Divider margin={0} />}
                <div
                  style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 0', cursor: 'pointer' }}
                  onClick={() => navigate(`/posts/${post.id}/edit`)}
                >
                  <div style={{ minWidth: 0, flex: 1, paddingRight: 12 }}>
                    <Paragraph ellipsis style={{ fontSize: 14, fontWeight: 500, margin: 0 }}>{post.title}</Paragraph>
                    <Text type="tertiary" size="small">{formatDate(post.createdAt)}</Text>
                  </div>
                  <Tag color={post.status === 'published' ? 'blue' : 'grey'}>
                    {post.status === 'published' ? '已发布' : '草稿'}
                  </Tag>
                </div>
              </div>
            ))
          ) : (
            <Empty description="还没有文章" />
          )}
        </Card>
      </div>
    </div>
  )
}
