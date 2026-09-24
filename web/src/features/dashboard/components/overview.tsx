import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

import { useI18n } from "@/i18n";
import { useMonthlyCounts } from "@/hooks/useContents";
import { QueryError } from "@/components/query-error";

/** Overview draws how much a kind published in each of the last twelve months. */
export function Overview({ kind }: { kind: string }) {
  const { locale } = useI18n();
  const month = new Intl.DateTimeFormat(locale, { month: "short" });
  const counts = useMonthlyCounts(kind, 12);
  if (counts.error) return <QueryError error={counts.error} onRetry={counts.refetch} />;
  const data = counts.months.map((entry) => ({
    name: month.format(entry.month),
    total: entry.total ?? 0,
  }));

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data}>
        <XAxis dataKey="name" stroke="#888888" fontSize={12} tickLine={false} axisLine={false} />
        <YAxis
          stroke="#888888"
          fontSize={12}
          tickLine={false}
          axisLine={false}
          allowDecimals={false}
          width={32}
        />
        <Tooltip
          cursor={{ className: "fill-muted" }}
          contentStyle={{
            background: "var(--popover)",
            border: "1px solid var(--border)",
            borderRadius: "var(--radius)",
            color: "var(--popover-foreground)",
            fontSize: 12,
          }}
        />
        <Bar dataKey="total" fill="currentColor" radius={[4, 4, 0, 0]} className="fill-primary" />
      </BarChart>
    </ResponsiveContainer>
  );
}
