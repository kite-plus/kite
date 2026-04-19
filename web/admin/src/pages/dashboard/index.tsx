import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router-dom";
import {
  Activity,
  ArrowUp,
  CheckCircle2,
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
} from "lucide-react";
import {
  adminFileApi,
  adminStatsApi,
  fileApi,
  statsApi,
  storageApi,
  tokenApi,
  userApi,
} from "@/lib/api";
import { useI18n, type Locale } from "@/i18n";
import { useAuth } from "@/hooks/use-auth";
import { cn, formatSize } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/components/ui/avatar";
import { Skeleton } from "@/components/ui/skeleton";
import { BrandIcon } from "@/components/storage-brand";
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
} from "./components";

/* ────────────────────────────────────────────────────────────
 * Types
 * ──────────────────────────────────────────────────────────── */
interface DashboardStats {
  total_files: number;
  total_size: number;
  images: number;
  videos: number;
  audios: number;
  others: number;
  users?: number;
}

interface DailyPoint {
  day: string;
  uploads: number;
  accesses: number;
  bytes_served: number;
}

interface TopUser {
  id: string;
  username: string;
  nickname?: string;
  avatar_url?: string;
  storage_used: number;
  storage_limit: number;
}

interface DashboardStorageBackend {
  id: string;
  name: string;
  driver: string;
  provider: string;
  capacity_limit_bytes: number;
  used_bytes: number;
  is_default: boolean;
  is_active: boolean;
}

/* ────────────────────────────────────────────────────────────
 * Helpers
 * ──────────────────────────────────────────────────────────── */

/** Chinese + English greeting based on hour of day. */
function buildGreeting(locale: Locale): { main: string; sub: string } {
  const h = new Date().getHours();
  if (locale === "zh") {
    if (h < 5) return { main: "夜深了", sub: "记得早些休息 🌙" };
    if (h < 11) return { main: "早安", sub: "新的一天从这里起飞" };
    if (h < 14) return { main: "午安", sub: "午后时光，轻盈整理" };
    if (h < 18) return { main: "下午好", sub: "收纳一下今天的灵感" };
    if (h < 22) return { main: "晚上好", sub: "今天又攒了不少好素材" };
    return { main: "夜安", sub: "最后看一眼，然后休息" };
  }
  if (h < 5) return { main: "Still up?", sub: "Get some rest soon 🌙" };
  if (h < 11) return { main: "Good morning", sub: "A fresh day, ready to fly" };
  if (h < 14) return { main: "Good afternoon", sub: "Tidy up in the soft midday" };
  if (h < 18) return { main: "Afternoon", sub: "File today's inspiration" };
  if (h < 22) return { main: "Good evening", sub: "Nice haul of material today" };
  return { main: "Wind down", sub: "One last look, then rest" };
}

/** Deterministic PRNG — used for the mocked heatmap & activity feed so the
 *  rendering is stable across renders without calling the backend. */
function rng(seed: number) {
  let s = seed;
  return () => {
    s = (s * 9301 + 49297) % 233280;
    return s / 233280;
  };
}

/** 7 rows (Mon-Sun) × 24 cols (hours) activity heatmap. Weighted toward work
 *  hours with a weekend damping. Replace with real data when the endpoint
 *  lands. */
function buildHeatmap(seed = 11): number[][] {
  const r = rng(seed);
  const grid: number[][] = [];
  for (let d = 0; d < 7; d++) {
    const row: number[] = [];
    for (let h = 0; h < 24; h++) {
      const workHours = h >= 9 && h <= 22 ? 1 : 0.2;
      const weekend = d >= 5 ? 0.7 : 1;
      const base = workHours * weekend;
      const val = Math.max(
        0,
        Math.round(base * (r() * 16 + 2) + r() * 4 - 2)
      );
      row.push(val);
    }
    grid.push(row);
  }
  return grid;
}

type ActivityWho = "system" | "you" | "other";
interface ActivityItem {
  who: ActivityWho;
  actor: string; // display name when who === "other"
  actionKey:
    | "uploaded"
    | "createdAlbum"
    | "revokedToken"
    | "downloaded"
    | "createdToken"
    | "backupDone";
  target: string;
  minutes: number;
}

