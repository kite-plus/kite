import { TriangleAlert, XCircle } from "lucide-react";
import { cn } from "cn";

import { useI18n, useProblem } from "@/i18n";
import type { DeliveryState, usePublish } from "@/hooks/usePublish";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";

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
export function DeliveryStages({ delivery }: { delivery?: DeliveryState }) {
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
      />
      <Stage
        label={t("publish.deployed")}
        step={delivery?.deployed}
        note={t("publish.deployedNote")}
      />
    </ol>
  );
}

function Stage({ label, step, note }: { label: string; step?: string; note?: string }) {
  return (
    <li className="flex items-center gap-2 text-sm">
      <span className={cn("size-1.5 shrink-0 rounded-full", tones[step ?? "pending"])} />
      <span className={cn(step === "not_applicable" && "opacity-50")}>{label}</span>
      {note && <span className="ml-auto truncate text-xs text-muted-foreground">{note}</span>}
    </li>
  );
}

/** Everything a refused publish said, problems first. */
export function PublishProblems({ publish }: { publish: ReturnType<typeof usePublish> }) {
  const problem = useProblem();
  const { failure, plan } = publish;
  if (!failure && !plan) return null;

  return (
    <div className="flex flex-col gap-2">
      {failure && <Problem tone="stop" {...problem(failure.code, failure.detail)} />}
      {plan?.problems?.map((p) => (
        <Problem key={p.code} tone="stop" {...problem(p.code, p.detail, p.fix)} />
      ))}
      {plan?.warnings?.map((p) => (
        <Problem key={p.code} tone="warn" {...problem(p.code, p.detail, p.fix)} />
      ))}
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
