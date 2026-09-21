import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Upload } from "lucide-react";
import { cn } from "cn";

import { api, type components } from "@/api/client";
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

  // In the rail there is room for the state but not for the argument, so the
  // button lives where the work is.
  if (compact) {
    return (
      <div className="px-2">
        <div className="mb-2 flex items-center justify-between">
          <span className="text-xs font-medium text-muted-foreground">Delivery</span>
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
        <span className="text-sm font-medium">Publish</span>
        {delivery?.branch && (
          <span className="font-mono text-xs text-muted-foreground">
            {delivery.branch}
            {delivery.remote ? ` → ${delivery.remote}` : ""}
          </span>
        )}
      </div>

      <Stages delivery={delivery} />

      {failure && <Alert tone="stop">{failure}</Alert>}
      {plan?.problems?.map((p) => (
        <Alert key={p.code} tone="stop" title={p.detail}>
          {p.fix}
        </Alert>
      ))}
      {plan?.warnings?.map((p) => (
        <Alert key={p.code} tone="warn" title={p.detail}>
          {p.fix}
        </Alert>
      ))}

      <Button
        className="w-full"
        variant={needsConfirmation ? "secondary" : "default"}
        disabled={publish.isPending || ids.length === 0}
        onClick={() => publish.mutate(needsConfirmation)}
      >
        <Upload className="size-4" />
        {publish.isPending ? "Publishing" : needsConfirmation ? "Publish anyway" : "Publish"}
      </Button>
    </div>
  );
}

const tone: Record<string, string> = {
  done: "bg-emerald-500",
  pending: "bg-border",
  failed: "bg-destructive",
  not_applicable: "bg-border opacity-40",
};

function Stages({ delivery }: { delivery?: DeliveryState }) {
  return (
    <ol className="space-y-1.5">
      <Stage label="Saved" step={delivery?.local} />
      <Stage
        label="Committed"
        step={delivery?.committed}
        note={count(delivery?.dirty?.length, "file", "uncommitted")}
      />
      <Stage
        label="Pushed"
        step={delivery?.pushed}
        note={count(delivery?.ahead, "commit", "to push")}
      />
      <Stage label="Deployed" step={delivery?.deployed} note="your host reports this" />
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

function count(n: number | undefined, noun: string, tail: string): string | undefined {
  if (!n) return undefined;
  return `${n} ${noun}${n === 1 ? "" : "s"} ${tail}`;
}