function buildActivity(locale: Locale): ActivityItem[] {
  // Stable mocked feed — same across renders. Replace when an activity
  // endpoint exists.
  const zh = locale === "zh";
  return [
    {
      who: "other",
      actor: zh ? "苏伟明" : "Suweiming",
      actionKey: "uploaded",
      target: "street-food-014.jpg",
      minutes: 4,
    },
    {
      who: "other",
      actor: zh ? "Ada Liu" : "Ada Liu",
      actionKey: "createdAlbum",
      target: zh ? "Studio · Q2" : "Studio · Q2",
      minutes: 18,
    },
    {
      who: "you",
      actor: zh ? "你" : "You",
      actionKey: "revokedToken",
      target: zh ? "只读分享" : "read-only share",
      minutes: 42,
    },
    {
      who: "other",
      actor: zh ? "于杰" : "Yujie",
      actionKey: "downloaded",
      target: "retro-console-007.mp4",
      minutes: 66,
    },
    {
      who: "system",
      actor: zh ? "系统" : "System",
      actionKey: "backupDone",
      target: zh ? "S3 · 主存储" : "S3 · primary",
      minutes: 120,
    },
    {
      who: "other",
      actor: zh ? "刘宇" : "Liuyu",
      actionKey: "createdToken",
      target: zh ? "博客自动化" : "blog automation",
      minutes: 240,
    },
  ];
}

function actionLabel(key: ActivityItem["actionKey"], locale: Locale): string {
  if (locale === "zh") {
    return {
      uploaded: "上传了",
      createdAlbum: "新建了相册",
      revokedToken: "撤销了 Token",
      downloaded: "下载了",
      createdToken: "创建了 Token",
      backupDone: "完成备份",
    }[key];
  }
  return {
    uploaded: "uploaded",
    createdAlbum: "created album",
    revokedToken: "revoked token",
    downloaded: "downloaded",
    createdToken: "created token",
    backupDone: "finished backup",
  }[key];
}

function relativeMinutes(mins: number, locale: Locale): string {
  if (locale === "zh") {
    if (mins < 60) return `${mins} 分钟前`;
    const h = Math.floor(mins / 60);
    if (h < 24) return `${h} 小时前`;
    return `${Math.floor(h / 24)} 天前`;
  }
  if (mins < 60) return `${mins}m ago`;
  const h = Math.floor(mins / 60);
  if (h < 24) return `${h}h ago`;
  return `${Math.floor(h / 24)}d ago`;
}

/* ────────────────────────────────────────────────────────────
 * DashboardPage
 * ──────────────────────────────────────────────────────────── */
