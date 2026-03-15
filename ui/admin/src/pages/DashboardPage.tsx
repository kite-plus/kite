import { useNavigate } from 'react-router'
import {
  FileText,
  FolderTree,
  Eye,
  MessageSquare,
  TrendingUp,
  Clock,
  PenSquare,
  ArrowUpRight,
  Activity,
  Server,
  Database,
  Cpu,
} from 'lucide-react'
import { cn } from '@/lib/cn'

/* ============================================
   Mock 仪表盘数据
   ============================================ */

/** 统计卡片数据 */
const statsCards = [
  { label: '文章总数', value: '34', change: '+3', icon: FileText, trend: 'up' as const },
  { label: '分类数量', value: '6', change: '0', icon: FolderTree, trend: 'neutral' as const },
  { label: '本月访问', value: '12,847', change: '+18%', icon: Eye, trend: 'up' as const },
  { label: '待审评论', value: '7', change: '+2', icon: MessageSquare, trend: 'up' as const },
]

/** 七日访问趋势 */
const weeklyTraffic = [
  { day: '周一', views: 1420, posts: 2 },
  { day: '周二', views: 1680, posts: 0 },
  { day: '周三', views: 2310, posts: 1 },
  { day: '周四', views: 1890, posts: 0 },
  { day: '周五', views: 2540, posts: 3 },
  { day: '周六', views: 1120, posts: 0 },
  { day: '周日', views: 980, posts: 1 },
]

/** 最近发布 */
const recentPosts = [
  { title: 'AI 驱动的博客引擎：DeepSeek 自动摘要集成方案', status: 'draft' as const, date: '2026-03-14', views: 0 },
  { title: 'Tailwind CSS v4：从零到一的设计系统构建', status: 'draft' as const, date: '2026-03-10', views: 0 },
  { title: 'Docker Compose 生产环境最佳实践', status: 'published' as const, date: '2026-03-02', views: 2104 },
  { title: '从 Webpack 到 Vite：大型项目迁移实录', status: 'published' as const, date: '2026-02-26', views: 1567 },
  { title: 'React 19 新特性全解析：Server Components 与 Actions', status: 'published' as const, date: '2026-02-21', views: 3456 },
]

/** 近期动态 */
const activityLog = [
  { action: '发布了文章', target: 'Docker Compose 生产环境最佳实践', time: '3 天前', type: 'publish' as const },
  { action: '创建了草稿', target: 'AI 驱动的博客引擎：DeepSeek 自动摘要集成方案', time: '1 天前', type: 'draft' as const },
  { action: '新增分类', target: '开源项目', time: '2 周前', type: 'category' as const },
  { action: '更新了设置', target: '站点描述与 SEO 关键词', time: '2 周前', type: 'setting' as const },
  { action: '修改了文章', target: 'GORM 高级用法：自定义类型、Hook 与性能优化', time: '3 周前', type: 'edit' as const },
  { action: '回复了评论', target: 'React 19 新特性全解析 下 #42', time: '3 周前', type: 'comment' as const },
]

/** 系统信息 */
const systemInfo = [
  { label: '引擎版本', value: 'Kite v0.1.0', icon: Cpu },
  { label: '运行环境', value: 'Go 1.23 / Linux', icon: Server },
  { label: '数据库', value: 'SQLite 3.45', icon: Database },
  { label: '运行时长', value: '14 天 6 小时', icon: Activity },
]

/* ============================================
   组件
   ============================================ */

const statusLabel: Record<string, { text: string; cls: string }> = {
  published: { text: '已发布', cls: 'border-emerald-300 bg-emerald-50 text-emerald-700' },
  draft: { text: '草稿', cls: 'border-amber-300 bg-amber-50 text-amber-700' },
}

const activityIcon: Record<string, string> = {
  publish: '📤',
  draft: '📝',
  category: '📁',
  setting: '⚙️',
  edit: '✏️',
  comment: '💬',
}

/**
 * 仪表盘页面
 * 包含统计卡片、七日趋势、最近文章、动态日志、快捷操作和系统信息
 */
