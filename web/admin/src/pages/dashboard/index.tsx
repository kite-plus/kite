import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useLocation, useNavigate } from 'react-router-dom'
import {
  Activity,
  ArrowRight,
  ArrowUp,
  Cpu,
  Eye,
  Files,
  HardDrive,
  Key,
  Plus,
  ShieldCheck,
  TrendingUp,
  Upload,
  User as UserIcon,
  Users,
} from 'lucide-react'
import {
  adminFileApi,
  adminStatsApi,
  fileApi,
  systemStatusApi,
  statsApi,
  storageApi,
  tokenApi,
  userApi,
} from '@/lib/api'
import { useI18n, type Locale } from '@/i18n'
import { useAuth } from '@/hooks/use-auth'
import { cn, formatSize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { StorageLogo, resolveLogoVendor } from '@/components/storage-logo'
import { UpdateBanner } from '@/components/update-banner'
import {
  CardLinkFooter,
  Donut,
  type DonutSegment,
  FileThumb,
  FileThumbSkeleton,
  Heatmap,
  HeatmapLegend,
  HeroKPI,
  PageHero,
  StackedStorageBar,
  type StorageSegment,
  type ThumbFile,
  TrendCombo,
  type TrendPoint,
} from './components'

/* ────────────────────────────────────────────────────────────
 * Types
 * ──────────────────────────────────────────────────────────── */
interface DashboardStats {
  total_files: number
  total_size: number
  images: number
  videos: number
  audios: number
  others: number
  images_size?: number
  videos_size?: number
  audios_size?: number
  others_size?: number
  users?: number
}

interface DailyPoint {
  day: string
  uploads: number
  accesses: number
  bytes_served: number
}

interface TopUser {
  id: string
  username: string
  nickname?: string
  avatar_url?: string
  storage_used: number
  storage_limit: number
}

interface DashboardStorageBackend {
  id: string
  name: string
  driver: string
  provider?: string
  capacity_limit_bytes: number
  used_bytes: number
  is_default: boolean
  is_active: boolean
}

interface DashboardHeatmap {
  weeks: number
  grid: number[][]
}

/* ────────────────────────────────────────────────────────────
 * Helpers
 * ──────────────────────────────────────────────────────────── */

/**
 * Greeting for the dashboard hero, picked by hour of day.
 *
 * The bilingual table is gone — both languages now live in the i18n
 * catalogue under `dashboard.greeting.*`, so adding a third locale is a
 * one-file change rather than touching this helper.
 */
function buildGreeting(t: (key: string) => string): {
  main: string
  sub: string
} {
  const h = new Date().getHours()
  if (h < 5)
    return {
      main: t('dashboard.greeting.lateNight'),
      sub: t('dashboard.greeting.lateNightSub'),
    }
  if (h < 11)
    return {
      main: t('dashboard.greeting.morning'),
      sub: t('dashboard.greeting.morningSub'),
    }
  if (h < 14)
    return {
      main: t('dashboard.greeting.noon'),
      sub: t('dashboard.greeting.noonSub'),
    }
  if (h < 18)
    return {
      main: t('dashboard.greeting.afternoon'),
      sub: t('dashboard.greeting.afternoonSub'),
    }
  if (h < 22)
    return {
      main: t('dashboard.greeting.evening'),
      sub: t('dashboard.greeting.eveningSub'),
    }
  return {
    main: t('dashboard.greeting.night'),
    sub: t('dashboard.greeting.nightSub'),
  }
}

function emptyHeatmapGrid(): number[][] {
  return Array.from({ length: 7 }, () => Array.from({ length: 24 }, () => 0))
}

function normalizeHeatmapGrid(input: unknown): number[][] {
  if (!Array.isArray(input) || input.length !== 7) return emptyHeatmapGrid()

  const grid = emptyHeatmapGrid()
  for (let d = 0; d < 7; d++) {
    const row = input[d]
    if (!Array.isArray(row) || row.length !== 24) {
      return emptyHeatmapGrid()
    }
    for (let h = 0; h < 24; h++) {
      const n = Number(row[h])
      grid[d][h] = Number.isFinite(n) && n > 0 ? Math.floor(n) : 0
    }
  }
  return grid
}

interface ActivityItem {
  id: string
  actor: string
  actorType: 'you' | 'system'
  actionKey: 'uploaded'
  target: string
  createdAt: string
}

function buildActivityFromRecent(
  t: (key: string) => string,
  recentItems: ThumbFile[],
  actorName: string
): ActivityItem[] {
  const fallbackActor = t('dashboard.activity.fallbackActor')
  return recentItems.slice(0, 6).map((item) => ({
    id: item.id,
    actor: actorName || fallbackActor,
    actorType: 'you',
    actionKey: 'uploaded',
    target: item.original_name,
    createdAt: item.created_at,
  }))
}

function actionLabel(
  key: ActivityItem['actionKey'],
  t: (k: string) => string
): string {
  // Single-key map for now — kept as a switch so a future second action
  // (e.g. `deleted`, `shared`) is a one-line addition rather than a
  // refactor.
  switch (key) {
    case 'uploaded':
      return t('dashboard.activity.uploaded')
    default:
      return key
  }
}

function formatRelativeTime(
  timestamp: string,
  t: (k: string) => string
): string {
  const diffMs = Date.now() - new Date(timestamp).getTime()
  const mins = Math.max(0, Math.floor(diffMs / 60_000))
  if (mins < 60)
    return t('relativeTime.minutesAgo').replace('{n}', String(mins))
  const h = Math.floor(mins / 60)
  if (h < 24) return t('relativeTime.hoursAgo').replace('{n}', String(h))
  const d = Math.floor(h / 24)
  return t('relativeTime.daysAgo').replace('{n}', String(d))
}

/* ────────────────────────────────────────────────────────────
 * DashboardPage
 * ──────────────────────────────────────────────────────────── */
export default function DashboardPage() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const { user } = useAuth()
  const isAdminWorkspace = location.pathname.startsWith('/admin')
  const displayName = user?.nickname?.trim() || user?.username || ''

  /* ── stats ──────────────────────────────────────────────── */
  const { data: stats, isLoading: statsLoading } = useQuery<DashboardStats>({
    queryKey: ['dashboard', 'stats', isAdminWorkspace ? 'admin' : 'user'],
    queryFn: () =>
      (isAdminWorkspace ? adminStatsApi.get() : statsApi.get()).then(
        (r) => r.data.data
      ),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  /* ── daily trend (30d for chart, 7d sum for KPI) ────────── */
  const { data: daily } = useQuery<{ days: DailyPoint[] }>({
    queryKey: ['dashboard', 'daily', isAdminWorkspace ? 'admin' : 'user'],
    queryFn: () =>
      (isAdminWorkspace ? adminStatsApi.daily(30) : statsApi.daily(30)).then(
        (r) => r.data.data
      ),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  const { data: heatmapData } = useQuery<DashboardHeatmap>({
    queryKey: ['dashboard', 'heatmap', isAdminWorkspace ? 'admin' : 'user', 12],
    queryFn: () =>
      (isAdminWorkspace
        ? adminStatsApi.heatmap(12)
        : statsApi.heatmap(12)
      ).then((r) => r.data.data),
    staleTime: 30_000,
    refetchInterval: 60_000,
  })

  /* ── recent uploads (user) / top users (admin) ───────────── */
  const { data: recent, isLoading: recentLoading } = useQuery<{
    items: ThumbFile[]
  }>({
    queryKey: ['dashboard', 'recent', isAdminWorkspace ? 'admin' : 'user'],
    queryFn: () =>
      (isAdminWorkspace
        ? adminFileApi.list({ page: 1, size: 8 })
        : fileApi.list({ page: 1, size: 8, only_self: true })
      ).then((r) => r.data.data),
    staleTime: 30_000,
  })

  const { data: topUsersData } = useQuery<{ items: TopUser[] }>({
    queryKey: ['dashboard', 'topUsers'],
    enabled: isAdminWorkspace,
    queryFn: () => userApi.list({ page: 1, size: 50 }).then((r) => r.data.data),
    staleTime: 60_000,
  })

  /* ── tokens count (user KPI) ────────────────────────────── */
  const { data: tokensData } = useQuery<{ items: Array<{ id: string }> }>({
    queryKey: ['dashboard', 'tokens'],
    enabled: !isAdminWorkspace,
    queryFn: () => tokenApi.list().then((r) => r.data.data),
    staleTime: 60_000,
  })

  /* ── admin storage backends ─────────────────────────────── */
  const { data: backendsData } = useQuery<DashboardStorageBackend[]>({
    queryKey: ['dashboard', 'storageBackends'],
    enabled: isAdminWorkspace,
    queryFn: () => storageApi.list().then((r) => r.data.data),
    staleTime: 60_000,
  })

  /* ── derived values ─────────────────────────────────────── */
  const greeting = useMemo(() => buildGreeting(t), [t])
  const heatmap = useMemo(
    () => normalizeHeatmapGrid(heatmapData?.grid),
    [heatmapData?.grid]
  )
  const activity = useMemo(
    () => buildActivityFromRecent(t, recent?.items ?? [], displayName),
    [displayName, t, recent?.items]
  )

  const days30: TrendPoint[] = useMemo(() => {
    return (daily?.days ?? []).map((d) => ({
      day: d.day,
      uploads: d.uploads,
      accesses: d.accesses,
    }))
  }, [daily?.days])

  const last7 = useMemo(() => days30.slice(-7), [days30])
  const weekUploads = last7.reduce((a, b) => a + b.uploads, 0)
  const weekAccesses = last7.reduce((a, b) => a + b.accesses, 0)
  const todayUploads = days30.length ? days30[days30.length - 1].uploads : 0

  const storageSegs: StorageSegment[] = useMemo(() => {
    const s = stats ?? {
      total_files: 0,
      total_size: 0,
      images: 0,
      videos: 0,
      audios: 0,
      others: 0,
    }
    const hasPerKindSize =
      typeof s.images_size === 'number' &&
      typeof s.videos_size === 'number' &&
      typeof s.audios_size === 'number' &&
      typeof s.others_size === 'number'

    const total = Math.max(1, s.total_files)
    const byKindApprox = (count: number) =>
      total > 0 ? Math.round((count / total) * s.total_size) : 0

    const byKind = (
      kind: 'image' | 'video' | 'audio' | 'other',
      count: number
    ) => {
      if (!hasPerKindSize) return byKindApprox(count)
      if (kind === 'image') return s.images_size ?? 0
      if (kind === 'video') return s.videos_size ?? 0
      if (kind === 'audio') return s.audios_size ?? 0
      return s.others_size ?? 0
    }

    return [
      {
        kind: 'image',
        label: t('dashboard.images'),
        count: s.images,
        bytes: byKind('image', s.images),
        color: 'hsl(var(--chart-3))',
      },
      {
        kind: 'video',
        label: t('dashboard.videos'),
        count: s.videos,
        bytes: byKind('video', s.videos),
        color: 'hsl(var(--chart-2))',
      },
      {
        kind: 'audio',
        label: t('dashboard.audio'),
        count: s.audios,
        bytes: byKind('audio', s.audios),
        color: 'hsl(var(--chart-1))',
      },
      {
        kind: 'other',
        label: t('dashboard.otherFiles'),
        count: s.others,
        bytes: byKind('other', s.others),
        color: 'hsl(var(--chart-4))',
      },
    ]
  }, [stats, t])

  const donutSegs: DonutSegment[] = storageSegs.map((s) => ({
    value: s.count,
    color: s.color,
    label: s.label,
  }))
  const totalBytes = storageSegs.reduce((a, b) => a + b.bytes, 0)
  const totalCount = storageSegs.reduce((a, b) => a + b.count, 0)

  const storageLimit = user?.storage_limit ?? 0
  const storageUsed = stats?.total_size ?? 0
  const isUnlimited = storageLimit < 0
  const hasLimit = storageLimit > 0
  const storagePct = hasLimit
    ? Math.min((storageUsed / storageLimit) * 100, 100)
    : 0

  const topUsers = useMemo(
    () =>
      [...(topUsersData?.items ?? [])]
        .sort((a, b) => (b.storage_used ?? 0) - (a.storage_used ?? 0))
        .slice(0, 6),
    [topUsersData?.items]
  )

  /* ── hero content ───────────────────────────────────────── */
  const heroTitle = isAdminWorkspace
    ? `${greeting.main}${locale === 'zh' ? '，' : ', '}${t('nav.roleAdmin')}`
    : `${greeting.main}${locale === 'zh' ? '，' : ', '}${displayName}`

  const heroSubUser =
    locale === 'zh'
      ? `${greeting.sub} · 今天已同步 ${todayUploads} 个文件`
      : `${greeting.sub} · ${todayUploads} file${
          todayUploads === 1 ? '' : 's'
        } synced today`
  const heroSubAdmin = t('dashboard.hero.adminSub')

  /* ── admin KPI derived counts ───────────────────────────── */
  const activeBackends = (backendsData ?? []).filter((b) => b.is_active).length
  const totalBackends = (backendsData ?? []).length
  const activeUsers = (topUsersData?.items ?? []).filter(
    (u) => (u.storage_used ?? 0) > 0
  ).length

  /* ── render ─────────────────────────────────────────────── */
  const trendHeatmapRow = (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="min-w-0 gap-3 overflow-hidden py-5 shadow-xs">
        <CardHeader className="px-5">
          <CardTitle className="text-sm">
            {isAdminWorkspace
              ? t('dashboard.trend.adminTitle')
              : t('dashboard.trend.title')}
          </CardTitle>
          <CardDescription className="text-xs">
            {isAdminWorkspace
              ? t('dashboard.trend.adminSub')
              : t('dashboard.trend.sub')}
          </CardDescription>
          <CardAction>
            <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
              <span className="flex items-center gap-1">
                <span
                  className="size-2 rounded-sm"
                  style={{ background: 'hsl(var(--chart-3))' }}
                />
                {t('dashboard.uploads')}
              </span>
              <span className="flex items-center gap-1">
                <span
                  className="size-2 rounded-sm"
                  style={{ background: 'hsl(var(--chart-2))' }}
                />
                {t('dashboard.accesses')}
              </span>
            </div>
          </CardAction>
        </CardHeader>
        <CardContent className="min-w-0 px-5">
          <div className="h-44 w-full sm:h-56">
            {!daily ? (
              <Skeleton className="h-full w-full" />
            ) : (
              <TrendCombo
                data={days30}
                height={220}
                uploadsLabel={t('dashboard.uploads')}
                accessesLabel={t('dashboard.accesses')}
              />
            )}
          </div>
        </CardContent>
      </Card>

      <Card className="min-w-0 gap-3 overflow-hidden py-5 shadow-xs">
        <CardHeader className="px-5">
          <CardTitle className="text-sm">
            {isAdminWorkspace
              ? t('dashboard.heatmap.adminTitle')
              : t('dashboard.heatmap.title')}
          </CardTitle>
          <CardDescription className="text-xs">
            {isAdminWorkspace
              ? t('dashboard.heatmap.adminSub')
              : t('dashboard.heatmap.sub')}
          </CardDescription>
        </CardHeader>
        <CardContent className="min-w-0 px-5">
          <Heatmap
            grid={heatmap}
            weekdayLabels={[
              t('dashboard.weekdays.mon'),
              t('dashboard.weekdays.tue'),
              t('dashboard.weekdays.wed'),
              t('dashboard.weekdays.thu'),
              t('dashboard.weekdays.fri'),
              t('dashboard.weekdays.sat'),
              t('dashboard.weekdays.sun'),
            ]}
          />
          <HeatmapLegend
            lowLabel={t('dashboard.heatmap.low')}
            highLabel={t('dashboard.heatmap.high')}
          />
        </CardContent>
      </Card>
    </div>
  )

  return (
    <div className="page-enter flex flex-col gap-5 sm:gap-6">
      {isAdminWorkspace && <UpdateBanner />}
      {/* ═════ HERO ═════════════════════════════════════════ */}
      <PageHero>
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between lg:gap-8">
          <div className="flex min-w-0 items-start gap-3 sm:gap-4">
            {isAdminWorkspace ? (
              <div className="flex size-12 shrink-0 items-center justify-center rounded-full border bg-background/70 backdrop-blur sm:size-[52px]">
                <ShieldCheck className="size-5" />
              </div>
            ) : (
              <Avatar className="size-12 shrink-0 ring-2 ring-background sm:size-[52px]">
                <AvatarImage src={user?.avatar_url} alt={displayName} />
                <AvatarFallback className="bg-foreground text-sm font-medium text-background">
                  {displayName.charAt(0).toUpperCase()}
                </AvatarFallback>
              </Avatar>
            )}
            <div className="min-w-0 space-y-1.5">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="text-xl font-semibold leading-tight tracking-tight sm:text-[26px]">
                  {heroTitle}
                </h1>
                {isAdminWorkspace ? (
                  <span className="inline-flex items-center gap-1 rounded-md border border-violet-500/20 bg-violet-500/10 px-1.5 py-0.5 text-[10px] font-medium text-violet-700 dark:text-violet-400">
                    <ShieldCheck className="size-3" />
                    ADMIN
                  </span>
                ) : (
                  user?.username && (
                    <span className="hidden rounded-md border bg-background/60 px-2 py-0.5 text-[11px] font-medium text-muted-foreground backdrop-blur sm:inline-flex">
                      @{user.username}
                    </span>
                  )
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                {isAdminWorkspace ? heroSubAdmin : heroSubUser}
              </p>
            </div>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {isAdminWorkspace ? (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigate('/admin/users')}
                >
                  <Users className="size-3.5" />
                  {t('dashboard.hero.inviteUsers')}
                </Button>
                <Button size="sm" onClick={() => navigate('/admin/storage')}>
                  <HardDrive className="size-3.5" />
                  {t('dashboard.hero.newBackend')}
                </Button>
              </>
            ) : (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigate('/user/folders')}
                >
                  <Plus className="size-3.5" />
                  {t('dashboard.hero.newFolder')}
                </Button>
                <Button size="sm" onClick={() => navigate('/user/files')}>
                  <Upload className="size-3.5" />
                  {t('common.upload')}
                </Button>
              </>
            )}
          </div>
        </div>

        {/* KPI strip */}
        <div className="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
          {isAdminWorkspace ? (
            <>
              <HeroKPI
                label={t('dashboard.kpi.totalUsers')}
                value={(stats?.users ?? 0).toLocaleString()}
                delta={t('dashboard.kpi.activeThisWeek').replace(
                  '{count}',
                  String(activeUsers)
                )}
                deltaIcon={<ArrowUp className="size-3" />}
                accent="hsl(var(--chart-2))"
              />
              <HeroKPI
                label={t('dashboard.kpi.totalResources')}
                value={(stats?.total_files ?? 0).toLocaleString()}
                delta={t('dashboard.kpi.newThisWeek').replace(
                  '{count}',
                  String(weekUploads)
                )}
                deltaIcon={<TrendingUp className="size-3" />}
                accent="hsl(var(--chart-3))"
              />
              <HeroKPI
                label={t('dashboard.kpi.storageUsed')}
                value={formatSize(stats?.total_size ?? 0)}
                delta={t('dashboard.kpi.backendsCount')
                  .replace('{total}', String(totalBackends))
                  .replace('{active}', String(activeBackends))}
                accent="hsl(var(--chart-1))"
              />
              <HeroKPI
                label={t('dashboard.kpi.weeklyAccesses')}
                value={weekAccesses.toLocaleString()}
                delta={t('dashboard.kpi.apiPlusDownloads')}
                deltaIcon={<Activity className="size-3" />}
                accent="hsl(var(--chart-5))"
              />
            </>
          ) : (
            <>
              <HeroKPI
                label={t('dashboard.totalFiles')}
                value={(stats?.total_files ?? 0).toLocaleString()}
                delta={
                  locale === 'zh'
                    ? `本周 +${weekUploads}`
                    : `+${weekUploads} this week`
                }
                deltaIcon={<TrendingUp className="size-3" />}
                accent="hsl(var(--chart-3))"
              />
              <HeroKPI
                label={t('dashboard.storageUsed')}
                value={formatSize(storageUsed)}
                delta={
                  isUnlimited
                    ? t('dashboard.unlimitedStorage')
                    : hasLimit
                      ? locale === 'zh'
                        ? `共 ${formatSize(storageLimit)} · ${Math.round(storagePct)}%`
                        : `of ${formatSize(storageLimit)} · ${Math.round(storagePct)}%`
                      : '—'
                }
                progress={hasLimit ? storagePct : undefined}
                accent="hsl(var(--chart-2))"
              />
              <HeroKPI
                label={t('dashboard.kpi.weeklyAccesses')}
                value={weekAccesses.toLocaleString()}
                delta={t('dashboard.kpi.resourceRefs')}
                deltaIcon={<Eye className="size-3" />}
                accent="hsl(var(--chart-1))"
              />
              <HeroKPI
                label={t('dashboard.kpi.activeTokens')}
                value={(tokensData?.items?.length ?? 0).toLocaleString()}
                delta={
                  locale === 'zh'
                    ? `共 ${tokensData?.items?.length ?? 0} 个 Token`
                    : `${tokensData?.items?.length ?? 0} total`
                }
                deltaIcon={<Key className="size-3" />}
                accent="hsl(var(--chart-5))"
              />
            </>
          )}
        </div>
      </PageHero>

      {isAdminWorkspace ? (
        <>
          {/* ═════ ROW 1: Service Health + Resource Usage ═════ */}
          <div className="grid gap-4 lg:grid-cols-2">
            <ServiceHealthCard t={t} />
            <ResourceUsageCard
              stats={stats}
              daily={daily}
              backends={backendsData}
              locale={locale}
              t={t}
            />
          </div>

          {/* ═════ ROW 2: Storage Backends + Top Users ═════ */}
          <div className="grid gap-4 lg:grid-cols-2">
            <StorageBackendsCard
              backends={backendsData ?? []}
              onManage={() => navigate('/admin/storage')}
              t={t}
            />
            <TopUsersCard users={topUsers} locale={locale} t={t} />
          </div>

          {/* ═════ ROW 3: Trend + Heatmap ═════ */}
          {trendHeatmapRow}
        </>
      ) : (
        <>
          {/* ═════ ROW 1: Storage breakdown + Composition ═════ */}
          <div className="grid gap-4 lg:grid-cols-5">
            <Card className="gap-4 py-5 shadow-xs lg:col-span-3">
              <CardHeader className="px-5">
                <CardTitle className="text-sm">
                  {t('dashboard.storage.breakdownTitle')}
                </CardTitle>
                <CardDescription className="text-xs">
                  {t('dashboard.storage.breakdownSub')}
                </CardDescription>
                <CardAction>
                  <span className="inline-flex items-center gap-1 rounded-md border bg-background/50 px-2 py-0.5 text-[11px] tabular-nums text-muted-foreground">
                    <HardDrive className="size-3" />
                    {formatSize(totalBytes)}
                  </span>
                </CardAction>
              </CardHeader>
              <CardContent className="space-y-5 px-5">
                {statsLoading ? (
                  <>
                    <Skeleton className="h-2.5 w-full rounded-full" />
                    <div className="grid grid-cols-2 gap-x-5 gap-y-3 sm:grid-cols-4">
                      {Array.from({ length: 4 }).map((_, i) => (
                        <div key={`sk-pct-${i}`} className="space-y-1.5">
                          <Skeleton className="h-3 w-20" />
                          <Skeleton className="h-2 w-14" />
                        </div>
                      ))}
                    </div>
                    <div className="grid grid-cols-2 gap-x-5 gap-y-4 pt-1 sm:grid-cols-4">
                      {Array.from({ length: 4 }).map((_, i) => (
                        <div key={`sk-cnt-${i}`} className="space-y-2">
                          <Skeleton className="h-2.5 w-12" />
                          <Skeleton className="h-7 w-16" />
                          <Skeleton className="h-2 w-10" />
                        </div>
                      ))}
                    </div>
                  </>
                ) : (
                  <StackedStorageBar data={storageSegs} total={totalBytes} />
                )}
              </CardContent>
            </Card>

            <Card className="gap-4 py-5 shadow-xs lg:col-span-2">
              <CardHeader className="px-5">
                <CardTitle className="text-sm">
                  {t('dashboard.composition.title')}
                </CardTitle>
                <CardDescription className="text-xs">
                  {t('dashboard.composition.sub')}
                </CardDescription>
              </CardHeader>
              <CardContent className="flex items-center justify-center px-5">
                {statsLoading ? (
                  <Skeleton className="size-40 rounded-full" />
                ) : (
                  <Donut
                    segments={donutSegs}
                    total={totalCount}
                    size={160}
                    stroke={18}
                    label={t('dashboard.composition.centerLabel')}
                  />
                )}
              </CardContent>
            </Card>
          </div>

          {/* ═════ ROW 2: Trend + Heatmap ═════ */}
          {trendHeatmapRow}

          {/* ═════ ROW 3: Recent + Activity ═════ */}
          <div className="grid gap-4 lg:grid-cols-5">
            <RecentUploadsCard
              items={recent?.items ?? []}
              isLoading={recentLoading}
              locale={locale}
              t={t}
            />

            <Card className="gap-3 py-5 shadow-xs lg:col-span-2">
              <CardHeader className="px-5">
                <CardTitle className="text-sm">
                  {t('dashboard.activity.title')}
                </CardTitle>
                <CardDescription className="text-xs">
                  {t('dashboard.activity.sub')}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4 px-5">
                {activity.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    {t('files.noFilesHint')}
                  </p>
                )}
                {activity.map((a, i) => (
                  <div key={i} className="flex items-start gap-3">
                    <div
                      className={cn(
                        'mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-medium',
                        a.actorType === 'system'
                          ? 'bg-muted text-muted-foreground'
                          : a.actorType === 'you'
                            ? 'bg-foreground text-background'
                            : 'bg-muted'
                      )}
                    >
                      {a.actorType === 'system' ? (
                        <Cpu className="size-4" />
                      ) : a.actorType === 'you' ? (
                        <UserIcon className="size-4" />
                      ) : (
                        a.actor.charAt(0).toUpperCase()
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="text-sm leading-snug">
                        <b className="font-medium">{a.actor}</b>
                        <span className="text-muted-foreground">
                          {' '}
                          {actionLabel(a.actionKey, t)}{' '}
                        </span>
                        <span className="truncate">
                          {locale === 'zh' ? '「' : '\u201c'}
                          {a.target}
                          {locale === 'zh' ? '」' : '\u201d'}
                        </span>
                      </p>
                      <p className="mt-1 text-[11px] tabular-nums text-muted-foreground">
                        {formatRelativeTime(a.createdAt, t)}
                      </p>
                    </div>
                  </div>
                ))}
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </div>
  )
}

/* ────────────────────────────────────────────────────────────
 * RecentUploadsCard — user-dashboard 3rd row, left column
 * ──────────────────────────────────────────────────────────── */
function RecentUploadsCard({
  items,
  isLoading,
  locale,
  t,
}: {
  items: ThumbFile[]
  isLoading: boolean
  locale: Locale
  t: (k: string) => string
}) {
  return (
    <Card className="gap-4 py-5 shadow-xs lg:col-span-3">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t('dashboard.recentUploads')}
        </CardTitle>
        <CardDescription className="text-xs">
          {t('dashboard.recentUploadsDesc').replace(
            '{count}',
            String(items.length)
          )}
        </CardDescription>
        <CardAction>
          <CardLinkFooter to="/user/files">
            {t('dashboard.viewAll')}
          </CardLinkFooter>
        </CardAction>
      </CardHeader>
      <CardContent className="px-5">
        {isLoading ? (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
            {Array.from({ length: 8 }).map((_, i) => (
              <FileThumbSkeleton key={i} />
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="py-12 text-center text-sm text-muted-foreground">
            {t('dashboard.noRecentFiles')}
          </div>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4">
            {items.slice(0, 8).map((f) => (
              <FileThumb key={f.id} file={f} locale={locale} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

/* ────────────────────────────────────────────────────────────
 * TopUsersCard — admin-dashboard 3rd row, left column
 * ──────────────────────────────────────────────────────────── */
function TopUsersCard({
  users,
  locale,
  t,
}: {
  users: TopUser[]
  locale: Locale
  t: (k: string) => string
}) {
  return (
    <Card className="gap-3 py-5 shadow-xs">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t('dashboard.topUsers.title')}
        </CardTitle>
        <CardDescription className="text-xs">
          {t('dashboard.topUsers.sub')}
        </CardDescription>
        <CardAction>
          <CardLinkFooter to="/admin/users">
            {t('dashboard.topUsers.all')}
          </CardLinkFooter>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-2.5 px-5">
        {users.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t('common.noData')}
          </div>
        ) : (
          users.map((u) => {
            const pct =
              u.storage_limit > 0
                ? Math.min(100, (u.storage_used / u.storage_limit) * 100)
                : 0
            const name = u.nickname?.trim() || u.username
            return (
              <div
                key={u.id}
                className="flex items-center gap-3 rounded-lg px-1.5 py-1.5 transition-colors hover:bg-muted/40"
              >
                <Avatar className="size-8">
                  <AvatarImage src={u.avatar_url} alt={name} />
                  <AvatarFallback className="text-[11px] font-medium">
                    {name.charAt(0).toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="min-w-0 flex-1">
                  <div className="flex items-baseline justify-between gap-2">
                    <span className="truncate text-sm font-medium">
                      {name}{' '}
                      <span className="text-muted-foreground">
                        @{u.username}
                      </span>
                    </span>
                    <span className="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                      {formatSize(u.storage_used)}
                      {u.storage_limit > 0 && (
                        <span>
                          {' '}
                          /{' '}
                          {u.storage_limit < 0
                            ? locale === 'zh'
                              ? '∞'
                              : '∞'
                            : formatSize(u.storage_limit)}
                        </span>
                      )}
                    </span>
                  </div>
                  <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="h-full bg-foreground/80 transition-[width]"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                </div>
              </div>
            )
          })
        )}
      </CardContent>
    </Card>
  )
}

/* ────────────────────────────────────────────────────────────
 * ServiceHealthCard — admin Row 3 left
 * 2×2 metric grid + bottom status bar (all operational + uptime)
 * ──────────────────────────────────────────────────────────── */

type LiveSystemStatus = {
  cpu_percent: number
  process_cpu_percent: number
  cpu_cores: number
  memory_used_bytes: number
  memory_total_bytes: number
  upload_mbps: number
  download_mbps: number
  upload_percent: number
  download_percent: number
  api_latency_ms: number
  p95_latency_ms: number
  cache_hit_rate_percent: number
  disk_io_mbps: number
  active_connections: number
  error_rate_percent: number
  uptime_days: number
  all_operational: boolean
}

function useLiveSystemStatus(): LiveSystemStatus | null {
  const [live, setLive] = useState<LiveSystemStatus | null>(null)

  useEffect(() => {
    let active = true
    let reconnectTimer: number | undefined
    let socket: WebSocket | null = null

    const connect = async () => {
      try {
        const ticketRes = await systemStatusApi.wsTicket()
        const ticket = ticketRes.data?.data?.ticket as string | undefined
        if (!ticket || !active) return

        const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
        const wsUrl = `${proto}://${window.location.host}/api/v1/admin/system-status/ws?ticket=${encodeURIComponent(ticket)}`
        socket = new WebSocket(wsUrl)

        socket.onmessage = (event) => {
          try {
            const payload = JSON.parse(event.data) as LiveSystemStatus
            if (active) setLive(payload)
          } catch {
            // ignore malformed payloads
          }
        }

        socket.onclose = () => {
          if (!active) return
          reconnectTimer = window.setTimeout(connect, 3000)
        }

        socket.onerror = () => {
          socket?.close()
        }
      } catch {
        if (!active) return
        reconnectTimer = window.setTimeout(connect, 3000)
      }
    }

    connect()

    return () => {
      active = false
      if (reconnectTimer != null) window.clearTimeout(reconnectTimer)
      if (socket && socket.readyState === WebSocket.OPEN) socket.close()
    }
  }, [])

  useEffect(() => {
    const tick = () => {
      void systemStatusApi.ping().catch(() => undefined)
    }
    tick()
    const timer = window.setInterval(tick, 10_000)
    return () => window.clearInterval(timer)
  }, [])

  return live
}

type MetricTone = 'default' | 'good' | 'warn' | 'err'

function HealthMetric({
  label,
  value,
  sub,
  tone = 'default',
}: {
  label: string
  value: string
  sub: string
  tone?: MetricTone
}) {
  // Tones desaturated one step vs. the old bg-muted/60 boxed version —
  // now that the number sits on the bare card surface there is nothing to
  // balance the previous saturated red/emerald/amber, and the softer tint
  // reads as a calm accent rather than an alert.
  const toneClass =
    tone === 'good'
      ? 'text-emerald-600 dark:text-emerald-400'
      : tone === 'warn'
        ? 'text-amber-600 dark:text-amber-400'
        : tone === 'err'
          ? 'text-red-600 dark:text-red-400'
          : 'text-foreground'
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className={cn(
          'mt-1 text-lg font-semibold leading-none tabular-nums',
          toneClass
        )}
      >
        {value}
      </div>
      <div className="mt-1 text-[10px] text-muted-foreground">{sub}</div>
    </div>
  )
}

function ServiceHealthCard({ t }: { t: (k: string) => string }) {
  const live = useLiveSystemStatus()
  const hasLive = !!live

  const cacheHitRate = live?.cache_hit_rate_percent ?? 0
  const p95 = live?.p95_latency_ms ?? 0
  const errRate = live?.error_rate_percent ?? 0
  const activeConn = live?.active_connections ?? 0
  const uptimeDays = live?.uptime_days ?? 0
  const allOperational = live?.all_operational ?? true

  const cacheTone: MetricTone = !hasLive
    ? 'default'
    : cacheHitRate >= 80
      ? 'good'
      : cacheHitRate >= 50
        ? 'warn'
        : 'err'
  const p95Tone: MetricTone = !hasLive
    ? 'default'
    : p95 < 100
      ? 'good'
      : p95 < 300
        ? 'default'
        : 'warn'
  const errTone: MetricTone = !hasLive
    ? 'default'
    : errRate < 1
      ? 'good'
      : errRate < 5
        ? 'warn'
        : 'err'

  const fmtPct = (n: number) =>
    hasLive ? `${n.toFixed(n < 10 ? 1 : 0)}%` : '--'
  const fmtMs = (n: number) =>
    hasLive ? `${n < 10 ? n.toFixed(1) : n.toFixed(0)}ms` : '--'

  return (
    // Card chrome aligned with StorageBackendsCard / TopUsersCard:
    // gap-3 py-5 shadow-xs, CardHeader + CardTitle + CardDescription,
    // px-5 content padding. Previously this card used a bespoke
    // gap-0 py-4 / px-[18px] shell with an inline dot-title row, which
    // broke the rhythm of the surrounding grid.
    <Card className="gap-3 py-5 shadow-xs">
      <CardHeader className="px-5">
        <CardTitle className="flex items-center gap-2 text-sm">
          {/* The live-status dot stays — it's a real signal that the
              widget is streaming current data, not decoration. */}
          <span
            className={cn(
              'size-1.5 shrink-0 rounded-full',
              allOperational ? 'bg-emerald-500' : 'bg-amber-500'
            )}
          />
          {t('dashboard.systemStatus.serviceHealth')}
        </CardTitle>
        <CardDescription className="text-xs">
          {t('dashboard.systemStatus.serviceHealthSub')}
        </CardDescription>
      </CardHeader>
      <CardContent className="px-5">
        {/* Flat 2×2 grid — the old bg-muted/60 blocks made each number
            read as an alert tile, which visually drowned the rest of
            the dashboard. With plain typography the tone colors pop
            just enough without dominating. */}
        <div className="grid grid-cols-2 gap-x-6 gap-y-4">
          <HealthMetric
            label={t('dashboard.systemStatus.cacheHitRate')}
            value={fmtPct(cacheHitRate)}
            sub={t('dashboard.systemStatus.hourAvg')}
            tone={cacheTone}
          />
          <HealthMetric
            label={t('dashboard.systemStatus.p95Latency')}
            value={fmtMs(p95)}
            sub={t('dashboard.systemStatus.hourAvg')}
            tone={p95Tone}
          />
          <HealthMetric
            label={t('dashboard.systemStatus.errorRate')}
            value={fmtPct(errRate)}
            sub={t('dashboard.systemStatus.errorRateSub')}
            tone={errTone}
          />
          <HealthMetric
            label={t('dashboard.systemStatus.activeConnections')}
            value={hasLive ? String(activeConn) : '--'}
            sub={t('dashboard.systemStatus.current')}
          />
        </div>

        {/* Footer strip: the border-t was replaced by a plain gap so
            the card surface stays uninterrupted, matching how other
            cards handle their auxiliary info. */}
        <div className="mt-5 flex items-center justify-between text-xs">
          <div className="flex items-center gap-1.5">
            <span
              className={cn(
                'size-1.5 rounded-full',
                allOperational ? 'bg-emerald-500' : 'bg-amber-500'
              )}
            />
            <span
              className={
                allOperational
                  ? 'text-emerald-600 dark:text-emerald-400'
                  : 'text-amber-600 dark:text-amber-400'
              }
            >
              {t('dashboard.systemStatus.allOperational')}
            </span>
          </div>
          <div className="tabular-nums text-muted-foreground">
            {t('dashboard.systemStatus.uptimeDays').replace(
              '{days}',
              String(uptimeDays)
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

/* ────────────────────────────────────────────────────────────
 * ResourceUsageCard — admin Row 3 right
 * Three progress rows (storage / bandwidth / file count) + two mini stats
 * ──────────────────────────────────────────────────────────── */
function ResourceBar({
  name,
  value,
  suffix,
  pct,
  color,
}: {
  name: string
  value: string
  suffix?: string
  pct: number
  color: string
}) {
  return (
    <div className="mb-3 last:mb-0">
      <div className="mb-[5px] flex items-baseline justify-between">
        <span className="text-xs text-muted-foreground">{name}</span>
        <span className="text-xs font-medium tabular-nums">
          {value}
          {suffix ? (
            <span className="ml-0.5 font-normal text-muted-foreground">
              {suffix}
            </span>
          ) : null}
        </span>
      </div>
      <div className="h-[5px] w-full overflow-hidden rounded-full bg-muted">
        <div
          className="h-full rounded-full transition-[width] duration-500"
          style={{
            width: `${Math.max(0, Math.min(100, pct))}%`,
            background: color,
          }}
        />
      </div>
    </div>
  )
}

function MiniStat({
  label,
  value,
  badge,
  sub,
}: {
  label: string
  value: string
  badge?: { text: string; up: boolean } | null
  sub: string
}) {
  // Same motivation as HealthMetric: remove the nested gray container so
  // the card surface stays clean. The delta badge keeps its saturated
  // emerald/red pill because it carries directional meaning (up / down)
  // that typography alone can't encode.
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 flex items-center gap-1.5">
        <span className="text-lg font-semibold leading-none tabular-nums">
          {value}
        </span>
        {badge && (
          <span
            className={cn(
              'rounded-[4px] px-[5px] py-px text-[10px] font-medium tabular-nums',
              badge.up
                ? 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400'
                : 'bg-red-500/15 text-red-700 dark:text-red-400'
            )}
          >
            {badge.text}
          </span>
        )}
      </div>
      <div className="mt-1 text-[10px] text-muted-foreground">{sub}</div>
    </div>
  )
}

function ResourceUsageCard({
  stats,
  daily,
  backends,
  locale,
  t,
}: {
  stats?: DashboardStats
  daily?: { days: DailyPoint[] }
  backends?: DashboardStorageBackend[]
  locale: Locale
  t: (k: string) => string
}) {
  const days = daily?.days ?? []

  const totalCapacity = (backends ?? []).reduce(
    (a, b) => a + (b.capacity_limit_bytes > 0 ? b.capacity_limit_bytes : 0),
    0
  )
  const storageUsed = stats?.total_size ?? 0
  const hasCapacity = totalCapacity > 0
  const storagePct = hasCapacity
    ? Math.min(100, (storageUsed / totalCapacity) * 100)
    : 0

  const monthBandwidthBytes = days
    .slice(-30)
    .reduce((a, b) => a + (b.bytes_served ?? 0), 0)
  // Visual-only scale: 1 GB reference fills ~10% of the bar. Label shows
  // raw usage without a cap since bandwidth is uncapped by default.
  const bandwidthPct = Math.min(
    100,
    (monthBandwidthBytes / (10 * 1024 * 1024 * 1024)) * 100
  )

  const totalFiles = stats?.total_files ?? 0
  // Visual-only: 1,000 files = 100%. Pure decoration — no denominator shown.
  const filesPct = Math.min(100, (totalFiles / 1000) * 100)

  const todayAccesses = days.length ? days[days.length - 1].accesses : 0
  const yesterdayAccesses = days.length > 1 ? days[days.length - 2].accesses : 0
  const todayUploads = days.length ? days[days.length - 1].uploads : 0
  const yesterdayUploads = days.length > 1 ? days[days.length - 2].uploads : 0

  const reqDelta =
    yesterdayAccesses > 0
      ? Math.round(
          ((todayAccesses - yesterdayAccesses) / yesterdayAccesses) * 100
        )
      : null
  const uploadDelta = todayUploads - yesterdayUploads

  const reqBadge =
    reqDelta !== null && reqDelta !== 0
      ? {
          text: `${reqDelta > 0 ? '↑' : '↓'}${Math.abs(reqDelta)}%`,
          up: reqDelta > 0,
        }
      : null
  const uploadBadge =
    uploadDelta !== 0
      ? {
          text: `${uploadDelta > 0 ? '↑' : '↓'}${Math.abs(uploadDelta)}`,
          up: uploadDelta > 0,
        }
      : null

  const filesUnit = t('dashboard.systemStatus.filesUnit')
  const vsYesterday = t('dashboard.systemStatus.vsYesterday')

  return (
    // Matches the ServiceHealthCard / StorageBackendsCard / TopUsersCard
    // shell so Row 1 and Row 2 read as a single visual family.
    <Card className="gap-3 py-5 shadow-xs">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t('dashboard.systemStatus.resourceUsage')}
        </CardTitle>
        <CardDescription className="text-xs">
          {t('dashboard.systemStatus.resourceUsageSub')}
        </CardDescription>
      </CardHeader>
      <CardContent className="px-5">
        <ResourceBar
          name={t('dashboard.systemStatus.storage')}
          value={formatSize(storageUsed)}
          suffix={
            hasCapacity
              ? ` / ${formatSize(totalCapacity)}`
              : ` / ${locale === 'zh' ? '∞' : '∞'}`
          }
          pct={storagePct}
          color="#7F77DD"
        />
        <ResourceBar
          name={t('dashboard.systemStatus.monthlyBandwidth')}
          value={formatSize(monthBandwidthBytes)}
          pct={bandwidthPct}
          color="#1D9E75"
        />
        <ResourceBar
          name={t('dashboard.systemStatus.totalFiles')}
          value={totalFiles.toLocaleString()}
          suffix={` ${filesUnit}`}
          pct={filesPct}
          color="#378ADD"
        />

        {/* Today stats: plain grid without border-t / gray boxes,
            consistent with the new ServiceHealthCard footer rhythm. */}
        <div className="mt-5 grid grid-cols-2 gap-x-6">
          <MiniStat
            label={t('dashboard.systemStatus.todayRequests')}
            value={todayAccesses.toLocaleString()}
            badge={reqBadge}
            sub={vsYesterday}
          />
          <MiniStat
            label={t('dashboard.systemStatus.todayUploads')}
            value={todayUploads.toLocaleString()}
            badge={uploadBadge}
            sub={vsYesterday}
          />
        </div>
      </CardContent>
    </Card>
  )
}

/* ────────────────────────────────────────────────────────────
 * StorageBackendsCard — admin dashboard Row 1, left (lg:col-span-3)
 *
 * Full-width rows: logo · name+badge · used/total · mono path · progress.
 * When there are more than 3 backends, a dashed-border footer card shows
 * stacked logos and "还有 N 个后端".
 * ──────────────────────────────────────────────────────────── */
function StorageBackendsCard({
  backends,
  onManage,
  t,
}: {
  backends: DashboardStorageBackend[]
  onManage: () => void
  t: (k: string) => string
}) {
  const sorted = [...backends].sort(
    (a, b) => Number(b.is_default) - Number(a.is_default)
  )
  const visible = sorted.slice(0, 3)
  const overflow = sorted.slice(3, 6)
  const remaining = Math.max(0, sorted.length - 3)

  return (
    <Card className="gap-3 py-5 shadow-xs">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t('dashboard.backends.title')}
        </CardTitle>
        <CardDescription className="text-xs">
          {t('dashboard.backends.sub')}
        </CardDescription>
        <CardAction>
          <Button
            variant="ghost"
            size="xs"
            onClick={onManage}
            className="text-muted-foreground hover:text-foreground"
          >
            {t('dashboard.backends.all')}
            <ArrowRight className="size-3" />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-3 px-5">
        {visible.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t('dashboard.backends.noBackends')}
          </div>
        ) : (
          visible.map((b) => {
            const hasLimit = b.capacity_limit_bytes > 0
            const pct = hasLimit
              ? Math.min(100, (b.used_bytes / b.capacity_limit_bytes) * 100)
              : 0
            const path = [b.driver, b.provider].filter(Boolean).join(' · ')
            return (
              <div
                key={b.id}
                className="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/30"
              >
                <StorageLogo
                  vendor={resolveLogoVendor(b.provider, b.driver)}
                  size={32}
                  rounded="rounded-md"
                  className="shrink-0"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex items-baseline justify-between gap-3">
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        {b.name}
                      </span>
                      {b.is_active ? (
                        <span className="shrink-0 rounded-md border border-emerald-500/30 bg-emerald-500/10 px-1.5 py-px text-[10px] font-medium text-emerald-700 dark:text-emerald-400">
                          active
                        </span>
                      ) : (
                        <span className="shrink-0 rounded-md border bg-muted/50 px-1.5 py-px text-[10px] font-medium text-muted-foreground">
                          idle
                        </span>
                      )}
                      {b.is_default && (
                        <span className="shrink-0 rounded-md border bg-muted/50 px-1.5 py-px text-[10px] font-medium uppercase text-muted-foreground">
                          {t('dashboard.backends.defaultBadge')}
                        </span>
                      )}
                    </div>
                    <span className="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                      {formatSize(b.used_bytes)} /{' '}
                      {hasLimit ? formatSize(b.capacity_limit_bytes) : '∞'}
                    </span>
                  </div>
                  <p className="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">
                    {path || b.id}
                  </p>
                  <div className="mt-2 h-1 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className={cn(
                        'h-full transition-[width]',
                        b.is_active
                          ? 'bg-foreground/70'
                          : 'bg-muted-foreground/40'
                      )}
                      style={{
                        width: hasLimit ? `${pct}%` : '100%',
                        opacity: hasLimit ? 1 : 0.3,
                      }}
                    />
                  </div>
                </div>
              </div>
            )
          })
        )}
        {remaining > 0 && (
          <button
            type="button"
            onClick={onManage}
            className="group flex w-full items-center justify-between rounded-lg border border-dashed p-3 text-left transition-colors hover:border-foreground/30 hover:bg-muted/30"
          >
            <div className="flex items-center gap-3">
              <div className="flex -space-x-1.5">
                {overflow.map((b) => (
                  <StorageLogo
                    key={b.id}
                    vendor={resolveLogoVendor(b.provider, b.driver)}
                    size={24}
                    rounded="rounded-md"
                    className="ring-2 ring-card"
                  />
                ))}
              </div>
              <span className="text-xs font-medium text-muted-foreground">
                {t('dashboard.backends.moreCount').replace(
                  '{count}',
                  String(remaining)
                )}
              </span>
            </div>
            <span className="text-[11px] font-medium text-muted-foreground group-hover:text-foreground">
              {t('dashboard.backends.viewAllMore')}
            </span>
          </button>
        )}
      </CardContent>
    </Card>
  )
}

/* Keep a no-op export of Files icon to satisfy tooling that inspects
 * imports from the old entry; TS allows unused imports to tree-shake. */
export { Files }