export default function DashboardPage() {
  const { t, locale } = useI18n();
  const navigate = useNavigate();
  const location = useLocation();
  const { user } = useAuth();
  const isAdminWorkspace = location.pathname.startsWith("/admin");
  const displayName = user?.nickname?.trim() || user?.username || "";

  /* ── stats ──────────────────────────────────────────────── */
  const { data: stats, isLoading: statsLoading } = useQuery<DashboardStats>({
    queryKey: ["dashboard", "stats", isAdminWorkspace ? "admin" : "user"],
    queryFn: () =>
      (isAdminWorkspace ? adminStatsApi.get() : statsApi.get()).then(
        (r) => r.data.data
      ),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  /* ── daily trend (30d for chart, 7d sum for KPI) ────────── */
  const { data: daily } = useQuery<{ days: DailyPoint[] }>({
    queryKey: ["dashboard", "daily", isAdminWorkspace ? "admin" : "user"],
    queryFn: () =>
      (isAdminWorkspace
        ? adminStatsApi.daily(30)
        : statsApi.daily(30)
      ).then((r) => r.data.data),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  /* ── recent uploads (user) / top users (admin) ───────────── */
  const { data: recent, isLoading: recentLoading } = useQuery<{
    items: ThumbFile[];
  }>({
    queryKey: ["dashboard", "recent", isAdminWorkspace ? "admin" : "user"],
    queryFn: () =>
      (isAdminWorkspace
        ? adminFileApi.list({ page: 1, size: 8 })
        : fileApi.list({ page: 1, size: 8, only_self: true })
      ).then((r) => r.data.data),
    staleTime: 30_000,
  });

  const { data: topUsersData } = useQuery<{ items: TopUser[] }>({
    queryKey: ["dashboard", "topUsers"],
    enabled: isAdminWorkspace,
    queryFn: () =>
      userApi
        .list({ page: 1, size: 50 })
        .then((r) => r.data.data),
    staleTime: 60_000,
  });

  /* ── tokens count (user KPI) ────────────────────────────── */
  const { data: tokensData } = useQuery<{ items: Array<{ id: string }> }>({
    queryKey: ["dashboard", "tokens"],
    enabled: !isAdminWorkspace,
    queryFn: () => tokenApi.list().then((r) => r.data.data),
    staleTime: 60_000,
  });

  /* ── admin tokens count & storage backends ──────────────── */
  const { data: adminTokensData } = useQuery<Array<{ id: string }>>({
    queryKey: ["dashboard", "adminTokens"],
    enabled: isAdminWorkspace,
    queryFn: () => tokenApi.list().then((r) => r.data.data),
    staleTime: 60_000,
  });

  const { data: backendsData } = useQuery<DashboardStorageBackend[]>({
    queryKey: ["dashboard", "storageBackends"],
    enabled: isAdminWorkspace,
    queryFn: () => storageApi.list().then((r) => r.data.data),
    staleTime: 60_000,
  });

  /* ── derived values ─────────────────────────────────────── */
  const greeting = useMemo(() => buildGreeting(locale), [locale]);
  const heatmap = useMemo(() => buildHeatmap(), []);
  const activity = useMemo(() => buildActivity(locale), [locale]);

  const days30: TrendPoint[] = useMemo(() => {
    return (daily?.days ?? []).map((d) => ({
      day: d.day,
      uploads: d.uploads,
      accesses: d.accesses,
    }));
  }, [daily?.days]);

  const last7 = useMemo(() => days30.slice(-7), [days30]);
  const weekUploads = last7.reduce((a, b) => a + b.uploads, 0);
  const weekAccesses = last7.reduce((a, b) => a + b.accesses, 0);
  const todayUploads = days30.length ? days30[days30.length - 1].uploads : 0;

  const storageSegs: StorageSegment[] = useMemo(() => {
    const s = stats ?? {
      total_files: 0,
      total_size: 0,
      images: 0,
      videos: 0,
      audios: 0,
      others: 0,
    };
    const total = Math.max(1, s.total_files);
    // Approximate per-kind bytes by proportion of count · total_size. Not
    // exact but avoids a new endpoint. Replace with bytes-per-kind when the
    // backend exposes it.
    const byKind = (count: number) =>
      total > 0 ? Math.round((count / total) * s.total_size) : 0;
    return [
      {
        kind: "image",
        label: t("dashboard.images"),
        count: s.images,
        bytes: byKind(s.images),
        color: "hsl(var(--chart-3))",
      },
      {
        kind: "video",
        label: t("dashboard.videos"),
        count: s.videos,
        bytes: byKind(s.videos),
        color: "hsl(var(--chart-2))",
      },
      {
        kind: "audio",
        label: t("dashboard.audio"),
        count: s.audios,
        bytes: byKind(s.audios),
        color: "hsl(var(--chart-1))",
      },
      {
        kind: "other",
        label: t("dashboard.otherFiles"),
        count: s.others,
        bytes: byKind(s.others),
        color: "hsl(var(--chart-4))",
      },
    ];
  }, [stats, t]);

  const donutSegs: DonutSegment[] = storageSegs.map((s) => ({
    value: s.count,
    color: s.color,
    label: s.label,
  }));
  const totalBytes = storageSegs.reduce((a, b) => a + b.bytes, 0);
  const totalCount = storageSegs.reduce((a, b) => a + b.count, 0);

  const storageLimit = user?.storage_limit ?? 0;
  const storageUsed = stats?.total_size ?? 0;
  const isUnlimited = storageLimit < 0;
  const hasLimit = storageLimit > 0;
  const storagePct = hasLimit
    ? Math.min((storageUsed / storageLimit) * 100, 100)
    : 0;

  const topUsers = useMemo(
    () =>
      [...(topUsersData?.items ?? [])]
        .sort((a, b) => (b.storage_used ?? 0) - (a.storage_used ?? 0))
        .slice(0, 5),
    [topUsersData?.items]
  );

  /* ── hero content ───────────────────────────────────────── */
  const heroTitle = isAdminWorkspace
    ? `${greeting.main}${locale === "zh" ? "，" : ", "}${t("nav.roleAdmin")}`
    : `${greeting.main}${locale === "zh" ? "，" : ", "}${displayName}`;

  const heroSubUser =
    locale === "zh"
      ? `${greeting.sub} · 今天已同步 ${todayUploads} 个文件`
      : `${greeting.sub} · ${todayUploads} file${
          todayUploads === 1 ? "" : "s"
        } synced today`;
  const heroSubAdmin =
    locale === "zh"
      ? "整个 Kite 实例运行正常 · 实时同步中"
      : "Kite instance is healthy · live sync";

  /* ── render ─────────────────────────────────────────────── */
  return (
    <div className="page-enter flex flex-col gap-5 sm:gap-6">
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
                  onClick={() => navigate("/admin/users")}
                >
                  <Users className="size-3.5" />
                  {t("dashboard.hero.inviteUsers")}
                </Button>
                <Button
                  size="sm"
                  onClick={() => navigate("/admin/storage")}
                >
                  <HardDrive className="size-3.5" />
                  {t("dashboard.hero.newBackend")}
                </Button>
              </>
            ) : (
              <>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => navigate("/user/folders")}
                >
                  <Plus className="size-3.5" />
                  {t("dashboard.hero.newFolder")}
                </Button>
                <Button
                  size="sm"
                  onClick={() => navigate("/user/files")}
                >
                  <Upload className="size-3.5" />
                  {t("common.upload")}
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
                label={t("dashboard.kpi.totalUsers")}
                value={(stats?.users ?? 0).toLocaleString()}
                delta={t("dashboard.kpi.platformWide")}
                deltaIcon={<ArrowUp className="size-3" />}
                accent="hsl(var(--chart-2))"
              />
              <HeroKPI
                label={t("dashboard.kpi.totalResources")}
                value={(stats?.total_files ?? 0).toLocaleString()}
                delta={
                  locale === "zh"
                    ? `本周 +${weekUploads}`
                    : `+${weekUploads} this week`
                }
                deltaIcon={<TrendingUp className="size-3" />}
                accent="hsl(var(--chart-3))"
              />
              <HeroKPI
                label={t("dashboard.kpi.storageUsed")}
                value={formatSize(stats?.total_size ?? 0)}
                delta={t("dashboard.kpi.platformWide")}
                accent="hsl(var(--chart-1))"
              />
              <HeroKPI
                label={t("dashboard.kpi.weeklyAccesses")}
                value={weekAccesses.toLocaleString()}
                delta={t("dashboard.kpi.apiPlusDownloads")}
                deltaIcon={<Activity className="size-3" />}
                accent="hsl(var(--chart-5))"
              />
            </>
          ) : (
            <>
              <HeroKPI
                label={t("dashboard.totalFiles")}
                value={(stats?.total_files ?? 0).toLocaleString()}
                delta={
                  locale === "zh"
                    ? `本周 +${weekUploads}`
                    : `+${weekUploads} this week`
                }
                deltaIcon={<TrendingUp className="size-3" />}
                accent="hsl(var(--chart-3))"
              />
              <HeroKPI
                label={t("dashboard.storageUsed")}
                value={formatSize(storageUsed)}
                delta={
                  isUnlimited
                    ? t("dashboard.unlimitedStorage")
                    : hasLimit
                      ? locale === "zh"
                        ? `共 ${formatSize(storageLimit)} · ${Math.round(storagePct)}%`
                        : `of ${formatSize(storageLimit)} · ${Math.round(storagePct)}%`
                      : "—"
                }
                progress={hasLimit ? storagePct : undefined}
                accent="hsl(var(--chart-2))"
              />
              <HeroKPI
                label={t("dashboard.kpi.weeklyAccesses")}
                value={weekAccesses.toLocaleString()}
                delta={t("dashboard.kpi.resourceRefs")}
                deltaIcon={<Eye className="size-3" />}
                accent="hsl(var(--chart-1))"
              />
              <HeroKPI
                label={t("dashboard.kpi.activeTokens")}
                value={(tokensData?.items?.length ?? 0).toLocaleString()}
                delta={
                  locale === "zh"
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

      {/* ═════ ROW 1: Storage breakdown + Donut ═════════════ */}
      <div className="grid gap-4 lg:grid-cols-5">
        <Card className="gap-4 py-5 shadow-xs lg:col-span-3">
          <CardHeader className="px-5">
            <CardTitle className="text-sm">
              {t("dashboard.storage.breakdownTitle")}
            </CardTitle>
            <CardDescription className="text-xs">
              {t("dashboard.storage.breakdownSub")}
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
              {t("dashboard.composition.title")}
            </CardTitle>
            <CardDescription className="text-xs">
              {t("dashboard.composition.sub")}
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
                label={t("dashboard.composition.centerLabel")}
              />
            )}
          </CardContent>
        </Card>
      </div>

      {/* ═════ ROW 2: Trend + Heatmap ══════════════════════ */}
      <div className="grid gap-4 lg:grid-cols-5">
        <Card className="min-w-0 gap-3 overflow-hidden py-5 shadow-xs lg:col-span-3">
          <CardHeader className="px-5">
            <CardTitle className="text-sm">
              {t("dashboard.trend.title")}
            </CardTitle>
            <CardDescription className="text-xs">
              {t("dashboard.trend.sub")}
            </CardDescription>
            <CardAction>
              <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
                <span className="flex items-center gap-1">
                  <span
                    className="size-2 rounded-sm"
                    style={{ background: "hsl(var(--chart-3))" }}
                  />
                  {t("dashboard.uploads")}
                </span>
                <span className="flex items-center gap-1">
                  <span
                    className="size-2 rounded-sm"
                    style={{ background: "hsl(var(--chart-2))" }}
                  />
                  {t("dashboard.accesses")}
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
                  todayLabel={t("dashboard.trend.today")}
                  dayLabel={t("dashboard.trend.dayAbbr")}
                />
              )}
            </div>
          </CardContent>
        </Card>

        <Card className="min-w-0 gap-3 overflow-hidden py-5 shadow-xs lg:col-span-2">
          <CardHeader className="px-5">
            <CardTitle className="text-sm">
              {t("dashboard.heatmap.title")}
            </CardTitle>
            <CardDescription className="text-xs">
              {t("dashboard.heatmap.sub")}
            </CardDescription>
          </CardHeader>
          <CardContent className="min-w-0 px-5">
            <Heatmap
              grid={heatmap}
              weekdayLabels={[
                t("dashboard.weekdays.mon"),
                t("dashboard.weekdays.tue"),
                t("dashboard.weekdays.wed"),
                t("dashboard.weekdays.thu"),
                t("dashboard.weekdays.fri"),
                t("dashboard.weekdays.sat"),
                t("dashboard.weekdays.sun"),
              ]}
            />
            <HeatmapLegend
              lowLabel={t("dashboard.heatmap.low")}
              highLabel={t("dashboard.heatmap.high")}
            />
          </CardContent>
        </Card>
      </div>

      {/* ═════ ROW 2.5: System status + Storage backends (admin only) ═ */}
      {isAdminWorkspace && (
        <div className="grid gap-4 lg:grid-cols-5">
          <SystemStatusCard
            stats={stats}
            daily={daily?.days ?? []}
            backends={backendsData ?? []}
            tokensCount={adminTokensData?.length ?? 0}
            t={t}
          />
          <StorageBackendsCard
            backends={backendsData ?? []}
            onManage={() => navigate("/admin/storage")}
            t={t}
          />
        </div>
      )}

      {/* ═════ ROW 3: Recent / Top users + Activity ═══════════ */}
      <div className="grid gap-4 lg:grid-cols-5">
        {isAdminWorkspace ? (
          <TopUsersCard users={topUsers} locale={locale} t={t} />
        ) : (
          <RecentUploadsCard
            items={recent?.items ?? []}
            isLoading={recentLoading}
            locale={locale}
            t={t}
          />
        )}

        <Card className="gap-3 py-5 shadow-xs lg:col-span-2">
          <CardHeader className="px-5">
            <CardTitle className="text-sm">
              {t("dashboard.activity.title")}
            </CardTitle>
            <CardDescription className="text-xs">
              {t("dashboard.activity.sub")}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4 px-5">
            {activity.map((a, i) => (
              <div key={i} className="flex items-start gap-3">
                <div
                  className={cn(
                    "mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-full text-xs font-medium",
                    a.who === "system"
                      ? "bg-muted text-muted-foreground"
                      : a.who === "you"
                        ? "bg-foreground text-background"
                        : "bg-muted",
                  )}
                >
                  {a.who === "system" ? (
                    <Cpu className="size-4" />
                  ) : a.who === "you" ? (
                    <UserIcon className="size-4" />
                  ) : (
                    a.actor.charAt(0).toUpperCase()
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm leading-snug">
                    <b className="font-medium">{a.actor}</b>
                    <span className="text-muted-foreground">
                      {" "}
                      {actionLabel(a.actionKey, locale)}{" "}
                    </span>
                    <span className="truncate">
                      {locale === "zh" ? "「" : "\u201c"}
                      {a.target}
                      {locale === "zh" ? "」" : "\u201d"}
                    </span>
                  </p>
                  <p className="mt-1 text-[11px] tabular-nums text-muted-foreground">
                    {relativeMinutes(a.minutes, locale)}
                  </p>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
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
  items: ThumbFile[];
  isLoading: boolean;
  locale: Locale;
  t: (k: string) => string;
}) {
  return (
    <Card className="gap-4 py-5 shadow-xs lg:col-span-3">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t("dashboard.recentUploads")}
        </CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.recentUploadsDesc").replace(
            "{count}",
            String(items.length),
          )}
        </CardDescription>
        <CardAction>
          <CardLinkFooter to="/user/files">
            {t("dashboard.viewAll")}
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
            {t("dashboard.noRecentFiles")}
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
  );
}

/* ────────────────────────────────────────────────────────────
 * TopUsersCard — admin-dashboard 3rd row, left column
 * ──────────────────────────────────────────────────────────── */
function TopUsersCard({
  users,
  locale,
  t,
}: {
  users: TopUser[];
  locale: Locale;
  t: (k: string) => string;
}) {
  return (
    <Card className="gap-3 py-5 shadow-xs lg:col-span-3">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t("dashboard.topUsers.title")}
        </CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.topUsers.sub")}
        </CardDescription>
        <CardAction>
          <CardLinkFooter to="/admin/users">
            {t("dashboard.topUsers.all")}
          </CardLinkFooter>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-2.5 px-5">
        {users.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t("common.noData")}
          </div>
        ) : (
          users.map((u) => {
            const pct =
              u.storage_limit > 0
                ? Math.min(100, (u.storage_used / u.storage_limit) * 100)
                : 0;
            const name = u.nickname?.trim() || u.username;
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
                      {name}{" "}
                      <span className="text-muted-foreground">
                        @{u.username}
                      </span>
                    </span>
                    <span className="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                      {formatSize(u.storage_used)}
                      {u.storage_limit > 0 && (
                        <span>
                          {" "}
                          /{" "}
                          {u.storage_limit < 0
                            ? locale === "zh"
                              ? "∞"
                              : "∞"
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
            );
          })
        )}
      </CardContent>
    </Card>
  );
}

/* ────────────────────────────────────────────────────────────
 * SystemStatusCard — admin dashboard row 2.5, left (lg:col-span-3)
 * ──────────────────────────────────────────────────────────── */
function Gauge({
  value,
  label,
  accent = "hsl(var(--chart-1))",
}: {
  value: number; // 0..100
  label: string;
  accent?: string;
}) {
  const clamped = Math.max(0, Math.min(100, value));
  const size = 96;
  const stroke = 10;
  const radius = (size - stroke) / 2;
  const circ = radius * 2 * Math.PI;
  const offset = circ - (clamped / 100) * circ;
  return (
    <div className="flex flex-col items-center">
      <svg width={size} height={size} className="-rotate-90">
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="hsl(var(--muted))"
          strokeWidth={stroke}
          opacity={0.4}
        />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={accent}
          strokeWidth={stroke}
          strokeDasharray={circ}
          strokeDashoffset={offset}
          strokeLinecap="round"
          className="transition-[stroke-dashoffset] duration-700 ease-out"
        />
      </svg>
      <div className="-mt-[64px] flex size-[96px] flex-col items-center justify-center">
        <span className="text-base font-semibold tabular-nums">
          {clamped.toFixed(0)}%
        </span>
      </div>
      <p className="mt-1.5 text-[11px] text-muted-foreground">{label}</p>
    </div>
  );
}

function MetricTile({
  icon: Icon,
  label,
  value,
  accent,
}: {
  icon: React.ElementType;
  label: string;
  value: string;
  accent: string;
}) {
  return (
    <div className="flex items-start gap-3 rounded-lg border bg-muted/30 p-3">
      <div
        className="flex size-8 shrink-0 items-center justify-center rounded-md"
        style={{ background: `${accent}22`, color: accent }}
      >
        <Icon className="size-4" />
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[11px] text-muted-foreground">{label}</p>
        <p className="mt-0.5 truncate text-sm font-semibold tabular-nums">
          {value}
        </p>
      </div>
    </div>
  );
}

function SystemStatusCard({
  stats,
  daily,
  backends,
  tokensCount,
  t,
}: {
  stats: DashboardStats | undefined;
  daily: DailyPoint[];
  backends: DashboardStorageBackend[];
  tokensCount: number;
  t: (k: string) => string;
}) {
  const last7 = daily.slice(-7);
  const weekUploads = last7.reduce((a, b) => a + b.uploads, 0);
  const weekBytes = last7.reduce((a, b) => a + (b.bytes_served ?? 0), 0);

  const totalCapacity = backends
    .filter((b) => b.is_active && b.capacity_limit_bytes > 0)
    .reduce((a, b) => a + b.capacity_limit_bytes, 0);
  const storageUsed = stats?.total_size ?? 0;
  const storageUtil =
    totalCapacity > 0
      ? Math.min(100, (storageUsed / totalCapacity) * 100)
      : 0;

  // "Activity" gauge: this week's uploads vs target (best-effort cap at 500/wk)
  const activityPct = Math.min(100, (weekUploads / 500) * 100);

  return (
    <Card className="gap-3 py-5 shadow-xs lg:col-span-3">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t("dashboard.systemStatus.title")}
        </CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.systemStatus.sub")}
        </CardDescription>
        <CardAction>
          <span className="inline-flex items-center gap-1 rounded-md border border-[color:var(--chart-2)]/30 bg-[color:var(--chart-2)]/10 px-2 py-0.5 text-[11px] font-medium text-[color:var(--chart-2)]">
            <CheckCircle2 className="size-3" />
            {t("dashboard.systemStatus.allHealthy")}
          </span>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-4 px-5">
        <div className="grid grid-cols-2 items-center gap-4">
          <Gauge
            value={storageUtil}
            label={t("dashboard.systemStatus.storageUtil")}
            accent="hsl(var(--chart-1))"
          />
          <Gauge
            value={activityPct}
            label={t("dashboard.weeklyAccesses")}
            accent="hsl(var(--chart-3))"
          />
        </div>
        <div className="grid grid-cols-2 gap-2">
          <MetricTile
            icon={Upload}
            label={t("dashboard.systemStatus.weeklyUploads")}
            value={weekUploads.toLocaleString()}
            accent="hsl(var(--chart-3))"
          />
          <MetricTile
            icon={Activity}
            label={t("dashboard.systemStatus.weeklyBandwidth")}
            value={formatSize(weekBytes)}
            accent="hsl(var(--chart-2))"
          />
          <MetricTile
            icon={Key}
            label={t("dashboard.systemStatus.activeTokens")}
            value={tokensCount.toLocaleString()}
            accent="hsl(var(--chart-5))"
          />
          <MetricTile
            icon={HardDrive}
            label={t("dashboard.kpi.storageUsed")}
            value={formatSize(storageUsed)}
            accent="hsl(var(--chart-1))"
          />
        </div>
      </CardContent>
    </Card>
  );
}

