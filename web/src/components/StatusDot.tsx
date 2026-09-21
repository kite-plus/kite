import { cn } from "cn";

import { useI18n, type Key } from "@/i18n";
import { Badge } from "@/components/ui/badge";

const tones: Record<string, string> = {
  published: "bg-success",
  scheduled: "bg-brand",
  draft: "bg-muted-foreground/50",
  archived: "bg-muted-foreground/25",
};

function Dot({ status }: { status: string }) {
  return (
    <span
      aria-hidden
      className={cn("size-1.5 shrink-0 rounded-full", tones[status] ?? tones.draft)}
    />
  );
}

/** A status as a colored dot and its name, for rows where a badge is too loud. */
export function StatusDot({ status, className }: { status: string; className?: string }) {
  const { t } = useI18n();
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-muted-foreground", className)}>
      <Dot status={status} />
      {t(`status.${status}` as Key)}
    </span>
  );
}

export function StatusBadge({ status }: { status: string }) {
  const { t } = useI18n();
  return (
    <Badge variant="outline" className="font-normal">
      <Dot status={status} />
      {t(`status.${status}` as Key)}
    </Badge>
  );
}
