import { cn } from "cn";

import { useI18n, type Key } from "@/i18n";

const dots: Record<string, string> = {
  published: "bg-success",
  scheduled: "bg-brand",
  draft: "bg-subtle",
  archived: "bg-border-strong",
};

const pills: Record<string, string> = {
  published:
    "border-green-200 bg-green-50 text-green-700 dark:border-green-500/25 dark:bg-green-500/10 dark:text-green-400",
  scheduled: "border-brand/25 bg-brand-soft text-brand",
  draft: "border-input bg-muted text-foreground-3",
  archived: "border-input bg-muted text-muted-foreground",
};

function Dot({ status, className }: { status: string; className?: string }) {
  return (
    <span
      aria-hidden
      className={cn("size-[7px] shrink-0 rounded-full", dots[status] ?? dots.draft, className)}
    />
  );
}

/** A status as a colored dot and its name, for rows where a badge is too loud. */
export function StatusDot({ status, className }: { status: string; className?: string }) {
  const { t } = useI18n();
  return (
    <span
      className={cn("inline-flex items-center gap-[7px] text-[12.5px] text-foreground-3", className)}
    >
      <Dot status={status} />
      {t(`status.${status}` as Key)}
    </span>
  );
}

/** A status in a tinted pill, for the one item a header is about. */
export function StatusPill({ status, className }: { status: string; className?: string }) {
  const { t } = useI18n();
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-[3px] text-[11.5px] whitespace-nowrap",
        pills[status] ?? pills.draft,
        className,
      )}
    >
      <Dot status={status} className="size-1.5" />
      {t(`status.${status}` as Key)}
    </span>
  );
}
