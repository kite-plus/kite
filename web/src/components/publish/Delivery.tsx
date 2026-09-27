import { Check, GitMerge, Minus, TriangleAlert, Upload, X, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

import { useI18n, useProblem, type Key, type Values } from "@/i18n";
import type { DeliveryState, RemoteChange, usePublish } from "@/hooks/usePublish";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

type Publish = ReturnType<typeof usePublish>;
type Translate = (key: Key, values?: Values) => string;

/**
 * How a step of the way stands: done, waiting on work, under way at the
 * host, failed, or not something this project can know.
 */
export type Standing = "done" | "waiting" | "working" | "failed" | "unknown";

export interface Step {
  key: "saved" | "committed" | "pushed" | "deployed";
  standing: Standing;
  /** label names the step as it stands, as "to commit" or "deploying". */
  label: string;
  /** note says what holds it up. */
  note?: string;
  /** brief says it in fewer words, under a step whose label already says the rest. */
  brief?: string;
}

/**
 * steps reads the four steps content takes to the site: saved on disk,
 * committed, pushed, and deployed by the host. They are told apart because
 * they fail apart: a post can be saved but not committed, committed but not
 * pushed, pushed but not yet deployed, and "published" alone cannot say
 * which.
 */
export function steps(delivery: DeliveryState | undefined, t: Translate): Step[] {
  const d = delivery;
  const dirty = d?.dirty?.length ?? 0;
  const pushed = d?.pushed ?? "pending";
  const deployed = d?.deployed ?? "pending";

  const stand = (step: string | undefined): Standing =>
    step === "done" ? "done" : step === "failed" ? "failed" : step === "not_applicable" ? "unknown" : "waiting";

  const deployStanding: Standing =
    deployed === "pending" && pushed === "done" ? "working" : stand(deployed);

  return [
    { key: "saved", standing: stand(d?.local), label: t(stepLabel("saved", stand(d?.local))) },
    {
      key: "committed",
      standing: stand(d?.committed),
      label: t(stepLabel("committed", stand(d?.committed))),
      note: dirty ? t("publish.uncommitted", { count: dirty }) : undefined,
      brief: dirty ? t("publish.files", { count: dirty }) : undefined,
    },
    {
      key: "pushed",
      standing: stand(pushed),
      label: t(stepLabel("pushed", stand(pushed))),
      note: d?.ahead ? t("publish.toPush", { count: d.ahead }) : undefined,
      brief: d?.ahead ? t("publish.commits", { count: d.ahead }) : undefined,
    },
    {
      key: "deployed",
      standing: deployStanding,
      label: t(stepLabel("deployed", deployStanding)),
      note: {
        done: t("publish.live"),
        working: t("publish.deployedNote"),
        waiting: undefined,
        failed: t("publish.deployFailed"),
        unknown: t("publish.deployedUnknown"),
      }[deployStanding],
    },
  ];
}

function stepLabel(step: Step["key"], standing: Standing): Key {
  return `publish.step.${step}.${standing}` as Key;
}

/** StepMark shows how a step stands; current marks the step with work to do. */
export function StepMark({
  standing,
  current,
  className,
}: {
  standing: Standing;
  current?: boolean;
  className?: string;
}) {
  return (
    <span
      aria-hidden
      className={cn(
        "flex shrink-0 items-center justify-center rounded-full [&_svg]:size-[60%]",
        standing === "done" && "bg-success text-white",
        standing === "failed" && "bg-destructive text-white",
        standing === "working" && "border-2 border-primary/30 text-primary",
        standing === "unknown" && "border border-dashed border-muted-foreground/40 text-muted-foreground",
        standing === "waiting" && (current ? "border-2 border-warning bg-warning/10" : "border border-border"),
        className,
      )}
    >
      {standing === "done" ? (
        <Check strokeWidth={3} />
      ) : standing === "failed" ? (
        <X strokeWidth={3} />
      ) : standing === "working" ? (
        <Spinner className="size-[70%]!" />
      ) : standing === "unknown" ? (
        <Minus />
      ) : current ? (
        <span className="size-[35%] rounded-full bg-warning" />
      ) : null}
    </span>
  );
}

/** current is the first step still to be done, the one with work waiting. */
export function currentStep(list: Step[]): number {
  return list.findIndex((step) => step.standing !== "done" && step.standing !== "unknown");
}

/**
 * DeliveryStages is how far the content has actually travelled, as a list
 * for a narrow place: each step as it stands, what holds it up, and the push
 * at hand where commits wait for it.
 */
export function DeliveryStages({
  delivery,
  publish,
}: {
  delivery?: DeliveryState;
  /** publish, when given, lets commits that are waiting be pushed from here. */
  publish?: Publish;
}) {
  const { t } = useI18n();
  const list = steps(delivery, t);
  const current = currentStep(list);

  return (
    <ol className="flex flex-col gap-2.5">
      {list.map((step, i) => (
        <li key={step.key} className="flex items-center gap-2.5 text-sm">
          <StepMark standing={step.standing} current={i === current} className="size-4" />
          <span className={cn(i > current && current >= 0 && "text-muted-foreground")}>{step.label}</span>
          {step.note && <span className="ml-auto truncate text-xs text-muted-foreground">{step.note}</span>}
          {step.key === "pushed" && publish && delivery?.ahead ? (
            <button
              type="button"
              className={cn("shrink-0 text-xs text-brand hover:underline disabled:opacity-50", !step.note && "ml-auto")}
              disabled={publish.pending}
              onClick={() => void publish.push()}
            >
              {t("publish.push")}
            </button>
          ) : null}
          {step.key === "deployed" && step.standing === "done" && delivery?.deployed_url ? (
            <a
              href={delivery.deployed_url}
              target="_blank"
              rel="noreferrer"
              className="shrink-0 text-xs text-brand hover:underline"
            >
              {t("publish.viewSite")}
            </a>
          ) : null}
        </li>
      ))}
    </ol>
  );
}

/**
 * DeliveryProgress is the same four steps drawn across, for a place with
 * room: each step's mark on one line, the stretch between two filled once
 * the later is done, and under each what holds it up.
 */
export function DeliveryProgress({ delivery }: { delivery?: DeliveryState }) {
  const { t } = useI18n();
  const list = steps(delivery, t);
  const current = currentStep(list);

  return (
    <ol className="grid grid-cols-4">
      {list.map((step, i) => (
        <li key={step.key} className="relative flex min-w-0 flex-col items-center gap-1.5 text-center">
          {i < list.length - 1 && (
            <span
              aria-hidden
              className={cn(
                "absolute top-3 right-[calc(-50%+18px)] left-[calc(50%+18px)] h-0.5 rounded-full",
                list[i + 1].standing === "done" ? "bg-success" : "bg-border",
              )}
            />
          )}
          <StepMark standing={step.standing} current={i === current} className="size-6" />
          <span
            className={cn(
              "text-sm",
              i === current && "font-medium",
              current >= 0 && i > current && "text-muted-foreground",
            )}
          >
            {step.label}
          </span>
          <span className="min-h-4 max-w-full px-1 text-xs text-muted-foreground">{step.brief ?? step.note}</span>
        </li>
      ))}
    </ol>
  );
}

/** Everything a refused publish said, problems first. */
export function PublishProblems({ publish }: { publish: Publish }) {
  const { t } = useI18n();
  const problem = useProblem();
  const { failure, plan, done } = publish;
  if (!failure && !plan) return null;

  const remote = done?.remote;
  // A commit was made and its push failed for some reason other than the
  // remote moving on, such as the network, so trying again may be all it takes.
  const retry = done?.commit && !done.pushed && !remote;

  return (
    <div className="flex flex-col gap-2">
      {failure && <Problem tone="stop" {...problem(failure.code, failure.detail, failure.fix)} />}
      {failure?.code === "hook_refused" && (
        <Button
          variant="outline"
          size="sm"
          className="self-start"
          disabled={publish.pending}
          onClick={() => void publish.runWithoutHooks()}
        >
          {publish.pending ? <Spinner /> : <Upload />}
          {t("publish.skipHooks")}
        </Button>
      )}
      {remote && <RemoteMoved remote={remote} publish={publish} />}
      {retry && (
        <Button
          variant="outline"
          size="sm"
          className="self-start"
          disabled={publish.pending}
          onClick={() => void publish.push()}
        >
          {publish.pending ? <Spinner /> : <Upload />}
          {t("publish.pushAgain")}
        </Button>
      )}
      {plan?.problems?.map((p) => (
        <Problem key={p.code} tone="stop" {...problem(p.code, p.detail, p.fix)} />
      ))}
      {plan?.warnings?.map((p) => (
        <Problem key={p.code} tone="warn" {...problem(p.code, p.detail, p.fix)} />
      ))}
    </div>
  );
}

// How many of a remote's new commits are named before the rest are counted.
const shownCommits = 3;

/**
 * What a remote that moved on has, and what can be done about it.
 *
 * Nothing is merged for the author. Where the remote changed none of what
 * was published, the commit can go on top of the remote's, and that is
 * offered; where it changed the same files, the remote's side is shown, and
 * what to keep is left to the person who knows.
 */
function RemoteMoved({ remote, publish }: { remote: RemoteChange; publish: Publish }) {
  const { t } = useI18n();
  const problem = useProblem();
  const hidden = remote.behind - Math.min(remote.commits.length, shownCommits);

  return (
    <div className="flex flex-col gap-2.5 rounded-lg border border-border px-3.5 py-3 text-[13px]">
      <p className="font-medium">
        {t("publish.remote.title", { count: remote.behind, upstream: remote.upstream })}
      </p>
      <ul className="flex flex-col gap-1">
        {remote.commits.slice(0, shownCommits).map((commit) => (
          <li key={commit.hash} className="flex min-w-0 items-baseline gap-2">
            <code className="shrink-0 font-mono text-xs text-muted-foreground">{commit.hash.slice(0, 7)}</code>
            <span className="truncate">{commit.subject}</span>
            <span className="ml-auto shrink-0 text-xs text-muted-foreground">{commit.author}</span>
          </li>
        ))}
        {hidden > 0 && (
          <li className="text-xs text-muted-foreground">
            {t("publish.remote.more", { count: hidden })}
          </li>
        )}
      </ul>

      {remote.rebase ? (
        <>
          <p className="text-muted-foreground">{t("publish.remote.rebaseNote")}</p>
          <Button
            size="sm"
            className="self-start"
            disabled={publish.pending}
            onClick={() => void publish.push(true)}
          >
            {publish.pending ? (
              <Spinner />
            ) : (
              <GitMerge />
            )}
            {t("publish.remote.rebase")}
          </Button>
        </>
      ) : (
        remote.blocked && (
          <Problem
            tone="stop"
            {...problem(remote.blocked.code, remote.blocked.detail, remote.blocked.fix)}
          />
        )
      )}

      {remote.diff && (
        <details className="group">
          <summary className="cursor-pointer text-xs text-muted-foreground select-none">
            {t("publish.remote.diff")}
          </summary>
          <pre className="mt-2 max-h-72 overflow-auto rounded-md bg-muted p-2.5 font-mono text-[11.5px] leading-[1.5] whitespace-pre">
            {remote.diff}
          </pre>
        </details>
      )}
    </div>
  );
}

function Problem({
  tone,
  title,
  detail,
  fix,
}: {
  tone: "stop" | "warn";
  title: string;
  detail?: string;
  fix?: string;
}) {
  return (
    <Alert variant={tone === "stop" ? "destructive" : "default"}>
      {tone === "stop" ? <XCircle /> : <TriangleAlert />}
      <AlertTitle>{title}</AlertTitle>
      {(detail || fix) && (
        <AlertDescription>
          {detail && <p>{detail}</p>}
          {fix && <p>{fix}</p>}
        </AlertDescription>
      )}
    </Alert>
  );
}
