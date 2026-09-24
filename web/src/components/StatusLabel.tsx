import { Archive, CalendarClock, CircleCheck, CircleDashed, type LucideIcon } from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { cn } from "@/lib/utils";

/**
 * Each status's icon and color, by the rule explore's console keeps: green is
 * live, blue is on its way, gray is not public.
 */
export const statuses: Record<string, { icon: LucideIcon; className: string }> = {
  published: { icon: CircleCheck, className: "text-success" },
  scheduled: { icon: CalendarClock, className: "text-info" },
  draft: { icon: CircleDashed, className: "text-muted-foreground" },
  archived: { icon: Archive, className: "text-muted-foreground" },
};

/** A status as its icon and name, both in the status's color. */
export function StatusLabel({ status, className }: { status: string; className?: string }) {
  const { t } = useI18n();
  const { icon: Icon, className: tone } = statuses[status] ?? statuses.draft;
  return (
    <span className={cn("inline-flex items-center gap-2 whitespace-nowrap", tone, className)}>
      <Icon className="size-4 shrink-0" />
      {t(`status.${status}` as Key)}
    </span>
  );
}
