import type { AppKind } from "@/hooks/useApps";

/**
 * AppsSearch is what the app center shows as its address says it, so a
 * search survives a reload and a link can open one package.
 */
export interface AppsSearch {
  /** The tab: themes, or plugins. */
  kind?: AppKind;
  q?: string;
  /** Official packages only, or community ones only. */
  source?: "official" | "community";
  /** The package whose details are open. */
  id?: string;
}

/** validateAppsSearch keeps what it understands and drops the rest. */
export function validateAppsSearch(search: Record<string, unknown>): AppsSearch {
  const out: AppsSearch = {};
  if (search.kind === "theme" || search.kind === "plugin") out.kind = search.kind;
  if (typeof search.q === "string" && search.q.trim()) out.q = search.q;
  if (search.source === "official" || search.source === "community") out.source = search.source;
  if (typeof search.id === "string" && /^[a-z0-9]+(-[a-z0-9]+)*$/.test(search.id)) out.id = search.id;
  return out;
}
