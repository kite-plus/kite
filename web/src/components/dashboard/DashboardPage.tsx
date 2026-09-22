import type { ReactNode } from "react";
import {
  ChevronRight,
  ExternalLink,
  FileText,
  LayoutTemplate,
  PanelsTopLeft,
  PenLine,
  Settings2,
  type LucideIcon,
} from "lucide-react";

import { cn } from "cn";

import { useI18n, type Key } from "@/i18n";
import {
  STATUSES,
  useContentTypes,
  useRecent,
  useSite,
  useStatusCounts,
  useTaxonomies,
  useTerms,
} from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery } from "@/hooks/usePublish";
import { useSession } from "@/hooks/useSession";
import { setListState } from "@/lib/listState";
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
import { Empty, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Progress, ProgressLabel, ProgressValue } from "@/components/ui/progress";
import { Skeleton } from "@/components/ui/skeleton";

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
            <PenLine data-icon="inline-start" />
            {t("dashboard.write", { kind: kindLabel.one(primary) })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <IndexProblems />

        <Card className="gap-0 py-0">
          <div className="grid grid-cols-[repeat(auto-fit,minmax(10.5rem,1fr))] gap-px bg-border">
            {shown.map((kind) => (
              <KindStat key={kind} kind={kind} total={site.data?.counts?.[kind]} />
            ))}
            <TermsStat />
            <DeliveryStat />
          </div>
        </Card>

        <div className="grid gap-3.5 lg:grid-cols-[minmax(0,1.55fr)_minmax(0,1fr)]">
          <div className="flex min-w-0 flex-col gap-3.5">
            <LatestCard kind={primary} />
            <DeliveryCard className="flex-1" />
          </div>
          <div className="flex min-w-0 flex-col gap-3.5">
            <QuickActions kinds={shown} />
            <SystemCard kind={primary} />
            <PendingCard kind={primary} problems={problems} />
            <TermsCard kind={primary} className="flex-1" />
          </div>
        </div>
      </div>
    </Page>
  );
}

type Tone = "neutral" | "brand" | "warn";

const tones: Record<Tone, string> = {
  neutral: "bg-muted text-muted-foreground",
  brand: "bg-brand/10 text-brand",
  warn: "bg-warning/15 text-warning",
};

/** A small count, colored by how much it should worry anyone. */
function Pill({ tone, children }: { tone: Tone; children: ReactNode }) {
  return (
    <span
      className={cn(
        "inline-flex h-[18px] items-center rounded-full px-2 text-[11px] font-semibold tabular-nums",
        tones[tone],
      )}
    >
      {children}
    </span>
  );
}

const Counting = () => <Skeleton className="h-6 w-10" />;

/** One cell of the strip: a number, with what it is made of underneath. */
function Stat({ label, value, note }: { label: string; value: ReactNode; note: ReactNode }) {
  return (
    <div className="min-w-0 bg-card px-5 py-4">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1.5 text-[26px] leading-none font-semibold tracking-tight tabular-nums">
        {value}
      </div>
      <div className="mt-2 flex items-center gap-1.5 text-[11px] text-muted-foreground">
        {note}
      </div>
    </div>
  );
}

function KindStat({ kind, total }: { kind: string; total?: number }) {
  const { t, locale } = useI18n();
  const kindLabel = useKindLabel();
  const counts = useStatusCounts(kind, STATUSES);

  // Every status that has anything in it, so the parts add up to the total.
  const parts = STATUSES.filter((status) => counts[status]).map(
    (status) => `${counts[status]} ${t(`status.${status}`).toLocaleLowerCase(locale)}`,
  );
  const counted = STATUSES.every((status) => counts[status] !== undefined);

  return (
    <Stat
      label={kindLabel.many(kind)}
      value={total ?? <Counting />}
      note={
        <span className="truncate">
          {!counted ? "–" : parts.length > 0 ? parts.join(" · ") : t("dashboard.recentEmpty")}
        </span>
      }
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
      value={taxonomies.data ? items.reduce((sum, item) => sum + item.terms, 0) : <Counting />}
      note={
        <span className="truncate">
          {items.map((item) => `${item.terms} ${taxonomyLabel(item.name)}`).join(" · ") ||
            t("dashboard.noTaxonomies")}
        </span>
      }
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
        value="–"
        note={<span className="truncate">{t("dashboard.noPublisher")}</span>}
      />
    );
  }
  const ahead = delivery.data?.ahead ?? 0;
  return (
    <Stat
      label={t("dashboard.uncommitted")}
      value={delivery.data ? (delivery.data.dirty?.length ?? 0) : <Counting />}
      note={
        ahead > 0 ? (
          <Pill tone="warn">{t("publish.toPush", { count: ahead })}</Pill>
        ) : (
          <span className="truncate">{t("dashboard.nothingToPush")}</span>
        )
      }
    />
  );
}

