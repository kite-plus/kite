import { Link } from "@tanstack/react-router";

import { useI18n } from "@/i18n";
import { useLatest, useTerms } from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { useDelivery, usePublish } from "@/hooks/usePublish";
import { isoDate } from "@/lib/dates";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { DeliveryStages } from "@/components/publish/Delivery";
import { QueryError } from "@/components/query-error";

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
        {latest.error && !latest.data ? (
          <QueryError error={latest.error} onRetry={() => void latest.refetch()} />
        ) : latest.isPending ? (
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
        {delivery.error && !delivery.data ? (
          <QueryError error={delivery.error} onRetry={() => void delivery.refetch()} />
        ) : delivery.isPending ? (
          <Skeleton className="h-24 w-full" />
        ) : (
          <DeliveryStages delivery={delivery.data} publish={publish} />
        )}
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
        {terms.error && !terms.data ? (
          <QueryError error={terms.error} onRetry={() => void terms.refetch()} />
        ) : items.length === 0 ? (
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
