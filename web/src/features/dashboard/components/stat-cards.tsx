import type { ReactNode } from "react";
import { Link, type LinkProps } from "@tanstack/react-router";
import {
  Activity,
  CalendarClock,
  ChevronRight,
  CircleDashed,
  GitCommitHorizontal,
  type LucideIcon,
} from "lucide-react";

import { useI18n } from "@/i18n";
import { useSite, useStatusCounts } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useDelivery } from "@/hooks/usePublish";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

type Tone = "success" | "warning" | "error" | "info" | "neutral";

// The toasts' state colors.
const toneClass: Record<Tone, string> = {
  success: "text-success",
  warning: "text-warning",
  error: "text-destructive",
  info: "text-info",
  neutral: "text-muted-foreground",
};

/**
 * StatCard is a figure with its state, after explore's console: as in the
 * toasts, only the icon carries the state.
 */
function StatCard({
  title,
  value,
  hint,
  icon: Icon,
  tone,
  link,
}: {
  title: string;
  /** Undefined while loading. */
  value: ReactNode | undefined;
  hint: ReactNode;
  icon: LucideIcon;
  tone: Tone;
  /** Where the figure is dealt with; without it the card is not a link. */
  link?: LinkProps;
}) {
  const className = "flex flex-col gap-3 rounded-xl border bg-card p-4 text-card-foreground shadow-xs";
  const body = (
    <>
      <div className="flex items-center gap-2 text-sm font-medium">
        <Icon
          className={cn("size-4 shrink-0", value === undefined ? toneClass.neutral : toneClass[tone])}
        />
        {title}
        {link && (
          <ChevronRight className="ms-auto size-4 text-muted-foreground opacity-0 transition-opacity group-hover:opacity-100" />
        )}
      </div>
      <div>
        {value === undefined ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <div className="text-2xl font-semibold tabular-nums">{value}</div>
        )}
        <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
      </div>
    </>
  );
  return link ? (
    <Link {...link} className={cn(className, "group transition-colors hover:bg-accent/50")}>
      {body}
    </Link>
  ) : (
    <div className={className}>{body}</div>
  );
}

/** StatCards are the four things a writer checks: what waits, and whether all is well. */
export function StatCards({ kind }: { kind: string }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const site = useSite();
  const counts = useStatusCounts(kind, ["draft", "scheduled"]);
  const delivery = useDelivery();

  const dirty = delivery.data?.dirty?.length ?? 0;
  const ahead = delivery.data?.ahead ?? 0;
  const problems = site.data?.problems?.length ?? 0;
  const many = kindLabel.many(kind);

  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <StatCard
        title={t("dashboard.unpublished")}
        icon={GitCommitHorizontal}
        value={delivery.data ? dirty : undefined}
        tone={dirty || ahead ? "warning" : "success"}
        hint={
          ahead
            ? t("publish.toPush", { count: ahead })
            : dirty
              ? t("dashboard.unpublishedNote")
              : t("dashboard.allPublished")
        }
      />
      <StatCard
        title={t("dashboard.scheduled")}
        icon={CalendarClock}
        value={counts.scheduled}
        tone={counts.scheduled ? "info" : "neutral"}
        hint={
          counts.scheduled ? t("dashboard.scheduledNote") : t("dashboard.noScheduled", { kind: many })
        }
        link={{ to: "/content/$kind", params: { kind }, search: { status: ["scheduled"] } }}
      />
      <StatCard
        title={t("dashboard.drafts")}
        icon={CircleDashed}
        value={counts.draft}
        tone="neutral"
        hint={counts.draft ? t("dashboard.draftsNote", { kind: many }) : t("dashboard.noDrafts")}
        link={{ to: "/content/$kind", params: { kind }, search: { status: ["draft"] } }}
      />
      <StatCard
        title={t("dashboard.health")}
        icon={Activity}
        value={site.data ? (problems ? problems : t("dashboard.healthOK")) : undefined}
        tone={problems ? "error" : "success"}
        hint={problems ? t("problems.notIndexed", { count: problems }) : t("dashboard.healthy")}
        // The listing says which files, and why.
        link={problems ? { to: "/content/$kind", params: { kind } } : undefined}
      />
    </div>
  );
}
