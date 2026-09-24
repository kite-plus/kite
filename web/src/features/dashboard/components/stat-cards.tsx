import type { ReactNode } from "react";
import { FileText, GitCommitHorizontal, Newspaper, Tags } from "lucide-react";

import { useI18n } from "@/i18n";
import { useSite, useStatusCounts, useTaxonomies } from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { useDelivery } from "@/hooks/usePublish";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

function Stat({
  title,
  icon: Icon,
  value,
  note,
}: {
  title: string;
  icon: React.ElementType;
  value?: ReactNode;
  note?: ReactNode;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <Icon className="size-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        {value === undefined ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <div className="text-2xl font-bold tabular-nums">{value}</div>
        )}
        <p className="text-xs text-muted-foreground">{note ?? " "}</p>
      </CardContent>
    </Card>
  );
}

/** StatCards are the four numbers a writer checks: what exists and what waits. */
export function StatCards({ primary, kinds }: { primary: string; kinds: string[] }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const taxonomyLabel = useTaxonomyLabel();
  const site = useSite();
  const counts = useStatusCounts(primary);
  const taxonomies = useTaxonomies();
  const delivery = useDelivery();

  const secondary = kinds.find((kind) => kind !== primary);
  const terms = taxonomies.data?.items.reduce((sum, taxonomy) => sum + taxonomy.terms, 0);
  const dirty = delivery.data?.dirty?.length;
  const ahead = delivery.data?.ahead ?? 0;

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Stat
        title={kindLabel.many(primary)}
        icon={Newspaper}
        value={site.data?.counts?.[primary] ?? counts.all}
        note={t("dashboard.statusNote", {
          published: counts.published ?? 0,
          draft: counts.draft ?? 0,
          scheduled: counts.scheduled ?? 0,
        })}
      />
      {secondary && (
        <Stat
          title={kindLabel.many(secondary)}
          icon={FileText}
          value={site.data?.counts?.[secondary]}
        />
      )}
      <Stat
        title={t("nav.taxonomies")}
        icon={Tags}
        value={terms}
        note={taxonomies.data?.items
          .map((taxonomy) => `${taxonomyLabel(taxonomy.name)} ${taxonomy.terms}`)
          .join(" · ")}
      />
      <Stat
        title={t("dashboard.unpublished")}
        icon={GitCommitHorizontal}
        value={delivery.data ? (dirty ?? 0) : undefined}
        note={ahead ? t("publish.toPush", { count: ahead }) : t("dashboard.unpublishedNote")}
      />
    </div>
  );
}
