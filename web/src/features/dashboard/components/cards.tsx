import { Link } from "@tanstack/react-router";
import { ChevronRight, Trash2 } from "lucide-react";

import { useI18n } from "@/i18n";
import { useLatest, useSite, useStatusCounts, useTerms } from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { useDelivery, usePublish } from "@/hooks/usePublish";
import { isoDate } from "@/lib/dates";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { DeliveryStages } from "@/components/publish/Delivery";
import { statuses } from "@/components/StatusLabel";

/** LatestPosts is what went out last, newest first, each one a way back in. */
export function LatestPosts({ kind }: { kind: string }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const latest = useLatest(kind, 6, "-published_at");
  const items = latest.data?.items ?? [];

  return (
    <Card className="col-span-1 lg:col-span-3">
      <CardHeader>
        <CardTitle>{t("dashboard.latest", { kind: kindLabel.many(kind) })}</CardTitle>
        <CardDescription>{t("dashboard.latestNote")}</CardDescription>
      </CardHeader>
      <CardContent>
        {latest.isPending ? (
          <div className="space-y-4">
            {Array.from({ length: 4 }, (_, i) => (
              <Skeleton key={i} className="h-9 w-full" />
            ))}
          </div>
        ) : items.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("dashboard.latestEmpty", { kind: kindLabel.many(kind) })}
          </p>
        ) : (
          <div className="space-y-5">
            {items.map((item) => (
              <Link
                key={item.id}
                to="/content/$kind/$id"
                params={{ kind: item.kind, id: item.id }}
                className="group flex items-center gap-4"
              >
                <div className="min-w-0 flex-1 space-y-1">
                  <p className="truncate text-sm leading-none font-medium group-hover:underline">
                    {item.title || item.slug}
                  </p>
                  <p className="truncate text-sm text-muted-foreground">
                    {(item.taxonomies?.tags ?? []).map((tag) => `#${tag}`).join(" ") || item.slug}
                  </p>
                </div>
                <div className="shrink-0 text-sm text-muted-foreground tabular-nums">
                  {isoDate(item.published_at)}
                </div>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

/** DeliveryCard says how far the content has travelled, with the push at hand. */
export function DeliveryCard() {
  const { t } = useI18n();
  const delivery = useDelivery();
  const publish = usePublish([]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("publish.delivery")}</CardTitle>
        <CardDescription>
          {delivery.data?.branch
            ? `${delivery.data.remote ?? "origin"}/${delivery.data.branch}`
            : t("publish.title")}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {delivery.isPending ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <DeliveryStages delivery={delivery.data} publish={publish} />
        )}
      </CardContent>
    </Card>
  );
}

/** PendingCard lists what is waiting on the author, each a link to it. */
export function PendingCard({ kind }: { kind: string }) {
  const { t } = useI18n();
  const site = useSite();
  const counts = useStatusCounts(kind, ["draft", "scheduled", "trash"]);
  const problems = site.data?.problems?.length ?? 0;

  const rows = [
    {
      key: "draft",
      label: t("dashboard.pendingDrafts"),
      count: counts.draft,
      search: { status: ["draft"] },
      ...statuses.draft,
    },
    {
      key: "scheduled",
      label: t("dashboard.pendingScheduled"),
      count: counts.scheduled,
      search: { status: ["scheduled"] },
      ...statuses.scheduled,
    },
    {
      key: "trash",
      label: t("status.trash"),
      count: counts.trash,
      search: { trash: true },
      icon: Trash2,
      className: "text-muted-foreground",
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("dashboard.pending")}</CardTitle>
        <CardDescription>
          {problems ? t("problems.notIndexed", { count: problems }) : t("dashboard.healthy")}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-1">
        {rows.map((row) => (
          <Button key={row.key} variant="ghost" className="w-full justify-between px-2" asChild>
            <Link to="/content/$kind" params={{ kind }} search={row.search}>
              <span className="flex items-center gap-2">
                <row.icon className={cn("size-4", row.className)} />
                {row.label}
              </span>
              <span className="flex items-center gap-1 text-muted-foreground tabular-nums">
                {row.count ?? "—"}
                <ChevronRight className="size-4" />
              </span>
            </Link>
          </Button>
        ))}
      </CardContent>
    </Card>
  );
}

/** TopTerms are the tags used most, each opening the listing it narrows to. */
export function TopTerms({ kind, taxonomy }: { kind: string; taxonomy: string }) {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();
  const terms = useTerms(taxonomy);
  const items = (terms.data?.items ?? []).slice(0, 16);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("dashboard.topTerms", { name: taxonomyLabel(taxonomy) })}</CardTitle>
        <CardDescription>{t("dashboard.termCount", { count: terms.data?.items.length ?? 0 })}</CardDescription>
      </CardHeader>
      <CardContent>
        {items.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t("dashboard.noTaxonomies")}</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {items.map((item) => (
              <Link
                key={item.term}
                to="/content/$kind"
                params={{ kind }}
                search={{ terms: [`${taxonomy}:${item.term}`] }}
              >
                <Badge variant="secondary" className="gap-1.5 font-normal hover:bg-secondary/70">
                  {item.term}
                  <span className="text-muted-foreground tabular-nums">{item.count}</span>
                </Badge>
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
