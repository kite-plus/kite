import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  type ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { fileApi } from "@/lib/api";
import { useI18n } from "@/i18n";

type FileItem = {
  id: string;
  created_at: string;
};

type TrendPoint = {
  dateKey: string;
  label: string;
  count: number;
};

function buildTrend(items: FileItem[], locale: string): TrendPoint[] {
  const now = new Date();
  const days = 7;
  const buckets: TrendPoint[] = [];

  for (let i = days - 1; i >= 0; i -= 1) {
    const d = new Date(now);
    d.setHours(0, 0, 0, 0);
    d.setDate(now.getDate() - i);
    const dateKey = d.toISOString().slice(0, 10);
    const label =
      locale === "zh"
        ? `${d.getMonth() + 1}/${d.getDate()}`
        : d.toLocaleDateString("en-US", { month: "short", day: "numeric" });

    buckets.push({ dateKey, label, count: 0 });
  }

  const indexByKey = new Map(buckets.map((b, idx) => [b.dateKey, idx]));
  items.forEach((item) => {
    const d = new Date(item.created_at);
    if (Number.isNaN(d.getTime())) return;
    const key = new Date(d.getFullYear(), d.getMonth(), d.getDate())
      .toISOString()
      .slice(0, 10);
    const idx = indexByKey.get(key);
    if (idx !== undefined) buckets[idx].count += 1;
  });

  return buckets;
}

export default function UploadTrendChart() {
  const { t, locale } = useI18n();

  const { data, isLoading } = useQuery({
    queryKey: ["dashboard", "upload-trend"],
    queryFn: () =>
      fileApi
        .list({ page: 1, size: 120, sort: "created_at", order: "desc" })
        .then((r) => r.data.data),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const points = useMemo<TrendPoint[]>(() => {
    const items: FileItem[] = data?.items ?? [];
    return buildTrend(items, locale);
  }, [data?.items, locale]);

  const total = useMemo(() => points.reduce((sum, p) => sum + p.count, 0), [points]);

  const chartConfig = {
    count: {
      label: t("dashboard.uploads"),
      color: "var(--color-chart-1)",
    },
  } satisfies ChartConfig;

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-24" />
          <Skeleton className="h-4 w-36" />
        </CardHeader>
        <CardContent>
          <Skeleton className="h-56 w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="col-span-1">
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          {t("dashboard.recentUploads")}
        </CardTitle>
        <CardDescription>{t("dashboard.trendLast7Days")}</CardDescription>
      </CardHeader>
      <CardContent>
        <ChartContainer config={chartConfig} className="h-56 w-full">
          <AreaChart
            accessibilityLayer
            data={points}
            margin={{ left: 4, right: 12, top: 8, bottom: 0 }}
          >
            <defs>
              <linearGradient id="uploadTrendFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="var(--color-count)" stopOpacity={0.35} />
                <stop offset="95%" stopColor="var(--color-count)" stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} strokeDasharray="3 3" opacity={0.4} />
            <XAxis
              dataKey="label"
              tickLine={false}
              axisLine={false}
              tickMargin={8}
              minTickGap={8}
            />
            <YAxis
              hide
              allowDecimals={false}
              domain={[0, (dataMax: number) => Math.max(4, Math.ceil(dataMax * 1.2))]}
            />
            <ChartTooltip
              cursor={{ strokeDasharray: "3 3", opacity: 0.6 }}
              content={
                <ChartTooltipContent
                  indicator="line"
                  labelFormatter={(label) => label}
                />
              }
            />
            <Area
              dataKey="count"
              type="monotone"
              stroke="var(--color-count)"
              strokeWidth={2}
              fill="url(#uploadTrendFill)"
              activeDot={{ r: 4, strokeWidth: 2, stroke: "var(--color-background)" }}
            />
          </AreaChart>
        </ChartContainer>

        <div className="mt-3 flex items-center justify-between border-t pt-3 text-xs text-muted-foreground">
          <span>{t("dashboard.totalUploads7d")}</span>
          <span className="font-medium text-foreground tabular-nums">
            {total.toLocaleString()}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}
