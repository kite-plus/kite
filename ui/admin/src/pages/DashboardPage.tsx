import { useNavigate } from 'react-router'
import { Card, Button, Tag, Divider, Empty, Typography } from '@douyinfe/semi-ui'
import { IconEdit, IconEyeOpened, IconSetting, IconGridView, IconArticle, IconComment, IconTick } from '@douyinfe/semi-icons'

const { Title, Text, Paragraph } = Typography

/* Mock 仪表盘数据 */
const statsCards = [
  { label: '文章', value: '34', change: '+3', changeLabel: '较上月', icon: <IconArticle style={{ fontSize: 20 }} /> },
  { label: '分类', value: '6', change: '', changeLabel: '', icon: <IconGridView style={{ fontSize: 20 }} /> },
  { label: '本月访问', value: '12,847', change: '+18%', changeLabel: '较上月', icon: <IconEyeOpened style={{ fontSize: 20 }} /> },
  { label: '待审评论', value: '7', change: '+2', changeLabel: '较上月', icon: <IconComment style={{ fontSize: 20 }} /> },
]

const weeklyTraffic = [
  { day: '周一', views: 1420 },
  { day: '周二', views: 1680 },
  { day: '周三', views: 2310 },
  { day: '周四', views: 1890 },
  { day: '周五', views: 2540 },
  { day: '周六', views: 1120 },
  { day: '周日', views: 980 },
]

const recentPosts = [
  { title: 'AI 驱动的博客引擎：DeepSeek 自动摘要集成方案', status: 'draft' as const, date: '2026-03-14' },
  { title: 'Tailwind CSS v4：从零到一的设计系统构建', status: 'draft' as const, date: '2026-03-10' },
  { title: 'Docker Compose 生产环境最佳实践', status: 'published' as const, date: '2026-03-02' },
  { title: '从 Webpack 到 Vite：大型项目迁移实录', status: 'published' as const, date: '2026-02-26' },
  { title: 'React 19 新特性全解析：Server Components 与 Actions', status: 'published' as const, date: '2026-02-21' },
]

const activityLog = [
  { action: '发布了文章', target: 'Docker Compose 生产环境最佳实践', time: '3 天前', icon: '📤' },
  { action: '创建了草稿', target: 'AI 驱动的博客引擎：DeepSeek 自动摘要集成方案', time: '1 天前', icon: '📝' },
  { action: '新增分类', target: '开源项目', time: '2 周前', icon: '📁' },
  { action: '更新了设置', target: '站点描述与 SEO 关键词', time: '2 周前', icon: '⚙️' },
  { action: '修改了文章', target: 'GORM 高级用法：自定义类型、Hook 与性能优化', time: '3 周前', icon: '✏️' },
  { action: '回复了评论', target: 'React 19 新特性全解析 下 #42', time: '3 周前', icon: '💬' },
]

/**
 * 仪表盘页面 — Semi Design
 */
