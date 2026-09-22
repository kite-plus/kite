import { Suspense, lazy, useState, type ReactNode } from "react";
import {
  ChevronRight,
  ExternalLink,
  FileText,
  GitBranch,
  LayoutTemplate,
  PanelsTopLeft,
  Plus,
  Settings2,
  Tags,
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
import { Progress, ProgressLabel, ProgressValue } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";

const TrendChart = lazy(() => import("@/components/dashboard/TrendChart"));

const ranges = [6, 12, 24];

const kindIcons: Record<string, LucideIcon> = {
  post: FileText,
  page: PanelsTopLeft,
};

export function DashboardPage() {
  const { t, locale } = useI18n();
  const site = useSite();
  const session = useSession();
  const types = useContentTypes();
  const kindLabel = useKindLabel();

  const kinds = types.data?.items.map((type) => type.kind) ?? ["post", "page"];
  const shown = kinds.slice(0, 2);
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

        {/* One four column grid for every row, so the gutters of one row
            fall on those of the next whatever the width. */}
        <div className="grid gap-3.5 sm:grid-cols-2 xl:grid-cols-4">
          {shown.map((kind) => (
            <KindStat key={kind} kind={kind} total={site.data?.counts?.[kind]} />
          ))}
          <TermsStat />
          <DeliveryStat />

          <TrendCard kind={primary} className="sm:col-span-2 xl:col-span-3 xl:row-span-2" />
          <QuickActions kinds={shown} />
          <SystemCard kind={primary} />

          <DeliveryCard className="sm:col-span-2 xl:col-span-1" />
          <RecentCard kind={primary} className="sm:col-span-2 xl:col-span-3" />
        </div>
      </div>
    </Page>
  );
}

/** One number, and where to go to see what it counts. */
function Stat({
  label,
  icon: Icon,
  value,
  note,
  route,
}: {
  label: string;
  icon: LucideIcon;
  value: ReactNode;
  note: ReactNode;
  route?: Route;
}) {
  return (
    <Card className={cn("relative", route && "transition-colors hover:bg-muted/40")}>
      <CardContent className="flex flex-col gap-2.5">
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span className="truncate">{label}</span>
          <Icon className="size-3.5 shrink-0" />
        </div>
        <div className="text-2xl leading-none font-semibold tracking-tight tabular-nums">
          {value}
        </div>
        <div className="truncate text-xs text-muted-foreground">{note}</div>
      </CardContent>
      {route && (
        <a
          {...linkProps(route)}
          aria-label={label}
          className="absolute inset-0 rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/50"
        />
      )}
    </Card>
  );
}

const Counting = () => <Skeleton className="h-6 w-10" />;

function KindStat({ kind, total }: { kind: string; total?: number }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const counts = useStatusCounts(kind, ["published", "draft"]);

  return (
    <Stat
      label={kindLabel.many(kind)}
      icon={kindIcons[kind] ?? FileText}
      value={total ?? counts.all ?? <Counting />}
      note={t("dashboard.statSplit", {
        published: counts.published ?? "–",
        draft: counts.draft ?? "–",
      })}
      route={{ name: "list", kind }}
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
      icon={Tags}
      value={taxonomies.data ? items.reduce((sum, item) => sum + item.terms, 0) : <Counting />}
      note={
        items.map((item) => `${item.terms} ${taxonomyLabel(item.name)}`).join(" · ") ||
        t("dashboard.noTaxonomies")
      }
      route={{ name: "taxonomies" }}
    />
  );
}

function DeliveryStat() {
  const { t } = useI18n();
  const delivery = useDelivery();

  if (delivery.data && !canPublish(delivery.data)) {
    return (
      <Stat
        label={t("dashboard.uncommitted")}
        icon={GitBranch}
        value="–"
        note={t("dashboard.noPublisher")}
      />
    );
  }
  return (
    <Stat
      label={t("dashboard.uncommitted")}
      icon={GitBranch}
      value={delivery.data ? (delivery.data.dirty?.length ?? 0) : <Counting />}
      note={
        delivery.data?.ahead
          ? t("publish.toPush", { count: delivery.data.ahead })
          : t("dashboard.nothingToPush")
      }
    />
  );
}

