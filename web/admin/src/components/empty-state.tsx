import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/**
 * EmptyKite — friendly empty state with the Kite brand motif
 * (a small kite flying over a horizon, with optional title/hint/action).
 * Use in place of the bland "dashed box with an icon" placeholder.
 */
export function EmptyKite({
  title,
  hint,
  action,
  className,
  size = "md",
}: {
  title: string;
  hint?: string;
  action?: ReactNode;
  className?: string;
  size?: "sm" | "md" | "lg";
}) {
  const illoSize =
    size === "sm"
      ? { w: 128, h: 80 }
      : size === "lg"
        ? { w: 208, h: 130 }
        : { w: 176, h: 110 };
  const pad =
    size === "sm" ? "py-10" : size === "lg" ? "py-20" : "py-16";

  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-4 rounded-xl border border-dashed px-6 text-center",
        pad,
        className
      )}
    >
      <div
        className="relative"
        style={{ width: illoSize.w, height: illoSize.h }}
      >
        <svg
          viewBox="0 0 180 110"
          className="h-full w-full"
          aria-hidden
        >
          <defs>
            <linearGradient id="empty-kite-grad" x1="0" x2="1" y1="0" y2="1">
              <stop
                offset="0%"
                stopColor="hsl(var(--chart-2))"
                stopOpacity="0.6"
              />
              <stop
                offset="100%"
                stopColor="hsl(var(--chart-3))"
                stopOpacity="0.4"
              />
            </linearGradient>
          </defs>
          {/* horizon */}
          <line
            x1="0"
            y1="96"
            x2="180"
            y2="96"
            className="chart-gridline"
          />
          {/* dotted string */}
          <path
            d="M 130 90 Q 100 70, 70 40"
            fill="none"
            stroke="hsl(var(--muted-foreground))"
            strokeWidth="1"
            strokeDasharray="2 3"
          />
          {/* tail */}
          <path
            d="M 70 40 q 3 6, -1 10 q -4 4, 0 8"
            fill="none"
            stroke="hsl(var(--chart-1))"
            strokeWidth="1.4"
            strokeLinecap="round"
          />
          {/* kite diamond */}
          <g transform="translate(70 40) rotate(-12)">
            <polygon
              points="0,-20 16,0 0,20 -16,0"
              fill="url(#empty-kite-grad)"
              stroke="hsl(var(--foreground))"
              strokeOpacity="0.3"
              strokeWidth="1"
            />
            <line
              x1="0"
              y1="-20"
              x2="0"
              y2="20"
              stroke="hsl(var(--foreground))"
              strokeOpacity="0.25"
              strokeWidth="0.8"
            />
            <line
              x1="-16"
              y1="0"
              x2="16"
              y2="0"
              stroke="hsl(var(--foreground))"
              strokeOpacity="0.25"
              strokeWidth="0.8"
            />
          </g>
          {/* small cloud */}
          <g opacity="0.5" fill="hsl(var(--muted-foreground))">
            <circle cx="145" cy="26" r="5" />
            <circle cx="152" cy="28" r="6" />
            <circle cx="138" cy="30" r="4" />
          </g>
        </svg>
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium text-foreground">{title}</p>
        {hint && (
          <p className="max-w-xs text-xs text-muted-foreground">{hint}</p>
        )}
      </div>
      {action}
    </div>
  );
}
