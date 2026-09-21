import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, ApiError } from "@/api/client";

interface Target {
  id: string;
  revision: string;
}

/**
 * useDeleteItems removes items one after another.
 *
 * Each carries the revision it was listed at, so an item somebody edited in
 * the meantime is refused rather than deleted out from under them.
 */
export function useDeleteItems() {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async (targets: Target[]) => {
      for (const { id, revision } of targets) {
        const { error } = await api.DELETE("/contents/{id}", {
          params: { path: { id }, header: { "If-Match": `"${revision}"` } },
        });
        if (error) throw new ApiError(error.error.code, error.error.message);
      }
      return targets.length;
    },
    // Some may have gone even when a later one failed.
    onSettled: async () => {
      await Promise.all(
        [["contents"], ["site"], ["publish"], ["taxonomies"], ["terms"]].map((queryKey) =>
          client.invalidateQueries({ queryKey }),
        ),
      );
    },
  });
}