function TrendCard({ kind, className }: { kind: string; className?: string }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const [months, setMonths] = useState(12);
  const trend = useTrend(kind, months);
  const empty = trend.data?.every((bucket) => bucket.count === 0);

  return (
    <Card className={cn("min-w-0", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("dashboard.trend")}</CardTitle>
        <CardDescription className="text-xs">
          {t("dashboard.trendNote", { kind: kindLabel.many(kind) })}
        </CardDescription>
        <CardAction className="self-center">
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
      <CardContent className="relative min-h-56 flex-1">
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
    ...kinds.map((kind) => ({
      label: t("list.newKind", { kind: kindLabel.one(kind) }),
      icon: kindIcons[kind] ?? FileText,
      route: { name: "edit", kind, id: null } as Route,
    })),
    { label: t("dashboard.manageTheme"), icon: LayoutTemplate, route: { name: "theme" } },
    { label: t("dashboard.siteSettings"), icon: Settings2, route: { name: "settings" } },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("dashboard.quickActions")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-0.5 px-2">
        {actions.map((action) => (
          <a
            key={action.label}
            {...linkProps(action.route)}
            className="flex h-9 items-center gap-3 rounded-lg px-2 text-sm outline-none transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
          >
            <span className="flex size-7 shrink-0 items-center justify-center rounded-md bg-brand/10 text-brand">
              <action.icon className="size-4" />
            </span>
            <span className="min-w-0 flex-1 truncate">{action.label}</span>
            <ChevronRight className="size-4 shrink-0 text-muted-foreground" />
          </a>
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
        <CardTitle className="text-sm font-semibold">{t("dashboard.system")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <dl className="divide-y text-xs">
          {rows.map(([label, value]) => (
            <div key={label} className="flex h-8 items-center justify-between gap-3">
              <dt className="shrink-0 text-muted-foreground">{label}</dt>
              <dd className="truncate font-mono">{value || "–"}</dd>
            </div>
          ))}
        </dl>
        <Progress value={total ? (published / total) * 100 : 0} className="gap-2">
          <ProgressLabel className="text-xs font-normal text-muted-foreground">
            {t("dashboard.publishedShare", { kind: kindLabel.many(kind) })}
          </ProgressLabel>
          <ProgressValue className="text-xs text-foreground tabular-nums">
            {() => `${published} / ${total}`}
          </ProgressValue>
        </Progress>
      </CardContent>
    </Card>
  );
}

function DeliveryCard({ className }: { className?: string }) {
  const { t } = useI18n();
  const delivery = useDelivery();
  const dirty = delivery.data?.dirty ?? [];
  const publisher = !delivery.data || canPublish(delivery.data);

  return (
    <Card className={cn("min-w-0", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("publish.delivery")}</CardTitle>
        {delivery.data?.branch && (
          <CardDescription className="truncate font-mono text-xs">
            {delivery.data.branch}
            {delivery.data.remote ? ` → ${delivery.data.remote}` : ""}
          </CardDescription>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {publisher ? (
          <DeliveryStages delivery={delivery.data} />
        ) : (
          <Empty className="py-6">
            <EmptyHeader>
              <EmptyTitle>{t("dashboard.noPublisher")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        )}
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

function RecentCard({ kind, className }: { kind: string; className?: string }) {
  const { t, relative } = useI18n();
  const recent = useRecent(6);

  return (
    <Card className={cn("min-w-0", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("dashboard.recent")}</CardTitle>
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
      <CardContent>
        {!recent.data ? (
          <div className="flex flex-col divide-y">
            {Array.from({ length: 4 }, (_, i) => (
              <div key={i} className="flex h-10 items-center">
                <Skeleton className="h-4 w-2/3" />
              </div>
            ))}
          </div>
        ) : recent.data.items.length === 0 ? (
          <Empty className="py-6">
            <EmptyHeader>
              <EmptyTitle>{t("dashboard.recentEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          // Fixed columns for the status and the time, so neither drifts
          // with the width of the word beside it.
          <ul className="divide-y">
            {recent.data.items.map((item) => (
              <li key={item.id}>
                <a
                  {...linkProps({ name: "edit", kind: item.kind, id: item.id })}
                  className="grid h-10 grid-cols-[minmax(0,1fr)_6.5rem] items-center gap-3 text-sm outline-none transition-colors hover:text-brand focus-visible:text-brand sm:grid-cols-[minmax(0,1fr)_5.5rem_6.5rem]"
                >
                  <span className="truncate font-medium">{item.title || item.slug}</span>
                  <StatusDot status={item.status} className="hidden text-xs sm:inline-flex" />
                  <span className="truncate text-right text-xs text-muted-foreground tabular-nums">
                    {relative(item.updated_at)}
                  </span>
                </a>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
