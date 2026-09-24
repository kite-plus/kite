import { GitMerge, TriangleAlert, Upload, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";

import { useI18n, useProblem } from "@/i18n";
import type { DeliveryState, RemoteChange, usePublish } from "@/hooks/usePublish";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";

type Publish = ReturnType<typeof usePublish>;

const tones: Record<string, string> = {
  done: "bg-success",
  pending: "bg-border",
  failed: "bg-destructive",
  not_applicable: "bg-border opacity-40",
};

/**
 * How far the content has actually travelled.
 *
 * The four steps are shown separately because they fail separately: a post
 * can be saved but not committed, committed but not pushed, pushed but not
 * yet deployed, and "published" alone cannot say which.
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
  return (
    <ol className="flex flex-col gap-2">
      <Stage label={t("publish.saved")} step={delivery?.local} />
      <Stage
        label={t("publish.committed")}
        step={delivery?.committed}
        note={
          delivery?.dirty?.length
            ? t("publish.uncommitted", { count: delivery.dirty.length })
            : undefined
        }
      />
      <Stage
        label={t("publish.pushed")}
        step={delivery?.pushed}
        note={delivery?.ahead ? t("publish.toPush", { count: delivery.ahead }) : undefined}
        action={
          publish && delivery?.ahead ? (
            <button
              type="button"
              className="shrink-0 text-xs text-brand hover:underline disabled:opacity-50"
              disabled={publish.pending}
              onClick={() => void publish.push()}
            >
              {t("publish.push")}
            </button>
          ) : undefined
        }
      />
      <Stage
        label={t("publish.deployed")}
        step={delivery?.deployed}
        note={
          {
            pending: t("publish.deployedNote"),
            not_applicable: t("publish.deployedUnknown"),
            failed: t("publish.deployFailed"),
          }[delivery?.deployed ?? "pending"]
        }
        action={
          delivery?.deployed === "done" && delivery.deployed_url ? (
            <a
              href={delivery.deployed_url}
              target="_blank"
              rel="noreferrer"
              className="ml-auto shrink-0 text-xs text-brand hover:underline"
            >
              {t("publish.viewSite")}
            </a>
          ) : undefined
        }
      />
    </ol>
  );
}

function Stage({
  label,
  step,
  note,
  action,
}: {
  label: string;
  step?: string;
  note?: string;
  action?: React.ReactNode;
}) {
  return (
    <li className="flex items-center gap-2 text-sm">
      <span className={cn("size-1.5 shrink-0 rounded-full", tones[step ?? "pending"])} />
      <span className={cn(step === "not_applicable" && "opacity-50")}>{label}</span>
      {note && <span className="ml-auto truncate text-xs text-muted-foreground">{note}</span>}
      {action}
    </li>
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
