import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type components } from "@/api/client";
import { Panel } from "@/components/ui";
import { cn } from "@/lib/cn";

type DeliveryState = components["schemas"]["DeliveryState"];
type Plan = components["schemas"]["Plan"];
type Problem = components["schemas"]["Problem"];

/**
 * How far the content has actually traveled, and the button that moves it.
 *
 * The four steps are shown separately because they fail separately. A post
 * can be saved but not committed, committed but not pushed, pushed but not
 * yet deployed, and an author who is told only "published" has no way to tell
 * which of those is true.
 */
export function PublishPanel({ ids, onDone }: { ids: string[]; onDone?: () => void }) {
  const queryClient = useQueryClient();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [failure, setFailure] = useState<string | null>(null);

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
    onError: (err: { error?: { message: string }; plan?: Plan }) => {
      // A refusal carries the plan, so every reason is shown at once rather
      // than one per attempt.
      setPlan(err.plan ?? null);
      setFailure(err.error?.message ?? "the publish did not finish");
      void queryClient.invalidateQueries({ queryKey: ["publish"] });
    },
  });

  const delivery = state.data;
  const needsConfirmation = Boolean(plan?.warnings?.length) && !plan?.problems?.length;

  return (
    <Panel className="p-4">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-sm font-semibold">Publish</h2>
        {delivery?.branch && (
          <span className="font-mono text-xs text-[var(--muted-foreground)]">
            {delivery.branch}
            {delivery.remote ? ` → ${delivery.remote}` : ""}
          </span>
        )}
      </div>

      <ol className="mb-4 space-y-1.5">
        <Stage label="Saved" step={delivery?.local} />
        <Stage label="Committed" step={delivery?.committed} note={countNote(delivery?.dirty?.length)} />
        <Stage label="Pushed" step={delivery?.pushed} note={aheadNote(delivery)} />
        <Stage label="Deployed" step={delivery?.deployed} note="your host reports this" />
      </ol>

      {failure && (
        <div className="mb-3 rounded-md bg-red-500/5 px-3 py-2 text-xs text-red-700 dark:text-red-400">
          {failure}
        </div>
      )}

      {plan?.problems?.map((p) => <Note key={p.code} problem={p} tone="stop" />)}
      {plan?.warnings?.map((p) => <Note key={p.code} problem={p} tone="warn" />)}

      <button
        type="button"
        disabled={publish.isPending || ids.length === 0}
        onClick={() => publish.mutate(needsConfirmation)}
        className={cn(
          "w-full rounded-md px-3 py-1.5 text-sm text-white disabled:opacity-40",
          needsConfirmation ? "bg-amber-600" : "bg-brand",
        )}
      >
        {publish.isPending
          ? "Publishing"
          : needsConfirmation
            ? "Publish anyway"
            : "Publish"}
      </button>
    </Panel>
  );
}

const tone: Record<string, string> = {
  done: "bg-emerald-500",
  pending: "bg-[var(--border)]",
  failed: "bg-red-500",
  not_applicable: "bg-[var(--border)] opacity-40",
};

function Stage({ label, step, note }: { label: string; step?: string; note?: string }) {
  return (
    <li className="flex items-center gap-2 text-sm">
      <span className={cn("size-2 shrink-0 rounded-full", tone[step ?? "pending"])} />
      <span className={cn(step === "not_applicable" && "opacity-50")}>{label}</span>
      {note && <span className="text-xs text-[var(--muted-foreground)]">{note}</span>}
    </li>
  );
}

function Note({ problem, tone }: { problem: Problem; tone: "stop" | "warn" }) {
  return (
    <div
      className={cn(
        "mb-2 rounded-md px-3 py-2 text-xs",
        tone === "stop"
          ? "bg-red-500/5 text-red-700 dark:text-red-400"
          : "bg-amber-500/5 text-amber-800 dark:text-amber-400",
      )}
    >
      <div>{problem.detail}</div>
      {problem.fix && <div className="mt-1 opacity-75">{problem.fix}</div>}
    </div>
  );
}

function countNote(dirty?: number): string | undefined {
  if (!dirty) return undefined;
  return `${dirty} file${dirty === 1 ? "" : "s"} uncommitted`;
}

function aheadNote(delivery?: DeliveryState): string | undefined {
  if (!delivery?.ahead) return undefined;
  return `${delivery.ahead} commit${delivery.ahead === 1 ? "" : "s"} to push`;
}
