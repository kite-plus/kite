import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts";

import { useI18n } from "@/i18n";
import type { TrendBucket } from "@/hooks/useContents";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

/**
 * What was published in each month, as columns.
 *
 * Loaded on its own because Recharts is the heaviest thing on the dashboard
 * and the rest of the page should not wait for it.
 */
export default function TrendChart({ data }: { data: TrendBucket[] }) {
  const { t, locale } = useI18n();

  const config = {
    count: { label: t("dashboard.trendSeries"), color: "var(--chart-1)" },
  } satisfies ChartConfig;

  const short = new Intl.DateTimeFormat(locale, { month: "short" });
  const long = new Intl.DateTimeFormat(locale, { year: "numeric", month: "long" });

  return (
    <>
      <ChartContainer config={config} className="aspect-auto size-full">
        <BarChart accessibilityLayer data={data} margin={{ top: 4, right: 0, left: 0, bottom: 0 }}>
          <CartesianGrid vertical={false} strokeDasharray="0" />
          <XAxis
            dataKey="month"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            minTickGap={16}
            tickFormatter={(month: string) => short.format(new Date(month))}
          />
          <YAxis allowDecimals={false} tickLine={false} axisLine={false} width={28} />
          <ChartTooltip
            cursor={false}
            content={
              <ChartTooltipContent
                hideIndicator
                labelFormatter={(_, payload) =>
                  long.format(new Date(payload[0]?.payload.month as string))
                }
              />
            }
          />
          <Bar dataKey="count" fill="var(--color-count)" radius={[4, 4, 0, 0]} maxBarSize={24} />
        </BarChart>
      </ChartContainer>

      {/* The same numbers for a reader who cannot use the plot. */}
      <table className="sr-only">
        <caption>{t("dashboard.trend")}</caption>
        <tbody>
          {data.map((bucket) => (
            <tr key={bucket.month}>
              <th scope="row">{long.format(new Date(bucket.month))}</th>
              <td>{bucket.count}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}
