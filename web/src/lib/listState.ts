import { useSyncExternalStore } from "react";

export interface ListState {
  /** A status, or "all". */
  status: string;
  /** The chosen term of each taxonomy, keyed by taxonomy name. */
  terms: Record<string, string>;
  q: string;
  sort: string;
}

const initial: ListState = { status: "all", terms: {}, q: "", sort: "" };

// Kept outside the tree so a listing is as it was left when the editor closes.
const states = new Map<string, ListState>();
const listeners = new Set<() => void>();

export function setListState(kind: string, patch: Partial<ListState>) {
  states.set(kind, { ...(states.get(kind) ?? initial), ...patch });
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void) {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function useListState(kind: string) {
  const state = useSyncExternalStore(subscribe, () => states.get(kind) ?? initial);
  return [state, (patch: Partial<ListState>) => setListState(kind, patch)] as const;
}
