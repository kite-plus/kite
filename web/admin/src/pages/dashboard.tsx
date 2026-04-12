import { useQuery } from "@tanstack/react-query";
import { Image, Video, Music, FileText, HardDrive, Users } from "lucide-react";
import { statsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export default function DashboardPage() {
  const { t } = useI18n();
  const { data, isLoading } = useQuery({
    queryKey: ["stats"],
    queryFn: () => statsApi.get().then((r) => r.data.data),
  });

  const cards = [
    {
      titleKey: "dashboard.totalFiles",
      value: data?.total_files ?? 0,
      icon: FileText,
    },
    {
      titleKey: "dashboard.storageUsed",
      value: data ? formatBytes(data.total_size) : "0 B",
      icon: HardDrive,
    },
    { titleKey: "dashboard.images", value: data?.images ?? 0, icon: Image },
    { titleKey: "dashboard.videos", value: data?.videos ?? 0, icon: Video },
    { titleKey: "dashboard.audio", value: data?.audios ?? 0, icon: Music },
    { titleKey: "dashboard.users", value: data?.users ?? 0, icon: Users },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("dashboard.title")}</h1>
        <p className="text-sm text-muted-foreground">
          {t("dashboard.description")}
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {cards.map((card) => (
          <Card key={card.titleKey}>
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">
                {t(card.titleKey)}
              </CardTitle>
              <card.icon className="size-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              {isLoading ? (
                <Skeleton className="h-7 w-20" />
              ) : (
                <div className="text-2xl font-bold">{card.value}</div>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
