import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api, ApiError, unwrap } from "@/api/client";

/**
 * useTerm reads one term with every item that carries it, and the revision a
 * rename or removal sends back so it changes exactly the items shown.
 */
export function useTerm(taxonomy: string, term: string | undefined) {
  return useQuery({
    queryKey: ["terms", taxonomy, "detail", term],
    enabled: Boolean(term),
    queryFn: async () => {
      const result = await api.GET("/taxonomies/{taxonomy}/terms/{term}", {
        params: { path: { taxonomy, term: term ?? "" } },
      });
      return { ...unwrap(result), revision: result.response.headers.get("ETag") ?? "" };
    },
  });
}

/** useTermChanges renames, merges and removes a taxonomy's terms. */
export function useTermChanges(taxonomy: string) {
  const client = useQueryClient();
  // A term change rewrites items, so everything counted from them is stale.
  // Once it has gone through, the term no longer exists under its old name,
  // and asking for it again would only fetch a 404 into the closing dialog.
  const settle = (term: string, failed: boolean) =>
    Promise.all([
      ...[["contents"], ["site"], ["publish"], ["taxonomies"]].map((queryKey) =>
        client.invalidateQueries({ queryKey }),
      ),
      client.invalidateQueries({
        queryKey: ["terms"],
        predicate: ({ queryKey }) => failed || queryKey[2] !== "detail" || queryKey[3] !== term,
      }),
    ]);

  const rename = useMutation({
    mutationFn: async ({ term, name, revision }: { term: string; name: string; revision: string }) =>
      unwrap(
        await api.PUT("/taxonomies/{taxonomy}/terms/{term}", {
          params: { path: { taxonomy, term }, header: { "If-Match": revision } },
          body: { name },
        }),
      ),
    onSettled: (_data, error, { term }) => settle(term, Boolean(error)),
  });

  const remove = useMutation({
    mutationFn: async ({ term, revision }: { term: string; revision: string }) => {
      const result = await api.DELETE("/taxonomies/{taxonomy}/terms/{term}", {
        params: { path: { taxonomy, term }, header: { "If-Match": revision } },
      });
      if (result.error) {
        const { code, message, field } = result.error.error;
        throw new ApiError(code, message, field);
      }
    },
    onSettled: (_data, error, { term }) => settle(term, Boolean(error)),
  });

  return { rename, remove };
}
