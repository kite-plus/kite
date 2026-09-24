import { useI18n, type Key } from "@/i18n";
import { cn } from "@/lib/utils";

const dots: Record<string, string> = {
  published: "bg-emerald-500",
  scheduled: "bg-sky-500",
  draft: "bg-slate-400",
  archived: "bg-slate-300 dark:bg-slate-600",
};

/** A status as a colored dot and its name, for rows where a badge is too loud. */
export function StatusDot({ status, className }: { status: string; className?: string }) {
  const { t } = useI18n();
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-muted-foreground", className)}>
      <span aria-hidden className={cn("size-2 shrink-0 rounded-full", dots[status] ?? dots.draft)} />
      {t(`status.${status}` as Key)}
    </span>
  );
}
