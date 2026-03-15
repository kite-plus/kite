/**
 * 仪表盘页面 - 占位组件
 * 用于验证 AdminLayout 布局渲染效果
 */
export function DashboardPage() {
  return (
    <div>
      {/* 页面标题 */}
      <div className="mb-8">
        <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
          仪表盘
        </h1>
        <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
          欢迎回到 Kite 后台管理
        </p>
      </div>

      {/* 统计卡片占位 */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {[
          { label: '文章总数', value: '—' },
          { label: '分类数量', value: '—' },
          { label: '今日访问', value: '—' },
          { label: '评论数', value: '—' },
        ].map((card) => (
          <div
            key={card.label}
            className="border border-[var(--kite-border)] bg-[var(--kite-bg)] p-6 transition-colors duration-100 hover:border-[var(--kite-border-hover)]"
          >
            <p className="text-xs font-medium uppercase tracking-wider text-[var(--kite-text-muted)]">
              {card.label}
            </p>
            <p className="mt-2 text-3xl font-semibold text-[var(--kite-text-heading)]">
              {card.value}
            </p>
          </div>
        ))}
      </div>
    </div>
  )
}