export function DashboardPage() {
  const navigate = useNavigate()
  const maxViews = Math.max(...weeklyTraffic.map((d) => d.views))

  return (
    <div>
      {/* 页面标题 */}
      <div className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
          仪表盘
        </h1>
        <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
          欢迎回到 Kite 后台管理 — 以下是你的站点概览
        </p>
      </div>

      {/* 统计卡片 */}
      <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {statsCards.map((card) => {
          const Icon = card.icon
          return (
            <div
              key={card.label}
              className="border border-[var(--kite-border)] bg-[var(--kite-bg)] p-5 transition-colors duration-100 hover:border-[var(--kite-border-hover)]"
            >
              <div className="flex items-center gap-4">
                <div className="flex h-11 w-11 flex-shrink-0 items-center justify-center border border-[var(--kite-border)] bg-[var(--kite-bg-hover)]">
                  <Icon className="h-5 w-5 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                </div>
                <div>
                  <p className="text-xs text-[var(--kite-text-muted)]">
                    {card.label}
                  </p>
                  <p className="mt-0.5 text-2xl font-bold text-[var(--kite-text-heading)]">
                    {card.value}
                  </p>
                </div>
              </div>
              {card.change !== '0' && (
                <div className="mt-3 flex items-center gap-1 border-t border-[var(--kite-border)] pt-3">
                  <TrendingUp className="h-3 w-3 text-emerald-600" strokeWidth={1.5} />
                  <span className="text-xs text-emerald-600">{card.change}</span>
                  <span className="text-xs text-[var(--kite-text-muted)]">较上月</span>
                </div>
              )}
            </div>
          )
        })}
      </div>

      {/* 中间区域：七日趋势 + 最近文章 */}
      <div className="mb-6 grid grid-cols-1 gap-4 lg:grid-cols-5">
        {/* 七日访问趋势 - 折线图 */}
        <div className="flex flex-col border border-[var(--kite-border)] bg-[var(--kite-bg)] p-5 lg:col-span-3">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">七日访问趋势</h2>
            <span className="text-xs text-[var(--kite-text-muted)]">最近 7 天</span>
          </div>
          {(() => {
            const chartW = 560
            const chartH = 140
            const padX = 40
            const padY = 10
            const padBottom = 18
            const innerW = chartW - padX * 2
            const innerH = chartH - padY - padBottom
            const minV = 0
            const maxV = maxViews * 1.05

            const points = weeklyTraffic.map((d, i) => ({
              x: padX + (i / (weeklyTraffic.length - 1)) * innerW,
              y: padY + innerH - ((d.views - minV) / (maxV - minV)) * innerH,
              views: d.views,
              day: d.day,
            }))

            const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ')
            const areaPath = `${linePath} L${points[points.length - 1].x},${padY + innerH} L${points[0].x},${padY + innerH} Z`

            // Y 轴刻度
            const yTicks = [0, Math.round(maxV * 0.33), Math.round(maxV * 0.66), Math.round(maxV)]

            return (
              <svg viewBox={`0 0 ${chartW} ${chartH}`} className="w-full flex-1" preserveAspectRatio="xMidYEnd meet">
                {/* 水平网格线 */}
                {yTicks.map((tick) => {
                  const y = padY + innerH - ((tick - minV) / (maxV - minV)) * innerH
                  return (
                    <g key={tick}>
                      <line x1={padX} y1={y} x2={chartW - padX} y2={y} stroke="var(--kite-border)" strokeWidth={0.5} />
                      <text x={padX - 6} y={y + 3} textAnchor="end" className="fill-[var(--kite-text-muted)]" fontSize={9}>
                        {tick >= 1000 ? `${(tick / 1000).toFixed(1)}k` : tick}
                      </text>
                    </g>
                  )
                })}

                {/* 面积填充 */}
                <path d={areaPath} fill="var(--kite-accent)" opacity={0.06} />

                {/* 折线 */}
                <path d={linePath} fill="none" stroke="var(--kite-accent)" strokeWidth={2} />

                {/* 数据点 + X 轴标签 */}
                {points.map((p, i) => (
                  <g key={i} className="group">
                    {/* X 轴标签 */}
                    <text x={p.x} y={padY + innerH + 14} textAnchor="middle" className="fill-[var(--kite-text-muted)]" fontSize={10}>
                      {p.day}
                    </text>
                    {/* 圆点 */}
                    <circle cx={p.x} cy={p.y} r={3} fill="var(--kite-bg)" stroke="var(--kite-accent)" strokeWidth={1.5} />
                    {/* hover 大圆点 */}
                    <circle cx={p.x} cy={p.y} r={5} fill="var(--kite-accent)" opacity={0} className="transition-opacity duration-100 hover:opacity-100" />
                    {/* hover 数值标签 */}
                    <rect x={p.x - 22} y={p.y - 24} width={44} height={18} fill="var(--kite-accent)" opacity={0} className="transition-opacity duration-100 hover:opacity-100 pointer-events-none" />
                    {/* 透明大 hitbox */}
                    <rect x={p.x - 25} y={padY} width={50} height={innerH} fill="transparent" className="peer" />
                    {/* 提示框 */}
                    <g className="opacity-0 transition-opacity duration-100 peer-hover:opacity-100" style={{ pointerEvents: 'none' }}>
                      <rect x={p.x - 24} y={p.y - 26} width={48} height={18} fill="var(--kite-accent)" />
                      <text x={p.x} y={p.y - 14} textAnchor="middle" fill="white" fontSize={10} fontWeight={600}>
                        {p.views.toLocaleString()}
                      </text>
                      {/* 竖虚线 */}
                      <line x1={p.x} y1={p.y + 4} x2={p.x} y2={padY + innerH} stroke="var(--kite-accent)" strokeWidth={0.5} strokeDasharray="3,3" />
                    </g>
                  </g>
                ))}
              </svg>
            )
          })()}
        </div>

        {/* 最近文章 */}
        <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)] lg:col-span-2">
          <div className="flex items-center justify-between border-b border-[var(--kite-border)] px-5 py-3">
            <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">最近文章</h2>
            <button
              onClick={() => navigate('/posts')}
              className="flex items-center gap-1 text-xs text-[var(--kite-text-muted)] transition-colors duration-100 hover:text-[var(--kite-text-heading)] cursor-pointer"
            >
              查看全部
              <ArrowUpRight className="h-3 w-3" strokeWidth={1.5} />
            </button>
          </div>
          <ul>
            {recentPosts.map((post, i) => {
              const st = statusLabel[post.status]
              return (
                <li
                  key={i}
                  className={cn(
                    'flex items-center justify-between px-5 py-3 transition-colors duration-100 hover:bg-[var(--kite-bg-hover)]',
                    i < recentPosts.length - 1 && 'border-b border-[var(--kite-border)]'
                  )}
                >
                  <div className="min-w-0 flex-1 pr-3">
                    <p className="truncate text-sm text-[var(--kite-text-heading)]">{post.title}</p>
                    <p className="mt-0.5 text-xs text-[var(--kite-text-muted)]">{post.date}</p>
                  </div>
                  <span className={cn('flex-shrink-0 border px-2 py-0.5 text-xs font-medium', st.cls)}>
                    {st.text}
                  </span>
                </li>
              )
            })}
          </ul>
        </div>
      </div>

      {/* 底部区域：动态日志 + 快捷操作 + 系统信息 */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {/* 近期动态 */}
        <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)] lg:col-span-2">
          <div className="border-b border-[var(--kite-border)] px-5 py-3">
            <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">近期动态</h2>
          </div>
          <ul>
            {activityLog.map((item, i) => (
              <li
                key={i}
                className={cn(
                  'flex items-start gap-3 px-5 py-3',
                  i < activityLog.length - 1 && 'border-b border-[var(--kite-border)]'
                )}
              >
                <span className="mt-0.5 flex h-6 w-6 flex-shrink-0 items-center justify-center text-sm">
                  {activityIcon[item.type]}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-[var(--kite-text)]">
                    <span className="font-medium text-[var(--kite-text-heading)]">{item.action}</span>
                    {' '}
                    <span className="text-[var(--kite-text-muted)]">「{item.target}」</span>
                  </p>
                </div>
                <div className="flex flex-shrink-0 items-center gap-1 text-xs text-[var(--kite-text-muted)]">
                  <Clock className="h-3 w-3" strokeWidth={1.5} />
                  {item.time}
                </div>
              </li>
            ))}
          </ul>
        </div>

        {/* 右侧：快捷操作 + 系统信息 */}
        <div className="space-y-4">
          {/* 快捷操作 */}
          <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)] p-5">
            <h2 className="mb-3 text-sm font-semibold text-[var(--kite-text-heading)]">快捷操作</h2>
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => navigate('/posts')}
                className="flex items-center gap-2 border border-[var(--kite-border)] px-3 py-2.5 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:bg-[var(--kite-bg-hover)] cursor-pointer"
              >
                <PenSquare className="h-4 w-4 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                写文章
              </button>
              <button
                onClick={() => navigate('/categories')}
                className="flex items-center gap-2 border border-[var(--kite-border)] px-3 py-2.5 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:bg-[var(--kite-bg-hover)] cursor-pointer"
              >
                <FolderTree className="h-4 w-4 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                管理分类
              </button>
              <button
                onClick={() => navigate('/settings')}
                className="flex items-center gap-2 border border-[var(--kite-border)] px-3 py-2.5 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:bg-[var(--kite-bg-hover)] cursor-pointer"
              >
                <Eye className="h-4 w-4 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                预览站点
              </button>
              <button
                onClick={() => navigate('/settings')}
                className="flex items-center gap-2 border border-[var(--kite-border)] px-3 py-2.5 text-sm text-[var(--kite-text)] transition-colors duration-100 hover:border-[var(--kite-border-hover)] hover:bg-[var(--kite-bg-hover)] cursor-pointer"
              >
                <Activity className="h-4 w-4 text-[var(--kite-text-muted)]" strokeWidth={1.5} />
                系统设置
              </button>
            </div>
          </div>

          {/* 系统信息 */}
          <div className="border border-[var(--kite-border)] bg-[var(--kite-bg)]">
            <div className="border-b border-[var(--kite-border)] px-5 py-3">
              <h2 className="text-sm font-semibold text-[var(--kite-text-heading)]">系统信息</h2>
            </div>
            <ul>
              {systemInfo.map((item, i) => {
                const Icon = item.icon
                return (
                  <li
                    key={i}
                    className={cn(
                      'flex items-center justify-between px-5 py-2.5',
                      i < systemInfo.length - 1 && 'border-b border-[var(--kite-border)]'
                    )}
                  >
                    <div className="flex items-center gap-2 text-sm text-[var(--kite-text-muted)]">
                      <Icon className="h-3.5 w-3.5" strokeWidth={1.5} />
                      {item.label}
                    </div>
                    <span className="text-sm text-[var(--kite-text-heading)]">{item.value}</span>
                  </li>
                )
              })}
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}