/** The newest items of the main kind, numbered as a list rather than tabled. */
function LatestCard({ kind }: { kind: string }) {
  const { t, date } = useI18n();
  const kindLabel = useKindLabel();
  const latest = useRecent(8, { kind });

  return (
    <Card className="min-w-0">
      <CardHeader>
        <CardTitle className="text-sm font-semibold">
          {t("dashboard.latest", { kind: kindLabel.many(kind) })}
        </CardTitle>
        <CardAction className="self-center">
          <Button
            variant="link"
            size="sm"
            className="h-auto p-0 text-xs text-brand"
            nativeButton={false}
            render={<a {...linkProps({ name: "list", kind })} />}
          >
            {t("dashboard.viewAll")}
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent>
        {!latest.data ? (
          <div className="divide-y border-t">
            {Array.from({ length: 8 }, (_, i) => (
              <div key={i} className="flex h-10 items-center">
                <Skeleton className="h-4 w-2/3" />
              </div>
            ))}
          </div>
        ) : latest.data.items.length === 0 ? (
          <Empty className="py-6">
            <EmptyHeader>
              <EmptyTitle>{t("dashboard.recentEmpty")}</EmptyTitle>
            </EmptyHeader>
          </Empty>
        ) : (
          // Fixed columns for the status and the date, so neither drifts
          // with the width of the word beside it.
          <ol className="divide-y border-t">
            {latest.data.items.map((item, i) => (
              <li key={item.id}>
                <a
                  {...linkProps({ name: "edit", kind: item.kind, id: item.id })}
                  className="group grid h-10 grid-cols-[1.625rem_minmax(0,1fr)_6.5rem] items-center gap-2.5 text-sm outline-none sm:grid-cols-[1.625rem_minmax(0,1fr)_5.5rem_6.5rem]"
                >
                  <span className="font-mono text-[11px] text-muted-foreground/50 tabular-nums">
                    {String(i + 1).padStart(2, "0")}
                  </span>
                  <span className="truncate font-medium transition-colors group-hover:text-brand group-focus-visible:text-brand">
                    {item.title || item.slug}
                  </span>
                  <span className="hidden justify-end sm:flex">
                    <StatusDot status={item.status} className="text-xs" />
                  </span>
                  <span className="text-right text-xs text-muted-foreground tabular-nums">
                    {date(item.published_at ?? item.updated_at)}
                  </span>
                </a>
              </li>
            ))}
          </ol>
        )}
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
      <CardContent className="grid grid-cols-2 gap-2">
        {actions.map((action) => (
          <a
            key={action.label}
            {...linkProps(action.route)}
            className="flex h-[42px] min-w-0 items-center gap-2.5 rounded-lg border px-2.5 text-xs font-medium outline-none transition-colors hover:border-foreground/20 hover:bg-muted/60 focus-visible:ring-3 focus-visible:ring-ring/50 dark:bg-input/30"
          >
            <span className="flex size-6.5 shrink-0 items-center justify-center rounded-md bg-brand/10 text-brand">
              <action.icon className="size-3.5" />
            </span>
            <span className="truncate">{action.label}</span>
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
  const counts = useStatusCounts(kind, ["published"]);

  const rows: [string, string | undefined][] = [
    [t("dashboard.version"), site.data?.version],
    [t("dashboard.store"), site.data?.store],
    [t("dashboard.runtime"), site.data?.runtime],
    [t("nav.theme"), site.data?.theme],
  ];
  const total = site.data?.counts?.[kind] ?? 0;
  const published = counts.published ?? 0;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("dashboard.system")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-2.5 text-xs">
        {rows.map(([label, value]) => (
          <div key={label} className="flex items-center justify-between gap-3">
            <span className="shrink-0 text-muted-foreground">{label}</span>
            <span className="truncate">{value || "–"}</span>
          </div>
        ))}
        <Progress
          value={total ? (published / total) * 100 : 0}
          className="mt-0.5 gap-1.5 [&_[data-slot=progress-indicator]]:rounded-full [&_[data-slot=progress-indicator]]:bg-linear-to-r [&_[data-slot=progress-indicator]]:from-brand/70 [&_[data-slot=progress-indicator]]:to-brand [&_[data-slot=progress-track]]:h-1.5"
        >
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

/** What is waiting on somebody: each row opens the listing that holds it. */
function PendingCard({ kind, problems }: { kind: string; problems: number }) {
  const { t } = useI18n();
  const counts = useStatusCounts(kind, ["draft", "scheduled"]);
  const delivery = useDelivery();

  const open = (status: string) => {
    setListState(kind, { status, q: "", terms: {} });
    navigate({ name: "list", kind });
  };

  const rows: { label: string; count?: number; tone: Tone; open?: () => void }[] = [
    {
      label: t("dashboard.pendingDrafts"),
      count: counts.draft,
      tone: "neutral",
      open: () => open("draft"),
    },
    {
      label: t("dashboard.pendingScheduled"),
      count: counts.scheduled,
      tone: "brand",
      open: () => open("scheduled"),
    },
  ];
  if (delivery.data && canPublish(delivery.data)) {
    rows.push({
      label: t("dashboard.pendingUncommitted"),
      count: delivery.data.dirty?.length ?? 0,
      tone: "warn",
    });
  }
  if (problems > 0) {
    rows.push({ label: t("dashboard.pendingProblems"), count: problems, tone: "warn" });
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">{t("dashboard.pending")}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-0.5 px-2">
        {rows.map((row) => {
          const inner = (
            <>
              <span className="min-w-0 flex-1 truncate text-[13px] text-foreground/80">
                {row.label}
              </span>
              <Pill tone={row.count ? row.tone : "neutral"}>{row.count ?? "–"}</Pill>
              {row.open && <ChevronRight className="size-3.5 shrink-0 text-muted-foreground" />}
            </>
          );
          return row.open ? (
            <button
              key={row.label}
              type="button"
              onClick={row.open}
              className="flex h-8 items-center gap-1.5 rounded-md px-2 text-left outline-none transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              {inner}
            </button>
          ) : (
            <div key={row.label} className="flex h-8 items-center gap-1.5 px-2">
              {inner}
            </div>
          );
        })}
      </CardContent>
    </Card>
  );
}

/** The terms most in use, each opening the listing narrowed to it. */
function TermsCard({ kind, className }: { kind: string; className?: string }) {
  const { t } = useI18n();
  const taxonomies = useTaxonomies();
  const types = useContentTypes();
  const taxonomyLabel = useTaxonomyLabel();

  // Tags where the project has them, and whatever comes first where not.
  const names = taxonomies.data?.items.map((item) => item.name) ?? [];
  const name = names.includes("tags") ? "tags" : names[0];
  const terms = useTerms(name);
  const total = taxonomies.data?.items.find((item) => item.name === name)?.terms ?? 0;
  const top = [...(terms.data?.items ?? [])].sort((a, b) => b.count - a.count).slice(0, 12);

  // A term opens the listing of whichever kind carries this taxonomy.
  const owner =
    types.data?.items.find((type) => name && type.taxonomies?.includes(name))?.kind ?? kind;
  const open = (term: string) => {
    if (!name) return;
    setListState(owner, { status: "all", q: "", terms: { [name]: term } });
    navigate({ name: "list", kind: owner });
  };

  return (
    <Card className={cn("min-w-0", className)}>
      <CardHeader>
        <CardTitle className="text-sm font-semibold">
          {t("dashboard.topTerms", { name: name ? taxonomyLabel(name) : t("nav.taxonomies") })}
        </CardTitle>
        {taxonomies.data && name && (
          <CardAction className="self-center text-[11px] text-muted-foreground">
            {t("taxonomies.terms", { count: total })}
          </CardAction>
        )}
      </CardHeader>
      <CardContent className="flex flex-wrap gap-1.5">
        {!taxonomies.data || (name && !terms.data) ? (
          Array.from({ length: 6 }, (_, i) => <Skeleton key={i} className="h-6 w-16 rounded-md" />)
        ) : top.length === 0 ? (
          <span className="text-xs text-muted-foreground">{t("dashboard.noTaxonomies")}</span>
        ) : (
          top.map((item) => (
            <button
              key={item.term}
              type="button"
              onClick={() => open(item.term)}
              className="inline-flex h-6 items-center gap-1 rounded-md bg-muted px-2.5 text-[11.5px] text-foreground/80 outline-none transition-colors hover:bg-brand/10 hover:text-brand focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              {item.term}
              <span className="text-muted-foreground tabular-nums">{item.count}</span>
            </button>
          ))
        )}
      </CardContent>
    </Card>
  );
}
