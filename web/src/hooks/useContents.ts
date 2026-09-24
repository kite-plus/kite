import { keepPreviousData, useQueries, useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/api/client";

export const STATUSES = ["published", "draft", "scheduled", "archived"] as const;
export type Status = (typeof STATUSES)[number];

export interface Filters {
  kind?: string;
  /** Statuses an item has one of. */
  status?: string[];
  /** taxonomy:term pairs an item must carry all of. */
  terms?: string[];
  q?: string;
  sort?: string;
  deletedOnly?: boolean;
  /** limit is the page size, PAGE_SIZE when not given. */
  limit?: number;
}

export const PAGE_SIZE = 20;

/**
 * useContentPage reads one page of a listing.
 *
 * The API has no offset, so the caller keeps the cursors it has been handed
 * and walks back through them; over a set that is being edited an offset
 * would repeat and skip rows.
 */
export function useContentPage(filters: Filters, cursor: string | undefined) {
  return useQuery({
    queryKey: ["contents", "page", filters, cursor ?? ""],
    placeholderData: keepPreviousData,
    queryFn: async () =>
      unwrap(
        await api.GET("/contents", {
          params: {
            query: {
              limit: filters.limit ?? PAGE_SIZE,
              count: true,
              cursor,
              kind: filters.kind ? [filters.kind] : undefined,
              status: filters.status?.length ? filters.status : undefined,
              term_all: filters.terms?.length ? filters.terms : undefined,
              q: filters.q || undefined,
              sort: filters.sort || undefined,
              deleted_only: filters.deletedOnly ? "true" : undefined,
            },
          },
        }),
      ),
  });
}

export type CountKey = "all" | "trash" | Status;
const everyCount: readonly CountKey[] = ["all", ...STATUSES, "trash"];

/** useStatusCounts sizes a kind by status; "all" is every status together. */
export function useStatusCounts(kind: string, keys: readonly CountKey[] = everyCount) {
  const results = useQueries({
    queries: keys.map((status) => ({
      queryKey: ["contents", "count", kind, status],
      queryFn: async () => {
        const page = unwrap(
          await api.GET("/contents", {
            params: {
              query: {
                kind: [kind],
                status: status === "all" || status === "trash" ? undefined : [status],
                deleted_only: status === "trash" ? "true" : undefined,
                limit: 1,
                count: true,
              },
            },
          }),
        );
        return page.total ?? 0;
      },
    })),
  });
  return Object.fromEntries(keys.map((key, i) => [key, results[i].data])) as Partial<
    Record<CountKey, number>
  >;
}

/** useLatest lists what went out last: published items, newest first. */
export function useLatest(kind: string, limit: number, sort: string) {
  return useQuery({
    queryKey: ["contents", "latest", kind, limit, sort],
    queryFn: async () =>
      unwrap(
        await api.GET("/contents", {
          params: { query: { limit, kind: [kind], status: ["published"], sort } },
        }),
      ),
  });
}

export function useSite() {
  return useQuery({
    queryKey: ["site"],
    queryFn: async () => unwrap(await api.GET("/site", {})),
  });
}

/** contentTypesQuery is shared with the route that turns away a kind the project lacks. */
export const contentTypesQuery = {
  queryKey: ["content-types"],
  staleTime: Infinity,
  queryFn: async () => unwrap(await api.GET("/content-types", {})),
};

export function useContentTypes() {
  return useQuery(contentTypesQuery);
}

export function useTaxonomies() {
  return useQuery({
    queryKey: ["taxonomies"],
    queryFn: async () => unwrap(await api.GET("/taxonomies", {})),
  });
}

const termsQuery = (taxonomy: string) => ({
  queryKey: ["terms", taxonomy],
  queryFn: async () =>
    unwrap(
      await api.GET("/taxonomies/{taxonomy}/terms", {
        params: { path: { taxonomy } },
      }),
    ),
});

export function useTerms(taxonomy: string | undefined) {
  return useQuery({ ...termsQuery(taxonomy ?? ""), enabled: Boolean(taxonomy) });
}

/** useTermsOf reads several taxonomies at once, keyed by name. */
export function useTermsOf(taxonomies: string[]) {
  const results = useQueries({ queries: taxonomies.map(termsQuery) });
  return Object.fromEntries(
    taxonomies.map((name, i) => [name, results[i].data?.items ?? []]),
  );
}

/**
 * useMonthlyCounts counts what a kind published in each of the last months,
 * oldest first, by asking for each month's total rather than every item.
 */
export function useMonthlyCounts(kind: string, months: number) {
  const now = new Date();
  const ranges = Array.from({ length: months }, (_, i) => {
    const start = new Date(now.getFullYear(), now.getMonth() - (months - 1 - i), 1);
    const end = new Date(start.getFullYear(), start.getMonth() + 1, 1);
    return { start, end };
  });
  const results = useQueries({
    queries: ranges.map(({ start, end }) => ({
      queryKey: ["contents", "month", kind, start.toISOString()],
      queryFn: async () => {
        const page = unwrap(
          await api.GET("/contents", {
            params: {
              query: {
                kind: [kind],
                status: ["published"],
                published_from: start.toISOString(),
                // The range is inclusive, so it stops a second before the next month.
                published_to: new Date(end.getTime() - 1000).toISOString(),
                limit: 1,
                count: true,
              },
            },
          }),
        );
        return page.total ?? 0;
      },
    })),
  });
  return {
    months: ranges.map((range, i) => ({ month: range.start, total: results[i].data })),
    // A month that was read before keeps its figure through a failed refetch.
    error: results.find((result) => result.error && result.data === undefined)?.error ?? null,
    refetch: () => results.forEach((result) => void result.refetch()),
  };
}
