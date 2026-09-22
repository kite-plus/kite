import type { ReactNode } from "react";
import { cn } from "cn";

import { useI18n, type Key } from "@/i18n";
import {
  STATUSES,
  useContentTypes,
  useLatest,
  useSite,
  useStatusCounts,
  useTaxonomies,
  useTerms,
} from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { useSession } from "@/hooks/useSession";
import { isoDate, longDay } from "@/lib/dates";
import { setListState } from "@/lib/listState";
import { linkProps, navigate, type Route } from "@/lib/router";

import { IndexProblems } from "@/components/IndexProblems";
import { Soon } from "@/components/Soon";
import {
  IconChevronRight,
  IconComments,
  IconExternal,
  IconPages,
  IconPlus,
  IconTheme,
  IconUpload,
} from "@/components/icons";
import { Page } from "@/components/shell/Page";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

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

  return (
    <Page
      hero
      flush
      title={name ? t("dashboard.greeting", { greeting, name }) : greeting}
      description={
        <span className="flex items-center gap-[7px]">
          <span
            aria-hidden
            className={cn("size-[7px] shrink-0 rounded-full", problems ? "bg-warning" : "bg-success")}
          />
          {problems ? t("problems.notIndexed", { count: problems }) : t("dashboard.healthy")}
          {" · "}
          {longDay(locale)}
        </span>
      }
      actions={
        <>
          <Button
            variant="outline"
            nativeButton={false}
            render={<a href="/" target="_blank" rel="noreferrer" />}
          >
            <IconExternal className="size-3.5" />
            {t("nav.viewSite")}
          </Button>
          <Button onClick={() => navigate({ name: "edit", kind: primary, id: null })}>
            <IconPlus className="size-[15px]" strokeWidth={2} />
            {t("dashboard.write", { kind: kindLabel.one(primary) })}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <IndexProblems />

        <div className="grid grid-cols-[repeat(auto-fit,minmax(170px,1fr))] gap-px overflow-hidden rounded-xl border bg-divider shadow-[0_1px_2px_rgb(0_0_0/0.03)]">
          <KindStat kind={primary} total={site.data?.counts?.[primary]} />
          <Stat label={t("dashboard.visits")} />
          <Stat label={t("dashboard.comments")} />
          <Stat label={t("dashboard.rss")} />
        </div>

        <div className="grid gap-3.5 lg:grid-cols-[minmax(280px,1.55fr)_minmax(240px,1fr)]">
          <div className="flex min-w-0 flex-col gap-3.5">
            <LatestCard kind={primary} />
            <CommentsCard className="flex-1" />
          </div>
          <div className="flex min-w-0 flex-col gap-3.5">
            <QuickActions kinds={kinds} />
            <SystemCard />
            <PendingCard kind={primary} />
            <TermsCard kind={primary} className="flex-1" />
          </div>
        </div>
      </div>
    </Page>
  );
}

function Panel({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <section
      className={cn(
        "min-w-0 rounded-xl border bg-card p-[18px] shadow-[0_1px_2px_rgb(0_0_0/0.03)]",
        className,
      )}
    >
      {children}
    </section>
  );
}

function PanelHead({ title, aside }: { title: string; aside?: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <h2 className="text-[13.5px] font-semibold">{title}</h2>
      {aside}
    </div>
  );
}

/** One cell of the strip; with no value it is a figure Kite cannot know yet. */
function Stat({ label, value, note }: { label: string; value?: ReactNode; note?: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="min-w-0 bg-card px-5 py-[17px]">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div
        className={cn(
          "mt-1.5 text-[26px] font-[650] tracking-[-0.5px] tabular-nums",
          value === undefined && "text-subtle",
        )}
      >
        {value ?? "—"}
      </div>
      {note !== undefined ? (
        <div className="mt-2 truncate text-[11px] text-subtle">{note}</div>
      ) : (
        // Where the design shows a trend, in the same pill, uncoloured.
        <div className="mt-2 flex">
          <span className="rounded-full bg-muted px-2 py-0.5 text-[10.5px] font-semibold whitespace-nowrap text-subtle">
            {t("common.soon")}
          </span>
        </div>
      )}
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
      value={total ?? <Skeleton className="my-1.5 h-6 w-10" />}
      note={!counted ? "–" : parts.join(" · ") || "0"}
    />
  );
}

