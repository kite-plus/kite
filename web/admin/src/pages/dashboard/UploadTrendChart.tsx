import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Area, AreaChart, XAxis, YAxis } from "recharts";
import {
  Card,
  CardAction,
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
import { adminStatsApi, statsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { formatSize } from "@/lib/utils";

type DailyPoint = {
  day: string;
  uploads: number;
  accesses: number;
  bytes_served: number;
};

type SeriesKey = "uploads" | "accesses" | "bytes_served";

function formatDayLabel(day: string, locale: string) {
  const [, m, d] = day.split("-");
  if (!m || !d) return day;
  const mNum = Number(m);
  const dNum = Number(d);
  if (locale === "zh") return `${mNum}/${dNum}`;
  const date = new Date(day);
  return Number.isNaN(date.getTime())
    ? day
    : date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

export default function UploadTrendChart({ admin = false }: { admin?: boolean }) {
  const { t, locale } = useI18n();
  const [active, setActive] = useState<SeriesKey>("uploads");

  const { data, isLoading } = useQuery({
    queryKey: ["dashboard", "daily-stats", admin ? "admin" : "user"],
    queryFn: () =>
      (admin ? adminStatsApi.daily(7) : statsApi.daily(7)).then(
        (r) => r.data.data
      ),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });

  const points = useMemo<(DailyPoint & { label: string })[]>(() => {
    const days: DailyPoint[] = data?.days ?? [];
    return days.map((p) => ({ ...p, label: formatDayLabel(p.day, locale) }));
  }, [data?.days, locale]);

  const seriesMeta: Record<SeriesKey, { label: string }> = {
    uploads: { label: t("dashboard.uploads") },
    accesses: { label: t("dashboard.accesses") },
    bytes_served: { label: t("dashboard.bandwidth") },
  };

  const chartConfig = {
    uploads: { label: t("dashboard.uploads"), color: "hsl(var(--foreground))" },
    accesses: { label: t("dashboard.accesses"), color: "hsl(var(--foreground))" },
    bytes_served: {
      label: t("dashboard.bandwidth"),
      color: "hsl(var(--foreground))",
    },
  } satisfies ChartConfig;

  return (
    <Card className="gap-0 py-0 shadow-xs">
      <CardHeader className="px-4 pt-4 pb-3 sm:px-6 sm:pt-5">
        <CardTitle className="text-sm">
          {t("dashboard.activityTrend")}
        </CardTitle>
        <CardDescription className="text-xs">
          {seriesMeta[active].label}
        </CardDescription>
        <CardAction>
          <div className="inline-flex rounded-md bg-muted p-[2px]">
            {(Object.keys(seriesMeta) as SeriesKey[]).map((key) => {
              const isActive = key === active;
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => setActive(key)}
                  className={`rounded-[4px] px-2 py-[2px] text-[10px] transition-colors ${
                    isActive
                      ? "bg-card font-medium text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground/80"
                  }`}
                >
                  {seriesMeta[key].label}
                </button>
              );
            })}
          </div>
        </CardAction>
      </CardHeader>
      <CardContent className="px-4 pb-4 sm:px-6 sm:pb-5">
        {isLoading ? (
          <>
            <Skeleton className="h-[110px] w-full sm:h-[120px]" />
            <Skeleton className="mt-2 h-3 w-full" />
          </>
        ) : (
          <>
            <ChartContainer
              config={chartConfig}
              className="h-[110px] w-full sm:h-[120px]"
            >
              <AreaChart
                accessibilityLayer
                data={points}
                margin={{ left: 0, right: 0, top: 4, bottom: 0 }}
              >
                <defs>
                  <linearGradient
                    id="dashboardAreaFill"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="0%"
                      stopColor="hsl(var(--foreground))"
                      stopOpacity={0.12}
                    />
                    <stop
                      offset="100%"
                      stopColor="hsl(var(--foreground))"
                      stopOpacity={0}
                    />
                  </linearGradient>
                </defs>
                <XAxis dataKey="label" hide />
                <YAxis
                  hide
                  domain={[
                    0,
                    (dataMax: number) =>
                      Math.max(
                        active === "bytes_served" ? 1024 : 4,
                        Math.ceil(dataMax * 1.2)
                      ),
                  ]}
                />
                <ChartTooltip
                  cursor={{ strokeDasharray: "3 3", opacity: 0.4 }}
                  content={
                    <ChartTooltipContent
                      indicator="line"
                      labelFormatter={(label) => label}
                      formatter={(value) => {
                        const displayValue =
                          active === "bytes_served"
                            ? formatSize(Number(value))
                            : Number(value).toLocaleString();
                        return (
                          <div className="flex w-full items-center justify-between gap-3">
                            <span className="text-muted-foreground">
                              {seriesMeta[active].label}
                            </span>
                            <span className="font-mono font-medium tabular-nums text-foreground">
                              {displayValue}
                            </span>
                          </div>
                        );
                      }}
                    />
                  }
                />
                <Area
                  type="monotone"
                  dataKey={active}
                  stroke="hsl(var(--foreground))"
                  strokeWidth={1.5}
                  fill="url(#dashboardAreaFill)"
                  dot={false}
                  activeDot={{
                    r: 2.5,
                    strokeWidth: 1.5,
                    stroke: "hsl(var(--foreground))",
                    fill: "hsl(var(--background))",
                  }}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ChartContainer>

            <div className="mt-2 flex justify-between text-[9px] tabular-nums text-muted-foreground">
              {points.map((p) => (
                <span key={p.day}>{p.label}</span>
              ))}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
