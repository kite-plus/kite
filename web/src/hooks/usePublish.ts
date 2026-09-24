import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, type components } from "@/api/client";

export type DeliveryState = components["schemas"]["DeliveryState"];
export type Plan = components["schemas"]["Plan"];
export type Result = components["schemas"]["Result"];
export type RemoteChange = components["schemas"]["RemoteChange"];

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
  done?: Result;
  problem?: components["schemas"]["Problem"];
}

interface Failure {
  code?: string;
  detail?: string;
  fix?: string;
}

async function post(path: string, body: unknown): Promise<Result> {
  let response: Response;
  try {
    response = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify(body),
    });
  } catch (err) {
    // Shaped like a refusal, so a lost connection is told like any other.
    if (err instanceof TypeError) throw { error: { code: "unreachable", message: err.message } };
    throw err;
  }
  const answer = await response.json();
  if (!response.ok) throw answer;
  return answer;
}

/**
 * usePublish commits the named items and pushes them.
 *
 * A refusal carries the whole plan, so every reason is shown at once rather
 * than one per attempt. Warnings alone are acknowledged by publishing again.
 */
export function usePublish(
  ids: string[],
  onDone?: (result: Result) => void,
  /** onRefused hears a refusal where the panel that shows it may be shut. */
  onRefused?: (failure: Failure, confirmable: boolean) => void,
) {
  const queryClient = useQueryClient();
  const [plan, setPlan] = useState<Plan | null>(null);
  const [failure, setFailure] = useState<Failure | null>(null);
  // What did happen when the rest did not: a commit whose push failed.
  const [done, setDone] = useState<Result | null>(null);
  // What was last asked for, so a publish a hook refused can be asked for
  // again without the hooks.
  const last = useRef<string[]>(ids);

  const needsConfirmation = Boolean(plan?.warnings?.length) && !plan?.problems?.length;

  const settled = async (result: Result) => {
    setPlan(null);
    setFailure(null);
    setDone(null);
    await queryClient.invalidateQueries({ queryKey: ["publish"] });
    // A rebase brings in whatever the remote had, other people's posts included.
    if (result.rebased) await queryClient.invalidateQueries({ queryKey: ["contents"] });
    onDone?.(result);
  };

  const refused = (refusal: Refusal, confirmable: boolean) => {
    // The publisher's own account says which step failed and what to do;
    // the envelope only says that something did.
    const failure: Failure = refusal.problem
      ? { code: refusal.problem.code, detail: refusal.problem.detail, fix: refusal.problem.fix }
      : { code: refusal.error?.code, detail: refusal.error?.message };
    setPlan(refusal.plan ?? null);
    setFailure(failure);
    setDone(refusal.done ?? null);
    void queryClient.invalidateQueries({ queryKey: ["publish"] });
    onRefused?.(failure, confirmable);
  };

  const mutation = useMutation({
    mutationFn: ({ ids, force, skipHooks }: { ids: string[]; force: boolean; skipHooks?: boolean }) =>
      post("/api/v1/publish", { ids, push: true, force, skip_hooks: skipHooks }),
    onSuccess: settled,
    onError: (refusal: Refusal) =>
      refused(refusal, Boolean(refusal.plan?.warnings?.length) && !refusal.plan?.problems?.length),
  });

  const pushing = useMutation({
    mutationFn: (rebase: boolean) => post("/api/v1/publish/push", { rebase }),
    onSuccess: settled,
    onError: (refusal: Refusal) => refused(refusal, false),
  });

  return {
    plan,
    failure,
    done,
    needsConfirmation,
    pending: mutation.isPending || pushing.isPending,
    /**
     * run resolves to whether the publish went through. An item saved a
     * moment ago has an id this hook was not rendered with, so it is passed.
     */
    run: async (override?: string[]) => {
      last.current = override ?? ids;
      try {
        await mutation.mutateAsync({ ids: last.current, force: needsConfirmation });
        return true;
      } catch {
        return false;
      }
    },
    /**
     * runWithoutHooks asks again for a publish a repository hook refused,
     * this time without running the hooks. It is only ever offered after the
     * refusal has been shown.
     */
    runWithoutHooks: async () => {
      try {
        await mutation.mutateAsync({ ids: last.current, force: true, skipHooks: true });
        return true;
      } catch {
        return false;
      }
    },
    /**
     * push sends what is already committed. With rebase, the unpushed commit
     * goes on top of a remote that moved on, which the server only does when
     * nothing the remote changed is anything the commit changed.
     */
    push: async (rebase = false) => {
      try {
        await pushing.mutateAsync(rebase);
        return true;
      } catch {
        return false;
      }
    },
    reset: () => {
      setPlan(null);
      setFailure(null);
      setDone(null);
    },
  };
}
