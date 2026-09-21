import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { api, unwrap } from "@/api/client";

export interface Filters {
  kind?: string;
  status?: string;
  term?: string;
  q?: string;
  sort?: string;
}

const PAGE = 25;

/**
 * useContents pages a listing with the cursor the server hands back.
 *
 * useInfiniteQuery rather than a page number, because the API has no offset:
 * over a set that is being edited an offset repeats and skips rows. Asking
 * for "the next page after this item" is the only question that stays
 * answerable while an author is writing.
 */
export function useContents(filters: Filters) {
  return useInfiniteQuery({
    queryKey: ["contents", filters],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(
        await api.GET("/contents", {
          params: {
            query: {
              limit: PAGE,
              count: true,
              cursor: pageParam,
              kind: filters.kind ? [filters.kind] : undefined,
              status: filters.status ? [filters.status] : undefined,
              term: filters.term ? [filters.term] : undefined,
              q: filters.q || undefined,
              sort: filters.sort || undefined,
            },
          },
        }),
      ),
    getNextPageParam: (last) => (last.has_more ? last.next_cursor : undefined),
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

export function useTerms(taxonomy: string | undefined) {
  return useQuery({
    queryKey: ["terms", taxonomy],
    enabled: Boolean(taxonomy),
    queryFn: async () =>
      unwrap(
        await api.GET("/taxonomies/{taxonomy}/terms", {
          params: { path: { taxonomy: taxonomy! } },
        }),
      ),
  });
}
