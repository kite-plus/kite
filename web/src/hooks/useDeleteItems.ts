import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, ApiError } from "@/api/client";

export interface Target {
  id: string;
  revision: string;
  title: string;
}

export interface BatchResult {
  succeeded: Target[];
  failed: { target: Target; error: Error }[];
}

function useItemsAction(action: "delete" | "restore") {
  const client = useQueryClient();

  return useMutation({
    mutationFn: async (targets: Target[]): Promise<BatchResult> => {
      const result: BatchResult = { succeeded: [], failed: [] };
      for (const target of targets) {
        try {
          const params = {
            path: { id: target.id },
            header: { "If-Match": `"${target.revision}"` },
          };
          const response = action === "delete"
            ? await api.DELETE("/contents/{id}", { params })
            : await api.POST("/contents/{id}/restore", { params });
          if (response.error) {
            throw new ApiError(response.error.error.code, response.error.error.message);
          }
          result.succeeded.push(target);
        } catch (error) {
          result.failed.push({ target, error: error instanceof Error ? error : new Error(String(error)) });
        }
      }
      return result;
    },
    onSettled: async () => {
      await Promise.all(
        [["contents"], ["site"], ["publish"], ["taxonomies"], ["terms"]].map((queryKey) =>
          client.invalidateQueries({ queryKey }),
        ),
      );
    },
  });
}

export function useDeleteItems() {
  return useItemsAction("delete");
}

export function useRestoreItems() {
  return useItemsAction("restore");
}
