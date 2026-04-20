import { memo } from "react";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  FileText,
  Image as ImageIcon,
  Music,
  Play,
} from "lucide-react";
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from "recharts";
import { cn, formatSize, formatRelativeTime } from "@/lib/utils";
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart";

/* ────────────────────────────────────────────────────────────
 * PageHero — backdrop + dot grid, used on dashboard
 * ──────────────────────────────────────────────────────────── */
export function PageHero({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-xl border bg-card",
        className
      )}
    >
      <div className="hero-backdrop" />
      <div className="dot-grid absolute inset-0 opacity-50" />
      <div className="relative p-5 sm:p-7">{children}</div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * HeroKPI — individual tile used in the hero strip
 * ──────────────────────────────────────────────────────────── */
interface HeroKPIProps {
  label: string;
  value: React.ReactNode;
  delta?: React.ReactNode;
  deltaIcon?: React.ReactNode;
  accent: string;
  progress?: number;
}

export function HeroKPI({
  label,
  value,
  delta,
  deltaIcon,
  accent,
  progress,
}: HeroKPIProps) {
  return (
    <div className="group relative overflow-hidden rounded-lg border bg-background/70 p-3.5 backdrop-blur transition-colors hover:border-foreground/20">
      <div className="flex items-baseline justify-between">
        <span className="text-[10px] font-semibold uppercase tracking-[0.08em] text-muted-foreground">
          {label}
        </span>
        <span
          className="size-1.5 rounded-full"
          style={{ background: accent }}
        />
      </div>
      <div className="mt-2 text-[22px] font-semibold leading-none tabular-nums tracking-tight">
        {value}
      </div>
      {progress != null ? (
        <div className="mt-3">
          <div className="h-1 w-full overflow-hidden rounded-full bg-muted/70">
            <div
              className="h-full bg-foreground/80 transition-[width]"
              style={{ width: `${Math.min(100, Math.max(0, progress))}%` }}
            />
          </div>
          {delta && (
            <p className="mt-1.5 text-[11px] tabular-nums text-muted-foreground">
              {delta}
            </p>
          )}
        </div>
      ) : (
        delta && (
          <p className="mt-2 flex items-center gap-1 text-[11px] text-muted-foreground">
            {deltaIcon}
            {delta}
          </p>
        )
      )}
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * StackedStorageBar — single stacked bar + two-row breakdown:
 *   • Row 1: compact legend (dot + name + %, bytes beneath)
 *   • Row 2: count tiles (dot + name, big count number, bytes beneath)
 * ──────────────────────────────────────────────────────────── */
export interface StorageSegment {
  kind: string;
  label: string;
  count: number;
  bytes: number;
  color: string;
}

export function StackedStorageBar({
  data,
  total,
}: {
  data: StorageSegment[];
  total?: number;
}) {
  const sum = total ?? data.reduce((a, b) => a + b.bytes, 0);
  const nonEmpty = data.filter((s) => s.bytes > 0);
  const cols = data.length >= 5 ? "sm:grid-cols-5" : "sm:grid-cols-4";

  return (
    <div className="space-y-5">
      {/* Stacked bar */}
      <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted/70">
        {nonEmpty.map((seg) => (
          <div
            key={seg.kind}
            style={{
              width: `${sum > 0 ? (seg.bytes / sum) * 100 : 0}%`,
              background: seg.color,
            }}
            className="transition-all hover:opacity-80"
            title={`${seg.label} · ${formatSize(seg.bytes)}`}
          />
        ))}
      </div>

      {/* Row 1 — compact legend with % */}
      <div className={cn("grid grid-cols-2 gap-x-5 gap-y-3", cols)}>
        {data.map((seg) => {
          const pct = sum > 0 ? Math.round((seg.bytes / sum) * 100) : 0;
          return (
            <div key={`pct-${seg.kind}`} className="min-w-0">
              <div className="flex items-center gap-1.5 text-xs">
                <span
                  className="size-1.5 shrink-0 rounded-full"
                  style={{ background: seg.color }}
                />
                <span className="truncate text-foreground">{seg.label}</span>
                <span className="ml-auto tabular-nums text-muted-foreground">
                  {pct}%
                </span>
              </div>
              <p className="mt-0.5 pl-3 text-[11px] tabular-nums text-muted-foreground">
                {formatSize(seg.bytes)}
              </p>
            </div>
          );
        })}
      </div>

      {/* Row 2 — hero count tiles */}
      <div className={cn("grid grid-cols-2 gap-x-5 gap-y-4 pt-1", cols)}>
        {data.map((seg) => (
          <div key={`cnt-${seg.kind}`} className="min-w-0">
            <div className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
              <span
                className="size-1.5 shrink-0 rounded-full"
                style={{ background: seg.color }}
              />
              <span className="truncate">{seg.label}</span>
            </div>
            <div className="mt-1.5 text-[26px] font-semibold leading-none tabular-nums tracking-tight">
              {seg.count.toLocaleString()}
            </div>
            <p className="mt-1.5 text-[11px] tabular-nums text-muted-foreground">
              {formatSize(seg.bytes)}
            </p>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * Donut — SVG donut with center label
 * ──────────────────────────────────────────────────────────── */
export interface DonutSegment {
  value: number;
  color: string;
  label: string;
}

export function Donut({
  segments,
  total,
  size = 160,
  stroke = 18,
  label,
}: {
  segments: DonutSegment[];
  total?: number;
  size?: number;
  stroke?: number;
  label: string;
}) {
  const r = (size - stroke) / 2;
  const c = 2 * Math.PI * r;
  const s = total ?? segments.reduce((a, b) => a + b.value, 0);
  let offset = 0;
  return (
    <div
      className="relative inline-flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <g transform={`rotate(-90 ${size / 2} ${size / 2})`}>
          <circle
            cx={size / 2}
            cy={size / 2}
            r={r}
            fill="none"
            stroke="hsl(var(--muted))"
            strokeWidth={stroke}
          />
          {s > 0 &&
            segments.map((seg, i) => {
              if (seg.value <= 0) return null;
              const pct = seg.value / s;
              const len = c * pct;
              const dash = `${len} ${c}`;
              const current = offset;
              offset += len;
              return (
                <circle
                  key={i}
                  cx={size / 2}
                  cy={size / 2}
                  r={r}
                  fill="none"
                  stroke={seg.color}
                  strokeWidth={stroke}
                  strokeDasharray={dash}
                  strokeDashoffset={-current}
                  strokeLinecap="butt"
                />
              );
            })}
        </g>
      </svg>
      <div className="absolute inset-0 flex flex-col items-center justify-center">
        <span className="text-[11px] font-medium text-muted-foreground">
          {label}
        </span>
        <span className="mt-0.5 text-[26px] font-semibold leading-none tabular-nums tracking-tight text-foreground">
          {s.toLocaleString()}
        </span>
      </div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * TrendCombo — bars for uploads + line+area for accesses.
 * Hand-rolled SVG so it inherits chart-* tokens cleanly.
 * ──────────────────────────────────────────────────────────── */
export interface TrendPoint {
  day: string; // ISO date or label
  uploads: number;
  accesses: number;
}

export function TrendCombo({
  data,
  uploadsLabel = "Uploads",
  accessesLabel = "Accesses",
}: {
  data: TrendPoint[];
  height?: number;
  uploadsLabel?: string;
  accessesLabel?: string;
}) {
  const chartConfig = {
    uploads: {
      label: uploadsLabel,
      color: "hsl(var(--chart-3))",
    },
    accesses: {
      label: accessesLabel,
      color: "hsl(var(--chart-2))",
    },
  } satisfies ChartConfig;

  if (!data.length) {
    return (
      <div className="flex h-full min-h-[160px] items-center justify-center text-xs text-muted-foreground">
        —
      </div>
    );
  }

  const fmtDate = (value: string) => {
    const d = new Date(value);
    if (Number.isNaN(d.getTime())) return value;
    return `${d.getMonth() + 1}/${d.getDate()}`;
  };

  return (
    <div className="h-full w-full">
      <ChartContainer config={chartConfig} className="h-full w-full aspect-auto">
        <AreaChart
          accessibilityLayer
          data={data}
          margin={{ left: 0, right: 8, top: 4, bottom: 0 }}
        >
          <CartesianGrid vertical={false} />
          <YAxis
            tickLine={false}
            axisLine={{
              stroke: "hsl(var(--border))",
              strokeDasharray: "4 4",
            }}
            orientation="left"
            tickMargin={4}
            width={28}
            allowDecimals={false}
            tickFormatter={(value) => Number(value).toLocaleString()}
          />
          <XAxis
            dataKey="day"
            tickLine={false}
            axisLine={false}
            tickMargin={8}
            minTickGap={28}
            tickFormatter={fmtDate}
          />
          <ChartTooltip
            cursor={false}
            content={
              <ChartTooltipContent
                indicator="dot"
                labelFormatter={(value) => fmtDate(String(value))}
              />
            }
          />
          <Area
            dataKey="uploads"
            type="natural"
            fill="var(--color-uploads)"
            fillOpacity={0.35}
            stroke="var(--color-uploads)"
            strokeWidth={2}
            stackId="traffic"
          />
          <Area
            dataKey="accesses"
            type="natural"
            fill="var(--color-accesses)"
            fillOpacity={0.35}
            stroke="var(--color-accesses)"
            strokeWidth={2}
            stackId="traffic"
          />
        </AreaChart>
      </ChartContainer>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * Heatmap — 7×24 SVG grid, responsive via viewBox
 * grid[weekday][hour] = count
 * ──────────────────────────────────────────────────────────── */
export function Heatmap({
  grid,
  weekdayLabels,
}: {
  grid: number[][];
  weekdayLabels: [string, string, string, string, string, string, string];
}) {
  const max = Math.max(1, ...grid.flat());
  const cellSize = 14;
  const gap = 3;
  const width = 24 * (cellSize + gap) + 24;
  const height = 7 * (cellSize + gap) + 18;

  const fill = (v: number) => {
    const t = v / max;
    if (t === 0) return "hsl(var(--muted))";
    return `color-mix(in oklab, hsl(var(--chart-3)) ${Math.round(
      30 + t * 70
    )}%, hsl(var(--muted)))`;
  };

  return (
    <div className="min-w-0">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="block h-auto w-full"
      >
        {[0, 6, 12, 18, 23].map((h) => (
          <text
            key={h}
            x={24 + h * (cellSize + gap) + cellSize / 2}
            y={10}
            textAnchor="middle"
            fontSize="9"
            fill="hsl(var(--muted-foreground))"
          >
            {h}
          </text>
        ))}
        {weekdayLabels.map((w, d) => (
          <text
            key={d}
            x={18}
            y={18 + d * (cellSize + gap) + cellSize * 0.75}
            textAnchor="end"
            fontSize="9"
            fill="hsl(var(--muted-foreground))"
          >
            {w}
          </text>
        ))}
        {grid.map((row, d) =>
          row.map((v, h) => (
            <rect
              key={`${d}-${h}`}
              x={24 + h * (cellSize + gap)}
              y={18 + d * (cellSize + gap)}
              width={cellSize}
              height={cellSize}
              rx="3"
              fill={fill(v)}
            >
              <title>{`${weekdayLabels[d]} ${h}:00 · ${v}`}</title>
            </rect>
          ))
        )}
      </svg>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * FileThumb — recent-uploads gallery tile
 * ──────────────────────────────────────────────────────────── */
export interface ThumbFile {
  id: string;
  original_name: string;
  file_type: string;
  size_bytes: number;
  url: string;
  thumb_url?: string;
  created_at: string;
}

/** Background gradient + subtle dot-grid tint per file type. The tint keeps
 *  placeholder tiles feeling intentional instead of flat gray. */
const THUMB_TINT: Record<string, string> = {
  image:
    "bg-[linear-gradient(135deg,hsl(var(--chart-3)/0.18),hsl(var(--chart-3)/0.05))]",
  video:
    "bg-[linear-gradient(135deg,hsl(var(--chart-2)/0.22),hsl(var(--chart-2)/0.05))]",
  audio:
    "bg-[linear-gradient(135deg,hsl(var(--chart-1)/0.22),hsl(var(--chart-1)/0.05))]",
  file: "bg-[linear-gradient(135deg,hsl(var(--muted)),hsl(var(--muted)/0.4))]",
};

/** 10-char uppercase badge: short extension if present, otherwise a type
 *  abbreviation. Keeps the tile legible on any theme. */
function badgeLabel(file: ThumbFile): string {
  const m = file.original_name.match(/\.([A-Za-z0-9]{1,5})$/);
  if (m) return m[1].toUpperCase();
  switch (file.file_type) {
    case "image":
      return "IMG";
    case "video":
      return "VID";
    case "audio":
      return "AUD";
    default:
      return "FILE";
  }
}

function PlaceholderIcon({ type }: { type: string }) {
  // Videos & audios get a prominent circular play button, per design.
  if (type === "video" || type === "audio") {
    return (
      <div className="flex size-11 items-center justify-center rounded-full bg-background/90 text-foreground shadow-sm backdrop-blur-sm">
        {type === "video" ? (
          <Play className="size-5 translate-x-px fill-current" />
        ) : (
          <Music className="size-5" strokeWidth={1.75} />
        )}
      </div>
    );
  }
  // Documents get a small card-style icon. Images already render the thumb,
  // so this branch only fires when the thumb URL is missing.
  if (type === "image") {
    return <ImageIcon className="size-8 text-muted-foreground/70" strokeWidth={1.5} />;
  }
  return (
    <div className="flex size-11 items-center justify-center rounded-md border border-border/60 bg-background/80 text-muted-foreground shadow-sm">
      <FileText className="size-5" strokeWidth={1.5} />
    </div>
  );
}

export const FileThumb = memo(function FileThumb({
  file,
  locale,
}: {
  file: ThumbFile;
  locale: string;
}) {
  const hasThumb = file.file_type === "image" && (file.thumb_url || file.url);
  const badge = badgeLabel(file);
  const tint = THUMB_TINT[file.file_type] ?? THUMB_TINT.file;
  return (
    <div className="group relative overflow-hidden rounded-lg border bg-card transition-colors hover:border-foreground/20">
      <div
        className={cn(
          "relative aspect-square w-full overflow-hidden",
          hasThumb ? "checker-bg" : tint,
        )}
      >
        {hasThumb ? (
          <img
            src={file.thumb_url || file.url}
            alt={file.original_name}
            className="h-full w-full object-contain transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center">
            <PlaceholderIcon type={file.file_type} />
          </div>
        )}
        {/* subtle dot grid overlay on placeholder tiles (no overlay on real images) */}
        {!hasThumb && (
          <div className="dot-grid pointer-events-none absolute inset-0 opacity-30" />
        )}
        {/* type badge — bottom-left */}
        <span className="absolute bottom-1.5 left-1.5 rounded-md bg-foreground/85 px-1.5 py-0.5 text-[9px] font-semibold tracking-wide text-background backdrop-blur-sm">
          {badge}
        </span>
      </div>
      <div className="px-2.5 py-2">
        <div
          className="truncate text-[12px] font-medium leading-tight"
          title={file.original_name}
        >
          {file.original_name}
        </div>
        <div className="mt-1 truncate text-[10.5px] tabular-nums text-muted-foreground">
          {formatSize(file.size_bytes)}
          <span className="mx-1 text-muted-foreground/60">·</span>
          {formatRelativeTime(file.created_at, locale)}
        </div>
      </div>
    </div>
  );
});

/* ────────────────────────────────────────────────────────────
 * FileThumbSkeleton — loading placeholder that matches FileThumb
 * ──────────────────────────────────────────────────────────── */
export function FileThumbSkeleton() {
  return (
    <div className="overflow-hidden rounded-lg border bg-card">
      <div className="aspect-square w-full animate-pulse bg-muted" />
      <div className="space-y-1.5 px-2.5 py-2">
        <div className="h-2.5 w-3/4 animate-pulse rounded bg-muted" />
        <div className="h-2 w-1/2 animate-pulse rounded bg-muted" />
      </div>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * HeatmapLegend — gradient strip under the heatmap
 * ──────────────────────────────────────────────────────────── */
export function HeatmapLegend({
  lowLabel,
  highLabel,
}: {
  lowLabel: string;
  highLabel: string;
}) {
  return (
    <div className="mt-3 flex items-center justify-between text-[10px] text-muted-foreground">
      <span>{lowLabel}</span>
      <div className="flex items-center gap-0.5">
        {[0.1, 0.3, 0.5, 0.7, 1].map((t, i) => (
          <span
            key={i}
            className="size-2.5 rounded-sm"
            style={{
              background: `color-mix(in oklab, hsl(var(--chart-3)) ${Math.round(
                30 + t * 70
              )}%, hsl(var(--muted)))`,
            }}
          />
        ))}
      </div>
      <span>{highLabel}</span>
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * CardLinkFooter — "View all →" row
 * ──────────────────────────────────────────────────────────── */
export function CardLinkFooter({
  to,
  children,
}: {
  to: string;
  children: React.ReactNode;
}) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-1 rounded-md text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
    >
      {children}
      <ArrowRight className="size-3" />
    </Link>
  );
}
