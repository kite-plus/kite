import { useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  CircleCheck,
  CircleDashed,
  CircleX,
  CloudUpload,
  Download,
  ExternalLink,
  GitBranch,
  GitPullRequestArrow,
  LoaderCircle,
  Upload,
} from "lucide-react";

import { useI18n, useProblem } from "@/i18n";
import { useLatest, useTerms } from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, useDelivery, usePublish, type DeliveryState } from "@/hooks/usePublish";
import { isoDate } from "@/lib/dates";
import { cn } from "@/lib/utils";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { DeliveryProgress, PublishProblems } from "@/components/publish/Delivery";
import { PublishDialog } from "@/components/publish/PublishDialog";
import { QueryError } from "@/components/query-error";
import { TermBadge } from "@/components/TermBadge";

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

/**
 * DeliveryCard says whether the site is up to date: in a sentence first,
 * then step by step, with the one thing to do next at hand.
 */
export function DeliveryCard() {
  const { t, locale } = useI18n();
  const delivery = useDelivery();
  const d = delivery.data;
  const checked = ago(d?.checked_at, locale, t);

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("publish.delivery")}</CardTitle>
        <CardDescription className="flex min-w-0 items-center gap-1.5">
          {d?.branch ? (
            <>
              <GitBranch className="size-3.5 shrink-0" />
              <span className="truncate font-mono text-xs">{`${d.remote ?? "origin"}/${d.branch}`}</span>
              {checked && <span className="shrink-0">· {checked}</span>}
            </>
          ) : (
            t("publish.title")
          )}
        </CardDescription>
        <CardAction>
          <Link to="/deploy" className="text-sm text-muted-foreground hover:text-foreground">
            {t("nav.deploy")}
          </Link>
        </CardAction>
      </CardHeader>
      <CardContent>
        {delivery.error && !d ? (
          <QueryError error={delivery.error} onRetry={() => void delivery.refetch()} />
        ) : delivery.isPending ? (
          <Skeleton className="h-40 w-full" />
        ) : canPublish(d) && d ? (
          <DeliveryOverview delivery={d} />
        ) : (
          // Without git there is nothing to travel through; the way out is an
          // export, which the deploy page makes.
          <div className="grid justify-items-start gap-3 text-sm">
            <p className="text-muted-foreground">{t("deploy.cardNote")}</p>
            <Button variant="outline" size="sm" asChild>
              <Link to="/deploy">
                <Download />
                {t("deploy.export")}
              </Link>
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function DeliveryOverview({ delivery: d }: { delivery: DeliveryState }) {
  const { t } = useI18n();
  const publish = usePublish([]);
  const [publishing, setPublishing] = useState(false);
  const dirty = d.dirty ?? [];

  return (
    <div className="grid gap-5">
      <Headline delivery={d} />
      <DeliveryProgress delivery={d} />
      {dirty.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 text-xs">
          <span className="text-muted-foreground">{t("publish.uncommittedFiles")}</span>
          {dirty.slice(0, shownFiles).map((path) => (
            <code key={path} className="max-w-full truncate rounded bg-muted px-1.5 py-0.5 font-mono">
              {path}
            </code>
          ))}
          {dirty.length > shownFiles && (
            <span className="text-muted-foreground">+{dirty.length - shownFiles}</span>
          )}
        </div>
      )}
      {dirty.length > 0 ? (
        <Button size="sm" className="justify-self-start" onClick={() => setPublishing(true)}>
          <Upload />
          {t("publish.publishAll", { count: dirty.length })}
        </Button>
      ) : d.ahead ? (
        <Button
          size="sm"
          className="justify-self-start"
          disabled={publish.pending}
          onClick={() => void publish.push()}
        >
          {publish.pending ? <Spinner /> : <Upload />}
          {t("publish.pushCommits", { count: d.ahead })}
        </Button>
      ) : d.deployed === "done" && d.deployed_url ? (
        <Button size="sm" variant="outline" className="justify-self-start" asChild>
          <a href={d.deployed_url} target="_blank" rel="noreferrer">
            <ExternalLink />
            {t("nav.viewSite")}
          </a>
        </Button>
      ) : null}
      <PublishProblems publish={publish} />
      <PublishDialog
        ids={[]}
        paths={dirty}
        open={publishing}
        onOpenChange={setPublishing}
        onDone={() => undefined}
        title={t("publish.allTitle")}
        description={t("publish.allNote", { paths: dirty.join(", ") })}
      />
    </div>
  );
}

// How many uncommitted files are named before the rest are counted.
const shownFiles = 3;

const tones = {
  success: { icon: CircleCheck, className: "bg-success/10 text-success" },
  warning: { icon: CloudUpload, className: "bg-warning/10 text-warning" },
  behind: { icon: GitPullRequestArrow, className: "bg-warning/10 text-warning" },
  info: { icon: LoaderCircle, className: "bg-primary/10 text-primary [&_svg]:animate-spin" },
  failed: { icon: CircleX, className: "bg-destructive/10 text-destructive" },
  quiet: { icon: CircleDashed, className: "bg-muted text-muted-foreground" },
} as const;

/**
 * Headline sums the steps up in a sentence: whether the site is up to date,
 * and if not, what is holding it back. What went wrong outranks what is
 * waiting, and what is waiting outranks what is under way.
 */
function Headline({ delivery: d }: { delivery: DeliveryState }) {
  const { t } = useI18n();
  const problem = useProblem();
  const dirty = d.dirty?.length ?? 0;

  let tone: keyof typeof tones;
  let title: string;
  let note: string;
  if (d.last_error) {
    tone = "failed";
    title = t("publish.headline.error");
    note = problem(d.last_error.code, d.last_error.detail).title;
  } else if (d.deployed === "failed") {
    tone = "failed";
    title = t("publish.headline.failed");
    note = t("publish.headline.failedNote");
  } else if (dirty || d.ahead) {
    // The steps below count what waits; this says what it takes.
    tone = "warning";
    title = t("publish.headline.pending");
    note = [
      t(dirty ? "publish.headline.commitAndPush" : "publish.headline.pushOnly"),
      d.behind ? t("publish.headline.behind", { count: d.behind }) : "",
    ]
      .filter(Boolean)
      .join(t("publish.headline.also"));
  } else if (d.behind) {
    tone = "behind";
    title = t("publish.headline.behind", { count: d.behind });
    note = t("publish.headline.behindNote");
  } else if (d.deployed === "pending") {
    tone = "info";
    title = t("publish.headline.deploying");
    note = t("publish.headline.deployingNote");
  } else if (d.deployed === "not_applicable") {
    tone = "quiet";
    title = t("publish.headline.pushed");
    note = t("publish.headline.pushedNote");
  } else {
    tone = "success";
    title = t("publish.headline.live");
    note = t("publish.headline.liveNote");
  }
  const { icon: Icon, className } = tones[tone];

  return (
    <div className="flex items-start gap-3">
      <span className={cn("flex size-9 shrink-0 items-center justify-center rounded-full [&_svg]:size-5", className)}>
        <Icon />
      </span>
      <div className="min-w-0">
        <p className="font-medium">{title}</p>
        <p className="text-sm text-muted-foreground">{note}</p>
      </div>
    </div>
  );
}

/** ago says how long since the remote was last looked at. */
function ago(
  iso: string | undefined,
  locale: string,
  t: ReturnType<typeof useI18n>["t"],
): string | null {
  if (!iso) return null;
  const seconds = (Date.now() - Date.parse(iso)) / 1000;
  if (!Number.isFinite(seconds)) return null;
  if (seconds < 60) return t("publish.checkedNow");
  const minutes = Math.round(seconds / 60);
  const words = new Intl.RelativeTimeFormat(locale, { numeric: "auto" });
  const when = minutes < 60 ? words.format(-minutes, "minute") : words.format(-Math.round(minutes / 60), "hour");
  return t("publish.checked", { when });
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
        <CardAction>
          <Link
            to="/taxonomies/$taxonomy"
            params={{ taxonomy }}
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            {t("dashboard.viewAll")}
          </Link>
        </CardAction>
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
                <TermBadge
                  term={item.term}
                  count={item.count}
                  className="transition-opacity hover:opacity-80"
                />
              </Link>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}