/* ────────────────────────────────────────────────────────────
 * StorageBackendsCard — admin dashboard row 2.5, right (lg:col-span-2)
 * ──────────────────────────────────────────────────────────── */
function StorageBackendsCard({
  backends,
  onManage,
  t,
}: {
  backends: DashboardStorageBackend[];
  onManage: () => void;
  t: (k: string) => string;
}) {
  const sorted = [...backends]
    .sort((a, b) => Number(b.is_default) - Number(a.is_default))
    .slice(0, 4);
  return (
    <Card className="gap-3 py-5 shadow-xs lg:col-span-2">
      <CardHeader className="px-5">
        <CardTitle className="text-sm">
          {t("dashboard.backends.title")}
        </CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.backends.sub")}
        </CardDescription>
        <CardAction>
          <Button
            variant="ghost"
            size="xs"
            onClick={onManage}
            className="text-muted-foreground hover:text-foreground"
          >
            {t("dashboard.backends.all")} →
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="space-y-2.5 px-5">
        {sorted.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            {t("dashboard.backends.noBackends")}
          </div>
        ) : (
          sorted.map((b) => {
            const pct =
              b.capacity_limit_bytes > 0
                ? Math.min(
                    100,
                    (b.used_bytes / b.capacity_limit_bytes) * 100,
                  )
                : 0;
            return (
              <div
                key={b.id}
                className="flex items-center gap-3 rounded-lg border bg-card/60 px-3 py-2.5 transition-colors hover:bg-muted/40"
              >
                <BrandIcon
                  provider={b.provider}
                  driver={b.driver}
                  className="size-7 shrink-0"
                />
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5">
                    <span className="truncate text-sm font-medium">
                      {b.name}
                    </span>
                    {b.is_default && (
                      <span className="rounded-md border border-[color:var(--chart-2)]/30 bg-[color:var(--chart-2)]/10 px-1 text-[9px] font-medium uppercase text-[color:var(--chart-2)]">
                        {t("dashboard.backends.defaultBadge")}
                      </span>
                    )}
                    {!b.is_active && (
                      <span className="rounded-md border border-border bg-muted px-1 text-[9px] font-medium uppercase text-muted-foreground">
                        {t("dashboard.backends.inactiveBadge")}
                      </span>
                    )}
                  </div>
                  <div className="mt-1.5 h-1 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className={cn(
                        "h-full transition-[width]",
                        b.is_active
                          ? "bg-foreground/70"
                          : "bg-muted-foreground/40",
                      )}
                      style={{
                        width:
                          b.capacity_limit_bytes > 0 ? `${pct}%` : "100%",
                        opacity: b.capacity_limit_bytes > 0 ? 1 : 0.3,
                      }}
                    />
                  </div>
                </div>
                <div className="shrink-0 text-right text-[11px] tabular-nums text-muted-foreground">
                  <div className="font-medium text-foreground">
                    {formatSize(b.used_bytes)}
                  </div>
                  <div>
                    {b.capacity_limit_bytes > 0
                      ? formatSize(b.capacity_limit_bytes)
                      : "∞"}
                  </div>
                </div>
              </div>
            );
          })
        )}
      </CardContent>
    </Card>
  );
}

/* Keep a no-op export of Files icon to satisfy tooling that inspects
 * imports from the old entry; TS allows unused imports to tree-shake. */
export { Files };
