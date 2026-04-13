import { useQuery } from "@tanstack/react-query";
import { FileText, HardDrive, Image, Video, Music } from "lucide-react";
import { statsApi, fileApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { formatSize, calcPercent, formatRelativeTime } from "@/lib/utils";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

interface Stats {
  total_files: number;
  total_size: number;
  images: number;
  videos: number;
  audios: number;
  others: number;
}

const fileTypeConfig: Record<string, { icon: typeof FileText; bg: string; color: string }> = {
  image: { icon: Image, bg: "bg-amber-50 dark:bg-amber-950/30", color: "text-amber-600 dark:text-amber-400" },
  video: { icon: Video, bg: "bg-violet-50 dark:bg-violet-950/30", color: "text-violet-600 dark:text-violet-400" },
  audio: { icon: Music, bg: "bg-blue-50 dark:bg-blue-950/30", color: "text-blue-600 dark:text-blue-400" },
  file: { icon: FileText, bg: "bg-gray-100 dark:bg-gray-800/30", color: "text-gray-500 dark:text-gray-400" },
};

export default function UserDashboard() {
  const { t, locale } = useI18n();

  const { data: stats, isLoading: statsLoading } = useQuery<Stats>({
    queryKey: ["dashboard", "stats"],
    queryFn: () => statsApi.get().then((r) => r.data.data),
    staleTime: 30_000,
  });

  const { data: recentData, isLoading: filesLoading } = useQuery({
    queryKey: ["files", "recent"],
    queryFn: () => fileApi.list({ page: 1, size: 5, sort: "created_at", order: "desc" }).then((r) => r.data.data),
    staleTime: 15_000,
  });

  const s = stats ?? { total_files: 0, total_size: 0, images: 0, videos: 0, audios: 0, others: 0 };
  const recentFiles = recentData?.items ?? [];

  const statCards = [
    { label: t("dashboard.totalFiles"), value: s.total_files, icon: FileText, bg: "bg-muted", color: "text-foreground" },
    { label: t("dashboard.storageUsed"), value: formatSize(s.total_size), icon: HardDrive, bg: "bg-blue-50 dark:bg-blue-950/30", color: "text-blue-600 dark:text-blue-400" },
    { label: t("dashboard.images"), value: s.images, icon: Image, bg: "bg-amber-50 dark:bg-amber-950/30", color: "text-amber-600 dark:text-amber-400" },
    { label: t("dashboard.videos"), value: s.videos, icon: Video, bg: "bg-violet-50 dark:bg-violet-950/30", color: "text-violet-600 dark:text-violet-400" },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">{t("dashboard.title")}</h1>
        <p className="mt-1 text-sm text-muted-foreground">{t("dashboard.description")}</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {statCards.map((card) => (
          <Card key={card.label}>
            <CardContent className="flex items-center gap-3 p-4 sm:p-5">
              <div className={`flex size-9 sm:size-10 items-center justify-center rounded-lg shrink-0 ${card.bg}`}>
                <card.icon size={18} className={card.color} />
              </div>
              <div className="min-w-0">
                {statsLoading ? (
                  <Skeleton className="h-7 w-16" />
                ) : (
                  <p className="text-xl sm:text-2xl font-semibold tracking-tight">
                    {typeof card.value === "number" ? card.value.toLocaleString() : card.value}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">{card.label}</p>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        {/* File type distribution */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">{t("dashboard.fileTypeDistribution")}</CardTitle>
            <CardDescription className="text-xs">{t("dashboard.byFileType")}</CardDescription>
          </CardHeader>
          <CardContent>
            {statsLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="h-4 w-full" />)}
              </div>
            ) : (
              <div className="space-y-3">
                {[
                  { name: t("dashboard.images"), count: s.images, color: "bg-amber-500" },
                  { name: t("dashboard.videos"), count: s.videos, color: "bg-violet-500" },
                  { name: t("dashboard.audio"), count: s.audios, color: "bg-blue-500" },
                  { name: t("dashboard.otherFiles"), count: s.others, color: "bg-gray-400" },
                ].map((ft) => (
                  <div key={ft.name} className="flex items-center gap-3">
                    <span className={`h-2 w-2 rounded-full ${ft.color}`} />
                    <span className="w-14 text-sm">{ft.name}</span>
                    <div className="flex-1 overflow-hidden rounded-full bg-muted h-1.5">
                      <div
                        className={`h-full rounded-full ${ft.color}`}
                        style={{ width: `${s.total_files > 0 ? (ft.count / s.total_files) * 100 : 0}%` }}
                      />
                    </div>
                    <span className="w-8 text-right text-sm font-medium tabular-nums">{ft.count}</span>
                    <span className="w-12 text-right text-xs text-muted-foreground tabular-nums">
                      {calcPercent(ft.count, s.total_files)}%
                    </span>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Recent files */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">{t("dashboard.recentUploads")}</CardTitle>
            <CardDescription className="text-xs">{t("dashboard.latestFiles")}</CardDescription>
          </CardHeader>
          <CardContent>
            {filesLoading ? (
              <div className="space-y-3">
                {Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="h-10 w-full" />)}
              </div>
            ) : recentFiles.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">{t("dashboard.noFilesYet")}</p>
            ) : (
              <div className="space-y-1">
                {recentFiles.map((file: { id: string; original_name: string; size_bytes: number; mime_type: string; file_type: string; created_at: string }) => {
                  const cfg = fileTypeConfig[file.file_type] ?? fileTypeConfig.file;
                  const Icon = cfg.icon;
                  return (
                    <div key={file.id} className="flex items-center gap-3 rounded-md px-2 py-2 hover:bg-accent/50">
                      <div className={`flex size-8 items-center justify-center rounded-lg ${cfg.bg}`}>
                        <Icon size={14} className={cfg.color} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="truncate text-sm">{file.original_name}</p>
                        <p className="text-[11px] text-muted-foreground">{formatSize(file.size_bytes)}</p>
                      </div>
                      <span className="shrink-0 text-[11px] text-muted-foreground">
                        {formatRelativeTime(file.created_at, locale)}
                      </span>
                    </div>
                  );
                })}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
