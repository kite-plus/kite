import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type components } from "@/api/client";

export type DeliveryState = components["schemas"]["DeliveryState"];
export type Plan = components["schemas"]["Plan"];

/** useDelivery reports how far the content has actually travelled. */
export function useDelivery() {
  return useQuery({
    queryKey: ["publish"],
    queryFn: async () => {
      const { data } = await api.GET("/publish", {});
      return data as DeliveryState;
    },
    refetchInterval: 10_000,
  });
}

/** canPublish is false where nothing commits, such as a database store. */
export function canPublish(delivery?: DeliveryState): boolean {
  return Boolean(delivery) && delivery?.committed !== "not_applicable";
}

interface Refusal {
  error?: { code?: string; message?: string };
  plan?: Plan;
}

/**
 * usePublish commits the named items and pushes them.
 *
 * A refusal carries the whole plan, so every reason is shown at once rather
 * than one per attempt. Warnings alone are acknowledged by publishing again.
 */
export function usePublish(
  ids: string[],
  onDone?: () => void,
  /** onRefused hears a refusal where the panel that shows it may be shut. */
  onRefused?: (failure: { code?: string; detail?: string }, confirmable: boolean) => void,
) {
  const queryClient = useQueryClient();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [failure, setFailure] = useState<{ code?: string; detail?: string } | null>(null);

  const needsConfirmation = Boolean(plan?.warnings?.length) && !plan?.problems?.length;

  const mutation = useMutation({
    mutationFn: async ({ ids, force }: { ids: string[]; force: boolean }) => {
      const response = await fetch("/api/v1/publish", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "same-origin",
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
    onError: (refusal: Refusal) => {
      const failure = { code: refusal.error?.code, detail: refusal.error?.message };
      setPlan(refusal.plan ?? null);
      setFailure(failure);
      void queryClient.invalidateQueries({ queryKey: ["publish"] });
      onRefused?.(
        failure,
        Boolean(refusal.plan?.warnings?.length) && !refusal.plan?.problems?.length,
      );
    },
  });

  return {
    plan,
    failure,
    needsConfirmation,
    pending: mutation.isPending,
    /**
     * run resolves to whether the publish went through. An item saved a
     * moment ago has an id this hook was not rendered with, so it is passed.
     */
    run: async (override?: string[]) => {
      try {
        await mutation.mutateAsync({ ids: override ?? ids, force: needsConfirmation });
        return true;
      } catch {
        return false;
      }
    },
    reset: () => {
      setPlan(null);
      setFailure(null);
    },
  };
}
