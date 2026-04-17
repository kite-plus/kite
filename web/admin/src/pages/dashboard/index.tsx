import { useQuery } from "@tanstack/react-query";
import { Files, HardDrive, Image, Upload, Users } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { statsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { calcPercent, formatSize } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import FileTypeChart from "./FileTypeChart";
import UploadTrendChart from "./UploadTrendChart";

interface DashboardStats {
  total_files: number;
  total_size: number;
  images: number;
  videos: number;
  audios: number;
  others: number;
  users: number;
}

export default function DashboardPage() {
  const { t } = useI18n();
  const navigate = useNavigate();

  const { data, isLoading } = useQuery<DashboardStats>({
    queryKey: ["dashboard", "stats"],
    queryFn: () => statsApi.get().then((r) => r.data.data),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const stats = data ?? {
    total_files: 0,
    total_size: 0,
    images: 0,
    videos: 0,
    audios: 0,
    others: 0,
    users: 0,
  };

  const summaryItems = [
    {
      key: "total",
      label: t("dashboard.totalFiles"),
      value: stats.total_files.toLocaleString(),
      icon: Files,
      hint: `${t("dashboard.images")} ${calcPercent(stats.images, stats.total_files)}%`,
    },
    {
      key: "storage",
      label: t("dashboard.storageUsed"),
      value: formatSize(stats.total_size),
      icon: HardDrive,
      hint: `${stats.total_files.toLocaleString()} ${t("dashboard.filesTotal")}`,
    },
    {
      key: "images",
      label: t("dashboard.images"),
      value: stats.images.toLocaleString(),
      icon: Image,
      hint: `${calcPercent(stats.images, stats.total_files)}% ${t("dashboard.percentOfTotal")}`,
    },
    {
      key: "users",
      label: t("dashboard.users"),
      value: stats.users.toLocaleString(),
      icon: Users,
      hint: "\u00A0",
    },
  ];

  return (
    <div className="space-y-5">
      <div className="flex flex-row items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
            {t("dashboard.title")}
          </h1>
          <p className="line-clamp-1 text-xs text-muted-foreground sm:text-sm">
            {t("dashboard.description")}
          </p>
        </div>
        <Button size="sm" className="shrink-0" onClick={() => navigate("/files")}>
          <Upload className="h-4 w-4" />
          <span className="ml-2">{t("common.upload")}</span>
        </Button>
      </div>

      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        {summaryItems.map((item) => {
          const Icon = item.icon;
          return (
            <Card key={item.key} className="gap-2 py-4 transition-colors hover:border-border">
              <CardHeader className="px-4 sm:px-6">
                <CardDescription className="truncate text-xs font-medium">
                  {item.label}
                </CardDescription>
                {isLoading ? (
                  <Skeleton className="mt-1 h-7 w-20" />
                ) : (
                  <CardTitle className="mt-1 truncate text-xl font-semibold tracking-tight tabular-nums sm:text-2xl">
                    {item.value}
                  </CardTitle>
                )}
                <CardAction>
                  <span className="inline-flex size-7 items-center justify-center rounded-md bg-muted text-muted-foreground sm:size-8">
                    <Icon className="size-3.5 sm:size-4" />
                  </span>
                </CardAction>
              </CardHeader>
              <CardContent className="px-4 sm:px-6">
                <p className="truncate text-xs text-muted-foreground tabular-nums">
                  {item.hint}
                </p>
              </CardContent>
            </Card>
          );
        })}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <FileTypeChart
          totalFiles={stats.total_files}
          imageCount={stats.images}
          videoCount={stats.videos}
          audioCount={stats.audios}
          otherCount={stats.others}
          isLoading={isLoading}
        />
        <UploadTrendChart />
      </div>
    </div>
  );
}
