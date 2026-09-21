import { Suspense, lazy, useState, type ReactNode } from "react";
import {
  ExternalLink,
  LayoutTemplate,
  PanelsTopLeft,
  Plus,
  Settings2,
  type LucideIcon,
} from "lucide-react";

import { cn } from "cn";

import { useI18n, type Key } from "@/i18n";
import {
  useContentTypes,
  useRecent,
  useSite,
  useStatusCounts,
  useTaxonomies,
  useTrend,
} from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { useSession } from "@/hooks/useSession";
import { linkProps, navigate, type Route } from "@/lib/router";

import { IndexProblems } from "@/components/IndexProblems";
import { StatusDot } from "@/components/StatusDot";
import { DeliveryStages } from "@/components/publish/Delivery";
import { Page } from "@/components/shell/Page";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Item, ItemActions, ItemContent, ItemGroup, ItemTitle } from "@/components/ui/item";
import { Progress, ProgressLabel, ProgressValue } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

const TrendChart = lazy(() => import("@/components/dashboard/TrendChart"));

const ranges = [6, 12, 24];

export function DashboardPage() {
  const { t, locale } = useI18n();
  const site = useSite();
  const session = useSession();
  const types = useContentTypes();
  const kindLabel = useKindLabel();

  const kinds = types.data?.items.map((type) => type.kind) ?? ["post", "page"];
  const primary = kinds[0] ?? "post";
  const problems = site.data?.problems?.length ?? 0;

  const hour = new Date().getHours();
  const part =
    hour < 5 ? "night" : hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening";
  const greeting = t(`dashboard.${part}` as Key);
  const name = session.data?.required ? session.data.user : undefined;

  const today = new Intl.DateTimeFormat(locale, {
    month: "long",
    day: "numeric",
    weekday: "long",
  }).format(new Date());

  return (
    <Page
      title={name ? t("dashboard.greeting", { greeting, name }) : greeting}
      description={
        <span className="flex items-center gap-2">
          <span
            aria-hidden
            className={cn("size-1.5 shrink-0 rounded-full", problems ? "bg-warning" : "bg-success")}
          />
          {problems ? t("problems.notIndexed", { count: problems }) : t("dashboard.healthy")}
          <span aria-hidden>·</span>
          {today}
        </span>
      }
      actions={
        <>
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href="/" target="_blank" rel="noreferrer" />}
          >
            <ExternalLink data-icon="inline-start" />
            {t("nav.viewSite")}
          </Button>
          <Button onClick={() => navigate({ name: "edit", kind: primary, id: null })}>
            <Plus data-icon="inline-start" />
            {t("list.newKind", { kind: kindLabel.one(primary) })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <IndexProblems />

        <Card className="py-0">
          <div className="grid grid-cols-2 gap-px bg-border lg:grid-cols-4">
            {kinds.slice(0, 2).map((kind) => (
              <KindStat key={kind} kind={kind} total={site.data?.counts?.[kind]} />
            ))}
            <TermsStat />
            <DeliveryStat />
          </div>
        </Card>

        <div className="grid gap-3.5 lg:grid-cols-[minmax(0,1.6fr)_minmax(0,1fr)]">
          <TrendCard kind={primary} />
          <div className="flex min-w-0 flex-col gap-3.5">
            <QuickActions kinds={kinds.slice(0, 2)} />
            <SystemCard kind={primary} />
          </div>
        </div>

        <div className="grid gap-3.5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.15fr)]">
          <DeliveryCard />
          <RecentCard kind={primary} />
        </div>
      </div>
    </Page>
  );
}

function Stat({ label, value, note }: { label: string; value: ReactNode; note: ReactNode }) {
  return (
    <div className="bg-card px-5 py-4">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-2xl font-semibold tracking-tight tabular-nums">{value}</div>
      <div className="mt-1 truncate text-xs text-muted-foreground">{note}</div>
    </div>
  );
}

function KindStat({ kind, total }: { kind: string; total?: number }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const counts = useStatusCounts(kind, ["published", "draft"]);

  return (
    <Stat
      label={kindLabel.many(kind)}
      value={total ?? counts.all ?? <Skeleton className="h-8 w-10" />}
      note={t("dashboard.statSplit", {
        published: counts.published ?? "–",
        draft: counts.draft ?? "–",
      })}
    />
  );
}

function TermsStat() {
  const { t } = useI18n();
  const taxonomies = useTaxonomies();
  const taxonomyLabel = useTaxonomyLabel();
  const items = taxonomies.data?.items ?? [];

  return (
    <Stat
      label={t("nav.taxonomies")}
      value={
        taxonomies.data ? (
          items.reduce((sum, item) => sum + item.terms, 0)
        ) : (
          <Skeleton className="h-8 w-10" />
        )
      }
      note={
        items.map((item) => `${item.terms} ${taxonomyLabel(item.name)}`).join(" · ") ||
        t("dashboard.noTaxonomies")
      }
    />
  );
}

function DeliveryStat() {
  const { t } = useI18n();
  const delivery = useDelivery();

  if (delivery.data && !canPublish(delivery.data)) {
    return (
      <Stat label={t("dashboard.uncommitted")} value="–" note={t("dashboard.noPublisher")} />
    );
  }
  return (
    <Stat
      label={t("dashboard.uncommitted")}
      value={delivery.data ? (delivery.data.dirty?.length ?? 0) : <Skeleton className="h-8 w-10" />}
      note={
        delivery.data?.ahead
          ? t("publish.toPush", { count: delivery.data.ahead })
          : t("dashboard.nothingToPush")
      }
    />
  );
}

function TrendCard({ kind }: { kind: string }) {
  const { t } = useI18n();
  const [months, setMonths] = useState(12);
  const trend = useTrend(kind, months);
  const empty = trend.data?.every((bucket) => bucket.count === 0);

  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="text-sm">{t("dashboard.trend")}</CardTitle>
        <CardAction>
          <ToggleGroup
            size="sm"
            variant="outline"
            value={[String(months)]}
            onValueChange={(value) => value[0] && setMonths(Number(value[0]))}
          >
            {ranges.map((range) => (
              <ToggleGroupItem key={range} value={String(range)}>
                {t("dashboard.months", { count: range })}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </CardAction>
      </CardHeader>
      {/* The plot takes whatever height the row gives the card. */}
      <CardContent className="relative min-h-48 flex-1">
        <div className="absolute inset-0 px-(--card-spacing)">
          {!trend.data ? (
            <Skeleton className="size-full" />
          ) : empty ? (
            <Empty className="h-full">
              <EmptyHeader>
                <EmptyTitle>{t("dashboard.trendEmpty")}</EmptyTitle>
                <EmptyDescription>{t("dashboard.trendEmptyNote")}</EmptyDescription>
              </EmptyHeader>
            </Empty>
          ) : (
            <Suspense fallback={<Skeleton className="size-full" />}>
              <TrendChart data={trend.data} />
            </Suspense>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function QuickActions({ kinds }: { kinds: string[] }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();

  const actions: { label: string; icon: LucideIcon; route: Route }[] = [
    ...kinds.map((kind, i) => ({
      label: t("list.newKind", { kind: kindLabel.one(kind) }),
      icon: i === 0 ? Plus : PanelsTopLeft,
      route: { name: "edit", kind, id: null } as Route,
    })),
    { label: t("dashboard.manageTheme"), icon: LayoutTemplate, route: { name: "theme" } },
    { label: t("dashboard.siteSettings"), icon: Settings2, route: { name: "settings" } },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{t("dashboard.quickActions")}</CardTitle>
      </CardHeader>
      <CardContent className="grid grid-cols-2 gap-2">
        {actions.map((action) => (
          <Button
            key={action.label}
            variant="outline"
            className="h-auto flex-col items-start gap-2 p-3"
            onClick={() => navigate(action.route)}
          >
            <span className="flex size-7 items-center justify-center rounded-md bg-brand/10 text-brand">
              <action.icon />
            </span>
            <span className="text-xs">{action.label}</span>
          </Button>
        ))}
      </CardContent>
    </Card>
  );
}

function SystemCard({ kind }: { kind: string }) {
  const { t } = useI18n();
  const site = useSite();
  const kindLabel = useKindLabel();
  const counts = useStatusCounts(kind, ["published", "draft"]);

  const rows: [string, string | undefined][] = [
    [t("dashboard.version"), site.data?.version],
    [t("dashboard.store"), site.data?.store],
    [t("dashboard.runtime"), site.data?.runtime],
    [t("nav.theme"), site.data?.theme],
  ];
  const total = site.data?.counts?.[kind] ?? counts.all ?? 0;
  const published = counts.published ?? 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{t("dashboard.system")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5 text-xs">
        {rows.map(([label, value]) => (
          <div key={label} className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">{label}</span>
            <span className="truncate font-mono">{value || "–"}</span>
          </div>
        ))}
        <Progress value={total ? (published / total) * 100 : 0} className="gap-2">
          <ProgressLabel className="text-xs font-normal text-muted-foreground">
            {t("dashboard.publishedShare", { kind: kindLabel.many(kind) })}
          </ProgressLabel>
          <ProgressValue className="text-xs text-foreground">
            {() => `${published} / ${total}`}
          </ProgressValue>
        </Progress>
      </CardContent>
    </Card>
  );
}

function DeliveryCard() {
  const { t } = useI18n();
  const delivery = useDelivery();
  const dirty = delivery.data?.dirty ?? [];

  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="text-sm">{t("publish.delivery")}</CardTitle>
        {delivery.data?.branch && (
          <CardDescription className="font-mono text-xs">
            {delivery.data.branch}
            {delivery.data.remote ? ` → ${delivery.data.remote}` : ""}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <DeliveryStages delivery={delivery.data} />
        {dirty.length > 0 && (
          <ul className="flex flex-col gap-1 border-t pt-3 font-mono text-xs text-muted-foreground">
            {dirty.slice(0, 4).map((path) => (
              <li key={path} className="truncate">
                {path}
              </li>
            ))}
            {dirty.length > 4 && <li>{t("dashboard.more", { count: dirty.length - 4 })}</li>}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

function RecentCard({ kind }: { kind: string }) {
  const { t, relative } = useI18n();
  const recent = useRecent(6);

  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="text-sm">{t("dashboard.recent")}</CardTitle>
        <CardAction>
          <Button
            variant="link"
            size="sm"
            className="h-auto p-0 text-brand"
            nativeButton={false}
            render={<a {...linkProps({ name: "list", kind })} />}
          >
            {t("dashboard.viewAll")}
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="px-1.5">
        {!recent.data ? (
          <div className="flex flex-col gap-2 px-2.5">
            {Array.from({ length: 4 }, (_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : recent.data.items.length === 0 ? (
          <Empty>
            <EmptyHeader>
              <EmptyTitle>{t("dashboard.recentEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          <ItemGroup className="gap-0">
            {recent.data.items.map((item) => (
              <Item
                key={item.id}
                size="xs"
                render={<a {...linkProps({ name: "edit", kind: item.kind, id: item.id })} />}
              >
                <ItemContent className="min-w-0">
                  <ItemTitle className="w-full">
                    <span className="truncate">{item.title || item.slug}</span>
                  </ItemTitle>
                </ItemContent>
                <ItemActions className="gap-4 text-xs text-muted-foreground">
                  <StatusDot status={item.status} className="hidden sm:inline-flex" />
                  <span className="w-24 text-right whitespace-nowrap">
                    {relative(item.updated_at)}
                  </span>
                </ItemActions>
              </Item>
            ))}
          </ItemGroup>
        )}
      </CardContent>
    </Card>
  );
}