export function DashboardPage() {
  const navigate = useNavigate()
  const maxViews = Math.max(...weeklyTraffic.map((d) => d.views))

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
                <Title heading={3} style={{ margin: 0 }}>{card.value}</Title>
              </div>
            </div>
            {card.change && (
              <>
                <Divider margin={12} />
                <Text type="success" size="small" style={{ fontWeight: 500 }}>
                  ↗ {card.change} <Text type="tertiary" size="small">{card.changeLabel}</Text>
                </Text>
              </>
            )}
          </Card>
        ))}
      </div>

      {/* 主内容：左 3 / 右 2 */}
      <div style={{ display: 'grid', gridTemplateColumns: '3fr 2fr', gap: 16 }}>
        {/* 左列 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* 七日访问趋势 */}
          <Card title="七日访问趋势" headerExtraContent={<Text type="tertiary" size="small">最近 7 天</Text>}>
            {(() => {
              const chartW = 560, chartH = 140, padX = 40, padY = 10, padBottom = 18
              const innerW = chartW - padX * 2, innerH = chartH - padY - padBottom
              const maxV = maxViews * 1.05
              const points = weeklyTraffic.map((d, i) => ({
                x: padX + (i / (weeklyTraffic.length - 1)) * innerW,
                y: padY + innerH - (d.views / maxV) * innerH,
                views: d.views, day: d.day,
              }))
              const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ')
              const areaPath = `${linePath} L${points[points.length - 1].x},${padY + innerH} L${points[0].x},${padY + innerH} Z`
              const yTicks = [0, Math.round(maxV * 0.33), Math.round(maxV * 0.66), Math.round(maxV)]
              return (
                <svg viewBox={`0 0 ${chartW} ${chartH}`} style={{ width: '100%' }} preserveAspectRatio="xMidYEnd meet">
                  {yTicks.map((tick) => {
                    const y = padY + innerH - (tick / maxV) * innerH
                    return (
                      <g key={tick}>
                        <line x1={padX} y1={y} x2={chartW - padX} y2={y} stroke="var(--semi-color-border)" strokeWidth={0.5} />
                        <text x={padX - 6} y={y + 3} textAnchor="end" fill="var(--semi-color-text-2)" fontSize={9}>
                          {tick >= 1000 ? `${(tick / 1000).toFixed(1)}k` : tick}
                        </text>
                      </g>
                    )
                  })}
                  <path d={areaPath} fill="rgba(0,100,250,0.06)" />
                  <path d={linePath} fill="none" stroke="var(--semi-color-primary)" strokeWidth={2} />
                  {points.map((p, i) => (
                    <g key={i}>
                      <text x={p.x} y={padY + innerH + 14} textAnchor="middle" fill="var(--semi-color-text-2)" fontSize={10}>{p.day}</text>
                      <circle cx={p.x} cy={p.y} r={3} fill="#fff" stroke="var(--semi-color-primary)" strokeWidth={1.5} />
                    </g>
                  ))}
                </svg>
              )
            })()}
          </Card>

          {/* 最近文章 */}
          <Card title="最近文章" headerExtraContent={<Button theme="borderless" size="small" onClick={() => navigate('/posts')}>查看全部 →</Button>}>
            {recentPosts.map((post, i) => (
              <div key={i}>
                {i > 0 && <Divider margin={0} />}
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '10px 0' }}>
                  <div style={{ minWidth: 0, flex: 1, paddingRight: 12 }}>
                    <Paragraph ellipsis style={{ fontSize: 14, fontWeight: 500, margin: 0 }}>{post.title}</Paragraph>
                    <Text type="tertiary" size="small">{post.date}</Text>
                  </div>
                  <Tag color={post.status === 'published' ? 'blue' : 'grey'}>
                    {post.status === 'published' ? '已发布' : '草稿'}
                  </Tag>
                </div>
              </div>
            ))}
          </Card>

          {/* 近期动态 */}
          <Card title="近期动态">
            {activityLog.map((item, i) => (
              <div key={i}>
                {i > 0 && <Divider margin={0} />}
                <div style={{ display: 'flex', alignItems: 'flex-start', gap: 10, padding: '10px 0' }}>
                  <span style={{ fontSize: 14 }}>{item.icon}</span>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <Text style={{ fontSize: 14 }}>
                      <Text strong>{item.action}</Text>
                      {' '}
                      <Text type="tertiary">「{item.target}」</Text>
                    </Text>
                  </div>
                  <Text type="tertiary" size="small">{item.time}</Text>
                </div>
              </div>
            ))}
          </Card>
        </div>

        {/* 右列 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {/* 快捷操作 */}
          <Card title="快捷操作">
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
              <Button icon={<IconEdit />} theme="light" block onClick={() => navigate('/posts/new')}>写文章</Button>
              <Button icon={<IconGridView />} theme="light" block onClick={() => navigate('/categories')}>管理分类</Button>
              <Button icon={<IconEyeOpened />} theme="light" block onClick={() => navigate('/')}>预览站点</Button>
              <Button icon={<IconSetting />} theme="light" block onClick={() => navigate('/settings')}>系统设置</Button>
            </div>
          </Card>

          {/* 通知 — 空态 */}
          <Card title="通知">
            <Empty
              image={<IconTick style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
              description="当前没有未读的消息"
            />
          </Card>
        </div>
      </div>
    </div>
  )
}