/** The newest published items of the main kind, numbered as a list. */
function LatestCard({ kind }: { kind: string }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const types = useContentTypes();
  const sortable = types.data?.items.find((type) => type.kind === kind)?.sortable ?? [];
  const latest = useLatest(
    kind,
    5,
    sortable.includes("published_at") ? "-published_at" : "-updated_at",
  );

  return (
    <Panel>
      <PanelHead
        title={t("dashboard.latest", { kind: kindLabel.many(kind) })}
        aside={
          <a {...linkProps({ name: "list", kind })} className="text-xs text-brand hover:underline">
            {t("dashboard.viewAll")}
          </a>
        }
      />
      <div className="mt-1.5">
        {!latest.data ? (
          Array.from({ length: 5 }, (_, i) => (
            <div key={i} className="flex h-10 items-center border-t border-muted">
              <Skeleton className="h-4 w-2/3" />
            </div>
          ))
        ) : latest.data.items.length === 0 ? (
          <div className="border-t border-muted py-6 text-center text-xs text-subtle">
            {t("dashboard.latestEmpty", { kind: kindLabel.many(kind) })}
          </div>
        ) : (
          latest.data.items.map((item, i) => (
            <a
              key={item.id}
              {...linkProps({ name: "edit", kind: item.kind, id: item.id })}
              className="group grid grid-cols-[26px_minmax(0,1fr)_80px] items-center gap-2.5 border-t border-muted py-[10.5px] outline-none sm:grid-cols-[26px_minmax(120px,1fr)_92px_80px]"
            >
              <span className="font-mono text-[11px] text-faint">
                {String(i + 1).padStart(2, "0")}
              </span>
              <span className="truncate text-[13px] font-medium transition-colors group-hover:text-brand group-focus-visible:text-brand">
                {item.title || item.slug}
              </span>
              {/* Visits are not counted yet. */}
              <span className="hidden text-right text-xs whitespace-nowrap text-muted-foreground sm:block">
                —
              </span>
              <span className="text-right text-xs whitespace-nowrap text-subtle tabular-nums">
                {isoDate(item.published_at ?? item.updated_at)}
              </span>
            </a>
          ))
        )}
      </div>
    </Panel>
  );
}

function CommentsCard({ className }: { className?: string }) {
  const { t } = useI18n();
  return (
    <Panel className={cn("flex flex-col", className)}>
      <PanelHead
        title={t("dashboard.recentComments")}
        aside={
          <Soon>
            <span tabIndex={0} className="cursor-default text-xs text-subtle outline-none">
              {t("dashboard.viewAll")}
            </span>
          </Soon>
        }
      />
      {/* As tall as the design's four comments, so the columns end together. */}
      <div className="mt-1.5 flex min-h-40 flex-1 flex-col items-center justify-center gap-2 border-t border-muted lg:min-h-[312px]">
        <IconComments className="size-5 text-faint" />
        <span className="text-[12.5px] text-subtle">{t("dashboard.commentsSoon")}</span>
      </div>
    </Panel>
  );
}

type Icon = (props: { className?: string }) => ReactNode;

function QuickActions({ kinds }: { kinds: string[] }) {
  const { t } = useI18n();
  const kindLabel = useKindLabel();
  const [first, second] = kinds;

  const actions: { key: string; label: string; icon: Icon; route?: Route }[] = [
    {
      key: "first",
      label: t("list.newKind", { kind: kindLabel.one(first ?? "post") }),
      icon: IconPlus,
      route: { name: "edit", kind: first ?? "post", id: null },
    },
  ];
  if (second) {
    actions.push({
      key: "second",
      label: t("list.newKind", { kind: kindLabel.one(second) }),
      icon: IconPages,
      route: { name: "edit", kind: second, id: null },
    });
  }
  actions.push(
    { key: "upload", label: t("dashboard.upload"), icon: IconUpload },
    { key: "theme", label: t("dashboard.manageTheme"), icon: IconTheme, route: { name: "theme" } },
  );

  const tile =
    "flex h-[42px] min-w-0 items-center gap-[9px] rounded-[9px] border bg-card px-[11px] outline-none transition-colors focus-visible:ring-3 focus-visible:ring-ring/50";

  return (
    <Panel>
      <PanelHead title={t("dashboard.quickActions")} />
      <div className="mt-3 grid grid-cols-2 gap-2">
        {actions.map((action) =>
          action.route ? (
            <a
              key={action.key}
              {...linkProps(action.route)}
              className={cn(tile, "hover:border-border-strong hover:bg-hover")}
            >
              <span className="flex size-[26px] shrink-0 items-center justify-center rounded-[7px] bg-brand-soft text-brand">
                <action.icon className="size-3.5" />
              </span>
              <span className="truncate text-xs font-medium">{action.label}</span>
            </a>
          ) : (
            <Soon key={action.key}>
              <div tabIndex={0} aria-disabled className={cn(tile, "cursor-default")}>
                <span className="flex size-[26px] shrink-0 items-center justify-center rounded-[7px] bg-muted text-subtle">
                  <action.icon className="size-3.5" />
                </span>
                <span className="truncate text-xs font-medium text-subtle">{action.label}</span>
              </div>
            </Soon>
          ),
        )}
      </div>
    </Panel>
  );
}

function SystemRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2.5 text-xs">
      <span className="whitespace-nowrap text-muted-foreground">{label}</span>
      {children}
    </div>
  );
}

function SystemCard() {
  const { t } = useI18n();
  const site = useSite();
  const unknown = (
    <Soon>
      <span tabIndex={0} className="cursor-default text-subtle outline-none">
        —
      </span>
    </Soon>
  );

  return (
    <Panel>
      <PanelHead title={t("dashboard.system")} />
      <div className="mt-3 flex flex-col gap-[9px]">
        <SystemRow label={t("dashboard.version")}>
          <span className="truncate">{site.data?.version || "—"}</span>
        </SystemRow>
        <SystemRow label={t("dashboard.lastBackup")}>{unknown}</SystemRow>
        <div>
          <SystemRow label={t("dashboard.storage")}>{unknown}</SystemRow>
          <div className="mt-[7px] h-[5px] rounded-full bg-muted" />
        </div>
      </div>
    </Panel>
  );
}

/** A small count, set in a pill. */
function Pill({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span
      className={cn(
        "rounded-full bg-muted px-2 py-px text-[11px] font-semibold text-foreground-3 tabular-nums",
        className,
      )}
    >
      {children}
    </span>
  );
}

/** What is waiting on somebody; a row with a count opens the listing that holds it. */
function PendingCard({ kind }: { kind: string }) {
  const { t } = useI18n();
  const counts = useStatusCounts(kind, ["draft"]);

  const row = "-mx-2 flex items-center justify-between gap-2.5 rounded-[7px] px-2 py-[7px]";
  const soon = (label: string) => (
    <Soon>
      <div tabIndex={0} aria-disabled className={cn(row, "cursor-default outline-none")}>
        <span className="truncate text-[12.5px] text-subtle">{label}</span>
        <Pill className="font-normal text-subtle">—</Pill>
      </div>
    </Soon>
  );

  return (
    <Panel>
      <PanelHead title={t("dashboard.pending")} />
      <div className="mt-2 flex flex-col gap-0.5">
        {soon(t("dashboard.pendingComments"))}
        <button
          type="button"
          onClick={() => {
            setListState(kind, { status: "draft", q: "", terms: {} });
            navigate({ name: "list", kind });
          }}
          className={cn(
            row,
            "text-left outline-none transition-colors hover:bg-hover focus-visible:ring-3 focus-visible:ring-ring/50",
          )}
        >
          <span className="truncate text-[12.5px] text-foreground-2">
            {t("dashboard.pendingDrafts")}
          </span>
          <span className="flex shrink-0 items-center gap-[5px]">
            <Pill>{counts.draft ?? "–"}</Pill>
            <IconChevronRight className="size-[13px] text-faint" />
          </span>
        </button>
        {soon(t("dashboard.pendingPlugins"))}
      </div>
    </Panel>
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
  const top = [...(terms.data?.items ?? [])].sort((a, b) => b.count - a.count).slice(0, 8);

  // A term opens the listing of whichever kind carries this taxonomy.
  const owner =
    types.data?.items.find((type) => name && type.taxonomies?.includes(name))?.kind ?? kind;
  const open = (term: string) => {
    if (!name) return;
    setListState(owner, { status: "all", q: "", terms: { [name]: term } });
    navigate({ name: "list", kind: owner });
  };

  return (
    <Panel className={className}>
      <PanelHead
        title={t("dashboard.topTerms", { name: name ? taxonomyLabel(name) : t("nav.taxonomies") })}
        aside={
          taxonomies.data &&
          name && (
            <span className="text-[11px] text-subtle">
              {t("dashboard.termCount", { count: total })}
            </span>
          )
        }
      />
      <div className="mt-3 flex flex-wrap gap-[7px]">
        {!taxonomies.data || (name && !terms.data) ? (
          Array.from({ length: 6 }, (_, i) => <Skeleton key={i} className="h-[23px] w-16 rounded-[6px]" />)
        ) : top.length === 0 ? (
          <span className="text-xs text-subtle">{t("dashboard.noTaxonomies")}</span>
        ) : (
          top.map((item) => (
            <button
              key={item.term}
              type="button"
              onClick={() => open(item.term)}
              className="rounded-[6px] bg-muted px-2.5 py-[3.5px] text-[11.5px] whitespace-nowrap text-foreground-2 outline-none transition-colors hover:bg-brand-soft hover:text-brand focus-visible:ring-3 focus-visible:ring-ring/50"
            >
              {item.term} <span className="text-subtle">{item.count}</span>
            </button>
          ))
        )}
      </div>
    </Panel>
  );
}
