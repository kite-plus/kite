import { useMemo } from "react";
import { Label, Pie, PieChart } from "recharts";
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
import { useI18n } from "@/i18n";
import { calcPercent } from "@/lib/utils";

interface FileTypeChartProps {
  totalFiles: number;
  imageCount: number;
  videoCount: number;
  audioCount: number;
  otherCount: number;
  isLoading: boolean;
}

export default function FileTypeChart({
  totalFiles,
  imageCount,
  videoCount,
  audioCount,
  otherCount,
  isLoading,
}: FileTypeChartProps) {
  const { t } = useI18n();

  const chartConfig = {
    count: { label: t("dashboard.filesTotal") },
    images: { label: t("dashboard.images"), color: "var(--color-chart-1)" },
    videos: { label: t("dashboard.videos"), color: "var(--color-chart-2)" },
    audio: { label: t("dashboard.audio"), color: "var(--color-chart-3)" },
    others: { label: t("dashboard.otherFiles"), color: "var(--color-chart-4)" },
  } satisfies ChartConfig;

  const data = useMemo(
    () => [
      { key: "images", label: t("dashboard.images"), count: imageCount, fill: "var(--color-images)" },
      { key: "videos", label: t("dashboard.videos"), count: videoCount, fill: "var(--color-videos)" },
      { key: "audio", label: t("dashboard.audio"), count: audioCount, fill: "var(--color-audio)" },
      { key: "others", label: t("dashboard.otherFiles"), count: otherCount, fill: "var(--color-others)" },
    ],
    [imageCount, videoCount, audioCount, otherCount, t],
  );

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <Skeleton className="h-5 w-28" />
          <Skeleton className="h-4 w-36" />
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4">
            <Skeleton className="mx-auto size-48 rounded-full" />
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-5 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card className="col-span-1">
      <CardHeader>
        <CardTitle className="text-sm font-medium">
          {t("dashboard.fileTypeDistribution")}
        </CardTitle>
        <CardDescription>{t("dashboard.byFileType")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col items-center gap-6 md:flex-row md:items-center md:justify-between">
          <ChartContainer
            config={chartConfig}
            className="mx-auto aspect-square w-full max-w-[220px]"
          >
            <PieChart>
              <ChartTooltip
                cursor={false}
                content={
                  <ChartTooltipContent
                    hideLabel
                    formatter={(value, _name, item) => {
                      const count = Number(value) || 0;
                      const pct = calcPercent(count, totalFiles);
                      return (
                        <div className="flex w-full items-center justify-between gap-4">
                          <div className="flex items-center gap-2">
                            <span
                              className="size-2.5 shrink-0 rounded-[2px]"
                              style={{ backgroundColor: item.payload.fill }}
                            />
                            <span className="text-muted-foreground">
                              {item.payload.label}
                            </span>
                          </div>
                          <div className="flex items-baseline gap-1.5 font-mono font-medium tabular-nums">
                            {count.toLocaleString()}
                            <span className="text-xs font-normal text-muted-foreground">
                              {pct}%
                            </span>
                          </div>
                        </div>
                      );
                    }}
                  />
                }
              />
              <Pie
                data={data}
                dataKey="count"
                nameKey="label"
                innerRadius={60}
                strokeWidth={4}
                paddingAngle={totalFiles > 0 ? 2 : 0}
              >
                <Label
                  content={({ viewBox }) => {
                    if (viewBox && "cx" in viewBox && "cy" in viewBox) {
                      return (
                        <text
                          x={viewBox.cx}
                          y={viewBox.cy}
                          textAnchor="middle"
                          dominantBaseline="middle"
                        >
                          <tspan
                            x={viewBox.cx}
                            y={viewBox.cy}
                            className="fill-foreground text-2xl font-semibold"
                          >
                            {totalFiles.toLocaleString()}
                          </tspan>
                          <tspan
                            x={viewBox.cx}
                            y={(viewBox.cy || 0) + 20}
                            className="fill-muted-foreground text-xs"
                          >
                            {t("dashboard.filesTotal")}
                          </tspan>
                        </text>
                      );
                    }
                  }}
                />
              </Pie>
            </PieChart>
          </ChartContainer>

          <div className="w-full space-y-2.5 md:max-w-[52%]">
            {data.map((item) => (
              <div key={item.key} className="flex items-center justify-between text-sm">
                <div className="flex min-w-0 items-center gap-2">
                  <span
                    className="size-2.5 shrink-0 rounded-[2px]"
                    style={{ backgroundColor: item.fill }}
                  />
                  <span className="truncate text-muted-foreground">{item.label}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="font-medium tabular-nums">
                    {item.count.toLocaleString()}
                  </span>
                  <span className="w-10 text-right text-xs text-muted-foreground tabular-nums">
                    {calcPercent(item.count, totalFiles)}%
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
