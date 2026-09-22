import { keepPreviousData, useQueries, useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/api/client";

export const STATUSES = ["published", "draft", "scheduled", "archived"] as const;
export type Status = (typeof STATUSES)[number];

export interface Filters {
  kind?: string;
  status?: string;
  /** taxonomy:term pairs an item must carry all of. */
  terms?: string[];
  q?: string;
  sort?: string;
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
              limit: PAGE_SIZE,
              count: true,
              cursor,
              kind: filters.kind ? [filters.kind] : undefined,
              status: filters.status ? [filters.status] : undefined,
              term_all: filters.terms?.length ? filters.terms : undefined,
              q: filters.q || undefined,
              sort: filters.sort || undefined,
            },
          },
        }),
      ),
  });
}

export type CountKey = "all" | Status;
const everyCount: readonly CountKey[] = ["all", ...STATUSES];

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
                status: status === "all" ? undefined : [status],
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

export function useContentTypes() {
  return useQuery({
    queryKey: ["content-types"],
    staleTime: Infinity,
    queryFn: async () => unwrap(await api.GET("/content-types", {})),
  });
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
