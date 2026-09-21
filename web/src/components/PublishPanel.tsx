import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Upload } from "lucide-react";
import { cn } from "cn";

import { api, type components } from "@/api/client";
import { useI18n, useProblem } from "@/i18n";
import { Alert } from "@/components/Alert";
import { Button } from "@/components/ui/button";

type DeliveryState = components["schemas"]["DeliveryState"];
type Plan = components["schemas"]["Plan"];

/**
 * How far the content has actually travelled, and the button that moves it.
 *
 * The four steps are shown separately because they fail separately. A post
 * can be saved but not committed, committed but not pushed, pushed but not
 * yet deployed, and an author told only "published" has no way to tell which
 * of those is true.
 */
export function PublishPanel({
  ids = [],
  compact = false,
  onDone,
}: {
  ids?: string[];
  compact?: boolean;
  onDone?: () => void;
}) {
  const { t } = useI18n();
  const problem = useProblem();
  const queryClient = useQueryClient();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [failure, setFailure] = useState<{ code?: string; detail?: string } | null>(null);

  const state = useQuery({
    queryKey: ["publish"],
    queryFn: async () => {
      const { data } = await api.GET("/publish", {});
      return data as DeliveryState;
    },
    refetchInterval: 10_000,
  });

  const publish = useMutation({
    mutationFn: async (force: boolean) => {
      const response = await fetch("/api/v1/publish", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ids, push: true, force }),
      });
      const body = await response.json();
      if (!response.ok) throw body;
      return body;
    },
    onSuccess: async () => {
      setPlan(null);
      setFailure(null);
      await queryClient.invalidateQueries({ queryKey: ["publish"] });
      onDone?.();
    },
    onError: (err: { error?: { code?: string; message?: string }; plan?: Plan }) => {
      // A refusal carries the plan, so every reason is shown at once rather
      // than one per attempt.
      setPlan(err.plan ?? null);
      setFailure({ code: err.error?.code, detail: err.error?.message });
      void queryClient.invalidateQueries({ queryKey: ["publish"] });
    },
  });

  const delivery = state.data;
  const needsConfirmation = Boolean(plan?.warnings?.length) && !plan?.problems?.length;

  // In the rail there is room for the state but not for the argument, so the
  // button lives where the work is.
  if (compact) {
    return (
      <div className="px-2">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">
            {t("publish.delivery")}
          </span>
          {delivery?.branch && (
            <span className="truncate font-mono text-[11px] text-muted-foreground">
              {delivery.branch}
            </span>
          )}
        </div>
        <Stages delivery={delivery} />
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium">{t("publish.title")}</span>
        {delivery?.branch && (
          <span className="font-mono text-xs text-muted-foreground">
            {delivery.branch}
            {delivery.remote ? ` → ${delivery.remote}` : ""}
          </span>
        )}
      </div>

      <Stages delivery={delivery} />

      {failure && <Problem tone="stop" {...problem(failure.code, failure.detail)} />}
      {plan?.problems?.map((p) => (
        <Problem key={p.code} tone="stop" {...problem(p.code, p.detail, p.fix)} />
      ))}
      {plan?.warnings?.map((p) => (
        <Problem key={p.code} tone="warn" {...problem(p.code, p.detail, p.fix)} />
      ))}

      <Button
        className="w-full"
        variant={needsConfirmation ? "secondary" : "default"}
        disabled={publish.isPending || ids.length === 0}
        onClick={() => publish.mutate(needsConfirmation)}
      >
        <Upload className="size-4" />
        {publish.isPending
          ? t("publish.working")
          : needsConfirmation
            ? t("publish.anyway")
            : t("publish.action")}
      </Button>
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
    <Alert tone={tone} title={title}>
      {detail && <p>{detail}</p>}
      {fix && <p className="mt-1">{fix}</p>}
    </Alert>
  );
}

const tone: Record<string, string> = {
  done: "bg-emerald-500",
  pending: "bg-border",
  failed: "bg-destructive",
  not_applicable: "bg-border opacity-40",
};

function Stages({ delivery }: { delivery?: DeliveryState }) {
  const { t } = useI18n();
  return (
    <ol className="space-y-1.5">
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
      <span className={cn("size-1.5 shrink-0 rounded-full", tone[step ?? "pending"])} />
      <span className={cn(step === "not_applicable" && "opacity-50")}>{label}</span>
      {note && <span className="truncate text-xs text-muted-foreground">{note}</span>}
    </li>
  );
}
