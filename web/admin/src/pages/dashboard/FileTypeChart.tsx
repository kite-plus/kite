import { useMemo } from "react";
import { Cell, Label, Pie, PieChart } from "recharts";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { ChartContainer } from "@/components/ui/chart";
import { Skeleton } from "@/components/ui/skeleton";
import { useI18n } from "@/i18n";

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

  const rows = useMemo(
    () => [
      { key: "images", name: t("dashboard.images"), value: imageCount, opacity: 1 },
      { key: "videos", name: t("dashboard.videos"), value: videoCount, opacity: 0.6 },
      { key: "audios", name: t("dashboard.audio"), value: audioCount, opacity: 0.35 },
      { key: "others", name: t("dashboard.otherFiles"), value: otherCount, opacity: 0.2 },
    ],
    [imageCount, videoCount, audioCount, otherCount, t]
  );

  const hasData = totalFiles > 0;
  const pieData = hasData
    ? rows.filter((r) => r.value > 0)
    : [{ key: "empty", name: "", value: 1, opacity: 0.08 }];

  return (
    <Card className="gap-0 py-0 shadow-xs">
      <CardHeader className="px-4 pt-4 pb-3 sm:px-6 sm:pt-5">
        <CardTitle className="text-sm">
          {t("dashboard.fileTypeDistribution")}
        </CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.byFileType")}
        </CardDescription>
      </CardHeader>
      <CardContent className="px-4 pb-4 sm:px-6 sm:pb-5">
        {isLoading ? (
          <div className="flex items-center gap-4 sm:gap-6">
            <Skeleton className="size-[110px] shrink-0 rounded-full" />
            <div className="flex-1 space-y-2">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-full" />
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-4 sm:gap-6">
            <div className="size-[110px] shrink-0 sm:size-[120px]">
              <ChartContainer config={{}} className="aspect-square w-full">
                <PieChart>
                  <Pie
                    data={pieData}
                    dataKey="value"
                    nameKey="name"
                    innerRadius={34}
                    outerRadius={50}
                    strokeWidth={0}
                    isAnimationActive={false}
                  >
                    {pieData.map((entry) => (
                      <Cell
                        key={entry.key}
                        fill="hsl(var(--foreground))"
                        fillOpacity={entry.opacity}
                      />
                    ))}
                    <Label
                      content={({ viewBox }) => {
                        if (!viewBox || !("cx" in viewBox) || !("cy" in viewBox))
                          return null;
                        const cx = Number(viewBox.cx ?? 0);
                        const cy = Number(viewBox.cy ?? 0);
                        return (
                          <text
                            x={cx}
                            y={cy}
                            textAnchor="middle"
                            dominantBaseline="middle"
                          >
                            <tspan
                              x={cx}
                              y={cy - 2}
                              className="fill-foreground font-semibold"
                              style={{
                                fontSize: 18,
                                fontVariantNumeric: "tabular-nums",
                              }}
                            >
                              {totalFiles.toLocaleString()}
                            </tspan>
                            <tspan
                              x={cx}
                              y={cy + 14}
                              className="fill-muted-foreground"
                              style={{ fontSize: 10 }}
                            >
                              {t("dashboard.filesTotal")}
                            </tspan>
                          </text>
                        );
                      }}
                    />
                  </Pie>
                </PieChart>
              </ChartContainer>
            </div>

            <div className="flex flex-1 flex-col gap-2 text-xs">
              {rows.map((row) => {
                const pct =
                  totalFiles > 0
                    ? Math.round((row.value / totalFiles) * 100)
                    : 0;
                return (
                  <div key={row.key} className="flex items-center gap-2">
                    <span
                      className="size-2 shrink-0 rounded-[2px]"
                      style={{
                        backgroundColor: "hsl(var(--foreground))",
                        opacity: row.opacity,
                      }}
                    />
                    <span className="flex-1 truncate text-muted-foreground">
                      {row.name}
                    </span>
                    <span className="tabular-nums text-foreground">
                      {row.value}
                    </span>
                    <span className="w-10 text-right tabular-nums text-muted-foreground">
                      {pct}%
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
