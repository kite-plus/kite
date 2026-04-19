import { memo } from "react";
import { Link } from "react-router-dom";
import {
  ArrowRight,
  FileIcon,
  Image as ImageIcon,
  Music,
  Video,
} from "lucide-react";
import { cn, formatSize, formatRelativeTime } from "@/lib/utils";

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
 * StackedStorageBar — horizontal bar + legend grid
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
  return (
    <div className="space-y-3">
      <div className="flex h-3 w-full overflow-hidden rounded-full bg-muted/70">
        {nonEmpty.length > 0 ? (
          nonEmpty.map((seg) => (
            <div
              key={seg.kind}
              style={{
                width: `${sum > 0 ? (seg.bytes / sum) * 100 : 0}%`,
                background: seg.color,
              }}
              className="transition-all hover:opacity-80"
              title={`${seg.label} · ${formatSize(seg.bytes)}`}
            />
          ))
        ) : null}
      </div>
      <div className="grid grid-cols-2 gap-x-5 gap-y-2 sm:grid-cols-4">
        {data.map((seg) => (
          <div key={seg.kind} className="flex items-center gap-2 text-xs">
            <span
              className="size-2 shrink-0 rounded-sm"
              style={{ background: seg.color }}
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline justify-between gap-1">
                <span className="truncate text-foreground">{seg.label}</span>
                <span className="tabular-nums text-muted-foreground">
                  {sum > 0 ? Math.round((seg.bytes / sum) * 100) : 0}%
                </span>
              </div>
              <div className="mt-0.5 text-[10px] tabular-nums text-muted-foreground">
                {formatSize(seg.bytes)}
              </div>
            </div>
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
        <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          {label}
        </span>
        <span className="text-xl font-semibold tabular-nums text-foreground">
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
  height = 220,
  todayLabel = "Today",
  dayLabel = "d",
}: {
  data: TrendPoint[];
  height?: number;
  todayLabel?: string;
  dayLabel?: string;
}) {
  if (!data.length) {
    return (
      <div
        className="flex items-center justify-center text-xs text-muted-foreground"
        style={{ height }}
      >
        —
      </div>
    );
  }
  const w = 640;
  const padL = 32;
  const padR = 8;
  const padT = 10;
  const padB = 22;
  const uploads = data.map((d) => d.uploads);
  const accesses = data.map((d) => d.accesses);
  const maxU = Math.max(...uploads, 5);
  const maxA = Math.max(...accesses, 5);
  const barW = (w - padL - padR) / data.length - 2;
  const yScaleU = (v: number) =>
    height - padB - (v / maxU) * (height - padT - padB);
  const xAt = (i: number) =>
    padL + i * ((w - padL - padR) / data.length) + 1;

  const linePts = data.map(
    (d, i) =>
      [
        xAt(i) + barW / 2,
        height - padB - (d.accesses / maxA) * (height - padT - padB),
      ] as [number, number]
  );
  const linePath = linePts
    .map((p, i) => (i === 0 ? `M${p[0]},${p[1]}` : `L${p[0]},${p[1]}`))
    .join(" ");
  const area = `${linePath} L${xAt(data.length - 1) + barW / 2},${
    height - padB
  } L${xAt(0) + barW / 2},${height - padB} Z`;

  const ticks = [0, 0.5, 1].map((t) => ({
    y: height - padB - t * (height - padT - padB),
    label: Math.round(maxU * t),
  }));

  const everyX = Math.max(1, Math.ceil(data.length / 6));
  const dotEvery = Math.max(1, Math.ceil(data.length / 8));

  return (
    <svg
      viewBox={`0 0 ${w} ${height}`}
      className="h-full w-full"
      preserveAspectRatio="none"
    >
      <defs>
        <linearGradient id="trend-area" x1="0" x2="0" y1="0" y2="1">
          <stop
            offset="0%"
            stopColor="hsl(var(--chart-2))"
            stopOpacity="0.16"
          />
          <stop
            offset="100%"
            stopColor="hsl(var(--chart-2))"
            stopOpacity="0"
          />
        </linearGradient>
        <linearGradient id="bar-grad" x1="0" x2="0" y1="0" y2="1">
          <stop
            offset="0%"
            stopColor="hsl(var(--chart-3))"
            stopOpacity="0.95"
          />
          <stop
            offset="100%"
            stopColor="hsl(var(--chart-3))"
            stopOpacity="0.55"
          />
        </linearGradient>
      </defs>
      {/* gridlines */}
      {ticks.map((t, i) => (
        <g key={i}>
          <line
            x1={padL}
            x2={w - padR}
            y1={t.y}
            y2={t.y}
            className="chart-gridline"
          />
          <text
            x={padL - 4}
            y={t.y + 3}
            textAnchor="end"
            fontSize="9"
            fill="hsl(var(--muted-foreground))"
          >
            {t.label}
          </text>
        </g>
      ))}
      {/* bars — uploads */}
      {data.map((d, i) => {
        const y = yScaleU(d.uploads);
        const bh = height - padB - y;
        return (
          <rect
            key={i}
            x={xAt(i)}
            y={y}
            width={Math.max(2, barW)}
            height={Math.max(1, bh)}
            rx="2"
            fill="url(#bar-grad)"
          >
            <title>{`${d.day} · ${d.uploads}`}</title>
          </rect>
        );
      })}
      {/* access area + line */}
      <path d={area} fill="url(#trend-area)" />
      <path
        d={linePath}
        fill="none"
        stroke="hsl(var(--chart-2))"
        strokeWidth="1.8"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      {linePts
        .filter((_, i) => i % dotEvery === 0 || i === linePts.length - 1)
        .map((p, i) => (
          <circle
            key={i}
            cx={p[0]}
            cy={p[1]}
            r="2.5"
            fill="hsl(var(--background))"
            stroke="hsl(var(--chart-2))"
            strokeWidth="1.5"
          />
        ))}
      {/* x labels */}
      {data.map((_d, i) => {
        if (i % everyX !== 0 && i !== data.length - 1) return null;
        return (
          <text
            key={i}
            x={xAt(i) + barW / 2}
            y={height - 6}
            textAnchor="middle"
            fontSize="9"
            fill="hsl(var(--muted-foreground))"
          >
            {i === data.length - 1
              ? todayLabel
              : `${data.length - 1 - i}${dayLabel}`}
          </text>
        );
      })}
    </svg>
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
    <div className="overflow-x-auto">
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className="w-full"
        style={{ minWidth: 480 }}
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

function renderFileKindIcon(type: string) {
  const cls = "size-7";
  const w = 1.5;
  switch (type) {
    case "image":
      return <ImageIcon className={cls} strokeWidth={w} />;
    case "video":
      return <Video className={cls} strokeWidth={w} />;
    case "audio":
      return <Music className={cls} strokeWidth={w} />;
    default:
      return <FileIcon className={cls} strokeWidth={w} />;
  }
}

export const FileThumb = memo(function FileThumb({
  file,
  locale,
}: {
  file: ThumbFile;
  locale: string;
}) {
  const hasThumb = file.file_type === "image" && (file.thumb_url || file.url);
  return (
    <div className="group relative overflow-hidden rounded-lg border bg-muted/20 transition-colors hover:border-foreground/20">
      <div className="relative aspect-square w-full overflow-hidden bg-muted">
        {hasThumb ? (
          <img
            src={file.thumb_url || file.url}
            alt={file.original_name}
            className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.03]"
            loading="lazy"
          />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-muted-foreground/60">
            {renderFileKindIcon(file.file_type)}
          </div>
        )}
      </div>
      <div className="px-2 py-1.5">
        <div
          className="truncate text-[11px] font-medium leading-tight"
          title={file.original_name}
        >
          {file.original_name}
        </div>
        <div className="mt-0.5 flex items-center justify-between gap-1 text-[10px] tabular-nums text-muted-foreground">
          <span>{formatSize(file.size_bytes)}</span>
          <span className="truncate">
            {formatRelativeTime(file.created_at, locale)}
          </span>
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
    <div className="overflow-hidden rounded-lg border bg-muted/20">
      <div className="aspect-square w-full animate-pulse bg-muted" />
      <div className="space-y-1 px-2 py-1.5">
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
